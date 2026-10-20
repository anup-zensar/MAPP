# Data Model: Campaigns and Initiatives

**Feature**: `005-campaigns-and-initiatives` · **Plan**: [plan.md](plan.md) · **Contract**:
[contracts/openapi.yaml](contracts/openapi.yaml)

The two records this feature stores, the values it derives from them, their rules and their history
kinds. Conventions (bookkeeping `record.Meta`, timestamps, text lengths counted in Unicode characters
after trimming, `ActorRef`, `RecordRef`, lists never `null`) are those of
[001's data model](../001-app-shell-and-sign-in/data-model.md). Field labels, types and markings come from
[field catalogue §2–§3](../../docs/product/field-catalogue.md); rules from
[domain model §1, §2.2, §2.6, §3, §4.1, §4.2, §4.5, §5](../../docs/product/domain-model.md).

Shared value types used below (owned by other features):

| Type | Owner | Shape |
|---|---|---|
| `Money` | 002 (`backend/internal/platform/money`) | `{"amount": "400000.00", "currency": "USD"}` — exact decimal string, never a float. Budgets and planned costs are always USD (D-010) |
| `AttributeValues` | 002 | The record's Shell attribute values keyed by attribute id; `{}` when none. Values of hidden or inactive attributes are kept (R9) |
| `PersonRef` | 003 | `{"id": "U-1002", "name": "Priya Raman", "initials": "PR", "active": true}` |
| `Date` | 005 (`backend/internal/platform/date`, new) | Calendar date, JSON `"YYYY-MM-DD"`, no time zone |

---

## 1. Campaign — table `campaigns`

A theme inside exactly one plan (field catalogue §2).

| Field | JSON | Type | Marking | Validation (all problems reported in one attempt, FR-005) |
|---|---|---|---|---|
| ID | `id` | string | — | `C-1001`, `C-1002`, … from the readable-ID counter `C`; permanent, never reused (D-019) |
| Plan | `planId` | string | R (implicit) | An existing plan (`P-…`). Set on create; changed only by **Move** (§5) |
| Campaign name | `name` | string | **Required** | Trimmed; 1–120 characters; unique among the campaigns of the same plan, ignoring letter case and surrounding spaces |
| Description | `description` | string | — | Trimmed; 0–2,000 characters; `""` when empty |
| Owner | `ownerId` | string or null | Needed to publish | An existing person. A **newly chosen** owner must be active; an owner already stored stays even when inactive (shown "(inactive)") |
| Start date | `startDate` | Date or null | Needed to publish (with end date) | Both dates or neither; start ≤ end. Outside the plan's year is a **warning** (R4, FR-008). Must contain every initiative's dates (FR-007) |
| End date | `endDate` | Date or null | Needed to publish (with start date) | As above |
| Campaign budget | `budget` | Money (USD) or null | — | Currency `USD`; at most 2 decimals; ≥ 0.00. May exceed the plan's remaining budget (D-015) |
| Shell attributes | `attributes` | AttributeValues | per attribute | Checked by feature 002's attribute rules for level `campaign` with the plan's values (R8, R9): option exists and is active (or already held), type fits, dependent values (Region) belong to a selected parent option (Cluster). Seed set: Cluster (multiple choice, P), Region (multiple choice depending on Cluster, P), Insight session completed, Workshop completed, Global GM sign-off completed (yes/no, default No) |
| *bookkeeping* | `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy` | `record.Meta` | — | As 001 |

Not stored (derived on every read, §4): remaining plan budget, planned cost, budget used, left to plan,
approval progress, timing, completeness, goal sums, counts, the R4 warning.

## 2. Initiative — table `initiatives`

A group of similar work inside exactly one campaign (field catalogue §3).

| Field | JSON | Type | Marking | Validation |
|---|---|---|---|---|
| ID | `id` | string | — | `I-1001`, … from the counter `I` |
| Campaign | `campaignId` | string | **Required** (implicit) | An existing campaign. Set on create; changed only by **Move** (§5). Shown read-only with "Change it with Move" |
| Initiative name | `name` | string | **Required** | Trimmed; 1–120 characters; unique among the initiatives of the same campaign, ignoring letter case and surrounding spaces |
| Description | `description` | string | — | 0–2,000 characters |
| Owner | `ownerId` | string or null | Needed to publish | As campaign |
| Start date / End date | `startDate`, `endDate` | Date or null | Needed to publish | Both or neither; start ≤ end; **inside the campaign's dates whenever the campaign has both dates** (R3, FR-006); must contain every activity's dates — for ongoing activities only the start date (FR-007) |
| Initiative budget | `budget` | Money (USD) or null | — | As campaign |
| Shell attributes | `attributes` | AttributeValues | per attribute | Level `initiative`. Seed set: Growth ambition (multiple choice), Function, Focus area (single choice), FIT4 initiative (yes/no, default No) — all optional |
| *bookkeeping* | | `record.Meta` | | |

The initiative does **not** store its plan: the plan is its campaign's `planId` (relational shape; a
campaign move then needs no change to its initiatives). PostgreSQL later: index `initiatives(campaign_id)`
and `campaigns(plan_id)`.

