# Data Model: Activities

**Feature**: `006-activities` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

The records this feature stores, their fields and rules, their statuses and the values derived from them.
Conventions (timestamps, text lengths, `ActorRef`, `RecordRef`, `record.Meta`, relational shape) are those
of [001's data model](../001-app-shell-and-sign-in/data-model.md). Field labels and markings come from the
[field catalogue §4](../../docs/product/field-catalogue.md); statuses and money rules from the
[domain model §2.3, §2.5, §2.6, §3, §4](../../docs/product/domain-model.md).

**Marking**: **R** Required (blocks saving) · **S** Needed to submit (blocks submitting) · — optional ·
calc calculated · later filled by Finance/GSAP later.

---

## 1. Activity — table `activities`

One piece of marketing work under an initiative. JSON names are those of the file and of the API's input.

| Field | Label (field catalogue §4) | Type | Mark | Default (new form) | Rules (save-blocking unless marked S) |
|---|---|---|---|---|---|
| `id` | ID | string | — | — | `A-1001`, … from the counter `A`; permanent, never reused (D-019) |
| `initiativeId` | Initiative (and through it Campaign) | reference (initiative, 005) | R | The initiative it was added under | Must exist; changes only with **Move** (same plan); campaign and plan are read through the initiative, never stored here |
| `name` | Activity name | text ≤ 120 | R | — | Trimmed; "Enter a name."; "Use 120 characters or fewer."; unique among the initiative's activities ignoring case and surrounding spaces: "'<initiative>' already has an activity called '<name>'." |
| `externalName` | External name | text ≤ 120 | — | empty | Trimmed; when empty the activity name is used by connected systems |
| `activityTypeId` | Activity type | reference (activity type, 002) | R | — | "Choose an activity type."; when chosen or changed: active and in the plan's model's allowed types (R7), else "Choose an activity type that the model '<model>' allows."; an unchanged inactive or no-longer-allowed type is kept (FR-013) |
| `ownerId` | Owner | person (001/003) | S | The initiative's owner (when active) | `""` = not set; when chosen or changed: an existing active person, else "Choose an active person."; an unchanged inactive owner is kept |
| `startDate` | Start date | date `YYYY-MM-DD` | S | The initiative's start date | `""` = not set; valid date ("Enter a valid date.") |
| `endDate` | End date | date `YYYY-MM-DD` | S (not when ongoing) | The initiative's end date | `""` = not set or ongoing; ≥ start ("End date must be on or after the start date (<start>)."); ≤ 31 Dec of plan year + 10 (R5: "The end date can't be after <date> — 10 years after the plan's year.") |
| — | (both dates) | | | | When start, end or ongoing is entered or changed: each date inside the initiative's dates when the initiative has both (R3): "Dates must be inside the initiative's dates (<start> – <end>)."; ongoing: only the start is checked |
| `ongoing` | Ongoing (no end date) | yes/no | — | No | True clears `endDate`; once saved true it cannot become false (R6): "Ongoing can't be undone — to plan with an end date, create a new activity." |
| `plannedCost` | Planned cost | money, USD | S | The type's default planned cost, if any (web form) | `null` = not set; 0.00 is a real value; currency `USD` ("Enter the planned cost in USD."); amount ≥ 0 ("Planned cost can't be negative."); more than 2 decimals rounded half-up to the cent |
| `approvalWorkflowId` | Approval workflow | reference (approval workflow, 008) | S | The only active workflow, when exactly one is active | `""` = not set; when chosen or changed: active ("Choose an active approval workflow.") and status `created` or `declined` ("The approval workflow can't change while the activity is submitted or approved."); an unchanged inactive one is kept but listed in the ready-to-submit check |
| `description` | Description | long text ≤ 2,000 | — | empty | "Use 2,000 characters or fewer." |
| `attributes` | Shell attributes (catalogue §4.2–4.4) | `AttributeValues` (002) | per attribute: S or — | empty | Save-blocking checks by 002 (`settings.CheckValues`): known attribute and option, a newly chosen option active, a dependent option linked to the parent's value, each share 0.00–100.00 with ≤ 2 decimals; unchanged inactive values pass; values of hidden business-unit groups are kept even when a save omits them (FR-015) |
| `inq` | INQ (inquiries) | whole number ≥ 0 | — | empty | "Enter a whole number of 0 or more." |
| `setLaterStagesMyself` | Set later stages myself | yes/no | — | No | When false, `mql` and `revenue` are not stored (calculated at read time) |
| `mql` | MQL | whole number ≥ 0 | calc / typed | — | Stored only when `setLaterStagesMyself`; "Enter a whole number of 0 or more." |
| `revenue` | Revenue | money, USD | calc / typed | — | Stored only when `setLaterStagesMyself`; ≥ 0 ("Revenue can't be negative."), rounded to the cent |
| `status` | Status | `created` · `submitted` · `declined` · `approved` | calc | `created` | §3 |
| *bookkeeping* | — | `record.Meta` | — | — | `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy` (001) |

Field catalogue §4.6 (Total actuals, Total invoiced, Total purchase requests) are **not stored**; the page
shows them as "—" (later).

Every save problem is reported at once (`validation_failed`, one `FieldError` per problem, `field` = JSON
path such as `goals.inq` or `attributes.<attributeId>`).

### Relations

```
plans.Plan (004) 1 ── * campaigns.Campaign (005) 1 ── * campaigns.Initiative (005) 1 ── * Activity 1 ── * lineItems.LineItem (007)
Activity * ── 1 settings.ActivityType (002)       Activity * ── 0..1 people.Person (owner)
Activity * ── 0..1 approval workflow (008)        Activity.attributes → settings.Attribute / options (002)
plan.modelId → settings.Model (002): allowed types, goal rates      plan.attributes → business units (R9)
```

## 2. Embedded values

### 2.1 Money (`internal/platform/money`, feature 002)

`{"amount": "120000.00", "currency": "USD"}` — the amount is an exact decimal string with two decimals;
planned cost and revenue are always USD (D-010).

### 2.2 Attribute values (feature 002)

`attributes` is 002's `AttributeValues`: one entry per attribute id with its value (single option,
several options, options with shares for "multiple choice with % split", text, number or yes/no). A share
is a decimal with two places; a fresh selection of *n* options is split evenly to two decimals with the
remainder on the last option (R10: 3 options → 33.33, 33.33, 33.34). Shares need not total 100.00 to
save; a total other than 100.00 is listed by the ready-to-submit check.

