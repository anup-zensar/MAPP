# Feature Specification: The Approval Engine Approves Any Record Type

**Feature Branch**: `feature/031-approval-any-record`

**Created**: 2026-10-27

**Status**: Draft

**Input**: Product owner instruction of 2026-10-27 (constitution 4.4.0; decisions D-323, D-324): the approval
engine of features 008 and 017 must approve purchase requests and modified purchase requests as well as
activities, so that feature 032 can approve purchase requests through it; Re-Submit must work from Submitted
(live Hive9 behaviour, FA figure 7). Client requirements: 009 FR-004, FR-007, FR-016, FR-017, FR-027, FR-034
and the approver types "Record owner" and "Custom attribute"; 013 FR-031 – FR-037 (the engine's part). FA C6,
B9, E6.1.2.

**Builds on** feature 008 ([Activity approvals](../008-activity-approvals/spec.md)) and feature 017
([Approval rules](../017-approval-rules/spec.md)), and on feature 014's access control. **Amends** 008
FR-009 (Re-submit is also offered on a Submitted activity), 008 FR-020 (a re-submission from Submitted
ends the open round), 017 FR-001 (two more approver sources), 017 FR-005 (conditions per object type) and
017 FR-018 / 014 FR-026 (the access rule "Nobody approves an activity they own or submitted" keeps applying
to activities only). **Prepares** feature 032 (purchase requests), which owns the purchase request, its
statuses and its screens; this feature owns the approval engine's side.

Demo people (014 seed): Alex Morgan Administrator · Carlos Mendes Global planner · Daniel Okoro Planner and
Approver (Lubricants Europe) · Priya Raman and Mei Lin Planners · Sara Novak Viewer.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Re-submit an activity that is waiting for approval (Priority: P1)

Priya submitted "Trade magazine ads" for approval. While it waits, she realises the wrong approval workflow
was chosen. She changes the workflow and chooses **Re-submit for approval** on the activity. The dialog tells
her that round 1, waiting at level 1 for Daniel Okoro, ends and a new round starts. She confirms: round 1
ends as "Replaced by a new submission", round 2 starts with the new workflow, and the new approvers are
told.

**Why this priority**: Hive9 offers Re-Submit on a Submitted record (client 009 FR-017, FA figure 7, D-323);
MAPP today forces a recall first, which is one more step and loses nothing in return. Purchase requests
rely on the same behaviour ("Save and re-submit", 013 FR-033).

**Independent Test**: Submit a seeded activity, change its approval workflow, choose Re-submit for approval
on the activity page and confirm; the activity stays Submitted, the earlier round shows "Replaced by a new
submission" and the new round waits at its first level.

**Acceptance Scenarios**:

1. **Given** a Submitted activity whose round 1 waits at level 1, **When** its owner opens the Approval
   section, **Then** **Re-submit for approval** is offered next to **Recall**, to the people who may submit
   it (014).
2. **Given** that activity, **When** the owner chooses **Re-submit for approval**, **Then** before anything
   changes the dialog states "Round 1, waiting at Level 1, ends and round 2 starts at the first level that
   needs a decision." (the level's label), with the new round's levels and approvers as for any submission.
3. **Given** the owner confirms, **Then** in one step round 1 ends with the outcome "Replaced by a new
   submission" at the level it waited at, decisions already made in round 1 stay in its history, round 2
   starts with the workflow the activity has now, the activity stays Submitted, the activity log records
   "re-submitted" with the round numbers, and the approvers of round 2's first waiting level are notified.
4. **Given** round 1 was replaced, **When** an approver who still has round 1 open approves it, **Then** the
   decision is refused with "The submission was re-submitted by Priya Raman" and the time, and nothing
   changes.
5. **Given** the activity has a problem that blocks submitting (for example no workflow), **When** the owner
   re-submits, **Then** nothing changes — round 1 keeps waiting — and every problem is listed at once.
6. **Given** a Declined activity, **When** its owner re-submits it, **Then** it behaves as before this
   feature (a new round, nothing replaced).

---

### User Story 2 - Approval workflows have an object type (Priority: P1)

Alex opens Settings › Approval workflows. Each workflow shows what it approves — Activity, Purchase request
or Modified purchase request — and the list can be filtered by it. Alex creates "PR approval — MoA" for
purchase requests. Activities never offer it, and purchase requests (feature 032) offer only purchase-request
workflows.

**Why this priority**: Client 009 FR-002, FR-004 and FR-016 and D-324: a record may only use a workflow made
for its type; without object types the purchase-request approval of feature 032 cannot be configured.

**Independent Test**: Create a purchase-request workflow in Settings; filter the list on "Purchase request";
open an activity and confirm the Approval workflow field does not offer it.

**Acceptance Scenarios**:

1. **Given** the workflow form for a new workflow, **When** Alex opens it, **Then** **What it approves** is
   asked first with the choices Activity (default), Purchase request and Modified purchase request.
2. **Given** Alex saves "PR approval — MoA" for purchase requests, **Then** the list shows it with
   "Purchase request" in its **Approves** column, and the filter **Approves: Purchase request** shows it
   and no activity workflow.
3. **Given** a workflow that a record names, or with which any round was started, **When** Alex edits it,
   **Then** **What it approves** is shown but cannot be changed, with the reason "It approves activities
   and is in use, so this can't change. Create a new workflow instead."; an attempt through the API is
   refused with that message.
4. **Given** an unused workflow, **When** Alex changes what it approves and saves, **Then** the change is
   saved and logged.
5. **Given** an activity, **When** Priya opens its Approval workflow field, **Then** only active activity
   workflows are offered; saving an activity that names a workflow of another type is refused as for a
   workflow that does not exist.
6. **Given** every workflow existing before this feature, **Then** each approves activities, and every
   activity, round and approval history entry reads exactly as before.

---

### User Story 3 - The record's owner or the person named on it approves (Priority: P2)

Alex gives "PR approval — MoA" one level, **Who approves: Person named on the record — MoA approver**, and
"PR modification approval" one level, **Who approves: Record owner**. When a purchase request is submitted,
the level is approved by the MoA approver named on that purchase request, or by its owner.

**Why this priority**: Client 009 FR-007 (approver types Record owner and Custom attribute) and FA C6:
Shell's purchase requests are approved by the MoA approver typed on the PR. The engine is ready before 032
builds the purchase request, so this follows the P1 stories.

**Independent Test**: With a purchase-request workflow whose level is "Person named on the record — MoA
approver", submit a stand-in purchase request naming Carlos Mendes as MoA approver: the round waits for
Carlos; with an empty MoA approver the submission is refused.

**Acceptance Scenarios**:

1. **Given** the level form, **When** Alex chooses **Who approves**, **Then** the choices are Named people,
   Line manager of the owner, People with a role, **Record owner** and — for purchase-request and
   modified-purchase-request workflows only — **Person named on the record**, which then asks which field
   (**MoA approver**).
2. **Given** a level "Record owner" or "Person named on the record", **Then** it has no mode choice: one
   person approves it, and its summary reads "Level 1 · Record owner" or "Level 1 · MoA approver".
3. **Given** a purchase request whose MoA approver is Carlos Mendes, **When** it is submitted, **Then** the
   level's approver is Carlos Mendes, shown "Carlos Mendes (MoA approver)", and he is notified.
4. **Given** the MoA approver is empty, **When** the purchase request is submitted, **Then** nothing changes
   and the problem list (with every other problem) says "Level 1: MoA approver is empty on this purchase
   request."
5. **Given** the person found is inactive or cannot approve, **Then** they are left out with the reason,
   and the backup approver approves the level, or — without one — someone allowed to decide on behalf
   decides it (017 behaviour).
6. **Given** an activity workflow, **When** Alex picks "Person named on the record", **Then** saving is
   refused: "Level 1: activities have no field that names an approver. Choose another option."

---

### User Story 4 - Purchase requests are approved with the same engine (Priority: P2)

When feature 032 submits a purchase request, the engine checks the purchase request's own gates with the
workflow, runs the round, tells the purchase request each event so that its status follows, and accepts
self-approval: the PR's owner may approve their own purchase request when the workflow names them.

**Why this priority**: Client 013 FR-031 – FR-037 and 009 FR-034; D-320, D-324. Its end-to-end journey is
feature 032's P1 story, because the purchase request itself is built there.

**Independent Test**: With a stand-in purchase request, submit, approve level by level, decline, recall,
re-submit from Submitted and submit a modification; check the events the purchase request receives, the
rounds and their kinds, and that the owner may approve their own purchase request.

**Acceptance Scenarios**:

1. **Given** a purchase request that fails its own gates (for example its activity is not approved), **When**
   it is submitted, **Then** nothing starts and the gates' problems and the workflow's problems are listed
   together.
2. **Given** a purchase request on a two-level workflow, **When** it is submitted, **Then** the purchase
   request is told that a round started; **When** level 1 is approved, **Then** it is told that a level was
   approved and the round waits at level 2; **When** level 2 is approved, **Then** it is told the round was
   approved.
3. **Given** a workflow whose only approver is the purchase request's owner, **When** the owner submits it
   and approves it, **Then** the approval is accepted and its history says "self-approval", even while the
   access rule "Nobody approves an activity they own or submitted" is on.
4. **Given** an approved purchase request, **When** a modification is submitted, **Then** a round of kind
   "Modified purchase request" runs on the purchase request's Approval workflow for modification, numbered
   after its earlier rounds; a decline or recall of it tells the purchase request so it restores its last
   approved values (032).
5. **Given** a submitted purchase request, **When** it is re-submitted, **Then** its open round ends as
   "Replaced by a new submission" and a new round of the same kind starts at once.
6. **Given** the purchase-request feature is not connected, **When** any purchase-request approval operation
   is called, **Then** it answers that the purchase request does not exist.

---

### User Story 5 - Approvers see what kind of record waits for them (Priority: P2)

On the Approvals page, each row says what it is — Activity, Purchase request or Modified purchase request —
in a **Record type** column, with a filter; notifications and the Monitoring tab name the record type too.
Until feature 032 is connected only activities appear.

**Why this priority**: Client 009 FR-041 – FR-042 (item type column and View By) and FA C6 sign-off grid;
needed once purchase requests arrive, harmless before.

**Independent Test**: Open Approvals with the seed: every row shows "Activity"; the Record type filter offers
only Activity. With a stand-in purchase request waiting, the row shows "Purchase request".

**Acceptance Scenarios**:

1. **Given** the Approvals page, **Then** each row shows its record type, and **Filters › Record type** offers
   only the types present in the tab's rows.
2. **Given** a notification about a purchase request, **Then** it names the purchase request and its type,
   and opening it goes to the purchase request.
3. **Given** the Monitoring tab, **Then** overdue rows show their record type, and **Waiting by level**
   names what each workflow approves.

---

### Edge Cases

- **A record's open round and its workflow's object type**: the object type cannot change once any round was
  started with the workflow, so an open round never runs on a workflow of another type.
- **A purchase request names a workflow of another type or an inactive one**: submitting lists "Approval
  workflow 'X' approves activities — choose a purchase request workflow." or the inactive-workflow problem.
- **Re-submit while a decision is being made**: the first to save wins; a re-submit that arrives after the
  round moved on still starts a new round from the record's state then, and a decision that arrives after
  the re-submission is refused with "The submission was re-submitted by …".
- **Re-submit of an activity approved at submission**: when no level of the new round needs a decision, the
  activity goes from Submitted straight to Approved (017 FR-014) and round 1 still ends as replaced.
- **Owner level on an activity while the access rule is on**: the owner is left out ("owns this activity"),
  so the backup approver approves the level or someone decides on behalf; the workflow form warns nothing,
  because the rule can be switched off.
- **The MoA approver changes after submission**: the round keeps the approver found at submission (rounds
  are frozen, 008 FR-006); a re-submission finds the new one.
- **A record type's label in old notifications**: notifications created before this feature are about
  activities and say so.

## Requirements *(mandatory)*

### Functional Requirements

#### Records and rounds

- **FR-001**: Every approval round, approval history entry and notification MUST belong to one record of
  one record type: **Activity**, **Purchase request** or **Modified purchase request** (a purchase request
  under modification, client 009 FR-004). Everything stored before this feature MUST read as belonging to
  its activity, unchanged.
- **FR-002**: The rounds of one record MUST be numbered together whatever their type (a purchase request's
  approval and modification rounds share one sequence), and a record MUST have at most one open round.
- **FR-003**: Every round MUST record its record type, and the approval section MUST show it for rounds
  that are not activity rounds ("Round 3 · Modified purchase request").

#### Object types of approval workflows

- **FR-004**: An approval workflow MUST have an object type — Activity, Purchase request or Modified
  purchase request — chosen when it is created (default Activity). Workflows existing before this feature
  MUST approve activities. Names stay unique across all workflows.
- **FR-005**: The object type MUST be changeable only while the workflow is unused: no record names it and
  no round was started with it. Otherwise the change MUST be refused with "It approves <type> and is in use,
  so this can't change. Create a new workflow instead.", and the form MUST show the field fixed with that
  reason.
- **FR-006**: Settings › Approval workflows MUST show each workflow's object type in an **Approves** column
  and offer a filter on it next to the status filter.
- **FR-007**: An activity's Approval workflow field MUST offer only active activity workflows, and saving an
  activity naming a workflow of another type MUST be refused like a workflow that does not exist (009
  FR-016). Feature 032's purchase-request fields offer only workflows of their type.
- **FR-008**: Submitting a record MUST list as a problem a workflow of another object type ("Approval
  workflow 'X' approves purchase requests — choose an activity workflow."), together with every other
  problem.
- **FR-009**: The details of a workflow MUST list the records that use it, whatever their type, with their
  status.

#### Who approves

- **FR-010**: A level's approvers MAY be the **Record owner**: the owner of the record being approved (client
  009 approver type Record owner).
- **FR-011**: A level's approvers MAY be the **Person named on the record**: the person a field of the record
  names (client 009 approver type Custom attribute). For purchase requests and modified purchase requests the
  field is **MoA approver**. Activities have no such field, so an activity workflow MUST refuse this choice:
  "Level <n>: activities have no field that names an approver. Choose another option."
- **FR-012**: Both new sources MUST resolve at submission like the other sources (017 FR-011): the person is
  left out when inactive or unable to approve (and, for activities only, when they own or submit it); the
  backup approver then approves the level, or the level waits for someone allowed to decide on behalf. An
  empty owner or field MUST block the submission with "Level <n>: <field> is empty on this <record type>."
  listed with every other problem.
- **FR-013**: Levels approved by the record owner or the person named on the record MUST have one approver,
  so they are always "any one approves", and their summary MUST name the source ("Record owner", "MoA
  approver"); the chain MUST show the person with the source ("Carlos Mendes (MoA approver)").
- **FR-014**: A level's conditions MUST suit the object type: purchase-request and modification workflows
  MAY use the amount conditions only (on the purchase request's total in USD, worded "amount is at least /
  below"); business-unit and activity-type conditions MUST be refused there with "Level <n>: only activity
  workflows can use business unit and activity type conditions."

#### Self-approval

- **FR-015**: The access rule "Nobody approves an activity they own or submitted" (D-302) MUST apply to
  activities only. For purchase requests and modified purchase requests the owner and the submitter MUST be
  kept as approvers and MAY decide (client 013 FR-037, FA C6 double MOA).
- **FR-016**: A decision made by the record's owner or submitter on a purchase request MUST be recorded as
  self-approval in the approval history and the activity log.

#### Re-submit from Submitted

- **FR-017**: **Re-submit** MUST be offered on a Submitted or Declined record (client 009 FR-017), to the
  people who may submit it. Re-submitting a Submitted record MUST, in one step, end its open round with
  the outcome **Replaced by a new submission** at the level it waited at (keeping the decisions already
  made), and start a new round at the first level that needs a decision with the record's workflow now. If
  the new submission has any problem, nothing MUST change.
- **FR-018**: The re-submit confirmation MUST state, before anything changes, which round ends and at which
  level, and that a new round starts — in addition to what any submission shows (017 FR-013).
- **FR-019**: A decision or recall on a round that was replaced MUST be refused with "The submission was
  re-submitted by <person>" and the time (008 FR-018).
- **FR-020**: Re-submitting a Submitted activity MUST keep it Submitted (or make it Approved when no level of
  the new round needs a decision), log "re-submitted" with both round numbers, and notify the approvers of
  the new round's waiting level and their delegates.

#### Purchase requests through the engine

- **FR-021**: Submitting a purchase request or a modification MUST run the purchase request's own
  submission gates (owned by feature 032, client 013 FR-030) with the workflow's problems, and report every
  problem at once; with any problem nothing changes.
- **FR-022**: A purchase request MUST use its **Approval workflow** for its first approval and its
  **Approval workflow for modification** for a modification (client 013 FR-031).
- **FR-023**: The engine MUST tell the purchase request each event of its round — a round started, a level
  was approved and the round waits at a later level, the round was approved, declined, recalled or replaced —
  so that feature 032 sets the purchase request's status (client 009 FR-034, 013 FR-031); the engine writes
  the approval history and the activity log entry of each event.
- **FR-024**: For purchase requests the engine MUST offer: Submit on a Created purchase request; Submit
  (modification) on an Approved one; Re-submit on a Declined one and while a round of the same kind is open;
  Recall to its owner or the round's submitter, or to someone allowed to decide on behalf; Approve and
  Decline to the waiting approvers, their delegates, the backup approver after escalation and someone allowed
  to decide on behalf (017 FR-017), with a comment required to decline (013 FR-035).
- **FR-025**: Until feature 032 is connected, every purchase-request approval operation MUST answer that the
  purchase request does not exist.

#### Approvals page, Monitoring and notifications

- **FR-026**: Each row of the Approvals page MUST carry its record type and its label; the page MUST show a
  **Record type** column and a **Record type** filter offering only the types present in the tab's rows
  (client 009 FR-041, FR-042).
- **FR-027**: The Monitoring tab's overdue rows MUST carry their record type, and **Waiting by level** MUST
  say what each group's workflow approves.
- **FR-028**: Notifications MUST carry the record and its type; titles MUST name the type ("Purchase request
  approved"); opening one MUST go to the record; activity notifications keep their texts.
- **FR-029**: Messages about a record MUST use its type's word ("Only the approvers of level 1 (Carlos Mendes)
  can approve or decline this purchase request.").
- **FR-030**: Every approval event of every record type MUST be written to the record's approval history and
  to the activity log with who, when and the round, as for activities (008 FR-026, 017 FR-035).

### Key Entities *(include if feature involves data)*

- **Record type** (object type): what an approval round and a workflow are about — Activity, Purchase
  request, Modified purchase request — with a label for screens.
- **Approval workflow** (amended): gains its object type, fixed once used.
- **Approval level** (amended): two more approver sources, Record owner and Person named on the record (with
  the record field it reads).
- **Approval round** (amended): belongs to a record and a record type; a new outcome, Replaced by a new
  submission.
- **Approval history entry** (amended): belongs to the round's record; a new action, replaced; a decision
  may be marked self-approval.
- **Notification** (amended): belongs to a record of a record type.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After the upgrade, 100% of the activity approval rounds, history entries and notifications
  stored before it read back unchanged, and every existing approval test and journey passes.
- **SC-002**: Re-submitting a Submitted activity takes one action and one confirmation (from three steps:
  recall, confirm, submit), and the confirmation names the round that ends.
- **SC-003**: An administrator creates a purchase-request workflow approved by the MoA approver in under
  two minutes, without help (design system §10 usability criterion).
- **SC-004**: In acceptance testing, 0 activity Approval workflow fields offer a workflow of another type.
- **SC-005**: In acceptance testing, a purchase request whose only approver is its owner is approved by its
  owner in 100% of attempts, and an activity in the same situation is never approved by its owner while the
  access rule is on.

## Assumptions

- The purchase request record, its statuses, its gates, its screens, its permissions and its HTTP operations
  are feature 032's (D-320). This feature provides the engine and a connection point; until 032 connects it
  purchase-request approvals answer "not found" (FR-025).
- Per-workflow settings of client 009 FR-003 (comment rules, a per-workflow self-approval switch, threshold
  currency) are not part of this feature: self-approval is decided by record type (D-324), the decline
  comment stays required (008), amounts stay in USD (017).
- Accrual requests, attribute-group approvals and campaign or initiative approvals (client 009 FR-004,
  FR-031 – FR-033, FR-035) stay out: accruals are out of scope, and campaigns and initiatives have no
  approval of their own (D-012, D-324).
- Basic and Advanced regimes and the switch between them (client 009 FR-001, FR-013) are not built: MAPP has
  only workflows (gap analysis §5D).
- The new decision `superseded` ("Replaced by a new submission") is the client's "superseded" round outcome
  (009 Key Entities, Approval Round).
- Clarifications below record the choices made from the sources without a product owner in the loop; each
  is listed for confirmation.

## Clarifications

### Session 2026-10-27

No product owner was available in the loop; each point was resolved from the sources (client spec > FA live
> workshop > guide; decision log) and is recorded as a documented assumption for confirmation.

- Q: Do a purchase request's approval rounds and modification rounds share one numbering, and may both be
  open at once? → A: One numbering per record and at most one open round per record (FR-002). Source: 013
  status model (a modification starts only from Approved; Re-submit of a modification makes a new
  modification round), 009 FR-039 (earlier rounds viewable newest first).
- Q: When a Submitted record is re-submitted, are the approvers of the replaced round told? → A: Only the
  approvers of the new round's waiting level (and their delegates) are notified, as for any submission;
  the replaced round shows "Replaced by a new submission" in its history. Source: 009 FR-017 names no
  notification; FR-053 notifies on submission.
- Q: Which field does "Person named on the record" read? → A: For purchase requests and modifications, the
  MoA approver (013 PM Step 6 "MoA/Approver Email", FA C6). Activities have no person field, so the source
  is refused on activity workflows (FR-011). Source: client 009 Approver Types, 013 field list.
- Q: What do conditions mean on purchase-request workflows? → A: Amount conditions only, on the purchase
  request's total in USD (009 FR-008 "PR total"); business unit and activity type are activity facts and
  are refused (FR-014).
- Q: Is a self-approval marked? → A: Yes, "self-approval" in the history and the log (013 US2 AS8, 009 US6
  AS5: "recorded as self-approval").
- Q: Can a workflow's object type change? → A: Only while it is unused (FR-005), because rounds and records
  depend on it; the client's list shows Object Type as a property of the workflow (009 FR-002, FR-003).
- Q: Who may recall a purchase request? → A: Its owner or the round's submitter, or someone allowed to decide
  on behalf (013 FR-034 "the owner"; 014 FR-027 for activities); feature 032's route rules add its own
  permission.
- Q: Does the Approvals page filter by record type on the server? → A: No: like its other filters, the
  Record type filter runs on the tab's rows in the web app (008-P12), offering only the types present.
