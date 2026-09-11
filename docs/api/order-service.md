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

| Method | Path | Auth | Status |
|---|---|---|---|
| `POST` | `/orders` | authenticated user | Live |

`POST /orders` (checkout) converts the customer's current cart into an order: revalidates every line against live prices/availability (`409 Conflict` if any item is no longer available), checks the restaurant supports the requested fulfillment method (`409 Conflict` otherwise) and that the subtotal meets its minimum order (`422` otherwise), geocodes and validates the delivery address against the restaurant's delivery radius for delivery orders (`422` outside the radius, `503` if geocoding itself is unavailable), creates the order, clears the cart, and calls payment-service's `CreatePayment` over gRPC (through a circuit breaker) to get back a real Mollie checkout URL (`503` if payment-service is unreachable or the breaker is open). The customer is expected to redirect to `checkoutUrl` to complete payment; the order stays `pending` until payment-service's `payment.succeeded`/`payment.failed` event (consumed asynchronously by this service's worker) confirms or cancels it — there is no synchronous "did it work" beyond getting a checkout URL back.

Request body: `{fulfillment: "delivery" | "pickup", deliveryAddress?: {house, street, postalCode, city}, contactPhone?}` — `deliveryAddress` is required when `fulfillment` is `"delivery"`, and an empty cart fails with `422`. Response: `{orderId, checkoutUrl}`.

`POST` returns `201 Created`; `GET` returns `200 OK`; `PATCH` returns `200 OK`; `DELETE` returns `204 No Content`.
