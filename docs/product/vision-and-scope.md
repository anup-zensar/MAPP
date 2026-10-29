# Vision and Scope

## Vision

**MAPP is the simplest way for Shell marketers to plan their year:** build a plan, break it into
campaigns, initiatives and activities, plan every cost as line items, get activities approved — and always
see the status of everything and what is missing.

It replaces Hive9's Plan module with the same business process and data, in a modern, friendly design
(Shell's design language, design system v2) that a first-time user understands without training.

## Why

- Shell users describe Hive9 as complex and not user-friendly: icon-only toolbars, hidden modes, one
  validation error per save, and a Business Unit field that could block plan creation entirely.
- Shell's PRISM Phase 1 findings (May 2026) show users plan in Excel and only *submit* in Hive9, and that
  trust breaks between planning and finance. A planning tool people actually want to use, with clean,
  standardised data, is the first step to fixing that.
- Hive9 must be replaced while GSAP integrations are frozen by the Nucleus programme (until about March
  2027), so the Plan module comes first and finance integration later.

## Users (this phase)

| Persona | Main jobs in MAPP |
|---|---|
| **Marketer** | Create and maintain plans, campaigns, initiatives, activities and line items; submit activities. |
| **Marketing team leader / approver** | Review and approve or decline activities; watch plan progress. |
| **Marketing administrator** | Maintain Settings: models, activity types, Shell attributes and option lists, currencies, finance budget items, approval workflows, people and teams. |
| **Finance advisor** | Look at planned cost, budget use and line-item finance coding. |

Shell's 19 Hive9 roles (Marketer, Business Champion, Marketing Team Leader, Client Admin, Local Finance
Manager, TA, …) are mapped to real permissions when role-based access arrives; in this phase every user
can do everything.

## In scope now — the Plan module

| Area | What users can do |
|---|---|
| Sign-in and shell | Demo sign-in page; layout in Shell's design language (design system v2), navigation, user menu, light/dark mode |
| Home | A dashboard of their plans, activities, approvals and what needs attention |
| Settings | Manage models, activity types, Shell attributes and options, currencies and rates, finance budget items, approval workflows |
| People and teams | Manage demo people and teams; plans use teams and owners |
| Plans | Create, view, edit, copy, publish, delete and restore plans with budget, goals and Shell attributes |
| Campaigns and initiatives | Create, view, edit, move, copy, delete; budgets flow down, costs roll up |
| Activities | Create, view, edit, move, copy, delete; Shell attributes with business-unit groups and % splits; goals; status and completeness |
| Line items | Plan each cost with currency, spread over months, finance coding, finance budget link and SWBS request flag; unassigned balance |
| Approvals | Submit, approve, decline and re-submit activities through named approval workflows; approvals inbox; notifications |
| All work views | Grid with roll-ups and filters, calendar, budget & cost by month, saved views, export |
| History and safety | Activity log for every record; recycle bin with restore |
| Financials (feature 015) | See GSAP's purchase orders, invoices, payments and accruals — read-only, from a simulated GSAP extract — as Committed, Invoiced, Paid and Accrued on every level, on the Financials page, in All work and on Home; import the extract; link a transaction GSAP could not match |
| Bulk and import (later in phase) | Select many rows to change owner/status, copy or move; import activities from a template |
| BCPIR & SP&A (to be clarified) | Shell's BCPIR and SP&A plan views |

## Out of scope now

- Finance module: finance budgets (beyond a simple list of budget items), purchase requests, accrual
  requests and calculation. GSAP transactions are only read (feature 015).
- Measure reporting dashboards.
- GSAP integration (SWBS requests are only recorded), e-mail, Microsoft Entra ID sign-in, role-based
  access, Azure hosting, PostgreSQL (all planned).
- An AI assistant, optimizer, attribution, Google Analytics, Kanban board, Trello/Jira and Google
  Calendar integrations (possible later, per Shell's PRISM guidance: foundations before AI).

## Success for this phase

1. A first-time marketer can build a complete plan — plan, campaign, initiative, activity, line item —
   and submit it for approval in under 15 minutes without help.
2. Every record always shows its status and what is missing.
3. The demo runs end to end on a laptop with the seed data.
4. The codebase is small and readable enough that a new developer can add a field to a level in a day.
