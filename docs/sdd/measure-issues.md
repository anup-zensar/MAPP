# Measure module — GitHub issues

One issue per feature and per cross-cutting task of [measure-module-build.md](measure-module-build.md). Create
them in the repository with the labels below, two milestones (**Measure wave A**, **Measure wave B**) and the
lane's developer as assignee; paste each body as it stands. Labels: `measure`, `wave-a` or `wave-b`, `lane-a`…
`lane-d`, `spec` for the kick-off tasks. Every feature issue follows the same checklist.

**Common checklist (copy into every feature issue)**

```
- [ ] Branch `feature/0NN-short-name` from `measure-module`
- [ ] `/speckit-specify` citing the client FRs and FA §E; `/speckit-clarify` answers recorded as `0NN-Cn`
- [ ] `/speckit-plan` (short), `contracts/api.md` handed to the integrator for the wave contract
- [ ] `/speckit-tasks`, `/speckit-analyze` with no CRITICAL issue
- [ ] `/speckit-implement` one user story at a time; `scripts/check-changed.ps1` green after each
- [ ] Scope-isolation test for every scoped store read; contract test for every operation; one test per widget state
- [ ] End-to-end test for every P1 story in `frontend/e2e/measure-<report>.spec.ts` (axe included)
- [ ] Benchmark at the 028 volumes recorded in `plan.md`
- [ ] `/speckit-converge` clean; feature-map status change written in the pull request
- [ ] Pull request into `measure-module`, three checks green, one approval
```

---

## #M0 · Kick-off: governance and context files — lane: integrator · 8 h · `spec`

Done on 2026-10-30: constitution 4.8.0, decisions D-366–D-370 and Q-028–Q-031, feature map, the build guide,
`hive9-measure-summary.md`, `shell-measure-requirements.md`, `measure-metrics.md`, CI on `measure-module`.
Remaining: `scripts/check-changed.ps1` and its Claude Code hook in `.claude/settings.json` (format, lint, comment
budget, file length, tests of changed packages and folders).

- [ ] `scripts/check-changed.ps1` runs in under two minutes on a one-feature change
- [ ] Hook runs it after each implementation step and prints only failures

## #039 · Measure foundation — lane A · 28 h · `wave-a`

**Goal.** The Reports area and everything every report shares: report frame, filter panel (year, plans, Shell
attributes per level with *Not filled in*, owner, activity type, status, team), periods (months, quarters),
display currency, saved report views (per person, a named address, star a default), table view and CSV/Excel
export of every widget, the `reports.view` permission, the route rules and the wiring hub with one stub per report.

**Owns.** `backend/internal/measure`, `access/routes_measure.go`, `app/measure.go` + `app/measure_<report>.go`
stubs, migration 00028 (`report_views`), `frontend/src/features/measure/` root and `shared/`, every
`/reports/…` route in `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts` with the stubs,
`e2e/measure.spec.ts`. Writes the **wave A contract** into `openapi.yaml` from every lane's `contracts/api.md`.

**References.** FA §E2.4, §E3.0, §E3.9.0; client 016 US2, US6–US8 (FR of the shared frame); D-367, D-368;
036 `dashboard` package and `features/dashboards` as the model.

**Stories.** US1 open the Reports area and a report with filters in the address (P1) · US2 save, star and delete
a view (P1) · US3 table view and export of a widget (P1) · US4 empty states that say why (P2).

**Order.** Backend types, registry, rules, hub and migration first (hour 16, pushed so lanes can import), then
the web frame and the stubs (hour 28). Common checklist.

## #040 · Plan budget report — lane A · 16 h · `wave-a`

**Goal.** Budget, planned cost, committed and actual by period split at today, summary cards, and the **ranked
list** (show plan, campaign, initiative or activity; group by none, plan, owner or a Shell attribute; sort by any
figure; order; page size; open a row to show its level below with the row on top; a link to the record). The
ranked list is exported for 049.

**Owns.** `measure/planbudget`, `app/measure_planbudget.go`, `queries/measure_planbudget.sql`,
`features/measure/plan-budget/` (exports `RankedList`), `e2e/measure-plan-budget.spec.ts`.

**References.** FA §E3.4, §E3.2.5 (drill); client 017 US1, US3, FR-001–FR-021 (money series, cards, list);
[measure-metrics.md](../product/measure-metrics.md) §1; 036 figures (D-342), 010 Budget & cost (D-266).

**Stories.** US1 spend pacing by period with the today split (P1) · US2 ranked list with drill and record link
(P1) · US3 group by and sort (P2) · US4 export (P2). Common checklist.

## #041 · Approvals report — lane C · 10 h · `wave-a`

**Goal.** Status mix of approval rounds by record type (activities, purchase requests, modified purchase
requests), average days to a decision, and an approvals grid (record, approver, level, submitted, decided,
outcome, amount, comments) with approver, status and record-type filters, search, paging and export.

**Owns.** `measure/approvalreport`, `app/measure_approvalreport.go`, `queries/measure_approvals.sql`,
`features/measure/approvals/`, `e2e/measure-approvals.spec.ts`.

