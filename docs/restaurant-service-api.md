# Restaurant Service API

Routes served under `/restaurants` via the Traefik gateway.

## Restaurants — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |
| `PATCH` | `/restaurants/{id}/delivery` | JWT | Update delivery settings (pickup, delivery type, radius, fee, minimum order) |
| `POST` | `/restaurants/{id}/payout-details` | JWT | Submit new payout bank details (account holder, IBAN, BIC, bank name) for verification |
| `PUT` | `/restaurants/{id}/payout-details` | JWT | Replace the pending payout submission (all four fields required) |

`POST /:id/payout-details` never overwrites data in place — it creates a new `pending` payout record. If one is already pending, the call fails with `409 Conflict`. An existing `active` record is never touched by a new submission; only a future review-promotion step changes it. Promoting a `pending` record to `active` — and flipping the prior `active` record to `superseded` — happens via a review step that isn't implemented yet. Only the `active` record is ever paid out.

`PUT /:id/payout-details` replaces the `pending` record's fields in place — allowed because a `pending` submission hasn't been reviewed yet, so there's nothing to lose by overwriting it (unlike `active`/`superseded` rows, which are never mutated). All four fields are required. `404 Not Found` if there is no `pending` record to replace — including when the restaurant only has an `active` record, since this never touches that. Deliberately `PUT`, not `PATCH` like `address`/`delivery`: the request body is the full resulting state (not a partial diff), so `PUT`'s create-or-replace-at-known-URI contract fits; it stays conditional on `pending` existing rather than unconditionally overwriting, which is why it's not treated as a strict "resource now equals this body no matter what."

`POST` returns `201 Created`. `PUT` returns `200 OK`.
