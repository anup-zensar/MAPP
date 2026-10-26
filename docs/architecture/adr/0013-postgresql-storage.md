# ADR-0013: PostgreSQL storage

**Status:** Accepted · **Date:** 2026-10-26 · **Supersedes:** ADR-0004 (JSON file store now, PostgreSQL later)

## Context
Constitution 3.0.0 asks for PostgreSQL in every environment — developer machines, CI and later Azure —
through pgx, sqlc and goose, with Docker optional (Principle IV, B1, B5). Since feature 001 every record
type has been stored in a JSON file store behind one repository interface per area (ADR-0004). Feature 023
specifies the move (`specs/023-postgresql-storage/`).

## Decision
- **PostgreSQL 18**, natively installed or through the optional `compose.yaml`, on `127.0.0.1`. The API and
  the seed read `DATABASE_URL`; the scope-isolation tests read `TEST_DATABASE_URL`.
- **One store per area**: each area's `store_postgres.go` implements its existing repository interface, so
  services keep their rules. `internal/stores` gathers the stores and the `Transactor` the API and the seed
  run on.
- **Schema by goose**: 16 embedded, forward-only migrations (50 tables, the scope functions, write-once
  triggers on the activity log, approval history and rate imports). The seed command creates the
  database when it is missing and migrates it; the API never migrates and refuses a database behind the
  latest migration.
- **Queries by sqlc** v1.31.1 (run with `go run`), generated into the committed `sqlcdb` package; CI fails
  when `sqlc generate` leaves a change.
- **Transactions**: `database.DB` implements `store.Transactor`. `Write` runs one transaction at a time
  behind a transaction-scoped advisory lock, so the services' read-check-write rules (versions, uniqueness,
  counters) stay correct without rewriting them; `Read` is a read-only snapshot in which what is worked out
  from the data (`store.Memo`) is kept for the rest of the transaction. Timestamps are scanned in UTC.
- **Scope in SQL**: `plan_in_scope()`, `activity_in_scope()` and `plan_in_permission_scope()` mirror the
  Go rules of `authz.Scope`; plan-owned stores add `…InScope` reads, and the plan lists and the Approvals
  board read through them.
- **Demo data**: JSON data folders are not converted; `go run ./cmd/seed` loads the demo data, and
  `-reset` empties every table with `TRUNCATE` first, and `-if-empty` leaves a database with data alone.

## Consequences
- Developers need a PostgreSQL 18 server and a role that may create databases (`CREATEDB`); nothing needs
  Docker. The end-to-end tests run on their own database, `mapp_e2e`.
- Writes do not run in parallel. Every Write raises the shared activity-log counter anyway, so this costs
  nothing today; revisit with a measurement if hosted load needs it.
- The JSON file store stays only as the stand-in of service, handler and contract tests until feature 025
  replaces it with fakes and deletes it.
- Many pages read reference records one at a time (a person, a finance budget, a workflow), which was free
  in memory and is a query each on PostgreSQL. The seeded pages answer within 0.4 seconds; the scale
  targets (004 SC-009, 009 SC-004) need a measurement with large data and, if missed, batched reads.
- Follow-up: database tracing (`otelpgx`) with the OpenTelemetry providers of ADR-0011.