## 3. Relations

```
Plan (004) 1 ──── * Campaign 1 ──── * Initiative 1 ──── * Activity (006) 1 ──── * Line item (007)
  planId ◄──────────┘   campaignId ◄────────┘     initiativeId ◄──────┘
Campaign.ownerId, Initiative.ownerId ──► Person (001/003)
Campaign.attributes, Initiative.attributes ──► Shell attributes and options (002)
```

- Created top-down: a campaign needs a saved plan; an initiative needs a saved campaign.
- Stored records never hold lists of their children; the API's `Campaign` view adds `initiatives`.
- Deleting removes everything below (to the recycle bin, §7).

## 4. Derived values (computed by the API on every read, never stored)

All money is USD and exact: sums of 2-decimal amounts need no rounding. "Has a value" means not null.
`today` = the API clock's calendar date in UTC (`date.Today(now())`, proposed decision in the plan).
The pure functions live in `backend/internal/rollup` (plan, "Roll-ups").

### 4.1 Budget flows down (domain model §4.1, FR-010)

| Value | JSON (in `budgetAndCost`) | Rule |
|---|---|---|
| Allocated to children | `allocatedToChildren` | Sum of the children's budgets that have a value (campaign: its initiatives; initiative: `0.00` — activities have no budget); `0.00` when none |
| Remaining for children | `remainingForChildren` | `budget − allocatedToChildren`; **null** when the record has no budget ("No budget to allocate from"). May be negative |
| Over-allocated by | `overAllocatedBy` | `−remainingForChildren` when it is below 0.00, else null |
| Parent's remaining budget | `parentRemainingBudget` (top level of the view) | Campaign: plan budget − sum of all campaign budgets of the plan. Initiative: campaign budget − sum of all initiative budgets of the campaign. Null when the parent has no budget. The record's own budget is included in the sum |

The form's live figure (FR-010, "including the amount being typed") is computed by the web app with exact
cent arithmetic: `shown = parentRemainingBudget + savedOwnBudget − typedBudget` (for a new record
`savedOwnBudget` = 0.00; an empty field counts as 0.00). Negative → red with "Over-allocated by X".

### 4.2 Cost flows up (domain model §4.2, FR-011)

| Value | JSON | Rule |
|---|---|---|
| Planned cost (initiative) | `budgetAndCost.plannedCost` | Sum of the planned costs of its activities that have one (from feature 006); **null** when none has one (shown "—"). 0.00 is a real value |
| Planned cost (campaign) | same | Sum of its initiatives' planned costs that are not null; null when all are null |
| Planned cost (plan) | `PlanCampaigns.totals.plannedCost`, `PlanRollup.PlannedCost` | Sum of its campaigns' planned costs; null when all are null |
| Budget used | `budgetAndCost.budgetUsedPercent` | Whole percent = `plannedCost × 100 ÷ budget`, **rounded half up** (124.5 → 125, 99.4 → 99); null when the budget is null or 0.00, or the planned cost is null |
| Over budget | `budgetAndCost.overBudget` | `plannedCost > budget` compared **exactly** (both present; budget 0.00 with planned cost > 0.00 is over budget). Drives the red bar, so 100.4 % shows "100 %" in red |
| Left to plan | `budgetAndCost.leftToPlan` | `budget − (plannedCost or 0.00)`; null when the budget is null; negative when over budget |

