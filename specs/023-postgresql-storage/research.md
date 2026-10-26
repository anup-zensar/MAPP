# Research: PostgreSQL Storage (023)

Findings from reading the JSON store (`internal/platform/store`), the repository interfaces and
`store_json.go` files of every area, and from building the foundation and the pilot against PostgreSQL 18
(Compose service, `127.0.0.1:5432`) with Go 1.27.3 on 2026-10-26.

## R1 — Dependencies (constitution B1, VII release age)

Each version was checked on `https://proxy.golang.org/<module>/@v/<version>.info` on 2026-10-26; all were
published more than 24 hours earlier.

| Module | Version | Published | Licence | Purpose and why the standard library is not enough |
|---|---|---|---|---|
| `github.com/jackc/pgx/v5` | v5.11.0 | 2026-09-07 | MIT | The approved PostgreSQL driver and pool (B1); `database/sql` has no PostgreSQL driver and no pool tuned for it. Actively maintained (monthly releases). |
| `github.com/pressly/goose/v3` | v3.29.0 | 2026-10-11 | MIT | The approved migration tool (B1): embedded SQL files, `NewProvider`, PostgreSQL session locker. |
| `github.com/google/go-cmp` | v0.7.0 | 2025-01-14 | BSD-3-Clause | The approved test comparison library (B9); test-only. |
| `github.com/sqlc-dev/sqlc` (tool, not in `go.mod`) | v1.31.1 | 2026-04-22 | MIT | The approved query generator (B1), run as `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate`, so its large dependency tree never enters the module graph. |

Brought in by them (indirect): `jackc/puddle/v2` v2.2.3 (2026-10-05), `jackc/pgpassfile` v1.0.0
(2019-03-30), `jackc/pgservicefile` v0.0.0-20240606120523 (2024-06-06), `mfridman/interpolate` v0.0.2
(2023-12-22), `sethvargo/go-retry` v0.5.0 (2026-10-05), `go.uber.org/multierr` v1.11.0 (2023-03-29). Raised
by goose's requirements: `golang.org/x/net` v0.60.0 → v0.61.0 (2026-10-09), `golang.org/x/sync` v0.23.0 →
v0.24.0 (2026-09-23), `golang.org/x/text` v0.42.0 → v0.43.0 (2026-10-09). No ADR is needed: all four are the
constitution's named stack (B1).

## R2 — Transactions: one `Write` at a time, snapshot `Read`

- **Decision**: `database.DB` implements today's `store.Transactor`. `Write` runs one `READ COMMITTED`
  transaction carried in the context whose first statement takes a transaction-scoped advisory lock
  (`pg_advisory_xact_lock`), so Writes run one at a time — across API processes too — and each statement
  sees everything committed before it. The function's own error is returned unchanged; there are no
  retries, so the function runs exactly once. `Read` runs `REPEATABLE READ READ ONLY` (one consistent
  snapshot) and never waits for Writes. Inside a transaction both join it; `Write` inside `Read` returns
  `ErrReadOnly`. `Queries(ctx)` gives the sqlc queries of the transaction in the context (or of the pool
  outside one).
- **Rationale**: the `Transactor` contract services rely on promises a `Write` "exclusive access" — the JSON
  store serialises writes with a mutex. Services are full of read-check-write rules (version checks, "name
  unique ignoring case", readable-ID counters, "one open round per activity"). One writer at a time keeps
  every one of them correct without rewriting a single service, exactly as today. It costs no
  concurrency the application has: every Write logs its change, and every log entry raises the one `LOG`
  counter, so any two Writes touch the same row anyway.
- **Measured**: the first design ran Writes `SERIALIZABLE` with up to 5 retries. Two writers in a loop on
  the counters starved each other (the loser's retry kept meeting the winner's next transaction) and
  failed with `40001` after 5 attempts — the same would happen to any two concurrent Writes, because all
  of them raise the `LOG` counter. With the lock, the racing tests of that time passed every run and the lock
  test failed (timed out) when the lock was removed; those tests were deleted under constitution 4.0.0.
- **Alternatives**: `SERIALIZABLE` with retries — rejected (above). `READ COMMITTED` without the lock plus
  conditional updates (`WHERE version = $n`) in every store, as in `GO_STANDARDS.md` §6 — rejected for
  this migration: it moves rules into SQL area by area and still leaves uniqueness and counter races; the
  guide's sample is corrected in the documentation task.
- **Consequence**: a function passed to `Write` must pass its context on: a Write started with a fresh
  context inside another would wait for the lock the outer one holds (the JSON store has the same rule).

## R3 — Types and UTC

- **Decision**: timestamps are `timestamptz`; the pool registers the `timestamptz` codec with
  `ScanLocation: time.UTC`, so every timestamp comes back in UTC whatever the machine's zone (the pilot test
  fails without it on a machine in IST). Microsecond precision loses nothing: `clock.System` truncates to
  milliseconds. sqlc maps `timestamptz` to `time.Time` and `integer` to `int`, so stores convert nothing.
