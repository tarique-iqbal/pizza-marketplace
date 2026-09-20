# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `customer-service` only. See the root `CLAUDE.md` for monorepo-wide architecture and how this service fits into the event flow, and `docs/services/customer.md`/`docs/api/customer.md` for the full technical writeup this file summarizes.

## What this service owns

A customer's own profile data (a phone number) and their saved delivery addresses. Name and email are not owned here: they are a read-only mirror of identity-service's `User`, kept fresh by consuming `user.registered`. Owns its own Postgres database. Its `cmd/worker` runs an inbound consumer and an outbox relay side by side, like order-service.

Routes (behind Traefik with the `jwt@docker` label, then `Middleware.Auth` in Gin — any authenticated user, no role gate):
- `GET /customers/me` — own profile.
- `PATCH /customers/me/phone` — `{phone}` only (`required,max=32`), publishes `customer.phone_updated` in the same transaction.
- `GET /customers/me/addresses` — saved addresses, default first. Bare JSON array, no pagination.
- `DELETE /customers/me/addresses/:id` — `204`; `404` if it doesn't exist or isn't the caller's.
- `POST /customers/me/addresses/:id/default` — makes it the default, returns the address.

There is deliberately **no `POST` to create an address**: addresses only arrive through the `order.address_saved` event (see below), so a client can never submit address text that didn't pass a real checkout.

## Commands

Run from inside `customer-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer + outbox relay (or `air -c .air.worker.toml`, the `customer-worker` container)
go test ./...
go test ./... -run TestName
```
Tests are integration-style against real Postgres and RabbitMQ (see root `CLAUDE.md` for `compose.test.yaml`; from the repo root, `make test-customer`). Needs `.env.test` populated like `.env.example`.

## Architecture specifics

- **`cmd/worker` runs two goroutines side by side** (`cmd/worker/bootstrap/runner.go`): a RabbitMQ consumer and the outbox relay, each with its own panic recovery. The consumer's `Exchanges` map (`internal/infrastructure/messaging/rabbitmq_consumer.go`) binds `customer_queue` to `identity.events` (`user.registered`) and `order.events` (`order.address_saved`). Handlers are registered by literal routing-key string in `internal/container/worker.go`. Manual ack, prefetch 1, in-process retry via `x-retry-count` (`MaxRetryAttempts = 3`), then the message goes to `customer_queue.dlq` via `customer_dlx`.
- **`customers` is an event mirror plus one owned column**: `UserRegistered` upserts id/email/first name/last name with `ON CONFLICT DO NOTHING` (the event fires once per user, so there is no newer version to reconcile). `phone` is the only column written by this service. A `PATCH` that lands before the customer's own `user.registered` is consumed returns `404` until the row exists.
- **Address creation is event-driven and content-deduplicated**: the `AddressSaved` handler (`internal/application/customer/handlers/address_saved.go`) checks `AddressRepository.ExistsForCustomer` on `(customer_id, house, street, city, postal_code)` and does nothing (logged at `Warn`) if it matches, so a client can send `saveAddress: true` on every checkout without creating duplicates. The first address a customer gets becomes their default (`len(ListByCustomer) == 0`).
- **One default address per customer, enforced by the database**: partial unique index `uq_customer_addresses_default (customer_id) WHERE is_default`. `SetDefault` (`internal/application/address/commands/set_default.go`) first loads the address with `FindByID(id, customerID)` for the ownership check (the repository's own `SetDefault(id)` is not customer-scoped), then runs `UnsetDefault` and `SetDefault` inside one `db.Transaction`. Deleting the default address does not promote another one.
- **Ownership is enforced in the query, never trusted from the path**: `Delete` and `FindByID` filter on `customer_id`, and both return `404` for "not yours" and "doesn't exist" alike.
- **Outbox**: `internal/domain/outbox/`, `internal/infrastructure/persistence/outbox.go`, `internal/application/outbox/{worker,relay}.go`. `UpdatePhone` writes the `customer.phone_updated` row in the same transaction as the phone update. `FetchAndMarkProcessing` claims `pending` rows and also reclaims `processing` rows whose `locked_until` lease has expired.
- **Address shape matches order-service's `AddressInput` exactly** (`house`, `street`, `city`, `postalCode`, camelCase JSON) so a frontend can copy a saved address into a checkout request with no mapping. No lat/lon and no geocoder here: validity is inherited from the checkout that produced the event, and order-service re-geocodes on every order anyway.
- **Error convention**: shared sentinels in `internal/shared/errors` (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrConflict`, `ErrInvalid`); `response.HandleError` owns the HTTP status for each. Wrapping a sentinel with a prefix string (`"failed to delete address: %w"`) is the repo's convention.
- **Auth model**: does not parse JWTs. `Middleware.Auth` reads the `X-User-ID`/`X-User-Role` headers Traefik injects after identity-service's forward-auth check, and returns `401` if either is missing.
- **Saved payment methods and notification preferences are not part of this service yet**: payment-service has no saved-card concept today, and notification-service has a single channel.

## Testing conventions

`tests/` mirrors `internal/`. `tests/testutil/db.go` gives a real GORM `TestDB` with `TruncateTables`; `tests/infrastructure/db/fixtures/` has `LoadCustomerFixtures`/`LoadAddressFixtures`. Anything using `db.Transaction`/`WithTx` (`UpdatePhone`, `SetDefault`) runs against real Postgres. Application-layer consumers with no transaction (`List`, `Delete`, the `AddressSaved` and `UserRegistered` handlers) use `tests/testutil.MockAddressRepository`/`MockCustomerRepository`. HTTP handlers are tested end to end in `tests/interfaces/http/handlers/`: `main_test.go` builds the real router with `routes.SetupRoutes`, real commands and queries, and the real `Middleware`, and requests carry `X-User-ID`/`X-User-Role` headers, so route registration and auth are under test.
