# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Pizza Marketplace: an online pizza ordering platform built as a Go microservices monorepo, following DDD and Clean Architecture. Each service owns its own Postgres database — there is no shared database. Services are fronted by Traefik and communicate asynchronously over RabbitMQ; the outbox pattern is used where cross-service consistency matters.

## Services and their state

| Service | Role | Status |
|---|---|---|
| `identity-service` | Auth, JWT, user/owner/customer registration | implemented (API + worker) |
| `restaurant-service` | Restaurant & menu CRUD, geocoding | implemented (API + worker) |
| `email-service` | Consumes email-related events, sends notifications | implemented (worker only) |
| `search-service` | Search API + Elasticsearch indexing | **not implemented** — `go.mod` only, no code |
| `web-user` | React frontend | **not implemented** — empty placeholder dir |

Don't assume `search-service` or `web-user` have code to read — check before referencing them. The `restaurant.launched` → search-indexing flow described in `README.md` / `docs/architecture.md` is documented but not yet wired up anywhere in Go code.

Each Go service (`identity-service`, `restaurant-service`, `email-service`) follows the same internal layout:

```
cmd/api/main.go             # HTTP entrypoint (identity/restaurant only)
cmd/worker/main.go          # background worker entrypoint (outbox relay / event consumer)
internal/domain/            # entities, repository interfaces, no framework deps
internal/application/       # use cases, orchestration
internal/infrastructure/    # gorm/persistence, messaging (RabbitMQ), redis, auth, geocoder, migrations
internal/interfaces/http/   # Gin handlers, middleware
internal/container/         # manual DI wiring (APIContainer / WorkerContainer, built from a shared base)
internal/shared/            # cross-cutting: event.Event interface, error types
tests/                      # mirrors internal/ structure; integration-style, hits real containers
```

`restaurant-service` additionally has `cmd/worker/bootstrap/` for its worker's app/runner setup.

## Commands

A root `Makefile` wraps the common `go`/`docker compose` commands below (`make up`, `make down`, `make down-v`, `make test-up`, `make test-down`, `make test-identity`/`test-restaurant`/`test-email`/`test`, `make fmt`/`vet`/`lint`) — see it for the exact underlying commands, which also still work directly.

**Local dev environment** (from repo root):
```bash
cp identity-service/.env.example   identity-service/.env
cp restaurant-service/.env.example restaurant-service/.env
cp email-service/.env.example      email-service/.env
docker compose up --build
```
- Gateway: `http://localhost:80`, Traefik dashboard: `http://localhost:8080`, RabbitMQ UI: `http://localhost:15672`
- `docker compose down` to stop, `docker compose down -v` to also wipe volumes
- Dev containers run `air` for live reload (`.air.toml` per service); `compose.yaml` does not define containers for `cmd/worker` — run workers manually (e.g. `go run ./cmd/worker` inside the service, or `air -c .air.toml` pointed at the worker build cmd) when working on outbox/event-consumer code.

**Tests**: `identity-service` and `restaurant-service` tests are integration-style against real Postgres/RabbitMQ/Redis (no DB mocking); `email-service` tests are plain unit tests with mocked collaborators and need no infrastructure.

`compose.test.yaml` (profile `test`) spins up `pg-identity-test`, `pg-restaurant-test`, `redis-test`, `rabbitmq-test`, one-shot `migrate-identity-test`/`migrate-restaurant-test` (golang-migrate, applies that service's migrations), and full app containers `identity-test`/`restaurant-test` (built from each service's Dockerfile `dev` target, code mounted live via volume). Each service's `.env.test` uses docker-network-only hostnames (e.g. `POSTGRES_HOST=pg-restaurant-test`) that aren't resolvable from the host shell, so `go test` for identity/restaurant must run *inside* its `-test` container:
```bash
docker compose -f compose.test.yaml --profile test up -d
docker compose -f compose.test.yaml exec -T restaurant-test sh -c "cd /app && go test ./..."  # or identity-test
cd email-service && go test ./...   # no container needed — plain unit tests, run from host
```
Run a single test: append `-run TestName` to the `go test` invocation. Each of `identity-service`/`restaurant-service` needs its own `.env.test` (not committed) alongside `.env.example`; `email-service` has neither a `compose.test.yaml` entry nor an `.env.test`.

Go runs test packages in parallel by default; `identity-service`'s and `restaurant-service`'s test packages share one live Postgres test DB and truncate tables in setup, so parallel runs can race across packages (spurious "record not found" / duplicate-key errors). If you see that, rerun with `go test -p 1 ./...` to force sequential package execution before assuming a real regression.

**Migrations**: `golang-migrate` SQL files live in `internal/infrastructure/migrations/` per service; the `migrate` CLI is baked into each service's Docker image.

## Architecture notes worth knowing before editing

- **DI is manual, not a framework**: `internal/container/{shared,api,worker}.go` construct dependencies by hand and wire them into `APIContainer` / `WorkerContainer` structs. When adding a new use case or handler, wire it here rather than introducing a DI library.
- **GORM + Postgres** for persistence; repository interfaces live in `internal/domain/<aggregate>/`, implementations in `internal/infrastructure/persistence/`.
- **Outbox pattern (identity-service only)**: `internal/domain/outbox/`, `internal/infrastructure/persistence/outbox.go`, `internal/application/outbox/{worker,relay}.go`. Business writes and the outbox row are created in the same `gorm.Transaction`; a separate poller (`cmd/worker`) claims pending rows with `SELECT ... FOR UPDATE SKIP LOCKED`, publishes to RabbitMQ, and retries with exponential backoff (up to `MaxRetries`) before marking a row `failed`. Only the cross-service-critical `restaurant.initiated` event goes through the outbox; `user.registered` and `email.verification_created` are published directly/best-effort in the same request. `restaurant-service` does not use the outbox pattern — its `PublishEvent`/RabbitMQ publisher exists but currently has no caller.
- **Events**: `internal/shared/event.Event` is the interface producers implement (`GetEventName()`); routing key = event name. Consumers live in the relevant service's worker/messaging layer.
- **Auth**: JWT-based; `identity-service` exposes `GET /auth/verify` as a Traefik forward-auth endpoint (see `traefik.http.middlewares.jwt.forwardauth.*` labels in `compose.yaml`) — other services don't validate JWTs themselves, they trust `X-User-ID`/`X-User-Role` headers Traefik injects after forward-auth succeeds.
- **Routing**: all traffic enters through Traefik on `:80`, path-routed by service (`/auth`, `/users` → identity; `/restaurants` → restaurant; `/search` → search, once it exists).
