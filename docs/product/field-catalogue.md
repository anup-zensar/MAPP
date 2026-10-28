# Field Catalogue — Plan Module

Every field MAPP shows on each planning level, with its on-screen label, the Shell/Hive9 name it
replaces, its type and its marking. Rules and calculations are in [domain-model.md](domain-model.md).

**Marking legend** (domain model §3):

| Mark | Meaning |
|---|---|
| **R** | Required — blocks saving |
| **P** | Needed to publish — blocks publishing the plan |
| **S** | Needed to submit — blocks submitting the activity |
| — | Optional |
| calc | Calculated by MAPP, read-only |
| later | Filled by Finance/GSAP in a future phase; shown read-only as "—" |

**Type legend:** text · long text · number (whole) · money (exact decimal + currency) · percent · date ·
yes/no · single choice · multiple choice · multiple choice with % split · person · people · reference
(another record or a Settings list).

*Shell attribute* means the field is a **custom attribute** configured in Settings (feature 002), not a
fixed field: its label, options, marking and business-unit condition can be changed by an administrator
without a release. The seed configuration below reproduces Shell's as-is form (28 Sep 2026). Option
values in the seed data are **fictitious samples** until Shell provides the real lists, except the lists
Shell's Hive9 tenant shows live (functional analysis E3.0.2): the zero-based cost hierarchy, spend type, GL
code, spend category and contracted spend carry Shell's own values (D-325).

---

## 1. Plan

| # | Label | Shell / Hive9 name | Type | Mark | Default / options | Notes |
|---|---|---|---|---|---|---|
| 1 | Plan name | Name | text (≤ 120) | R | — | Unique across MAPP |
| 2 | Year | Year | single choice | R | Current year; options current year … + 10, no past year (D-326) | Locked after first save; a plan keeps its year after it has passed |
| 3 | Model | Model | reference (Model) | R | Only active models | Drives activity types and goals |
| 4 | Teams | Team | reference (Teams, many) | P | The signed-in user's teams | |
| 5 | Owner | Owner | person | P | The signed-in user if eligible | Must belong to one of the plan's teams (R1) |
| 6 | Strategic / AOP plan | Set plan as Strategic/AOP | yes/no | — | No | Shown with an "AOP" tag in lists |
| 7 | Description | Description | long text (≤ 2,000) | — | — | |
| 8 | Plan budget | Budget › Plan budget | money (USD) | — | — | Top of "budget flows down" |
| 9 | Default finance budget | Default Finance Budget | reference (finance budget, plan year) | — | — | Default for line-item links |
| 10 | Default line-item spread | Default Line-Item Allocation | single choice | — | Evenly | Evenly · All in first month · All in last month · Custom |
| 11 | BCPIR plan | BCPIR Plan? | yes/no | — | No | Marks a BCPIR plan: marker and filter on the Plans list and in All work (013, D-334) |
| 12 | Business unit | Business Unit (Mobility/Lubricants) | Shell attribute, multiple choice | P | — | Mobility · Lubricants. Controls which attribute groups appear below (R9) |
| 13 | Strategic Marketing Plan (SMP) | Strategic Marketing Plan? | Shell attribute, yes/no | — | No | Shown only when business unit includes Mobility |
| 14 | Goal stage | Waterfall Goal | single choice | P | INQ | INQ · MQL · Revenue |
| 15 | Goal amount | Goal Amount | number / money | — | — | Other stages calculated from the model |
| 16 | MQL, Revenue (goal) | MQL, Revenue | calc | calc | — | Domain model §4.5 |
| 17 | Planned cost | (roll-up) | money | calc | — | Sum of campaigns |
| 18 | Status | Status | Draft · Published | calc | Draft | Domain model §2.1 |

