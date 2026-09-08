# order-service — technical overview

Owns the customer-facing basket (`Cart`) and the `Order` aggregate: placing an order, snapshotting what was
bought, and (once wired) driving payment via a separate `payment-service` over gRPC. Like restaurant-service,
it owns its own Postgres database and runs the transactional outbox pattern; like search-service, it keeps a
local read-model fresh by consuming restaurant-service's events off RabbitMQ, rather than calling
restaurant-service synchronously.

**Current state**: cart (add/update-quantity/remove/get) is fully built, wired, and live. `Checkout`
(`POST /orders`) is fully built and tested but **not yet reachable** — it depends on `order.PaymentProvider`,
whose only planned implementation is a gRPC client to `payment-service`, which does not exist as a running
service yet (design finished, build not started). Order lifecycle
beyond creation (`MarkReady`/`Complete`/`Cancel`, owner-facing order queries, the `payment.succeeded`/
`payment.failed` consumer that actually confirms an order) is designed but not built.

## Layered architecture

```
cmd/api                              → Gin HTTP server: /cart, /cart/items[/:itemId] (live);
                                        /orders exists in code but is not wired into the router yet
cmd/worker                           → RabbitMQ consumer (restaurant.events + identity.events) + outbox relay,
                                        two goroutines side by side — same shape as restaurant-service's cmd/worker
internal/domain/order                → Order/OrderItem aggregate + state machine, DomainEvent/OrderConfirmed,
                                        OrderRepository, GeocodeEntry/GeocodeRepository, Geocoder, PaymentProvider
                                        interfaces, errors
internal/domain/cart                 → Cart/CartItem aggregate, CartRepository, errors
internal/domain/readmodel            → local mirror of restaurant-service's data: Restaurant, Pizza, PizzaPrice,
                                        ToppingPrice, Customer + their repository interfaces, EventDispatcher/
                                        EventHandler/EventPayload
internal/domain/outbox               → OutboxEvent/OutboxStatus/OutboxRepository — verbatim port of
                                        restaurant-service's own outbox domain
internal/application/cart            → schema.go (request/response DTOs), commands/{AddItem,UpdateItemQuantity,
                                        RemoveItem}, queries/GetCart
internal/application/order           → schema.go (CheckoutRequest/Response), commands/Checkout
internal/application/readmodel       → one handler per consumed event: UpsertRestaurant, UpdateRestaurant,
                                        SyncPizza, SyncToppingPrices, UpsertCustomer; EventDispatcher impl
internal/application/outbox          → Worker/Relay — verbatim port of restaurant-service's own outbox application layer
internal/infrastructure/persistence  → one file per aggregate/read-model table: cart.go, order.go, geocode.go,
                                        restaurant.go, pizza.go, pizza_price.go, topping_price.go, customer.go,
                                        outbox.go — all thin GORM wrappers, domain structs double as GORM models
internal/infrastructure/geocoder     → OpenCageGeocoder (order-service's own OpenCage client/quota, independent
                                        of restaurant-service's and search-service's) + CachingGeocoder
                                        (Postgres-backed decorator, mirrors search-service's ES-backed one)
internal/infrastructure/messaging    → RabbitMQ consumer + publisher, same shape as every other service's copy
internal/infrastructure/observability → copied verbatim from restaurant-service
internal/interfaces/http             → middleware (Auth/RequireRole — Customer/Owner, no Admin role needed here),
                                        response.HandleError, validation.ExtractValidationErrors, handlers/
                                        (CartHandler, OrderHandler), routes/ (cart_routes.go — live;
                                        order_routes.go — compiled, not wired)
internal/container                   → shared.go / api.go / worker.go, manual DI
internal/shared/{errors,event,money,geo} → errors sentinels, Event interface, Money (JSON trailing-zero fix),
                                        geo.HaversineKm + geo.AddressHash (both ported from search-service)
```

No `jwt@docker` Traefik label — same as search-service, auth is enforced per-route-group in Gin
(`middleware.Middleware{Auth, EnsureOwner}`) reading Traefik-injected `X-User-ID`/`X-User-Role` headers, not by
parsing a JWT itself. Cart/checkout routes require only `Auth` (any authenticated role — `customer`, `owner`,
or `admin`), not a specific role: identity-service's `Role` is a single fixed value per account, so requiring
`customer` specifically would have locked out an `owner`-role account from ever placing an order themselves.
`EnsureOwner` is kept for the still-unbuilt owner-facing order routes (`GET /orders/restaurants/:id`,
`prepare`/`ready`/`complete`).

## Domain model

