# Data Model: Activity Approvals

**Feature**: `008-activity-approvals` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

The records this feature stores, their fields, rules and life cycles, and the fields of other features'
records it reads or changes. Conventions (timestamps, text lengths, `ActorRef`, `RecordRef`,
`record.Meta`, readable IDs, "arrays are never null") are those of
[001's data model](../001-app-shell-and-sign-in/data-model.md). Money is `{amount, currency}` with a
decimal-string amount (feature 002's `money.Money`). Records are relational in shape: one table per
record type (`backend/data/<table>.json`, an object keyed by id), foreign keys by id.

Field markings use the field catalogue legend: **R** required to save · **S** needed to submit · — optional
· calc calculated, read-only.

---

## 1. Approval workflow — table `approval_workflows`, kind `approvalWorkflow`

A named, ordered set of approval levels that an activity is submitted through (domain model §6,
spec FR-001 – FR-008). A Settings record: users identify it by name; its id is internal but readable.

| Field | JSON | Type | Mark | Rules (all checked together, FR-003) |
|---|---|---|---|---|
| ID | `id` | string | calc | `WF-1001`, `WF-1002`, … (counter `WF`); never changes, never reused |
| Name | `name` | text | R | Trimmed; 1–120 characters; unique among **all** workflows (active and inactive) ignoring letter case and surrounding spaces. Messages: "Enter a name." · "Use 120 characters or fewer." · "There is already an approval workflow called '<other name as stored>'. Choose another name." |
| Description | `description` | long text | — | Trimmed; 0–2,000 characters ("Use 2,000 characters or fewer.") |
| Active | `active` | yes/no | calc | `true` on create; changed only by **Set inactive / Set active** (FR-007) |
| Levels | `levels` | ordered list of **Level** | R | At least one ("Add at least one level."). Order = position: the first element is level 1. Added, removed and reordered in the edit panel |
| *bookkeeping* | `version`, `createdAt`, `createdBy`, `updatedAt`, `updatedBy` | `record.Meta` | calc | Version checked on every update and status change (`version_conflict`, D-030) |

**Level** (a value inside the workflow; no id, no table of its own — Proposed decision 008-P11):

| Field | JSON | Type | Mark | Rules |
|---|---|---|---|---|
| Name | `name` | text | — | Trimmed; 0–60 characters ("Level <n>: use 60 characters or fewer."). Empty means the label "Level <n>" |
| Approvers | `approverIds` | list of person ids | R | At least one ("Level <n>: choose at least one approver."); each an existing person ("Level <n>: choose approvers from the list."); no person twice in one level ("Level <n>: <name> is listed twice."); a person may appear in several levels; a person **added** to the level by this save must be active ("Level <n>: <name> is inactive — choose an active person."). A person already in that level of the stored workflow may stay when they become inactive (008-P8) |

`<n>` in messages is the level's position (1-based) in the request. Field keys of the problems:
`name`, `description`, `levels`, `levels.<n>.name`, `levels.<n>.approverIds`, `version`.

**Derived (computed by the API, never stored):**

| Value | JSON | Rule |
|---|---|---|
| Level label | `levels[].label` | The level's name, or "Level <n>" when empty |
| Levels text | `levelsText` | Each level as "<label> · <approver names joined ', '>" (inactive people get " (inactive)"), levels joined " → ": "Level 1 · Carlos Mendes → Level 2 · Alex Morgan" (FR-005) |
| Used by | `usedByCount` | Number of **stored** activities whose Approval workflow field is this workflow **or** that have an approval round with this workflow (distinct activities) |
| Waiting | `waitingCount` | Number of stored activities with an **open** round of this workflow |
| Warnings | `warnings` | One per level whose approvers are all inactive: "Level <n> has no active approver" (edge case) |
| In use | — | `usedByCount > 0`: Delete is refused with `409 in_use` "Used by 1 activity — set it inactive instead." / "Used by <n> activities — set it inactive instead." (FR-008) |

**Life cycle:**

```
 (none) ──create──► Active ──Set inactive (confirmed)──► Inactive
                     │  ▲                                   │
                     │  └────────────Set active─────────────┘
                     └──Delete (only when usedByCount = 0)──► recycle bin ──Undo / Restore──► back
```

- Inactive: not offered in the activity's Approval workflow field (feature 006 reads `active`); activities
  that use it keep it, labelled "(inactive)"; open rounds finish with their copied levels (FR-007).
- An edit never changes an open round: rounds hold their own copy of the levels (FR-006).
- Undo of a delete is refused with `409 name_taken` when another workflow has meanwhile taken the name;
  Restore renames to "<name> (restored)", "<name> (restored 2)", … with the note "Renamed to '<new
  name>' because the name was taken." (same rule as 001's sample folders).

**History kind `approvalWorkflow`** (registered by `approvals.Service.Kinds()`): Singular "approval
workflow", Plural "approval workflows", no parent, no children. `Describe`: `name` → "Name";
`description` → "Description"; `levels` → "Levels", text = levels text; `active` → "Status", "Active" /
"Inactive". `PrepareRestore`: the name rule above.

Activity-log entries (one per change, in the same Write; FR-026):

| Person's action | Entry |
|---|---|
| Create | `created` |
| Edit (version matches, something changed) | `changed` with every changed field (`history.Diff`) |
| Edit that changes nothing | none, no new version |
| Set inactive / Set active | `statusChanged`, verb `deactivated` / `reactivated`, change `active` true → false / false → true |
| Delete / Undo / Restore | `deleted` / `restored` (written by `history`) |

---

## 2. Approval round — table `approval_rounds`

One attempt to get one activity approved (spec Key Entities). Created by **Submit** / **Re-submit**.

| Field | JSON | Type | Rules |
|---|---|---|---|
| ID | `id` | string | `AR-1001`, … (counter `AR`) |
| Activity | `activityId` | string | The activity (`A-…`); never changes |
| Number | `number` | integer ≥ 1 | 1 for the activity's first round, then the activity's highest round number + 1 (FR-020) |
| Workflow | `workflowId` | string | The activity's workflow at submission |
| Workflow name | `workflowName` | text | Copy of the name at submission (history stays true after renames or deletes) |
| Levels | `levels` | ordered list of **RoundLevel** | Copy of the workflow's levels at submission (FR-006, FR-011) |
| Current level | `currentLevel` | integer ≥ 1 | 1 at submission; +1 on each approval of a non-last level (FR-015); after the round ends: the level it ended at |
| Outcome | `outcome` | enum | `open`, `approved`, `declined`, `recalled`, `restarted`, `cancelled` (see life cycle) |
| Submitted at / by | `submittedAt`, `submittedBy` | timestamp, ActorRef | Who chose **Submit** / **Re-submit**, and when |
| Comment | `comment` | long text | The optional submit comment, 0–2,000 characters |
| Ended at | `endedAt` | timestamp or null | null while `open` |
| *bookkeeping* | | `record.Meta` | Every change (level moves on, round ends) calls `Touch` |

**RoundLevel**: `number` (1-based), `name` (copied; "" = "Level <n>"), `approvers` (list of ActorRef:
id and name at submission, in the workflow's order).

**Life cycle** (FR-011, FR-015, FR-016, FR-021 – FR-023):

```
          Submit / Re-submit
 (none) ───────────────────► open (currentLevel = 1)
                              │ ├── Approve level n < m ──► open (currentLevel = n + 1)
                              │ ├── Approve level m ──────► approved
                              │ ├── Decline (any level) ──► declined
                              │ ├── Recall ───────────────► recalled
                              │ ├── re-approval change ───► restarted        (hook from 006 / 007)
                              │ └── activity restored from the recycle bin ─► cancelled
                              │       (an Undo puts the activity back exactly: the round stays open)
```

- At most one `open` round per activity, and the activity is `submitted` exactly while it has one.
- While the activity is in the recycle bin its open round stays stored unchanged (Proposed 008-P3): the
  activity is not stored, so the round appears in no list or count. Undo → the activity is back
  `submitted` with the same open round (D-031). Restore → the activity comes back `created` (006) and the
  round becomes `cancelled`, `endedAt` = the delete time, with a `cancelled` history entry by the person
  who deleted it.
- An `approved` round keeps its outcome when a later re-approval change returns the activity to Created;
  the restart is recorded as a history entry on that round.

## 3. Approval history entry — table `approval_actions`

One step of a round, shown newest first in the Approval section (FR-025). Written once, never changed or
deleted (no `record.Meta`, like 001's log entries — Proposed 008-P4).

| Field | JSON | Type | Rules |
|---|---|---|---|
| ID | `id` | string | `AH-1001`, … (counter `AH`); the number orders entries written in the same millisecond |
| Round | `roundId`, `roundNumber` | string, integer | The round and its number |
| Activity | `activityId` | string | Redundant with the round, for direct lookups |
| Action | `action` | enum | `submitted`, `resubmitted`, `approved`, `declined`, `recalled`, `restarted`, `cancelled` |
| Level | `level` | integer ≥ 0 | The level the action concerns: 1 for submitted/re-submitted; the decided level; the waiting level for recalled, restarted (of a submitted activity) and cancelled; **0** for a restart of an approved activity (no level waits) |
| Level name | `levelName` | text | Copied from the round level ("" = "Level <n>"); "" when `level` = 0 |
| Designated approvers | `approvers` | list of ActorRef | The level's approvers from the round's copy; empty when `level` = 0 |
| Actor | `actor` | ActorRef | Who acted (for `cancelled`: who deleted the activity) |
| On behalf of | `onBehalfOf` | list of ActorRef | For `approved` / `declined` only: the level's approvers when the actor is not one of them (FR-017); otherwise empty |
| At | `at` | timestamp | When it happened (for `cancelled`: the delete time) |
| Comment | `comment` | long text | Submit comment, approve comment, decline comment (required, not blank), recall reason; 0–2,000 characters |
| Changes | `changes` | list of **ChangeText** | `restarted` only: the changed values (FR-022); otherwise empty |

**ChangeText**: `label`, `oldText`, `newText` — copied from the `history.Change` values the saving
feature's log entry described (e.g. "Planned cost", "$50,000.00", "$65,000.00"). The API adds `text`:

| Case | `text` |
|---|---|
| both texts filled | "<label> changed from <oldText> to <newText>" ("Planned cost changed from $50,000.00 to $65,000.00") |
| `oldText` = "Not filled in" | "<label> added: <newText>" |
| `newText` = "Not filled in" | "<label> removed (was <oldText>)" |
| both texts empty (a ready-made sentence, e.g. 007's restart reasons) | "<label>" unchanged ("Line item 'Agency fee' (L-1003) added: 1,200.00 USD") |

The **decline comment is the activity's comment** of domain model §7 (Proposed 008-P5): no separate
comment record exists in this phase; the latest `declined` entry of the activity heads its Approval
section while the activity is Declined (FR-016).

## 4. Notification — table `notifications`

An in-app message to one person (domain model §7, FR-033 – FR-036). Not a business record: no version,
and marking it read writes no activity-log entry (Proposed 008-P9, like D-038).

| Field | JSON | Type | Rules |
|---|---|---|---|
| ID | `id` | string | `NT-1001`, … (counter `NT`) |
| Recipient | `recipientId` | string | A person id; never the actor of the action (FR-036) |
| Kind | `kind` | enum | `approvalNeeded`, `levelApproved`, `approved`, `declined`, `recalled`, `restarted` |
| Title | `title` | text | Fixed at creation (plan, messages table) |
| Text | `text` | text | One line naming the activity and the actor, fixed at creation |
| Activity | `activityId`, `activityName` | string, text | The activity and its name at the time |
| Actor | `actor` | ActorRef | Who caused it |
| At | `at` | timestamp | Same time as the action |
| Read | `read`, `readAt` | yes/no, timestamp or null | `false` / null on creation; set by opening it or **Mark all as read** |

Derived for the API: `activity.exists` (the activity is stored now). A notification about an activity
that is not stored says "This activity was deleted — find it in the recycle bin." and opens the recycle
bin (edge case).

Recipients (one notification per person per action, duplicates removed, the actor removed):

| Action | Recipients and kind |
|---|---|
| Submit, Re-submit | Level-1 approvers → `approvalNeeded` |
| Approve level n < m | Owner → `levelApproved`; level n+1 approvers → `approvalNeeded` |
| Approve last level | Owner → `approved` |
| Decline | Owner → `declined` (text holds the comment's first 80 characters, then "…" when longer) |
| Recall | Owner → `recalled`; approvers of the waiting level → `recalled` |
| Re-approval change (restart) | Owner → `restarted`; if it was Submitted: approvers of the waiting level; if it was Approved: the actors of the `approved` entries of its last approved round → `restarted` |
| Delete, Undo, Restore, time passing (In progress / Complete) | nobody (FR-019; 011 Assumptions) |

"Owner" is the activity's owner (activity field Owner), not the submitter (spec Assumptions); no owner →
nobody in that row.

## 5. Fields of other features' records this feature uses

| Record (owner) | Field | Read / written by 008 |
|---|---|---|
| Activity (006) | `status` (stored: `created`, `submitted`, `declined`, `approved`) | **Written** by Submit, Re-submit, Approve, Decline, Recall (with `record.Meta.Touch`, so the activity gets a new version and "last changed by") |
| Activity (006) | derived display status (`inProgress`, `complete`) | Read (rows, preview, chips) |
| Activity (006) | `approvalWorkflowId` (field catalogue §4.1 #9, **S**) | Read: the workflow a submission uses; usage counts. Locked by 006 while Submitted ("Recall the activity to choose another workflow.", FR-013) or Approved |
| Activity (006) | id, name, version, owner, start/end date, ongoing, planned cost, type, description, unassigned planned cost, completeness, path (plan › campaign › initiative) | Read (rows, preview, notifications, check) |
| Line item (007) | id, name, planned cost (local), planned cost (USD), spend type, GL code, finance budget item | Read (preview, FR-032) |
| Person (001/003) | `name`, `active` | Read: approver names, "(inactive)", validation |

**Activity status transitions made or triggered by this feature** (domain model §2.3):

| From | Event | To | Approval effect |
|---|---|---|---|
| created | Submit (check passes, confirmed) | submitted | new round, number = last + 1 (1 for the first) |
| declined | Re-submit | submitted | new round, next number, current workflow levels (FR-020) |
| submitted | Approve level n < m | submitted | round moves to level n + 1 |
| submitted | Approve level m | approved | round `approved` |
| submitted | Decline | declined | round `declined` |
| submitted | Recall | created | round `recalled` |
| submitted, approved | re-approval change saved (006 / 007) | created | hook: open round `restarted` / restart entry on the approved round |
| (in recycle bin) | Restore (not Undo) | created (006) | hook: open round `cancelled` |

## 6. Roll-ups and counts

| Value | Rule | Used by |
|---|---|---|
| Waiting for me | Stored activities with status `submitted` whose open round's current level lists the signed-in person | Tab, card, sidebar badge (FR-027 – FR-029, SC-007) |
| All open | Stored activities with status `submitted` | Tab, card |
| Approved this month | Rounds of stored activities with outcome `approved` and `endedAt` in [monthStart, monthEnd) | Card (FR-028) |
| Declined this month | Rounds of stored activities with outcome `declined` and `endedAt` in [monthStart, monthEnd) | Card |
| Decided by me | Stored activities with at least one `approved` or `declined` entry whose actor is the signed-in person; the latest such entry is "my decision" | Tab |
| Unread | The signed-in person's notifications with `read` = false (all of them, not only the 50 listed) | Bell badge ("9+" above 9) |
| Workflow usage | §1 derived values | Workflows list and details |

`monthStart` / `monthEnd` are the viewer's local calendar month, sent by the browser (Proposed 008-P13);
without them the API uses the current UTC month.

## 7. Readable-ID counters added

| Prefix | Record |
|---|---|
| `WF` | Approval workflow |
| `AR` | Approval round |
| `AH` | Approval history entry |
| `NT` | Notification |

## 8. Demo data (seed)

Added to `backend/seed/seed.go` after the activities and line items of features 006–007 (FR-037). All
times are offsets from the seed's `now` (tests: 2026-03-12 09:00 UTC).

| Record | Seed |
|---|---|
| `WF-1001` **Lubricants EU approval** | Description "Lubricants Europe activities. The marketing team leader approves."; level 1 (no name): Daniel Okoro (`U-1003`); active |
| `WF-1002` **China BC approval workflow** | Description "Mobility China activities: the business champion first, then the marketing administrator."; level 1: Carlos Mendes (`U-1005`); level 2: Alex Morgan (`U-1001`); active |
| Activity workflow field | Every seeded FY2026 Lubricants Europe activity uses `WF-1001`; every FY2026 Mobility China activity uses `WF-1002` (FY2027 Global Brand activities, if any, stay empty) |
| Lubricants activity seeded **Submitted** (one; 006's seed) | Round 1: submitted by its owner at now − 9 d, declined by Daniel Okoro at now − 8 d ("Please split the agency fee by country before re-submitting."); round 2: re-submitted at now − 2 d, **open, waiting for Daniel Okoro** |
| Lubricants activity seeded **Declined** | Round 1: submitted at now − 6 d, declined by Daniel Okoro at now − 5 d with "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3." |
| Lubricants activities seeded **Approved** (incl. shown In progress / Complete) | Round 1: submitted at now − 40 d, approved by Daniel Okoro at now − 38 d |
| Lubricants / Mobility activities seeded **Created** | No round |
| The complete FY2026 Mobility China activity (seeded **Submitted** by 006/007's seed, or set Submitted here) | Round 1: submitted by Mei Lin at now − 4 d; level 1 approved by Carlos Mendes at now − 2 h; **open, waiting at level 2 for Alex Morgan** |
| Notifications | Daniel: `approvalNeeded` for the round-2 Lubricants activity (unread). Alex: `approvalNeeded` "Carlos Mendes approved level 1 of … — level 2 waits for you" (unread). Mei Lin: `levelApproved` (unread). Owner of the Declined activity: `declined` (unread). Owners of approved activities: `approved` (read). Owner of the round-2 activity: `declined` from round 1 (read) |
| Activity log | One back-dated `statusChanged` entry per seeded approval entry (plan, "Activity-log entries") |