- Money: `money.Amount` → `bigint` cents (`<name>_minor`); `money.Money` → `<name>_minor` plus
  `<name>_currency`; `money.Rate` → `bigint` millionths (`<name>_micros`); `money.Percent` → `bigint`
  hundredths (`<name>_hundredths`): the same exact fixed-point integers the money package holds
  (`RateFromMicros`, `PercentFromHundredths`), so stores convert nothing; `numeric` would need big-number
  conversion for no gain (B5 allows either). `date.Date` → `date` (sqlc: `time.Time`); a month ("2026-03")
  → `date` on the first of the month. Nullable columns map to Go pointers (`sqlc.yaml` overrides).

## R4 — Constraints that are never stricter than the services

- **Foreign keys** are `DEFERRABLE INITIALLY DEFERRED`: checked at commit, so the order in which a service
  writes parent and child inside one transaction (copy, move, restore) does not matter, while no orphan can
  be committed.
- **Soft references**: a reference that may outlive its target — to a record in the recycle bin (approval
  rounds and notifications of a deleted activity, transactions of a deleted line item, a deleted pinned
  plan, the source of a split activity), to a record whose delete rule ignores it (a workflow level's role,
  an activity's workflow, an allocation's global budget, a transaction's currency) or to history (actor
  ids, which include `system`) — is a plain id without a foreign key. Each area's delete and restore rules
  were read to find them; the list is in the data model §4.
- **Uniqueness a Write may break for a moment** (a second open round, a plan's allocation, a GSAP identity,
  overlapping delegations) is a deferrable constraint or exclusion, checked at commit.
- **Unique "ignoring case"**: the services compare with `strings.EqualFold`. A unique index on
  `lower(name COLLATE "C")` folds ASCII letters only, so it can never refuse a name the service accepts,
  and it still stops the common duplicates if a rule is ever bypassed.
- Length limits are not `CHECK`ed where a rule allows exceptions (a restored folder's "(restored)" suffix may
  pass 80 characters, feature 011 FR-025); non-empty names and enumerations are.

## R5 — Test databases (`dbtest`)

- **Decision**: `dbtest.NewDB(t)` reads `TEST_DATABASE_URL` (a `postgres://` URL), builds once a template
  database named `mapp_template_<hash of the migration files>`, and gives each test
  `CREATE DATABASE mapp_test_<random> TEMPLATE …`, dropped `WITH (FORCE)` in `t.Cleanup`. The template is
  built under a PostgreSQL advisory lock (parallel `go test` processes build it once) under a temporary
  name and renamed when complete (an interrupted build never leaves a broken template). Under
  `-short` it skips; without the variable it **fails**, so CI can never skip the scope-isolation tests
  silently. Since constitution 4.0.0 they are its only users: no Go test checks queries, constraints,
  migrations or the write lock on a database.
- **Rationale**: a copy from a template takes ~0.15 s, so every test can run in parallel on its own data.
  Templates of older migration sets stay on the server and may be dropped by hand
  (`DROP DATABASE mapp_template_… WITH (FORCE)`; the backend README gives the command). The role in `TEST_DATABASE_URL` needs `CREATEDB`; migration 00013
  creates the trusted extension `btree_gist`, which the owner of a database may create.
- **Alternatives**: testcontainers (banned: Docker), one shared database with transactions rolled back
  (cannot test commit-time constraints or concurrent transactions).

## R6 — Migrations