Dropped Shell fields: **ShellAllPlans** (greyed out and unused in Shell's form — decision D-018);
**Related plans** and **Custom goals** (future, see feature map).

## 2. Campaign

| # | Label | Shell / Hive9 name | Type | Mark | Default / options | Notes |
|---|---|---|---|---|---|---|
| 1 | Campaign name | Name | text (≤ 120) | R | — | Unique in its plan |
| 2 | Owner | Owner | person | P | Plan owner | |
| 3 | Start date / End date | Start date, End date | date range | P | Plan year 1 Jan – 31 Dec | Warning outside plan year (R4) |
| 4 | Description | Description | long text | — | — | |
| 5 | Campaign budget | Budget › Campaign Budget | money (USD) | — | — | |
| 6 | Remaining plan budget | Remaining Plan Budget | money | calc | — | May be negative (shown red) |
| 7 | Planned cost | Planned Cost | money | calc | — | Sum of initiatives |
| 8 | Cluster | Cluster | Shell attribute, multiple choice | P | — | e.g. Retail-China, Lubricants-Europe |
| 9 | Region | Region | Shell attribute, multiple choice, depends on Cluster | P | Auto-filled when a cluster has one region | R8 |
| 10 | Insight session completed | Insight Session Completed | Shell attribute, yes/no | — | No | From Shell's in-scope list |
| 11 | Workshop completed | Workshop Completed | Shell attribute, yes/no | — | No | |
| 12 | Global GM sign-off completed | Global GM Signoff Completed | Shell attribute, yes/no | — | No | |
| 13 | Goals (MQL, Revenue) | MQL, Revenue | calc | calc | — | Sum of activities |
| 14 | Status | — | Approval progress, Timing | calc | — | Domain model §2.2 |

## 3. Initiative

| # | Label | Shell / Hive9 name | Type | Mark | Default / options | Notes |
|---|---|---|---|---|---|---|
| 1 | Initiative name | Name | text (≤ 120) | R | — | Unique in its campaign |
| 2 | Campaign | Parent Campaign | reference | R | The campaign it was added to | Change with **Move** |
| 3 | Owner | Owner | person | P | Campaign owner | |
| 4 | Start date / End date | Start date, End date | date range | P | Campaign dates | Inside campaign dates (R3) |
| 5 | Description | Description | long text | — | — | |
| 6 | Initiative budget | Budget › Initiative Budget | money (USD) | — | — | |
| 7 | Remaining campaign budget | Remaining Campaign Budget | money | calc | — | May be negative |
| 8 | Planned cost | Planned Cost | money | calc | — | Sum of activities |
| 9 | Growth ambition | Growth Ambition | Shell attribute, multiple choice | — | — | Shell in-scope list |
| 10 | Function | Function | Shell attribute, single choice | — | — | |
| 11 | Focus area | Focus Area | Shell attribute, single choice | — | — | |
| 12 | FIT4 initiative | FIT4 Initiative | Shell attribute, yes/no | — | No | |
| 13 | Goals (MQL, Revenue) | MQL, Revenue | calc | calc | — | Sum of activities |
| 14 | Status | — | Approval progress, Timing | calc | — | Domain model §2.2 |

## 4. Activity

### 4.1 Basics

| # | Label | Shell / Hive9 name | Type | Mark | Default / options | Notes |
|---|---|---|---|---|---|---|
| 1 | Activity name | Name | text (≤ 120) | R | — | Unique in its initiative |
| 2 | External name | Edit External Name | text (≤ 120) | — | Empty (uses the activity name) | Name used by connected systems |
| 3 | Campaign, Initiative | Parent Campaign, Parent Initiative | reference | R | Where it was added | Change with **Move** |
| 4 | Activity type | Activity Type | reference (types allowed by the plan's model) | R | — | e.g. "MOBILITY – Long-term contract", "Agency fees – Creative" (R7) |
| 5 | Owner | Owner | person | S | Initiative owner | |
| 6 | Start date / End date | Start Date, End Date | date range | S | Initiative dates | Inside initiative dates (R3, R5) |
| 7 | Ongoing (no end date) | Ongoing Activity / Evergreen | yes/no | — | No | Irreversible once saved (R6) |
| 8 | Planned cost | Planned Cost | money (USD) | S | Activity type default, if any | Re-approval when changed (§2.5) |
| 9 | Approval workflow | Approval Workflow | reference (workflows) | S | — | e.g. "China BC Approval Workflow" |
| 10 | Description | Description | long text | — | — | |
| 11 | Status | Status | Created · Submitted · Declined · Approved · In progress · Complete | calc | Created | Domain model §2.3 |

### 4.2 Shell attributes — always shown

| # | Label | Shell name | Type | Mark | Notes |
|---|---|---|---|---|---|
| 12 | Channel / audience | Channel/Audience | multiple choice | S | e.g. External, Internal |
| 13 | Country / OU | Country/OU | multiple choice with % split | S | Splits must total 100.00 % (R10) |
| 14 | Company code | Company Code | single choice | S | e.g. CN04, GB01 — the legal entity GSAP uses |
| 15 | Cost centre | Cost Centre | single choice (searchable) | S | e.g. CN04000220 |
| 16 | Global campaign | Global Campaign | multiple choice | S | e.g. "Go Well integrated campaign" |
| 17 | Growth ambition | Growth Ambition | multiple choice | S | e.g. "MOBILITY – Integrated offer" |
| 18 | PIR | PIR | single choice (Yes / No) | S | Post-investment review needed |
| 19 | FIT4 initiative | Fit4 Initiative | single choice | S | |
| 20 | FIT4 ID | Fit4 ID | text | S | |
| 21 | Zero-based cost hierarchy | (Zero based) Cost Hierarchy | multiple choice | — | Shell's 10 values: LUBES - Growth drivers, … , RETAIL - Tactical Investments (D-325) |

### 4.3 Shell attributes — shown when the plan's business unit includes Lubricants

| # | Label | Shell name | Type | Mark |
|---|---|---|---|---|
| 22 | Brands | Brands (LUBES) | multiple choice with % split | S |
| 23 | Global key accounts | Global Key Accounts (LUBES) | multiple choice | S |
| 24 | Sector | Sector (LUBES) | multiple choice | S |
| 25 | Sub-sector | Sub-Sector (LUBES) | multiple choice | S |
| 26 | Product launch | Product Launch (LUBES) | single choice | S |

### 4.4 Shell attributes — shown when the plan's business unit includes Mobility

| # | Label | Shell name | Type | Mark |
|---|---|---|---|---|
| 27 | Function (Retail) | Function (RETAIL) | single choice | — |
| 28 | Focus area (Retail) | Focus Area (RETAIL) | single choice | — |
| 29 | Function (Mobility) | Function (Mobility) | single choice | — |
| 30 | Focus area (Mobility) | Focus Area (Mobility) | single choice | — |
| 31 | Innovation | Innovation | long text | — |

### 4.5 Goals

| # | Label | Shell name | Type | Mark | Notes |
|---|---|---|---|---|---|
| 32 | INQ (inquiries) | INQ | number (whole) | — | |
| 33 | Set later stages myself | Override Model | yes/no | — | When on, MQL and revenue become editable |
| 34 | MQL, Revenue | MQL, Revenue | calc (or typed when overridden) | calc | Domain model §4.5 |

### 4.6 From finance (coming later)

| # | Label | Shell name | Mark |
|---|---|---|---|
| 35 | Total actuals | Activity – Total Actuals | later |
| 36 | Total invoiced | Total Invoiced Amount | later |
| 37 | Total purchase requests | Total Purchase Request Amount | calc — the activity's Purchase requests section (feature 032) |

## 5. Line item

| # | Label | Shell / Hive9 name | Type | Mark | Default / options | Notes |
|---|---|---|---|---|---|---|
| 1 | Line item name | Name | text (≤ 120) | R | "<activity name> – line item" | Unique in its activity |
| 2 | Owner | Owner | person | S | Activity owner | |
| 3 | Description | Description | long text | — | — | |
| 4 | Currency | Local Currency | reference (active currencies) | R | USD | |
| 5 | Planned cost | Planned Cost in Local Currency | money (local) | R | Current unassigned balance | |
| 6 | Planned cost in USD | Planned Cost in Preferred Currency | money | calc | — | Domain model §4.3 |
| 7 | Spread over time | (allocation C/E/F/L) | single choice + month amounts | — | Plan's default spread | Domain model §4.4 |
| 8 | Spend type | Spend Type | Shell attribute, single choice | S | — | Shell's spend types, e.g. "ATL - Digital", "BTL - Event": the 25 the functional analysis names (of 57); the rest are added in Settings (D-325) |
| 9 | GL code | GL Code | Shell attribute, single choice | S | — | Shell's 13 marketing GL codes, e.g. "7250080 - Advertising - Digital"; independent of spend type until Settings links them (R11, D-325) |
| 10 | Spend category | Spend Category | Shell attribute, single choice | — | — | Shell's 19 values, e.g. "B2B Customer Specific", "Not Applicable" (D-325) |
| 11 | Contracted spend | Contracted Spend | Shell attribute, single choice | — | — | Not Applicable · No · Yes - Both · Yes - LT · Yes - ST (D-325) |
| 12 | Contracted amount | Contracted Amount | money (local) | — | — | |
| 13 | Request SWBS code from GSAP | Create SWBS Code | yes/no | — | Yes | Status per domain model §2.4; locked on once the code is received (030 FR-004) |
| 14 | Finance budget year | Link to Finance Budget › Budget Year | single choice | S | Plan year | |
| 15 | Finance budget item | Linked Budget Item | reference (finance budget items of that year) | S | Items of the plan's default finance budget listed first; pre-selected when it has one active item | R12 |
| 16 | SWBS code | SWBS Code | text, read-only | calc | — | Returned by GSAP — by the simulated GSAP until live integration (`SIM-<company code>-<nnnn>`, feature 030); "Filled by the simulated GSAP once the activity is approved and the request is sent" until then; kept for good once received; never copied |
| 16a | SWBS requested / received | — | timestamps, read-only | calc | — | When the request was sent and the code received (the run's time while simulated) |
| 16b | HX key | HX + line-item ID | text, read-only | calc | — | "HX" + the digits of the ID (L-1006 → HX1006), "Generated from the line-item number", with Copy; GSAP matches PR, PO and invoice data on it (feature 030, 015) |
| 17 | Total actuals, % of actual cost, Actuals at monthly rate, Total invoiced | (same) | money / percent | later | — | Domain model §4.6 |
| 17a | Total purchase requests, Available to request | Total Purchase Request Amount | money, read-only | calc | — | The line item's Purchase requests section (feature 032, domain model §2.4a) |

Not in this phase: the **Purchase requests & purchase orders** section of the Shell line-item form
(Finance module) and **Related primary activities** (future, see feature map).

## 6. Settings records

Fields of Settings records (models, activity types, attributes, currencies, finance budget items,
approval workflows, people and teams) are specified in features 002, 003 and 008 and summarised in
domain model §6.

## 7. Purchase request (feature 032)

Built-in fields: name ("PR for <line item>" by default), description, owner, MoA approver, approval workflow,
modification workflow; per line: line item, currency, amount, delivery date, overall limit, description;
read-only: ID, status, company code and cost centres (from the line items), HX keys, GSAP PR number and date,
and per line the SWBS code, Sent date, PO number, PO line, status, date and value.

Shell attributes at the levels *Purchase request* and *PR line* (Settings, D-325), in the demo:

| Level | Attributes (S = needed to submit) | MAPP reads |
|---|---|---|
| Purchase request | Material or service (S) · PR type (S) · Short text (S) · Requestor user ID · Behalf of requestor · Invoice approver name (S) · Vendor (S) · Vendor code · Vendor contact name · Vendor email address · Vendor address · Contract holder · Outline agreement · CP lead · Email group · PR recipients · OA vendor (China specific) · Sub category (China specific) | Vendor (grouping), Vendor code, Vendor email, Short text, PR type, Email group and PR recipients (routing) |
| PR line | Material group · Remarks / comments · Material / service code | — |
