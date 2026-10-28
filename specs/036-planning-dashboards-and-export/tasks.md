# Tasks: Planning Dashboards and Export to PowerPoint and PDF

**Input**: Design documents from `specs/036-planning-dashboards-and-export/` — [plan.md](plan.md), [spec.md](spec.md),
[research.md](research.md), [data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: required by constitution VI — unit tests for every calculation, a test for every acceptance scenario,
a contract test per endpoint, scope tests per persona, a test per view state, an end-to-end test for each P1
story (US1, US2). Write each new test first and see it fail.

## Phase 1: Setup (governance)

- [ ] T001 Record D-342–D-345 and Q-022, Q-023 in `docs/sdd/decision-log.md`; ADR-0017 in
  `docs/architecture/adr/0017-dashboards-charts-and-office-exports.md` with its row in `adr/README.md`
- [ ] T002 [P] Glossary terms (Dashboard, Committed, Actual, Unspent budget, Budget spent) in
  `docs/product/glossary.md`; chart tokens §4.7 and the Dashboards page in `docs/product/design-system.md`

## Phase 2: Foundational (blocking)

- [ ] T003 Contract first: add the five paths and the `Dashboard*` schemas to `backend/api/openapi.yaml` (tag
  Dashboards); run `pnpm run api:types`
- [ ] T004 Access rules in `backend/internal/access/routes_dashboards.go` (`need(plans.view)` ×2,
  `need(data.export)` ×2, `inPlan(data.export)` for the plan export) and join them in `routes.go`; relabel
  `data.export` "Export from All work, plans and dashboards" in `backend/internal/platform/authz/catalogue.go`,
  `frontend/src/session/access.ts`, `frontend/src/features/people/access/access-words.ts` (FR-027)
- [ ] T005 [P] Tests first for `backend/internal/platform/deck`: layout (cards, table continuation with header
  repeated, chart beside table, fitting with "…", footer page numbers), .pptx (valid zip, the part set, one slide
  per sheet, well-formed XML, escaped text, a real table), .pdf (header, page count, xref offsets, WinAnsi
  mapping, escaped strings) — `deck_test.go`, `layout_test.go`, `pptx_test.go`, `pdf_test.go`, `text_test.go`
- [ ] T006 Implement `backend/internal/platform/deck` (`deck.go`, `layout.go`, `text.go`, `pptx.go`,
  `pptx_parts.go`, `pptx_slide.go`, `pdf.go`) until T005 passes; validate a sample file with python-pptx and
  Quick Look by hand

**Checkpoint**: contract, rules and the file writers exist.

## Phase 3: User Story 1 — Executive dashboard (P1) 🎯 MVP

**Goal**: one answer with every panel, limited to the person's scope, from the owning features' figures.

### Tests (first)

- [ ] T007 [P] [US1] `allwork/planned_by_month_test.go`: `TestPlannedByActivity_010_FR023` (spread months, positive
  unassigned spread evenly, negative not, No dates, Other years)
- [ ] T008 [P] [US1] `financials/paid_by_month_test.go`: `TestPaidByMonth_036_FR014` (payments only, linked only,
  posting month, current rate, currency without a rate left out)
- [ ] T009 [P] [US1] `approval/round_summaries_test.go`: `TestRoundSummaries_036_FR016` (open round waiting
  since its level start, ended rounds) and `TestAverageDecisionDays` (half-up to one decimal, none decided)
- [ ] T010 [P] [US1] `purchaserequest/summaries_test.go`: `TestSummaries_036_FR017` (USD amount, waiting for
  GSAP, switched off, scope through the fake)
- [ ] T011 [P] [US1] `dashboard/figures_test.go` `TestFigures_FR008_FR011`; `tables_test.go`
  `TestMarkets_PlanInTwoTeams_FR012`, `TestPlans_FR013`, `TestCampaignLists_FR015`; `months_test.go`
  `TestMonths_FR014`; `panels_test.go` `TestApprovals_FR016`, `TestPurchaseRequests_FR017`,
  `TestCompleteness_FR018`, `TestGlobalBudgets_FR004`; `query_test.go` `TestParseQuery_FR006`
- [ ] T012 [P] [US1] `dashboard/handler_test.go` contract tests on `apitest`: `TestExecutive_Contract`
  (Carlos 2026: totals equal the plan pages and Financials, SC-001), `TestExecutive_Scope_FR002` (Priya: no
  Mobility China anywhere), `TestExecutive_FinanceHidden_FR003` (Lea: committed/actual null),
  `TestExecutive_GlobalBudgets_FR004`, `TestExecutive_Validation_FR006`, `TestExecutive_NoPlans`

### Implementation

- [ ] T013 [P] [US1] `allwork/planned_by_month.go` `PlannedByActivity`
- [ ] T014 [P] [US1] `financials/paid_by_month.go` `PaidByMonth`
- [ ] T015 [P] [US1] `approval/round_summaries.go` `RoundSummaries`, `AverageDecisionDays` (monitoring uses it)
- [ ] T016 [P] [US1] `purchaserequest/summaries.go` `Summaries`
- [ ] T017 [US1] `internal/dashboard`: `model.go`, `query.go`, `store.go`, `ports.go`, `input.go`, `figures.go`,
  `tables.go`, `months.go`, `panels.go`, `service.go` (`Executive`), `handler.go`
- [ ] T018 [US1] Wiring `backend/internal/app/dashboards.go` (adapters over allwork source, plans, financials,
  approvals, purchase requests, budgets, people, settings) and `areas.go`
- [ ] T019 [P] [US1] Web-app tests first: `DashboardsPage.test.tsx` (loading, error with retry, forbidden, empty,
  data; filters change the address and the query), `MonthlyChart.test.tsx` (chart summary, Table view),
  panel tests, `schemas.test.ts` with fast-check (any address parses to valid filters), axe in each
- [ ] T020 [US1] `frontend/src/features/dashboards` (api, schemas, links, text, page, filters, cards, tables,
  chart, panels) and chart tokens in `frontend/src/ui/tokens.css`
- [ ] T021 [US1] Sidebar item, routes and pages (`app/navigation.ts`, `app/routes.tsx`, `app/pages.tsx`)
- [ ] T022 [US1] `frontend/e2e/dashboards.spec.ts` "036 US1": Carlos's Executive dashboard shows both 2026
  plans and the panels; Priya's shows no Mobility China; Table view; axe

## Phase 4: User Story 2 — Export a plan (P1)

- [ ] T023 [P] [US2] Tests first: `dashboard/export_plan_test.go` (`TestPlanDeck_FR023`: pages, figures,
  activities continued, finance hidden note, file name), contract `TestPlanExport_Contract` (pptx and pdf,
  Content-Disposition, 403 without `data.export` in the plan, 404 out of scope, 400 bad format)
- [ ] T024 [US2] `dashboard/export.go`, `export_plan.go`; handler route `GET /plans/:planId/export`
- [ ] T025 [P] [US2] `frontend/src/features/plans/usePlanExport.tsx` with tests (items for holders of
  `data.export`, busy, error toast); `PlanPage.tsx` More menu
- [ ] T026 [US2] e2e "036 US2": Priya exports FY2026 Lubricants Europe to PowerPoint and PDF; check the
  signatures (`PK`, `%PDF-`), slide and page counts; axe on the open menu

## Phase 5: User Story 3 — Regional dashboard (P2)

- [ ] T027 [P] [US3] Tests first: `TestRegional_DefaultTeam_FR006`, `TestRegional_PlanCampaigns_FR021`,
  `TestRegional_TeamOutsideScope_FR002` (contract); web-app: market filter, plan row chooses the plan, market row
  link (FR-022)
- [ ] T028 [US3] `service.go` `Regional`, handler route; web-app Regional tab, `CampaignTable`

## Phase 6: User Story 4 — Export a dashboard (P2)

- [ ] T029 [P] [US4] Tests first: `dashboard/export_view_test.go` (`TestDashboardDeck_FR024`), contract
  `TestDashboardExport_Contract` (both kinds, both formats, 403 without `data.export`)
- [ ] T030 [US4] `export_view.go`, handler routes; web-app `ExportMenu.tsx` with tests

## Phase 7: Polish

- [ ] T031 Comment budget and 300-line limit on every touched file; `golangci-lint fmt`/`run`, `go vet`, sqlc
  generate+vet (no diff), `go test -race` with `TEST_DATABASE_URL`, govulncheck; `pnpm run check`, `pnpm run
  build`, `pnpm audit --prod --audit-level high`; the full e2e suite
- [ ] T032 Feature map row 036 "Built — …"; tick this file; quickstart walk-through

## Dependencies

T003–T006 before every story. US1 (T007–T022) before US3 and US4 (they reuse its View). US2 needs T006 and
T017. US4 needs T024's export plumbing.

## Requirement coverage

| FR | Tasks |
|---|---|
| FR-001 | T021, T022 |
| FR-002 | T012, T017, T027 |
| FR-003 | T011, T012, T017 |
| FR-004 | T011, T012 |
| FR-005 | T017 (read-only), T012 |
| FR-006, FR-007 | T011, T019, T020, T027 |
| FR-008–FR-011 | T011, T012, T017 |
| FR-012, FR-013 | T011, T017 |
| FR-014 | T007, T008, T011, T013, T014 |
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
