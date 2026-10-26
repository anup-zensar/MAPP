# Implementation Plan: PostgreSQL Storage

**Branch**: `feature/023-postgresql-storage` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/023-postgresql-storage/spec.md`

## Summary

A new `internal/platform/database` package opens the pgx pool, applies embedded goose migrations under a
session lock and runs `Write` transactions one at a time and snapshot `Read` transactions, both carried
in the context; it
implements today's `store.Transactor`, so services keep their code. sqlc compiles one `.sql` file per area
into committed `sqlcdb` code, and `dbtest` gives each store test its own database copied from a migrated
template. Each area gains a `store_postgres.go` beside its JSON repository, implementing the same
interface; when all exist, `internal/app` and the seed command switch to PostgreSQL at once and the JSON
store is removed ([research](research.md) R1–R11, [data model](data-model.md)).

**Built on this branch**: the foundation, the pilot area (samples, R8) and every other store, with scoped
reads for plan-owned data (R12); the application still runs on the JSON store until the switch (T025–T028).

## Technical Context

**Language/Version**: Go 1.27 (`toolchain go1.27.3`); PostgreSQL 18

**Primary Dependencies**: pgx v5.11.0, goose v3.29.0, go-cmp v0.7.0 (tests); sqlc v1.31.1 and
golangci-lint v2.15.0 as pinned tools run with `go run` (R1)

**Storage**: PostgreSQL 18 on `127.0.0.1` — native install or the optional `compose.yaml`; `DATABASE_URL`
for the API and seed, `TEST_DATABASE_URL` for store tests

**Testing**: `go test -race ./...` with `TEST_DATABASE_URL`; `go test -short ./...` without a database;
store tests per interface promise, constraint and race (FR-004–FR-008)

**Target Platform**: developers' macOS, Windows and Linux machines; GitHub Actions `ubuntu-latest`

**Project Type**: Web application (Go API + React web app); this feature changes the API's storage only

**Performance Goals**: today's page times on the demo data (SC-005)

**Constraints**: no change to services, handlers, the contract or the web app (FR-008); no Docker in any
test (FR-010); new files ≤ 300 non-blank lines and ≤ 5% comment lines; release age for every module (R1)

**Scale/Scope**: 50 tables in 15 migrations and 3 scope functions; 15 stores (14 areas and the id counters)
with 158 repository methods plus the scoped reads

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec FR-001–FR-012 with acceptance scenarios; branch `feature/023-postgresql-storage` | Pass |
| III Documented design | Plan, research, data model; ADR-0004 superseded by a new ADR in T030 | Pass |
| IV PostgreSQL everywhere | pgx pool, sqlc, goose; Docker optional | Pass |
| V Simplicity | One platform package, one store file per area, no new layer; files within 300 lines | Pass |
| VI Meaningful tests | Store tests per interface promise on real PostgreSQL; racing-save, rollback, UTC and constraint tests each fail without the code they prove (checked by mutation in the pilot) | Pass |
| VII Security | Scope in SQL for plan-owned data (FR-009, after 022); no DSN in errors; release age checked (R1) | Pass |
| VIII Clean construction | Comment budget met in every new file; generated `sqlcdb` excluded | Pass |
| B1 Stack | pgx, sqlc, goose, go-cmp — all named in B1; reasons in R1 | Pass |
| B2 Layout | `internal/platform/database/{migrations,queries,sqlcdb,dbtest}`; `store_postgres.go` per area | Pass |
| B3 Lifecycle | Pool opened once, closed on shutdown (wiring task) | Pass |
| B5 Data | SQL in `.sql` via sqlc; goose forward-only, embedded; `bigint` minor units (rates in millionths, percentages in hundredths), `date`, `timestamptz` UTC; service-owned transactions; change and log entry in one transaction | Pass |
| B9 Testing | `dbtest` per-test databases from a migrated template, skipped under `-short`, no Docker | Pass |
| Gates 3–4 | `sqlc generate` diff and migrations on an empty database run in CI (T029) | Pass |
| Governance → Alignment | No area is switched as a side effect: stores are added beside the JSON ones and switched together in one task | Pass |

**Post-design re-check (after Phase 1)**: unchanged — Pass.

### Threat model (STRIDE) for the storage change

| Threat | Where | Response |
|---|---|---|
| Tampering: SQL injection | every query | sqlc parameters only; the test helper's DDL quotes identifiers it generates itself |
| Information disclosure: credentials in logs | `Open`, `dbtest` | errors never include the URL; `TEST_DATABASE_URL` is named, never printed |
| Information disclosure: another market's plans | plan-owned reads | scope filter in SQL plus an isolation test per read (FR-009) |
| Tampering: lost update between concurrent saves | every `Write` | one Write at a time (R2); lock test and racing-save test |
| Repudiation: a change without its log entry | every service | change and log entry in one `Write`; append-only trigger on `activity_log` |
| Denial of service: connection exhaustion | pool | one pool per process; `pool_max_conns` in `DATABASE_URL` per environment |

### Risk model

| Risk | Response | Owner |
|---|---|---|
| A service relies on JSON-store behaviour no interface states (ordering, nil lists, `store.Memo`) | The full backend and end-to-end suites run on PostgreSQL before the JSON store is removed (T025–T028); R9 conventions | Team |
| One writer at a time limits write throughput | As today on the JSON store; every Write raises the shared `LOG` counter anyway (R2); revisit with a measurement if Azure load needs it | Team |
| A foreign key refuses a delete the service allows | Each area task tests its deletes against the constraints (data model §4) | Team |
| 022/024 change the wiring and the store interfaces while this runs | Area tasks for plan-owned data and the switch wait for them (tasks.md) | Product owner |

## Migration path

1. **Foundation** (done): `database` (`Open`, `Migrate`, `Write`, `Read`, `Queries`), `dbtest`, `sqlc.yaml`.
2. **Pilot** (done): samples — migration 00001, queries, `store_postgres.go`, store tests.
3. **Remaining areas**, in migration order (data model §2): platform (ids counters, history) → people and
   auth → settings → access → plans → campaigns → workflows → activities and line items → approvals →
   budgets → financials → All work. Each adds its migration, queries, `store_postgres.go` and store tests;
   plan-owned areas take the scope in their store methods once 022 has settled it.
4. **Switch** (one task, after 022 and 024): `internal/app` opens the pool from `DATABASE_URL`, refuses a
   database behind the latest migration, wires every PostgreSQL store and `database.DB` as the
   `Transactor`; `cmd/seed` migrates and loads the demo data; end-to-end tests run on PostgreSQL. Areas
   switch together because one `Write` spans several areas' stores.
5. **Remove the JSON store**: `internal/platform/store`, every `store_json.go`, the data-folder setting and
   `-reset`'s file handling.

## Project Structure

### Documentation (this feature)

```text
specs/023-postgresql-storage/
├── spec.md          # stories, FR-001–FR-012
├── research.md      # R1–R11
├── data-model.md    # every table, constraint and soft reference
├── plan.md          # this file
└── tasks.md         # dependency-ordered tasks
```

### Source Code (this branch)

```text
backend/
├── go.mod / go.sum                               # pgx, goose, go-cmp (R1)
├── sqlc.yaml                                     # R7
└── internal/
    ├── platform/database/
    │   ├── database.go                           # Open, Close, Write, Read, Queries
    │   ├── migrate.go                            # Migrations, Migrate
    │   ├── database_test.go                      # rollback, read-only, UTC, concurrent migrations
    │   ├── migrations/00001_sample_records.sql
    │   ├── queries/samples.sql
    │   ├── sqlcdb/                               # generated, committed
    │   └── dbtest/dbtest.go                      # NewDB, NewEmptyDB
    └── samples/
        ├── store_postgres.go                     # PostgresStore implements Repository
        └── store_postgres_test.go
```

**Structure Decision**: the target layout of GO_STANDARDS §2 for the new package; the samples area keeps
its current files (`repository.go` is renamed `store.go` by 025).

## Complexity Tracking

| Deviation | Why needed | Simpler alternative rejected because |
|---|---|---|
| `Write` takes a transaction-scoped advisory lock so Writes run one at a time (not in GO_STANDARDS §6's sample) | keeps every service's read-check-write rule correct without rewriting it, as the JSON store did (R2) | `READ COMMITTED` alone lets two saves both pass a version or uniqueness check; `SERIALIZABLE` with retries starved under two writers (measured, R2) |
