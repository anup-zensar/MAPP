---
description: "Task list for feature 013 SP&A Plan View and BCPIR Marker"
---

# Tasks: SP&A Plan View and BCPIR Marker

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml), [contracts/routes.md](contracts/routes.md),
[quickstart.md](quickstart.md)

**Tests**: required (constitution VI). Write each test first and see it fail. Go test names carry the FR, e.g.
`TestHeadline_FR015_TheCampaignsPlannedCommittedActualRemaining` (outside `spa` with the feature:
`TestList_BCPIRFilterAndMarker_013_FR003`); web tests `describe('013 FR-003: …')`.

## Format: `[ID] [P?] [Story?] Description`

---

## Phase 1: Setup

- [x] T001 Merge the contract (contracts/openapi.yaml, rules 1–4) into `backend/api/openapi.yaml`; add the tag;
  add `getSpaView` to the served operations check in `backend/internal/app/app_test.go` if it lists them.
- [x] T002 [P] Regenerate the web app's API types (`pnpm run api:types`) and fix type errors.

## Phase 2: Foundation

- [x] T003 [P] `backend/internal/financials`: test first `facts_month_test.go`
  (`TestFactsByLineItemMonth_FR010_EachTransactionCountsInTheMonth`, `TestFactsByLineItemMonth_FR010_TheMonthsAddUpToThe`),
  then `FactsByLineItemMonth` sharing the loop of `FactsByLineItem`.
- [x] T004 [P] `backend/internal/purchaserequest`: `drawn.go` (`DrawnOnLineItems`, `RaiseAvailability`), proven
  through the contract test of the SP&A view on the demo data (PR-1001 on the UK plan, view scope of `OnLineItems`).
- [x] T005 Create `backend/internal/spa`: `model.go` (API types of data-model §3, package doc), `ports.go` (Planning,
  Finance, Purchases, Displays and the input types), `service.go` (`NewService`, `View`), `scope.go` (the tree and
  the activities' figures); the fakes of the ports are in `fixture_test.go` (the handler and contract tests run on
  the demo data through `apitest`, so no `spatest` package is needed).
- [x] T006 `backend/internal/access/routes_spa.go` with the rule of contracts/routes.md; join it in `routes.go`.

## Phase 3: User Story 1 — SP&A view of a plan or campaign (P1) 🎯 MVP

- [x] T007 [P] [US1] Test first `headline_test.go` (`TestHeadline_FR015_*`, `TestHeadline_FR016_ActualAbovePlannedIsOverspend`,
  `TestPercentOf`: scenarios 1–3, percent rounding, overspend), then `headline.go` and `figures.go` (`percentOf`).
- [x] T008 [P] [US1] Test first `periods_test.go` (`TestPeriods_FR017_*`, `TestYear_FR010_*`, `TestMonthIndex`:
  scenario 4, balance over months, outside the year, no dates, months add up to quarters), then `periods.go` and
  the month placing in `scope.go`.
- [x] T009 [P] [US1] Test first `dimensions_test.go` (`TestDimension_FR018_*`, `TestShareOfAmount`: scenarios 5–6,
  several values, not filled in, not yet itemised, order), then `dimensions.go`.
- [x] T010 [US1] Test first in `service_test.go`: `TestView_FR019_TheActivityRows` (rows, line items, start quarter,
  channel) and `TestTraceSums_FR007_SC002_EveryListAddsUpToItsFigure` (every trace sums to its figure), then
  `activities.go` and the traces in `figures.go`.
- [x] T011 [US1] Test first `service_test.go` (`TestView_FR005_*`, `TestView_FR008_*`, `TestView_FR009_*`,
  `TestView_FR012_*`: campaign scope, campaign of another plan 404, plan out of scope 404, display currency, data
  as of, empty campaign; no finance.view in `TestHeadline_FR012_*`; unknown dimension 400 in
  `TestDimension_FR018_ADimensionThatIsNotOffered`), then the service's assembly.
- [x] T012 [US1] `handler.go` + `handler_test.go` (external package through `apitest` on the demo data:
  `TestGetSpaView_*` and `TestGetSpaViewAccess_*` — query parsing, 400, 404, contract of `getSpaView`).
