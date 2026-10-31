# Implementation Plan: Plan Details Report

**Branch**: `feature/043-plan-details-report` (from `measure-module`) | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/043-plan-details-report/spec.md` (clarifications 043-C1–043-C4)

**Design artifacts**: [research.md](research.md) (R1–R5) · [contracts/api.md](contracts/api.md) ·
[quickstart.md](quickstart.md). No `data-model.md`: no new table, no migration (build guide §8).

## Summary

1. **A read-only `measure/plandetails` package** works out one `PlanDetailsReport` per request for one level:
   summary cards, the built-in widgets and one row per active Shell attribute of the level. It has no store: it
   reads the year through a `Planning` port over All work's `Year`, `Attributes` and `Choices`, adapted in
   `app/measure_plandetails.go` as 036 does (research R1), so scope, planned cost and completeness come from
   their one definition and the statements do not grow with the records (028).
2. **016's attribution rule** (FR-011) is a 40-line copy in `attribution.go`, pinned to All work's Breakdown by
   an equality test; the integrator is asked to export it from `allwork` instead (research R2). **Does not
   apply** (043-C2) is worked out from the attribute's business-unit condition (R3).
3. **039's frame** gives the filter, period, currency, saved views, states and export: the handler parses
   039's types and adds `level`; the report implements `measure.Report` and its `Sheets` (R4).
4. **Web** (`features/measure/plan-details/`): the report page inside `ReportFrame`, level tabs in the address,
   cards, the built-in section and attribute rows of 036 SVG bars with Table views, links into All work.

Backend and web code wait for 039's hub on `measure-module` (build guide §3, about hour 16).

## Technical Context

**Language/Version**: Go 1.27 (`go.mod`); TypeScript 5.9 strict + React 19

**Primary Dependencies**: none new — 039's `measure` package, `allwork` types, `platform/money`, `platform/sheet`;
web: TanStack Query, Zod, MSW, vitest-axe, Playwright, 039's `features/measure/shared`, 036's SVG bars

**Storage**: PostgreSQL through All work's existing scoped year read (`PlanStore.ListInScope`, 023's
scope-isolation test). **No new store read, no sqlc query, no migration.** `queries/measure_plandetails.sql` is
reserved but not needed (R1).

**Testing**: Go unit tests of every rule on an in-memory year (fake `Planning`); handler and contract tests on
`apitest` for both operations with personas (scope, no `reports.view`, no `data.export`); Vitest + Testing
Library + MSW `reply()` + vitest-axe, one test per widget state; Playwright + axe for US1
(`frontend/e2e/measure-plan-details.spec.ts`)

**Target Platform**: developers' machines (127.0.0.1); GitHub Actions on `measure-module`

**Project Type**: web application (Go API + React web app)

**Performance Goals**: SC-001 — each tab within 3 s at 50 plans, 5,000 activities, 20,000 line items and 30
attributes; the calculation adds under 300 ms to the year read (`BenchmarkPlanDetails_SC001`, R5). One year read,
one attributes read, one choices read per request; one pass over the nodes per tab.

**Constraints**: constitution VII (scope from the actor only), IV (server works out every figure, "Other"
included), F6 (every chart has a table), X (four chart tokens, D-345), V (≤ 300 non-blank lines per file),
VIII (comments ≤ 5 %); build guide §4 — only issue #36's "Owns" paths

**Scale/Scope**: 2 operations (1 served by 039's export); 1 Go package (~10 files); 1 wiring file; 1 web
folder (~15 files); 1 e2e spec

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.* — **Pass, before and after design.**

| Principle | Check | Result |
|---|---|---|
| I Spec first | Spec with 043-C1–C4; FR IDs in test names; every FR has a test below | Pass |
| II Shell language | Plan, Campaign, Initiative, Activity; Not filled in; Does not apply (new label, 043-C2, glossary entry by the integrator) | Pass |
| III Documented design | plan, research R1–R5, contracts/api.md, quickstart | Pass |
| IV Server is the source of truth | Counts, amounts, shares, per cents, "Other" and the start periods come from the API; the web app formats | Pass |
| V Simplicity | No store, no query, no dependency; reuses All work's read and 039's frame | Pass (one copy of a rule: Complexity Tracking) |
| VI Meaningful tests | A test per rule and per widget state; equality with Breakdown; contract test per operation; scope persona tests; e2e for US1. No new scoped store read, so no new scope-isolation test (the year read has 023's) | Pass |
| VII Security | `reports.view` route rule (039); scope from `authz.ScopeOf(ctx)` inside the year read; out-of-scope plans never counted or named; `level` and `widget` allowlisted; export needs `data.export`; files never stored | Pass |
| VIII Clean construction | Comment budget per file; one concept per file | Pass |
| IX Experience | Tabs in the address; Back keeps them; states say why (039); links open All work filtered | Pass |
| X Palette | 036's bars with the chart tokens; Not filled in and Does not apply in neutral tokens, labelled, never colour alone | Pass |
| B1–B9, F1–F10 | Gin only in `handler.go`; ctx first; contract first; tests on fakes; TanStack Query hooks in `api.ts`; Zod for `level`; no chart library | Pass |

## Project Structure

### Documentation (this feature)

```text
specs/043-plan-details-report/
├── spec.md
├── plan.md              # this file
├── research.md          # R1–R5
├── quickstart.md
├── contracts/api.md     # for the wave A contract
├── checklists/requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root) — only issue #36's "Owns" paths

