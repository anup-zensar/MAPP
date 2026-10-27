# Feature Specification: Backend Layout

**Feature Branch**: `feature/025-backend-layout`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "Backend layout (feature map row 025): bring the Go API to GO_STANDARDS §2
(layout), §3 (interfaces where they are consumed) and §11 (conventions and metrics), with behaviour and the
API contract unchanged. Every Go file within 300 non-blank lines, tests included, split by responsibility.
Each business area's service depends on a small Store interface it declares in store.go, with only the
methods it uses, instead of today's big repository interfaces. Then, as far as it goes cleanly: service
tests on small hand-written fakes instead of the JSON stand-in, deleting the JSON store, and singular
package names."

**References**: Constitution 4.2.0 — Principle V (simplicity, quality metrics, file length), VI (meaningful
tests), VIII (comments; feature 025 is exempt from the whole-file rule), B2 (layout, consumer-defined
interfaces), B9 (fakes, not mocks) · `backend/docs/GO_STANDARDS.md` §2, §3, §11, §12 · features 023
(PostgreSQL storage; the JSON store stays as the tests' stand-in until 025), 027 (test hygiene) and 028
(query efficiency: `internal/platform/apibench`, the per-request statement count)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A developer finds and reads any backend file in one sitting (Priority: P1)

As a developer new to MAPP I open any Go file in `backend/` and it holds one concept, named after it, and
fits within 300 non-blank lines, so I can read it whole and know where its neighbours are.

**Why this priority**: 113 files are over the limit today (the largest 1,201 lines); the constitution
never merges a file over 400 and asks a justification above 300.

**Independent Test**: count the non-blank lines of every Go file outside generated code.

**Acceptance Scenarios**:

1. **Given** the backend after this feature, **When** the non-blank lines of every Go file outside
   `internal/platform/database/sqlcdb` are counted, **Then** none is above 300.
2. **Given** a file that was split, **When** its new files are read, **Then** each holds one
   responsibility (a use case, a concept, the tests of one behaviour, a fixture) and is named after it.
3. **Given** the split code, **When** the whole test suite, the end-to-end suite and the contract check
   run, **Then** they pass unchanged: no behaviour and no endpoint changed.

---

### User Story 2 - Each service names exactly what it needs from storage (Priority: P1)

As a developer I open an area's `store.go` and see the small `Store` interface its service uses, so I know
what the service reads and writes and can give a test a short fake of it.

**Why this priority**: today's repository interfaces list every method any caller uses (up to 45); the
service depends on all of them and a fake must implement all of them (GO_STANDARDS §3).

**Independent Test**: for each area, every method of `Store` is called by the area's service code, and the
PostgreSQL store and the JSON stand-in both satisfy it.

**Acceptance Scenarios**:

1. **Given** any business area, **When** its `store.go` is read, **Then** it declares `Store` with only the
   methods the area's service code calls.
2. **Given** a method that only the wiring, the seed or another area's adapter uses, **When** it is looked
   for, **Then** it is in the consumer's own interface (`internal/stores/<area>.go` or the consuming
   area), not in the area's `Store`.
3. **Given** any service constructor, **When** its parameters are counted, **Then** they are at most four
   (the shared `history.Tools` carries the transaction, IDs, activity log and clock).

---

### User Story 3 - Service tests use small fakes, and the JSON store is gone (Priority: P2)

As a developer I read a service test and see a hand-written fake of the area's `Store`, not a JSON file
store standing in for PostgreSQL; and packages have singular names (`plan`, `campaign`, ...).

**Why this priority**: the target of GO_STANDARDS §2 and §12; it builds on stories 1 and 2.

**Independent Test**: no package imports `internal/platform/store`; package names are singular.

**Acceptance Scenarios**:

1. **Given** an area whose service tests run on fakes, **When** they run, **Then** they prove the same
   rules as before, with the same names.
2. **Given** every area on fakes, **When** the JSON store is deleted, **Then** the build, the tests and the
   end-to-end suite pass.
3. **Given** the singular rename, **When** it is reviewed, **Then** it is one mechanical commit (directory
   moves and import paths), with no other change.

### Edge Cases

- A file whose single responsibility is longer than 300 lines (a route table, a long table-driven test):
  split it by the sub-areas it covers, keeping one table assembled from parts.
- Code moved to a new file counts as new for CI's lint (new and changed lines only): old findings in moved
  code are fixed in this feature, never suppressed beyond the "needs justification" band.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every Go file outside generated code MUST be within 300 non-blank lines, tests included, split
  by responsibility (Principle V).
- **FR-002**: Each business area's service MUST depend on a `Store` interface declared in its `store.go`
  with only the methods its service code calls; the PostgreSQL store and, while it exists, the JSON stand-in
  satisfy it; every service constructor takes at most four parameters (B2, Principle V).
- **FR-003**: Behaviour, the API contract (`api/openapi.yaml`), the SQL, the migrations and the generated
  queries MUST NOT change.
- **FR-004**: Every existing test MUST keep proving what it proves today, under its name; feature 027's
  `t.Parallel` and `apitest.CopyDemoData` and feature 028's benchmarks and statement count keep working.
- **FR-005** (P2): Service tests SHOULD run on hand-written fakes of the area's `Store`; once none uses it,
  the JSON store (`internal/platform/store`, each area's `store_json*.go`) SHOULD be deleted.
- **FR-006** (P2): Package names SHOULD be singular, renamed in one mechanical commit.
- **FR-007**: The backend gates MUST pass: format, vet, lint on new lines, sqlc (no diff) and vet, tests with
  the race detector and with `-short`, govulncheck, the end-to-end suite and the contract check.

## Success Criteria *(mandatory)*

- **SC-001**: 0 Go files over 300 non-blank lines (113 today).
- **SC-002**: 0 service constructors over four parameters; each area's `Store` holds only methods its
  service calls.
- **SC-003**: The test suite, the end-to-end suite and `api:check` pass unchanged.
- **SC-004** (P2): 0 imports of `internal/platform/store`; 0 plural package names.
