# search-service — technical overview

Read-only search API over restaurant/pizza data, backed by Elasticsearch and kept fresh by consuming four
restaurant-service events off RabbitMQ: `restaurant.launched` (full snapshot on first index), `restaurant.updated`
(restaurant-field delta), `restaurant.pizza_updated` (single-pizza delta, including pizza pricing), and
`restaurant.topping_prices_updated` (restaurant-wide extra-topping pricing). `restaurant.reactivated`/
`restaurant.deactivated` and a `Delete` repository method are still not built — restaurant-service has no
reactivate/deactivate endpoints yet to publish those events, so there's nothing to consume.

## Layered architecture

```
cmd/api                           → Gin HTTP server, GET /search only, no auth
cmd/worker                        → RabbitMQ consumer
cmd/worker/bootstrap              → app/runner setup (graceful shutdown, signal handling) — copied from
                                     restaurant-service's cmd/worker/bootstrap
internal/domain/index             → IndexedRestaurant, IndexedPizza, IndexedPizzaPrice, IndexedToppingPrice,
                                     RestaurantFields, GeoPoint, SearchRepository, SearchQuery, Address,
                                     Geocoder, EventDispatcher, EventHandler, EventPayload — interfaces +
                                     plain data, no business logic of its own
internal/application/index        → UpsertSnapshot, UpdateRestaurantFields, SyncPizza, SyncToppingPrices
                                     (one handler per consumed event), EventDispatcher impl
internal/application/query        → SearchRestaurants (the /search use case — resolves address, then searches)
internal/infrastructure/elasticsearch → client wrapper, index_setup.go (EnsureIndex, both indices),
                                     search_repository.go, geocode_repository.go (CachingGeocoder)
internal/infrastructure/geocoder  → OpenCageGeocoder — search-service's own copy, independent of
                                     restaurant-service's (see "Geocoding" below)
internal/infrastructure/messaging → RabbitMQ consumer, copied from email-service's shape (the bug-fixed version —
                                     checks both conn and channel closed before reconnecting)
internal/infrastructure/observability → copied verbatim from restaurant-service (slog-based logger, Gin middleware)
internal/interfaces/http          → SearchHandler, routes.go — no middleware package at all, this API has no auth
internal/container                → api.go / worker.go
```

Unlike identity/restaurant/email-service, there is no database — `internal/infrastructure/` has no
`gorm`/`persistence`/`redis`/`auth`/migrations at all. There is a `compose.test.yaml` entry (`search-test` +
`elasticsearch-test`, same "needs a `-test` container" shape identity/restaurant use for Postgres), used by the
real-ES integration suite — see "Testing" below.

## Domain model

```go
type IndexedOpeningHours struct {
    Weekday, Open, Close string // e.g. "monday", "09:00", "22:00" — one entry per range, flattened across
                                 // the whole week (restaurant-service's own shape is day-keyed instead;
                                 // flattening lets one Painless script branch check "is it open right now"
                                 // for any weekday without picking a field name dynamically)
}

type IndexedRestaurant struct {
    ID, Name, Slug, City string
    Location     GeoPoint // always present — see note below
    Timezone     string   // IANA name (e.g. "Europe/Berlin"), sourced from OpenCage's own geocoding
                           // response — see "GET /search" below for how it drives openNow
    Currency     string
    Pickup       bool
    DeliveryType string
    DeliveryKm   *int16 // nil when DeliveryType is "none" — see "GET /search" below for what that means for matching
    MinimumOrder float64 // real numeric field, not a display-only keyword string like the price fields
                          // below — needed for actual range/sort, see "Elasticsearch index" below
    Tags         []string
    OpeningHours []IndexedOpeningHours
    Rating       float64
    TotalReviews int32
    Pizzas       []IndexedPizza
    ToppingPrices []IndexedToppingPrice // restaurant-wide extra-topping prices, unrelated to any one pizza
    UpdatedAt    time.Time // guards restaurant.updated/restaurant.launched redelivery — see "Events" below
    ToppingPricesUpdatedAt time.Time // separate guard for ToppingPrices — see "Events" below for why
}

type IndexedPizza struct {
    ID           uuid.UUID
    Name         string
    IsVegetarian bool
    Toppings     []string  // topping names, not IDs — this is a search document, not a pizza-editing DTO
    Prices       []IndexedPizzaPrice // active sizes/prices only — see below
    UpdatedAt    time.Time // guards restaurant.pizza_updated redelivery, per-pizza — see "Events" below
}

type IndexedPizzaPrice struct {
    SizeID     uuid.UUID
    DiameterCm int16
    Price      string // wire-format decimal string, matching restaurant-service's Money — never computed on,
                       // only displayed, so no decimal dependency was added just for this
}

type IndexedToppingPrice struct {
    ToppingID  uuid.UUID
    Name       string
    ExtraPrice string // same wire-format-string convention as IndexedPizzaPrice.Price
}

// RestaurantFields is IndexedRestaurant minus Pizzas/ToppingPrices — restaurant.updated never carries pizza
// or topping-pricing data, so this type has no slot for either (not a shared struct with an ignored field).
type RestaurantFields struct {
    Name, Slug, City string
    Location     GeoPoint
    Timezone     string
    Currency     string
    Pickup       bool
    DeliveryType string
    DeliveryKm   *int16
    MinimumOrder float64
    Tags         []string
    OpeningHours []IndexedOpeningHours
    Rating       float64
    TotalReviews int32
    UpdatedAt    time.Time
}
```