```go
// internal/domain/order/order.go
type OrderStatus string
const (
    StatusPending   OrderStatus = "pending"
    StatusConfirmed OrderStatus = "confirmed"
    StatusReady     OrderStatus = "ready"
    StatusCompleted OrderStatus = "completed"
    StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
    ID, CustomerID, RestaurantID uuid.UUID
    Status          OrderStatus
    Fulfillment     Fulfillment  // "delivery" | "pickup"
    ContactEmail    string       // resolved server-side from the customers read-model, never client-submitted
    ContactPhone    *string      // client-submitted, optional — genuinely order-specific (courier contact)
    DeliveryAddress *Address     // JSONB snapshot, nil for pickup
    DeliveryLat, DeliveryLon *float64
    Items           []OrderItem
    Subtotal, DeliveryFee, Total decimal.Decimal
    Currency        string
    PaymentID       *string      // payment-service's own Payment.ID, set after Checkout's gRPC call
    PlacedAt        time.Time    // gorm autoCreateTime
    ConfirmedAt, ReadyAt, CompletedAt, CancelledAt *time.Time
}

func (o *Order) Confirm() error   // pending -> confirmed; raises OrderConfirmed
func (o *Order) MarkReady() error // confirmed -> ready
func (o *Order) Complete() error  // ready -> completed
func (o *Order) Cancel() error    // pending/confirmed -> cancelled (too late once ready); no domain event
```

**No separate `preparing` stage** — this was in the original design (`confirmed → preparing → ready`, with its
own `StartPreparing()` method and `PrepStartedAt` timestamp) and was deliberately removed: the `status` column
already records "currently being prepared," and a timestamp for exactly when that started wasn't judged worth
the extra owner-facing action. `MarkReady` now transitions directly from `confirmed`.

```go
// internal/domain/order/order_item.go — a snapshot, not a live reference
type OrderItem struct {
    ID, OrderID, PizzaID, SizeID uuid.UUID
    PizzaName       string    // snapshotted — order_items.pizza_id/size_id carry no FK, since a read-model
    SizeDiameter    int16     // pizza row can be hard-deleted on archive (SyncPizza's RemovePizza)
    ExtraToppingIDs []uuid.UUID // customer-requested extras, references topping_prices — NOT a pizza's own
                                 // baked-in default toppings, which order-service's read-model doesn't even mirror
    Quantity        int16
    UnitPrice, TotalPrice decimal.Decimal
}
```

```go
// internal/domain/cart/cart.go
type Cart struct {
    ID, CustomerID, RestaurantID uuid.UUID // one cart per customer (carts.customer_id UNIQUE), scoped to
    Items        []CartItem                // a single restaurant at a time
    CreatedAt    time.Time
    UpdatedAt    *time.Time
}
func (c *Cart) EnsureRestaurant(restaurantID uuid.UUID) error // -> ErrCartRestaurantMismatch (409) on a
                                                                 // cross-restaurant add attempt

// internal/domain/cart/cart_item.go
type CartItem struct {
    ID, CartID, PizzaID, SizeID uuid.UUID
    Quantity        int16
    ExtraToppingIDs []uuid.UUID // sorted before persistence (NewCartItem does this), so identical combos
}                                // always produce the same JSONB value for the upsert's conflict target
```

```go
// internal/domain/order/geocode.go — a permanent cache, not a TTL'd one; "geocode", not "geocode_cache"
// (this repo's tables are named for the entity, not its storage role)
type GeocodeEntry struct {
    AddressHash string    // SHA-256 hex of the normalized address (geo.AddressHash) — CHAR(64)
    Lat, Lon    float64
    CreatedAt   time.Time
}
```

