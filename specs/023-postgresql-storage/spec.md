# Feature Specification: PostgreSQL Storage

**Feature Branch**: `feature/023-postgresql-storage`

**Created**: 2026-10-26

**Status**: Draft

**Input**: User description: "PostgreSQL storage (alignment roadmap, feature map row 023): pgx, sqlc and
goose replace the JSON file store. Every record type stored today gets a relational table; services keep
their rules and own their transactions; store tests run on real PostgreSQL without Docker. Build the
foundation and one pilot area first, with the application still on the JSON store."

**References**: Constitution 4.0.0 — Principle IV (PostgreSQL in every environment), VI (only end-to-end
and scope-isolation tests use a database; no Docker), VII (scope isolation, release age), B1, B2, B5, B9, Governance → Alignment ·
`backend/docs/GO_STANDARDS.md` §6, §12 · ADR-0004 (JSON file store now, PostgreSQL later) ·
`docs/product/domain-model.md` · features 001 (store, record bookkeeping), 011 (activity log, recycle bin),
014 (scopes) · feature map rows 022 (API v1), 024 (Gin), 025 (backend layout)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - MAPP keeps its data in PostgreSQL (Priority: P1)

As a developer I start PostgreSQL (natively or with `docker compose up -d postgres`), run the seed
command and the API with `DATABASE_URL`, and MAPP behaves exactly as it does on the JSON store today: the
same screens, the same answers, the same activity log and recycle bin.

**Why this priority**: the constitution requires PostgreSQL in every environment; the JSON store is the
largest remaining deviation.

**Independent Test**: seed an empty database, run the end-to-end suite against the API on PostgreSQL;
every test passes unchanged.

**Acceptance Scenarios**:

1. **Given** an empty PostgreSQL database, **When** `go run ./cmd/seed` runs, **Then** every migration is
   applied and the demo data is loaded; running it again changes nothing.
2. **Given** a seeded database, **When** the API starts with `DATABASE_URL`, **Then** every endpoint answers
   as it does on the JSON store (contract and end-to-end tests pass unchanged).
3. **Given** a database behind the latest migration, **When** the API starts, **Then** it stops with a
   message saying to run the seed command; it never migrates by itself.

---

### User Story 2 - Saved data stays correct when people work at the same time (Priority: P1)

As a planner I trust that a change and its activity-log entry are saved together or not at all, that a
save from an out-of-date copy never overwrites someone else's change, and that the database itself
refuses data that breaks a rule it can check.

**Why this priority**: the JSON store gave every save exclusive access; PostgreSQL runs saves
concurrently, so this guarantee must be rebuilt, not assumed.

**Independent Test**: the service tests prove each version check and that a failed save keeps nothing; the
end-to-end tests run those services on PostgreSQL. The transaction, the lock and the constraints are not
tested on a database of their own (constitution 4.0.0, Principle VI).

**Acceptance Scenarios**:

1. **Given** a save whose function fails after changing data, **When** the transaction ends, **Then**
   nothing it changed is stored, including the changes of services it called.
2. **Given** two people who both opened version 1 of a record, **When** both save at once, **Then** one
   save is stored as version 2 and the other is refused with `version_conflict`.
3. **Given** a change that breaks a stored invariant (a child without its parent, a duplicate name the
   rule forbids), **When** it is committed, **Then** the database refuses it.

---

### User Story 3 - Scope-isolation tests run on real PostgreSQL without Docker (Priority: P2)

As a developer I run `go test ./...` with `TEST_DATABASE_URL` pointing at any PostgreSQL 18 server; each
scope-isolation test gets its own fresh, migrated database, dropped afterwards. `go test -short ./...` needs no
database at all.

**Why this priority**: the scope filter is a security boundary written in SQL, which a fake cannot check
(Principle VI).

**Independent Test**: run the suite twice in parallel processes against one server; both pass and no
test database is left behind.

**Acceptance Scenarios**:

1. **Given** `TEST_DATABASE_URL`, **When** scope-isolation tests run in parallel, **Then** each test has its own
   database copied from a migrated template and dropped when it ends.
2. **Given** `-short`, **When** the suite runs, **Then** database tests are skipped and nothing connects to a
   database.
3. **Given** no `TEST_DATABASE_URL` and no `-short`, **When** a database test runs, **Then** it fails with a
   message naming the variable — it is never skipped silently.

### Edge Cases

- A JSON data folder from before this feature: not converted; the demo data is re-seeded (Assumptions).
- Two seed runs at once: migrations wait for each other on a PostgreSQL session lock; none is applied twice.
- Two saves at the same moment: the second waits until the first has committed, then sees its change
  (Writes run one at a time, as on the JSON store); reads never wait.
- A record that refers to one in the recycle bin (a notification about a deleted activity, a transaction
  linked to a deleted line item, a pinned plan that was deleted): the reference is kept as a plain id.
- A name that is unique "ignoring case" by the service's rule: the database index is never stricter than
  the rule, so it cannot refuse a name the service accepts.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The API and the seed command MUST open one pgx pool per process from `DATABASE_URL`, ping it
  at start-up and close it on shutdown; the connection string MUST NOT appear in logs or errors.
