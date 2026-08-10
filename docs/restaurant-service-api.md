# Restaurant Service API

Routes served under `/restaurants` via the Traefik gateway.

## Restaurants — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/contact` | JWT | Update contact info (email, phone, website) |
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |
| `PATCH` | `/restaurants/{id}/delivery` | JWT | Update delivery settings (pickup, delivery type, radius, fee, minimum order) |
| `POST` | `/restaurants/{id}/payout-details` | JWT | Submit new payout bank details (account holder, IBAN, BIC, bank name) for verification |
| `PUT` | `/restaurants/{id}/payout-details` | JWT | Replace the pending payout submission (all four fields required) |
| `PATCH` | `/restaurants/{id}/opening-hours` | JWT | Replace the weekly opening hours (full replace, one or more `{open, close}` ranges per weekday) |
| `GET` | `/restaurants/{id}/pizzas` | JWT | List the restaurant's pizzas (excludes `archived`) |
| `POST` | `/restaurants/{id}/pizzas` | JWT | Create a pizza — requires full onboarding checklist complete |
| `PUT` | `/restaurants/{id}/pizzas/{pizzaId}` | JWT | Replace a pizza's own fields (name, image, vegetarian flag, status, sort order, default toppings) |
| `PUT` | `/restaurants/{id}/pizzas/{pizzaId}/prices` | JWT | Set per-size prices (toggle: sizes omitted become inactive, not deleted) |
| `PUT` | `/restaurants/{id}/topping-prices` | JWT | Set the restaurant's own price for extra toppings a customer can add at order time (upsert, additive) |

`PATCH /:id/contact` requires `email` and `phone`; `website` is optional and cleared when omitted. `phone` accepts digits with an optional leading `+`, spaces, hyphens, and parentheses (at least 6 digits) — a loose format check, not strict E.164 validation.

`POST /:id/payout-details` never overwrites data in place — it creates a new `pending` payout record. If one is already pending, the call fails with `409 Conflict`. An existing `active` record is never touched by a new submission; only a future review-promotion step changes it. Promoting a `pending` record to `active` — and flipping the prior `active` record to `superseded` — happens via a review step that isn't implemented yet. Only the `active` record is ever paid out.

`PUT /:id/payout-details` replaces the `pending` record's fields in place — allowed because a `pending` submission hasn't been reviewed yet, so there's nothing to lose by overwriting it (unlike `active`/`superseded` rows, which are never mutated). All four fields are required. `404 Not Found` if there is no `pending` record to replace — including when the restaurant only has an `active` record, since this never touches that. Deliberately `PUT`, not `PATCH` like `address`/`delivery`: the request body is the full resulting state (not a partial diff), so `PUT`'s create-or-replace-at-known-URI contract fits; it stays conditional on `pending` existing rather than unconditionally overwriting, which is why it's not treated as a strict "resource now equals this body no matter what."

`PATCH /:id/opening-hours` takes each weekday (`monday`...`sunday`) as a list of `{open, close}` ranges in `HH:MM` 24-hour format. An empty or omitted list means closed that day. Each range must stay within a single calendar day (`close` must be later than `open` — no crossing midnight in one range). A place open past midnight represents that as two entries: e.g. `monday: [{open: "15:00", close: "23:59"}]` and `tuesday: [{open: "00:00", close: "03:00"}, ...]` for the carryover, rather than one `monday` range crossing into Tuesday. Multiple ranges per day are supported natively (e.g. a lunch/dinner split), not just the overnight case.

The pizza menu is a separate resource from the restaurant itself — `GET /:id/pizzas` (and the other pizza endpoints) never appear nested inside the restaurant's own response body.

`POST /:id/pizzas` requires the restaurant's onboarding checklist to be fully complete (all of `contact`, `address`, `delivery`, `payment`, `openinghours`, plus `basic`); returns `403 Forbidden` otherwise. `name` is required; `image`, `isVegetarian`, `status` (`available`/`unavailable`/`archived`, defaults `available`), `sortOrder`, and `toppingIds` are all optional. There is no `description` field. `toppingIds`, if given, sets the pizza's **default** toppings (the ones it comes with, already included in its own price) — each just has to exist in the `toppings` catalog, with no duplicates; unlike an early version of this design, a default topping does **not** need a `topping_prices` entry, since `topping_prices` is a different concept entirely (see below).

`PUT /:id/pizzas/{pizzaId}` replaces a pizza's own fields (`name`, `image`, `isVegetarian`, `status`, `sortOrder`, `toppingIds`) — no checklist gate. `isVegetarian`/`status` keep their current value when omitted from the request rather than resetting, and so does `toppingIds` — **omitting the key entirely leaves the pizza's default toppings untouched; sending `toppingIds: []` explicitly clears them**. When `toppingIds` is sent (empty or not), it's a full replace: the pizza's defaults become exactly that list, same existence/no-duplicate validation as creation. There is no `DELETE` endpoint for pizzas — retiring one from the menu is `status: archived` via this endpoint instead, so a pizza's price history is never destroyed. `status: unavailable` is a lighter, reversible pause (e.g. temporarily out of stock); `GET /:id/pizzas` excludes `archived` pizzas but still shows `unavailable` ones.

`PUT /:id/pizzas/{pizzaId}/prices` takes the full desired set of active `{sizeId, price}` pairs. Sizes included become active at the given price; sizes with an existing price that are omitted from the request become inactive (`isActive: false`) rather than being deleted — the price is preserved so re-enabling a size later doesn't require re-entering it. `404 Not Found` if a `sizeId` doesn't exist; `409 Conflict` on a duplicate `sizeId` within the same request.

Toppings (`toppings`) are a global, marketplace-wide catalog — name, description, whether it's vegetarian — shared by every restaurant, the same way `pizza_sizes` is, and carry no price of their own. **`pizzas.toppings` (a pizza's defaults, set via `toppingIds` above) and `topping_prices` (below) are unrelated concerns**, not one gating the other: a default topping is already priced into the pizza itself, the same way "mozzarella" isn't a line item on a Margherita's menu price. `topping_prices` is instead what a restaurant charges for a topping a *customer* adds on top at order time — a per-restaurant add-on price list, relevant to a future ordering flow, not to what a pizza comes with by default.

`PUT /:id/topping-prices` is how a restaurant sets its own add-on price for a topping, stored per `(restaurantId, toppingId)`. The request body is the full desired set of `{toppingId, extraPrice}` pairs, but this endpoint is an **upsert, not a full replace**: toppings included are created or re-priced, but toppings priced earlier and omitted from the request stay priced and untouched — there's currently no way to un-price a topping through this endpoint. `extraPrice` must be between `1` and `3` inclusive — `422 Unprocessable Entity` otherwise. `404 Not Found` if a `toppingId` doesn't exist; `409 Conflict` on a duplicate `toppingId` within the same request. Returns the restaurant's complete current set of topping prices (not just the ones just sent).

`POST` returns `201 Created`. `PUT`/`PATCH` return `200 OK`.
