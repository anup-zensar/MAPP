# Tasks: Overspend Control — Warn or Refuse per Budget

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/](contracts/)

Tests come first in each phase (constitution VI); test names cite the requirement they prove. `[P]` = can run in
parallel with the other `[P]` tasks of its phase.

## Phase 1: Setup

- [x] T001 Migration `backend/internal/platform/database/migrations/00025_overspend.sql`: `overspend` columns on
  plans, campaigns, initiatives, global_budgets; table `overspend_configuration` (data-model §1)
- [x] T002 sqlc: the new column in every put/copy query of plans, campaigns, initiatives, global budgets; queries
  `ListOverspendBudgets`, `ListOverspendGlobalBudgets`, `GetOverspendConfiguration`, `PutOverspendConfiguration`
  in `queries/overspend.sql`; `sqlc generate` and `vet`
- [x] T003 Contract first in `backend/api/openapi.yaml`: schemas and properties of contracts/openapi.yaml, error
  code `over_budget` with `overBudget`; `pnpm run api:types`

## Phase 2: Foundational — package `internal/overspend`

- [x] T004 [P] Failing unit tests `rules_test.go`: refusal rule (raised, lowered amount, new budget, already over
  and kept/lowered, exactly at the budget, budget 0.00, Warn never refuses, several budgets listed in order),
  overage and message wording, limits (FR-008 – FR-011, FR-014)
- [x] T005 [P] Failing tests `guard_test.go`: refused save rolls back everything; nested writes checked once;
  global budget planned cost in USD at the current rates (FR-008 – FR-010)
- [x] T006 [P] Failing tests `access_test.go`: owner, budget manager in scope, other planner (reason), internal
  call (FR-003)
