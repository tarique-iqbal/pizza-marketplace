# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `payment-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, event flow across services, and how this service fits into the rest of the platform. See `docs/services/payment-service.md` for the full technical writeup this file summarizes.

## What this service owns

Payment processing via Mollie, on behalf of any other service in this system. It is the only service with no route-based REST API — its primary surface is **gRPC** (`PaymentService`, internal Docker network only, port `:50051`), and it has exactly one HTTP route, the public Mollie webhook. It owns its own Postgres database (`payment_db`) and, like identity-service/restaurant-service, runs the transactional outbox pattern for everything it publishes.

Deliberately generic: `Payment` is keyed by `subject_type`/`subject_id` (e.g. `"order"` + order-service's own `Order.ID`), not an order-specific concept — this service never imports or knows about order-service's types.

gRPC surface, `PaymentService`:
- `CreatePayment(subject_type, subject_id, restaurant_id, customer_id, amount, currency, redirect_url) → (payment_id, checkout_url, status)` — idempotent on `(subject_type, subject_id)`. Amounts are decimal strings over the wire (`"24.50"`), not native numbers.
- `CancelPayment(payment_id) → (status)` — best-effort; a Mollie `422` (payment already in a final state) is swallowed, not propagated.

HTTP route (behind Traefik, **no auth** — the one deliberately public route this service has):
- `POST /webhooks/mollie` — Mollie's own callback, form-encoded body with a single `id` field. `400` if `id` is missing, `200` on success (including every no-op case below), `500` only if something on this service's own side actually failed (so Mollie retries). Never trusts the POSTed body for payment status — always re-fetches the true status from Mollie's API first.

## Commands

Run from inside `payment-service/`:
```bash
go run ./cmd/api      # gRPC server (:50051) + HTTP server (:8080, webhook only) — same process, two listeners
go run ./cmd/worker   # outbox relay only, no inbound consumer — see below
go test ./...
go test ./... -run TestName
```
Needs `.env.test` populated like `.env.example` (`POSTGRES_*`, `RABBITMQ_URL`, `MOLLIE_API_KEY`, `PUBLIC_BASE_URL`). Regenerate gRPC code after editing the `.proto` file: `make proto` from the repo root (see root `CLAUDE.md` — requires `protoc` plus the `protoc-gen-go`/`protoc-gen-go-grpc` Go plugins on `PATH`).

## Architecture specifics

- **`payment-worker` has no inbound consumer — the only worker in this repo that doesn't.** Every other service's `cmd/worker` pairs an outbox relay with a RabbitMQ consumer in the same process; this one only relays, since payment-service consumes nothing (it only ever publishes `payment.succeeded`/`payment.failed`). `internal/container/worker.go` has no `Consumer`/`Dispatcher` fields, and `cmd/worker/bootstrap/runner.go` starts exactly one goroutine, not two. Don't treat the missing consumer as an oversight when porting this worker's shape elsewhere.
- **`Payment.Status` has exactly one writer**: `HandleMollieWebhook` (`internal/application/payment/commands/handle_mollie_webhook.go`) is the only code path that ever calls `MarkSucceeded`/`MarkFailed`. `CreatePayment`'s own idempotent-replay branch calls `GetStatus` to show a live status/checkout URL but deliberately never mutates the stored `Payment` — keeping "who's allowed to change this enum" to one place.
- **`CreatePayment` has three branches, not the two a naive idempotency check would produce** (`internal/application/payment/commands/create_payment.go`): no existing row → create, call Mollie, attach the gateway reference; an existing row with no gateway reference yet (a prior attempt crashed between the local insert and the Mollie call) → self-heal by calling Mollie now for that same row instead of creating a second one; an existing row that already has a gateway reference → genuine replay, re-fetch live status and checkout URL via `GetStatus` rather than trying to persist a checkout URL locally (it would go stale the moment the payment leaves Mollie's `open` state).
- **`payments.gateway`/`gateway_payment_id` are deliberately gateway-agnostic column names, not `mollie_payment_id`** — added a second gateway later (Stripe, PayPal, …) needs no rename, just a new `PaymentGateway` implementation and a small selection layer in `container.go`; today there's a single `PaymentGateway` field, no registry, since only Mollie exists. `gateway` has no `DEFAULT` in the migration — it's always set explicitly by application code, avoiding the same GORM zero-value-default class of bug already fixed elsewhere in this repo.
- **The webhook never trusts its own POST body.** Mollie's webhook carries only a payment `id`, no signature — `HandleMollieWebhook` calls `gateway.GetStatus(id)` to fetch the true status directly from Mollie's API before doing anything else. This is also why `MollieGateway`'s `GetStatus` returns a `CheckoutURL` (from the same `_links.checkout.href` Mollie's create-response has) alongside status — it's what makes `CreatePayment`'s replay branch above work without persisting anything extra.
- **Idempotent against Mollie's own webhook retries.** Mollie retries a failed webhook delivery for up to ~26 hours (exponential backoff, ~10 attempts) — `HandleMollieWebhook` treats an already-terminal local `Payment` as a no-op (checked before calling `GetStatus` at all), so a late or duplicate redelivery never double-processes a payment.
- **No domain-event/`PullEvents()` machinery on `Payment`**, unlike `Order` in order-service. `payment.succeeded`/`payment.failed` need no snapshot enrichment — every field the event needs already lives directly on the loaded `Payment` aggregate — so `internal/application/payment/events.go`'s payload structs are built straight from it inside `HandleMollieWebhook`, no enricher step.
- **`MollieGateway` (`internal/infrastructure/gateway/mollie_gateway.go`) is hand-rolled `net/http`, no SDK** — matches restaurant-service's own `OpenCageGeocoder` convention for a small external API surface. Its constructor takes `baseURL` as a real parameter (not a test-only hook) specifically so tests can point it at an `httptest` server; production wiring passes the exported `gateway.MollieBaseURL` constant.
- **First gRPC surface in this monorepo.** The canonical `.proto` lives in `api/proto/payment/v1/payment.proto`; generated code is committed to `internal/interfaces/grpc/pb/`. No shared Go module for the generated code — order-service's own gRPC client (`internal/infrastructure/payment/`) copies the generated client stub manually, kept in sync by hand (same per-service-duplication convention this repo uses everywhere else, e.g. the RabbitMQ consumer/publisher, the OpenCage client). Transport is plaintext (`insecure.NewCredentials()` equivalent — no TLS setup at all yet), internal network only, no Traefik exposure for the gRPC port.

## Testing conventions

Same shape as identity-service/restaurant-service: `tests/` mirrors `internal/`, integration-style against real Postgres/RabbitMQ (`tests/testutil/db.go` gives a real GORM `TestDB` + table truncation), no DB mocking. `MollieGateway` itself is tested against `httptest.NewServer`, not the real Mollie API — this doesn't conflict with the real-infra convention above, since that convention is about *this repo's own* infrastructure (Postgres/RabbitMQ), not a third-party API with no local equivalent to run. Application-layer command tests (`CreatePayment`/`CancelPayment`/`HandleMollieWebhook`) use a locally-defined `fakeGateway` implementing `payment.PaymentGateway`, defined directly in each test file rather than a shared mock package (this repo's established local-fake convention). The gRPC server layer (`tests/interfaces/grpc/server_test.go`) is tested by calling `Server.CreatePayment`/`CancelPayment` directly as plain Go method calls, no `bufconn` — the only real risk at that layer is field-translation/error-code mapping, not wire behavior, so a network round-trip through the transport isn't needed to exercise it.
