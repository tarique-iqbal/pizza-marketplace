# Search Service API

Routes served under `/search` via the Traefik gateway. No auth — public by design.

## Search — `/search`

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/search` | — | Search restaurants available to a given address |
| `GET` | `/search/restaurant/{id}` | — | Get a single restaurant by id |

`GET /search` requires `house`, `street`, `city`, `postalCode` — a full delivery address, not raw `lat`/`lon` coordinates a caller would otherwise have to already know. `400 Bad Request` if any is missing. `q` is optional free text; empty means browse everything available to that address.

By default (no `fulfillment` param), a restaurant matches only if the address falls within its own delivery radius (`deliveryKm`) — a pickup-only restaurant is excluded from a plain search. `fulfillment=pickup` opts into pickup-enabled (`pickup: true`) restaurants instead; `fulfillment=delivery` is equivalent to the default.

`tags` (comma-separated) filters to restaurants carrying every listed tag — AND semantics, not either/or.

`openNow=true` filters to restaurants currently open, evaluated in each restaurant's own local timezone (not the server's) against its weekly opening hours. A restaurant with no resolved timezone is excluded rather than treated as always open.

`sort` replaces the default relevance ordering outright (it doesn't blend with it): `distance` (ascending, nearest first), `minimumOrder` (ascending), or `deliveryTime` (ascending on the restaurant's own fastest-case delivery-time estimate — a pickup-only restaurant with no estimate sorts last). With no `sort`, results rank by text relevance across the restaurant's name and its pizzas' names/toppings (typo-tolerant), boosted additively by rating.

Returns `{"results": [...]}` — an array of restaurant objects under a `results` key, each entry carrying `deliveryKm`/`deliveryTimeMin`/`deliveryTimeMax` only when the restaurant has them set. `500 Internal Server Error` if the address can't be geocoded.

`GET /search/restaurant/{id}` returns a single restaurant object, the same shape as one entry from `GET /search`. `id` must be the restaurant's UUID — there is no slug-based lookup, since search-service already keys every indexed document by id, making an id lookup a direct point fetch rather than a search. `400 Bad Request` for a malformed id, `404 Not Found` if no restaurant with that id is indexed (never launched, or removed).
