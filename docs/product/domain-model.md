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
   ▲                 ▲    │  │                              │
   │        Re-submit│    │  └──Recall (owner)──► Created   │
   │                 │    ▼                                 │
   │              Declined ◄── Decline (comment required)   │
   │                                                        │
   └─────────────── a "re-approval" change (§2.5) ◄─────────┘

Created / Declined ──Submit, when no level needs a decision──► Approved   (approved automatically)
```

- **Re-submit** sends a Declined activity straight back to Submitted and starts a new approval round at
  level 1. **Recall** lets the owner withdraw a Submitted activity back to Created. The approval rules
  (levels, approvers, notifications) are specified in feature 008 and extended by feature 017.
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
| **SWBS request** | Not requested · Waiting for activity approval · Ready to send to GSAP | *Not requested* when "Request SWBS code" is off. *Waiting…* while the activity is not approved. *Ready…* once the activity is approved. (Sending to GSAP and receiving the code are future features; the code field shows "Filled by GSAP later".) |

### 2.5 Changes that need re-approval

When an activity is **Submitted** or **Approved**, these changes send it back to **Created** after the
user confirms ("This activity will need approval again"):

- its planned cost, start date or end date;
- adding, deleting or changing the planned cost or currency of one of its line items.

Other edits (description, owner, attributes, line item coding) are allowed without re-approval and are
recorded in the activity log (decision D-013).

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
D-010): budgets and planned costs are stored in USD; a line item may be entered in a local currency and
is converted to USD. Users can choose a **display currency**; every amount on screen is converted for
display and labelled with its currency.

### 4.1 Budget flows down (marketing "plan budget")

| Level | Field | Shown to the child |
|---|---|---|
| Plan | Plan budget | — |
| Campaign | Campaign budget | **Remaining plan budget** = plan budget − sum of all campaign budgets in the plan |
| Initiative | Initiative budget | **Remaining campaign budget** = campaign budget − sum of all initiative budgets in the campaign |

- Budgets are optional. When the parent has no budget, the child's remaining budget shows "—" ("No budget set on the campaign"). A remaining budget **may become negative** (Shell's as-is document); it is shown
  in red with "Over-allocated by X" and never blocks saving (decision D-015).
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
    block saving; it **does** block submitting (decision D-016).
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
Finance module and GSAP are connected". They are never typed by users.

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
| R11 | A line item's GL code options depend on its spend type. | Workshop |
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
| **Currency** | Code (ISO), name, symbol, rate = units per 1 USD (USD fixed at 1), active | Line items, display currency |
| **Finance budget item** | Budget year, budget name (e.g. "2026 Lubricants Budget"), item name, code, active. A stand-in list until the Finance module exists. | Line item link, plan default finance budget |
| **Approval workflow** | Name, ordered levels, active; remind after (1–60 days) and escalate after (1–90 days, more than remind after), both optional. Each level: who approves (named people · the activity owner's line manager · the holders of a role that approves, for the plan), any one or all must approve, optional "only needed when" and "approve automatically when" conditions (planned cost at least / below in USD, business unit is one of, activity type is one of; the parts combine with AND), and a backup approver (required on every level when escalation is on) | Activity |
| **Person** | Name, e-mail, job title, teams, line manager (optional; never oneself, no loops) | Owners, approvers, "line manager" approval levels, demo sign-in |
| **Team** | Name, members, colour (assigned automatically from the chart palette, changeable) | Plan teams, owner choice, "My teams" filter, coloured team pills |

Configuration records are never hard-deleted while in use: they are set **inactive** (no longer offered
for new choices, existing values kept and labelled "(inactive)").

---

## 7. Cross-cutting records

| Record | Purpose |
|---|---|
| **Activity log entry** | Who did what to which record, when, with the changed fields (before → after). Written for every create, edit, status change, move, copy, delete and restore. Read-only. |
| **Recycle bin entry** | A deleted record and everything that was under it, who deleted it and when; restorable. |
| **Comment** | Free-text note on an activity (also used for decline reasons). |
| **Notification** | In-app message to a person, e.g. "Activity A-1042 is waiting for your approval", also for activities approved automatically, approval reminders and escalations. |
| **Approval delegation** | A person's approvals handed to a delegate (who can approve) for a date range of at most 90 days, with an optional note; planned, current, ended or cancelled; never deleted. While current, the delegate approves the person's waiting levels "as delegate of" them. |
| **Saved view** | A user's named combination of filters, columns and view type on the planning views. |

---

## 8. What changes when real sign-in and roles arrive (later)

- Plans become visible only to members of the plan's teams (Shell's rule). Today a **My teams / All
  plans** switch on the Plans list shows this without enforcing it.
- Approve/decline is limited on the server (feature 017): the waiting approvers of the current level, their
  current delegates, the backup approver once the level is escalated, or someone allowed to decide on
  behalf. While the access rule "Nobody approves an activity they own or submitted" is on, the owner and
  the submitter can never decide and are left out when approvers are resolved (segregation of duties). The
  history records the capacity (as delegate of, as backup approver, on behalf of).
- Line managers are kept in MAPP today; later they come from Entra ID. Reminders and escalations run
  when someone allowed to create and change approval workflows runs them (a job endpoint); later a
  scheduler runs them and notifications are also sent by e-mail.
- Permissions per level (create/edit/delete) come from roles mapped to Shell's 19 Hive9 roles.
