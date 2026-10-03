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
| `search-service` | Search API + Elasticsearch indexing, own geocoder | implemented (API + worker) — scoped-down first slice, see `docs/services/search.md` |
| `order-service` | Cart + order placement, own geocoder | implemented (API + worker) — cart and checkout both live, calling payment-service over gRPC, see `docs/services/order.md` |
| `payment-service` | Payment processing via Mollie, on behalf of any other service | implemented (gRPC API + worker) — order-service is its first caller, see `docs/services/payment.md` |
| `customer-service` | Customer profile (phone) + saved delivery addresses | implemented (API + worker) — addresses are created only from `order.address_saved`, see `docs/services/customer.md` |

This repo holds backend services only — the React frontend (`web-user`) lives in a separate repo.

`search-service` consumes `restaurant.launched`, `restaurant.updated`, `restaurant.pizza_updated`, and
`restaurant.topping_prices_updated` today — `restaurant.reactivated`/`restaurant.deactivated` still have no
publisher in restaurant-service (Part A's `Reactivate`/`Deactivate` remain unimplemented), so don't assume
those are wired up. Check
`docs/services/search.md` before assuming search-service coverage beyond that.

Each Go service follows the same internal layout (search-service and notification-service have small deviations, noted below):

```
cmd/api/main.go             # HTTP entrypoint (all except notification-service, worker only)
cmd/worker/main.go          # background worker entrypoint (outbox relay / event consumer)
internal/domain/            # entities, repository interfaces, no framework deps
internal/application/       # commands, queries, orchestration
internal/infrastructure/    # gorm/persistence, messaging (RabbitMQ), redis, auth, geocoder, migrations
internal/interfaces/http/   # Gin handlers, middleware
internal/container/         # manual DI wiring (APIContainer / WorkerContainer, built from a shared base)
internal/shared/            # cross-cutting: event.Event interface, error types
tests/                      # mirrors internal/ structure; integration-style, hits real containers
```
`search-service` has no database — its `internal/infrastructure/` is Elasticsearch + its own geocoder +
messaging, no `gorm`/`persistence`/`redis`/`auth`/migrations.

`restaurant-service`, `search-service`, `order-service`, `payment-service`, and `customer-service` additionally have `cmd/worker/bootstrap/` for their worker's app/runner setup; `identity-service` and `notification-service` don't.

## Commands

A root `Makefile` wraps the common `go`/`docker compose` commands below (`make up`, `make down`, `make down-v`, `make test-up`, `make test-down`, one `make test-<service>` per service, `make test` to run them all, `make fmt`/`vet`/`lint`) — see it for the exact underlying commands, which also still work directly.

**Local dev environment** (from repo root):
```bash
cp .env.example                      .env
cp identity-service/.env.example     identity-service/.env
cp restaurant-service/.env.example   restaurant-service/.env
cp notification-service/.env.example notification-service/.env
cp search-service/.env.example       search-service/.env
cp order-service/.env.example        order-service/.env
cp payment-service/.env.example      payment-service/.env
cp customer-service/.env.example     customer-service/.env
docker compose up --build
```
- Gateway: `http://localhost:80`, Traefik dashboard: `http://localhost:8080`, RabbitMQ UI: `http://localhost:15672`
- `docker compose down` to stop, `docker compose down -v` to also wipe volumes
- Dev containers run `air` for live reload (`.air.toml` per service). Every service with a `cmd/worker` (`identity-service`, `restaurant-service`, `search-service`, `order-service`, `payment-service`, `customer-service`) gets its own `compose.yaml` container (`identity-worker`, `restaurant-worker`, `search-worker`, `order-worker`, `payment-worker`, `customer-worker`, each `air -c .air.worker.toml`) and starts by default — none need to be run manually. Without it:
  - `identity-worker`: no outbox event leaves identity-service (no OTP/welcome email, no restaurant row for new owners).
  - `restaurant-worker`: `restaurant.initiated` is never consumed (no restaurant row), and restaurant-service's own outbox events never leave either.
  - `search-worker`: the search index stays permanently empty, `/search` returns nothing.
  - `order-worker`: the local read-model never syncs, orders never confirm/cancel in response to payment events, its own outbox never drains.
  - `payment-worker`: `payment.succeeded`/`payment.failed` never leave payment-service (this worker has no inbound consumer, it only relays its own outbox).
  - `customer-worker`: no customer profile row or saved address is ever created, `customer.phone_updated` never leaves.

**Tests**: `identity-service`, `restaurant-service`, `order-service`, `payment-service`, and `customer-service` tests are integration-style against real Postgres/RabbitMQ (identity also Redis; no DB mocking). `notification-service` tests are plain unit tests with mocked collaborators and need no infrastructure. `search-service` is a middle case — most tests mock `SearchRepository`/`Geocoder` like notification-service, but `tests/infrastructure/elasticsearch/` runs integration-style against a real Elasticsearch (no DB, since search-service has none).

`compose.test.yaml` includes one `compose/<service>-test.yaml` per tested service (everything but notification-service), each spinning up that service's own Postgres/Redis/Elasticsearch test container(s), a one-shot `migrate-<service>-test` container, and the full app container itself (built from the service's Dockerfile `dev` target, code mounted live via volume). Each service's `.env.test` uses docker-network-only hostnames that aren't resolvable from the host shell, so `go test` must run *inside* its own `-test` container:
```bash
docker compose -f compose.test.yaml --profile test up -d
docker compose -f compose.test.yaml exec -T restaurant-test sh -c "cd /app && go test ./..."  # or identity-test, order-test, payment-test, customer-test, search-test
cd notification-service && go test ./...   # no container needed — plain unit tests, run from host
```
Run a single test: append `-run TestName` to the `go test` invocation. Every tested service needs its own `.env.test` (not committed) alongside `.env.example`; `notification-service` has neither a `compose.test.yaml` entry nor an `.env.test`. `--profile <service>-test` instead of `--profile test` starts just that one service's test stack.

