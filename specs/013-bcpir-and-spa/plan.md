# Implementation Plan: SP&A Plan View and BCPIR Marker

**Branch**: `013-bcpir-and-spa` (worked on an isolated worktree branch) | **Date**: 2026-10-28 |
**Spec**: [spec.md](spec.md)

**Input**: Feature specification `specs/013-bcpir-and-spa/spec.md`, re-baselined on Shell's spec 006 (D-333,
D-334–D-337).

## Summary

A new read-only business package `internal/spa` serves `GET /api/v1/plans/{planId}/spa`: the SP&A view of a
plan or of one of its campaigns. It reads the live plan tree with the values the owning features calculate
(USD planned costs, line-item spreads, budgets, display statuses, attribute values) through a consumer port
that `internal/app` implements over All work's plan reader (010/016), GSAP transaction figures by line item
and posting month from `financials` (015, a new `FactsByLineItemMonth` beside `FactsByLineItem`), and the
purchase requests the person sees on the scope's line items from `purchaserequest` (032, a new
`DrawnOnLineItems`). Everything the view shows is worked out in `spa` from those inputs — headline figures,
periods, dimension rows, activity table, budget and procurement panels, traceability lists — in USD and
converted once to the display currency. The BCPIR marker and filter are small additions to the Plans list
(004), All work (010: marker, filter, Breakdown grouping) and the plan page. The web app gets a
`features/spa` folder with the SP&A page at `/plans/:planId/spa`, linked from the plan page's header, with a
Scope picker for each campaign.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9, React 19.
**Primary Dependencies**: Gin (handler layer), existing packages; TanStack Query, React Router, Radix, lucide-react,
CSS Modules. No new dependency.
**Storage**: none new — no migration, no table; the view is computed on read (spec Key Entities).
**Testing**: Go `testing` + `go-cmp` with hand-written fakes of the `spa` ports; contract tests through
`internal/platform/apitest` on the demo data; Vitest + Testing Library + `user-event` + MSW (`reply()`) +
`vitest-axe`; Playwright + axe (`E2E_API_PORT=18201 E2E_WEB_PORT=14201`, database `mapp_e2e_a`).
**Target Platform**: local laptop, servers on 127.0.0.1.
**Project Type**: web application (Go API + React web app).
**Performance Goals**: SC-006 — one read transaction; the plan tree, transactions and purchase requests are each
read once per request (028: no statement per activity); measured by `BenchmarkSpaView`.
**Constraints**: constitution 4.6.0 — files ≤ 300 non-blank lines, functions within the metrics, comments ≤ 5 %,
money in integer cents, scope only from the actor, 404 outside it.
**Scale/Scope**: 1 new operation, 2 changed operations (Plans list, All work query/breakdown/export), 1 new
package, 1 new web page.

## Constitution Check

| Principle | How 013 meets it |
|---|---|
| I Spec first | Every operation and screen traces to spec FR-001–FR-022; test names carry `013 FR-0xx`. |
| II Shell business | Shell's words: SP&A, BCPIR, Committed, Actual spend, Pending approval; the vocabulary Plan → Campaign → Initiative → Activity → Line item. |
| III Documented design | research.md (R1–R7), data-model.md (computed view and rules), contracts/; decisions D-334–D-337; no ADR (no cross-cutting change). |
| IV Server is the source of truth | Every figure, percentage, share, bar length and traceability list comes from `spa`; the web app only formats and sorts rows. Other areas are reached through consumer ports implemented in `internal/app`. |
| V Simplicity | One package; calculation split by concern (headline, periods, dimensions, panels); reuses 015's Facts and 032's PR reads instead of recomputing them. |
| VI Meaningful tests | Unit tests per rule (table below); a contract test for the new operation and the changed ones; handler test for 404/400; `spa` adds no store read: the scoped reads it calls are existing ones with their PostgreSQL isolation tests (`plan` `GetInScope`, `purchaserequest` `OnLineItems`), and a contract test proves 404 for a plan outside the actor's scope; web tests of each state; e2e with axe for US1 and US2. |
| VII Security | Route rule `view()` with the service's scoped plan read (404); `finance.view` in the plan gates financial figures; PRs filtered by 032's view scope; query parameters validated against allowlists. |
| VIII Clean construction | Comment budget per file; one concept per file. |
| IX Experience | Read-only page on the record template: breadcrumb, title, status and BCPIR chips, one primary action at most (none here: the page is read-only), scope picker, figures with ⓘ tips and reasons for every "—", overspend in red with the word. |
| X Palette | Existing tokens only: red `--danger` for overspend, neutral bars from the progress tokens. |
| B1–B9 | Gin only in `handler.go`; the service declares its ports; no SQL; actor passed to every call; no write, so no activity-log entry. |
| F1–F10 | Feature folder `features/spa` with `index.ts`, `api.ts` hooks, `schemas.ts` (URL state in Zod); MSW typed from the contract; no `any`. |