**Visibility** (R9): an activity-level attribute with a business-unit condition is *shown* only when the
plan's business unit (plan-level attribute, 004) includes one of its units. Hidden and inactive
attributes never count as missing; hidden values stay stored.

## 3. Status

Stored values and the values shown (domain model §2.3):

| Stored `status` | Shown (`displayStatus`) | Colour (design system §4) | Set by |
|---|---|---|---|
| `created` | Created | neutral grey | create, copy, re-approval save (006); recall (008); restore from the recycle bin (006 `PrepareRestore`) |
| `submitted` | Submitted | amber | submit / re-submit (008, through `SetStatus`) |
| `declined` | Declined | red | decline (008) |
| `approved` | Approved — or **In progress** (today ≥ start) — or **Complete** (not ongoing and today > end) | green / blue / slate | last-level approval (008) |

```
create · copy · restore from the bin
        │
        ▼
     Created ──Submit (008)──────────► Submitted ──Approve last level (008)──► Approved
        ▲                               │  ▲  │                                  │
        ├──── Recall (008) ─────────────┘  │  └──Decline (008)──► Declined        │
        │                                  └──── Re-submit (008) ───┘            │
        ├──── re-approval save (006), from Submitted ◄──────────────────────────  │
        └──── re-approval save (006), from Approved ◄────────────────────────────┘
```

| From | To | Trigger | Feature |
|---|---|---|---|
| (new) | Created | Create, copy | 006 |
| Created | Submitted | Submit | 008 |
| Submitted | Approved | Approve the last level | 008 |
| Submitted | Declined | Decline (comment required) | 008 |
| Declined | Submitted | Re-submit | 008 |
| Submitted | Created | Recall | 008 |
| Submitted, Approved | Created | Confirmed save changing planned cost, start date, end date or ongoing | 006 |
| any | (recycle bin) | Delete; Undo puts it back with the same status | 006 / 001 |
| (recycle bin) | Created | Restore from the recycle bin (R14) | 006 kind / 011 |

