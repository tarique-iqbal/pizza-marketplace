# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `payment-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, event flow across services, and how this service fits into the rest of the platform. See `docs/services/payment.md` for the full technical writeup this file summarizes.

## What this service owns

Payment processing via Mollie, on behalf of any other service in this system. It is the only service with no route-based REST API — its primary surface is **gRPC** (`PaymentService`, internal Docker network only, port `:50051`, canonical spec at `api/proto/payment/v1/payment.proto`), and it has exactly one HTTP route, the public Mollie webhook. It owns its own Postgres database (`payment_db`) and, like identity-service/restaurant-service, runs the transactional outbox pattern for everything it publishes.

Deliberately generic: `Payment` is keyed by `subject_type`/`subject_id` (e.g. `"order"` + order-service's own `Order.ID`), not an order-specific concept — this service never imports or knows about order-service's types.

gRPC surface, `PaymentService`:
- `CreatePayment(subject_type, subject_id, restaurant_id, customer_id, amount, currency, redirect_url) → (payment_id, checkout_url, status)` — idempotent on `(subject_type, subject_id)`. Amounts are decimal strings over the wire (`"24.50"`), not native numbers.
- `CancelPayment(payment_id) → (status)` — best-effort; a Mollie `422` (payment already in a final state) is swallowed, not propagated.
- `GetPaymentStatus(payment_id) → (status)` — local status read, no Mollie call.

HTTP route (behind Traefik, **no auth** — the one deliberately public route this service has):
- `POST /webhooks/mollie` — Mollie's own callback, form-encoded body with a single `id` field. `400` if `id` is missing, `200` on success (including every no-op case below), `500` only if something on this service's own side actually failed (so Mollie retries).

## Commands

Run from inside `payment-service/`:
```bash
go run ./cmd/api      # gRPC server (:50051) + HTTP server (:8080, webhook only) — same process, two listeners
go run ./cmd/worker   # outbox relay only, no inbound consumer — see below
go test ./...
go test ./... -run TestName
```
Needs `.env.test` populated like `.env.example` (`POSTGRES_*`, `RABBITMQ_URL`, `MOLLIE_API_KEY`, `PUBLIC_BASE_URL`). Regenerate gRPC code after editing the `.proto` file: `make proto` from the repo root (see root `CLAUDE.md` — requires `protoc` plus the `protoc-gen-go`/`protoc-gen-go-grpc` Go plugins on `PATH`).

## Testing conventions

Same shape as identity-service/restaurant-service: `tests/` mirrors `internal/`, integration-style against real Postgres/RabbitMQ, no DB mocking. `MollieGateway` and the gRPC server layer each use their own local fakes, no live Mollie API and no `bufconn`.

## Invariants and gotchas

- **`payment-worker` has no inbound consumer, by design** — the only worker in this repo that doesn't. It only relays its own outbox, since this service consumes nothing (it only ever publishes `payment.succeeded`/`payment.failed`).
- **`HandleMollieWebhook` is the only writer of `Payment.Status`** — `CreatePayment`'s replay branch calls `GetStatus` to show a live status but never mutates the stored row.
- **`CreatePayment` has three branches**: no existing row → create and call Mollie; an existing row with no gateway reference yet (a prior attempt crashed mid-flight) → self-heal by calling Mollie for that same row; an existing row with a gateway reference → genuine replay, re-fetch live status rather than persisting a checkout URL that would go stale.
- **The webhook never trusts its own POST body** — it carries only a payment `id`, no signature, so `HandleMollieWebhook` always re-fetches the true status from Mollie's API first.
- **An already-terminal payment is a no-op on redelivery** — Mollie retries a failed webhook for up to ~26 hours, so this keeps a late or duplicate delivery from double-processing.
- **The generated gRPC `pb` code is hand-copied into order-service**, kept in sync manually — no shared Go module between services.
- **`PaymentRepository.FindByID` returns `(nil, nil)` on a miss**, not an error sentinel — unlike order-service's equivalent lookups, which return `apperr.ErrNotFound`.
