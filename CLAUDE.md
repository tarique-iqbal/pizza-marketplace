# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Pizza Marketplace: an online pizza ordering platform built as a Go microservices monorepo, following DDD and Clean Architecture. Each service owns its own Postgres database — there is no shared database. Services are fronted by Traefik and communicate asynchronously over RabbitMQ; the outbox pattern is used where cross-service consistency matters.

## Services and their state

| Service | Role | Status |
|---|---|---|
| `identity-service` | Auth, JWT, user/owner/customer registration | implemented (API + worker) |
| `restaurant-service` | Restaurant & menu CRUD, geocoding | implemented (API + worker) |
| `notification-service` | Consumes domain events, sends notifications via channel adapters (email today) | implemented (worker only) |
| `search-service` | Search API + Elasticsearch indexing, own geocoder | implemented (API + worker) — scoped-down first slice, see `docs/services/search-service.md` |

This repo holds backend services only — the React frontend (`web-user`) lives in a separate repo.

`search-service` consumes `restaurant.launched`, `restaurant.updated`, `restaurant.pizza_updated`, and
`restaurant.topping_prices_updated` today — `restaurant.reactivated`/`restaurant.deactivated` still have no
publisher in restaurant-service (Part A's `Reactivate`/`Deactivate` remain unimplemented), so don't assume
those are wired up. Check
`docs/services/search-service.md` before assuming search-service coverage beyond that.

Each Go service (`identity-service`, `restaurant-service`, `notification-service`, `search-service`) follows the same internal layout:

```
cmd/api/main.go             # HTTP entrypoint (identity/restaurant/search only)
cmd/worker/main.go          # background worker entrypoint (outbox relay / event consumer)
internal/domain/            # entities, repository interfaces, no framework deps
internal/application/       # use cases, orchestration
internal/infrastructure/    # gorm/persistence, messaging (RabbitMQ), redis, auth, geocoder, migrations
internal/interfaces/http/   # Gin handlers, middleware
internal/container/         # manual DI wiring (APIContainer / WorkerContainer, built from a shared base)
internal/shared/            # cross-cutting: event.Event interface, error types
tests/                      # mirrors internal/ structure; integration-style, hits real containers
```
`search-service` has no database — its `internal/infrastructure/` is Elasticsearch + its own geocoder +
messaging, no `gorm`/`persistence`/`redis`/`auth`/migrations.

`restaurant-service` and `search-service` additionally have `cmd/worker/bootstrap/` for their worker's app/runner setup.

## Commands

A root `Makefile` wraps the common `go`/`docker compose` commands below (`make up`, `make down`, `make down-v`, `make test-up`, `make test-down`, `make test-identity`/`test-restaurant`/`test-notification`/`test`, `make fmt`/`vet`/`lint`) — see it for the exact underlying commands, which also still work directly.

**Local dev environment** (from repo root):
```bash
cp identity-service/.env.example     identity-service/.env
cp restaurant-service/.env.example   restaurant-service/.env
cp notification-service/.env.example notification-service/.env
cp search-service/.env.example       search-service/.env
docker compose up --build
```
- Gateway: `http://localhost:80`, Traefik dashboard: `http://localhost:8080`, RabbitMQ UI: `http://localhost:15672`
- `docker compose down` to stop, `docker compose down -v` to also wipe volumes
- Dev containers run `air` for live reload (`.air.toml` per service); `compose.yaml` does not define a container for `restaurant-service`'s `cmd/worker` — run it manually (`go run ./cmd/worker` inside the service, or `air -c .air.toml` pointed at the worker build cmd) when working on its outbox/consumer code. **`search-worker` and `identity-worker` are the two exceptions** — each gets its own `compose.yaml` container (`air -c .air.worker.toml`): without `search-worker` the search index stays permanently empty and `/search` returns nothing; without `identity-worker` no outbox event ever leaves identity-service, so registration never gets its OTP/welcome email and no restaurant record ever gets created for a new owner — both judged a worse default dev experience than an optional worker.

**Tests**: `identity-service` and `restaurant-service` tests are integration-style against real Postgres/RabbitMQ/Redis (no DB mocking); `notification-service` tests are plain unit tests with mocked collaborators and need no infrastructure. `search-service` is a middle case — most tests mock `SearchRepository`/`Geocoder` like notification-service, but `tests/infrastructure/elasticsearch/` runs integration-style against a real Elasticsearch (no DB, since search-service has none), the same "needs a `-test` container" shape as identity/restaurant.

`compose.test.yaml` (profile `test`) spins up `pg-identity-test`, `pg-restaurant-test`, `redis-test`, `rabbitmq-test`, `elasticsearch-test`, one-shot `migrate-identity-test`/`migrate-restaurant-test` (golang-migrate, applies that service's migrations), and full app containers `identity-test`/`restaurant-test`/`search-test` (built from each service's Dockerfile `dev` target, code mounted live via volume). Each service's `.env.test` uses docker-network-only hostnames (e.g. `POSTGRES_HOST=pg-restaurant-test`, `ELASTICSEARCH_URL=http://elasticsearch-test:9200`) that aren't resolvable from the host shell, so `go test` for identity/restaurant/search must run *inside* its `-test` container:
```bash
docker compose -f compose.test.yaml --profile test up -d
docker compose -f compose.test.yaml exec -T restaurant-test sh -c "cd /app && go test ./..."  # or identity-test, search-test
cd notification-service && go test ./...   # no container needed — plain unit tests, run from host
```
Run a single test: append `-run TestName` to the `go test` invocation. Each of `identity-service`/`restaurant-service`/`search-service` needs its own `.env.test` (not committed) alongside `.env.example`; `notification-service` has neither a `compose.test.yaml` entry nor an `.env.test`.

Go runs test packages in parallel by default; `identity-service`'s and `restaurant-service`'s test packages share one live Postgres test DB and truncate tables in setup, so parallel runs can race across packages (spurious "record not found" / duplicate-key errors). If you see that, rerun with `go test -p 1 ./...` to force sequential package execution before assuming a real regression.

**Migrations**: `golang-migrate` SQL files live in `internal/infrastructure/migrations/` per service; the `migrate` CLI is baked into each service's Docker image.

## Architecture notes worth knowing before editing

- **DI is manual, not a framework**: `internal/container/{shared,api,worker}.go` construct dependencies by hand and wire them into `APIContainer` / `WorkerContainer` structs. When adding a new use case or handler, wire it here rather than introducing a DI library.
- **GORM + Postgres** for persistence; repository interfaces live in `internal/domain/<aggregate>/`, implementations in `internal/infrastructure/persistence/`.
- **Outbox pattern (identity-service and restaurant-service)**: `internal/domain/outbox/`, `internal/infrastructure/persistence/outbox.go`, `internal/application/outbox/{worker,relay}.go` — the same shape in both services (restaurant-service's is a verbatim port). Business writes and the outbox row are created in the same `gorm.Transaction`; a separate poller (`cmd/worker`) claims pending rows with `SELECT ... FOR UPDATE SKIP LOCKED`, publishes to RabbitMQ, and retries with exponential backoff (up to `MaxRetries`) before marking a row `failed`. Both services now outbox every event they raise, with no best-effort publish path left in either: identity-service (`restaurant.initiated`, `user.registered`, `email.verification_created`) and restaurant-service (`restaurant.ready_for_review`, `restaurant.approved`, `restaurant.launched`, `restaurant.updated`, `restaurant.pizza_updated`, `restaurant.topping_prices_updated`).
- **Events**: `internal/shared/event.Event` is the interface producers implement (`GetEventName()`); routing key = event name. Consumers live in the relevant service's worker/messaging layer.
- **Auth**: JWT-based; `identity-service` exposes `GET /auth/verify` as a Traefik forward-auth endpoint (see `traefik.http.middlewares.jwt.forwardauth.*` labels in `compose.yaml`) — other services don't validate JWTs themselves, they trust `X-User-ID`/`X-User-Role` headers Traefik injects after forward-auth succeeds.
- **Routing**: all traffic enters through Traefik on `:80`, path-routed by service (`/auth`, `/users` → identity; `/restaurants` → restaurant; `/search` → search-service, no auth).
