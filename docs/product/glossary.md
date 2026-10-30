# Glossary

Plain-English meaning of every term used on SP&A One's screens (the project MAPP) and in its specs. The ⓘ tips in the
application use these definitions. Hive9 names are given in brackets where users may know them.

## Planning levels

| Term | Meaning |
|---|---|
| **Plan** | One fiscal year of marketing for one business, e.g. "FY2026 Lubricants Europe". Holds the year, model, teams, owner, budget and goal. Plan names are unique across MAPP. |
| **Campaign** | A theme or message that runs for most of the year inside a plan, e.g. "Helix launch". |
| **Initiative** (Hive9: Program) | A group of similar work inside a campaign, e.g. "Digital & social". |
| **Activity** (Hive9: Tactic) | One real piece of marketing work — a film, an event, an e-mail programme. It has a planned cost and is the record that goes through approval. |
| **Line item** | One cost inside an activity, e.g. "Agency fee". It carries the finance coding (spend type, GL code, finance budget item) and is the only level that will later talk to GSAP. |

## Money

| Term | Meaning |
|---|---|
| **Plan budget** | Marketing's own budget, typed on plans, campaigns and initiatives. It flows down: each child shows how much of its parent's budget remains. |
| **Remaining budget** | A parent's budget minus the budgets of its children. It can be negative ("over-allocated"). |
| **Planned cost** | What you intend to spend. Typed on activities and line items; added up automatically on initiatives, campaigns and plans. |
| **Budget used** | Planned cost as a percentage of the budget. |
| **Left to plan** | Budget minus planned cost — how much of a budget is not yet planned. Negative means planned cost is over the budget. |
| **Unassigned planned cost** (Sys_Gen_Balance) | The part of an activity's planned cost not yet split into line items. Aim for zero. |
| **Approval over planned cost** | Approving an activity, or a purchase request drawing on it, while its line items add up to more than its planned cost. Settings › Overspend control decides whether the approver is warned (default) or approval is refused (feature 038). |
| **Spread** (Hive9: allocation) | How a line item's planned cost is divided over the months: Evenly, All in first month, All in last month, or Custom amounts. |
| **Local currency** | The currency a line item's cost is entered in. MAPP converts it to USD using the rate in Settings. |
| **Base currency** | USD — the currency all totals are calculated in. |
| **Display currency** | The currency you choose to see amounts in; it does not change stored values. |
| **Finance budget item** | A row of a finance budget (e.g. "2026 Lubricants Budget › Europe – Digital") that a line item is linked to. Managed by Finance; a simple list in Settings for now. |
| **Plan currency** | The currency a plan's budgets and planned costs are typed in (feature 016); USD unless chosen otherwise, fixed once the plan has amounts. |
| **Current rate** | The latest exchange rate of a currency dated today or earlier; every conversion uses it. |
| **Rate history** | Every dated rate of a currency, newest first; a rate dated in the future changes nothing before its date. |
| **GSAP rate import** | Adding the rates of a GSAP daily-rate file (simulated in this phase) after a review of what will be added, skipped or refused. |
| **Budget by month** (phased budget) | A budget divided over the months of its year — evenly or by hand — so planned cost can be compared with it per period. |
| **Global budget** | Money for a year owned centrally and shared out to market plans by allocations (feature 014). |
| **Allocation** | The part of a global budget given to one plan; it becomes that plan's budget, which the plan's owner cannot change. |
| **Finance budget** | Finance's budget for a year (e.g. "2026 Lubricants Budget"), made up of finance budget items. Only the list of items exists in this phase. |
| **Actuals** | What was really spent: the payments GSAP has released (feature 015's **Paid**). The SP&A view calls it **Actual spend** (feature 013). |
| **Committed** | The value of the GSAP purchase orders on a record's line items (feature 015). **Not yet committed** is planned cost minus committed; "Committed above planned cost" is said in red words. The SP&A view and the dashboards count only the purchase orders posted in the plan's year (D-335, D-358). |
| **Invoiced** | The supplier invoices GSAP has received against a record's line items, less credit notes (feature 015). |
| **Paid** | The payments GSAP has released on a record's line items — Shell's **actuals** (feature 015). |
| **Accrued** | Money owed for work received but not yet invoiced, as GSAP accrued it for the latest accrual month in the data, e.g. "Accrued (Sep 2026)" (feature 015). |
| **Actual (paid)** | What GSAP reports as paid on a record's line items — Shell's actuals (features 015, 036). The dashboards, like the SP&A view's Actual spend, count only the payments posted in the plan's year (D-357). |
| **Unspent budget** | Budget minus actual: how much of a budget is not yet paid out (feature 036). |
| **Budget spent** | Actual as a percentage of the budget (feature 036; Hive9's burn rate). |
| **Dashboard** | A read-only page that adds up the plans you may see — Executive for every market, Regional for one (feature 036). |

## Goals

| Term | Meaning |
|---|---|
| **INQ** | Inquiries — the number of people who show interest. |
| **MQL** | Marketing-qualified leads — inquiries good enough for sales to follow up. |
| **Revenue** | Expected sales value from the deals the leads produce. |
| **Goal stage** (Hive9: Waterfall Goal) | The stage at which a plan's goal is set (INQ, MQL or Revenue). |
| **Model** | The rulebook a plan uses: conversion rates from INQ to MQL and from MQL to deals, the average deal size, and which activity types are allowed. |
| **Set later stages myself** (Hive9: Override model) | Type MQL and revenue yourself instead of letting the model calculate them. |

## Statuses and workflow

| Term | Meaning |
|---|---|
| **Draft / Published** | A plan being prepared / the agreed plan of record. |
| **Created** | An activity being planned, not yet sent for approval. |
| **Submitted** | An activity waiting for its approvers. |
| **Declined** | An activity sent back with a comment; fix and submit again. |
| **Approved** | An activity that has been agreed. It then shows **In progress** from its start date and **Complete** after its end date. |
| **Approval workflow** | The named list of approval levels and approvers an activity goes through, e.g. "China BC Approval Workflow". |
| **Approvals inbox** | The list of activities waiting for your decision. |
| **Completeness** | Whether every field needed for the next step is filled. "3 fields needed" links to them. |
| **Needed to publish / Needed to submit** | Fields you may leave empty while drafting, but must fill before publishing the plan / submitting the activity. |
| **Ongoing** (Hive9: Evergreen) | An activity with no end date. Cannot be undone once saved. |

## Shell business terms

| Term | Meaning |
|---|---|
| **AOP** | Annual operating plan. A plan marked "Strategic / AOP" is a high-level strategic plan, shown with an AOP tag. |
| **SMP** | Strategic Marketing Plan — a Shell flag on Mobility plans. |
| **BCPIR** | Business Case & Post-Investment Review — a yes/no on a plan ("BCPIR plan"); a BCPIR plan carries a BCPIR marker and can be filtered on the Plans list and in All work (feature 013, D-334). The BCPIR return view waits for performance data. |
| **SP&A** | Sales, Promotion & Advertising: the **SP&A view** of a plan or campaign shows planned cost against committed and actual spend, by period, by dimension and by activity, with the budget and the purchase requests (feature 013, Shell spec 006). |
| **Business unit** | Lubricants or Mobility. Decides which extra Shell attributes an activity shows. |
| **Cluster / Region** | Where a campaign runs, e.g. cluster "Retail – China", region "RETAIL – CHINA". |
| **Country / OU** | Countries or operating units an activity covers, with a percentage share each. |
| **Company code** | The Shell legal entity in SAP, e.g. GB01. |
| **Cost centre** | The SAP cost centre that pays, e.g. CN04000220. |
| **Spend type / GL code** | What kind of spend a line item is, and the general-ledger account it posts to. |
| **SWBS code** | An accounting code that only GSAP creates for a line item; needed before purchasing. The line item stores the code GSAP returns; until live integration it comes from the simulated GSAP. |
| **HX key** | "HX" followed by the line item's number (L-1006 → HX1006): the tracking number requisitioners type into GSAP, on which GSAP's PR, PO and invoice data is matched back to the line item. |
| **Simulated GSAP** | The stand-in for GSAP until live integration (ADR-0016): it answers SWBS requests with `SIM-…` codes and, later, creates GSAP PR numbers and purchase orders. Everything it produces is labelled "Simulated GSAP". |
| **GSAP** | Shell's SAP finance system — the system of record for money. |
| **Purchase request (PR)** | A request to buy from one vendor, raised in MAPP on line items of one plan (feature 032). Each **PR line** draws on one line item. Statuses: Created · Submitted · In review · Approved · Declined · Closed. |
| **Available to request** | What a line item can still be asked for: its planned cost less the amounts of the purchase requests drawing on it, in the line item's currency at today's rate. |
| **MoA approver** | The person a purchase request names to approve it under Shell's Manual of Authorities; the seeded purchase-request workflows send it to this person. MAPP does not check MoA limits. |
| **Requisitioner** | The person who keys an approved purchase request into GSAP. The demo role *Requisitioner* opens the Purchase requests page and runs the simulated GSAP. |
| **Routing / Requisitioner queue** | Sending an approved purchase request — Original, Modification, Resend or Close — to the recipients of its Email group and PR recipients, or to the Requisitioner queue when it names none. **Waiting for GSAP** lists the routed purchase requests GSAP has not answered. |
| **PR form** | The spreadsheet or CSV file a routing produces: one row per PR line with what GSAP needs. Its file name holds the purchase request's ID. |
| **Purchase order (PO)** | GSAP's order to a vendor; each PO line is a transaction that counts as Committed on the line item it links to (feature 015). |
| **Invoice** | A supplier's bill GSAP has received, usually against a PO line; a credit note is a negative invoice. |
| **Accrual** | GSAP's month-end entry for work received but not yet invoiced; MAPP shows accruals and does not calculate them. |
| **GSAP transaction** | One PO line, invoice, payment or accrual as GSAP sent it, read-only in MAPP; listed on the Financials page (feature 015). |
| **GSAP extract** | The CSV file of GSAP transactions imported on the Financials page, checked before anything is saved; simulated until live integration, with a sample file to download (feature 015). |
| **Not linked** | A GSAP transaction no rule could match to one line item (by line-item ID, HX key, PO number or SWBS code), shown with the reason; a planner may link it by hand. |
| **GSAP PR number / PO line** | GSAP's number for a purchase request and the purchase-order line each PR line became. Until live integration the simulated GSAP gives `SIM-0010…` and `SIM-4500…` numbers with PO lines 10, 20, … (feature 033). |
| **PIR** | Post-investment review — whether the activity's results will be reviewed. |
| **FIT4** | A Shell programme flag and ID attached to initiatives and activities. |

## Application terms

| Term | Meaning |
|---|---|
| **SP&A One** | The product's name on every screen, in messages and in downloaded file names (feature 035). MAPP is the project's name in code and documents. Changes the product makes by itself are shown as made by "SP&A One". |
| **Details** (Hive9: Inspection Window) | The view of one record, in a side panel or a full page. |
| **Select** (Hive9: Honeycomb) | Tick several rows to act on them together. |
| **Plan workbook** | The Excel layout of one plan (feature 012): sheets Campaigns, Initiatives, Activities and Line items, then Allowed values and Instructions. Downloaded empty (*Download template*) or with the plan's records (*Export to Excel*); a row without an ID creates a record, a row with an ID updates it (*Import from Excel*). |
| **Summary cards** (Hive9: HUD) | The row of totals at the top of a page. |
| **All work** | The cross-plan view of every campaign, initiative and activity with filters, calendar and budget views. |
| **Activity log** (Hive9: Plan Log) | The history of who changed what and when. |
| **Recycle bin** | Where deleted records go; they can be restored. |
| **Demo sign-in** | The temporary sign-in used until Shell single sign-on is added. |
| **Actor** | The signed-in person who performs an action; recorded on every change. |
| **Shell attribute** (Hive9: custom attribute) | A Shell-specific field (e.g. Cluster, Country / OU, GL code) defined in Settings rather than built in, so administrators can change its options without a new release. |
| **Inactive / Deactivate** | A Settings record or person that is no longer offered for new choices but is kept, so existing records still show it, labelled "(inactive)". Can be made active again. |
| **Permission** | One thing a person may do, such as "Edit plan details" or "Export from All work" (the fixed list of feature 014). |
| **Role** | A named set of permissions given to people, such as Planner or Viewer; administrators change roles in Settings › Roles and access. |
| **Scope** | Where a role applies: All markets, My teams or Chosen teams. |
| **Market** | A team, as roles and global budgets see it (e.g. Lubricants Europe). |
| **View only** | The note on a page where the person may see but change nothing; it names their roles and whom to ask. |
| **Split activity** | Dividing one activity into two inside the same plan, moving chosen line items or an amount to the new one (feature 016). |
| **Plan type** | Operational, Strategic or Cross-market; plans stay one year, and views can cover up to five years. |
| **Breakdown** | The All work view and plan card that add up planned cost by a Shell attribute or another grouping, optionally "then by" a second. |
| **Field help** | The ⓘ text of a built-in field, kept in Settings › Field help so administrators can change it without a release. |

## Reports (the Measure module, features 039–050)

| Term | Meaning |
|---|---|
| **Reports** (Hive9: Measure) | The read-only reporting area. Every figure is worked out by the server from plan, finance and performance data; a report changes nothing. |
| **Report frame** | What every report shares: the filter panel, the period and display-currency controls, saved views, the "as of" line, and a table view and export for every widget. |
| **Filter panel** | Year, plans, Shell attributes per level, owner, activity type, status and team. Filters live in the page address, so Back keeps them and a link shares them; nothing persists silently. |
| **Not filled in** (Hive9: (Not Completed)) | The value in a filter and a chart that counts records where an attribute has no value. |
| **Period** (Hive9: Monthly/Quarterly) | The buckets of a time chart: Months or Quarters of the plan year. |
| **Today marker** | The line on a time chart left of which figures are actual and right of which they are planned or projected. |
| **Saved view** (report) | A named set of filters a person saves for a report; one can be starred as the default. Saved views are personal; the address is how a view is shared. |
| **Summary cards** (Hive9: HUD) | The strip of figure cards at the top of a report. |
| **Ranked list** (Hive9: stack ranking) | A list of plans, campaigns, initiatives or activities sorted by a figure, with group by, page size and order; opening a row shows the level below with the row on top, and every row links to its record. |
| **Activity category** (Hive9: Tactic Category) | A named group of activity types the reports use to split goals and ROI (feature 047). |
| **Funnel stages** (Hive9: Waterfall) | INQ → MQL → Revenue, as the model's goal waterfall (domain model §4.5). |
| **Performance data** | Facts from outside systems — inquiries, MQLs, deals, revenue, web metrics — matched to activities (feature 046). In this phase they come from a simulated feed behind an adapter. |
| **Data source** | A named feed of performance data with its column mappings, kept in Settings › Performance data. |
| **Media code** | A code generated per activity and placed in campaign links so web data can be matched back to the activity. |
| **External name** · **Integration ID** | The keys that match an activity with outside systems; the external name is set when the activity is approved. |
| **Is integrated** | Whether an activity has a data source matched to it. |
| **Projection** | Projected stage volumes and revenue from an activity's goal through the model's conversion rates, placed in time by its **velocity** (months between stages), up to the end of the plan year (feature 047). |
| **Attribution model** | The rule that shares revenue credit between the activities that touched a deal (first touch, last touch, equally distributed, linear time decay, U-shaped). The chosen model is named next to every credited figure (feature 050). |
| **ROI** | (Attributed revenue − cost) ÷ cost as a whole per cent, with its cost basis named. |
| **GSAP link reports** | SWBS codes requested and not received, purchase requests waiting for GSAP, and the purchase orders per line item (feature 044). |