- [x] T013 [US1] `backend/internal/app/spa.go` and `spa_planning.go`: the adapters (planning over All work's `PlanYear`, attributes and
  activity types; finance over `financials` and 015's access; purchases over 032) and the wiring.
- [x] T014 [P] [US1] Web: `frontend/src/features/spa/` — `index.ts`, `api.ts` (`useSpaView`), `schemas.ts` (URL
  state, fast-check property), `spa-text.ts`, `SpaPage.tsx` with header, scope picker, headline cards, periods
  card, dimension card, activity table, trace dialog; route in `app/routes.tsx` and `app/pages.tsx`.
- [x] T015 [US1] Web tests: `SpaPage.test.tsx` (loading, error, not found, empty, success; overspend; scope switch;
  trace dialog; sort; line items; axe).
- [x] T016 [US1] Entry point: "SP&A view" on the plan page header; a campaign's view is chosen with the page's Scope picker (the campaign panel keeps its own figures, no second entry point needed).

## Phase 4: User Story 2 — BCPIR marker and filter (P1)

- [x] T017 [P] [US2] Backend Plans list: test first `plan/list_bcpir_test.go` `TestList_BCPIRFilterAndMarker_013_FR003`
  (Only, Exclude, wrong value 400, marker), then `PlanSummary.BCPIR` and the `bcpir` filter.
- [x] T018 [P] [US2] Backend All work: test first `allwork/bcpir_test.go` (`TestBCPIR_013_FR003_FilterOnlyAndExclude`,
  `TestBCPIR_013_FR002_PlanRowsCarryTheMarker`, `TestBCPIR_013_FR003_SavedViewsKeepTheFilter`,
  `TestBCPIR_013_FR003_ExportNamesTheFilter`, `TestBCPIR_013_FR003_BreakdownGroupsByTheFlag`), then `Node.BCPIR`, `Row.BCPIR`, query, tree filter, views, export
  notes, grouping; set `BCPIR` in `app/allwork_source.go`.
- [x] T019 [P] [US2] Web Plans list: marker beside the name, filter "BCPIR plans" with its chip; plan header marker;
  tests.
- [x] T020 [P] [US2] Web All work: filter, chip, URL state, saved view, row marker; tests.
- [x] T021 [US2] Seed: FY2026 Lubricants Europe is a BCPIR plan; adjust the tests that read its flag.
- [x] T021a [US2] FR-004 is inherited from 004: the flag is a plan field saved by 004's plan edit and logged with
  the label "BCPIR plan" (`plan` `TestUpdatePlan_200_409_404_FR018_FR023`, `TestDescribe_ReadableLog_FR031`); the
  lists and views read it live, so nothing is built here.

## Phase 5: User Story 3 — Budget and procurement (P2)

- [x] T022 [US3] Test first `panels_test.go` (`TestBudgetPanel_FR020_*`, `TestProcurement_FR021_*`, `TestRaise_FR022_*`:
  scenarios 1–5, lines in scope only), then `panels.go` and the raise availability (`raiseFor` in `activities.go`).
- [x] T023 [US3] Web: budget panel, procurement panel (open a PR), Raise purchase request menu; tests (scenarios 6–8).

## Phase 6: Polish

- [x] T024 End-to-end `frontend/e2e/spa.spec.ts`: US1 (plan view, quarter/month, dimension, trace, campaign scope,
  Back) and US2 (Plans list and All work filter, marker) with axe.
- [x] T025 Docs: glossary (BCPIR, SP&A, Actual spend), field catalogue §1 #11, design system (SP&A page), feature map
  row "Built — …".
- [x] T026 Gates: `golangci-lint fmt` (no diff), `golangci-lint run --new-from-rev=587eade2`, sqlc generate + vet (no
  diff), `go vet`, `go test -race` with `TEST_DATABASE_URL`, govulncheck, `pnpm run check`, `pnpm run build`,
  full e2e; comment budget and 300-line limit on every touched file.

## Phase 7: Analyze findings (2026-10-29)

- [x] T027 [US1] FR-011: `spa` `TestView_FR011_TransactionsOfRecordsInTheRecycleBinAreLeftOut` — an activity and
  a line item in the recycle bin (absent from the plan reader, their GSAP transactions still there) count in no
  figure, period, trace or row. Done: the test passed at once — the rule was already met (spend is read only for
  the live tree's line items), so no code changed.
- [x] T028 SC-006 benchmark `backend/internal/platform/apibench/spa_test.go` `BenchmarkSpaView`: the view of the
  demo plan, then of the same plan grown to 2,000 activities by copying its campaigns (whole plan and one campaign);
  reports p95-ms and queries/op; one sub-benchmark per data set (`demo`, `volume`). Done, demo data, 20 requests
  each (Apple M5, 2026-10-29): demo plan p95 22 ms, 2,002 activities p95 316 ms, its campaign p95 325 ms — 42
  statements in every case, so none per activity; SC-006 met. Volume half not run yet: `TEST_DATABASE_URL=… go test -run='^$' -bench='BenchmarkSpaView/volume' -benchtime=1x -timeout=6h ./internal/platform/apibench/` (from `backend/`; the first run builds the volume template, which took more than 1.5 hours on 2026-10-29 and was stopped).
- [x] T029 Docs: plan.md and these task texts name the real files and tests; contracts/routes.md places the
  `/all-work/breakdown` rule in `routes_settings.go`; spec status Built.

## Notes

- Every touched file was brought within the comment budget (constitution VIII); `allwork`, `plan` and web
  files lost their comments where needed.
- Left open: none of the scope of D-333; what 006 asks and is not built is listed in spec "Not built" (D-337).

## Dependencies

T001 → everything; T003–T006 → US1; US1 backend (T007–T013) → T014–T016 can start on MSW after T002; US2
independent of US1; US3 after T011. T024 after US1 and US2.
