# Domain Model — Plan Module

The single definition of MAPP's planning records, how they relate, their **statuses** and the
**money rules**. Specs refer to this document instead of redefining it. Field-by-field detail (labels,
types, required or not) is in [field-catalogue.md](field-catalogue.md); words are defined in
[glossary.md](glossary.md).

Sources: Shell's *Hive9 Plan Module* as-is document (28 Sep 2026), the Hive9 functional analysis
(Part A and B), the GSAP workshop (10 Sep 2026) and the Hive9 application itself. See
[../reference/README.md](../reference/README.md) for precedence.

---

## 1. The planning hierarchy

```
Plan                one fiscal year of marketing for one business (e.g. "FY2026 Lubricants Europe")
 └─ Campaign        a theme that runs for most of the year (e.g. "Helix launch")
     └─ Initiative  a group of similar work inside a campaign (e.g. "Digital & social")
         └─ Activity    one real piece of work (e.g. "Launch film") — the record that is approved
             └─ Line item   one cost inside the activity (e.g. "Agency fee") — where money is planned
```

| Rule | Detail |
|---|---|
| Created top-down | A campaign needs a plan, an initiative needs a campaign, an activity needs an initiative, a line item needs a saved activity. |
| One parent | Every record has exactly one parent. Moving a record to another parent is an explicit **Move** action. |
| Unique names | Plan names are unique across MAPP. Other names are unique **under the same parent** (two campaigns in one plan cannot share a name). Comparison ignores upper/lower case and surrounding spaces. |
| Readable IDs | Every record gets a short, permanent, human-readable ID: `P-1001` (plan), `C-1001` (campaign), `I-1001` (initiative), `A-1001` (activity), `L-1001` (line item). IDs are never reused. |
| One fiscal year | A plan belongs to one fiscal year (= calendar year, January–December, decision D-011). The year cannot change after the plan is saved. |
| Delete | Deleting a record also deletes everything under it. All of it goes to the **recycle bin** together and can be restored (parents first). |

Supporting records (managed in Settings, see §6): **Model**, **Activity type**, **Custom attribute**
(with its options), **Currency** and **exchange rate**, **Finance budget item**, **Approval workflow**,
**Person** and **Team**.

---

## 2. Statuses

Every record shows its status as a **word with a colour** (never colour alone). Status colours come
from [design-system.md](design-system.md) §4.

### 2.1 Plan

```
Draft ──Publish──► Published
  ▲                    │
  └─────Unpublish──────┘
```

| Status | Meaning | Who changes it |
|---|---|---|
| **Draft** | Being prepared. New and copied plans start here. | System on create/copy |
| **Published** | The agreed plan of record. Still editable; edits are recorded in the activity log. | Any user with **Publish** (all users in this phase) |

- **Publish** is allowed only when the plan and all its campaigns and initiatives are complete (no
  "needed to publish" field is empty — see §3). Otherwise the action stays visible and lists what is
  missing.
- **Unpublish** returns the plan to Draft and asks for a short reason, which is logged.
- A plan also shows the derived **Approval progress** of §2.2 for all activities in the plan.

### 2.2 Campaign and initiative

Campaigns and initiatives have **no approval of their own** in this phase (decision D-012). They show
two derived statuses, recalculated whenever something below them changes (plans show Approval progress
too):

| Derived status | Values | Rule |
|---|---|---|
| **Approval progress** | No activities · In progress · All approved | "In progress" shows a count, e.g. "3 of 5 activities approved" (also "0 of 5" when none is approved yet). "All approved" when every activity below is Approved, In progress or Complete. |
| **Timing** | Not started · Running · Finished | Compared with today: before the start date, between start and end, after the end date. Shown as "—" with the reason "No dates yet" while a record has no dates. |

### 2.3 Activity (the record that is approved)

```
Created ──Submit──► Submitted ──Approve (last level)──► Approved ──(start date reached)──► In progress ──(end date passed)──► Complete
   ▲                 ▲ ↺  │  │                              │
   │        Re-submit│    │  └──Recall (owner)──► Created   │
   │                 │    ▼                                 │
   │              Declined ◄── Decline (comment required)   │
   │                                                        │
   └─────────────── a "re-approval" change (§2.5) ◄─────────┘

Created / Declined ──Submit, when no level needs a decision──► Approved   (approved automatically)
```