**Currency of the answers (D-061, D-220).** Everything above is worked out and answered in USD, which the
forms use. Each campaign and initiative also carries `display` (`ContainerDisplay`: `budget`,
`parentRemainingBudget`, `budgetAndCost`, `goals`) and the tree carries `displayTotals`: the same amounts
converted once into the reader's display currency, half up to the cent (`budgetUsedPercent` and `overBudget`
stay as worked out in USD). The plan tree's rows, the totals row and the details panels show these. A move or
copy destination's `remainingAfter` and `overAllocatedBy` are only shown, so they are answered in the display
currency.

### 4.3 Statuses (domain model §2.2, FR-013, FR-015)

| Value | JSON | Rule |
|---|---|---|
| Approval progress | `approvalProgress` `{state, approved, total}` | `total` = activities below; `approved` = activities whose stored status is Approved (shown as Approved, In progress or Complete). `state`: `noActivities` when total = 0; `allApproved` when approved = total > 0; otherwise `inProgress` (also 0 of n) |
| Timing | `timing` | null when the record has no dates; `notStarted` when today < start; `running` when start ≤ today ≤ end; `finished` when today > end |
| Outside the plan's year (campaigns only) | `outsidePlanYear` | true when start < 1 Jan or end > 31 Dec of the plan's year (R4); never blocks saving or publishing |

Status words and tones (design system §4): No activities — neutral; In progress — review (amber);
All approved — positive; Not started — neutral; Running — active (blue); Finished — done (slate).

### 4.4 Completeness (domain model §2.6, FR-014)

`completeness = {missing: MissingField[], itemsBelowNeedingAttention: integer}`.

`missing` lists the record's **own** empty needed-to-publish fields, in form order, each `{field, label}`:

| Field id | Label | Missing when |
|---|---|---|
| `ownerId` | Owner | `ownerId` is null |
| `startDate` | Dates | start and end are null (they are saved both or neither, so they count as one field) |
| `attributes.<attributeId>` | the attribute's label | the attribute is active, shown (its business-unit condition holds for the plan, R9), marked Needed to publish, empty, and — for a dependent attribute — at least one option exists for the chosen parent options; with no parent value chosen it counts as missing (D-034). Decided by feature 002's attribute rules |

`itemsBelowNeedingAttention` counts the records below that are not complete: for a campaign, its
initiatives with a non-empty `missing`, plus every activity below whose own completeness is not "All
filled in" and every incomplete line item below (from feature 006's activity summaries); for an
initiative, the activities and line items below it. Feature 004's publish check uses the `missing` lists
of every campaign and initiative (plan, `PlanRollup.PublishProblems`).

### 4.5 Goals and counts (domain model §4.5, FR-016)

| Value | JSON | Rule |
|---|---|---|
| Goal sums | `goals {inq, mql, revenue}` | Sum of the activities' INQ, MQL (whole numbers) and revenue (USD) below, as feature 006 reports them (calculated or typed); each null when no activity below has that value |
| Counts | `counts` | Campaign: `{initiatives, activities, approvedActivities, lineItems}`; initiative: `{activities, approvedActivities, lineItems}` — used by rows, details and the delete and copy dialogs |

Until feature 006 is wired, `campaigns.NoActivities` reports no activities: planned cost "—", Approval
progress "No activities", goals "—", counts 0.

## 5. Life cycle

Campaigns and initiatives have **no stored status** (D-012); only the derived values of §4.

```
                    create (name required)                    edit (version must match)
      (none) ─────────────────────────────► stored ◄───────────────────────────────────────┐
                                              │  ├──── move (same year / same plan) ───────►┤ keeps ID, values, children
                                              │  └──── copy ─► new record + copies below (new IDs, R13)
                                              │ delete
                                              ▼
                                         recycle bin ──Undo (≤ 2 min, exact)──► stored (same IDs, versions)
                                              └──────Restore (011; parent must exist; name clash → " (restored)")──► stored
```

