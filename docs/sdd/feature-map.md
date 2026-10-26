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
| 012 | [012-bulk-actions-and-import](../../specs/012-bulk-actions-and-import/spec.md) | Select many rows: change owner/status fields, copy, move, delete; import activities from a template | P3 | 010 | Draft |
| 013 | [013-bcpir-and-spa](../../specs/013-bcpir-and-spa/spec.md) | Shell's BCPIR and SP&A plan views (meaning to be clarified with Shell) | P3 | 004–008 | Draft |
| 014 | [014-access-control](../../specs/014-access-control/spec.md) | Roles and access: fixed permissions, five default roles, team scopes, view-only vs edit enforced by the API, segregation of duties; global budgets allocated to market plans (BUS.ADM.020, BUS.BUD.040) | P1 | 001–011 | Built — finish open: hide or explain actions per role (T043–T051), e2e (T061) |
| 015 | [015-financial-transactions](../../specs/015-financial-transactions/spec.md) | Read-only GSAP purchase orders, invoices, payments and accruals from a simulated GSAP extract, shown on the activity, line item, plan, campaign, initiative, All work, Home and a Financials page (BUS.FIN.080) | P1 | 004–010, 014 | Built — finish open: e2e (T021, T025, T030, T035), Home no-access state (T044), container delete dialogs (T045) |
| 016 | [016-planning-and-currency-extensions](../../specs/016-planning-and-currency-extensions/spec.md) | Dated exchange rates and a simulated GSAP rate import, plan currency, budget by month, split activity, plan type and multi-year views, Shell-attribute filters and Breakdown, field help (BUS.BUD.050, BUS.PLN.020, BUS.PLN.040, BUS.BUD.020, BUS.UX.020) | P1 | 002, 004–007, 010, 014 | Built — finish open: attribute filters and Breakdown screens (T074–T077), field-help tips (T084–T088), Import GSAP rates route (T028), e2e (T089) |
| 017 | [017-approval-rules](../../specs/017-approval-rules/spec.md) | Approval rules: approver sources (named people, line manager, role), any or all, conditional and automatic levels, backup approvers, delegation, reminders and escalation, Monitoring (BUS.WFL.010) | P1 | 003, 008, 014 | Built — finish open: approvals read roles from 014 (T001, T008), line manager on People (T036), e2e (T047–T049) |
| 018 | [018-pinned-plans](../../specs/018-pinned-plans/spec.md) | Pinned plans: pin plans to the top of the Plans list with a star, saved per person as the display preference pinnedPlanIds (D-317) | P2 | 004 | Done |
| 019 | [019-alignment-guardrails](../../specs/019-alignment-guardrails/spec.md) | Alignment guardrails: CI with four checks, branch rules, npm release age, ADR-0010 (constitution 3.1.0) | P1 | — | In progress |
| 020 | — | Shell palette: design tokens with Shell's exact colours (constitution X) | P1 | 019 | Planned |
| 021 | — | Comment budget: comments cut to at most 5% of each file (constitution VIII) | P2 | 019 | Planned |
| 022 | — | API v1: `/api/v1` paths, one error envelope, 404 for out-of-scope records, `/healthz` and `/readyz` | P1 | 019 | Planned |
| 023 | — | PostgreSQL storage: pgx, sqlc and goose replace the JSON file store | P1 | 022 | Planned |
| 024 | — | Gin transport, middleware and OpenTelemetry | P2 | 022 | Planned |
| 025 | — | Backend layout: singular packages, `internal/platform/database`, files within 300 lines | P2 | 023, 024 | Planned |
| 026 | — | Web-app structure: constitution F2 folders, React Hook Form + Zod, MSW, per-route splitting | P2 | 019 | Planned |
| 027 | — | Test hygiene: drop tests that prove nothing; coverage threshold once SonarQube exists | P3 | 019 | Planned |

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
| BCPIR / SP&A views | 013 | The BCPIR flag field itself (004) |
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
6. **Out of scope:** Finance budgets module, purchase requests, accrual calculation, Measure, live GSAP
   integration, e-mail, real sign-in (SSO), AI. GSAP transactions and daily rates come from simulated feeds
   behind adapters (015, 016; D-242, D-259, D-308). Mention the rest only as future integration points.

## Later (not in this phase)

Finance module (finance budgets, purchase requests, accrual calculation), Measure dashboards, live GSAP
integration (SWBS request/response, PR/PO/invoice sync, rate feed), Microsoft Entra ID sign-in (roles from Entra),
PostgreSQL and Azure hosting, templates, snapshots, related plans/activities, multi-year activity linking,
custom goals, latest estimates, media codes, calendar feeds, AI assistant.
