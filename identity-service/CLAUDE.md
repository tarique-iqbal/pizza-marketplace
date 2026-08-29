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

## Architecture specifics

- **Two binaries share one container image**: `cmd/api` (Gin HTTP server) and `cmd/worker` (outbox poller). DI is wired separately per binary in `internal/container/api.go` and `internal/container/worker.go`, both building on common dependencies from `internal/container/shared.go`.
- **Auth domain split across three storage backends**:
  - `users` (Postgres) — the account itself, via `internal/infrastructure/persistence/user.go`.
  - `email_verifications` (Postgres) — OTP codes for signup, via `internal/infrastructure/persistence/email_verification.go`. Verified once (`is_used`), single active code per email; see `internal/infrastructure/auth/email_verifier.go` for the state checks (`ErrCodeInvalid` / `ErrCodeExpired` / `ErrCodeUsed` / `ErrCodeNotIssued`), mapped to HTTP statuses in `internal/interfaces/http/response/error_response.go`'s `HandleError`.
  - Refresh tokens (Redis, **not** Postgres) — `internal/infrastructure/persistence/refresh_token.go`. Stored as `refresh:<sha256(token)>` → JSON `UserClaims`, TTL-based expiry (7 days, see `internal/application/auth/login.go`). The raw token is only ever returned to the client; the store holds only its hash. `Find` distinguishes a genuine Redis miss (key never existed or TTL expired) from a real backend failure: the former returns `auth.ErrRefreshTokenInvalid` (`internal/domain/auth/errors.go`, → 401), the latter propagates the raw error (→ logged 500) — don't collapse these back into one generic error if touching this file.
- **Registration is asymmetric between roles** (`internal/application/user/register_owner.go` vs `register_customer.go`): both create the `User` row and a `user.registered` outbox row in the same DB transaction, but registering an **owner** also writes a `restaurant.initiated` outbox row in that same transaction (see root `CLAUDE.md` for the outbox mechanics), because a restaurant record must reliably get created downstream in `restaurant-service`. Registering a **customer** has no such downstream dependency, so it writes only the one outbox row. Keep this asymmetry in mind if adding new registration flows — the deciding factor is whether another service *must* eventually observe the event.
- **JWT vs session state**: access tokens are stateless JWTs (`internal/infrastructure/security/jwt_manager.go`); refresh tokens are opaque random values, hashed before storage, looked up in Redis on `/auth/refresh` and deleted on `/auth/logout`.
- **Duplicate-email registration is rejected at OTP-request time, not just at `Create`**: `RequestEmailOTP` (`internal/application/auth/request_email_otp.go`) checks `UserRepository.EmailExists(ctx, email)` before generating/sending a code and returns `user.ErrEmailAlreadyExists` (`internal/domain/user/errors.go`) if one exists — proving control of an inbox via OTP was never proof that inbox wasn't already registered, so this closes a real gap where a duplicate registration used to silently 500. `userRepo.Create` (`internal/infrastructure/persistence/user.go`) still separately translates a Postgres unique-violation (`23505`) into the same sentinel — kept as the race-safe backstop, since the OTP-time check is inherently check-then-act. Note: `POST /auth/email/verify` is public/unauthenticated, so explicitly rejecting on an already-registered email is a mild email-enumeration side channel, and it's now the *only* place in this service that leaks registration status — `login.go` deliberately collapses "no such user" and "wrong password" into one generic response (see below), so this endpoint no longer has that same cover. Accepted as-is; revisit if enumeration hardening is ever prioritized.
- **`login.go` deliberately collapses "no such user" and "wrong password" into one sentinel**: both cases return bare `apperr.ErrUnauthorized` (`internal/shared/errors/errors.go`) — a client can't tell them apart, which is the point (avoids user enumeration on the login endpoint specifically). A genuine backend failure (e.g. `FindByEmail` erroring on a DB problem) is a separate path that propagates the real error instead of being folded into the same generic response — don't re-collapse these if touching this file; that was a real bug (see `notes/error_handling_fix.md`'s history) where a DB outage during login was literally indistinguishable from bad credentials, with no logging and no way to tell from the outside.
- **Error convention**: `internal/interfaces/http/response/error_response.go`'s `HandleError` is the single canonical error-to-HTTP-status dispatcher for this service's whole HTTP layer (`errors.Is`-based, matches restaurant-service's shape) — every handler in `user_handler.go` and `auth_handler.go` routes through it now; there is no other mechanism (`internal/interfaces/http/mapper/error_mapper.go` was deleted, fully superseded). It covers `auth.ErrCode*` (email verification), `auth.ErrRefreshTokenInvalid`, the shared sentinels (`ErrUnauthorized`, `ErrForbidden`, `ErrNotFound`, `ErrConflict` in `internal/shared/errors/errors.go`), and domain-specific ones like `user.ErrEmailAlreadyExists` — logs unhandled errors via `logobs` before returning a generic 500 on its default branch. Prefer reusing an existing sentinel over inventing a new bare error string in a handler or use case; if a new domain error needs its own HTTP status, add a case to `HandleError` rather than handling it ad hoc in a handler.
- **Structured logging via `internal/infrastructure/observability/logger`** (`logobs`, ported from restaurant-service — same `NewLogger`/`WithContext`/`FromContext` API): `cmd/api/main.go` uses `gin.New()` + `gin.Recovery()` + `observability.Middleware(logger)` (replacing `gin.Default()`) to inject a request-scoped logger carrying `request_id`/`method`/`path` into each request's context; `cmd/worker/main.go` seeds the same logger into the worker's `ctx` via `WithContext` before starting. `rabbitmq_publisher.go` reads it back via `FromContext(ctx)` instead of stdlib `log` for reconnect/publish-failure warnings, on the relay's own path in `cmd/worker` — ctx already flows from the Gin middleware through to request-scoped code, so no extra wiring was needed to get request-scoped fields where they're used.

## Testing conventions

- `tests/` mirrors `internal/` 1:1 (`tests/application/...`, `tests/infrastructure/...`, `tests/interfaces/...`).
- `tests/testutil/db.go` provides a singleton `TestDB` (real GORM connection) with a `TruncateTables(t, testutil.TableUser, ...)` method — call this between tests instead of dropping/recreating schema.
- `tests/infrastructure/db/fixtures/*.go` are `LoadXFixtures(t, db)` helpers that insert known rows (e.g. `john.doe@example.com` / `existing@example.com` users with password `plainPassword`) for tests that need pre-existing data rather than building it inline.
- `tests/testutil/redis.go` provides an equivalent singleton `TestRedis` with a `Flush` helper for Redis-backed state. `tests/testutil/identity.go` is just UUID-generation helpers (`MustNewID`/`MustNewIDString`), not JWT/claims setup — JWT and claims test coverage instead lives directly in `tests/infrastructure/security/jwt_manager_test.go` and `tests/interfaces/http/middlewares/auth_middleware_test.go`.