- **Re-submit** sends a Declined activity straight back to Submitted and starts a new approval round at
  level 1. A **Submitted** activity can be re-submitted too (↺, feature 031): in one step its open round
  ends as *Replaced by a new submission* at the level it waited at (decisions already made stay in its
  history) and a new round starts with the workflow the activity has now; the dialog says which round
  ends before anything changes, and a problem leaves everything as it was. **Recall** lets the owner
  withdraw a Submitted activity back to Created. The approval rules (levels, approvers, notifications)
  are specified in feature 008 and extended by features 017 and 031.
- **Any record type** (feature 031): approval rounds, their history and notifications belong to one
  record of one type — **Activity**, **Purchase request** or **Modified purchase request** (feature 032
  connects the purchase requests). The rounds of a record share one numbering and at most one is open.
  "Nobody approves an activity they own or submitted" holds for activities only; on a purchase request
  such a decision is allowed and recorded as a self-approval.
- **Approval rules** (feature 017) are evaluated **once, at submission**, on the activity's planned cost
  (USD), its activity type and the plan's business units; the outcome is frozen in the approval round.
  Each level is then *not needed* (its "only needed when" condition does not match), *approved
  automatically* (its "approve automatically when" condition matches) or *waiting for a decision*. When
  no level waits for a decision, the activity goes from Created or Declined **straight to Approved** at
  submission; the history shows each level as "Not needed" or "Approved automatically" by MAPP, and
  the owner is notified when someone else submitted. Only the changes of §2.5 restart approval; changing
  the activity type or the plan's business units does not re-evaluate an open or approved round.

| Status | Stored or shown | Meaning |
|---|---|---|
| **Created** | Stored | Being planned. New, copied and restored activities start here. |
| **Submitted** | Stored | Waiting for the approver(s) of its approval workflow. |
| **Declined** | Stored | Sent back with a comment. The owner fixes it and submits again. |
| **Approved** | Stored | Agreed. Shown as *Approved* until its start date. |
| **In progress** | Shown only | Approved and today is on or after the start date. |
| **Complete** | Shown only | Approved and today is after the end date (never for ongoing activities). |

- **Submit** requires every "needed to submit" field of the activity **and of all its line items** to be
  filled (§3). The action lists everything missing at once.
- Approval levels and approvers come from the activity's **approval workflow** (feature 008).

### 2.4 Line item

Line items have **no approval of their own**: they are approved as part of their activity. They show:

| Status | Values | Rule |
|---|---|---|
| **Completeness** | All filled in · *n* fields needed | All "needed to submit" fields filled. |
| **SWBS request** | Not requested · Waiting for activity approval · Ready to send to GSAP · SWBS code received | *SWBS code received* once GSAP returned the line item's code — it wins over everything else and stays when the activity leaves approval, the line item is edited, moved, deleted and restored; "Request SWBS code" can then no longer be switched off. Otherwise *Not requested* when "Request SWBS code" is off, *Ready…* once the activity is approved (also while In progress or Complete), *Waiting…* before. Until live GSAP integration, a **simulated GSAP** answers: a person with "Import GSAP extracts" sends the ready line items from the Financials page or one line item from its page, and each receives `SIM-<company code>-<nnnn>` at once, numbered per company code (feature 030, ADR-0016). Copies never take the code. |

### 2.4a Purchase request (feature 032)

A purchase request asks to buy from one vendor. It belongs to one plan and has one or more **PR lines**,
each on a different line item of that plan, all under one company code; a line has its own currency,
amount, delivery date and overall limit.

| Status | Values | Rule |
|---|---|---|
| **Status** | Created · Submitted · In review · Approved · Declined · Closed | *Submitted* when sent for approval, *In review* once a level approves, *Approved* after the last level; a declined round makes it *Declined*, a recall *Created*. A change to an Approved purchase request is a **modification**: *In review* with the approved version kept, restored if the change is declined. *Closed* by the person closing it; each line becomes the smaller of its amount and what was invoiced. |
| **Completeness** | All filled in · *n* fields needed | As every record (§2.6), over the purchase request's Shell attributes and its lines. |

