# Tasks: Planning Dashboards and Export to PowerPoint and PDF

**Input**: Design documents from `specs/036-planning-dashboards-and-export/` — [plan.md](plan.md), [spec.md](spec.md),
[research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Status (2026-10-29)**: every task done but the volume run of T034. Deviations from the plan, all recorded: the data.export label stays
"Export from All work" (D-343; its web-app copies are over the comment budget, so widening it waits for the
next change of the access screens); `purchaserequest.Summaries` is proven through the dashboard contract tests
(2027's PR-1001, scope through `OnLineItems`) rather than a unit test of its own (T010); the dashboard routes
live in `app/dashboard-routes.tsx` because `routes.tsx` is at its 300-line limit; the Dashboards item sits
after Purchase requests so the existing sidebar order stays. T033–T036 (2026-10-29) answer the analysis of the
built feature: reads scoped in SQL (D-356), the SC-003 benchmark (its volume run still open, T034), Actual in
the plan's year (D-357), and the documents aligned with the code.

**Tests**: required by constitution VI — unit tests for every calculation, a test for every acceptance scenario,
a contract test per endpoint, scope tests per persona, a test per view state, an end-to-end test for each P1
story (US1, US2). Write each new test first and see it fail.

## Phase 1: Setup (governance)

- [X] T001 Record D-342–D-345 and Q-022, Q-023 in `docs/sdd/decision-log.md`; ADR-0017 in
  `docs/architecture/adr/0017-dashboards-charts-and-office-exports.md` with its row in `adr/README.md`
- [X] T002 [P] Glossary terms (Dashboard, Committed, Actual, Unspent budget, Budget spent) in
  `docs/product/glossary.md`; chart tokens §4.7 and the Dashboards page in `docs/product/design-system.md`

## Phase 2: Foundational (blocking)

- [X] T003 Contract first: add the five paths and the `Dashboard*` schemas to `backend/api/openapi.yaml` (tag
  Dashboards); run `pnpm run api:types`
- [X] T004 Access rules in `backend/internal/access/routes_dashboards.go` (`need(plans.view)` ×2,
  `need(data.export)` ×2, `inPlan(data.export)` for the plan export) and join them in `routes.go` (FR-027)
- [X] T005 [P] Tests first for `backend/internal/platform/deck`: layout (cards, table continuation with header
  repeated, chart beside table, fitting with "…", footer page numbers), .pptx (valid zip, the part set, one slide
  per sheet, well-formed XML, escaped text, a real table), .pdf (header, page count, xref offsets, WinAnsi
  mapping, escaped strings) — `layout_test.go` (layout, fitting, WinAnsi) and `writers_test.go` (.pptx, .pdf)
- [X] T006 Implement `backend/internal/platform/deck` (`deck.go`, `layout.go`, `layout_chart.go`, `text.go`,
  `pptx.go`, `pptx_parts.go`, `pptx_slide.go`, `pdf.go`) until T005 passes; validate a sample file with
  python-pptx and Quick Look by hand

**Checkpoint**: contract, rules and the file writers exist.

## Phase 3: User Story 1 — Executive dashboard (P1) 🎯 MVP

**Goal**: one answer with every panel, limited to the person's scope, from the owning features' figures.

### Tests (first)

- [X] T007 [P] [US1] `allwork/planned_by_month_test.go`: `TestPlannedByActivity_010_FR023` (spread months, positive
  unassigned spread evenly, negative not, No dates, Other years)
- [X] T008 [P] [US1] `financials/paid_by_month_test.go`: `TestPaidByMonth_036_FR014` (payments only, linked only,
  posting month, current rate, currency without a rate left out)
- [X] T009 [P] [US1] `approval/round_summaries_test.go`: `TestRoundSummaries_036_FR016` (open round waiting
  since its level start, ended rounds) and `TestAverageDecisionDays` (half-up to one decimal, none decided)
- [X] T010 [P] [US1] `purchaserequest.Summaries` (USD amount, waiting for GSAP, switched off, scope through the
  fake): proven through the dashboard contract tests (`TestExecutive_Filters_FR006`, 2027's PR-1001) rather
  than a `summaries_test.go` of its own (see Status)
- [X] T011 [P] [US1] `dashboard/board_test.go` `TestFigures_FR008_FR011`, `TestMarkets_PlanInTwoTeams_FR012`,
  `TestPlans_ByNameWithTheirFinance_FR013`, `TestCampaignLists_FR015`, `TestPlanCampaigns_FR021`,
  `TestCompleteness_FR018`; `panels_test.go` `TestMonths_FR014`, `TestApprovals_FR016`,
  `TestPurchaseRequests_FR017`, `TestGlobalBudgets_ShownInTheCurrency_FR004`, `TestParseQuery_FR006`
- [X] T012 [P] [US1] `dashboard/handler_test.go` contract tests on `apitest`:
  `TestExecutive_EqualsTheRecordPages_SC001` (Carlos 2026: totals equal the plan pages and Financials, global
  budgets shown), `TestExecutive_OnlyThePlansInScope_FR002` (Priya: no Mobility China anywhere, no global
  budgets), `TestExecutive_FinanceHidden_FR003` (Lea: committed/actual null), `TestExecutive_Filters_FR006`
  (currency, plan type, year, an empty year), `TestExecutive_Validation_FR006`

### Implementation

- [X] T013 [P] [US1] `allwork/planned_by_month.go` `PlannedByActivity`
- [X] T014 [P] [US1] `financials/paid_by_month.go` `PaidByMonth`
- [X] T015 [P] [US1] `approval/round_summaries.go` `RoundSummaries`, `AverageDecisionDays` (monitoring uses it)
- [X] T016 [P] [US1] `purchaserequest/summaries.go` `Summaries`
- [X] T017 [US1] `internal/dashboard`: `model.go`, `query.go`, `ports.go` (with the `Transactor`), `input.go`,
  `board.go`, `figures.go`, `tables.go`, `months.go`, `panels.go`, `service.go` (`Executive`), `handler.go`
- [X] T018 [US1] Wiring `backend/internal/app/dashboards.go` (adapters over allwork source, plans, financials,
  approvals, purchase requests, budgets, people, settings) and `areas.go`
- [X] T019 [P] [US1] Web-app tests first: `DashboardsPage.test.tsx` (loading, error with retry, forbidden, empty,
  data; filters change the address and the query; the chart's Table view), `DashboardPanels.test.tsx` (panels
  and the chart's notes), `schemas.test.ts` with fast-check (any address parses to valid filters), axe in each
- [X] T020 [US1] `frontend/src/features/dashboards` (api, schemas, links, text, page, filters, cards, tables,
  chart, panels) and chart tokens in `frontend/src/ui/tokens.css`
- [X] T021 [US1] Sidebar item, routes and pages (`app/navigation.ts`, `app/dashboard-routes.tsx`, `app/pages.tsx`)
- [X] T022 [US1] `frontend/e2e/dashboards.spec.ts` "036 US1": Carlos's Executive dashboard shows both 2026
  plans and the panels; Priya's shows no Mobility China; Table view; axe

## Phase 4: User Story 2 — Export a plan (P1)

- [X] T023 [P] [US2] Tests first: `dashboard/export_test.go` (`TestPlanDeck_FR023`: pages, figures,
  activities, finance hidden note), contract `TestPlanExport_FR023_FR025` (pptx and pdf, Content-Disposition,
  403 without `data.export` in the plan, 404 out of scope, 400 bad format)
- [X] T024 [US2] `dashboard/export.go`, `export_pages.go`, `export_plan.go`, `words.go`; handler route
  `GET /plans/:planId/export`
- [X] T025 [P] [US2] `frontend/src/features/plans/usePlanExport.ts` with tests (items for holders of
  `data.export`, busy, error toast); `PlanPage.tsx` More menu
- [X] T026 [US2] e2e "036 US2": Priya exports FY2026 Lubricants Europe to PowerPoint and PDF; check the
  signatures (`PK`, `%PDF-`), slide and page counts; axe on the open menu

## Phase 5: User Story 3 — Regional dashboard (P2)

- [X] T027 [P] [US3] Tests first: contract `TestRegional_FR020_FR022` (own market by default, a plan's
  campaigns, a market row's equal, a market outside the scope shows none of its plans, unknown market refused),
  unit `TestPlanCampaigns_FR021`; web-app: market filter, plan row chooses the plan, market row link (FR-022)
- [X] T028 [US3] `service.go` `Regional`, handler route; web-app Regional tab, `PlanCampaignsTable`

## Phase 6: User Story 4 — Export a dashboard (P2)

- [X] T029 [P] [US4] Tests first: `dashboard/export_test.go` (`TestDashboardDeck_FR024`), contract
  `TestDashboardExport_FR024` (both kinds, both formats, 403 without `data.export`)
- [X] T030 [US4] `export_view.go`, handler routes; web-app `ExportMenu.tsx` with tests

## Phase 7: Polish

- [X] T031 Comment budget and 300-line limit on every touched file; `golangci-lint fmt`/`run`, `go vet`, sqlc
  generate+vet (no diff), `go test -race` with `TEST_DATABASE_URL`, govulncheck; `pnpm run check`, `pnpm run
  build`, `pnpm audit --prod --audit-level high`; the full e2e suite
- [X] T032 Feature map row 036 "Built — …"; tick this file; quickstart walk-through

## Phase 8: Analysis follow-up (2026-10-29)

- [X] T033 Reads scoped in SQL (constitution B5, D-356): new store reads `financials.Store.PaymentsOnLineItems`,
  `approval.Store.RoundsOfSubjects` and `ActionsOfSubjects` (`= ANY(…)` over the records of the plans in scope)
  with PostgreSQL tests `TestPostgresStorePaymentsOnLineItemsScopeIsolation_036_FR002` and
  `TestPostgresStoreRoundsOfSubjectsScopeIsolation_036_FR002`, and `budget.Store.AllocationsOfBudgets` behind the
  new `budget.Service.YearFigures` (`TestYearFigures_AsTheGlobalBudgetsPage_036_FR004`: one statement, not one
  per budget; no plan scope by 014's rule, plan Complexity Tracking); each in both fakes. Done:
  `PaidByMonth(ctx, lineItemIDs)` and `RoundSummaries(ctx, subjectIDs)` no longer read every transaction, round
  or history entry
- [ ] T034 SC-003 benchmark `BenchmarkDashboards` in `backend/internal/platform/apibench/dashboards_test.go`
  (Executive and Regional dashboards, the Executive export, a plan's export) on the demo data and on 028's
  volume (the plan of 500 activities). Written; the demo half measured (15–21 ms, 45–51 statements, research
  R9). Open: run the volume half and record it in research R9 —
  `go test -run '^$' -bench 'BenchmarkDashboards/volume/' -benchtime 5x -timeout 3h ./internal/platform/apibench/`
  from `backend/` with `TEST_DATABASE_URL` set
- [X] T035 Actual in the plan's year (D-357, 013 FR-010): failing test first
  `TestActual_PaidInThePlansYearOnly_D357`, then `board.countPaid` for plans, campaigns and a plan export's
  activities; "Paid outside the year (not counted)" on screen and in files; spec FR-011, glossary, contract
  descriptions. Done
- [X] T036 Documents aligned with the code: file, test and port names in tasks.md, plan.md and contracts/api.md;
  spec status Built. Done

## Dependencies

T003–T006 before every story. US1 (T007–T022) before US3 and US4 (they reuse its View). US2 needs T006 and
T017. US4 needs T024's export plumbing.

## Requirement coverage

| FR | Tasks |
|---|---|
| FR-001 | T021, T022 |
| FR-002 | T012, T017, T027, T033 |
| FR-003 | T011, T012, T017 |
| FR-004 | T011, T012, T033 |
| FR-005 | T017 (read-only), T012 |
| FR-006, FR-007 | T011, T019, T020, T027 |
| FR-008–FR-011 | T011, T012, T017, T035 |
| FR-012, FR-013 | T011, T017 |
| FR-014 | T007, T008, T011, T013, T014, T035 |
| FR-015 | T011 |
| FR-016 | T009, T011, T015 |
| FR-017 | T010, T011, T016 |
| FR-018 | T011 |
| FR-019 | T012 |
| FR-020–FR-022 | T027, T028 |
| FR-023 | T023–T026 |
| FR-024 | T029, T030 |
| FR-025 | T023, T029 |
| FR-026 | T005, T023 |
| FR-027 | T004 |
| FR-028, FR-029 | T019, T020 |
| SC-003 | T034 |
