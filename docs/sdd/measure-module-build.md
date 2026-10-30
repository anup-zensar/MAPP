# Measure module build guide (features 039–050)

**Read this first** when you work on the Measure module, then your feature's folder under `specs/`. Search the
rest (constitution, design system, decision log) instead of reading it whole. This guide comes from the Measure
kick-off of 2026-10-30 (decisions D-366–D-370) and applies the Plan module's lessons: a fixed contract per wave,
one owner per file, quality checked while building, one user story per implementation prompt.

Companion files: [hive9-measure-summary.md](../reference/hive9-measure-summary.md) (how Hive9's Measure works),
[shell-measure-requirements.md](../reference/shell-measure-requirements.md) (Shell's client specs 016–020, 025,
condensed with their FR IDs), [measure-metrics.md](../product/measure-metrics.md) (every figure defined once) and
[measure-issues.md](measure-issues.md) (the GitHub issues, one per feature).

## 1. What the module is

Measure is SP&A One's **read-only reporting layer**, shown on screen as **Reports**. Every figure is one of three
things: added up from plan and finance records MAPP already holds, loaded from a simulated performance feed
(wave B), or calculated from those two. No report changes business data; the only things a report saves are a
person's saved views. The server works out every figure (constitution IV); the web app only draws.

| Wave | Features | Data | Status |
|---|---|---|---|
| **A** — reports from data MAPP holds | 039 foundation · 040 Plan budget · 041 Approvals · 042 Purchase requests · 043 Plan details · 044 GSAP link · 045 User access | plans, budgets, line items, approvals, purchase requests, GSAP transactions, people and roles | In scope (constitution 4.8.0, D-366) |
| **B** — performance reporting | 046 performance data from a simulated feed · 047 projections · 048 Summary · 049 Revenue and funnel stages · 050 attribution and ROI | goals and models (002, 004, 006) plus performance facts from a simulated feed behind an adapter (D-370) | In scope; starts once wave A's foundation has landed |
| **C** — blocked by modules that are out of scope | Finance budget report · AR invoice details · China budget pack · vendor delivery reliability · custom dashboard engine and self-service builder · report sharing and scheduling by e-mail | Finance budgets module, accrual requests, goods receipts, e-mail | Out; mention only as future integration points |

## 2. Features at a glance

Hours are all-in with Claude Code (specify, clarify, plan, tasks, analyse, implement, end-to-end test, converge,
review fixes), for one developer. The lane letters are the four developers; A and B carry the complex features.

| # | Feature | Lane | Hours | Go package (`backend/internal/…`) | Web folder (`frontend/src/features/…`) | Routes (`/api/v1/…`, web) | Tables · migrations | Depends on |
|---|---|---|---|---|---|---|---|---|
| 039 | **Measure foundation**: Reports area, report frame, filter panel (year, plans, Shell attributes per level with *Not filled in*, owner, activity type, status, team), periods (months, quarters), display currency, saved report views, table view and CSV/Excel export of every widget, `reports.view`, route rules, wiring hub | A | 28 | `measure` (filters, periods, views, export, registry); `access/routes_measure.go`; `app/measure.go` | `measure/` (ReportsPage, `shared/` ReportFrame, FilterPanel, PeriodControl, ReportTable, ReportExportMenu, SavedViewsMenu, `api.ts`, `schemas.ts`, `index.ts` with every report's stub) | `reports`, `reports/filters`, `reports/views…`; `/reports` | `report_views` RV · 00028 | 010, 014, 016, 036 |
| 040 | **Plan budget report**: budget, planned cost, committed and actual by period split at today; summary cards; **ranked list** by plan, campaign, initiative or activity with group by, sort, order, page size, two-level drill and a link to the record (the template 049 reuses) | A | 16 | `measure/planbudget`; `app/measure_planbudget.go`; `queries/measure_planbudget.sql` | `measure/plan-budget/` (+ `RankedList` exported for 049) | `reports/plan-budget`, `…/ranked`, `…/export`; `/reports/plan-budget` | — | 039, 036, 010, 015 |
| 041 | **Approvals report**: status mix by record type, average days to a decision, approvals grid with approver, status and record-type filters, export | C | 10 | `measure/approvalreport`; `app/measure_approvalreport.go`; `queries/measure_approvals.sql` | `measure/approvals/` | `reports/approvals`, `…/rows`, `…/export`; `/reports/approvals` | — | 039, 008, 017, 031 |
| 042 | **Purchase request report**: counts and amounts by status, commitment by vendor and by owner per delivery month, invoiced per line, vendor › purchase request › line › line item grid, export | C | 14 | `measure/purchasereport`; `app/measure_purchasereport.go`; `queries/measure_purchases.sql` | `measure/purchase-requests/` | `reports/purchase-requests`, `…/grid`, `…/export`; `/reports/purchase-requests` | — | 039, 032, 033, 015 |
| 043 | **Plan details report**: generated from Settings — for each Shell attribute at each level, how many records have each value, their planned cost and how many have it filled in; Planned cost entered, Goal entered | D | 10 | `measure/plandetails`; `app/measure_plandetails.go`; `queries/measure_plandetails.sql` | `measure/plan-details/` | `reports/plan-details`, `…/export`; `/reports/plan-details` | — | 039, 002, 006 |
| 044 | **GSAP link reports**: SWBS requested and not received, purchase requests waiting for GSAP, purchase orders per line item (the PO ledger) | D | 10 | `measure/gsaplink`; `app/measure_gsaplink.go`; `queries/measure_gsaplink.sql` | `measure/gsap-link/` | `reports/gsap-link/{report}`, `…/export`; `/reports/gsap-link` | — | 039, 030, 032, 033 |
| 045 | **User access report**: people with role, teams and last sign-in; export | D | 6 | `measure/useraccess`; `app/measure_useraccess.go`; `auth` records the sign-in time | `measure/user-access/` | `reports/user-access`, `…/export`; `/reports/user-access` | `people.last_signed_in_at` · 00029 | 039, 003, 014 |
| 046 | **Performance data from a simulated feed**: data sources and column mappings, upload or simulated pull of performance facts (inquiries, MQLs, deals, revenue), matching to activities (activity ID, media code, integration ID, external name), rows to review, external name set on approval, *Is integrated* | B | 30 | `performance` (facts, sources, imports, matching, review); `simulatedgsap`-style adapter `simulatedfeed`; `app/performance.go` | `performance/` (Settings › Performance data, review); a Performance section on the activity page | `performance/…`; `/settings/performance-data`, `/activities/:id#performance` | `performance_sources`, `performance_imports`, `performance_facts`, `media_codes` · 00030–00032 | 006, 008, 015 (import pattern) |
| 047 | **Projections**: velocity and allowed goal stage on models, activity categories in Settings, projected INQ, MQL and revenue per period from the model's rates | B | 14 | `measure/projection` (pure); model fields in `settings` | `settings/models/` (velocity, categories), `measure/shared/Projection…` | `settings/models…` (extended); `settings/activity-categories` | `models` columns, `activity_categories` · 00033 | 002, 046 |
| 048 | **Summary report**: projected against goal with variance, four league tables (revenue, performance, cost, ROI) by a chosen attribute, stage volumes and conversion against goal, links to the Revenue and Funnel reports | C | 14 | `measure/summary`; `app/measure_summary.go` | `measure/summary/` | `reports/summary`, `…/export`; `/reports/summary` | — | 039, 046, 047 |
| 049 | **Revenue and funnel-stage reports**: summary cards per stage and activity category, actual, projected and goal by period split at today, 040's ranked list sorted by revenue or a stage | B | 14 | `measure/revenue`; `app/measure_revenue.go` | `measure/revenue/`, `measure/funnel/` | `reports/revenue`, `reports/funnel`, `…/ranked`, `…/export`; `/reports/revenue`, `/reports/funnel` | — | 040, 046, 047 |
| 050 | **Attribution and ROI**: a chosen attribution model credits revenue to activities; ROI with its cost basis and model named next to every figure | A | 16 | `measure/attribution`; `app/measure_attribution.go` | `settings/attribution/`, ROI parts in `measure/shared/` | `settings/attribution`; ROI fields of 048 and 049 | `attribution_settings` · 00034 | 046, 048, 049 |

Cross-cutting work: wave A integration and contract fixes (A, 6 h), wave B integration (B, 6 h), a browser QA
pass on the Reports screens after each wave (C for A, D for B, 8 h each), the Measure demo-data story and
end-to-end fixtures (D, 8 h). Total 218 h plus the kick-off's 8 h; plan on about 280 h with a 25% buffer.

| Lane | Features | Hours |
|---|---|---|
| A | 039, 040, 050, wave A integration | 66 |
| B | 046, 047, 049, wave B integration | 64 |
| C | 041, 042, 048, wave A QA pass | 46 |
| D | 043, 044, 045, demo data and fixtures, wave B QA pass | 42 |

## 3. Build order

| Step | Who | What | Lands by |
|---|---|---|---|
| 0 | kick-off | This guide, the context files, constitution 4.8.0, feature map, decisions D-366–D-370, CI on `measure-module` | done |
| 1 | all | Specs and clarifications of the wave A features, each developer for their lane; A writes 039's spec and the **wave A contract** (every operation of 039–045 in `openapi.yaml`, from each lane's `contracts/api.md`); B specs 046 and 047 | hour 8 |
| 2 | A | 039 backend first: `measure` package with the filter model, periods and the report registry, `routes_measure.go` with every wave A rule, `app/measure.go` calling one stub function per report, migration 00028; pushed to `measure-module` so lanes can import it | hour 16 |
| 3 | A · C · D | 040–045 backends against the contract and 039's types; A continues with 039's web frame and the `features/measure/index.ts` stubs | hour 28 |
| 4 | C · D | 041–045 web parts inside the frame; A: 040 | hour 56 |
| 5 | B | 046 from hour 0 (it needs no report frame), then 047 | hour 44 |
| 6 | A | Wave A integration on `measure-module`; C: QA pass; D: demo data and fixtures | hour 70 |
| 7 | A · B · C | 048, 049, 050; the **wave B contract** written by B before the reports start | hour 130 |
| 8 | B · D | Wave B integration; QA pass; `measure-module` → `develop` by pull request | hour 150 |

Calendar: wave A in about 70 working hours of the four lanes (two weeks), both waves in about 150 (four weeks),
inside the programme's 10 Nov – 4 Dec 2026 slot.

## 4. Lanes and ownership — how four people never edit the same file

1. **Own your folders.** A feature changes only its Go package, its web folder, its `specs/` folder, its wiring
   file `internal/app/measure_<report>.go`, its query file `queries/measure_<report>.sql`, its end-to-end file
   `frontend/e2e/measure-<report>.spec.ts` and the migration numbers reserved above. Exclusive ownership also
   covers the few outside files a feature must touch: 045 owns the sign-in change in `auth`, 046 the Performance
   section slot in `features/activities/ActivityPage.tsx`, 047 the model changes in `settings`.
2. **Registration points are created once, by 039.** `app/measure.go` calls `registerPlanBudget(w)`,
   `registerApprovals(w)`… from the start, each an empty function in its own `measure_<report>.go`;
   `features/measure/index.ts` exports every report page, each a stub until its lane replaces it;
   `app/navigation.ts`, `app/pages.tsx` and `app/router.tsx` carry every `/reports/…` route from the start;
   `routes_measure.go` holds every wave A rule. Lanes fill their stub, never the hub.
3. **The contract is fixed per wave** (D-192 again). `backend/api/openapi.yaml` gets the whole wave's operations
   in step 1 (wave A) and step 7 (wave B). While building, a lane that finds a contract problem reports it in its
   pull request; the integrator applies the fix. Remove your operations from `notYetServed` in
   `internal/app/app_test.go`: they are listed in a group per feature, so the deletions never touch another lane's lines.
4. **Shared files a lane never edits:** `openapi.yaml`, `routes_measure.go`, `app/measure.go`, `app.go`,
   `navigation.ts`, `pages.tsx`, `router.tsx`, `features/measure/index.ts`, `components/index.ts`,
   `styles/tokens.css`, `seed/seed.go`, `docs/sdd/feature-map.md`, `docs/sdd/decision-log.md`, the constitution.
   Shared components a report needs go into `features/measure/shared/` through 039's lane.
5. **Decisions.** A clarification answered while specifying is written in that spec's *Clarifications* section
   with a provisional label (`039-C1`). The integrator moves them into the decision log as D-… at the end of the
   wave, so four people never append to the same table. Feature-map status changes go in the pull-request
   description and are applied by the integrator the same way.
6. **Migrations and sqlc.** Numbers are reserved in §2; a feature with no reserved number adds no migration. Run
   `sqlc generate` after every rebase: the generated per-query files are yours, `models.go` only changes with a
   migration.
7. **Demo data.** D adds `seed/measure.go` (goals, dates and amounts that make every report meaningful) behind one
   call that 039 creates in `seed.go`; the performance facts of wave B are seeded by 046 in the same file's wave B part.
8. **Branches.** `feature/0NN-short-name` from `measure-module`; pull requests into `measure-module` (CI runs the
   three checks there since the kick-off); `measure-module` → `develop` after each wave. Rebase on
   `measure-module` before opening the pull request.

## 5. The report pattern — backend

- Package files as every area: `model.go · store.go · store_postgres.go · service.go · handler.go` plus tests and a
  `ports.go` declaring the small consumer interfaces the report reads through (the 036 `dashboard` package is the
  model: `Planning`, `Spending`, `Rounds`, `Purchases`); `internal/app` holds the adapters.
- **Reads in scope in SQL** (D-356): the plans of the year in the actor's scope come from one query; everything
  else is read for those plans only. A scope-isolation test for every scoped store read (constitution VI).
- **Figures reuse the existing definitions** of [measure-metrics.md](../product/measure-metrics.md); a report never
  defines Committed, Actual or Planned cost itself. One `Read` snapshot per request; statements never grow with the
  records (028); a benchmark at the 028 volumes for every report (036 T034 is the model).
- **Filter and period** come from 039's `measure.Filter` and `measure.Periods`, parsed once in the handler from
  the query string; the service never reads the request.
- Money as `{amount, currency}` in the display currency at the current rate (D-303); "—" where the actor may not
  see financial data (`finance.view` per plan, D-301, D-343) with the count of plans the figure covers.
- Route rule per operation in `routes_measure.go`: `need(authz.ReportsView)` for every report, `inPlan` where a
  record is named, `need(authz.DataExport)` for exports, `need(authz.AccessManage)` for the user access report.
- Export: `measure.Export` writes the widget's table as CSV or Excel through `platform/sheet`; nothing is stored.

## 6. The report pattern — web

- `ReportFrame` (039) gives every report its page header, the filter panel, the period and currency controls,
  saved views and an "as of" line; a report supplies its widgets. Filters live in the page address so Back keeps
  them and a link shares them (constitution F3, D-368); a saved view is a named address.
- Every widget has a **Table** view and an export; charts are the 036 SVG bars and lines with the four chart
  tokens (D-345), never a chart library. The "today" marker splits actual from planned or projected.
- Empty states say why (D-368): no plans you may see for the year · you can't see financial data · no data yet
  (and what would create it) · the filters exclude everything, with a one-click reset.
- Data through TanStack Query from `api.ts` typed by the generated contract; Zod schemas in `schemas.ts` for the
  address; tests with MSW `reply()` and one test per state a widget renders; axe on every page.

## 7. Words and conventions

| Hive9 says | SP&A One says |
|---|---|
| Measure | Reports (the module stays "Measure" in docs) |
| HUD / heads-up display | Summary cards |
| (Not Completed) | Not filled in |
| Monthly / Quarterly | Period: Months · Quarters |
| My Saved Views | Saved views |
| View By / Group By / Sort By | Show · Group by · Sort by |
| Tactic Category · Tactic Type | Activity category · Activity type |
| Waterfall | Funnel stages (INQ → MQL → Revenue) |
| Program · Tactic | Initiative · Activity |
| Share Report | not offered (e-mail is out of scope); the address is the link |
| Stack ranking, drill | Ranked list, open a row |

Everything else as CLAUDE.md: sentence case, money as exact decimals, dates `YYYY-MM-DD` on the wire, comments at
most 5% of a file, files at most 300 non-blank lines, release age of every module, Shell palette tokens only.

## 8. The cycle per feature (Spec Kit) — and how to keep prompts short

| Step | Command | Rule for this module |
|---|---|---|
| Specify | `/speckit-specify` | Cite the client FRs ("client 017 FR-005") and FA §E sections; read-only reports have no CRUD scenarios — write the state scenarios instead (data, empty, no access, filtered to nothing) |
| Clarify | `/speckit-clarify` | At most five questions; answers go into *Clarifications* as `0NN-Cn` with the reasoning; anything that changes scope goes to the integrator the same day |
| Plan | `/speckit-plan` | `plan.md` is short: rules and the tests that prove them, queries and their scope filter, permissions, benchmark; `contracts/api.md` lists the operations for the wave contract; `data-model.md` only when there is a new table; `research.md` only for real choices |
| Tasks | `/speckit-tasks` | Tests first; one phase per user story; 10–20 tasks per story |
| Analyse | `/speckit-analyze` | Before implementing, not after |
| Implement | `/speckit-implement` **one user story at a time** | Say which story in the prompt ("implement US1 of 041"); backend and web of a story may run as two prompts in two worktrees from the fixed contract |
| Check | `scripts/check-changed.ps1` | After every story: format, lint, comment budget, file length, the tests of changed packages and folders |
| End to end | Playwright + axe | Every P1 story before the pull request; the file is yours (`measure-<report>.spec.ts`) |
| Converge | `/speckit-converge` | Remaining work appended to `tasks.md` or "clean" |

A story is done when its tests, its end-to-end test (P1) and the check script are green and the spec's scenarios
pass on the demo data. A feature is done when every story is, the benchmark is recorded, the pull request into
`measure-module` is approved with the three checks green and the status change is in its description.

## 9. Commands

Backend (`backend/`): `golangci-lint fmt ./...`, `golangci-lint run ./...`, `sqlc generate && sqlc vet`,
`go test -race -short ./...` (every test on fakes), `go test -race ./internal/measure/...` with `TEST_DATABASE_URL`
set for the scope-isolation tests, `go run ./cmd/seed`, `go run ./cmd/api`. Frontend (`frontend/`):
`pnpm run api:types` after a contract change, `pnpm run check`, `pnpm run test:e2e -- measure`. Both:
`scripts/check-changed.ps1` (kick-off task; until it exists run the checks of [quality-gates.md](quality-gates.md)).