- **Reservation.** A line reserves its amount on its line item as soon as it is saved, in every status;
  it may not exceed the line item's *available to request* (other currencies at today's rate). Closing
  releases what was not invoiced; deleting a Created or Declined purchase request releases it all.
- **Submit gates.** Each line item's activity is Approved and has its SWBS code, and every needed field is
  filled; the approval follows the purchase request's workflows (031), and its owner may approve it.
- **Routing.** Approval routes it (Original or Modification); Resend and Close route it again. Each line's
  Sent date is set at its first routing and never changes.
- **Protection.** A line item, activity, initiative or campaign a purchase request draws on cannot be
  deleted, the campaign cannot move to another plan, and the line item's planned cost cannot fall below
  what is requested.
- **GSAP data** (feature 033). Until live integration the simulated GSAP fills the GSAP PR number and, per
  line, the PO number, PO line, status *Released*, date and value, and books each PO line as a
  purchase-order transaction linked by HX key, which the line item counts as Committed. After an approved
  change it sends only the changed and added lines, keeping its numbers (top-up).

### 2.5 Changes that need re-approval

When an activity is **Submitted** or **Approved**, these changes send it back to **Created** after the
user confirms ("This activity will need approval again"):

- its planned cost, start date, end date or ongoing flag (a split changes its planned cost too).

Line items never send their activity back for approval: adding, duplicating, deleting, restoring or
changing a line item — its planned cost and currency included — keeps the activity's status, because
Shell adds and adjusts line items after the activity is approved, then requests the SWBS code and raises
the purchase request (decision D-322, feature 034; supersedes the line-item part of D-013). A negative
unassigned balance (§4.3) still shows its warning and still blocks submitting and re-submitting (D-016).

Other edits (description, owner, attributes) are allowed without re-approval. Every change is recorded in
the activity log (decision D-013).

A change of exchange rate in Settings never sends an activity back for approval, even though it changes
the USD value of its line items (and may make its unassigned balance negative).

**Moving** an activity to another initiative keeps its status, unless its dates must change to fit the
new initiative — then the date change counts as a re-approval change above. Moving never changes the
planned cost or line items.

### 2.6 Completeness (all levels)

Every record shows a **completeness indicator**: "All filled in" or "*n* fields needed" (never the word "Complete", which is an activity status), which opens a list
linking to each empty field. It counts the record's own "needed to publish/submit" fields only; parents
additionally show "*n* items below need attention" with a link, counting every level below them (campaigns, initiatives, activities and line items).

---

## 3. Required fields: "to save" versus "to submit"

To keep MAPP friendly, records can be saved as drafts with very little information (decision
D-009):

| Marking | Blocks | Examples |
|---|---|---|
| **Required** | Saving | Plan name, year, model; activity name and type; line item name and planned cost |
| **Needed to publish** | Publishing the plan (plan, campaign, initiative fields) | Plan teams, owner, business unit, goal stage; campaign dates, cluster, region; initiative dates |
| **Needed to submit** | Submitting the activity (activity and line item fields) | Activity owner, dates, planned cost, approval workflow, required Shell attributes; line item spend type, GL code, finance budget item |
| (none) | Nothing | Description, budgets, optional attributes |

Fields filled by the system (calculated values, GSAP values) are never marked required. A dependent attribute whose parent value has **no options set up** (e.g. a cluster with no regions) is not counted as missing; it shows "No options set up for this cluster — ask an administrator" (avoids Hive9's unfillable Business Unit trap, FA B10 #5). Shell attributes
that are hidden (their business-unit condition does not apply) or inactive never block publishing or
submitting.
The marking of every field is in [field-catalogue.md](field-catalogue.md).

---

## 4. Money rules

All money is an exact decimal amount **with a currency**. MAPP's **base currency is USD** (decision
D-010): totals across plans are calculated in USD. Since feature 016 every plan has a **plan currency**
in which its budgets and planned costs are typed (§4.7); a line item may be entered in a local currency
and is converted. Users can choose a **display currency**; every amount on screen is converted for
display and labelled with its currency.

### 4.1 Budget flows down (marketing "plan budget")

