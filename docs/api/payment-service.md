# Payment Service API

This service's primary surface is **gRPC** (`PaymentService`, internal Docker network only, port `:50051` — no Traefik exposure, no auth layer of its own since only trusted internal services can reach it). It has exactly one HTTP route, served via the Traefik gateway.

## gRPC — `PaymentService`

Contract: `payment-service/api/proto/payment/v1/payment.proto`. Amounts are decimal strings on the wire (e.g. `"24.50"`), not native numeric types.

### `CreatePayment`

| Field | Type | Notes |
|---|---|---|
| `subject_type` | string | What this payment is for, e.g. `"order"` — payment-service never interprets this itself |
| `subject_id` | string (UUID) | The caller's own id for that thing, e.g. order-service's `Order.ID` |
| `restaurant_id` | string (UUID) | Carried for reporting; not used for any split-payment logic today |
| `customer_id` | string (UUID) | Who's paying |
| `amount` | string | Decimal, e.g. `"24.50"` |
| `currency` | string | ISO code, e.g. `"EUR"` |
| `redirect_url` | string | Where Mollie sends the customer back to after paying |

Response: `payment_id` (this service's own `Payment.ID`), `checkout_url` (Mollie's hosted checkout page — redirect the customer here), `status` (`"pending"` on a fresh call).

Idempotent on `(subject_type, subject_id)` — calling twice with the same pair returns the existing payment rather than creating a second one. On a genuine replay of an already-completed create, `checkout_url` is re-fetched live from Mollie rather than a cached value, since it stops being valid once the payment leaves Mollie's `open` state.

### `CancelPayment`

| Field | Type | Notes |
|---|---|---|
| `payment_id` | string (UUID) | This service's own `Payment.ID` |

Response: `status`. Best-effort — cancelling a payment that already reached a final state at Mollie is treated as a normal outcome, not an error.

## Webhook — `/webhooks/mollie`

| Method | Path | Auth | Description |
|---|---|---|---|
| `POST` | `/webhooks/mollie` | — (public) | Mollie's own payment-status callback |

Form-encoded body, single field `id` (Mollie's own payment reference — no JSON, no signature). `400` if `id` is missing. `200` on every other outcome, including every no-op case (unknown payment, already-processed payment, a still-open/non-terminal Mollie status) — only a real failure on this service's own side returns `500`, so Mollie's retry mechanism only fires when retrying could actually help. The webhook body is never trusted for the actual payment status; this handler always re-fetches the true status from Mollie's API before updating anything.
