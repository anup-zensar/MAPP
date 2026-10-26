---

description: "Tasks for 027 Test Hygiene"
---

# Tasks: Test Hygiene

**Input**: Design documents from `specs/027-test-hygiene/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md)

**Tests**: this feature changes tests. Every remaining test must pass after each task; a removed or
merged test is recorded with its Principle VI rule in its commit and in research R6 (FR-001).

**Done after each task**: the touched package's or folder's tests, lint and format green; changed
files ≤ 300 non-blank lines; new lines within the comment budget.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files)
- **[Story]**: US1 tests that prove something, US2 fast and deterministic, US3 timing report

---

## Phase 1 (now; outside the packages feature 028 changes)

### Measurements

- [X] T001 [US3] Baseline: backend `go test -race -json` per package and per test, frontend
  `npm run test:coverage` (junit and coverage), end-to-end run; CI develop step times (research R1–R4)
- [X] T002 [US2] Measure the contract validator and the demo-data seeding of `apitest` alone, with and
  without `-race` (R2)

### Backend (US1, US2)

- [X] T003 [US2] `internal/platform/apitest`: compile schemas on first use, check one request at a
  time, copy the demo data seeded once per binary; validate the contract document directly (FR-005)
- [X] T004 [P] [US1] `internal/platform/{errs,actor,record,ids,clock}`: remove restating tests; the
  rest call `t.Parallel` (FR-001, FR-004)
- [X] T005 [P] [US1] `internal/auth`: one handler case for incorrect credentials; all tests parallel;
  `service_test.go` split within 300 lines
- [X] T006 [P] [US1] `internal/samples`: remove the two handler tests repeating service rules and the
  repeated taken-name case; all tests parallel; `service_test.go` split within 300 lines
- [X] T007 [P] [US1] `internal/app`: remove the routes duplicate, merge the environment tests; all
  tests parallel except the logger test; `app_test.go`, `access_enforcement_test.go` and
  `usage_test.go` split within 300 lines
- [X] T008 [US1] `cmd/api`, `cmd/seed`, `internal/platform/{logging,telemetry,race}` reviewed: nothing
  to remove (logging and telemetry already parallel)

### Web app (US1, US2)

- [X] T009 [US1] Review every test file in six slices against VI and F10 (R6)
- [X] T010 [P] [US1] `ui/`, `api/`, `session/`, `testing/`: remove framework and restating tests
- [X] T011 [P] [US1] `features/dev`, `features/recycle-bin`, `features/settings`: one gallery render;
  remove duplicates; merge dialog success tests into the page tests; smaller OptionsCard pages
- [X] T012 [P] [US1] `app/`, `features/home`, `features/all-work`: remove duplicates and templates
- [X] T013 [P] [US1] `features/plans`, `features/global-budgets`, `features/notifications`
- [X] T014 [P] [US1] `features/activities`, `features/line-items`, `features/financials`
- [X] T015 [P] [US1] `features/approvals`, `features/approval-workflows`, `features/people`,
  `features/activity-log`
- [X] T016 [US2] Replace the three fixed real waits with fake time or an awaited outcome (FR-003)
- [X] T017 [US1] Second pass over the folders with no removal yet: `approval-workflows`,
  `global-budgets`, `notifications`, `sign-in`, `placeholders`, `format`, the rest of `settings`
  and `session`
- [X] T018 [US1] End-to-end review: no fixed waits; every P1 story keeps its journey (R4, SC-005)

### Close phase 1

- [X] T019 [US3] After-measurements: backend phase-1 packages and full run, frontend coverage run,
  end-to-end run; fill research R9
- [X] T020 Gates: `npm run check`, `npm run test:coverage`, `npm run build`, `npm run test:e2e`;
  backend gofmt/goimports, `golangci-lint run --new-from-merge-base=origin/develop`, `go vet`,
  `go test -race ./...`
- [X] T021 Update `docs/sdd/feature-map.md` row 027; merge `origin/develop`; pull request into develop (#12, merged)
- [X] T021a [US3] Read-only review of every remaining package; the phase-2 plan is research R10

---

## Phase 2 (after feature 028 is merged; per-package plan in research R10)

- [ ] T022 [US2] Stress tests: skip under the race detector (or move to 028's benchmarks) and run
  them without `-race` in a CI step of their own (R7.1)
- [ ] T023 [P] [US1] Review `internal/{access,activities,allwork,approvals,budgets,campaigns,
  financials,history,home,lineitems,people,plans,rollup,settings}` and `seed`: handler tests that
  repeat service rules, restating and library tests (R7.4)
- [ ] T024 [P] [US2] `t.Parallel` in those packages where tests share no state (R7.3)
- [ ] T025 [P] [US2] `seed`: one seeded data folder per test binary (R7.5)
- [ ] T026 [P] [US1] `internal/platform/{store,memo,money,date,authz,sheet,web,database}` review
  (R7.6); the JSON stand-in's cost goes with feature 025's fakes (R7.2)
- [ ] T027 [US3] Re-measure and update research; coverage before and after

## Follow-ups (not 027)

- Coverage gate in SonarQube once it exists (US4, FR-011)
- Axe checks in every screen and dialog component test (R8)
- Dead code and Home test gaps listed in R8