**Move** (FR-023 – FR-025, D-033): a campaign → another plan **of the same year** (Draft or Published); an
initiative → another campaign **of the same plan**. The target must not already have a child with that
name (ignoring case and spaces) and, for an initiative, the target campaign's dates (when it has both)
must contain the initiative's dates (when it has both). Nothing below changes: IDs, dates, values,
activity statuses and approvals are kept; roll-ups of old and new parents follow automatically because
they are derived. The moved record gets a new version.

**Copy** (FR-026, FR-027, R13): a new record of the same level with every stored field of the source
(name as chosen in the dialog, `planId`/`campaignId` = the destination), new bookkeeping (the copier is
creator), and a copy of everything below — initiatives with their stored fields and new IDs (names
unchanged), activities and line items through feature 006 (new IDs, Created, no approvals, comments,
history or GSAP values). Dates are never changed by a copy within the year. All or nothing, in one
transaction. Suggested name: `"<name> (copy)"`, then `" (copy 2)"`, `" (copy 3)"`, … — the first one free
in the destination; when it would exceed 120 characters the original part is cut (and trailing spaces
trimmed) so that the suffix fits.

**Plan copy support** (feature 004, FR-027 – FR-028): `CopyPlanContents` copies every campaign and
initiative into a new plan with the **same names** and every date moved by `shiftYears` (0 or 1; 29 Feb →
28 Feb), and returns the old → new IDs so that the activity and line-item levels (006, 007) registered in feature 004's `plans.Level` registry copy their own records. It writes no log entry of its own (004 logs the plan copy).

## 6. Readable IDs

| Prefix | Record | Counter | Example |
|---|---|---|---|
| `C` | Campaign | `counters["C"]` | `C-1001` |
| `I` | Initiative | `counters["I"]` | `I-1001` |

No new prefix. Copies take new numbers; moves and restores keep theirs.

## 7. History kinds (registered in `internal/app`, after feature 004's `plan` kind)

| Part | Campaign | Initiative |
|---|---|---|
| `Name` | `campaign` | `initiative` |
| `Singular` / `Plural` | `campaign` / `campaigns` | `initiative` / `initiatives` |
| `Parent` | `plan` (feature 004) | `campaign` |
| `Snapshot(id)` | the stored campaign; `Name` = name; `ParentID` = `planId` | the stored initiative; `ParentID` = `campaignId` |
| `ChildIDs(parentID)` | ids of the campaigns whose `planId` = parent | ids of the initiatives whose `campaignId` = parent |
| `Remove` / `Put` | delete / write the stored campaign exactly | same for initiatives |
| `Describe` | field labels and readable values (below) | same |
| `PrepareRestore` | name rule within the plan: an **Undo** whose name is now taken by another campaign of the plan is refused (`name_taken`); a **Restore** renames to `"<name> (restored)"`, `" (restored 2)"`, … (011 FR-022), calls `Touch`, returns the note "Renamed to '<new name>' because the name was taken." | same rule within the campaign |

Feature 006 registers `activity` with `Parent: "initiative"`, so deleting a campaign takes its
initiatives, activities and line items in one recycle-bin entry.

`Describe` (readable log values, fixed at the time of the change):

| Field | Label | Text |
|---|---|---|
| `name`, `description` | Name, Description | the text |
| `ownerId` | Owner | the person's name ("Priya Raman") |
| `startDate`, `endDate` | Start date, End date | `DD MMM YYYY` ("01 Mar 2026") |
| `budget` | Budget | "400,000.00 USD" |
| `parent` (moves) | Plan (campaign) / Campaign (initiative) | the parent's name |
| `attributes.<id>` | the attribute's label | option labels joined with ", ", "Yes"/"No", the text (feature 002's describe helper) |

## 8. Activity-log entries

