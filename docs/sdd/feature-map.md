# Feature Map — MAPP (Plan module phase)

Every feature of this phase, one Spec Kit folder each under `specs/`. **Numbers are the build order** for 001–013;
014–017 close the MVP gaps of Shell's BRD and are built in the order of D-307
([brd-gap-closure-plan.md](../reports/brd-gap-closure-plan.md)).
The map sets scope boundaries so no requirement is specified twice and none is missed. Statuses as of the final merge (`plan-module` at `a1ae8fe`, 21 October 2026); the open tasks of each feature are listed at the top of its `tasks.md` (014–017) and in the [final validation report](../reports/final-validation-report.md) §7.

**Status values:** Draft (spec written, waiting for product-owner review) · Clarifying · Planned ·
In progress · Built (merged on `plan-module` and working on the demo data; the product owner's acceptance and the items left open in its `tasks.md` remain) · Done.

| # | Folder | Feature | Priority | Depends on | Status |
|---|---|---|---|---|---|
| 001 | [001-app-shell-and-sign-in](../../specs/001-app-shell-and-sign-in/spec.md) | Demo sign-in, application shell, navigation, user menu, getting-started home, shared building blocks (activity-log and recycle-bin core) | P1 | — | Built |
| 002 | [002-settings-and-shell-attributes](../../specs/002-settings-and-shell-attributes/spec.md) | Settings: models, activity types, Shell attributes and options, currencies and rates, finance budget items | P1 | 001 | Built |
| 003 | [003-people-and-teams](../../specs/003-people-and-teams/spec.md) | People directory, teams and memberships, person picker, My profile | P1 | 001 | Built |
| 004 | [004-plans](../../specs/004-plans/spec.md) | Plans list and plan page; create, view, edit, copy, publish/unpublish, delete plans; budget, goals, Shell attributes | P1 | 002, 003 | Built |
| 005 | [005-campaigns-and-initiatives](../../specs/005-campaigns-and-initiatives/spec.md) | Campaigns and initiatives inside a plan; budgets flow down, costs roll up; move, copy, delete | P1 | 004 | Built |
| 006 | [006-activities](../../specs/006-activities/spec.md) | Activities: basics, dates, planned cost, Shell attributes with business-unit groups and % splits, goals, status, completeness | P1 | 005 | Built |
| 007 | [007-line-items](../../specs/007-line-items/spec.md) | Line items: currency, spread over months, finance coding, finance budget link, SWBS request flag, unassigned balance | P1 | 006 | Built |
| 008 | [008-activity-approvals](../../specs/008-activity-approvals/spec.md) | Approval workflows (Settings), submit / approve / decline / re-submit, approvals inbox, notifications | P1 | 006, 007 | Built |
| 009 | [009-home-dashboard](../../specs/009-home-dashboard/spec.md) | Home dashboard: summary cards, my plans, my activities, waiting for my approval, needs attention | P1 | 004–008 | Built |
| 010 | [010-all-work-views](../../specs/010-all-work-views/spec.md) | All work: grid with roll-ups, filters and search, calendar, budget & cost by month, saved views, export | P2 | 004–008 | Built |
| 011 | [011-activity-log-and-recycle-bin](../../specs/011-activity-log-and-recycle-bin/spec.md) | Activity log page and record history; recycle bin page with restore and permanent delete | P2 | 001, 004–007 | Done |
| 012 | [012-bulk-actions-and-import](../../specs/012-bulk-actions-and-import/spec.md) | Select many rows in All work: change owner or a field, submit, copy, move, delete with Undo; the plan workbook — Excel template and export of a plan, import of its skeleton and of updates, with Undo import (RFP §8, D-338–D-341) | P1 | 010, 014 | Built — bulk actions in All work and the plan workbook (template, export, import, Undo import); open: checkboxes in the plan tree (Q-021), Shell to confirm Q-020 |
| 013 | [013-bcpir-and-spa](../../specs/013-bcpir-and-spa/spec.md) | SP&A plan view of a plan or campaign (planned, committed, actual by period and dimension, budget and procurement panels) and the BCPIR marker and filter, re-baselined on Shell's spec 006 (D-333, D-334–D-337) | P1 | 004–008, 010, 015, 032 | Built — SP&A view of a plan or campaign (API + page), BCPIR marker and filter on the Plans list, plan page and All work, Breakdown by BCPIR plan; BCPIR return view, register and export not built (D-337); all gates and the full e2e green |
| 014 | [014-access-control](../../specs/014-access-control/spec.md) | Roles and access: fixed permissions, five default roles, team scopes, view-only vs edit enforced by the API, segregation of duties; global budgets allocated to market plans (BUS.ADM.020, BUS.BUD.040) | P1 | 001–011 | Built — every task done: screens hide or explain actions per role (D-352), e2e `access-control.spec.ts` green |
| 015 | [015-financial-transactions](../../specs/015-financial-transactions/spec.md) | Read-only GSAP purchase orders, invoices, payments and accruals from a simulated GSAP extract, shown on the activity, line item, plan, campaign, initiative, All work, Home and a Financials page (BUS.FIN.080) | P1 | 004–010, 014 | Built — finish open: e2e (T021, T025, T030, T035), Home no-access state (T044), container delete dialogs (T045) |
| 016 | [016-planning-and-currency-extensions](../../specs/016-planning-and-currency-extensions/spec.md) | Dated exchange rates and a simulated GSAP rate import, plan currency, budget by month, split activity, plan type and multi-year views, Shell-attribute filters and Breakdown, field help (BUS.BUD.050, BUS.PLN.020, BUS.PLN.040, BUS.BUD.020, BUS.UX.020) | P1 | 002, 004–007, 010, 014 | Built — attribute filters, Breakdown and Planned cost by (D-350), field-help tips on every record screen (D-351); finish open: e2e (T089, end-to-end lane) |
| 017 | [017-approval-rules](../../specs/017-approval-rules/spec.md) | Approval rules: approver sources (named people, line manager, role), any or all, conditional and automatic levels, backup approvers, delegation, reminders and escalation, Monitoring (BUS.WFL.010) | P1 | 003, 008, 014 | Built — approvals read roles from 014, line manager on People; finish open: e2e (T047–T049, end-to-end lane) |
| 018 | [018-pinned-plans](../../specs/018-pinned-plans/spec.md) | Pinned plans: pin plans to the top of the Plans list with a star, saved per person as the display preference pinnedPlanIds (D-317) | P2 | 004 | Done |
| 019 | [019-alignment-guardrails](../../specs/019-alignment-guardrails/spec.md) | Alignment guardrails: CI with three checks, branch rules, npm release age, ADR-0010 (constitution 3.1.0) | P1 | — | In progress |
| 020 | — | Shell palette: design tokens with Shell's exact colours (constitution X) | P1 | 019 | Planned |
| 021 | — | Comment budget: comments cut to at most 5% of each file (constitution VIII) | P2 | 019 | Dropped: each change brings the files it touches within the budget (constitution 4.2.0) |
| 022 | [022-api-v1](../../specs/022-api-v1/spec.md) | API v1: `/api/v1` paths, one error envelope, 404 for out-of-scope records, `/healthz` and `/readyz` | P1 | 019 | In progress |
| 023 | [023-postgresql-storage](../../specs/023-postgresql-storage/spec.md) | PostgreSQL storage: pgx, sqlc and goose replace the JSON file store; scoped reads in SQL (ADR-0013) | P1 | 022 | In review — the JSON store stayed as the tests' stand-in until 029 deleted it |
| 024 | [024-gin-transport](../../specs/024-gin-transport/spec.md) | Gin transport, middleware and OpenTelemetry | P2 | 022 | In progress |
| 025 | [025-backend-layout](../../specs/025-backend-layout/spec.md) | Backend layout: singular packages, a small `Store` per service, files within 300 lines | P2 | 023, 024 | In review — service tests on fakes and deleting the JSON store done by 029 |
| 026 | [026-web-app-structure](../../specs/026-web-app-structure/spec.md) | Web-app structure: constitution F2 folders, React Hook Form + Zod, MSW, per-route splitting | P2 | 019, 020, 022 | Done |
| 027 | [027-test-hygiene](../../specs/027-test-hygiene/spec.md) | Test hygiene: drop tests that prove nothing, fast and deterministic suites; coverage threshold once SonarQube exists | P3 | 019 | In review — phase 1 merged (#12); phase 2 (every other backend package) in review |
| 028 | [028-query-efficiency](../../specs/028-query-efficiency/spec.md) | Query efficiency: no page or operation sends statements that grow with its records (N+1); reference data read once per transaction; statements per request in the request log; benchmarks at the scale targets' volumes | P1 | 023 | In review |
| 029 | [029-test-fakes](../../specs/029-test-fakes/spec.md) | Test fakes: service, handler and contract tests on hand-written in-memory fakes; the JSON file store deleted | P2 | 023, 025 | In review |
| 030 | [030-simulated-gsap-swbs](../../specs/030-simulated-gsap-swbs/spec.md) | SWBS codes from a simulated GSAP: requests for the line items of approved activities, a simulated GSAP batch that returns one code per line item, the code and its request status on the line item, the HX key shown (D-321, ADR-0016) | P1 | 007, 008, 015 | Built — simulated GSAP SWBS run (API + Financials panel + line-item action), code and HX key on the line item; all gates and e2e green |
| 031 | [031-approval-any-record](../../specs/031-approval-any-record/spec.md) | The approval engine approves any record type: workflow object types (activity, purchase request, modified purchase request), owner and record-field approvers, re-submit from Submitted, self-approval for purchase requests (D-323, D-324) | P1 | 008, 017 | Built — subjects, object types, owner and record-field approvers, Re-submit from Submitted; all gates and e2e green |
| 032 | [032-purchase-requests](../../specs/032-purchase-requests/spec.md) | Purchase requests on MAPP's side: raise from line items, lines and reservation, approval and modification, routing to the requisitioner in MAPP with a PR form, close-out, Purchase requests page (client 013; D-320) | P1 | 030, 031 | Built — PR record, reservation, approval and change approval, in-app routing with PR form, close, page, settings, Requisitioner role; all gates and e2e green; converge items T096–T100 open |
| 033 | [033-simulated-gsap-pr-po](../../specs/033-simulated-gsap-pr-po/spec.md) | GSAP PR number and purchase order created by the simulated GSAP for a routed purchase request, shown on the purchase request and its lines, with Committed figures from the PO lines (D-321) | P1 | 032, 015 | Built — simulated GSAP PR and PO, booked as GSAP transactions, top-up; all gates and the whole PR→PO e2e green |
| 034 | [034-planning-alignment](../../specs/034-planning-alignment/spec.md) | Planning rules aligned with Shell's process: line-item changes keep the activity's approval, Shell's finance lists and zero-based cost hierarchy, plan year current to +10 (D-322, D-325, D-326) | P1 | 006, 007, 002 | Built — line-item changes keep approval, Shell's lists, plan year current to +10; all gates and e2e green |
| 035 | [035-product-name-spa-one](../../specs/035-product-name-spa-one/spec.md) | The product is shown as SP&A One: the coloured name is the logo (blue "&", yellow "One"), a "1" tile where the name does not fit and as the favicon, and every text, message and file name that names the product (D-332; constitution 4.5.0) | P1 | 001, 020 | In review — unit, contract and Go tests green; e2e in CI |
| 036 | [036-planning-dashboards-and-export](../../specs/036-planning-dashboards-and-export/spec.md) | Executive and regional planning dashboards from plan and finance data (budget, planned, committed, actual by region, market, campaign and month; approvals, purchase requests, completeness), and export of plans and dashboards to PowerPoint and PDF (D-333) | P2 | 009, 010, 014, 015, 032 | Built — Executive and Regional dashboards, plan and dashboard export to .pptx and .pdf (ADR-0017); all gates and the full e2e suite green; open: widen the data.export label, Q-022, Q-023 |
| 037 | [037-overspend-control](../../specs/037-overspend-control/spec.md) | Overspend control: each budget owner chooses whether going over a plan, campaign, initiative or global budget warns (default) or is refused, enforced by the API for every save that adds cost; Settings › Overspend control; purchase requests refused while line items exceed the activity's planned cost (D-333, D-346 – D-349) | P2 | 004, 005, 014, 016, 032 | Built — Go, unit, contract and e2e tests green (D-346 – D-349; Q-024, Q-025 open for Shell) |
| 038 | [038-qa-findings](../../specs/038-qa-findings/spec.md) | QA findings: fixes for the 54 findings of exploratory QA on `develop@85a3d495` — purchase-request reservation during a change, financial-transaction scope and the link permission, workbook import at 5,000 rows, dashboard link fallback, delete with Undo, sign-in limit and security headers, approval over planned cost (warn or refuse), money grids that add up, and the medium and low fixes (D-353 – D-357) | P1 | 012, 013, 015, 032, 036, 037 | Built |

## Scope boundaries

| Topic | Owned by | Not in |
|---|---|---|
| Sign-in page, session, actor, sign-out | 001 | — |
| Sidebar, page header, design tokens, shared components, toasts, empty states, environment banner | 001 | other features only reuse |
| Writing activity-log entries and putting deletes in the recycle bin (the mechanism every feature uses) | 001 | 011 adds the screens and restore choices |
| "Getting started" home | 001 | 009 replaces it with the dashboard |
| Models, activity types, Shell attributes and their option lists, currencies and rates, finance budget items | 002 | Approval workflows (008) |
| People, teams, memberships, person picker, My profile (display currency, date format) | 003 | Sign-in itself (001) |
| Plan record, plans list, plan page frame and its hierarchy tree | 004 | Campaign/initiative/activity rows (005, 006) |
| Campaign and initiative records and their rows in the plan tree | 005 | — |
| Activity record and activity page | 006 | Line items (007), approval actions (008) |
| Line items, spreads, unassigned balance, finance budget link, SWBS flag | 007 | Sending to GSAP (future) |
| Approval workflows, approval actions, approvals inbox, notifications bell | 008 | — |
| Home dashboard | 009 | — |
| Cross-plan grid, calendar, budget & cost, saved views, export | 010 | — |
| Activity log page, History tab, recycle bin page | 011 | — |
| Bulk actions, import | 012 | Single-record copy/move (004–006) |
| SP&A view, BCPIR marker and filter | 013 | The BCPIR flag field itself (004); the BCPIR return view (Measure, later) |
| Permissions, roles, scopes, access rules, who may decide an approval; global budgets and allocations | 014 | Approval rules (017) |
| GSAP transactions, their simulated import and matching, financial figures | 015 | Creating POs or invoices (GSAP owns them) |
| Exchange-rate history and rate import, plan currency, budget phasing, split activity, plan type, multi-year views, attribute filters and Breakdown, field help | 016 | A line item's own currency (007) |
| Approval rules, delegation, reminders and escalation, Monitoring | 017 | The check order of who may decide (014, extended by 017) |

## Cross-cutting rules every spec follows

1. **Vocabulary:** Plan → Campaign → Initiative → Activity → Line item, per `docs/product/glossary.md`.
2. **Data and rules** come from `docs/product/domain-model.md` and `docs/product/field-catalogue.md`;
   specs reference them by section instead of copying.
3. **Screens** follow `docs/product/design-system.md`: the standard CRUD pattern, status chips,
   completeness chips, all errors at once, delete to the recycle bin with Undo.
4. **Permissions (014, D-223):** every action needs its permission of 014 Table A, enforced by the API (fail
   closed). Specs 001–013 describe "the user"; 014's Table C gives the rule of each of their operations, and
   specs 015–017 name the permission each action needs.
5. **Every change is logged** with the actor (mechanism from 001).
6. **Out of scope:** Finance budgets module, accrual calculation, invoice approval, supporting documents on
   purchase requests, Measure's performance reporting, live GSAP integration, e-mail and sharing to Outlook or
   Teams, real sign-in (SSO), AI. Purchase requests on
   MAPP's side and a simulated GSAP for SWBS codes, GSAP PRs and POs are in scope since constitution 4.4.0
   (features 030–034); planning dashboards, PowerPoint/PDF export and overspend control since 4.6.0 (036, 037).
   GSAP transactions and daily rates come from simulated feeds
   behind adapters (015, 016; D-242, D-259, D-308). Mention the rest only as future integration points.

## Later (not in this phase)

Finance module (finance budgets, accrual calculation, invoice approval, PR supporting documents), Measure
performance dashboards (revenue, attribution, waterfall, ROI), live GSAP integration (the real SWBS request/response, PR/PO/invoice sync and rate feed behind the
adapters the simulated GSAP uses), Microsoft Entra ID sign-in (roles from Entra),
Azure hosting, templates, snapshots, related plans/activities, multi-year activity linking,
custom goals, latest estimates, media codes, calendar feeds, AI assistant.
