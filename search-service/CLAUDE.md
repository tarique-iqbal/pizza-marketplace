# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `search-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, event flow across services, and how this service fits into the rest of the platform. See `docs/services/search-service.md` for the full technical writeup this file summarizes.

## What this service owns

Read-only search over restaurant/menu data, backed by Elasticsearch and kept fresh by consuming `restaurant.launched` off RabbitMQ. It is the only service with no Postgres database, and the only service besides `restaurant-service` that calls OpenCage geocoding — its own independent client/quota, not shared with `restaurant-service`.

This is a deliberately **scoped-down first slice**, not the full originally-planned design: only `restaurant.launched` is consumed; `restaurant.updated`/`reactivated`/`deactivated` and a pizza/topping-owned event have no publisher in `restaurant-service` yet, so the corresponding `SearchRepository` methods (`Delete`, partial-update variants) aren't built — don't add them speculatively, they'd have no caller.

Route (behind Traefik, **no auth** — public by design):
- `GET /search` — `house`/`street`/`city`/`postalCode` all **required** (a full delivery address, `400` if any missing — never raw `lat`/`lon`, a customer shouldn't need to know their own coordinates), `q` optional (empty = browse everything deliverable to that address). Coverage is judged against each restaurant's own indexed `deliveryKm`, not a caller-supplied radius — a restaurant with no delivery configured (`DeliveryType: "none"`, pickup-only) is excluded outright from every address-scoped search, a known gap not addressed in this slice. Ranking: `multi_match` with `fuzziness: "AUTO"` (typo tolerance) across `name`/`pizzas.name`/`pizzas.toppings`, boosted additively (`function_score`, `boost_mode: "sum"`) by `log1p(rating)`.

## Commands

Run from inside `search-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer — IS started by compose.yaml (search-worker), unlike every
                       # other service's cmd/worker; without it the index stays permanently empty
go test ./...
go test ./... -run TestName
```
Testing style is mixed, not uniform like the other three services: `tests/application/`, `tests/interfaces/` mock `SearchRepository`/`Geocoder` and need no infrastructure (matches `email-service`'s approach); `tests/infrastructure/elasticsearch/` is integration-style against a **real** Elasticsearch and must run *inside* the `search-test` container (see root `CLAUDE.md`'s Tests section) — same "docker-network-only `.env.test` hostnames" reason `identity`/`restaurant` need their own `-test` containers. `tests/infrastructure/messaging/` is a pure unit-test exception needing no broker at all (copied pattern from `email-service`'s `rabbitmq_consumer_test.go` — fake `messageSource` + fake `Acknowledger`).

No migrations here — no database, nothing for `golang-migrate` to apply.

## Architecture specifics

- **`internal/domain/index`** is the one domain package (this service isn't an aggregate-owning DDD service the way `restaurant-service` is — it's an index + a geocoding boundary): `IndexedRestaurant`/`IndexedPizza`/`GeoPoint` (plain data, real `json` tags — without them Gin emits Go's capitalized field names), `SearchRepository`/`SearchQuery` (`Location` is a plain `GeoPoint`, not a pointer — every search resolves an address first, no "no location" case to model), `Address`/`Geocoder`, `EventDispatcher`/`EventHandler`/`EventPayload` (same shape as `email-service`'s `domain/email`).
- **`internal/infrastructure/elasticsearch`** holds two unrelated concerns in one package: `SearchRepository` (the `restaurants` index — search/rank) and `CachingGeocoder` (the `geocode` index — a disposable key/value cache, unrelated to search, safe to delete anytime since a cache miss just re-populates it). `EnsureIndex` creates both indices idempotently, called at both API and worker startup. `pizzas` is mapped as a plain object array, not `nested` — trades away precise per-pizza cross-field matching for a simpler `multi_match` (no `nested` query needed).
- **Geocoding is cached, not called on every request**: `CachingGeocoder` wraps `internal/infrastructure/geocoder.OpenCageGeocoder` (search-service's own copy — deliberately not shared with `restaurant-service`'s, matching this repo's per-service-infrastructure-duplication convention already used for the RabbitMQ consumer). Cache key = SHA-256 of the normalized (`lower`, whitespace-collapsed) address. Observed: ~5ms cache hit vs. ~500ms+ OpenCage round-trip.
- **`UpsertSnapshot`** (`internal/application/index/upsert_snapshot.go`) parses `restaurant.launched` into a **local, independent copy** of restaurant-service's `RestaurantLaunchedPayload` shape — never imports restaurant-service's Go code, the two services only agree on the JSON wire contract. Rejects the message outright (goes through the consumer's DLX/retry path) if `lat`/`lon` are missing from the payload — `restaurant-service`'s own checklist invariant guarantees they're always set by launch time, so a missing value here means a real contract violation, not a state to silently tolerate.
- **`search-worker` is the one `cmd/worker` that gets a `compose.yaml` container** — every other service's worker is deliberately excluded (root `CLAUDE.md`), but without this one running the index is permanently empty and `/search` returns nothing, judged a worse default dev experience than an optional worker.
- **Dockerfile builds two separate final-stage targets** (`api`, `worker`), unlike `identity-service`/`restaurant-service`'s Dockerfiles which only build `cmd/api` — search-service didn't inherit that gap since it was built from scratch this session.

## Testing conventions

`tests/testutil.MockSearchRepository`/`MockGeocoder` back the mocked-boundary tests, following this repo's local-fake-struct-plus-compile-time-assertion pattern (`var _ index.X = (*mockX)(nil)`), not a mocking library. `tests/testutil.ES(t)` resets both indices (delete-if-exists, `EnsureIndex`) before each real-ES test — call it at the top of any new integration test in `tests/infrastructure/elasticsearch/`, and call `testutil.RefreshIndex(t, es, index)` after writing before asserting a search sees it (ES's near-realtime refresh means a just-indexed doc isn't searchable for ~1s otherwise).