All domain structs (`IndexedRestaurant`, `IndexedPizza`, `IndexedPizzaPrice`, `IndexedToppingPrice`, `GeoPoint`)
carry explicit camelCase `json` tags, matching every other response type in this repo — without them, Gin's
default marshaling emits Go's capitalized field names instead.

`IndexedPizza.Prices` only ever holds a pizza's currently-*active* prices — `SyncPizza`/`UpsertSnapshot` filter
out inactive sizes before mapping, the same way a whole pizza with no active price at all gets excluded/removed
rather than indexed unorderable. `IndexedToppingPrice` is restaurant-scoped, not per-pizza (`ToppingPriceRepository.
ListByRestaurant` in restaurant-service is not keyed by pizza) — it lives directly on `IndexedRestaurant`, not
nested inside `IndexedPizza.Toppings` (which stays plain topping names, unchanged).

`Location` is a plain value, not a pointer — `restaurant-service`'s `Lat`/`Lon` are nullable `*float64` at the
domain/DB level (a `draft` restaurant that hasn't completed its `address` checklist item yet has neither), but
`UpdateAddress.Execute` (restaurant-service) requires successful geocoding before it marks the `address`
checklist item complete, and `address` gates `draft→review`, which gates every path to `active`. So `Lat`/`Lon`
are always set by the time `restaurant.launched` fires — this is a real guarantee from restaurant-service's own
checklist invariant, not an assumption. `UpsertSnapshot.Handle` still validates the incoming payload has both
before mapping it (an event crossing a service boundary is a validation point even when the source's own
invariant is solid) — a payload missing either is rejected outright rather than silently indexed with a
fabricated `(0,0)` location, which would be a real point in the ocean, not "unknown."

`SearchRepository`:

```go
type SearchRepository interface {
    UpsertSnapshot(ctx context.Context, r IndexedRestaurant) error
    UpdateFields(ctx context.Context, id uuid.UUID, fields RestaurantFields) error
    UpsertPizza(ctx context.Context, restaurantID uuid.UUID, pizza IndexedPizza) error
    RemovePizza(ctx context.Context, restaurantID, pizzaID uuid.UUID, updatedAt time.Time) error
    UpdateToppingPrices(ctx context.Context, restaurantID uuid.UUID, prices []IndexedToppingPrice, updatedAt time.Time) error
    Search(ctx context.Context, q SearchQuery) ([]IndexedRestaurant, error)
}
```

`SearchQuery.Location` is a plain `GeoPoint`, not optional — every search resolves a customer address to
coordinates before querying (see "Geocoding" and "`GET /search`" below), so there's no "no location" case to
model. There's no `RadiusKm` on `SearchQuery` either — coverage is judged per-restaurant against its own
`DeliveryKm`, not a value the caller supplies. `SearchQuery` also carries `Fulfillment string` (`"delivery"` \|
`"pickup"` \| `""`), `Tags []string`, `OpenNow bool`, and `Sort string` (`"distance"` \| `"minimumOrder"` \| `""`)
— all zero-value by default, so an old-style request with none of them behaves exactly as before. See
"`GET /search`" below for how each shapes the query.

A `Delete` method for pulling a restaurant back out of the index on `restaurant.deactivated` is not implemented —
restaurant-service doesn't publish `deactivated`/`reactivated` yet, so it would have zero callers.

## Elasticsearch index

Two indices, both created (idempotently — checks existence first, no-ops if already there) by `EnsureIndex`,
called at both API and worker startup:

- **`restaurants`** — the search-facing index. `pizzas` is mapped as a plain object array, not `nested` —
  object-array fields flatten into multi-valued arrays (`pizzas.name`, `pizzas.toppings`), which a top-level
  `multi_match` can search directly with no `nested` query. Trade-off: this loses precise per-pizza cross-field
  matching (e.g. "vegetarian AND has pepperoni" could match two *different* pizzas on the same restaurant
  document rather than one pizza satisfying both) — accepted at this index's current scope, revisit if that
  precision is ever needed. `deliveryKm` is mapped `short` — see "`GET /search`" for how it's used.
  `pizzas.prices`/`toppingPrices` are both plain object arrays too, each price kept as a `keyword` (a wire-format
  decimal string, not `float` — display-only, never range-queried or sorted on in this slice). `timezone` is
  `keyword` (an IANA name, compared exactly, never analyzed). `minimumOrder` is the one deliberate departure from
  that display-only-string convention: `scaled_float` (`scaling_factor: 100`) — a real numeric field, because
  unlike every other price in this index it needs to be sorted (`sort=minimumOrder`), and a plain `keyword` would
  sort `"10.00"` before `"9.00"` lexicographically. `openingHours` is a plain object array (same non-`nested`
  tradeoff as `pizzas`) with `weekday`/`open`/`close` all `keyword` — flattened this way, the three sub-fields
  become parallel same-order doc-value arrays, which is what the `openNow` filter script below relies on. The
  top-level `updatedAt`, the nested `pizzas.updatedAt`, and the top-level `toppingPricesUpdatedAt` are all mapped
  `date` — three separate ordering guards (restaurant, per-pizza, and topping-prices-as-a-whole), not one shared
  field; see "Events" below for why a pizza edit or a topping-price edit can't reuse the restaurant-level guard.
- **`geocode`** — a disposable key/value cache for resolved addresses, unrelated to search. See "Geocoding"
  below.

## Geocoding

A customer shouldn't need to know their own coordinates, and `/search` filters by whether a restaurant can
actually deliver to them — both require resolving an address to `lat`/`lon`. `internal/infrastructure/geocoder/`
is search-service's **own** OpenCage client, independent of restaurant-service's — deliberately not shared code
(this repo duplicates per-service infrastructure rather than sharing a library across services, the same
pattern already used for the RabbitMQ consumer), and a deliberately separate cost/quota from
restaurant-service's own OpenCage usage.

Because OpenCage is a paid, rate-limited external API and `/search` can be hit far more often than a restaurant's
own address ever changes, geocoding is cached: `CachingGeocoder` (`internal/infrastructure/elasticsearch/geocode_repository.go`)
wraps the raw `OpenCageGeocoder` and checks the `geocode` index first. The cache key is a SHA-256 hash of the
normalized address (lowercased, whitespace-collapsed `house|street|city|postalCode`) — a cache hit costs one ES
point lookup (~5ms observed) instead of a network round-trip to OpenCage (~500ms+ observed); many customers
plausibly share an address (same street, same building), so the cache benefits more than just repeat searches
from the same person. A cache miss calls OpenCage, then writes the result before returning it.

`CachingGeocoder` has no unit tests, same reasoning as `SearchRepository`'s ES-backed methods — it's only
meaningfully testable against a real cluster (or a real OpenCage response), not mocked. Verified live instead —
see "Testing" below.

## Events

```mermaid
flowchart LR
    E1["restaurant.launched\n(full snapshot — restaurant\nfields + priced pizzas +\ntopping prices)"] --> H1["UpsertSnapshot"]
    E2["restaurant.updated\n(restaurant-field delta,\nno pizzas/topping prices)"] --> H2["UpdateRestaurantFields"]
    E3["restaurant.pizza_updated\n(single-pizza delta,\nincl. pricing)"] --> H3["SyncPizza"]
    E4["restaurant.topping_prices_updated\n(restaurant-wide extra-\ntopping price list)"] --> H4["SyncToppingPrices"]
    H1 --> ES[("Elasticsearch\nrestaurants index")]
    H2 --> ES
    H3 --> ES
    H4 --> ES
    API["GET /search?house=&street=&city=&postalCode=&q=\n&fulfillment=&tags=&openNow=&sort="] --> GEO["Geocoder\n(cached via geocode index)"]
    GEO --> ES
```

Each handler parses its event into a local, independent copy of restaurant-service's payload shape
(`restaurantLaunchedPayload`, `restaurantUpdatedPayload`, `pizzaUpdatedPayload`, `toppingPricesUpdatedPayload`) —
search-service never imports restaurant-service's Go code, the two services only agree on the JSON wire contract.
All four routing keys are bound in `messaging.Exchanges["restaurant.events"]`; `restaurant.reactivated`/
`restaurant.deactivated` are not, since restaurant-service doesn't publish them yet.

- **`restaurant.launched`** (`internal/application/index/upsert_snapshot.go`) — restaurant-service composes the
  full snapshot in-process (`launch_restaurant.go`'s `Enricher`) and publishes it directly, so search-service
  never makes a synchronous call back into restaurant-service. `UpsertSnapshot.Handle` calls
  `SearchRepository.UpsertSnapshot`, which writes via a Painless-scripted `_update` carrying an `upsert:` body for
  the doc-doesn't-exist-yet case (first index) — the whole document is replaced only if the incoming `updatedAt`
  is strictly newer than what's already stored, otherwise the write is a no-op (`ctx.op = 'noop'`).
- **`restaurant.updated`** (`internal/application/index/update_restaurant_fields.go`) — fires on every
  post-launch edit to restaurant-level fields (address/contact/delivery/opening-hours; see restaurant-service's
  own doc for `NotifyUpdated()`'s call sites). `UpdateRestaurantFields.Handle` calls
  `SearchRepository.UpdateFields`, a field-by-field scripted update (deliberately **excluding** `pizzas` — a
  restaurant-field edit must never touch the indexed pizzas) guarded the same way, by the same top-level
  `updatedAt`. No `upsert:` fallback — a missing parent doc here is a contract violation (this event can only
  fire once `restaurant.launched` has already created it), so it 404s through to the consumer's DLX/retry path
  instead of silently creating a partial document.
- **`restaurant.pizza_updated`** (`internal/application/index/sync_pizza.go`) — fires on pizza create/update/
  price-change (see restaurant-service's own doc for exactly which commands publish it). The payload carries the
  pizza's full current price list (`sizeId`/`diameterCm`/`price`/`isActive` per size), already flowing through
  the wire since it reuses restaurant-service's `PizzaResponse` shape. `SyncPizza.Handle` checks the payload's
  pizza status and prices: archived or with no active price row (unsellable) routes to
  `SearchRepository.RemovePizza`; everything else routes to `SearchRepository.UpsertPizza`, mapping only the
  *active* prices into `IndexedPizza.Prices` (an inactive size isn't orderable, so it's dropped the same way an
  entirely-unpriced pizza is). Both do a **per-pizza** partial merge into the document's `pizzas` array
  (find-by-id, then replace-in-place-or-append / remove) via their own Painless scripts, guarded by that one
  pizza's own `pizzas[i].updatedAt` — **not** the restaurant-level `updatedAt` guard, since a pizza edit (e.g. a
  price-only change) never touches the `restaurants` row in restaurant-service, so the two timestamps are
  unrelated. Same no-`upsert:`-fallback contract as `UpdateFields`, and `RemovePizza` is additionally idempotent
  against a pizza that's already absent (`ctx.op = 'noop'` when not found, not an error) — redelivery-safe either
  way.
- **`restaurant.topping_prices_updated`** (`internal/application/index/sync_topping_prices.go`) — fires whenever
  an owner sets the restaurant's own extra-topping prices via `SetToppingPrices` (restaurant-scoped, not
  per-pizza — `ToppingPriceRepository.ListByRestaurant`). The payload always carries the *full current* topping-
  price list for that restaurant (not a delta), since `SetToppingPrices` re-reads the whole list after its own
  upsert. `SyncToppingPrices.Handle` calls `SearchRepository.UpdateToppingPrices`, which wholesale-replaces
  `IndexedRestaurant.ToppingPrices` via its own Painless script, guarded by its own top-level
  `toppingPricesUpdatedAt` field — separate from both the restaurant-level `updatedAt` and any pizza's
  `pizzas[i].updatedAt`, since `SetToppingPrices` touches neither the `restaurants` row nor any `pizzas` row.
  Same no-`upsert:`-fallback contract as `UpdateFields`/`SyncPizza`. `restaurant.launched` also seeds
  `ToppingPrices` at first-index time (if any were set pre-launch) as part of its full-document write, but
  leaves `ToppingPricesUpdatedAt` at its zero value there — harmless, since any real future update's timestamp
  is always later than a zero value, so the guard can only become *more* permissive, never incorrectly reject a
  genuine update (the same reasoning already applies to launch-time pizzas' `UpdatedAt`).

All four guards compare epoch milliseconds (`t.UnixMilli()` on the Go side, `Instant.parse(...).toEpochMilli()`
in Painless) via a strict `>`, entirely inside the scripted update — there's no read-then-write race between two
out-of-order deliveries, since ES applies the script atomically per document. This exists because RabbitMQ
redelivery can reorder messages: `Republish` (the consumer's retry path) re-sends a failed message to the *back*
of the queue with an incremented `x-retry-count`, while newer messages for the same restaurant/pizza/topping-price
list get processed in between — so a transient failure on an older event, followed by a newer one succeeding
first, would otherwise let the retried older event silently clobber the newer data once it's finally reprocessed.

## `GET /search`

Query params: `house`, `street`, `city`, `postalCode` (all **required** — a full delivery address, not raw
coordinates the caller has to already know), `q` (free text, optional — empty means "browse everything
deliverable to this address"), `fulfillment` (`delivery` \| `pickup`, optional), `tags` (comma-separated,
optional), `openNow` (bool, optional), `sort` (`distance` \| `minimumOrder`, optional). A request missing any
address field is a `400`, not silently ignored. No auth — public by design, matching the marketplace's core
purpose. No `/health` route, matching restaurant-service's own precedent (it has none either, unlike
identity-service).

`SearchHandler` binds the address and new params via `ShouldBindQuery` (`form` tags, `binding:"required"` on the
address fields only), splits `tags` on `,`, builds an `index.Address` plus a `query.SearchRestaurantsRequest`, and
hands off to `SearchRestaurants.Execute`, which resolves the address to `lat`/`lon` via the geocoder (cached —
see "Geocoding") before ever touching `SearchRepository`. A geocode failure (address doesn't resolve, OpenCage
unreachable) surfaces as a `500` — there's no fallback to an unscoped, address-less search.

**Fulfillment coverage defaults to "either way works."** `deliveryClause` is a Painless script comparing
`doc['location'].arcDistance(customerLat, customerLon)` (meters) against `doc['deliveryKm'].value * 1000` —
excluded outright if the restaurant has no `deliveryKm` (`DeliveryType: "none"`), same "unknown state excluded"
convention `openNow` uses below. `pickupClause` is a plain `{"term": {"pickup": true}}`. With no `fulfillment`
param, the query filters on `should: [pickupClause, deliveryClause]` with `minimum_should_match: 1` — a restaurant
qualifies if it can serve the customer *either* way, which is what lets a pickup-only restaurant surface in a
plain, filter-less search (an earlier version of this endpoint excluded pickup-only restaurants from every
search outright; that gap is closed). `fulfillment=delivery`/`fulfillment=pickup` narrow to just one clause
instead of the `should`.

**`tags`** adds one `{"term": {"tags": <tag>}}` filter per requested tag (implicit AND — requesting `vegan` and
`halal` means both, not either).

**`openNow`** adds a script filter that reads the *document's own* `timezone` field, not a query param, so "now"
is localized per-restaurant: `ZonedDateTime.ofInstant(Instant.ofEpochMilli(nowMillis), ZoneId.of(doc['timezone']
.value))`, then a linear scan of the flattened `openingHours` arrays for a range on the current weekday where
`open <= now < close` (zero-padded `"HH:mm"` string comparison). A restaurant with no stored timezone is excluded,
not treated as always-open or always-closed.

Within whatever passes those filters, relevance is not plain keyword matching:

- **Typo tolerance**: the `multi_match` query carries `"fuzziness": "AUTO"` across `name`/`pizzas.name`/
  `pizzas.toppings`, so e.g. `q=Pizzeriaa` still matches `Pizzeria Roma`.
- **Rating boost**: results are wrapped in a `function_score` query — `field_value_factor` on `rating` with a
  `log1p` modifier, `boost_mode: "sum"` (added to, not multiplied against, the text-match score, so a 4.9-rated
  restaurant with a weak text match still can't outrank a 3.2-rated restaurant with a strong one) and
  `"missing": 0` (an unrated restaurant gets no boost rather than being penalized).

Ranking stays relevance+rating by default even after the fulfillment/tags/openNow filters narrow the candidate
set — every remaining hit already matches what was asked for, so which one is a few hundred meters closer
matters less than which one actually matches what the customer typed. **`sort=distance`/`sort=minimumOrder`
replace that ordering outright** rather than blending with it (`_geo_distance` ascending, or the `minimumOrder`
field ascending) — a customer who explicitly asks to sort by distance wants distance order, not
distance-nudged-by-text-relevance. The underlying `bool` query (text match + every filter above) is unchanged
either way, only the ordering differs. (An earlier version of this endpoint took raw `lat`/`lon`/`radiusKm` and
always sorted geo-scoped results by distance; that was replaced by the address-required, filter-based design
described here, with distance-sort now opt-in via `sort=distance`.)

## Testing

Mixed, not uniform: `tests/application/index/` and `tests/application/query/` mock `SearchRepository` via a
shared `tests/testutil.MockSearchRepository` (matches email-service's plain-unit-test approach, no
infrastructure needed) — one test file per handler (`upsert_snapshot_test.go`, `update_restaurant_fields_test.go`,
`sync_pizza_test.go`, `sync_topping_prices_test.go`). `tests/interfaces/http/handlers/` builds the real
`SearchRestaurants` use case over that same mock and drives the handler through `httptest`, the same pattern
restaurant-service's handler tests use (real use case, faked boundary dependency), just without a real DB behind
it.

`tests/infrastructure/elasticsearch/search_repository_test.go`, though, is a **real**-Elasticsearch integration
suite — `toESRestaurant`/`fromESRestaurant`/the Painless scripts are only meaningfully testable against a real
cluster, not mocked. It runs against the `compose.test.yaml` `elasticsearch-test` service and must execute
*inside* the `search-test` container (its `.env.test` uses docker-network-only hostnames), the same
"needs a `-test` container" shape as identity/restaurant's Postgres-backed tests. Covers all four ordering guards
per event type (`UpsertSnapshot`/`UpdateFields`/`UpsertPizza`/`RemovePizza`/`UpdateToppingPrices`, each with a
stale-redelivery-ignored case), the pizzas-preserved-through-a-field-only-update and
pizzas-preserved-through-a-topping-price-update cases, and the document-missing-404 case for
`UpdateFields`/`UpsertPizza`/`RemovePizza`/`UpdateToppingPrices`'s no-`upsert:`-fallback contract.
`tests/testutil.ES(t)` resets both indices before each test; `RefreshIndex` forces visibility (ES's ~1s
near-realtime refresh would otherwise hide a just-written doc from an immediate assertion).

End-to-end path verified manually against a live stack (elasticsearch, rabbitmq, search-service, search-worker),
including a real OpenCage account (search-service's own key, `OPENCAGE_API_KEY`): `restaurant.launched` messages
published to the `restaurant.events` exchange are indexed by the worker; `GET /search` with a real Hamburg
address resolves via OpenCage, hits the restaurant within its 10km delivery radius (name match, empty-`q`
browse-all, and rating-boosted ordering all confirmed), and a Berlin address (~250km away, outside that radius)
correctly returns no results. Cache behavior confirmed via request timing: a first search from a new address
took ~540ms (real OpenCage call); an identical repeat search took ~5ms (`geocode` index hit, no external call).
Typo/fuzzy matching was also confirmed separately against synthetic data. `search-service` is not host-exposed,
same as every other service in this repo — only reachable through Traefik.

While standing up this stack, `search-service` hit the same Traefik network-IP-pinning bug already documented for
`identity-service` (see the root-level compose infra notes) — Traefik advertised the `internal`-network IP
instead of `backend` despite the correct `traefik.docker.network=backend` label. Notably, restarting/recreating
the `search-service` container itself did **not** fix it this time; only restarting the `traefik` container did.
Worth checking both remedies if this recurs on a third service.