| Level | Field | Shown to the child |
|---|---|---|
| Plan | Plan budget | — |
| Campaign | Campaign budget | **Remaining plan budget** = plan budget − sum of all campaign budgets in the plan |
| Initiative | Initiative budget | **Remaining campaign budget** = campaign budget − sum of all initiative budgets in the campaign |

- Budgets are optional. When the parent has no budget, the child's remaining budget shows "—" ("No budget set on the campaign"). A remaining budget **may become negative** (Shell's as-is document); it is shown
  in red with "Over-allocated by X" and never blocks saving (decision D-015).
- **Overspend control** (feature 037, D-346 – D-348): every plan, campaign, initiative and global budget says
  what happens "When planned cost goes over this budget": **Warn** (the default, the rule above) or
  **Refuse**. With Refuse, any save after which the planned cost under the budget (§4.2; for a global budget,
  its allocated plans' planned cost in USD at the current rates) is above it, and that raised the planned
  cost, lowered or newly set the budget, or created it (a copy), is refused naming the budget, its amount and
  the overage. Saves that keep or lower the cost always pass, so Refuse may be chosen while already over.
  Line items and exchange rates are never refused; over-allocation keeps warning. Only the record's owner or
  someone who manages global budgets changes the setting; an allocated plan's is set with its allocation. New
  budgets start with Settings › Overspend control (Warn as delivered); copies keep their source's.
- Activities and line items have **no budget**; they have **planned cost**.

### 4.2 Cost flows up (planned cost)

| Level | Planned cost |
|---|---|
| Activity | Typed by the user (needed to submit). |
| Initiative | Sum of its activities' planned cost (read-only). |
| Campaign | Sum of its initiatives' planned cost (read-only). |
| Plan | Sum of its campaigns' planned cost (read-only). |

Each container also shows **Budget used** = planned cost ÷ budget (when a budget is set), as a progress
bar, red above 100 %, and **Left to plan** = budget − planned cost (negative, in red, when the planned
cost exceeds the budget).

### 4.3 Line items and the unassigned balance

- A line item's planned cost is typed in its **local currency** and converted to USD:
  `planned cost (USD) = planned cost (local) ÷ rate`, where the rate is "units of local currency per
  1 USD" (Settings), rounded half-up to 2 decimals.
- **Unassigned planned cost (Sys_Gen_Balance)** = activity planned cost − sum of its line items'
  planned cost (USD). It appears as a system row under the line items. The target is **0.00**.
  - Positive: "X not yet assigned to a line item".
  - Negative: shown in red, "Line items exceed the activity's planned cost by X". It warns but does not
    block saving; it **does** block submitting (decision D-016) and submitting a purchase request that draws
    on the activity's line items (decision D-349).
  - **Approval over planned cost** (feature 038 FR-019 – FR-021, D-360): Settings › Overspend control says what
    happens when an activity's line items exceed its planned cost while it, or a purchase request drawing on it, is
    approved: **Warn the approver** (the default) or **Refuse approval**. The overage is the negative unassigned
    balance in the plan currency, worked out when approving. With Warn, the approval section and the Approve dialog
    show "Line items of activity '<name>' are <overage> above its planned cost" and the approval's log entry records
    it. With Refuse, approving any level of an activity, purchase-request or change round is refused naming each
    activity and its overage; Decline and Recall stay allowed.
- When adding a line item, its planned cost is pre-filled with the current unassigned balance.

### 4.4 Spreading a line item over time

Each line item spreads its planned cost over the **months** of its activity (or quarters — a display
choice) using one of four methods (default from the plan's *Default line-item spread*):

| Method | Label | Rule |
|---|---|---|
| E | Evenly | Split equally over the activity's months; each month gets the amount rounded down to the cent, and the remainder goes to the last month. Example: 10,000.00 over 3 months → 3,333.33 / 3,333.33 / 3,333.34. |
| F | All in first month | Everything in the activity's first month. |
| L | All in last month | Everything in the activity's last month. |
| C | Custom | The user types an amount per month; the months may lie outside the activity's dates (e.g. a deposit); the amounts must add up to the line item's planned cost before the activity can be submitted. |

- Spread amounts are in the line item's **local currency** and rounded to the cent. The USD value of
  each month is that month's amount converted at the current rate (§4.3), rounded to the cent, with any
  rounding difference placed in the last month so the months always add up to the line item's USD
  planned cost.
- The **spread grid** (feature 038 FR-022) converts each row total once into the display currency and gives the
  rounding difference to the row's last month with an amount, so in any display currency each row adds up to its total, each
  column to its column total and the column totals to the grand total. The grand total may differ by a cent from
  the separately converted "Assigned" figure (D-061).
- For an **ongoing** activity (no end date), Evenly and Last behave like First (only Custom spreads).
- While an activity has **no dates yet** (dates are only needed to submit), its E/F/L line items cannot
  be spread: they show "Spread once the activity has dates" and their amounts appear in a **No dates**
  column in time-based views. Custom amounts are kept as typed.
- A **negative** unassigned balance is never spread over months; a positive one is spread evenly over the
  activity's months in time-based views (feature 010).
- If the activity's dates change, E/F/L line items are re-spread automatically; Custom amounts are kept
  and flagged if they no longer add up.

### 4.5 Goals (the model's waterfall)

Shell's funnel has three stages: **INQ** (inquiries) → **MQL** (marketing-qualified leads) →
**Revenue**. Each plan uses a **model** that holds the conversion rates and the average deal size.

| Calculation | Formula |
|---|---|
| MQL | INQ × the model's INQ→MQL rate, rounded half-up to a whole number |
| Deals | MQL × the model's MQL→deal rate, rounded half-up to a whole number |
| Revenue | Deals × the model's average deal size (USD) |

Example with rates 2 % and 5 % and deal size 10,000: INQ 250,000 → MQL 5,000 → 250 deals →
revenue 2,500,000 USD.

- **Plan goal:** the user picks the goal stage (INQ, MQL or Revenue) and an amount; the other stages are
  calculated from the model (working backwards when the stage is MQL or Revenue).
- **Activity goal:** the user types INQ; MQL and revenue are calculated. **Set later stages myself**
  (Hive9: *Override model*) lets the user type all three.
- Campaign, initiative and plan show the **sum of their activities' goals** next to the plan goal.

### 4.6 Values that come from Finance or GSAP later

Shell's forms show several read-only fields fed by finance and GSAP (actuals, invoiced amount, purchase
request amount, SWBS code, actuals at monthly rate, % of actual cost). In this phase they are shown in a
collapsed **"From finance (coming later)"** section with "—" and the explanation "Filled when the
Finance module and GSAP are connected". They are never typed by users. The SWBS code is the exception
since feature 030: the line item stores the code the simulated GSAP returned and shows it on its SWBS
request card with its dates, its source and the HX key (§2.4). The purchase-request totals of a line item
and an activity are live since feature 032 (§2.4a), and the simulated GSAP's purchase orders count as
Committed since feature 033.

### 4.7 Plan currency and dated exchange rates (feature 016)

- A plan's budget, its campaigns' and initiatives' budgets and its activities' planned costs are typed and
  stored in the **plan currency** (USD by default). It can change only while nothing in the plan has an
  amount in it.
- A currency keeps a **rate history**: rates per 1 USD, each valid from a date. A conversion uses the
  latest rate dated on or before today (the **current rate**); a future rate changes nothing before its
  date, and the last current rate cannot be deleted. Rates are added in Settings or imported from the
  simulated **GSAP rate file**, which is checked before anything is saved.
- Any amount converts to any currency through USD (`Convert`): from its currency to USD, then to the other,
  rounded half up to the cent once. The **unassigned planned cost** of an activity is worked out in the plan
  currency (§4.3); line items in other currencies are converted at the current rate.

### 4.8 Budget by month (feature 016)

A plan, campaign or initiative budget may be **phased**: None, Evenly (the remainder in the last month) or
custom amounts per month of the plan's year. The months must add up to the budget; while they do not, the
card says by how much ("Budget by month adds up to 590,000.00 GBP, not the budget 600,000.00 GBP"). Removing
the budget removes its phasing. Budget & cost can compare each period's planned cost with the phased budget.

### 4.9 Global budgets (feature 014)

A **global budget** is money for a year owned by someone who manages budgets for all markets. It is shared
out by **allocations** to plans of the same year: an allocation *is* the plan's budget, which the plan's
owner sees but cannot change ("Allocated from '<global budget>' — ask <owner> to change it."). The global
budget shows how much is allocated, unallocated (red, "Over-allocated by X", when negative) and planned;
people see only the allocations to plans they can see, and the rest as one "Other markets" row.

---

## 5. Other rules

| # | Rule | Source |
|---|---|---|
| R1 | A plan's owner must be a member of at least one of the plan's teams; the owner list offers only those members. | PM, FA B10 |
| R2 | A campaign's or initiative's owner defaults to its parent's owner. | MAPP (fewer steps) |
| R3 | An initiative's dates must lie inside its campaign's dates; an activity's dates must lie inside its initiative's dates, except ongoing activities (no end date). | FA B5 |
| R4 | Campaign dates are expected inside the plan's fiscal year; a campaign starting before 1 January or ending after 31 December is allowed with a warning. | FA B4 (multi-year), simplified D-017 |
| R5 | An activity's end date may not be more than 10 years after its plan's year. | FA B6 |
| R6 | **Ongoing (evergreen)** cannot be undone once the activity is saved; the user is warned before saving. | PM, FA B6 |
| R7 | An activity's type must be one the plan's model allows. Changing the type offers the type's default planned cost (if one is set). | FA B6 |
| R8 | A campaign's **Region** options depend on its **Cluster**; when a cluster has one region it is filled in automatically. | PM step 2 |
| R9 | Attribute groups tied to a business unit (Lubricants, Mobility/Retail) appear only when the plan's business unit includes it; hidden groups keep their values. | PM step 4, FA B10 |
| R10 | Multi-select attributes with a percentage split (Country/OU, Brands) must total exactly 100.00 % before submitting; selections start with an even split (remainder on the last value). | PM step 4 |
| R11 | A line item's GL code options depend on its spend type only when an administrator links them in Settings; Shell's seed has no such links, because no source gives the mapping (D-325). | Workshop; FA E3.0.2 |
| R12 | A line item must link to one **finance budget item** of the chosen budget year before submitting. The year defaults to the plan's year; the items of the plan's **default finance budget** are listed first, and the item is pre-selected only when that budget has a single active item. | PM step 5 |
| R13 | Copying a record copies everything below it; copies start as Draft (plan) / Created (activity) and never copy approvals, comments, activity-log history or GSAP values. Copy names get " (copy)", then " (copy 2)", … | FA B8 |
| R14 | **Undo** right after a delete (within its display time) restores everything exactly as it was, statuses and open approval rounds included. **Restoring later from the recycle bin page** requires the parent to exist (restore the parent first, or choose "Restore both"), keeps the readable IDs, and resets statuses: restored plans come back as Draft, restored activities as Created (their approval history is kept, open approval rounds are not resumed). | FA B8, D-031 |

---

## 6. Supporting records (Settings)

| Record | Key data | Used by |
|---|---|---|
| **Model** | Name, INQ→MQL rate, MQL→deal rate, average deal size (USD), allowed activity types, active | Plan (required), activity goals, activity type list |
| **Activity type** | Name, optional default planned cost (USD), active | Activity |
| **Custom attribute** | Label, level (plan · campaign · initiative · activity · line item), type (single choice · multiple choice · multiple choice with % split · yes/no · text · long text · number), options, marking (optional / needed to publish / needed to submit), business-unit condition, parent attribute for dependent options, display order, active | Every level's form |
| **Currency** | Code (ISO), name, symbol, active, and a **rate history** of dated rates per 1 USD (USD fixed at 1) whose latest rate on or before today is the current rate (§4.7) | Plan currency, line items, display currency |
| **Finance budget item** | Budget year, budget name (e.g. "2026 Lubricants Budget"), item name, code, active. A stand-in list until the Finance module exists. | Line item link, plan default finance budget |
| **Approval workflow** | Name, what it approves (object type: activity · purchase request · modified purchase request, chosen at creation and fixed once used), ordered levels, active; remind after (1–60 days) and escalate after (1–90 days, more than remind after), both optional. Each level: who approves (named people · the record owner · the owner's line manager · the holders of a role that approves, for the plan · the person named on the record, for purchase requests the MoA approver), any one or all must approve (one approver for the owner, the line manager and the named person), optional "only needed when" and "approve automatically when" conditions (planned cost — for purchase requests the amount — at least / below in USD; for activities also business unit is one of, activity type is one of; the parts combine with AND), and a backup approver (required on every level when escalation is on) | Activity, purchase request (feature 032) |
| **Person** | Name, e-mail, job title, teams, line manager (optional; never oneself, no loops) | Owners, approvers, "line manager" approval levels, demo sign-in |
| **Team** | Name, members, colour (assigned automatically from the chart palette, changeable) | Plan teams, owner choice, "My teams" filter, coloured team pills, role scopes (a market is a team) |
| **Role** (feature 014) | Name, description, permissions from the fixed list of 014 Table A, default scope; five built in (Administrator, Global planner, Planner, Approver, Viewer); the Administrator role holds every permission and cannot change | Role assignments: a person's roles, each for All markets, My teams or Chosen teams |
| **Access rules** (feature 014) | "Nobody approves an activity they own or submitted" (on by default) and the role new people get (Viewer) | Approvals, new people |
| **Field help** (feature 016) | One help text (0–300 characters, plain text) per built-in field and calculated value; MAPP supplies a default, an administrator may edit or reset it; empty hides the ⓘ | Every form, details page and summary card |

Configuration records are never hard-deleted while in use: they are set **inactive** (no longer offered
for new choices, existing values kept and labelled "(inactive)").

---

## 7. Cross-cutting records

| Record | Purpose |
|---|---|
| **Activity log entry** | Who did what to which record, when, with the changed fields (before → after). Written for every create, edit, status change, move, copy, delete and restore. Read-only. |
| **Recycle bin entry** | A deleted record and everything that was under it, who deleted it and when; restorable. |
| **Comment** | Free-text note on an activity (also used for decline reasons). |
| **Notification** | In-app message to a person about one record of one type, e.g. "Activity A-1042 is waiting for your approval" or "Purchase request approved", also for records approved automatically, approval reminders and escalations. |
| **Approval delegation** | A person's approvals handed to a delegate (who can approve) for a date range of at most 90 days, with an optional note; planned, current, ended or cancelled; never deleted. While current, the delegate approves the person's waiting levels "as delegate of" them. |
| **Saved view** | A user's named combination of filters, columns and view type on the planning views. |
| **Exchange rate** (feature 016) | One rate of a currency per 1 USD, valid from a date, with its source (typed or a GSAP rate import) and who recorded it. |
| **Rate import** (feature 016) | One import of the simulated GSAP rate file: file name, who, when, and how many rates were added, skipped or refused. Log only. |
| **Global budget** and **Budget allocation** (feature 014) | Money for a year and the parts of it given to plans (§4.9); deleted to the recycle bin like any record. |

---

## 8. Who may see and do what (feature 014)

- Every action needs its permission of 014 Table A, given by a person's **roles**; each role assignment has
  a **scope**: All markets, My teams (the person's teams, as they are now) or Chosen teams. The API checks
  every request and refuses an operation without a rule (fail closed).
- A person sees a plan and everything in it only when a role gives them "View plans" for it or they own it;
  they also see an activity they own or are named to approve. A record outside their access looks exactly
  like a missing one; 403 refuses only an action on a record they can see, with the reason.
- Screens follow the API's decisions: where a person may change nothing, one **View only** note names their
  roles and the administrators to ask; where they may change some things, an action they may not take is
  shown disabled with its reason (D-352).
- Approve/decline is limited on the server (feature 017): the waiting approvers of the current level who hold
  "Approve or decline when named as an approver", their current delegates, the backup approver once the level
  is escalated, or someone allowed to decide on behalf. While the access rule "Nobody approves an activity
  they own or submitted" is on, the owner and the submitter never decide and are left out when approvers are
  resolved (segregation of duties). The history records the capacity (as delegate of, as backup approver, on
  behalf of). A role changed in Settings changes who may approve at once.
- Sign-in stays a demo until Microsoft Entra ID; line managers are kept in MAPP and later come from Entra ID.
  Reminders and escalations run when someone allowed to change approval workflows runs them; later a
  scheduler runs them and notifications are also sent by e-mail.
