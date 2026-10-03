# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `restaurant-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, `docs/services/restaurant.md` for the full technical writeup, and `docs/api/restaurant.md` for the full route reference.

## What this service owns

Restaurant records and their onboarding checklist, plus the pizza menu (pizzas, toppings, pricing). It is the only service that talks to `postgres-restaurant` and the OpenCage geocoding API. It implements the transactional outbox pattern (see root `CLAUDE.md`) for **every** event it raises — its `cmd/worker` runs two goroutines: the inbound `restaurant.initiated` consumer, and an outbox relay. There is no best-effort/direct-publish path anywhere in this service.

Routes (behind Traefik, JWT-protected + owner-only unless noted): contact/address/delivery/tags/opening-hours updates, payout-details submission and replacement, pizza CRUD (create is checklist-gated), per-size pricing, and topping pricing, all under `/restaurants/:id/...`. See `docs/api/restaurant.md` for the full per-route reference.

## Commands

Run from inside `restaurant-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # RabbitMQ consumer for identity events (or `air -c .air.worker.toml` for live
                       # reload, matches the dev container) — IS started by compose.yaml (restaurant-worker)
go test ./...
go test ./... -run TestName
```
Tests are integration-style against real Postgres (see root `CLAUDE.md` for `compose.test.yaml`); no mocked-DB path. Needs `.env.test` populated like `.env.example` (including `OPENCAGE_API_KEY`).

Migrations (`internal/infrastructure/migrations/*.sql`, golang-migrate) define `restaurants`, `pizza_sizes`, `payout_details`, `pizzas`, `pizza_prices`, `toppings`, and `topping_prices`.

## Testing conventions

Same shape as identity-service: `tests/` mirrors `internal/`, real GORM `TestDB` + fixtures (varying checklist completeness). `tests/infrastructure/messaging/` needs no DB or broker — pure unit tests against `messaging.Run` with a fake message source (see `docs/services/restaurant.md` for the fake shapes, same as notification-service's own copy).

## Invariants and gotchas

- **`CreatePizza` is the only caller of `Checklist.IsCompleted()`** — pizza creation is gated on full onboarding (`403` otherwise); the restaurant's own status transitions are a separate, ungated mechanism.
- **Every event goes through `DispatchEventsTx`** (the outbox) — there is no direct-publish path anywhere in this service, for any event.
- **`cmd/worker` runs two goroutines** (inbound consumer + outbox relay), and `ensureConnected` checks both `c.conn.IsClosed()` and `c.channel.IsClosed()` before reconnecting — a channel can die from an AMQP exception while the connection stays open.
- **Geocoding is conditional**: `UpdateAddress` only calls OpenCage when the address actually changed, reusing the stored `Lat`/`Lon` otherwise.
- **No numeric validator tags on `decimal.Decimal` request fields** — `go-playground/validator`'s numeric tags panic on that type; the DB's own `CHECK` constraint is the only guard.
- **`PizzaPriceRepository.ReplacePrices` sets `UpdatedAt` manually** before its bulk upsert — GORM's `autoUpdateTime` omits a nil value from the upsert's `VALUES` client-side, which then propagates into `EXCLUDED.updated_at` too, leaving genuinely-updated rows `NULL`.
- **Pizzas are archived, never deleted** — `Pizza.Status` (`available`/`unavailable`/`archived`) is the retirement mechanism, to avoid orphaning a future order-history reference.
- **Auth comes from Traefik headers, not local JWT parsing**: `X-User-ID`/`X-User-Role`, injected after identity-service's forward-auth check. `JWT_SECRET` in `.env.example` is unused/vestigial.
- **`TRUNCATE ... CASCADE` on `restaurants` clears the pizza tables too** in tests — `db.TruncateTables(t, testutil.TableRestaurant)` alone also clears `pizzas`/`pizza_prices`/`topping_prices`/`payout_details`.
