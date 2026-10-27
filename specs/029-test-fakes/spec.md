# Feature Specification: Test Fakes

**Feature Branch**: `feature/029-test-fakes`

**Created**: 2026-10-27

**Status**: Draft

**Input**: User description: "Test fakes (feature map row 029): every service, handler and contract test
runs on hand-written in-memory fakes, and the JSON file store is deleted: `internal/platform/store`, every
`store_json*.go`, `stores.JSON` and anything only they use. Behaviour and the API contract don't change.
Every existing test keeps its name, requirement ID and assertions unless Principle VI forbids it."

**References**: Constitution 4.3.0 — Principle VI (meaningful tests: only end-to-end and scope-isolation
tests touch a database), VIII (comments; the whole-file rule applies to every file this feature touches),
B2 (consumer-defined interfaces), B9 (small hand-written fakes, no mocking frameworks) ·
`backend/docs/GO_STANDARDS.md` §3 (interfaces where consumed), §6 (transactions), §12 (fakes, HTTP tests) ·
feature 023 (PostgreSQL; the JSON store stayed as the tests' stand-in) · feature 025 research R4 (this
work, left open)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Service tests run on fakes of the service's own Store (Priority: P1)

As a developer I read a service test and see the service built on an in-memory fake of the `Store` it
declares, with a fake transaction, not on a JSON file store in a temporary folder.

**Why this priority**: constitution VI and B9 require fakes for every test but end-to-end and
scope-isolation tests; 31 service-test files build their services on the JSON store today.

**Independent Test**: no test file imports `internal/platform/store` or calls a `NewJSONStore`.

**Acceptance Scenarios**:

1. **Given** any business area, **When** its service tests run, **Then** they run on an in-memory fake of
   its `Store` and prove the same rules as before, under the same names and requirement IDs.
2. **Given** a service rule that relies on a Write changing nothing when it fails (a refused restore, a
   failed import), **When** its test runs on the fake transaction, **Then** nothing it changed stays.
3. **Given** a rule about scope (a record outside the actor's scope looks missing), **When** it runs on a
   fake, **Then** the fake filters by the same scope rule as the PostgreSQL store.

---

### User Story 2 - Handler and contract tests run the whole API in memory (Priority: P1)

As a developer I run a handler, contract or access-matrix test and the real API answers on in-memory fakes
holding the demo data, with every request and answer checked against `api/openapi.yaml`.

**Why this priority**: about 80 test files use `internal/platform/apitest`, which runs the whole API on the
JSON store; the access matrix and the contract checks depend on it.

**Independent Test**: `apitest.New` builds the API on an in-memory store set; `go test -short ./...` runs
with no database and writes no data folder.

**Acceptance Scenarios**:

1. **Given** `apitest.New`, **When** a test starts, **Then** the API runs on a fresh in-memory copy of the
   demo data, loaded once per test binary by `seed.Load`, and the clock starts at 12 Mar 2026 09:00 UTC.
2. **Given** every existing handler, contract and access test, **When** it runs on the in-memory set,
   **Then** it passes with the same names and assertions.

---

### User Story 3 - The JSON store is gone (Priority: P1)

As a developer I find one storage implementation per area, on PostgreSQL, and test fakes in test-only
places; nothing ships a second database.

**Why this priority**: the JSON store is about 2,500 lines that the API no longer runs on.

**Independent Test**: `internal/platform/store`, every `store_json*.go`, `stores.JSON` and the JSON ID
counters no longer exist; the API, the seed and the end-to-end suite run on PostgreSQL unchanged.

**Acceptance Scenarios**:

1. **Given** the backend after this feature, **When** it is searched, **Then** no file of the JSON store is
   left and no package imports `internal/platform/store`.
2. **Given** the API, the seed and the end-to-end suite, **When** they run on PostgreSQL, **Then** they
   pass unchanged.

### Edge Cases

- An area's own internal tests cannot import its `<area>test` package (Go forbids the import cycle): they
  keep a fake of the area's `Store` in a `_test.go` file (research R1).
- A test of the JSON store itself (reading older rows, files on disk, a broken data folder) tests code
  that is deleted: it goes with it (Principle VI).
- A test that two servers share data, or that readiness fails, keeps its assertion on the in-memory set.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every service, handler and contract test MUST run on hand-written in-memory fakes, with no
  database and no files (Principle VI, B9).
- **FR-002**: Each business area MUST have an in-memory fake of its store in a test-only place: records by
  id, lists ordered by id as PostgreSQL orders them and never nil, the same scope filtering, a missing
  record reported as missing; version conflicts stay the services' rule on the stored version.
- **FR-003**: The fake transaction MUST behave like `database.DB`: Writes one at a time, a Write inside a
  Write joins it, a change inside a Read is refused, and a failed Write puts back everything it changed,
  readable-ID counters included.
- **FR-004**: `apitest` MUST run the whole API on an in-memory store set loaded with the demo data by
  `seed.Load`, a fresh copy per test, so the access matrix and the contract checks keep working.
- **FR-005**: Each service MUST declare the `Transactor` it needs next to its `Store` (GO_STANDARDS §6);
  `history.Tools` carries history's.
- **FR-006**: The JSON store MUST be deleted: `internal/platform/store`, every `store_json*.go`,
  `stores.JSON`, the JSON ID counters and anything only they use.
- **FR-007**: Every existing test MUST keep its name, requirement ID and assertions, unless Principle VI
  forbids it; a removed test names the rule.
- **FR-008**: Behaviour, the API contract, the SQL, the migrations and the generated queries MUST NOT
  change; no test is added on a database (only scope isolation and end-to-end touch PostgreSQL).
- **FR-009**: The backend gates MUST pass: format, vet, lint on new lines, sqlc (no diff) and vet, tests
  with the race detector and with `-short`, the timing tests, govulncheck, the contract check and the
  end-to-end suite.

## Success Criteria *(mandatory)*

- **SC-001**: 0 imports of `internal/platform/store`; 0 `store_json*.go` files; 0 calls of `stores.JSON`.
- **SC-002**: The number of Go tests is unchanged except the tests of the deleted JSON store (research R5).
- **SC-003**: The backend tests take less time than before (research R5 records both).
- **SC-004**: Every gate of FR-009 passes.
