# Feature Specification: Test Hygiene

**Feature Branch**: `feature/027-test-hygiene`

**Created**: 2026-10-26

**Status**: Phase 1 in review; phase 2 waits for feature 028

**Input**: Feature map 027: "Test hygiene: drop tests that prove nothing; coverage threshold once
SonarQube exists." The test suites keep only tests that catch a defect that could really happen
(constitution Principle VI, F10, B9; `backend/docs/GO_STANDARDS.md` §12), and they run fast and
deterministically. The coverage threshold waits for SonarQube and is a recorded follow-up only.

**References**: Constitution 4.2.0 — Principle VI (Meaningful Tests), V (file length), VIII
(comment budget; 027 is exempt from the whole-file rule of 4.2.0), B9 (Go testing), F10 (frontend
testing) · [research.md](research.md) (measurements R1–R8, every removal and its rule) ·
`backend/docs/GO_STANDARDS.md` §12 "Tests not to write"

## User Scenarios & Testing *(mandatory)*

The users of this feature are the developers, reviewers and AI agents who read, run and change the
tests. People using MAPP notice nothing.

### User Story 1 - Every test proves a behaviour someone relies on (Priority: P1)

As a reviewer I open any test and find that it proves a business rule, an access rule, a validation,
a status transition, an acceptance scenario, an API contract, an error the user sees, a state a
screen renders or a reproduced bug. Tests that restate the code, test React, Radix, the Go standard
library or generated code, only check that a callback was called, repeat another test at the same
level, snapshot the DOM or only raise coverage are gone, and for each one removed I can read which
rule of Principle VI it broke.

Mei reviews a change to the Button component. She no longer has to keep four "offers the %s variant"
tests green that asserted only that a button renders; the test left proves that Button never submits
a form by accident.

**Why this priority**: tests that prove nothing cost time to write, read, maintain and run, and
erode trust in the suite (Principle VI rationale).

**Independent Test**: pick any removed test from [research.md](research.md) R6 and find the rule it
broke and, for a duplicate, the test that still proves the behaviour; run the suites and see them
pass.

**Acceptance Scenarios**:

1. **Given** the test suites after the feature, **When** a reviewer reads the list of removed and
   merged tests, **Then** each entry names its Principle VI rule and, for a duplicate, the test
   that still proves the behaviour.
2. **Given** a required test (a business rule or calculation, an acceptance scenario, a P1 story's
   end-to-end journey, an endpoint's contract test, a scope-isolation test, a data view's state, a
   bug reproduction), **When** it was the only proof of its behaviour, **Then** it is still there.
3. **Given** a test kept although it looks like a forbidden kind, **When** the reviewer looks for
   the reason, **Then** research.md lists it under "kept despite doubt" with one line why.

---

### User Story 2 - The suites run fast and the same way every time (Priority: P1)

As a developer I run the backend tests of a package, or the whole web-app suite, and wait only for
work that proves something. No test waits a fixed time, every Go test that shares no state runs in
parallel, and expensive read-only fixtures are built once per test binary.

Anup runs `go test -race ./internal/auth/` and it finishes in a few seconds instead of half a
minute, because the contract validator no longer compiles all of `openapi.yaml` up front.

**Why this priority**: the backend `go test -race ./...` takes about 7.5 minutes in CI and the
web-app tests about 5 minutes; slow suites are run less often.

**Independent Test**: time the phase-1 packages and the web-app suite before and after (research
R1–R3); search the tests for fixed waits.

**Acceptance Scenarios**:

1. **Given** a test binary that uses the API test helper, **When** it starts, **Then** it no longer
   spends 13–30 seconds compiling every schema of the contract; each schema is compiled when a
   request first needs it.
2. **Given** a backend test of a phase-1 package that shares no state, **When** the package runs,
   **Then** the test runs in parallel with the others.
3. **Given** any test, **When** it needs time to pass, **Then** it advances a fake clock or waits for
   what a user would see; no test sleeps or waits a fixed time.

---

### User Story 3 - Phase 2 knows where the backend time goes (Priority: P2)

As the developer of phase 2 I read research.md and find, for every backend package, how long it
takes, which tests dominate and what to change, so the work after feature 028 is planned, not
rediscovered.

**Why this priority**: feature 028 changes the services, stores and the JSON test stand-in of every
business package at the same time; phase 1 must not touch them, but phase 2 should start with the
facts.

**Independent Test**: read research R4 and R7; every package of `go test ./...` appears with its
time and a recommendation.

**Acceptance Scenarios**:

1. **Given** research.md, **When** phase 2 starts, **Then** it lists every backend package with its
   time before and after phase 1, the slowest tests and the cause measured for each.

---

### User Story 4 - A coverage threshold the pipeline enforces (Priority: P3, deferred)