Governance → Alignment: `plan`, `allwork`, `financials` and `purchaserequest` are touched only by small
additions in their existing style.

### Threat model (STRIDE) for the new operation

| Threat | Response |
|---|---|
| Spoofing | session actor only (001). |
| Tampering | read-only operation; `campaignId` and `dimension` checked against the plan's campaigns and the offered dimensions (400 otherwise). |
| Repudiation | nothing changes, nothing to log. |
| Information disclosure: another market's plan | the plan is read with `GetInScope` (scope from the actor) → 404; a campaign of another plan → 404. |
| Information disclosure: financial data | committed, actual, invoiced need `finance.view` in the plan (015 FR-028) — null with a reason otherwise. |
| Information disclosure: purchase requests | only those 032's view scope returns (`OnLineItems` with `viewScope`). |
| Denial of service | one plan per request; reads bounded by the plan's size; no query by free text. |
| Elevation of privilege | Raise purchase request only offers 032's own create page, which checks `purchaseRequests.edit` again. |

## Business rules and the tests that prove them

Go test names carry the FR after the subject (`TestHeadline_FR015_…`); outside `spa` they carry the feature too
(`TestList_BCPIRFilterAndMarker_013_FR003`).

| Rule | Test |
|---|---|
| Headline: Remaining, Utilisation, Variance, overspend, "—" without planned cost (FR-015, FR-016) | `backend/internal/spa/headline_test.go` `TestHeadline_FR015_TheCampaignsPlannedCommittedActualRemaining`, `TestHeadline_FR015_WithoutPlannedCostUtilisationAndVariance`, `TestHeadline_FR016_ActualAbovePlannedIsOverspend` |
| Percent with one decimal, half up, signed (FR-015) | `TestPercentOf` (`percentOf` in `figures.go`) |
| Periods: spreads, positive balance over activity months, Not yet scheduled, months add up to quarters (FR-017) | `periods_test.go` `TestPeriods_FR017_PlannedByLineitemSpreadsAndActual`, `TestPeriods_FR017_TheMonthsAddUpToThe`, `TestPeriods_FR017_APositiveBalanceIsSpreadEvenly`, `TestPeriods_FR017_AnActivityWithoutDatesIsNot`, `TestPeriods_FR017_LineItemsAboveThePlannedCost`, `TestMonthIndex` |
| Only the plan's year counts, with a note (FR-010) | `TestYear_FR010_PlannedCostSpreadToAnotherYear`, `TestYear_FR010_TransactionsPostedInAnotherYearAre` |
| Records in the recycle bin left out (FR-011) | `service_test.go` `TestView_FR011_TransactionsOfRecordsInTheRecycleBinAreLeftOut` |
| Dimension: split by share, several values count in each, Not filled in, Not yet itemised (FR-018) | `dimensions_test.go` `TestDimension_FR018_*` (five), `TestShareOfAmount` |
| Traceability: each list sums to its figure (FR-007, SC-002) | `service_test.go` `TestTraceSums_FR007_SC002_EveryListAddsUpToItsFigure` |
| Budget panel: Planned vs budget, No budget set, over budget (FR-020) | `panels_test.go` `TestBudgetPanel_FR020_*` (three) |
| Procurement: Approved, Pending approval, Draft, Declined listed but not counted, only lines in scope (FR-021) | `TestProcurement_FR021_ApprovedPendingApprovalAndDraftAmounts` |
| Raise purchase request availability (FR-022) | `TestRaise_FR022_AnActivityWithLineItemsOffers`, `TestRaise_FR022_AnActivityWithoutLineItemsSays` |
| Scope, display currency, data as of, activity rows (FR-005, FR-008, FR-009, FR-012, FR-019) | `service_test.go` `TestView_FR005_*`, `TestView_FR008_AmountsInTheDisplayCurrencyEach`, `TestView_FR009_WhenTheFinanceDataWasReceived`, `TestView_FR012_APlanThePersonDoesNot`, `TestView_FR012_ACampaignThatIsNotIn`, `TestView_FR019_TheActivityRows` |
| Finance access: figures null with reason (FR-012) | `TestHeadline_FR012_WithoutFinanceviewTheFinancialFiguresAre`; contract `TestGetSpaViewAccess_FR012_WithoutFinanceviewTheFinancialFiguresAre` |
| Contract and scope on the demo data (FR-005, FR-012, FR-018, FR-021, SC-003) | `handler_test.go` `TestGetSpaView_SC003_ThePlansPlannedCostAndCommitted`, `TestGetSpaView_FR005_ACampaignsView`, `TestGetSpaView_FR012_ACampaignOfAnotherPlanIs`, `TestGetSpaView_FR018_AWrongCampaignOrDimensionIs`, `TestGetSpaView_FR021_ThePurchaseRequestsOnThePlans`, `TestGetSpaViewAccess_FR012_APlanOutsideThePersonsTeams` |
| Plans list BCPIR filter and marker (FR-002, FR-003) | `backend/internal/plan/list_bcpir_test.go` `TestList_BCPIRFilterAndMarker_013_FR003` |
| All work BCPIR filter, marker, export note, grouping, saved view (FR-002, FR-003) | `backend/internal/allwork/bcpir_test.go` `TestBCPIR_013_FR003_FilterOnlyAndExclude`, `TestBCPIR_013_FR002_PlanRowsCarryTheMarker`, `TestBCPIR_013_FR003_ExportNamesTheFilter`, `TestBCPIR_013_FR003_BreakdownGroupsByTheFlag`, `TestBCPIR_013_FR003_SavedViewsKeepTheFilter` |
| Flag change at once and logged (FR-004, inherited from 004) | 004's `plan` tests: `TestUpdatePlan_200_409_404_FR018_FR023` (`handler_edit_test.go`), `TestDescribe_ReadableLog_FR031` (`kinds_test.go`, "BCPIR plan") |
| Transactions by posting month (financials) | `backend/internal/financials/facts_month_test.go` `TestFactsByLineItemMonth_FR010_EachTransactionCountsInTheMonth`, `TestFactsByLineItemMonth_FR010_TheMonthsAddUpToThe` |
| PR lines drawn on line items, view scope (purchaserequest) | contract `TestGetSpaView_FR021_ThePurchaseRequestsOnThePlans` (the demo data's PR-1001) |
| SC-006: 2,000 activities, p95 within 2 s, no statement per activity | `backend/internal/platform/apibench/spa_test.go` `BenchmarkSpaView` (p95-ms and queries/op for the demo plan and that plan grown to 2,000 activities; demo data: p95 316 ms, 42 statements at both sizes) |

## Project Structure

### Documentation (this feature)

```text
specs/013-bcpir-and-spa/
├── spec.md · plan.md · research.md · data-model.md · quickstart.md · tasks.md
├── checklists/requirements.md
└── contracts/openapi.yaml · contracts/routes.md
```

### Source Code

```text
backend/
├── api/openapi.yaml                         # + getSpaView, SpaView schemas; bcpir on PlanSummary, Plans list and All work
├── internal/spa/                            # NEW: model.go ports.go service.go scope.go handler.go headline.go periods.go
│                                            #      dimensions.go activities.go panels.go figures.go + tests (fakes in fixture_test.go)
├── internal/access/routes_spa.go            # NEW: the route rule; routes.go joins it
├── internal/app/spa.go, spa_planning.go     # NEW: the adapters (planning over All work's reader, finance, PRs) and wiring
├── internal/platform/apibench/spa_test.go   # NEW: BenchmarkSpaView (SC-006)
├── internal/financials/service.go           # + FactsByLineItemMonth
├── internal/purchaserequest/drawn.go        # NEW: DrawnOnLineItems
├── internal/plan/{model_list,list,list_query}.go  # + bcpir marker and filter
├── internal/allwork/…                       # + Node.BCPIR, Row.bcpir, bcpir filter, Breakdown grouping
└── seed/plans.go                            # FY2026 Lubricants Europe is a BCPIR plan
frontend/src/
├── features/spa/                            # NEW: SpaPage and its cards, api.ts, schemas.ts, spa-text.ts, index.ts
├── features/plans/…                         # BCPIR marker and filter on the Plans list, marker and SP&A link on the plan page
├── features/all-work/…                      # BCPIR filter, chip and marker
├── app/routes.tsx, app/pages.tsx            # the /plans/:planId/spa route
└── e2e/spa.spec.ts                          # US1 and US2 end to end with axe
```

**Structure Decision**: a new business package for a new screen of computed figures, as `financials` and
`home` are; the plan reader is reused through an adapter instead of a third copy of the tree reading.

## Complexity Tracking

None: no metric outside the acceptable band is planned.
