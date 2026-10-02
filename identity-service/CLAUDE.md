# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `identity-service` only. See the root `CLAUDE.md` for monorepo-wide architecture, event flow across services, and how this service fits into the rest of the platform.

## What this service owns

Auth, JWT issuance, and user (owner/customer) registration. It is the only service that talks to `postgres-identity` and `redis`. It implements the outbox pattern (see root `CLAUDE.md`) for every event it raises (`restaurant.initiated`, `user.registered`, `email.verification_created`) — same scope as `restaurant-service`'s own outbox, no best-effort publish path left in either service.

Routes (all behind Traefik, see `internal/interfaces/http/routes/`):
- `POST /auth/email/verify` — request an OTP email verification code
- `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`
- `GET /auth/verify` — **not a client-facing route**: this is Traefik's forward-auth target (`traefik.http.middlewares.jwt.forwardauth.*` in root `compose.yaml`). It validates the JWT and returns `X-User-ID`/`X-User-Role` headers, which Traefik then injects into the downstream request to `restaurant-service` etc. Other services trust these headers rather than parsing JWTs themselves.
- `POST /users/owners`, `POST /users/customers` — registration (public)
- `GET /users/:id` — JWT-protected
- `GET /health/live`, `GET /health/ready` — liveness/readiness probes (`internal/interfaces/http/routes/health_routes.go`)

## Commands

Run from inside `identity-service/`:
```bash
go run ./cmd/api      # HTTP API (or `air -c .air.toml` for live reload, matches the dev container)
go run ./cmd/worker   # outbox relay worker — IS started by compose.yaml (identity-worker), unlike
                       # restaurant-service's; without it no outbox event ever leaves this service
                       # (or `air -c .air.worker.toml` for live reload, matches the dev container)
go test ./...
go test ./... -run TestName
```
Tests are integration-style against real Postgres/Redis (see root `CLAUDE.md` for spinning up `compose.test.yaml`); there's no mocked-DB path. Needs `.env.test` populated like `.env.example`.

Migrations (`internal/infrastructure/migrations/*.sql`, golang-migrate) currently define three tables: `email_verifications`, `users`, `outbox_events`.

## Invariants and gotchas

- **Aggregate-ID trap**: `User`'s `dispatch.go` (`internal/application/user/dispatch.go`) assigns a per-event aggregate ID, not one fixed ID per aggregate. `RestaurantInitiated`'s outbox row uses `evt.RestaurantID`, not the owner's `User.ID`, because `RegisterOwner` raises an event for a different aggregate than the one it operates on.
- **Login-enumeration collapse**: `login.go` (`internal/application/auth/commands/login.go`) returns the same `apperr.ErrUnauthorized` for both "no such user" and "wrong password" — a client can't tell them apart, by design. A genuine backend failure (e.g. `FindByEmail` erroring on a DB problem) is a separate path that propagates the real error instead of folding into that same response.
- **Email-lowercasing requirement**: every entry point that accepts an email (`login.go`, `request_email_otp.go`, `register_owner.go`, `register_customer.go`) normalizes it via `strings.ToLower` before using it for a lookup, a rate-limiter key, or storage. Without it, `Foo@Example.com` and `foo@example.com` would be treated as different accounts depending on which path handled the request.