Sequentially numbered goose files (`00001_sample_records.sql`), `-- +goose Up` only (forward-only, B5),
embedded with `//go:embed`; `database.Migrate` uses `goose.NewProvider` with the PostgreSQL session
locker. The concurrent-migration test proves two runs at once both succeed. The API's start-up check
(FR-002) uses the provider's `HasPending` in the wiring task.

## R7 — sqlc

`backend/sqlc.yaml` reads the migrations as the schema and `internal/platform/database/queries/*.sql`
(one file per area), and writes package `sqlcdb` (pgx/v5, empty slices, `timestamptz` → `time.Time`,
`int4` → `int`). Queries list their columns in table order so sqlc reuses the table's model type, and a
store converts the model to the `Put…Params` type with a plain Go conversion. Generated code is committed;
regenerate with `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate` from `backend/`.

## R8 — The pilot area: samples

- **Decision**: `internal/samples` (sample folders and notes).
- **Rationale**: it depends on nothing but the platform (no other area reads or references it), its
  interface is small (9 methods), and it still exercises everything the others need: `record.Meta` with
  versions, a parent and child with a foreign key, a case-insensitive unique name, ordering by id, and the
  recycle-bin order of deletes. Its records are not plan-owned, so it has no scope to isolate; the scoped
  pattern comes with the plans area, after 022.
- **Alternatives**: `auth` (sessions reference people, so it waits for the people tables), `people` (17
  methods and the hooks of four other areas), `history` (needs every area's kinds to restore).

## R9 — Mapping conventions for the remaining areas

`record.Meta` → `version`, `created_at`, `created_by_id`, `created_by_name`, `updated_at`,
`updated_by_id`, `updated_by_name` (names as they were, like the JSON store). An optional reference stored
as `""` in Go is `NULL` in SQL (so the foreign key holds); optional pointers are `NULL`. A list of
references a record owns is a child table (with a `position` when its order matters); a list of plain
values (permissions, column keys, soft ids) is a `text[]` column. The parent's `Put` replaces its child
rows. Every store returns exactly what was saved: a `nil` list stays `nil` and an empty one stays empty
(`NULL` versus `'{}'` / `'[]'`), and a field the JSON store drops when empty (`omitempty`) reads back `nil`.
Lists keep the JSON store's order: `ORDER BY length(id), id COLLATE "C"` (byte order, shorter ids first).

## R10 — Where `json` is used, and why

Only for values whose shape is not relational and that no query looks inside: Shell-attribute values (the
value's type depends on the attribute, seven types); recycle-bin snapshots and activity-log `changes`,
`path` and `items` (immutable copies of the past); approval-round level snapshots and rule facts (a frozen
copy of the workflow at submission); a workflow level's two conditions; saved-view state (screen state).
They are `json`, not `jsonb`: `json` keeps the text byte for byte, so a value reads back exactly as it was
written. `jsonb` re-orders keys and spacing, which would make `history.Diff`, comparing stored and new
values, report changes nobody made. Every such column has a `CHECK (json_typeof(…) IN (…))`.

## R11 — Existing data and `store.Memo`

- JSON data folders are not converted (demo data only); the seed command loads the demo data into
  PostgreSQL.
- `store.Memo` (settings forms decoded once per request) has no PostgreSQL counterpart yet. The settings
  task first measures the pages that use it on PostgreSQL; only if they miss today's times does
  `database` gain a per-transaction memo with the same contract (Principle V: no optimisation without a
  measurement).

## R12 — Scope in SQL (FR-009)

- **Decision**: `authz.Scope` (from `authz.ScopeOf` / `PermissionScopeOf`) carries where the person holds a
  permission. Three SQL functions mirror the Go rules: `plan_in_scope()` = `authz.CanView`,
  `plan_in_permission_scope()` = `authz.InScope`, `activity_in_scope()` = `authz.CanViewActivity` (plan
  scope, activity owner, approvers named in its rounds). Each plan-owned store keeps its interface's
  unscoped reads (internal work: usage counts, restores, the seed) and adds `…InScope` reads that call the
  functions; the plans scope-isolation test compares the SQL with `authz.CanView` for six people and roles.
- **Rationale**: 022 kept today's unscoped store interfaces and filters in the services, so changing the
  interfaces would change every service; the scoped reads let T025a move each person-facing read into SQL
  without a second schema change.