**References.** FA §E3.6; client 018 US1, US4, FR-001–FR-012 (widgets, grid columns, filters); 017's
average-days rule (D-342); 031 record types.

**Stories.** US1 workload and cycle time (P1) · US2 find and export approval records (P2). Common checklist.

## #042 · Purchase request report — lane C · 14 h · `wave-a`

**Goal.** Counts and amounts by status, commitment by vendor and by owner per delivery month (Approved and Closed
lines by delivery date), invoiced per line, and a grid vendor › purchase request › line › line item with
column choice, paging and export. Vendor delivery reliability is **not** built (Q-030).

**Owns.** `measure/purchasereport`, `app/measure_purchasereport.go`, `queries/measure_purchases.sql`,
`features/measure/purchase-requests/`, `e2e/measure-purchase-requests.spec.ts`.

**References.** FA §E3.7; client 018 US2, US5, FR-013–FR-030; 032 statuses and amounts (D-335), 033 GSAP PR
and PO, 015 invoices.

**Stories.** US1 forward commitment by vendor and owner (P1) · US2 the hierarchical grid (P2) · US3 export (P2).
Common checklist.

## #043 · Plan details report — lane D · 10 h · `wave-a`

**Goal.** Generated from Settings: for each active Shell attribute of each level, a chart of records per value, a
chart of planned cost per value and a count of records with the attribute *Not filled in*; plus Planned cost
entered, Goal entered and, once 046 exists, Is integrated. Level tabs Plan · Campaign · Initiative · Activity.

**Owns.** `measure/plandetails`, `app/measure_plandetails.go`, `queries/measure_plandetails.sql`,
`features/measure/plan-details/`, `e2e/measure-plan-details.spec.ts`.

**References.** FA §E3.9.1–E3.9.3; client 019 US2, FR-010–FR-020; 016 Breakdown rules (D-270, D-336); 036
completeness panel; domain model §2.6.

**Stories.** US1 attribute spread and completeness at the activity level (P1) · US2 the other levels (P2) · US3
export (P2). Common checklist.

## #044 · GSAP link reports — lane D · 10 h · `wave-a`

**Goal.** Three tables in one report page with tabs: SWBS requested and not received (line item, plan, requested
on, days waiting), purchase requests waiting for GSAP (approved and routed, no GSAP PR number), and the purchase
orders per line item (the PO ledger: line item, purchase request, GSAP PR number, PO number and date, PO amount,
invoiced, paid). Each with paging and export; financial columns need `finance.view`.

**Owns.** `measure/gsaplink`, `app/measure_gsaplink.go`, `queries/measure_gsaplink.sql`,
`features/measure/gsap-link/`, `e2e/measure-gsap-link.spec.ts`.

**References.** FA §E3.10.3 (PO report columns), §F4.4 (SWBS Not Received, Pending PR); client 019 US3,
FR-021–FR-030; 030 SWBS statuses, 032 routing, 033 GSAP PR and PO.

**Stories.** US1 SWBS not received (P1) · US2 waiting for GSAP (P2) · US3 purchase orders per line item (P2).
Common checklist.

## #045 · User access report — lane D · 6 h · `wave-a`

**Goal.** People with role, teams, job title and last sign-in; search, sort, export; opens with `access.manage`.
Demo sign-in records the time of a successful sign-in (migration 00029, `people.last_signed_in_at`, UTC).

**Owns.** `measure/useraccess`, `app/measure_useraccess.go`, the sign-in change in `auth`, migration 00029,
`features/measure/user-access/`, `e2e/measure-user-access.spec.ts`.

**References.** FA §E3.9.5; client 019 US7, FR-040–FR-044; 003 people and teams, 014 roles.

**Stories.** US1 the report with last sign-in (P1) · US2 export (P2). Common checklist.

## #046 · Performance data from a simulated feed — lane B · 30 h · `wave-b`

**Goal.** The record types behind every performance figure, built like the simulated GSAP (ADR-0016, D-370):
data sources with column mappings (Settings › Performance data), performance facts (inquiries, MQLs, deals,
revenue, web metrics) loaded by file upload or a simulated pull, matched to activities on activity ID, media
code, integration ID or external name, rows to review and fix, media codes generated per activity, the external
name set when an activity is approved, and *Is integrated* on the activity. A Performance section on the
activity page shows the activity's facts once it is approved. Permissions `performance.import` and
`performance.link` (reserved by D-367).

**Owns.** `backend/internal/performance`, the `simulatedfeed` adapter, `app/performance.go`, migrations
00030–00032, `features/performance/`, the Performance section slot in `features/activities/ActivityPage.tsx`,
`e2e/performance-data.spec.ts`, wave B demo facts in `seed/measure.go`. Writes ADR-0019 (simulated performance
feed) with its plan and, with 047, the **wave B contract**.

**References.** FA §E4.1, §E7.2–E7.4, §G7; client 020 US1–US6, FR-001–FR-040; 015 import, matching and review
pattern; 030 external keys.

