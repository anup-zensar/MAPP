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

## R2 — Transactions: serializable `Write` with retry, snapshot `Read`

- **Decision**: `database.DB` implements today's `store.Transactor`. `Write` runs one `SERIALIZABLE`
  transaction carried in the context and runs the function again (at most 5 attempts) when PostgreSQL
  aborts it with a serialization failure (`40001`) or a deadlock (`40P01`); the function's own error is
  returned unchanged. `Read` runs `REPEATABLE READ READ ONLY` (one consistent snapshot). Inside a
  transaction both join it; `Write` inside `Read` returns `ErrReadOnly`. `Queries(ctx)` gives the sqlc
  queries of the transaction in the context (or of the pool outside one).
- **Rationale**: the `Transactor` contract services rely on promises a `Write` "exclusive access" — the JSON
  store serialises writes with a mutex. Services are full of read-check-write rules (version checks, "name
  unique ignoring case", readable-ID counters, "one open round per activity"). Under `READ COMMITTED`
  every one of them would race; serializable isolation keeps them correct without rewriting a single
  service. A retried attempt re-reads, so a lost race becomes an ordinary `version_conflict`. Measured in
  the pilot: the racing-renames test fails under `READ COMMITTED` (both saves succeed) and passes under
  `SERIALIZABLE`.
- **Alternatives**: `READ COMMITTED` plus conditional updates (`WHERE version = $n`) in every store, as in
  `GO_STANDARDS.md` §6 — rejected for this migration: it moves rules into SQL area by area and still leaves
  uniqueness and counter races; the guide's sample is corrected in the documentation task. Retrying in
  each service — rejected: duplicated in every service.
- **Consequence**: a function passed to `Write` must change nothing outside the transaction (it may run
  again). Today's services only assign captured result variables, which a retry overwrites.

## R3 — Types and UTC

- **Decision**: timestamps are `timestamptz`; the pool registers the `timestamptz` codec with
  `ScanLocation: time.UTC`, so every timestamp comes back in UTC whatever the machine's zone (the pilot test
  fails without it on a machine in IST). Microsecond precision loses nothing: `clock.System` truncates to
  milliseconds. sqlc maps `timestamptz` to `time.Time` and `integer` to `int`, so stores convert nothing.
- Money: `money.Amount` → `bigint` cents (`<name>_minor`); `money.Money` → `<name>_minor` plus
  `<name>_currency`; `money.Rate` → `numeric(13,6)`; `money.Percent` → `numeric(5,2)` (B5 allows `numeric`
  where more scale is needed); both converted through their exact text form (`ParseRate`, `String`).
  `date.Date` → `date`; a month ("2026-03") → `date` on the first of the month.

## R4 — Constraints that are never stricter than the services

- **Foreign keys** are `DEFERRABLE INITIALLY DEFERRED`: checked at commit, so the order in which a service
  writes parent and child inside one transaction (copy, move, restore) does not matter, while no orphan can
  be committed.
- **Soft references**: a reference that may outlive its target — to a record in the recycle bin (approval
  rounds and notifications of a deleted activity, transactions of a deleted line item, a deleted pinned
  plan, the source of a split activity) or to history (actor ids, which include `system`) — is a plain id
  without a foreign key. The list is in the data model.
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
  name and renamed when complete (an interrupted build never leaves a broken template). `NewEmptyDB(t)`
  gives an unmigrated database for migration tests. Under `-short` it skips; without the variable it
  **fails**, so CI can never skip store tests silently.
- **Rationale**: a copy from a template takes ~0.15 s, so every test can run in parallel on its own data.
  Templates of older migration sets stay on the server and may be dropped by hand
  (`DROP DATABASE mapp_template_…`).
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
as `""` in Go is `NULL` in SQL (so the foreign key holds); optional pointers are `NULL`. Ordered lists a
record owns are child tables with a `position`; the parent's `Put` replaces them. Lists keep the JSON
store's order: `ORDER BY length(id), id COLLATE "C"` (byte order, shorter ids first).

## R10 — Where `jsonb` is used, and why

Only for values whose shape is not relational or must stay exactly as written: Shell-attribute values (the
value's type depends on the attribute, seven types); recycle-bin snapshots and activity-log `changes`,
`path` and `items` (immutable copies of the past); approval-round level snapshots and rule facts (a frozen
copy of the workflow at submission); a workflow level's two conditions; saved-view state and All-work
column choices (screen state). Every `jsonb` column has a `CHECK (jsonb_typeof(…))`.

## R11 — Existing data and `store.Memo`

- JSON data folders are not converted (demo data only); the seed command loads the demo data into
  PostgreSQL.
- `store.Memo` (settings forms decoded once per request) has no PostgreSQL counterpart yet. The settings
  task first measures the pages that use it on PostgreSQL; only if they miss today's times does
  `database` gain a per-transaction memo with the same contract (Principle V: no optimisation without a
  measurement).