- **Re-approval** (FR-027, domain model §2.5, D-013): while stored `submitted` or `approved`, a confirmed
  save that changes `plannedCost`, `startDate`, `endDate` or `ongoing` sets `created`. Other edits never
  change the status (FR-028). A *Declined* activity stays *Declined* when edited.
- **Move** keeps the status (D-023; moves never change dates, D-033).
- **Undo** of a delete restores the status exactly; a later **restore from the recycle bin** sets
  `created` (R14).
- **Display status** uses today = the server clock's UTC calendar date (plan, D-091); no dates →
  shown as stored.

## 4. Derived values (calculated when read, never stored)

| Value | Rule | Shown as |
|---|---|---|
| Campaign, plan, initiative of the activity | Through `initiativeId` | Breadcrumb, `plan`/`campaign`/`initiative` `RecordRef`s |
| Goals (calculated) | `setLaterStagesMyself` false: MQL = round-half-up(INQ × INQ→MQL rate); deals = round-half-up(MQL × MQL→deal rate); revenue = deals × average deal size (USD), with the model's **current** values (domain model §4.5) — 250,000 → 5,000 → 250 → 2,500,000.00 USD with Lubricants 1.0; INQ empty → all empty | MQL, revenue "Calculated" + ⓘ |
| Goals (typed) | `setLaterStagesMyself` true: stored `mql`, `revenue`; deals not shown | "Typed" |
| Unassigned planned cost (Sys_Gen_Balance) | planned cost (0.00 when empty) − sum of the line items' USD planned costs (007); `null` when there is neither a planned cost nor a line item | Header; red when negative |
| Line-item totals | From 007 (`LineItemFacts`): count, USD sum, line items re-spread on a date change, custom spreads kept | Delete dialog, re-approval dialog |
| Ready-to-submit check (`SubmitCheck`) | §4.1 | Ready to submit panel |
| Completeness | `missingFields` = items of the "This activity" group; `lineItemsNeedingAttention` = distinct line items with a problem (domain model §2.6, plan D-109) | Completeness chip |
| Roll-ups for 004/005 | Initiative planned cost = sum of its activities' planned costs (empty counts 0); campaign and plan sum below; approved count = activities shown Approved, In progress or Complete; goal sums (domain model §4.2, §4.5) | Computed by 004/005 from `Rollups` |

### 4.1 Ready-to-submit check

All problems at once, in this order; each has `code`, `field` (the form field to open), optional
`lineItemId`/`lineItemName`, and `message`.

| Group | Code | When | Message |
|---|---|---|---|
| This activity | `missingField` | Owner, Start date, End date (not when ongoing), Planned cost, Approval workflow empty; any shown, active, needed-to-submit attribute empty (not a dependent one whose parent value has no options, D-034) | "<Label> — needed to submit" |
| This activity | `splitIncomplete` | A shown % split attribute with values whose shares do not total 100.00 | "<Label> shares add up to <total> % — they must add up to 100.00 %" |
| This activity | `datesOutsideInitiative` | A date outside the initiative's dates (initiative with both dates; ongoing: start only) | "Dates are outside the initiative's dates (<start> – <end>)" |
| This activity | `typeNotAllowed` | The type is not in the plan's model's allowed types | "Activity type '<type>' isn't allowed by the plan's model '<model>' — choose another type" |
| This activity | `inactiveWorkflow` | The chosen workflow is inactive | "Approval workflow '<workflow>' is no longer active — choose an active one" |
| Line items | `negativeUnassigned` | Unassigned planned cost < 0 (D-016) | "Line items exceed the activity's planned cost by <amount>" |
| Line items | (codes of 007) | Each problem 007 reports (empty needed fields, custom spread not adding up, inactive finance budget item, line-item % split) | "Line item '<name>' (<id>): <problem>" |
| Good to know (advice, never blocks) | `unassignedRemaining` | Unassigned planned cost > 0 (Q-008 default) | "<amount> not yet assigned to a line item" |

`problemCount` = number of items in "This activity" and "Line items"; `ready` = `problemCount` is 0.

