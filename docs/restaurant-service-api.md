# Restaurant Service API

Routes served under `/restaurants` via the Traefik gateway.

## Restaurants — `/restaurants`

| Method | Path | Auth | Description |
|---|---|---|---|
| `PATCH` | `/restaurants/{id}/contact` | JWT | Update contact info (email, phone, website) |
| `PATCH` | `/restaurants/{id}/address` | JWT | Update address |
| `PATCH` | `/restaurants/{id}/delivery` | JWT | Update delivery settings (pickup, delivery type, radius, fee, minimum order) |
| `POST` | `/restaurants/{id}/payout-details` | JWT | Submit new payout bank details (account holder, IBAN, BIC, bank name) for verification |
| `PUT` | `/restaurants/{id}/payout-details` | JWT | Replace the pending payout submission (all four fields required) |
| `PATCH` | `/restaurants/{id}/opening-hours` | JWT | Replace the weekly opening hours (full replace, one or more `{open, close}` ranges per weekday) |

`PATCH /:id/contact` requires `email` and `phone`; `website` is optional and cleared when omitted. `phone` accepts digits with an optional leading `+`, spaces, hyphens, and parentheses (at least 6 digits) — a loose format check, not strict E.164 validation.

`POST /:id/payout-details` never overwrites data in place — it creates a new `pending` payout record. If one is already pending, the call fails with `409 Conflict`. An existing `active` record is never touched by a new submission; only a future review-promotion step changes it. Promoting a `pending` record to `active` — and flipping the prior `active` record to `superseded` — happens via a review step that isn't implemented yet. Only the `active` record is ever paid out.

`PUT /:id/payout-details` replaces the `pending` record's fields in place — allowed because a `pending` submission hasn't been reviewed yet, so there's nothing to lose by overwriting it (unlike `active`/`superseded` rows, which are never mutated). All four fields are required. `404 Not Found` if there is no `pending` record to replace — including when the restaurant only has an `active` record, since this never touches that. Deliberately `PUT`, not `PATCH` like `address`/`delivery`: the request body is the full resulting state (not a partial diff), so `PUT`'s create-or-replace-at-known-URI contract fits; it stays conditional on `pending` existing rather than unconditionally overwriting, which is why it's not treated as a strict "resource now equals this body no matter what."

`PATCH /:id/opening-hours` takes each weekday (`monday`...`sunday`) as a list of `{open, close}` ranges in `HH:MM` 24-hour format. An empty or omitted list means closed that day. Each range must stay within a single calendar day (`close` must be later than `open` — no crossing midnight in one range). A place open past midnight represents that as two entries: e.g. `monday: [{open: "15:00", close: "23:59"}]` and `tuesday: [{open: "00:00", close: "03:00"}, ...]` for the carryover, rather than one `monday` range crossing into Tuesday. Multiple ranges per day are supported natively (e.g. a lunch/dinner split), not just the overnight case.

`POST` returns `201 Created`. `PUT`/`PATCH` return `200 OK`.