```text
backend/internal/measure/plandetails/        # NEW, read-only, no store
├── model.go          # PlanDetailsReport, cards, rows, widgets, values (API types)
├── ports.go          # Transactor, Planning
├── service.go        # Report(ctx, actor, filter, level): one Read, the tab for a level
├── level.go          # the records of a level, filtered by 039's measure.Filter; cards
├── builtin.go        # activity type, status, timing, start periods, planned cost entered, goal entered
├── profile.go        # one row per active attribute of the level; filled in; Does not apply
├── attribution.go    # 016 FR-041 for one record: values, % split, counts in each (R2)
├── widget.go         # ordering, "Other" after ten, totals, per cents
├── sheets.go         # measure.Report: Sheets for 039's export
├── handler.go        # level and 039's parameters; GET /api/v1/reports/plan-details
└── *_test.go         # rules, equality with Breakdown, handler, benchmark
backend/internal/app/measure_plandetails.go  # registerPlanDetails (039's stub): adapter + mount
backend/internal/app/app_test.go             # remove 043's group from notYetServed only (build guide §4.3)
frontend/src/features/measure/plan-details/  # replaces 039's stub page
├── PlanDetailsPage.tsx, LevelTabs.tsx, PlanDetailsCards.tsx, BuiltInSection.tsx, AttributeRow.tsx,
│   ValueWidget.tsx, FilledInWidget.tsx, allWorkLink.ts
├── api.ts, schemas.ts (level), index.ts
└── *.test.tsx
frontend/e2e/measure-plan-details.spec.ts
```

**Structure Decision**: web application; one business package and one feature folder as constitution B2 and F2
lay them out, wired through 039's stubs.

## Business rules and the tests that prove them