## 5. History kind `activity` (registered after 005's `initiative`)

| Part | Value |
|---|---|
| `Name` / `Singular` / `Plural` | `activity` / `activity` / `activities` |
| `Parent` | `initiative` (005); `ChildIDs(initiativeID)` = ids of its activities |
| `Snapshot` | The stored activity; `Name` = activity name; `ParentID` = `initiativeId` |
| `Remove` / `Put` | `Repository.Delete` / `Repository.Put` of the snapshot's data |
| Children | `lineItem` (007) registers with `Parent: "activity"` |
| `PrepareRestore` | Name taken in the initiative: Undo → `name_taken` "Can't undo — there is now another activity called '<name>'. Restore it from the recycle bin instead."; Restore → " (restored)", " (restored 2)", … + note "Renamed to '<new name>' because the name was taken.". Restore (not Undo) with status other than `created` → `created` + note "Status reset to Created because it was restored from the recycle bin." Changed snapshot → `Touch` |

**`Describe`** (labels and readable texts in the activity log, fixed at the time):

| Field | Label | Text |
|---|---|---|
| `name` / `externalName` / `description` | Activity name / External name / Description | as is |
| `activityTypeId` | Activity type | the type's name |
| `ownerId` | Owner | the person's name |
| `startDate` / `endDate` | Start date / End date | `DD MMM YYYY` |
| `ongoing` / `setLaterStagesMyself` | Ongoing (no end date) / Set later stages myself | Yes / No |
| `plannedCost` / `revenue` | Planned cost / Revenue | "120,000.00 USD" |
| `approvalWorkflowId` | Approval workflow | the workflow's name |
| `inq` / `mql` | INQ / MQL | "250,000" |
| `status` | Status | Created · Submitted · Declined · Approved |
| `attributes.<attributeId>` | the attribute's label | option labels joined with ", "; shares "Germany 50.00 %, France 50.00 %" (002 `DescribeValue`) |
| `parent` (moves) | Initiative | "<campaign> › <initiative>" |

Activity-log entries per action: plan, *Activity-log entries*.

## 6. API shapes (contract)

Stored record and API answers are separate types (001 research R10):

| Schema | Built from | Notes |
|---|---|---|
| `Activity` | stored record + derived values | `plan`, `campaign`, `initiative` (`RecordRef`); `activityType`, `approvalWorkflow` (`{id, name, active}`, type also `allowed`); `owner` (`PersonRef`, 003); dates `null` when not set; `goals` (`ActivityGoals`); `status`, `displayStatus`; `unassignedPlannedCost`; `lineItems` (`ActivityLineItemTotals`); `completeness`; `readiness` (`ActivitySubmitCheck`); bookkeeping |
| `ActivitySummary` | stored record + derived values | Plan-tree row: no attributes, goals or readiness list |
| `ActivityForm` | plan context + options (+ defaults for a new activity) | `existingNames` lets the web app repeat the uniqueness check |
| `ActivityDestination` | initiatives of the same plan | `moveProblem`, `copyProblem` (null = allowed) |
| `ActivityCreate` / `ActivityUpdate` | request bodies | Update = every editable field + `version` + `restartApproval` (plan, D-107) |

## 7. Approval workflow — table `approval_workflows` (seed-only until feature 008)

Owned by feature 008, which adds the screens, levels' rules and rounds. Until then the seed writes the two
demo workflows and activities only read them (`activities.NewSeededWorkflows`). Shape (008 may extend it):

| Field | Type | Notes |
|---|---|---|
| `id` | string | Internal id, `WF-1001`, `WF-1002` (plan, D-103) |
| `name` | string | "Lubricants EU approval", "China BC approval workflow" |
| `description` | string | empty |
| `active` | boolean | true |
| `levels` | list of `{name, approverIds}` | Level 1 · Daniel Okoro (U-1003); Level 1 · Carlos Mendes (U-1005), Level 2 · Alex Morgan (U-1001) |
| *bookkeeping* | `record.Meta` | created by `actor.System` |

## 8. Readable IDs

No new planning prefix: activities use `A` (D-019). Proposed internal prefix `WF` for approval workflows
until 008 (or 002's Settings-id convention) decides otherwise.