**Stories.** US1 data source and mappings (P1) · US2 upload and match (P1) · US3 performance on an approved
activity (P1) · US4 external names and integration IDs (P1) · US5 review unmatched rows (P2) · US6 media codes
(P2). Common checklist.

## #047 · Projections — lane B · 14 h · `wave-b`

**Goal.** Models gain velocity (months between stages) and the allowed goal stages; Settings gains activity
categories (groups of activity types used by the reports); a pure projection calculation gives projected INQ,
MQL and revenue per period from each activity's goal, forward from its start date to the plan year's end.

**Owns.** `measure/projection`, the model and category changes in `settings` and `features/settings/models/`,
`features/settings/activity-categories/`, migration 00033, `e2e/projections.spec.ts`.

**References.** FA §E3.2.6, §E7.5.1; client 020 US7, FR-041–FR-050; domain model §4.5.

**Stories.** US1 velocity and allowed stages on a model (P1) · US2 projected figures per period (P1) · US3
activity categories (P2). Common checklist.

## #048 · Summary report — lane C · 14 h · `wave-b`

**Goal.** Projected against goal with the variance word, the trend split at today, four league tables (top by
revenue, performance, cost, ROI for a chosen Shell attribute, each refreshing alone), stage volumes and
conversion rates against goal, and links into the Revenue and Funnel reports.

**Owns.** `measure/summary`, `app/measure_summary.go`, `features/measure/summary/`, `e2e/measure-summary.spec.ts`.

**References.** FA §E3.1; client 016 US1, US4, FR-020–FR-045; [measure-metrics.md](../product/measure-metrics.md) §2.

**Stories.** US1 are we on track (P1) · US2 league tables (P2) · US3 funnel health and links (P2). Common checklist.

## #049 · Revenue and funnel-stage reports — lane B · 14 h · `wave-b`

**Goal.** Two reports on 040's template: summary cards per stage and activity category (plan goal, goal
coverage, activity goals, ROI), actual, projected and goal by period split at today with the data table, and the
ranked list sorted by actual revenue, projected revenue, ROI or a stage; the Funnel report chooses the stage.

**Owns.** `measure/revenue`, `app/measure_revenue.go`, `features/measure/revenue/`, `features/measure/funnel/`,
`e2e/measure-revenue.spec.ts`.

**References.** FA §E3.2, §E3.3; client 016 US3, US5, US9, FR-046–FR-084; 040's ranked list.

**Stories.** US1 rank and drill revenue (P1) · US2 a funnel stage (P2) · US3 summary cards per category (P3).
Common checklist.

## #050 · Attribution and ROI — lane A · 16 h · `wave-b`

**Goal.** Settings › Attribution chooses the model that credits revenue facts to activities (first touch, last
touch, equally distributed, linear time decay, U-shaped — the five that need no weights); attributed revenue and
ROI appear on 048 and 049 with the model and the cost basis named next to every figure; *Use planned cost* is a
labelled choice, never silent.

**Owns.** `measure/attribution`, `app/measure_attribution.go`, migration 00034, `features/settings/attribution/`,
ROI parts in `features/measure/shared/`, `e2e/attribution.spec.ts`.

**References.** FA §E7.5; client 020 US8, FR-051–FR-065; measure-metrics §2 (ROI, confirm the definition in the spec).

**Stories.** US1 choose the model and see credited revenue (P1) · US2 ROI with its basis named (P1) · US3 change
the model and see every figure follow (P2). Common checklist.

## #MA · Wave A integration — lane A · 6 h · `wave-a`

- [ ] Merge 040–045 into `measure-module`; apply reported contract fixes; `pnpm run api:types`
- [ ] Move the lanes' `0NN-Cn` clarifications into the decision log (D-371 onwards); apply feature-map statuses
- [ ] Full check on `measure-module`: backend, frontend and e2e green; pull request `measure-module` → `develop`

## #MB · Wave B integration — lane B · 6 h · `wave-b`

- [ ] Wave B contract written before 048–050 start; merge 046–050; decisions and statuses moved as in #MA
- [ ] Pull request `measure-module` → `develop`

## #QA-A · Wave A QA pass — lane C · 8 h · `wave-a`

- [ ] Browser QA of every Reports screen on the demo data under four roles (every scope, no financial access)
- [ ] Findings as issues on the owning feature; the critical and high ones fixed before the wave merges

## #QA-B · Wave B QA pass — lane D · 8 h · `wave-b`

- [ ] As #QA-A for the Summary, Revenue, Funnel, Performance data and Attribution screens

## #DEMO · Measure demo data and fixtures — lane D · 8 h · `wave-a`

- [ ] `seed/measure.go` behind the call 039 creates: goals on the demo activities, dates and spreads that fill
      every period, approvals and purchase requests in every status, a routed request without a GSAP number, a
      line item waiting for an SWBS code
- [ ] `docs/product/demo-users-and-data.md` gains the Measure story; e2e fixtures in `e2e/measure.ts`