- [x] T007 Implement `model.go`, `rules.go`, `guard.go`, `access.go`, `store.go`, `store_postgres.go`,
  `overspendtest/store.go` (fake over the other areas' fake tables); the refusal's code is `overspend.CodeOverBudget`
  with `overspend.Refusal` in `rules.go` (no change to `errs`)
- [x] T008 Failing tests then implementation of the configuration: `handler_test.go` (contract, 403 without
  settings.manage, version conflict, validation, log entry; the service is tested through it) — `service.go`, `kinds.go`,
  `handler.go`; routes in `backend/internal/access/routes_overspend.go` (FR-006)

## Phase 3: User Story 1 — a budget owner refuses overspending (P1) 🎯

- [x] T009 [P] [US1] Failing API tests in `backend/internal/overspend`: create takes the default or the sent mode
  (`handler_test.go`); update logs "Overspend" and a planner who is not the owner gets 403 (`enforcement_test.go`);
  allocated plan 409 (`budgets_test.go`); view shows control (FR-001 – FR-005, FR-007)
- [x] T010 [P] [US1] The same rules for campaigns and initiatives, in the same files
- [x] T011 [US1] Implement plan and campaign/initiative fields, inputs, views, checks, log labels, store columns,
  fakes; plan defaults carry the organisation default
- [x] T012 [US1] Wire the guard in `backend/internal/app` (plans, campaigns, activities, budgets, history) and the
  default and limits ports; seed writes `warn`
- [x] T013 [US1] Failing API tests `backend/internal/overspend/enforcement_test.go` (contract-validated): activity create and update over a
  Refuse plan, campaign and initiative budget refused with the message; under it saved; Warn saved (US1
  scenarios 2, 4, 5); lowering a Refuse budget below planned cost refused (scenario 7)
- [x] T014 [US1] Activity form `overspendLimits` (activity port, service, tests) (FR-014)
- [x] T015 [US1] Web: `features/overspend` (field, note, texts, API), plan form, campaign and initiative forms,
  "Refuses overspending" on the records; activity form check; `over_budget` shown in the error summaries;
  component tests of each state (FR-002, FR-014 – FR-016)

## Phase 4: User Story 2 — every path (P2)

- [x] T016 [US2] Failing API tests `backend/internal/overspend/paths_test.go` and `budgets_test.go`: activity move, copy, split; initiative move and
  copy; campaign move and copy; plan copy with contents; restore and Undo from the recycle bin; a delete over a
  Refuse budget passes; line-item create and update never refused (US2 scenarios 1–5, FR-009, FR-012)
- [x] T017 [US2] Fix any path the tests show unguarded

## Phase 5: User Story 3 — global budgets and allocations (P2)

- [x] T018 [US3] Failing API tests `backend/internal/overspend/budgets_test.go` and `budget` service tests
  (`allocation_change_test.go`): global budget mode on create/update and
  view; allocation input sets the plan's mode through `SetBudgetFromAllocation`; allocate refused when the plan
  does not fit a Refuse global budget; lowering an allocation below planned cost refused (FR-004, FR-009)
- [x] T019 [US3] Implement budget model, inputs, rows, options, store column, fake; plan's allocation input
- [x] T020 [US3] Web: global budget form and allocation dialogs with the field and before-save checks

## Phase 6: User Story 4 — organisation default (P2)

- [x] T021 [US4] Web: Settings › Overspend control page and card; tests of loading, error, read-only and save
  (FR-006, FR-007)

## Phase 7: User Story 5 — purchase-request gate (P1)

- [x] T022 [US5] Failing test `purchaserequest` gate: line items above planned cost refuse Submit, Re-submit and
  Submit changes with the message, with the other problems; saving stays allowed (FR-017 – FR-019)
- [x] T023 [US5] Implement `LineItemFacts.ActivityOverBy`, the gate and message; wiring from line-item summaries

## Phase 8: End to end and polish

- [x] T024 E2E `frontend/e2e/overspend-control.spec.ts`: US1 (owner sets Refuse, over-budget save refused on the
  form and by the API, axe) and US5 (purchase request refused, axe)
- [x] T025 Docs: domain model §4.1, demo-users-and-data (Settings), feature map row 037 "Built"
- [x] T026 Gates: fmt, lint (new from 587eade2), sqlc diff, vet, tests with PostgreSQL, govulncheck, `pnpm run
  check`, `pnpm run build`, full e2e; comment budget and file length of every touched file

## Phase 9: Analyze findings (2026-10-29)

- [x] T027 [US2] API test `backend/internal/overspend/paths_test.go`
  `TestRefuse_CampaignMoveIntoARefusePlan_Contract_037_US2`: a campaign moved into a plan whose allocation refuses
  overspending is refused naming the plan and changes nothing; with room it moves (US2 scenario 1, FR-009 –
  FR-011). Note: passed on first run — the guard already covered the campaign move; no code change
- [x] T028 API test `backend/internal/overspend/budgets_test.go`
  `TestOverAllocationUnderARefusePlan_KeepsWarning_Contract_037_FR013`: a campaign budget above its Refuse plan's
  budget still saves (FR-013, D-346). Note: passed on first run; no code change
- [x] T029 Benchmark `backend/internal/platform/apibench/overspend_test.go` `BenchmarkOverspend`: an allowed and a
  refused activity create under a campaign set to Refuse, on the demo and volume databases (plan, Performance
  Goals).
  Note: results recorded in plan.md
- [x] T030 Record the unscoped guard read as a deviation: plan.md Complexity Tracking, D-355, ADR-0019 (proposed,
  for the product owner's approval) (constitution VII, B5)
- [x] T031 Test US1 scenario 4 on an initiative budget: Print & outdoor set to Refuse with 100.00 of room
  takes an activity of 100.00 and refuses one more cent, naming the initiative
  (`internal/overspend/initiative_test.go`; fails when the setting is Warn)

## Dependencies

Phase 1 → Phase 2 → US1 (T009–T015) → US2, US3, US4, US5 (independent of each other) → Phase 8 → Phase 9.
