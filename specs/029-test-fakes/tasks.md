# Tasks: Test Fakes

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md)

`[x]` done, `[ ]` open. Each area task is done by one helper in branch `029-<area>` and merged by the lead.

## Phase 1: Setup

- [x] T001 Branch `feature/029-test-fakes` from `develop`; spec, checklist, plan, research, tasks; feature
  map row 029
- [x] T002 Measure the backend tests before (research R5)

## Phase 2: Foundation (lead)

- [x] T003 `internal/platform/fake`: `DB` (Read, Write, rollback, Copy), `Table[T]`, `Counters`, with
  tests of what every other test relies on (FR-003)
- [x] T004 history: `Transactor` in `store.go`, `history.Tools.Tx`, `historytest`, the internal fake and
  the fixtures (FR-001, FR-002, FR-005; the worked example)

## Phase 3: User Story 1 — service tests on fakes (P1)

Per area: `<area>test`, `store_fake_test.go`, `Transactor`, fixtures and tests on fakes; whole-file
comment budget for every touched file (FR-001, FR-002, FR-005, FR-007).

- [x] T005 [US1] settings (and `formOf` on `memo.Get`)
- [x] T006 [US1] people
- [x] T007 [US1] access
- [x] T008 [US1] plan
- [x] T009 [US1] campaign
- [x] T010 [US1] approval
- [x] T011 [US1] lineitem
- [x] T012 [US1] budget
- [x] T013 [US1] financials
- [x] T014 [US1] sample and auth
- [x] T015 [US1] allwork and home
- [x] T016 [US1] activity (all but the demo-data fixture)

## Phase 4: User Story 2 — the whole API in memory (P1)

- [x] T017 [US2] `internal/stores/storestest`: `New` and `Demo` (seed.Load once, a copy per call);
  `stores.Transactor` (FR-004, FR-005)
- [x] T018 [US2] `apitest` on `storestest`; its options for shared data and an empty set (FR-004)
- [x] T019 [US2] activity's demo-data fixture, `seed` tests, `internal/app` tests and `ids` tests on fakes

## Phase 5: User Story 3 — delete the JSON store (P1)

- [x] T020 [US3] Delete `internal/platform/store`, every `store_json*.go` and its tests, `stores.JSON`, the
  JSON ID counters; list removed tests in research R6 (FR-006, FR-007)

## Phase 6: Polish

- [x] T021 Gates: format, vet, lint on new lines, sqlc, `go test -race`, `go test -short`, timing tests,
  govulncheck (FR-009)
- [x] T022 `pnpm run api:check` and the end-to-end suite on PostgreSQL (FR-008, FR-009)
- [x] T023 Measure after; research R5; pull request into `develop`
