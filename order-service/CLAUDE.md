# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `order-service` only. See the root `CLAUDE.md` for monorepo-wide architecture and how this service fits into the event flow, and `docs/services/order.md`/`docs/api/order.md` for the full technical writeup this file summarizes.

## What this service owns

The customer-facing basket (`Cart`) and the `Order` aggregate: adding items, checking out, and driving payment via a separate `payment-service` over gRPC. Owns its own Postgres database and runs the transactional outbox pattern, same as identity-service/restaurant-service. Keeps a local read-model of restaurant/pizza/pricing data fresh by consuming restaurant-service's events off RabbitMQ (never importing restaurant-service's Go types) rather than calling it synchronously — the same pattern search-service uses.

Routes (behind Traefik, no `jwt@docker` label — auth enforced per-route-group in Gin instead, same as search-service):
- `GET`/`POST`/`PATCH`/`DELETE` `/cart`, `/cart/items[/:itemId]` — any authenticated user, no role gate (an `owner` account can place an order too; see `Middleware.Auth` only, no `RequireRole`). No `DELETE /cart` (clear-whole-cart) endpoint exists.
- `POST /orders` — checkout: converts the current cart into an `Order`, revalidates every line against live prices/availability, checks fulfillment support and minimum order, geocodes+validates the delivery radius, then calls payment-service for a real checkout URL. See `internal/application/order/commands/checkout.go`.
- `GET /orders/:id`, `POST /orders/:id/cancel` — customer-or-owner (`m.Auth` only, branch on `X-User-Role` inside the query/command using `FindByIDAndCustomer`/`FindByIDAndRestaurantOwner`, never fetch-then-compare). `GET /orders` — the caller's own orders (customer), cursor-paginated.
- `GET /orders/restaurants/:id`, `POST /orders/:id/ready`, `POST /orders/:id/complete` — owner-only (`m.EnsureOwner`), scoped to the given restaurant ID with the owner check re-verified server-side, never trusted from the path alone.

## Commands

Run from inside `order-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer (restaurant.events/identity.events/payment.events) + outbox relay
go test ./...
go test ./... -run TestName
```
Tests are integration-style against real Postgres (see root `CLAUDE.md` for `compose.test.yaml`); no mocked-DB path for anything touching a transaction. Needs `.env.test` populated like `.env.example` (`POSTGRES_*`, `RABBITMQ_URL`, `OPENCAGE_API_KEY`, `PAYMENT_SERVICE_ADDR`, `FRONTEND_BASE_URL`).

## Testing conventions

Same shape as restaurant-service: `tests/` mirrors `internal/`, real GORM `TestDB` + fixtures. Anything using `db.Transaction`/`WithTx` (`Checkout`, `PaymentSucceededHandler`, `PaymentFailedHandler`) is tested against real Postgres; everything else uses local `Mock*Repository` fakes, including `MockOrderRepository`, which also records each `ListByCustomer`/`ListByRestaurant` call's cursor/limit. The gRPC client and circuit breaker are tested with their own dedicated fakes, no real network or payment-service.

## Invariants and gotchas

- **Order totals are always computed server-side** from the local read-model. `Checkout`'s request body carries no price fields at all, only IDs and quantities — a `decimal.Decimal` field can't carry a `go-playground/validator` numeric tag, so keeping prices off the wire entirely avoids the problem.
- **A cart is scoped to one restaurant at a time** (`carts.customer_id UNIQUE`) — adding an item from a different restaurant than the cart's current one returns `409` rather than silently clearing the cart.
- **The payment-service gRPC call happens outside the checkout transaction** (`internal/application/order/commands/checkout.go`) — a network call shouldn't hold DB locks open; a `payment_id` is recorded with a separate `UPDATE` afterward.
- **The payment saga closes asynchronously, never synchronously**: order-service only learns a payment outcome by consuming `payment.succeeded`/`payment.failed`. Both handlers treat `order.ErrInvalidStatusTransition` as an idempotent no-op — Mollie retries its webhook for up to ~26h, and RabbitMQ redelivery must never double-confirm/double-cancel.
- **Read-model writes are upserts guarded by `updated_at`** (`ON CONFLICT ... WHERE updated_at < excluded.updated_at`, except `customers` which has no `updated_at` and uses plain `DoNothing`) — never trust event delivery order.
- **The generated gRPC `pb` client code** (`internal/infrastructure/payment/pb/`) is copied by hand from payment-service's own `internal/interfaces/grpc/pb/` — no shared Go module between services, kept in sync manually.
- **Owner checks are enforced in the query, never trusted from the path alone** — e.g. `orders.restaurant_id = ? AND restaurants.owner_id = ?`, not a fetch-then-compare.