As the technical lead I want the pipeline to refuse a change whose new code is below 80% decision
coverage. This needs SonarQube's "coverage on new code"; it is out of scope until SonarQube exists
and is recorded as a follow-up.

### Edge Cases

- A test removed as a duplicate was the only test of a branch: the coverage drop is listed in
  research R5 with the code path, and the path is either proven elsewhere at another level or is
  dead code (recorded for removal).
- A test asserts a CSS-module class name because jsdom cannot see a colour or an alignment: kept
  where it is the only proof of a visual state the specification requires, and listed as kept
  despite doubt.
- Feature 028 changes a file this feature changes: phase 1 stays out of 028's packages; the
  coordinator moved `internal/platform/web` and `internal/platform/database` to phase 2 during the
  work.
- A component that no screen uses still has tests: the tests are kept and the dead code is recorded
  (research R6 "kept despite doubt").

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Every removed or merged test MUST be named, with the Principle VI rule it broke, in
  research.md or the commit that removes it; a duplicate MUST name the test that still proves the
  behaviour.
- **FR-002**: The tests Principle VI requires MUST stay: unit tests of every rule and calculation
  with edge and negative cases, at least one test per acceptance scenario, an end-to-end test per
  P1 story, a contract test per endpoint, a scope-isolation test per scoped store read, a test per
  state of a data view, and a reproducing test per fixed bug. When in doubt a test stays and is
  listed.
- **FR-003**: No test MAY wait a fixed time: Go tests do not call `time.Sleep`; web-app tests do not
  wait on a real timer; both advance a fake or injected clock or wait for an observable outcome.
- **FR-004**: Backend tests of the phase-1 packages that share no state MUST call `t.Parallel()`
  (B9).
- **FR-005**: Read-only fixtures that cost seconds MUST be built once per test binary: the contract
  validator compiles each schema on first use, and the demo data folder is loaded once and copied
  into each test's folder.
- **FR-006**: Coverage MUST NOT drop on rules: frontend branch and backend statement coverage are
  compared before and after; any drop is explained by removed tests that proved nothing or by dead
  code.
- **FR-007**: research.md MUST report the time of every backend package before and after phase 1,
  the slowest tests, their measured causes and the phase-2 recommendations.
- **FR-008**: Phase 1 MUST change only the web-app tests, the end-to-end tests and the backend
  packages feature 028 does not touch (`internal/auth`, `internal/samples`,
  `internal/platform/{errs,logging,telemetry,clock,ids,record,actor,apitest,race}`, the tests of
  `internal/app`, `cmd/…`). Phase 2 covers the other backend packages after 028 merges.
- **FR-009**: Every file a phase changes MUST stay within 300 non-blank lines (Principle V); new and
  changed lines MUST follow the comment budget and say why (Principle VIII; 027 is exempt from the
  whole-file rule).
- **FR-010**: No dependency MAY be added.
- **FR-011**: The coverage threshold of SonarQube's new-code gate is NOT part of this feature; it is
  recorded as a follow-up for when SonarQube exists.

### Key Entities

- **Removal record**: a removed or merged test, its file, its rule (R1 restates the code, R2 tests a
  language, framework, library or generated code, R3 only checks a call, R4 repeats a test at the
  same level, R5 snapshots the DOM, R6 only raises coverage) and, for R4, the test that still
  proves the behaviour.
- **Timing record**: a backend package or web-app file, its time before and after, and its slowest
  tests.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every removal in research R6 names its rule; 0 required tests removed without another
  proof of the same behaviour.
- **SC-002**: The sum of the backend package times falls by at least 25% (local measurement before
  phase 1: 955 s), and no package using the API test helper spends more than 2 seconds building the
  contract validator (today 13–30 s each).
- **SC-003**: 0 fixed waits in the tests (today 3 in the web app).
- **SC-004**: Frontend branch coverage stays within 0.5 percentage point of 85.26% and backend
  total statement coverage stays at 80.2%, every drop explained (R5).
- **SC-005**: Every P1 story keeps its end-to-end journey.

## Assumptions

- Feature 028 (query efficiency) runs at the same time and changes the business packages' services
  and stores, `internal/platform/store`, `memo`, `database` and `web`; phase 1 does not touch them,
  and phase 2 waits until 028 is merged.
- Local timings are taken on a shared 10-core laptop and vary with its load; CI (4 vCPU) is the
  reference for the pull request (research R1).
- The comment budget's whole-file rule (constitution 4.2.0) does not apply to 027; files already
  edited keep the comment cuts made.
- Out of scope: axe checks in every component test of a screen or dialog (F10; most screen tests
  have none today) — a separate follow-up, recorded in research R8; the SonarQube coverage gate.