- **FR-002**: The schema MUST be created and changed only by goose SQL migrations embedded in the binary,
  forward-only and never edited once merged. The seed command applies them; concurrent runs wait on a
  PostgreSQL session lock. The API never migrates; it refuses to start on a database behind the latest
  migration.
- **FR-003**: All application SQL MUST live in `.sql` files compiled by sqlc (pinned version, run with
  `go run`) into committed code under `internal/platform/database/sqlcdb`; no SQL is built by
  concatenation. The only exceptions take identifiers, which cannot be query parameters: the test
  helper's `CREATE`/`DROP DATABASE`, the seed's `CREATE DATABASE` for a missing database and its
  `TRUNCATE` of the tables the catalogue lists, each identifier quoted by pgx.
- **FR-004**: Every record type stored today MUST have a table as in [data-model.md](data-model.md): keys,
  foreign keys, `NOT NULL`, `CHECK` and unique constraints for every invariant the database can check
  without being stricter than the service's rule.
- **FR-005**: Money MUST be stored as `bigint` minor units (exchange rates in millionths, percentages in hundredths),
  calendar dates as `date` and timestamps as `timestamptz`; timestamps MUST come back in UTC.
- **FR-006**: Services MUST own their transactions: `Write` runs one transaction carried in the context,
  which every store and the activity log called with that context join; a change and its log entry
  commit or roll back together; Writes run one at a time, so a stale save is refused with
  `version_conflict` and never overwrites a newer one.
- **FR-007**: `Read` MUST run in a read-only transaction that sees one consistent snapshot; a `Write` inside
  it is refused.
- **FR-008**: Each area's PostgreSQL store MUST implement the area's existing store interface with the same
  observable behaviour as its JSON implementation (not-found answers, replace-on-save, ordering by id with
  shorter ids first, lists never nil), so services and handlers do not change.
- **FR-009**: Every store query on plan-owned data MUST filter by the actor's scope in SQL, and each such
  read MUST have a scope-isolation test (B5); person-owned records are filtered by the person in SQL.
- **FR-010**: Only the scope-isolation tests (one per scoped area, covering each of its `…InScope` reads)
  and the end-to-end tests MUST use a database; queries, constraints, migrations and the write lock are not
  tested on one. A scope-isolation test gets a fresh database, copied from a template migrated once per
  migration set on the server of `TEST_DATABASE_URL`, dropped when the test ends; skipped under `-short`;
  no test requires Docker.
- **FR-011**: Once every area has its PostgreSQL store, the API, the seed command and the end-to-end tests
  MUST run on PostgreSQL, and the JSON store and its data folder setting MUST be removed from them. The
  JSON store stays only as the stand-in of service and handler tests until feature 025 replaces it with
  fakes.
- **FR-012**: The CI pipeline MUST start PostgreSQL 18 for the `backend` and `e2e` checks, apply the
  migrations to an empty database, and fail when `sqlc generate` leaves a diff (quality gates 3 and 4).

### Key Entities

- **Migration**: a numbered SQL file under `internal/platform/database/migrations`, applied once, in order.
- **Table**: one per stored record type, plus child tables for the ordered or many-valued lists a record
  owns (data model §2).
- **Transaction**: a unit of work carried in the context; `Write` (one at a time) or `Read` (read-only
  snapshot).
- **Test database**: a per-test copy of the migrated template on the test server.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With the API on PostgreSQL, 100% of today's end-to-end tests pass without changes to their
  assertions, and the backend and contract tests still pass.
- **SC-002**: Every scoped area has one scope-isolation test on PostgreSQL that covers each of its
  `…InScope` reads; no other Go test opens a database.
- **SC-003**: Two concurrent saves from the same version never both succeed: every version check runs
  inside a `Write`, and Writes run one at a time (R2); the version checks are proved by the service tests.
- **SC-004**: `go test -short ./...` passes on a machine with no PostgreSQL; `go test ./...` passes against a
  native install and against the Compose service alike.
- **SC-005**: Pages answer within today's times on the seeded demo data (the 2-second activity-log
  target of feature 011 included).

## Assumptions

- The JSON data folders hold demo data only, so they are not converted: the seed command loads the demo
  data into PostgreSQL. `-reset` empties every table with `TRUNCATE` (which the write-once triggers let
  through) before loading.
- PostgreSQL 18 runs on `127.0.0.1` for development (native or Compose), with a role that may create
  databases (`CREATEDB`) for the test helper.
- Feature 022 settles how the actor's scope reaches services and the 404 rule for records outside it;
  the scoped store interfaces of plan-owned areas follow it. Feature 024 rewrites the HTTP wiring in
  `internal/app`, where the pool is opened and `/readyz` pings it. Feature 025 renames `repository.go`
  to `store.go` and the packages to singular names.
- Until an area is switched, it keeps its JSON store (Governance → Alignment); the PostgreSQL stores are
  added beside the JSON ones and switched over together in the wiring task.