| Rule | Where | Test |
|---|---|---|
| Records of a level in scope, filtered by 039's filter; cards: count, planned cost, all filled in % (FR-003, FR-005) | `level.go` | `TestCards_FR005`, `TestLevel_OnlyPlansInScope_FR003` (contract, persona with one of two plans) |
| Planned cost follows "View plans", no finance gate (FR-004, D-237) | service, routes | `TestReport_NoFinanceView_ShowsPlannedCost_FR004` |
| Built-in widgets: type, status, timing, start periods with No dates and Outside year, planned cost entered, goal entered (FR-006, FR-007, 043-C1) | `builtin.go` | `TestBuiltIn_Activity_FR006`, `TestStartPeriods_MonthsQuarters_FR006`, `TestGoalEntered_FR006`, `TestBuiltIn_Containers_FR007` |
| One row per active attribute of the level, Settings' order; inactive attribute absent; inactive option labelled (FR-008) | `profile.go` | `TestRows_ActiveInDisplayOrder_FR008`, `TestRows_InactiveOption_FR008` |
| Three widgets for choice and yes/no, filled-in only for text and number (FR-009, FR-010, 043-C3) | `profile.go` | `TestRows_WidgetsByType_FR009_FR010` |
| Attribution: one value; % split counts once per value with share > 0 and divides cost half up, remainder last; multiple choice counts whole in each, total once (FR-011) | `attribution.go` | `TestAttribution_Split_FR011`, `TestAttribution_CountsInEach_FR011`, `TestAttribution_ZeroShare_FR011` |
| Same figures as All work's Breakdown (SC-002) | `attribution.go`, `profile.go` | `TestAttribution_EqualsAllWorkBreakdown_SC002` (every choice and yes/no attribute of the demo year, every level) |
| Not filled in; Does not apply from the business-unit condition, out of the filled-in per cent (FR-012, 043-C2) | `profile.go` | `TestDoesNotApply_FR012`, `TestFilledInPercent_LeavesOutDoesNotApply_FR012` |
| Display currency at the current rate, summed in USD then converted once (FR-013) | `widget.go` | `TestPlannedCost_InDisplayCurrency_FR013` |
| "Other" after ten values; table and export list every value (FR-017) | `widget.go`, `sheets.go` | `TestOther_AfterTenLargest_FR017`, `TestSheets_EveryValueAndTotal_FR017` |
| States: no plans, no data, no attributes set up, filters exclude everything (FR-016) | `service.go` | `TestStates_FR016` |
| `level` and `widget` allowlisted; access rules (FR-001, FR-003) | `handler.go`, 039's routes | `TestHandler_Level_FR001`, contract tests `TestPlanDetails_Contract`, `TestPlanDetails_NoReportsView_FR003`, `TestPlanDetailsExport_NeedsDataExport` |
| Each tab within 3 s at the 028 volume (SC-001) | — | `BenchmarkPlanDetails_SC001` (R5) |
| Web: one test per widget state; tab in the address; link opens All work with the filter (FR-001, FR-014, FR-016) | `features/measure/plan-details` | `PlanDetailsPage.test.tsx`, `AttributeRow.test.tsx`, `allWorkLink.test.ts` (axe in each) |
| US1 end to end on the demo data, axe (spec US1) | e2e | `measure-plan-details.spec.ts` |

Benchmark result: _recorded here after implementation_.

## Dependencies and hand-offs

- **039 (lane A)** — `measure.Filter`, `measure.Periods`, `measure.Report`/`Sheets`, `ReportState`, the
  `registerPlanDetails` stub, the `/reports/plan-details` route and rules, `ReportFrame`. Not pushed on
  2026-10-31; names in `contracts/api.md` are marked *(039)* for the integrator to align.
- **Integrator** — fold `contracts/api.md` into the wave A contract; decide on exporting the attribution rule
  from `allwork` (R2); add "Does not apply" to the glossary; confirm 046 adds Is integrated (043-C4).
- **#DEMO (lane D)** — demo data with an activity-level attribute hidden by a business-unit condition, empty
  values at every level, a % split and an activity with a goal, so every widget state shows.

## Complexity Tracking

| Violation | Why needed | Simpler alternative rejected because |
|---|---|---|
| A local copy of 016's attribution rule (`attribution.go`, ~40 lines) | The rule is unexported in `allwork`, which is outside lane D's paths | Exporting it is better and is proposed (R2); until approved, the copy is pinned value-for-value to All work's Breakdown by `TestAttribution_EqualsAllWorkBreakdown_SC002` |
