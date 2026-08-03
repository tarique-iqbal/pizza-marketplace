# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `restaurant-service` only. See the root `CLAUDE.md` for monorepo-wide architecture and how this service fits into the event flow.

## What this service owns

Restaurant records and their onboarding checklist. It is the only service that talks to `postgres-restaurant` and the OpenCage geocoding API. It does **not** implement the outbox pattern (see root `CLAUDE.md`) — its `cmd/worker` is a plain RabbitMQ *consumer*, not an outbox relay.

Routes (behind Traefik, JWT-protected + owner-only):
- `PATCH /restaurants/:id/address` — updates address, re-geocoding only if it changed (`internal/interfaces/http/handlers/address_handler.go`).
- `PATCH /restaurants/:id/delivery` — updates pickup/delivery-type/radius/fee/minimum-order (`internal/interfaces/http/handlers/delivery_handler.go`).

## Commands

Run from inside `restaurant-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer for identity events — NOT started by docker-compose.yml, run manually
go test ./...
go test ./... -run TestName
```
Tests are integration-style against real Postgres (see root `CLAUDE.md` for `docker-compose.test.yml`); no mocked-DB path. Needs `.env.test` populated like `.env.example` (including `OPENCAGE_API_KEY`).

Migrations (`internal/infrastructure/migrations/*.sql`, golang-migrate) define `restaurants` and `pizza_sizes`. Note `pizza_sizes` has a Go model (`internal/domain/restaurant/pizza_size.go`) and a test fixture but **no repository or application code uses it yet** — schema is ahead of the code.

## Architecture specifics

- **`cmd/worker` here is the `restaurant-worker` from the root architecture diagram**: it consumes `restaurant.initiated` off RabbitMQ (published by `identity-service`'s outbox) and creates the local `Restaurant` row. Wiring: `internal/container/worker.go` registers `events.RestaurantInitiated` against routing key `restaurant.initiated` on an in-process `EventDispatcher` (`internal/application/restaurant/events/dispatcher.go`); `internal/infrastructure/messaging/rabbitmq_consumer.go` pulls messages and calls `dispatcher.Dispatch`. To handle a new inbound event type, add a handler implementing `restaurant.EventHandler` and register it in `container/worker.go` plus add its routing key to `messaging.Exchanges["identity.events"]`.
- **Consumer reliability**: manual ack, QoS prefetch 1, a dead-letter exchange (`restaurant_dlx`) wired to the main queue, and in-process retry via `x-retry-count` header (linear backoff, `MaxRetryAttempts = 3` before the message is discarded). This is a different reliability mechanism from identity-service's outbox — don't conflate the two when reasoning about delivery guarantees.
- **Restaurant onboarding is checklist-driven**: `internal/domain/restaurant/checklist.go` defines five required items (`basic`, `contract`, `address`, `delivery`, `payment`) and `Checklist.IsCompleted()`. `RestaurantInitiated` (the `restaurant.initiated` handler) marks `basic` complete on creation; `UpdateAddress` marks `address` complete; `UpdateDelivery` marks `delivery` complete. `contract` and `payment` have no endpoint/handler yet, so `IsCompleted()` can never return true until those are implemented. **No code currently checks `IsCompleted()` or transitions `Status` to `active`/publishes `restaurant.launched`** — the launch flow described in the root README/architecture doc is not yet implemented here. If asked to implement `payment`, follow the `UpdateAddress`/`UpdateDelivery` pattern (ownership check → mark checklist item → mutate → persist → map response).
- **Geocoding is conditional**: `internal/application/restaurant/commands/update_address.go` only calls `geocoder.GeocodeAddress` (OpenCage, `internal/infrastructure/geocoder/opencage.go`, 5s HTTP timeout) when the incoming `Address` struct differs from the stored one — otherwise it reuses the existing `Lat`/`Lon`. Preserve this comparison if touching address-update logic; it's a deliberate perf/cost optimization (avoids unnecessary paid API calls), not an oversight.
- **Slug generation**: `UpdateAddress.generateUniqueSlug` builds a slug from name+city+street and probes `-2` through `-9` suffixes for collisions, giving up after 9 attempts. It's called on every address update (not just creation), so a restaurant's slug can change when its address changes.
- **Auth model differs from identity-service**: this service does **not** parse JWTs itself. `internal/interfaces/http/middleware/auth_middleware.go` only reads `X-User-ID`/`X-User-Role` headers (injected by Traefik after identity-service's forward-auth check) and 401s if missing — see root `CLAUDE.md`. `RequireRole`/`EnsureOwner` then gate by role. Note: `JWT_SECRET` is present in `.env.example` but is not read by any current middleware here — treat it as unused/vestigial rather than load-bearing.
- **`decimal.Decimal` request fields can't carry numeric validator tags**: `go-playground/validator/v10`'s numeric tags (`gte`, `lte`, `min`, `max`, ...) panic on `shopspring/decimal.Decimal` fields (`Bad field type decimal.Decimal`) — it isn't a `Kind` the validator understands. `UpdateDeliveryRequest.DeliveryFee`/`MinimumOrder` (`internal/application/restaurant/schema.go`) therefore carry no validation tag; the DB's `CHECK (... >= 0)` constraints are the only guard. Follow the same tag-less pattern for any new `decimal.Decimal` request field (e.g. future `payment` amounts).
- **Error convention**: same shared-sentinel pattern as identity-service — `internal/shared/errors/errors.go` (`ErrForbidden`, etc.) plus domain-specific errors in `internal/domain/restaurant/errors.go` (currently just `ErrEmailAlreadyExists`).
- **Observability**: `internal/infrastructure/observability/` provides a Gin middleware that injects a per-request `slog.Logger` (with a generated request ID) into context, retrieved via `logger.FromContext`/`logger.WithContext`. Use this instead of a package-level logger when adding logging in request-scoped code.

## Testing conventions

Same shape as identity-service: `tests/` mirrors `internal/`, `tests/testutil/db.go` gives a real GORM `TestDB` + table truncation, `tests/infrastructure/db/fixtures/*.go` provide `LoadXFixtures(t, db)` helpers (`restaurant_fixture.go` includes restaurants with varying checklist completeness — check there before writing new ones covering onboarding states).
