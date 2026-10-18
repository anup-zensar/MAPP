# Feature Map — NorthStar (Plan module phase)

Every feature of this phase, one Spec Kit folder each under `specs/`. **Numbers are the build order.**
The map sets scope boundaries so no requirement is specified twice and none is missed.

**Status values:** Draft (spec written, waiting for product-owner review) · Clarifying · Planned ·
In progress · Done.

| # | Folder | Feature | Priority | Depends on | Status |
|---|---|---|---|---|---|
| 001 | [001-app-shell-and-sign-in](../../specs/001-app-shell-and-sign-in/spec.md) | Demo sign-in, application shell, navigation, user menu, getting-started home, shared building blocks (activity-log and recycle-bin core) | P1 | — | In progress |
| 002 | [002-settings-and-shell-attributes](../../specs/002-settings-and-shell-attributes/spec.md) | Settings: models, activity types, Shell attributes and options, currencies and rates, finance budget items | P1 | 001 | Draft |
| 003 | [003-people-and-teams](../../specs/003-people-and-teams/spec.md) | People directory, teams and memberships, person picker, My profile | P1 | 001 | Draft |
| 004 | [004-plans](../../specs/004-plans/spec.md) | Plans list and plan page; create, view, edit, copy, publish/unpublish, delete plans; budget, goals, Shell attributes | P1 | 002, 003 | Draft |
| 005 | [005-campaigns-and-initiatives](../../specs/005-campaigns-and-initiatives/spec.md) | Campaigns and initiatives inside a plan; budgets flow down, costs roll up; move, copy, delete | P1 | 004 | Draft |
| 006 | [006-activities](../../specs/006-activities/spec.md) | Activities: basics, dates, planned cost, Shell attributes with business-unit groups and % splits, goals, status, completeness | P1 | 005 | Draft |
| 007 | [007-line-items](../../specs/007-line-items/spec.md) | Line items: currency, spread over months, finance coding, finance budget link, SWBS request flag, unassigned balance | P1 | 006 | Draft |
| 008 | [008-activity-approvals](../../specs/008-activity-approvals/spec.md) | Approval workflows (Settings), submit / approve / decline / re-submit, approvals inbox, notifications | P1 | 006, 007 | Draft |
| 009 | [009-home-dashboard](../../specs/009-home-dashboard/spec.md) | ZenReach-style home: summary cards, my plans, my activities, waiting for my approval, needs attention | P1 | 004–008 | Draft |
| 010 | [010-all-work-views](../../specs/010-all-work-views/spec.md) | All work: grid with roll-ups, filters and search, calendar, budget & cost by month, saved views, export | P2 | 004–008 | Draft |
| 011 | [011-activity-log-and-recycle-bin](../../specs/011-activity-log-and-recycle-bin/spec.md) | Activity log page and record history; recycle bin page with restore and permanent delete | P2 | 001, 004–007 | Draft |
| 012 | [012-bulk-actions-and-import](../../specs/012-bulk-actions-and-import/spec.md) | Select many rows: change owner/status fields, copy, move, delete; import activities from a template | P3 | 010 | Draft |
| 013 | [013-bcpir-and-spa](../../specs/013-bcpir-and-spa/spec.md) | Shell's BCPIR and SP&A plan views (meaning to be clarified with Shell) | P3 | 004–008 | Draft |

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

## Cross-cutting rules every spec follows

1. **Vocabulary:** Plan → Campaign → Initiative → Activity → Line item, per `docs/product/glossary.md`.
2. **Data and rules** come from `docs/product/domain-model.md` and `docs/product/field-catalogue.md`;
   specs reference them by section instead of copying.
3. **Screens** follow `docs/product/design-system.md`: the standard CRUD pattern, status chips,
   completeness chips, all errors at once, delete to the recycle bin with Undo.
4. **No permissions in this phase:** every signed-in user can do everything; specs describe "the user".
   Where Shell has a permission rule, the spec states the rule and notes it is enforced when roles arrive.
5. **Every change is logged** with the actor (mechanism from 001).
6. **Out of scope:** Finance, Measure, GSAP integration, e-mail, real sign-in, AI. Mention only as future
   integration points.

## Later (not in this phase)

Finance module (finance budgets, transactions, purchase requests, accruals), Measure dashboards, GSAP
integration (SWBS request/response, PR/PO/invoice sync), Microsoft Entra ID sign-in and role-based access,
PostgreSQL and Azure hosting, templates, snapshots, related plans/activities, multi-year activity linking,
custom goals, latest estimates, media codes, calendar feeds, AI assistant.