Go runs test packages in parallel by default; each integration-tested service's test packages share one live Postgres test DB and truncate tables in setup, so parallel runs can race across packages (spurious "record not found" / duplicate-key errors). If you see that, rerun with `go test -p 1 ./...` to force sequential package execution before assuming a real regression.

**Migrations**: `golang-migrate` SQL files live in `internal/infrastructure/migrations/` per service; the `migrate` CLI is baked into each service's Docker image.

## Invariants and gotchas

- **Outbox pattern** (identity-service, restaurant-service, order-service, payment-service, and customer-service — byte-identical shape in all 5): `internal/domain/outbox/`, `internal/infrastructure/persistence/outbox.go`, `internal/application/outbox/{worker,relay}.go`. The business write and the outbox row are created in the same `gorm.Transaction`; a separate poller (`cmd/worker`) claims pending rows with `SELECT ... FOR UPDATE SKIP LOCKED` (also reclaiming `processing` rows whose lease expired), publishes to RabbitMQ, and retries with exponential backoff before marking a row `failed`. All 5 outbox every event they raise, no best-effort publish path left in any of them. `payment-worker` is the exception among these: it has no inbound consumer, it only relays its own outbox.
- **Auth/JWT forward-auth**: `identity-service` exposes `GET /auth/verify` as a Traefik forward-auth endpoint (`traefik.http.middlewares.jwt.forwardauth.*` labels in `compose.yaml`) — other services don't validate JWTs themselves, they trust the `X-User-ID`/`X-User-Role` headers Traefik injects after forward-auth succeeds.
- **Routing map**: all traffic enters through Traefik on `:80`, path-routed by service (`/auth`, `/users` → identity; `/restaurants` → restaurant; `/search` → search-service, no auth; `/customers` → customer-service, JWT-protected).
- **Compose file split**: root `compose.yaml`/`compose.test.yaml` are just `name:` + an `include:` list; real service definitions live in `compose/base(-test).yaml` (Traefik/RabbitMQ, always active) plus one `compose/<service>(-test).yaml` per service. Relative paths inside an included file resolve against that file's own directory, not the repo root, so each uses `../<service>`/`context: ..`. Every service carries its own profile name plus a shared `"all"`/`"test"` profile (e.g. `["identity", "all"]`); passing `--profile <name>` on the CLI replaces `COMPOSE_PROFILES` entirely rather than adding to it. `order-service` hard-`depends_on: payment-service`, so payment's services also carry the `"order"` profile — Compose errors on an unresolvable `depends_on` reference rather than pulling a missing profile in automatically.
