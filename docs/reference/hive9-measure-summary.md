# Hive9 Measure module — as-is summary

Condensed from the Zensar analysis of Shell's Hive9 sandbox (*Hive9 Measure Module Functional & Technical
Reference*, 11 September 2026, kept outside the repository) and from Part E of the
[functional analysis](hive9-functional-analysis.md), which repeats it; `FA §E…` pointers below lead to the full
detail. It is a **source, not a requirement**: MAPP's requirements are the specs under `specs/` and the product
docs. Hive9's Program and Tactic are written Initiative and Activity. No sandbox data, people or vendors are
reproduced here.

## 1. What Measure is

Measure is Hive9's read-mostly reporting layer. The Plan module is the system of record for plans, campaigns,
initiatives, activities and line items; the Finance module for budgets, transactions, purchase requests and
accruals; Measure reads both and joins them to performance data from outside systems. Every number is (a) rolled
up from Plan and Finance records, (b) loaded from an integrated system, or (c) derived from (a) and (b). Nothing
in Measure is editable except a user's own grid and filter preferences (FA §E1, §E7.7).

Hive9 keeps two stores: a relational store for everything users type, and a star schema of performance facts
(touches, inquiries, MQLs, opportunities, revenue) each tied to one **Activity**. Once a fact is tied to an
activity it inherits every cost and attribute of that activity and its parents — that is how "top ROI by cost
centre" works without the cost centre existing in the CRM (FA §E4.1).

## 2. The screens

| Screen | Use it to… | Notes (FA §) |
|---|---|---|
| **Summary** | answer "are we on track this year?": projected revenue against goal with the variance, four league tables (top by revenue, performance, cost, ROI for a chosen attribute), funnel stage volumes and conversion rates against goal | E3.1 |
| **Revenue** | rank plans, campaigns, initiatives or activities by revenue and drill two levels at a time, with a link back to the record | E3.2 |
| **Waterfall** | the same ranking for one funnel stage (INQ, MQL, Closed Won) | E3.3 |
| **Plan Budget** | the marketing view of money over time: Plan Budget, Allocated Cost, Actual Cost, Actual at monthly FX; only the first two project into the future | E3.4 |
| **Finance Budget** | the finance view of one finance budget: annual, total and year-to-date budget, planned, actual, forecast, burn rate; tabs generated from the budget levels | E3.5 |
| **Approvals Dashboard** | approval status mix, average days to approve, a grid of every approval record across all approvable object types | E3.6 |
| **PR Dashboard** | commitment to vendors by delivery month, counts of purchase requests, lines and vendors, goods-receipt and invoice totals, vendor delivery performance | E3.7 |
| **AR Invoice Details** | period-end reconciliation: accrual request › line › purchase order › invoice, with "planned − actual in period" and a recommended accrual | E3.8 |
| **Plan Details** (custom) | data quality: for each attribute, how activities and planned cost are spread across its values and how many records have it filled in; Planned cost entered, Goal entered, Is integrated | E3.9 |
| **China Budget Reports** (custom) | budget release and occupancy, an activity summary and a 30-column purchase-order ledger joined to the plan | E3.10 |
| Other tenant dashboards | Actuals at monthly FX, PR Track Invoices, Transactions, SWBS Not Received, Pending PR, Accrual Finance, Overlapping Actuals, SP&A Lubes, SP&A Mobility, User Detail Report | F4.4 |

Two user interfaces coexist: the **standard reports** (year checkboxes, one filter panel shared across reports,
Monthly/Quarterly, one page-level Share Report) and the **custom dashboards** (a date range defaulting to the
last six months, filter groups defined per dashboard, View by years/quarters/months/weeks, export per widget).
They share no filter state (FA §E2.4).

## 3. The shared filter panel (standard reports)

Present on Summary, Revenue, Waterfall, Plan Budget and Approvals (FA §E3.0).

| Region | What it is |
|---|---|
| My Saved Views | named presets of filters, columns and column order; personal or shared; a default can be starred |
| Year | multi-select; drives which plans are offered |
| Plan | multi-select with search; "(selected / total)" counter |
| Attributes | four tabs — Plan, Campaign, Initiative, Activity — each with that level's attribute groups, Select all / Clear all, search above ten values |
| Plan view | All plans · Strategic/AOP only · Exclude strategic/AOP |
| Date sets | five date pickers on the activity (start, end, due, build, completion) |
| Owner | one list per level |
| Activity type | with a usage count per value |
| Status | Created, Submitted, Declined, Approved, In-Progress, Complete, all ticked by default |
| Line item | its own section: contracted spend, GL code, spend category, spend type, owner |
| Show hierarchies | Any levels · All levels only · No initiatives · No activities |
| Update | nothing re-queries until pressed |

Rules worth knowing:

- Every nullable attribute shows an extra value **(Not Completed)** meaning "nothing chosen" — the way to find
  records with missing segmentation. It is left out of the "(selected / total)" counter.
- Year and Plan cascade into the other lists; a new plan loads with every attribute selected.
- Filter state is saved per user and **survives logout** — the most common cause of "why is my dashboard
  empty?" support calls.
- Date bounds are an **overlap test**: an activity running November to February appears in both years' reports.
- Attributes with the same label exist at several levels with different value sets (Growth Ambition, Function,
  Focus Area); they are different fields.

