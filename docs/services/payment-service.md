# payment-service — technical overview

Processes payments via Mollie on behalf of any other service in this system. Owns its own Postgres database
(`payment_db`) and runs the transactional outbox pattern, like identity-service and restaurant-service. Unlike
every other service here, its primary surface is **gRPC**, not REST — `internal/interfaces/http` exists only
for the one public route Mollie itself needs to reach: the webhook.

Deliberately generic: a `Payment` is keyed by `subject_type`/`subject_id` (e.g. `"order"` + order-service's
own `Order.ID`), never an order-specific concept. payment-service has no dependency on order-service's code
or database, only on the wire contract (gRPC request fields, the two published events' JSON shape).

## Layered architecture

```
cmd/api                              → two listeners in one process: gRPC server (:50051, internal network
                                        only) implementing PaymentService, and a Gin HTTP server (:8080) with
                                        exactly one route, POST /webhooks/mollie
cmd/worker                           → outbox relay only — no inbound consumer, the only worker in this repo
                                        that doesn't pair one; payment-service consumes nothing, it only
                                        publishes payment.succeeded/payment.failed
internal/domain/payment               → Payment aggregate, PaymentStatus, PaymentRepository, PaymentGateway
                                        interfaces, errors
internal/domain/outbox                → OutboxEvent/OutboxStatus/OutboxRepository — verbatim port of
                                        identity-service's/restaurant-service's own outbox domain
internal/application/payment          → schema.go (CreatePaymentRequest/Response DTOs), events.go
                                        (PaymentSucceededPayload/PaymentFailedPayload), commands/
                                        {CreatePayment,CancelPayment,HandleMollieWebhook}
internal/application/outbox           → Worker/Relay — verbatim port of the same outbox application layer
internal/infrastructure/persistence   → payment.go, outbox.go — thin GORM wrappers, domain structs double as
                                        GORM models
internal/infrastructure/gateway       → MollieGateway — hand-rolled net/http client, no SDK, implements
                                        PaymentGateway against Mollie's real v2 API
internal/infrastructure/messaging     → RabbitMQ publisher only — no consumer, matching the worker's own shape
internal/infrastructure/observability → copied verbatim from restaurant-service
internal/interfaces/grpc              → Server implementing PaymentService (pb.UnimplementedPaymentServiceServer
                                        embedded), pb/ — generated code from api/proto/payment/v1/payment.proto
internal/interfaces/http              → handlers/WebhookHandler, routes/ — the one public route
internal/container                    → shared.go / api.go / worker.go, manual DI
internal/shared/{errors,event}        → errors sentinels, Event interface — no money/geo packages, this
                                        service's only wire format is gRPC (amount as a decimal string)
api/proto/payment/v1/payment.proto    → the canonical gRPC contract this service owns
```

No `jwt@docker` Traefik label anywhere — the gRPC port has no Traefik exposure at all (internal Docker
network only, plaintext transport, no TLS setup yet), and the one HTTP route is deliberately public
(Mollie's servers can't authenticate via this system's own JWT scheme).

## Domain model

```go
// internal/domain/payment/payment.go
type PaymentStatus string
const (
    StatusPending   PaymentStatus = "pending"
    StatusSucceeded PaymentStatus = "succeeded"
    StatusFailed    PaymentStatus = "failed"  // covers Mollie's failed/expired/canceled uniformly
)

type Payment struct {
    ID               uuid.UUID
    SubjectType      string          // "order"
    SubjectID        uuid.UUID
    RestaurantID     uuid.UUID       // carried for future reporting — no payout automation consumes it yet
    CustomerID       uuid.UUID
    Amount           decimal.Decimal
    Currency         string
    PlatformFee      decimal.Decimal // always 0 today — no payment splitting, full amount collects to the
                                      // platform's own Mollie account
    Status           PaymentStatus
    Gateway          string          // "mollie" — which processor, not tied to Mollie by name
    GatewayPaymentID *string         // Mollie's own "tr_xxxxx" reference, set after CreatePayment's gateway call
    FailureReason    *string
    CreatedAt        time.Time       // gorm autoCreateTime
    UpdatedAt        *time.Time      // gorm autoUpdateTime
}

func (p *Payment) AttachGatewayReference(gatewayPaymentID string) // no status check — doesn't change Status
func (p *Payment) MarkSucceeded() error                           // pending -> succeeded
func (p *Payment) MarkFailed(reason string) error                 // pending -> failed
```

**`Gateway`/`GatewayPaymentID`, not `MolliePaymentID`.** The schema was generalized before the first commit
landed, specifically so a second gateway (Stripe, PayPal, …) needs no rename later — just a new
`PaymentGateway` implementation and, when that day comes, a small selection layer in `container.go` (today
there's exactly one `PaymentGateway` field wired in, no registry). `gateway` has no `DEFAULT` in the
migration — always set explicitly by application code, avoiding the same GORM zero-value-plus-`default:`-tag
bug class already fixed elsewhere in this repo (a `DEFAULT` on a column the app always sets risks silently
masking a bug where it forgot to).

**`Payment.Status` has exactly one writer.** `MarkSucceeded`/`MarkFailed` are only ever called from
`HandleMollieWebhook` (below) — every other code path that touches a `Payment` (`CreatePayment`'s replay
branch, `CancelPayment`) reads or attaches a reference but never changes `Status`. No domain-event/
`PullEvents()` machinery on `Payment`, unlike `Order` in order-service — `payment.succeeded`/`payment.failed`
need no snapshot enrichment, since every field they carry already lives directly on `Payment`, so the
application layer builds the event payload straight from the loaded aggregate.

```go
// internal/domain/payment/gateway.go — the seam MollieGateway implements
type CreatePaymentRequest struct {
    RestaurantID uuid.UUID
    Amount       decimal.Decimal
    Currency     string
    RedirectURL  string
    WebhookURL   string  // built by the application layer from PUBLIC_BASE_URL, not the gRPC caller
}
type CreatePaymentResult struct{ GatewayPaymentID, CheckoutURL string }
type PaymentStatusResult struct {
    Status      PaymentStatus
    Reason      string  // only meaningful when Status == StatusFailed
    CheckoutURL string  // re-fetched live from Mollie's GET /payments/{id} — never persisted locally
}
type PaymentGateway interface {
    CreatePayment(ctx context.Context, req CreatePaymentRequest) (CreatePaymentResult, error)
    CancelPayment(ctx context.Context, gatewayPaymentID string) error
    GetStatus(ctx context.Context, gatewayPaymentID string) (PaymentStatusResult, error)
}
```

`PaymentRepository.FindBySubject`/`FindByGatewayPaymentID`/`FindByID` all return `nil, nil` on a miss — the
"optional lookup" convention this repo already uses for `CartRepository.FindByCustomer`/
`GeocodeRepository.FindByHash`, appropriate here since every caller treats "not found" as a normal outcome.

## Application layer

### `CreatePayment` — three branches, not the two a naive idempotency check would produce

```go
// internal/application/payment/commands/create_payment.go
func (uc *CreatePayment) Execute(ctx, req) (Response, error) {
    existing, _ := uc.repo.FindBySubject(ctx, req.SubjectType, req.SubjectID)
    if existing != nil {
        return uc.handleExisting(ctx, existing, req)  // branches 2/3 below
    }
    p := payment.NewPayment(uuid.Must(uuid.NewV7()), ..., decimal.Zero, "mollie")  // branch 1
    uc.repo.Create(ctx, p)
    return uc.callGatewayAndAttach(ctx, p, req)
}
```

1. **No existing row** — create it (status `pending`), call `gateway.CreatePayment`, `AttachGatewayReference`,
   `Update`, return the checkout URL.
2. **Existing row, `GatewayPaymentID == nil`** — a prior attempt persisted the row but crashed before ever
   reaching Mollie. Self-heals: calls the gateway now for this same row instead of creating a second one.
3. **Existing row, has a `GatewayPaymentID`** — a genuine replay (e.g. order-service's own gRPC client timed
   out on a call that actually succeeded server-side, then retried). Calls `gateway.GetStatus` to return a
   *live* checkout URL rather than a persisted one — a checkout URL is only meaningful while Mollie's payment
   is still `open`, so nothing is cached locally; Mollie is asked fresh every time this branch runs.

`CancelPayment` is best-effort: looks up the payment by its own `ID`, calls `gateway.CancelPayment`, and
swallows a gateway error rather than propagating it — a `422` "payment can no longer be canceled" from Mollie
is an expected outcome here (the payment already reached a final state), not a bug.

### `HandleMollieWebhook` — never trusts the POSTed body

```go
// internal/application/payment/commands/handle_mollie_webhook.go
func (uc *HandleMollieWebhook) Execute(ctx, gatewayPaymentID string) error {
    p, _ := uc.repo.FindByGatewayPaymentID(ctx, gatewayPaymentID)
    if p == nil || p.Status != payment.StatusPending {
        return nil  // unknown payment, or already processed — both are no-ops, not errors
    }
    result, err := uc.gateway.GetStatus(ctx, gatewayPaymentID)  // the trust mechanism
    ...
    switch result.Status {
    case payment.StatusSucceeded: p.MarkSucceeded()
    case payment.StatusFailed:    p.MarkFailed(result.Reason)
    default: return nil  // still open/pending/authorized — nothing to do yet
    }
    return uc.db.Transaction(func(tx *gorm.DB) error {
        uc.repo.WithTx(tx).Update(ctx, p)
        outboxRepo.WithTx(tx).Create(ctx, outbox.NewOutboxEvent(p.ID, eventName, payload))
        return nil
    })
}
```

Mollie's webhook body carries only a payment `id`, no signature — Mollie's own stated security model is
"retrieve the full entity to understand what changed" rather than trust the callback. `GetStatus`'s live
re-check is that trust mechanism. The `Payment` update and the outbox insert land in one transaction — the
business write and the event that tells the rest of the system about it either both happen or neither does.

**Idempotent against Mollie's own retry behavior.** Mollie retries a failed webhook delivery (non-`200`
response, or no response within 15s) with exponential backoff, up to ~10 attempts over roughly 26 hours. The
`p.Status != StatusPending` check above means a late or duplicate redelivery for an already-terminal payment
is a genuine no-op — checked *before* even calling `GetStatus` again, not relying on `MarkSucceeded`'s own
guard (which would also correctly reject a second call, but the goal is to never attempt it on a redelivery
in the first place).

## Published events

Both on exchange `payment.events`, routing key = event name:

```go
// internal/application/payment/events.go
type PaymentSucceededPayload struct {
    PaymentID, SubjectID, RestaurantID uuid.UUID
    SubjectType, Currency             string
    Amount                            string  // StringFixed(2) — not a bare decimal.Decimal field, dodges
                                               // decimal.Decimal's default MarshalJSON trimming trailing
                                               // zeros ("1.50" -> "1.5"), the same bug class Money fixes
                                               // in restaurant-service's own HTTP responses
    EventName                         string
    OccurredAt                        time.Time
}
// PaymentFailedPayload: same shape + Reason string
```

order-service consumes both: its worker's `PaymentSucceededHandler`/`PaymentFailedHandler` call
`Order.Confirm()`/`Order.Cancel()` in response, and its `Checkout` command calls `CreatePayment` over gRPC
(via a circuit-breaker-wrapped client) to get the checkout URL in the first place — see order-service's own
`CLAUDE.md` for that side.

## gRPC contract — the first gRPC surface in this monorepo

Canonical `.proto`: `api/proto/payment/v1/payment.proto`. Generated code committed to
`internal/interfaces/grpc/pb/` (`payment.pb.go`, `payment_grpc.pb.go`) — regenerate via `make proto` from the
repo root after editing the `.proto` file (requires `protoc` plus the `protoc-gen-go`/`protoc-gen-go-grpc`
Go plugins on `PATH`; `--go_out`/`--go-grpc_out` must both be `payment-service`, not `.`, combined with
`--go_opt=module=payment-service` — the `module` option *strips* that prefix from the `.proto`'s own
`go_package` path rather than adding it, so `--go_out` has to supply the prefix back).

No shared Go module for the generated code — order-service's own gRPC client
(`internal/infrastructure/payment/`) copies the generated client stub into its own tree by hand, kept in
sync manually (same per-service-duplication convention this repo uses for the RabbitMQ consumer/publisher
and every OpenCage client copy). Transport is plaintext, no
TLS, internal Docker network only — matches this repo's current security posture everywhere else (RabbitMQ/
Postgres connections are also unencrypted internal traffic).

## Testing conventions

Same shape as identity-service/restaurant-service: `tests/` mirrors `internal/`, real Postgres/RabbitMQ via
`compose.test.yaml`, no DB mocking (`tests/testutil/db.go`). `MollieGateway` is tested against
`httptest.NewServer`, not the real Mollie API or a mock of this repo's own infrastructure — request shape
(method/path/headers/body), every Mollie status value's mapping to `PaymentStatus`, and non-2xx error
surfacing are all covered without a real network call or API key. Application-layer command tests
(`CreatePayment`/`CancelPayment`/`HandleMollieWebhook`) use a locally-defined `fakeGateway` implementing
`PaymentGateway`, per test file rather than a shared mock package. The gRPC server layer
(`tests/interfaces/grpc/server_test.go`) is exercised by calling `Server.CreatePayment`/`CancelPayment`
directly as plain Go methods — no `bufconn`, since the only real risk at that layer is field-translation and
gRPC status-code mapping, not wire/transport behavior.

Verified live, twice, against real external systems: a throwaway gRPC client (inside the running container)
calling `CreatePayment` reached Mollie's actual v2 API and got a genuine `400 Bad Request` back for a dummy
test key — proving the request/auth/JSON shape end to end — with the `payments` row still persisting
correctly as `pending`, ready to self-heal on retry. And `curl` through a real Traefik instance confirmed
`POST /webhooks/mollie` is reachable from outside the internal Docker network, returning `200`/`400` exactly
as designed.
