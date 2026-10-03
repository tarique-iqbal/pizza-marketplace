# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Scope: this file covers `notification-service` only. See the root `CLAUDE.md` for monorepo-wide architecture and how this service fits into the event flow.

## What this service owns

Sends transactional notifications (welcome emails, verification codes) in response to RabbitMQ events, via a channel-adapter design: `internal/domain/notification.Sender` is the channel-agnostic interface, and `internal/infrastructure/notification/email` is its one implementation today (SMTP). A second channel (SMS, web-push) would be a new sibling package implementing the same `Sender` interface, see `internal/domain/notification/sender.go`. It has no database, no HTTP API, and no `cmd/api`/`cmd/worker` split — `cmd/main.go` is the only binary, a pure consumer, started directly by the root `compose.yaml`.

Consumes (`internal/infrastructure/messaging/rabbitmq_consumer.go`), all published via outbox, not best-effort:
- `email.verification_created` (identity-service) → `EmailVerificationCreated` handler
- `user.registered` (identity-service) → `UserRegistered` handler
- `restaurant.ready_for_review` (restaurant-service) → `RestaurantReadyForReview` handler, sent to `ADMIN_EMAIL`
- `restaurant.approved` (restaurant-service) → `RestaurantApproved` handler, sent to the restaurant's own contact email on the event payload
- `order.confirmed` (order-service) → `OrderConfirmed` handler, sends two emails: a customer confirmation and an owner new-order notification

## Commands

Run from inside `notification-service/`:
```bash
go run ./cmd      # or `air -c .air.toml` for live reload, matches the dev container
go test ./...
go test ./... -run TestName
```
No database or `.env.test`/`compose.test.yaml` needed here — tests are pure unit tests against mocked `notification.Sender`/`notification.TemplateLoader`/`notification.EventDispatcher` interfaces, including for the RabbitMQ consumer.

## Testing conventions

Everything is mocked at the interface boundary (`notification.Sender`, `notification.TemplateLoader`, `notification.EventDispatcher`) — no `tests/testutil/db.go`, no fixtures. Follow the existing pattern of a local `mockX` struct implementing the relevant domain interface plus a `var _ notification.X = (*mockX)(nil)` compile-time assertion, rather than reaching for a mocking library.

## Invariants and gotchas

- **Templates are selected dynamically by role for `user.registered`**: filenames are built as `fmt.Sprintf("%s_welcome_email_subject.html", payload.Role)` — adding a new role means adding matching `<role>_welcome_email_subject.html`/`_body.html` files under `internal/infrastructure/notification/email/templates/`. There's no fallback: an unrecognized role fails validation before rendering is attempted. `email_verification_created`'s templates are fixed-named instead (`email_verification_subject.html`/`_body.html`), not role-based.
- **`ensureConnected` checks both `c.conn.IsClosed()` and `c.channel.IsClosed()`** before reconnecting — a channel can die from a channel-level AMQP exception while the connection stays open. This is an independent copy of restaurant-service's consumer, not shared code, so mirror any retry/DLX/reconnect change there too.
- **Handlers are wired by literal routing-key strings** in `internal/container/container.go` (`dispatcher.Register("user.registered", userRegistered)`, etc.).
- **Config is read directly from `os.Getenv` in application handlers** (`APP_NAME`, `SUPPORT_EMAIL`, `TOKEN_EXPIRY_MINUTES`), not just in `container.go` — there's no central config struct passed down.
- **Templates render via `text/template`, not `html/template`**: every template is plain-text content despite the `.html` filenames, so HTML-escaping would be pointless overhead.
- **SMTP headers are hand-assembled** in `internal/infrastructure/notification/email/smtp_sender.go` — add new ones there explicitly; `net/smtp.SendMail` sends only the raw bytes it's given.
