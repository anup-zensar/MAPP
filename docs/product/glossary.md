# Glossary

Plain-English meaning of every term used on MAPP's screens and in its specs. The ⓘ tips in the
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
| **Spread** (Hive9: allocation) | How a line item's planned cost is divided over the months: Evenly, All in first month, All in last month, or Custom amounts. |
| **Local currency** | The currency a line item's cost is entered in. MAPP converts it to USD using the rate in Settings. |
| **Base currency** | USD — the currency all totals are calculated in. |
| **Display currency** | The currency you choose to see amounts in; it does not change stored values. |
| **Finance budget item** | A row of a finance budget (e.g. "2026 Lubricants Budget › Europe – Digital") that a line item is linked to. Managed by Finance; a simple list in Settings for now. |
| **Finance budget** | Finance's budget for a year (e.g. "2026 Lubricants Budget"), made up of finance budget items. Only the list of items exists in this phase. |
| **Actuals** | What was really spent, reported by finance. Coming in a later phase. |

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
| **BCPIR** | Business Case & Post-Investment Review — a flag on a plan; its full use is defined in feature 013. |
| **SP&A** | Shell's spend view of marketing plans (feature 013). The expansion "Sales, Promotion & Advertising" is an assumption still to be confirmed with Shell (Q-001). |
| **Business unit** | Lubricants or Mobility. Decides which extra Shell attributes an activity shows. |
| **Cluster / Region** | Where a campaign runs, e.g. cluster "Retail – China", region "RETAIL – CHINA". |
| **Country / OU** | Countries or operating units an activity covers, with a percentage share each. |
| **Company code** | The Shell legal entity in SAP, e.g. GB01. |
| **Cost centre** | The SAP cost centre that pays, e.g. CN04000220. |
| **Spend type / GL code** | What kind of spend a line item is, and the general-ledger account it posts to. |
| **SWBS code** | An accounting code that only GSAP creates for a line item; needed before purchasing. MAPP only records the request in this phase. |
| **GSAP** | Shell's SAP finance system — the system of record for money. |
| **PIR** | Post-investment review — whether the activity's results will be reviewed. |
| **FIT4** | A Shell programme flag and ID attached to initiatives and activities. |

## Application terms

| Term | Meaning |
|---|---|
| **Details** (Hive9: Inspection Window) | The view of one record, in a side panel or a full page. |
| **Select** (Hive9: Honeycomb) | Tick several rows to act on them together. |
| **Summary cards** (Hive9: HUD) | The row of totals at the top of a page. |
| **All work** | The cross-plan view of every campaign, initiative and activity with filters, calendar and budget views. |
| **Activity log** (Hive9: Plan Log) | The history of who changed what and when. |
| **Recycle bin** | Where deleted records go; they can be restored. |
| **Demo sign-in** | The temporary sign-in used until Shell single sign-on is added. |
| **Actor** | The signed-in person who performs an action; recorded on every change. |
| **Shell attribute** (Hive9: custom attribute) | A Shell-specific field (e.g. Cluster, Country / OU, GL code) defined in Settings rather than built in, so administrators can change its options without a new release. |
| **Inactive / Deactivate** | A Settings record or person that is no longer offered for new choices but is kept, so existing records still show it, labelled "(inactive)". Can be made active again. |
