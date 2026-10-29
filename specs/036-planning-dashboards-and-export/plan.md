# Implementation Plan: Planning Dashboards and Export to PowerPoint and PDF

**Branch**: `feature/036-planning-dashboards-and-export` (from `plan-module`) | **Date**: 2026-10-28 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/036-planning-dashboards-and-export/spec.md`

**Design artifacts**: [research.md](research.md) (R1–R10) · [data-model.md](data-model.md) ·
[contracts/api.md](contracts/api.md) · [contracts/ui.md](contracts/ui.md) · [quickstart.md](quickstart.md) ·
[ADR-0017](../../docs/architecture/adr/0017-dashboards-charts-and-office-exports.md)

## Summary

1. **A read-only `dashboard` package** (Go) works out one `View` per request — Executive (every market the
   person may see) or Regional (one team) — from what the owning features already calculate: All work's year
   data (`allwork.Source.Year`: the plans in the person's scope with budgets, planned costs, spreads, GSAP
   figures, completeness), All work's Budget & cost spread rule (a new exported `allwork.PlannedByActivity`),
   015's payments by posting month (new `financials.Service.PaidByMonth`), 017's approval rounds (new
   `approval.Service.RoundSummaries` and the extracted `approval.AverageDecisionDays`), 032's purchase
   requests (new `purchaserequest.Service.Summaries`, over the existing scoped `OnLineItems`), and 014's global
   budgets (`budget.Service.List`). It has no store of its own, like Home (009): `internal/app` adapts the
   other areas to its ports (D-196).
2. **Export** (`GET /api/v1/plans/{planId}/export`, `GET /api/v1/dashboards/{executive|regional}/export`):
   the dashboard package turns a View (or a one-plan View plus its activities) into a format-free `deck.Deck`;
   a new platform package `internal/platform/deck` lays every page out once and writes it as a .pptx (Office
   Open XML, `archive/zip` + `encoding/xml`) or a PDF (hand-written PDF 1.4 with the standard Helvetica fonts),
   standard library only (ADR-0017, D-344).
3. **Access** (`access/routes_dashboards.go`): the two views need `plans.view`, the dashboard exports
   `data.export`, the plan export `data.export` in the plan (the middleware answers 404 for a plan the person
   cannot see). Every figure counts only the plans of `authz.ScopeOf(ctx)` (the year data comes from
   `PlanStore.ListInScope`); finance figures only the plans of `finance.view` (D-301); purchase requests use
   032's view scope; global budgets need `globalBudgets.view` (D-343).
4. **Web app** (`features/dashboards`): a Dashboards page with Executive and Regional tabs, filters in the
   address (Zod), summary figures, tables, an SVG monthly bar chart with a Table view, campaign lists and the
   approvals, purchase-request, plan-details and global-budget panels; an Export menu (PowerPoint, PDF) on the
   dashboards and on the plan page. Charts use four new chart tokens (design system §4.7, D-345).

## Technical Context

**Language/Version**: Go (pinned in `go.mod`, 1.27); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new — Go standard library (`archive/zip`, `encoding/xml`, `bytes`) for the
files; existing web-app stack (TanStack Query, Zod, Radix, lucide-react, CSS Modules) for the screens

**Storage**: PostgreSQL; **no schema change, no migration, no new store method**. The dashboards read through
existing scoped store reads, each already covered by a PostgreSQL scope-isolation test:
`PlanStore.ListInScope` (023), `PlanStore.GetInScope` (023), `purchaserequest.PostgresStore.OnLineItems`
(032 `TestPostgresStore_ScopeIsolation_032_FR050`); the other reads it adds (`ListTransactions`,
`ListRounds`, `ListActions`, `ListBudgets`) read unscoped data that the dashboard filters to the plans in
scope, as All work and the Approvals board already do (research R9)

**Testing**: Go unit tests of every calculation (figures, markets, months, campaign lists, approvals,
purchase requests, completeness, deck layout and writers); handler and contract tests on `apitest`
(in-memory demo data, `libopenapi-validator`) for the five endpoints, with scope tests per persona; Vitest +
Testing Library + MSW + vitest-axe for every view state; Playwright + axe for the two P1 stories
(`e2e/dashboards.spec.ts`), checking the downloaded .pptx and .pdf (signature and slide/page count)

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions

**Project Type**: Web application (Go API + React web app)

**Performance Goals**: SC-003 — the Executive dashboard of one year at the 028 reference volume within 3 s;
a plan export within 5 s. The dashboard reads the year once (All work's Year, 028-checked: statements do not
grow with records), transactions once, rounds and actions once, purchase requests in one scoped statement;
no read per row (B5). Files of the demo plans are 30–60 kB

**Constraints**: constitution VII scope isolation; IV server works out every figure (the web app formats
only); X palette (chart tokens outside the palette, never yellow or red); F6 every chart has a table; F8 the
dashboards load with their own route chunk; V file and function metrics; VIII comment budget

**Scale/Scope**: 5 endpoints; 1 new Go business package (~12 files), 1 new platform package (~8 files), small
additions to `allwork`, `financials`, `approval`, `purchaserequest`, `app`, `access`; 1 new web-app feature
(~25 files), an Export menu on the plan page, navigation and routes; 1 e2e spec

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Check | Result |
|---|---|---|
| I Spec first | Spec with clarifications; FR IDs in test names; every FR has a task | Pass |
| II Shell language | Plan, Campaign, Activity, Line item; Committed, Actual (paid), Budget used, Left to plan; two new terms in the glossary (Unspent budget, Budget spent) | Pass |
| III Documented design | plan, research (R1–R10, threat and risk model R10), data model, contracts, quickstart, ADR-0017 | Pass |
| IV Server is the source of truth | Every figure, percent and month is worked out by the API; the web app formats; files are made by the API from the same View | Pass |
| V Simplicity | No new dependency; no store; deck lays a page out once for both formats; functions within metrics | Pass |
| VI Meaningful tests | Unit tests per rule, contract test per endpoint, scope tests per persona, view-state tests, e2e for P1 stories; no DB test beyond existing scope-isolation tests (no new store read) | Pass |
| VII Security | Scope from the actor only; route rules fail closed; out-of-scope plan 404; finance and global budgets gated; files never stored; input allowlists (format, plan type, currency, year range, team) | Pass |
| VIII Clean construction | Comment budget per touched file; one concept per file; no TODOs | Pass |
| IX Experience | One page title, tabs, filters in the address, Back keeps them; Export is a secondary menu (no yellow); states for loading, empty, error, forbidden; "—" never 0 | Pass |
| X Palette | Text and borders from palette tokens; four chart tokens outside the palette (§4.7), ≥ 3:1 on white, never the only carrier of meaning | Pass |
| B1–B9 | Gin only in `handler.go`; service has no `net/http`; ctx first; errors wrapped; contract first; tests on fakes | Pass |
| F1–F10 | TanStack Query hooks in `api.ts`; Zod for the address; no chart library; tests by role; axe in component and e2e tests | Pass |

## Project Structure

### Documentation (this feature)

```text
specs/036-planning-dashboards-and-export/
├── spec.md
├── plan.md              # this file
├── research.md          # R1–R10
├── data-model.md        # View, figures, deck
├── quickstart.md
├── contracts/
│   ├── api.md           # the five operations (openapi.yaml holds the schemas)
│   └── ui.md            # routes, components, query keys
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
backend/
├── api/openapi.yaml                         # 5 paths, Dashboard* schemas (contract first)
├── internal/dashboard/                      # NEW business area (read-only, no store)
│   ├── model.go          # View and its parts (API types)
│   ├── query.go          # Query, parsing and validation of the filters
│   ├── store.go          # Transactor
│   ├── ports.go          # Planning, Plans, Finance, Approvals, PurchaseRequests, GlobalBudgets, Directory
│   ├── service.go        # Executive, Regional
│   ├── input.go          # reading everything a View needs in one transaction
│   ├── figures.go        # Figures sums and derived values
│   ├── tables.go         # markets, plans, campaigns
│   ├── months.go         # the monthly planned and actual
│   ├── panels.go         # approvals, purchase requests, completeness, global budgets
│   ├── export.go         # PlanExport, DashboardExport, file names
│   ├── export_plan.go    # a plan's deck
│   ├── export_view.go    # a dashboard's deck
│   ├── handler.go
│   └── *_test.go
├── internal/platform/deck/                  # NEW: pages → .pptx and .pdf
│   ├── deck.go           # Deck, Page, Card, Table, Chart
│   ├── layout.go         # one laid-out Sheet per page; table continuation
│   ├── text.go           # Helvetica/Arial metrics, fitting, Windows-1252
│   ├── pptx.go, pptx_parts.go, pptx_slide.go
│   ├── pdf.go
│   └── *_test.go
├── internal/allwork/planned_by_month.go     # exported Budget & cost spread per activity
├── internal/financials/paid_by_month.go     # payments by line item and posting month
├── internal/approval/round_summaries.go     # rounds as the dashboard counts them; AverageDecisionDays
├── internal/purchaserequest/summaries.go    # PRs on line items, amount in USD, status
├── internal/access/routes_dashboards.go     # the five rules
└── internal/app/dashboards.go               # wiring and adapters

