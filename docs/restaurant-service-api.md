# Restaurant Service API

Routes served under `/restaurants` via the Traefik gateway.

## Restaurants — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |
| `PATCH` | `/restaurants/{id}/delivery` | JWT | Update delivery settings (pickup, delivery type, radius, fee, minimum order) |
| `POST` | `/restaurants/{id}/payout-details` | JWT | Submit new payout bank details (account holder, IBAN, BIC, bank name) for verification |

`POST /:id/payout-details` never overwrites data in place — it creates a new `pending` payout record. If one is already pending, the call fails with `409 Conflict`. An existing `active` record is never touched by a new submission; only a future review-promotion step changes it. Promoting a `pending` record to `active` — and flipping the prior `active` record to `superseded` — happens via a review step that isn't implemented yet. Only the `active` record is ever paid out.

`POST` returns `201 Created`.