All domain structs double as GORM models (explicit `gorm:` tags, `TableName()` methods) — no separate
persistence-layer DTO, matching restaurant-service's own convention. `OrderRepository`/`CartRepository` both
expose `WithTx(tx *gorm.DB) X` (mirrors restaurant-service's `RestaurantRepository.WithTx`), since `Checkout`
needs one transaction spanning both repositories.

## Local read-model (consumes restaurant-service's and identity-service's events)

Same pattern as search-service: JSON-contract-only, local mirror structs, order-service never imports
restaurant-service's or identity-service's Go types.

| Event (exchange) | Handler | Table(s) |
|---|---|---|
| `restaurant.launched` | `UpsertRestaurant` | `restaurants`, `pizzas`, `pizza_prices`, `topping_prices` |
| `restaurant.updated` | `UpdateRestaurant` | `restaurants` |
| `restaurant.pizza_updated` | `SyncPizza` | `pizzas`, `pizza_prices` |
| `restaurant.topping_prices_updated` | `SyncToppingPrices` | `topping_prices` |
| `user.registered` (`identity.events`) | `UpsertCustomer` | `customers` |

`customers.id` upsert is `ON CONFLICT DO NOTHING` (redelivery-safe only, no `updated_at` guard needed — a
`user.registered` event fires exactly once per user, ever, so there's no later event that could race it,
unlike the restaurant tables below). `restaurants`/`pizzas`/`pizza_prices`/`topping_prices` upserts are
guarded by `updated_at` (`ON CONFLICT ... WHERE excluded.updated_at > table.updated_at`), the same
out-of-order-redelivery protection search-service's own Painless-scripted guards provide.

**A real, previously-shipped bug found and fixed while building `Checkout`**: `readmodel.Restaurant.Pickup`
and `readmodel.PizzaPrice.IsActive` both carried a GORM `default:true` tag. GORM's `Create`/upsert path omits
a field from the `INSERT` entirely whenever it's the Go zero value *and* carries a `default:` tag — for a
`bool`, the zero value is `false`, which is also a completely legitimate business value ("no pickup," "this
size is deactivated"). So every real sync of a pickup-disabled restaurant or a deactivated price was silently
corrupted to `true`/active. Fixed by dropping the gorm-tag default (the migration's own `DEFAULT true` is
untouched — that's a harmless DB-level backstop, not the bug).

## Geocoding — Postgres-backed, ported from search-service

Checkout needs to re-verify a delivery address is within the restaurant's radius without paying for an
OpenCage lookup on every checkout. `CachingGeocoder` (`internal/infrastructure/geocoder/caching_geocoder.go`)
mirrors search-service's own ES-backed `CachingGeocoder` shape exactly, backed by Postgres instead (order-service
has its own DB; search-service doesn't). Cache key: `geo.AddressHash` — SHA-256 hex of the normalized address
(lowercased, whitespace-collapsed `house|street|city|postalCode`), same algorithm as search-service's, ported
directly. No TTL — street coordinates don't change.

`internal/shared/geo/haversine.go` — `HaversineKm` — no Haversine/PostGIS precedent existed anywhere in this
repo before this; the only prior distance check was Elasticsearch's `arcDistance` script in search-service,
not reusable in a Postgres-only service.

order-service has its own **third** independent OpenCage client (separate API key/quota from
restaurant-service's and search-service's — this repo duplicates infrastructure per service rather than
sharing it), slimmer than restaurant-service's copy (no timezone field, matching search-service's shape).

Geocoder failures fail closed: `Checkout` returns `ErrGeocodingUnavailable` (503) rather than skipping the
radius check.

## Cart

One active cart per customer, scoped to a single restaurant. Adding an item from a different restaurant is
rejected (`409 ErrCartRestaurantMismatch`), never auto-cleared — the customer must explicitly clear the cart
first. No expiry/TTL: a cart persists until checkout consumes it or the customer clears it.

`AddOrMergeItem` is a Postgres upsert — `ON CONFLICT (cart_id, pizza_id, size_id, extra_toppings) DO UPDATE SET
quantity = cart_items.quantity + excluded.quantity`. Adding the same pizza+size+extras combo twice increments
quantity; the same pizza+size with *different* extras stays a separate line (they have different actual prices).
`extra_toppings` (renamed from an original bare `toppings` — see below) is why: two lines that are the same
pizza+size but different extras need different conflict-target values to correctly stay separate rows.

**`extra_toppings`, not `toppings`** (`cart_items`/`order_items` column, `ExtraToppingIDs` in Go, `extraToppingIds`
in the JSON API): renamed for clarity — restaurant-service has its own, unrelated "pizza's own default
toppings" concept (`pizzas.toppings`), and this column is specifically customer-requested *extras* referencing
`topping_prices`, never a pizza's own baked-in toppings (which order-service's read-model doesn't even mirror).

**`RemoveItem` deletes the cart row too when it removes the last item** — otherwise an empty-but-existing cart
row would linger, still locking the customer to its `RestaurantID` via `Cart.EnsureRestaurant` even though it
holds nothing. Implemented as one transaction: delete the item, count what's left, delete the cart if zero
remain. Makes "missing cart" and "empty cart" the same state everywhere, matching `Checkout`'s own
`ErrCartEmpty` (which treats both as one case).

`GetCart` resolves every item's current name/price/availability fresh from the read-model on every call — no
price is ever stored on a `CartItem`. If a pizza was archived or a size deactivated since being added, that
item comes back flagged `available: false` rather than silently vanishing, so the customer can see it and
remove it explicitly.

**`ClearCart` (a customer-facing `DELETE /cart`) was built, then deliberately removed** on request ("I do not
want it now") — the command, handler, and route were all pulled back out. `cart.CartRepository.Clear` itself
was **not** removed; it's a separate, still-needed piece `Checkout` uses to clear the cart after creating an
`Order`. Only the standalone customer action was deferred.

## Checkout

`Checkout` (`internal/application/order/commands/checkout.go`) is the one command in this service using
`db.Transaction`/`WithTx` — tested with real repositories + real Postgres throughout (only `Geocoder` and
`PaymentProvider`, the genuinely external dependencies, get local test fakes), matching restaurant-service's
own precedent for any command with real transactional behavior.

1. Load the customer's `Cart` — `ErrCartEmpty` (422) if missing or empty.
2. Look up the restaurant; reject if it doesn't offer the requested `Fulfillment` (`pickup=false` for pickup,
   or `delivery_type='none'` for delivery) — `ErrFulfillmentNotSupported` (409).
3. Resolve `ContactEmail` from the `customers` read-model by the JWT-derived customer id — never client-submitted.
4. Re-resolve every cart line from the read-model, snapshotting into an `OrderItem` — a pizza that's gone, a
   deactivated size, or an extra that's no longer priced all produce `ErrCartItemUnavailable` (409).
5. Reject if subtotal < the restaurant's `minimum_order` — `ErrBelowMinimumOrder` (422).
6. If delivery: geocode the address (`CachingGeocoder`) + `geo.HaversineKm` against the restaurant's
   `deliveryKm` — `ErrOutsideDeliveryRadius` (422) if too far.
7. **One transaction**: insert `Order` (status `pending`, no `payment_id` yet) + `OrderItems`, clear the cart.
8. Outside that transaction (a gRPC call shouldn't hold DB locks open): call `PaymentProvider.CreatePayment`
   with a redirect URL built from `FRONTEND_BASE_URL + "/orders/" + orderID` → `{paymentID, checkoutURL}`.
9. `UPDATE orders SET payment_id = ?`, return `checkoutUrl` to the caller.

**Not reachable yet**: `OrderHandler.Checkout` and `POST /orders`'s route exist and are fully unit-tested, but
`routes.Handlers` has no `OrderHandler` field and `SetupRoutes` doesn't call `SetupCheckoutRoutes` —
deliberately. `Checkout` depends on `order.PaymentProvider`, and the only planned implementation (a gRPC
client to `payment-service`) doesn't exist yet, because `payment-service` itself hasn't been built (design is
finished, build not started). Wiring the route now with a nil handler
would panic on every real request, since cart's own container wiring already made this app live. Both the
`OrderHandler` field and the route registration land together once the real `PaymentProvider` exists.

**Also not yet built**: `MarkReady`/`Complete`/`Cancel` commands and their owner-facing endpoints, `GetOrder`/
list queries, the `payment.succeeded`/`payment.failed` consumer that actually calls `Order.Confirm()`/
`Cancel()` (this is currently the *only* way an order would ever leave `pending`), and the
`order.confirmed` → notification-service handler.

## HTTP API

| Method | Path | Auth | Status |
|---|---|---|---|
| `GET` | `/cart` | authenticated user | live |
| `POST` | `/cart/items` | authenticated user | live |
| `PATCH` | `/cart/items/:itemId` | authenticated user | live |
| `DELETE` | `/cart/items/:itemId` | authenticated user | live |
| `POST` | `/orders` | authenticated user | built, not wired (see above) |

No `DELETE /cart` (deferred, see Cart section above). No owner-facing order routes yet.

## Migrations

`001_create_outbox_events` (built in from day one — order-service's outbox isn't a later retrofit like
restaurant-service's, so it has to be numbered `001`, not last) → `002_create_customers` →
`003_create_restaurants` → `004_create_pizzas` → `005_create_pizza_prices` → `006_create_topping_prices` →
`007_create_geocode` → `008_create_carts` → `009_create_cart_items` → `010_create_orders` (FK to
`restaurants`; `CHECK (total = subtotal + delivery_fee)`; partial unique index on `payment_id WHERE payment_id
IS NOT NULL`) → `011_create_order_items` (FK to `orders` `ON DELETE CASCADE`; no FK on `pizza_id`/`size_id` —
those read-model rows can be hard-deleted on archive).

## Testing

Same shape as restaurant-service: `tests/` mirrors `internal/`, integration-style against real Postgres via
`compose.test.yaml` (`pg-order-test`, `migrate-order-test`, `order-test`), `tests/testutil/db.go` +
`TruncateTables`, `tests/infrastructure/db/fixtures/*.go` for seed data. Application-layer commands that don't
need real transactional behavior (`AddItem`, `UpdateItemQuantity`, `RemoveItem`, `GetCart`) use
`tests/testutil.Mock*Repository` fakes; `Checkout` (the one command using `db.Transaction`/`WithTx`) uses real
repositories throughout, with only `Geocoder`/`PaymentProvider` faked locally in its own test file.

A new endpoint or handler change is also verified live — booting the real `air`-reload dev container and
`curl`-ing the happy path plus the main error paths (not-found, validation failure, missing/wrong auth) —
before being considered done. This caught a real bug once already: a mocked test suite can pass while broken
if the mock's own zero-value default happens to match the code's own wrong assumption about a dependency's
contract (see the read-model bug above for a related but distinct case — a schema/ORM interaction, not an
HTTP one).