frontend/
├── src/features/dashboards/                 # NEW feature
├── src/features/plans/usePlanExport.ts     # Export › PowerPoint / PDF on the plan page
├── src/app/{navigation.ts,pages.tsx,routes.tsx}
├── src/ui/tokens.css                        # chart tokens
└── e2e/dashboards.spec.ts
```

**Structure Decision**: web application; one new business package and one platform package in the API, one
feature folder in the web app, as constitution B2 and F2 lay them out.

## Business rules and the tests that prove them

| Rule | Where | Test |
|---|---|---|
| Budget, planned, committed, actual sums; budget used, budget spent, left to plan, unspent budget, over budget (FR-008–FR-011) | `dashboard/figures.go` | `TestFigures_FR008_FR011` (table-driven: no budget, no planned, finance hidden, over budget, rounding) |
| A plan in two markets counts in both rows and once in the total (FR-012) | `dashboard/tables.go` | `TestMarkets_PlanInTwoTeams_FR012` |
| By plan by name; campaigns over budget and left to plan, top five, no budget excluded (FR-013, FR-015) | `dashboard/tables.go` | `TestPlans_FR013`, `TestCampaignLists_FR015` |
| Monthly planned = Budget & cost spread; no dates and other years apart; actual by posting month; running totals; paid outside the year (FR-014) | `dashboard/months.go`, `allwork/planned_by_month.go`, `financials/paid_by_month.go` | `TestMonths_FR014`, `TestPlannedByActivity_010_FR023`, `TestPaidByMonth_036_FR014` |
| Approvals waiting, oldest, decided, auto, average days (FR-016) | `dashboard/panels.go`, `approval/round_summaries.go` | `TestApprovals_FR016`, `TestRoundSummaries_036_FR016`, monitoring tests unchanged |
| Purchase requests by status and amount, waiting for GSAP, switched off (FR-017) | `dashboard/panels.go`, `purchaserequest/summaries.go` | `TestPurchaseRequests_FR017`, `TestSummaries_036_FR017` |
| Completeness per level (FR-018) | `dashboard/panels.go` | `TestCompleteness_FR018` |
| Scope: only plans in scope; finance only where visible; global budgets only with the permission (FR-002–FR-004) | service + adapters | contract tests per persona `TestExecutive_Scope_FR002`, `TestExecutive_FinanceHidden_FR003`, `TestGlobalBudgets_Permission_FR004` |
| Filters: defaults, allowlists (FR-006) | `dashboard/query.go` | `TestParseQuery_FR006` |
| Regional default team, plan drill (FR-020–FR-022) | `dashboard/service.go` | `TestRegional_DefaultTeam_FR006`, `TestRegional_PlanCampaigns_FR021` |
| Files: page layout, continuation, fitting, escaping, valid .pptx and .pdf (FR-023–FR-026) | `platform/deck` | `TestLayout_*`, `TestWritePPTX_*`, `TestWritePDF_*` |
| Export access and file names (FR-023–FR-025) | handler + routes | `TestPlanExport_Contract`, `TestDashboardExport_Contract`, access route tests |

## Complexity Tracking

| Violation | Why needed | Simpler alternative rejected because |
|---|---|---|
| Font metric tables as package-level arrays (`//nolint:gochecknoglobals`) | Text must be fitted to cells before both writers draw it | Computing widths per call from a `switch` is longer and slower |
