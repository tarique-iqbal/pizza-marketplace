# Order Service API

Routes served under `/cart` and `/orders` via the Traefik gateway.

## Cart — `/cart`

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/cart` | authenticated user | Get the current customer's cart, with live prices and availability |
| `POST` | `/cart/items` | authenticated user | Add an item to the cart (creates the cart on first add) |
| `PATCH` | `/cart/items/{itemId}` | authenticated user | Update an item's quantity |
| `DELETE` | `/cart/items/{itemId}` | authenticated user | Remove an item from the cart |

Auth is any authenticated user — no `customer`-only role gate, so an `owner` or `admin` account can also hold a cart (e.g. an owner ordering from a different restaurant as a customer). Identity comes from Traefik's forward-auth headers, not a role check.

A cart holds items from **one restaurant at a time**. `POST /cart/items` requires `pizzaId`, `sizeId`, `quantity` (`>= 1`), and an optional `extraToppingIds` array; if the cart already has items from a different restaurant, it fails with `409 Conflict` rather than silently clearing the cart. Each id in `extraToppingIds` must be a topping the pizza's *own restaurant* has actually priced as an add-on (`topping_prices`) — not merely a topping that exists in the marketplace-wide catalog — otherwise `404 Not Found`. Adding the same pizza+size+topping combination again increments the existing line's quantity rather than creating a duplicate.

`PATCH /cart/items/{itemId}` takes `{quantity}` (`>= 1`); to remove a line entirely, use `DELETE` rather than setting quantity to `0`. `DELETE /cart/items/{itemId}` returns `204 No Content`; removing a cart's last remaining item deletes the cart row itself, so a customer isn't left locked to an empty cart's restaurant. There is currently no `DELETE /cart` (clear the whole cart) endpoint.

No price is ever stored on a cart line — `GET /cart` resolves every item's current name/price/availability fresh from the local restaurant read-model on every call. A pizza or size that's since been archived, or a topping that's no longer priced, comes back with `available: false` and no `unitPrice`/`lineTotal` rather than being silently dropped, so the customer can see and remove it explicitly. `subtotal` only counts available items. Prices are serialized as fixed 2-decimal strings (e.g. `"12.50"`), not raw JSON numbers, to preserve trailing zeros.

## Orders — `/orders`

| Method | Path | Auth | Notes |
|---|---|---|---|
| `POST` | `/orders` | authenticated user | Checkout: convert the cart into an order and get a payment `checkoutUrl` |
| `GET` | `/orders` | authenticated user (customer) | The caller's own orders, cursor-paginated |
| `GET` | `/orders/:id` | customer or owning restaurant's owner | One order |
| `POST` | `/orders/:id/cancel` | customer or owning restaurant's owner | Cancel a `pending` order |
| `GET` | `/orders/restaurants/:id` | owner | A restaurant's orders, cursor-paginated |
| `POST` | `/orders/:id/ready` | owner | Mark a `confirmed` order ready |
| `POST` | `/orders/:id/complete` | owner | Complete a `ready` or `confirmed` order |

`POST /orders` (checkout) converts the customer's current cart into an order: revalidates every line against live prices/availability (`409 Conflict` if any item is no longer available), checks the restaurant supports the requested fulfillment method (`409 Conflict` otherwise) and that the subtotal meets its minimum order (`422` otherwise), geocodes and validates the delivery address against the restaurant's delivery radius for delivery orders (`422` outside the radius, `503` if geocoding itself is unavailable), creates the order, clears the cart, and calls payment-service's `CreatePayment` over gRPC (through a circuit breaker) to get back a real Mollie checkout URL (`503` if payment-service is unreachable or the breaker is open). The customer is expected to redirect to `checkoutUrl` to complete payment; the order stays `pending` until payment-service's `payment.succeeded`/`payment.failed` event (consumed asynchronously by this service's worker) confirms or cancels it — there is no synchronous "did it work" beyond getting a checkout URL back.

Request body: `{fulfillment: "delivery" | "pickup", deliveryAddress?: {house, street, postalCode, city}, saveAddress?}` — `deliveryAddress` is required when `fulfillment` is `"delivery"`, and an empty cart fails with `422`. There is no phone field: the order takes the customer's own phone from the profile mirror, and a delivery order fails with `422` if the customer has none saved. Response: `{orderId, checkoutUrl}`.

`GET /orders/:id` and `POST /orders/:id/cancel` accept either the order's own customer or the owner of the restaurant it belongs to — the only role check is `m.Auth` (any authenticated user), and the command itself branches on `X-User-Role` to pick the right ownership-scoped repository lookup (`FindByIDAndCustomer` vs `FindByIDAndRestaurantOwner`). Neither "doesn't exist" nor "exists but you don't own it" is distinguishable from outside — both return `403 Forbidden`.

`GET /orders` returns the caller's own orders (customer), newest first, cursor-paginated: `?cursor=&limit=` (default `limit` 20, max 100). `GET /orders/restaurants/:id` is the owner-facing equivalent, scoped to one restaurant — an owner running multiple restaurants sees only that restaurant's orders, with ownership re-verified server-side rather than trusted from the URL. Both return `{orders: [...], nextCursor?: "..."}` — `nextCursor` is present only when there are more results; pass it back as `?cursor=` to fetch the next page. The cursor is an opaque, forward-only token (encodes the last row's `placedAt`/`id`) — it isn't a page number and can't be decremented.

`POST /orders/:id/ready` (`confirmed → ready`) and `POST /orders/:id/complete` (`confirmed` or `ready` → `completed`) are owner-only. `MarkReady` is optional, not a prerequisite for `Complete` — an order can be completed directly from `confirmed` if the intermediate "ready" step was never recorded. Either returns `409 Conflict` on an invalid transition (e.g. completing an order that's still `pending`).

`POST /orders/:id/cancel` only succeeds while the order is still `pending`. Before cancelling, it re-checks the *real* payment status directly with payment-service (not the local, possibly-stale order status) — if the payment already succeeded there, it returns `409 Conflict` rather than cancelling an order whose money was already captured (this system has no refund mechanism yet). A payment that already failed is still allowed to cancel through. On success it also best-effort cancels the payment at the gateway (failure there is logged, not surfaced — the order stays cancelled regardless).

`POST` returns `201 Created` (checkout only); `GET` returns `200 OK`; `PATCH` returns `200 OK`; `DELETE` returns `204 No Content`; the three lifecycle `POST` endpoints (`ready`/`complete`/`cancel`) return `200 OK` with the updated order.
