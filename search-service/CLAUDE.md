# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `search-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, `docs/services/search.md` for the full technical writeup, and `docs/api/search.md` for the full route reference.

## What this service owns

Read-only search over restaurant/pizza data, backed by Elasticsearch and kept fresh by consuming `restaurant.launched`, `restaurant.updated`, `restaurant.pizza_updated`, and `restaurant.topping_prices_updated` off RabbitMQ. It is the only service with no Postgres database, and the only service besides `restaurant-service` that calls OpenCage geocoding — its own independent client/quota, not shared with `restaurant-service`.

Routes (behind Traefik, **no auth** — public by design): `GET /search` (required `house`/`street`/`city`/`postalCode`, optional `q`/`fulfillment`/`tags`/`openNow`/`sort`), `GET /search/restaurant/:id` (direct ES doc lookup by id, no slug route). See `docs/api/search.md` for the full filter/sort reference.

## Commands

Run from inside `search-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer — IS started by compose.yaml (search-worker), unlike every
                       # other service's cmd/worker; without it the index stays permanently empty
go test ./...
go test ./... -run TestName
```
`tests/application/`/`tests/interfaces/` mock `SearchRepository`/`Geocoder`; `tests/infrastructure/elasticsearch/` is integration-style against a real Elasticsearch and must run inside the `search-test` container. No migrations here — no database.

## Testing conventions

`tests/testutil.ES(t)` resets both indices before each real-ES test; call `testutil.RefreshIndex(t, es, index)` after writing, before asserting a search sees it (ES's near-realtime refresh means a just-indexed doc isn't searchable for ~1s otherwise).

## Invariants and gotchas

- **No Postgres, and every route is public** — `/search` has no auth by design.
- **`restaurant.reactivated`/`restaurant.deactivated` aren't consumed** — restaurant-service doesn't publish them yet, so don't add `SearchRepository.Delete` speculatively; it'd have no caller.
- **Parses its own local copy of the event payloads** — never imports restaurant-service's Go code; the two services only agree on the JSON wire contract.
- **A `restaurant.launched` message missing `lat`/`lon` is rejected outright**, going through the consumer's retry/DLQ path — restaurant-service's own checklist invariant guarantees they're always set by launch time, so a missing value means a real contract violation, not a state to tolerate.
- **Every ES write is a Painless-scripted conditional `_update`**, guarded against redelivery reordering by three separate timestamps: the document's own top-level `updatedAt`, each pizza's own `pizzas[i].updatedAt`, and the top-level `toppingPricesUpdatedAt` — each scoped to what it's guarding, since a pizza or topping-price edit never touches the restaurant's own row.
- **`search-worker` is the one `cmd/worker` that gets a `compose.yaml` container** — every other service's worker is deliberately excluded, but without this one the index stays permanently empty and `/search` returns nothing.