| Person's action | Entries |
|---|---|
| Create a campaign / initiative | 1 × `created` (path: plan; plan › campaign) |
| Save a change | 1 × `changed`, `changes` = every changed stored field from `history.Diff` on the record without `attributes`, plus one change per changed attribute (`field` = `attributes.<id>`) |
| Save without any change | none (no new version) |
| Save from an out-of-date copy | none (`version_conflict`) |
| Move | 1 × `moved`, `changes` = `[{field: "parent", old: <old parent id>, new: <new parent id>}]` with the parents' names as texts; path = the new place |
| Copy | 1 × `copied` on the **new** record, `details` "Copied from 'Helix launch' (C-1001), with 2 initiatives, 7 activities and 15 line items" ("Copied from 'Helix launch' (C-1001)" when nothing was below); no entries for the records below (one entry per business change, SC-004 of 001) |
| Delete | 1 × `deleted` by `history.MoveToBin`, details "with 2 initiatives, 7 activities and 15 line items" |
| Undo / Restore | 1 × `restored` by history |

## 9. Demo data (`backend/seed/seed.go`, after feature 004's plans)

Created through the repositories with ids from the generator, by `actor.System` at the seed time.
Attribute values are looked up by label in feature 002's seeded attributes. People: Alex Morgan U-1001,
Priya Raman U-1002, Daniel Okoro U-1003, Mei Lin U-1004. Plans as seeded by 004: P-1001 FY2026 Lubricants
Europe (budget 1,200,000.00), P-1002 FY2026 Mobility China (budget 800,000.00), P-1003 FY2027 Global
Brand (no budget).

| ID | Name | Parent | Owner | Dates | Budget (USD) | Shell attributes | Shows |
|---|---|---|---|---|---|---|---|
| C-1001 | Helix launch | P-1001 | Priya Raman | 01 Jan – 31 Dec 2026 | 700,000.00 | Cluster Lubricants – Europe; its region; Insight session completed Yes; Workshop completed Yes; GM sign-off No | All filled in; remaining for initiatives 150,000.00 |
| I-1001 | Digital & social | C-1001 | Priya Raman | 01 Jan – 30 Jun 2026 | 300,000.00 | Growth ambition, Function, Focus area: first seeded option; FIT4 No | 006's "Launch film" lives here |
| I-1002 | Print & outdoor | C-1001 | Priya Raman | 01 Jan – 31 Dec 2026 | 250,000.00 | FIT4 No | |
| C-1002 | Rimula fleet | P-1001 | Daniel Okoro | 01 Feb – 31 Dec 2026 | 450,000.00 | Cluster Lubricants – Europe; its region | plan remaining 50,000.00 |
| I-1003 | Events | C-1002 | Daniel Okoro | 01 Mar – 30 Sep 2026 | 200,000.00 | FIT4 No | 012's "Rimula fleet › Events" |
| I-1004 | Retail events | C-1002 | Daniel Okoro | 01 Jun – 31 Dec 2026 | 150,000.00 | FIT4 No | 006's move target |
| C-1003 | Retail growth | P-1002 | Mei Lin | 01 Jan – 31 Dec 2026 | 600,000.00 | **no Cluster, no Region** | "2 fields needed"; 004's publish example |
| I-1005 | Station promotions | C-1003 | Mei Lin | 01 Mar – 30 Sep 2026 | 350,000.00 | FIT4 No | |
| I-1006 | Fleet cards | C-1003 | Mei Lin | **no dates** | 200,000.00 | FIT4 No | "1 field needed"; Timing "—" |
| C-1004 | Brand platform | P-1003 | Alex Morgan | 01 Jan 2027 – **31 Mar 2028** | 300,000.00 | Cluster Lubricants – Europe; its region | R4 warning; plan without budget ("—") |
| I-1007 | Brand film | C-1004 | Alex Morgan | 01 Jan – 30 Jun 2027 | 250,000.00 | FIT4 No | |
| I-1008 | Sponsorships | C-1004 | Alex Morgan | 01 Apr – 31 Dec 2027 | 100,000.00 | FIT4 No | campaign remaining **−50,000.00** (over-allocated) |

After the seed the counters are `C` = 1004 and `I` = 1008 (next IDs C-1005, I-1009). The deleted
initiative of the demo story's recycle bin is created and deleted by feature 011's seed step (it needs
feature 006's activity kind).