## 4. The stack-ranking template (Revenue, Waterfall, Plan Budget)

One template, three flavours (FA §E3.2.5.1): a generated strip of summary cards (per stage and activity category —
70 cards on Plan Budget, with a settings gear, scrolling and drag-to-reorder), a trend chart split at **today**
(actuals left, projection right, goal across) with a data table beneath, a control bar (View by · Group by · Sort
by · Compare · Items per page · Order) and a ranked list that always shows two hierarchy levels: clicking a
lower row promotes it and reveals the level below; an orange breadcrumb opens the record in Plan. Absent metrics
render as "--", not zero. With several years selected the axis mixes yearly and monthly buckets without saying so.

## 5. Where every figure comes from (FA §E4.3)

| Figure | Source | Manual or integrated |
|---|---|---|
| Goal revenue, goal per stage | plan and activity goals; defaults from the activity type's model | manual |
| Actual revenue, actual per stage | CRM and marketing-automation connectors → performance facts | integrated |
| Projected revenue, projected per stage | the plan's model: conversion rates, velocity, average deal size | derived |
| Conversion % and conversion goal | stage ÷ previous stage; planned rate | derived / manual |
| ROI | attributed revenue against actual cost (or planned cost when "Use Planned Cost as Actual Cost" is on) | derived |
| Plan budget, allocated (planned) cost | plan entry, line-item allocation | manual |
| Actual cost, actual at monthly FX | finance transactions at the standard rate; at the rate of the month | integrated / derived |
| Annual budget, budget YTD, forecast, burn rate | finance budgets | manual / derived |
| Average approval days | workflow timestamps | system |
| Goods receipt (GRSP), invoice amount, PR line amount | ERP receipts and invoices; purchase-support entry | integrated / manual |
| Vendor performance % | delivery dates against receipts; empty without receipts | derived |
| Planned − actual in period, recommended accrual | accrual lines | derived |
| Activity and PO occupied amount and rate, released amount | budget release data from SAP | derived |

Enumerations (FA §E4.4): plan-object status Created · Submitted · Declined · Approved · In-Progress · Complete;
approval status adds Closed; purchase-request status adds In-Review; accrual statuses differ between header and
line. Aggregation Monthly · Quarterly; view by Plan · Campaign · Initiative · Activity; order High to Low · Low to
High; page sizes 5–100 (standard) or up to 500 (custom tables); 30 display currencies.

## 6. Models, attribution and projection (FA §E7.5)

- A **model** is attached to a plan and defines the funnel stages (INQ → MQL → Closed Won in this tenant), the
  conversion rate between stages, the **velocity** (time lag between stages), the **average deal size** (turns
  won deals into revenue), which stages a goal may be set against, the integration mapping (origin of *Is
  integrated*) and the **external name** rule (generated when the activity is approved, used to match
  third-party data).
- Projection runs forward from the stage where a goal was set, from the activity's start date to the end of the
  plan year. No goal, or no integrated activity, means every revenue and funnel figure is zero — the state of
  the sandbox (FA §E7.1).
- Nine **attribution models** share revenue credit between touchpoints: first touch, last touch, equally
  distributed, interaction weighted, linear time decay, half-life time decay, U-shaped, W-shaped and a custom
  statistically inferred one. The chosen model drives every ROI figure, and the dashboards do not name it.
- Multi-select attributes split cost and performance credit between values (50/50 by default, editable
  separately, each summing to 100%).
- Join keys between facts and the plan: activity ID, media code, integration ID, external name; WBS code and cost
  centre on the SAP side.

## 7. Access (FA §E6, §F4.4)

A user has one role and many teams. The role's **Measure tab** says which reports and dashboards the role may
open — two users of one tenant see different Measure menus — and a separate switch hides **financial
information** entirely, so two users on the same dashboard can see different tiles and columns, not only
different rows. Teams decide which plans and budgets appear. Custom dashboards are provisioned per tenant by the
vendor's customer-success team; each has separate view and export rights. Shared saved views share the
configuration, never the data.

## 8. Faults not to repeat (FA §E8.2)

| Hive9 behaviour | What MAPP does instead |
|---|---|
| Filters persist across sessions and reports; dashboards look empty weeks later | Filters live in the page address; nothing persists silently (D-368) |
| "No data available" does not say whether data, configuration or permission is missing | Every empty state says why |
| The time axis mixes yearly and monthly buckets unlabelled | One granularity per chart; plans are one year |
| ROI shown without its cost basis or attribution model | Both named next to every ROI figure (050) |
| Over-commitment rates above 100% shown without warning | Red words, as "Committed above planned cost" |
| Status vocabularies differ between related objects | MAPP's domain model statuses, one list per record type |
| Overlap date semantics double-count multi-year activities | Plan-year semantics; multi-year linking is out of scope |
| Two filter paradigms | One report frame (039) |
| Custom dashboards changed only by the vendor | Reports generated from Settings (043); a builder stays out |

## 9. Hive9 controls in SP&A One words

Measure → Reports · HUD → Summary cards · (Not Completed) → Not filled in · Monthly/Quarterly → Period ·
My Saved Views → Saved views · View By / Group By / Sort By → Show / Group by / Sort by · Tactic Category →
Activity category · Tactic Type → Activity type · Waterfall → Funnel stages · Share Report → not offered (the
address is the link) · orange breadcrumb → the record link on every row.
