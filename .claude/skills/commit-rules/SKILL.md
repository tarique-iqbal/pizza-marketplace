---
name: commit-rules
description: Layered-commit conventions for adding a new feature to a service in this repo — shared rules plus a per-service reference table (layer names, prefixes, doc paths, test commands)
---

# Commit rules

Use this when adding a new feature/endpoint to any service in this monorepo and
sequencing the resulting commits. Read the target service's own `CLAUDE.md`
first for its architecture — this skill only covers *how to sequence the
work*, not domain rules.

This is planning/execution guidance, not a shortcut past confirmation: still
show the plan, the diff, and each commit message before running `git commit`,
one commit at a time, per this repo's established practice.

## 1. Shared rules (apply to every service)

- One commit per layer, in the service's own layer order (§2).
- Tests land **with their layer's commit** — never a separate "tests" commit.
- The final commit is always `docs:`, updating both of the service's doc
  files together (§3).
- Run the service's build/lint/test verification (§4) before proposing the
  commit plan, and again before the final commit if anything changed.
- Before every individual `git commit`, run `git diff --cached --stat` and
  confirm it matches exactly the file list for that layer — never assume
  earlier staging is still accurate.

## 2. Per-service layer shapes

### restaurant-service

Not every feature needs every layer. Pick the smallest set of commits that
actually applies, based on precedent in this repo's git history:

| Feature shape | Commits | Precedent |
|---|---|---|
| Reuses the existing `restaurants` row, no new table | **5**: domain → commands → http → container wiring → docs | `contact` (checklist item, 2026), `opening-hours` (`993e456`..`e1f2a97`) |
| Reuses the existing `restaurants` row, but needs a column/constraint change | **6**: migrations → domain → commands → http → container wiring → docs | `tags` (`32015b1`..`805821a`) — a `CHECK` constraint added via migration, no separate persistence commit since the domain method (`Restaurant.WithTags`) writes through the existing row |
| New persistence logic, but no new table | **6**: domain → persistence → commands → http → container wiring → docs | `payout-details` update (`f4ecd2b`..`21ea096`) |
| Needs a brand-new table | **7**: migrations → domain → persistence → commands → http → container wiring → docs | `payout-details` create (`414dc6e`..`2555298`) |

Note the two 6-commit shapes differ in *which* layer the migration displaces:
a migration commit appears only when there's no dedicated persistence-layer
commit to fold schema-adjacent changes into, and vice versa. If a feature
needs both a migration *and* new persistence code without a new table, that's
a 7-commit shape (`migrations → domain → persistence → commands → http →
container wiring → docs`) — no precedent yet, but follows directly from the
pattern above.

Commit message prefixes: `feat(migrations)`, `feat(domain)`,
`feat(persistence)`, `feat(commands)`, `feat(http)`, `refactor(container)`,
`docs:`.

Layer-specific rules:
- The domain commit also picks up any file outside `internal/domain/` that
  breaks if the domain change lands alone — most commonly
  `tests/infrastructure/db/fixtures/restaurant_fixture.go` when a checklist
  constant is renamed/added and the fixture's map literal references it. If
  omitting a file would make that commit fail to compile in isolation, it
  belongs in that commit regardless of which directory it physically lives in.
- The `http` commit also includes any new custom validator tags added to
  `internal/interfaces/http/validation/validators.go` for this feature's
  request struct.
- The `container` commit is `cmd/api/main.go` + `internal/container/api.go`
  only.

### search-service

Every feature shipped so far (restaurant-lookup-by-id, delivery-time-range)
followed the same **4-commit** shape: `domain → infrastructure → application
→ interfaces → docs`. Unlike restaurant-service, container wiring is folded
into the `interfaces` commit rather than split out — search-service has no
`persistence`/`gorm` layer to separate from wiring, and there's no precedent
yet for a shape needing more or fewer layers.

Commit message prefixes: `feat(domain)`, `feat(infrastructure)`,
`feat(application)`, `feat(interfaces)`, `docs:`.

Precedent: `d4a9e53`..`8d69b90` (restaurant lookup), `b486ea5`..`87964eb`
(delivery time range).

Only 2 features' worth of precedent — treat this shape as provisional and
update it if a feature comes along that doesn't fit.

### identity-service / notification-service

No precedent captured yet. Add a row here (layer shape + prefixes +
precedent commits) once a feature ships in either service — don't assume
identity-service's shape carries over to notification-service when it does:
notification-service is worker-only (no HTTP layer, no DB, no compose test
container — plain `go test ./tests/...` from the host), structurally unlike
either of the other two services.

## 3. Docs to update, per service

Both files listed for a service must be updated together in the `docs:`
commit — they've historically been kept in sync together, never just one:

| Service | Docs |
|---|---|
| restaurant-service | `restaurant-service/CLAUDE.md` (routes list + relevant "Architecture specifics" bullet) + `docs/api/restaurant-service.md` (route table row + prose paragraph on required/optional fields and non-obvious validation) |
| search-service | `search-service/CLAUDE.md` (routes/behavior description) + `docs/api/search-service.md` (route table row + prose paragraph) |

## 4. Pre-commit verification, per service

| Service | Commands |
|---|---|
| restaurant-service | `gofmt -l .`, `go vet ./...`, `go build ./...` from inside `restaurant-service/` (or `make fmt vet` from repo root — covers identity/restaurant/notification together, no build check); real-DB test: `make test-up`, then `make test-restaurant`; teardown `docker compose -f compose.test.yaml --profile test down -v` — **include `--profile test` on the `down`**, not just `up`; the root `Makefile`'s own `test-down` target omits it and leaves `-test` containers running |
| search-service | Same `make test-up` / teardown flow, then `make test-search` (real Elasticsearch, not Postgres) — **not** covered by `make fmt`/`make vet`/`make lint`, which only loop over identity/restaurant/notification; run `gofmt -l .`, `go vet ./...`, `go build ./...` manually from inside `search-service/` |
