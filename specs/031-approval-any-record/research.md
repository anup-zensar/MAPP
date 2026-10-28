# Research: The Approval Engine Approves Any Record Type

Each entry: decision, rationale, alternatives considered. Sources: client specs 009 and 013
(`context/specs/`), FA C6, B9, E6.1.2 and figure 7, the gap analysis §5D and §6 L1/L2/L3, decisions D-302,
D-320 – D-324, and the code of `internal/approval` (features 008, 017).

## R1 — A round belongs to a subject (kind + record id)

- **Decision**: `Round`, `Action` and `Notification` carry `SubjectKind` and `SubjectID` (replacing
  `ActivityID`); notifications also `SubjectName`. The record id alone is unique across types (prefixes
  `A-`, `PR-`), so numbering (`UNIQUE (subject_id, number)`) and the one-open-round exclusion stay keyed on
  `subject_id`: a purchase request's approval and modification rounds share one sequence and never overlap.
- **Rationale**: client 009 FR-039 (earlier rounds viewable newest first, per record) and the 013 status
  model (a modification starts only from Approved). Keeping the constraints on the id keeps the schema as
  strict as today.
- **Alternatives**: a separate table per record type (duplicates every query); keying constraints on
  `(subject_kind, subject_id)` (would allow an open approval round and an open modification round of the
  same purchase request at once, which 013 forbids).

## R2 — Migration: rename, add, backfill by default (00018)

- **Decision**: forward-only `00018_approval_subjects.sql`: rename `activity_id` → `subject_id` in
  `approval_rounds`, `approval_actions`, `notifications` (and `activity_name` → `subject_name`); add
  `subject_kind text NOT NULL` with default `'activity'` for the existing rows, then drop the default so
  every insert names its kind; add `approval_actions.self_approval boolean NOT NULL` (existing rows
  `false`); add `approval_workflows.object_type` (existing `'activity'`) and
  `approval_workflow_levels.record_field`; widen the `outcome`, `action` and `approver_type` checks; recreate
  `activity_in_scope` on `subject_id` and `subject_kind = 'activity'`.
- **Rationale**: `ADD COLUMN … DEFAULT` and `RENAME` do not fire row triggers, so the write-once trigger on
  `approval_actions` (00016) is untouched; every invariant stays a constraint (B5).
- **Alternatives**: keep the column name `activity_id` (misleading for purchase requests forever); update
  rows with `UPDATE` (refused by the write-once trigger).

## R3 — The status hand-off: `RoundChanged` events

- **Decision**: the engine owns the activity's status mapping (as today) and hands purchase requests the
  events through `PurchaseRequests.RoundChanged(ctx, a, id, kind, outcome)`: `OutcomeOpen` when a round
  starts **and again after every decision that approves a level while later levels remain** (the round stays
  open; 013's status model shows the PR In-Review, 009 FR-034 "level completed"); `OutcomeApproved`,
  `OutcomeDeclined`, `OutcomeRecalled` and `OutcomeSuperseded` when it ends. A partial approval of an "all
  must approve" level changes nothing and reports nothing. A re-submission reports `OutcomeSuperseded` for
  the ended round and then `OutcomeOpen` for the new one, in the same transaction; a round approved at
  submission reports `OutcomeOpen` then `OutcomeApproved`. `RoundChanged` returns the purchase request as
  stored now, so the log records its old and new status.
- **Rationale**: the brief fixes the port's signature; the interface addition agreed with the 032 planner
  (coordinator, 2026-10-27) asks for `OutcomeOpen` after a decision that leaves the round open.
- **Alternatives**: a distinct reported outcome `levelApproved` (first draft; replaced by the agreed
  addition); a fifth parameter (breaks the agreed signature).

## R4 — Internal `subject` replaces `ActivityFacts` inside the engine

- **Decision**: one unexported struct `subject` (kind, id, name, status, display status, path, owner, the
  person a record field names, amount in USD, workflow ids, start date, record meta, access and plan facts,
  activity type and business units) built by `activitySubject(ActivityFacts)` or
  `purchaseSubject(kind, PurchaseRequestFacts)`. Every engine step (submission check, resolution, authority,
  notifier, log, rows, escalations) takes a `subject`. The activity routes keep their types
  (`ActivityApproval`, `Submission`, `Preview`) by converting the generic `Section`.
- **Rationale**: one code path for every record type (YAGNI-compliant: two real implementations exist with
  032); activity behaviour stays byte-identical.
- **Alternatives**: a `Subject` interface with methods per record type (more indirection than a value); a
  second engine for purchase requests (duplication).

## R5 — Self-approval by record type

- **Decision**: the access rule (D-302) is applied only when `subject.Kind == KindActivity`: in resolution
  (owner/submitter left out), in authority (owner/submitter refused), and in escalation (backup approver who
  owns or submitted). For purchase-request kinds a decision by the owner or the round's submitter sets
  `Action.SelfApproval`, shown as "self-approval" in the history and the log details.
- **Rationale**: D-324; 013 FR-037 and US2 AS8 ("recorded as self-approval"); FA C6 double MOA.
- **Alternatives**: a per-workflow "allow self-approval" switch (client 009 FR-003) — not asked for now,
  and it would overlap D-302; recorded as a later option.

## R6 — New approver sources

- **Decision**: `SourceOwner = "owner"` (candidate: the record's owner) and `SourceRecordField =
  "recordField"` with `Level.RecordField` naming the field; the only field is `moaApprover` ("MoA
  approver"), offered for purchase-request and modification workflows. Both are single-person sources, so
  their mode is always `any` (as the line manager). An empty owner or field is a submission problem: "Level
  <n>: <field> is empty on this <record type>." The round level's `Source` text is "record owner" or "MoA
  approver", so the chain reads "Carlos Mendes (MoA approver)".
- **Rationale**: client 009 approver types Record Owner and Custom Attribute ("resolved from a field on the
  record", FA E6.1.2); 013's MoA approver; 009 US5 AS4 (empty attribute refuses the submission).
- **Alternatives**: free-text field names (no validation possible); offering the record field on activity
  workflows (activities have no person field).

## R7 — Conditions per object type

- **Decision**: purchase-request workflows may use the amount parts only, compared with the PR's total in
  USD and worded "amount is at least / below"; business-unit and activity-type parts are refused there. The
  log's text of a changed levels list keeps the activity wording (history's describe hook has no record
  context) — a cosmetic limit recorded here.
- **Rationale**: 009 FR-008 ("record's amount (Activity planned cost; PR total)"); `PurchaseRequestFacts`
  carries no business units or activity type.

## R8 — Object type fixed once used

- **Decision**: "used" = a record names the workflow (activities' Approval workflow field) or any round
  was started with it. Purchase requests' workflow fields are not visible to the engine (the port has no
  such read), so a PR that names a workflow without a round does not lock it; such a PR meets the "approves
  activities — choose a purchase request workflow" problem at submission (FR-008). `WorkflowView` carries
  `objectTypeFixed` so the form shows the reason without recomputing the rule.
- **Rationale**: 009 FR-012 (used workflows), the agreed port; the server stays the source of truth.

## R9 — Re-submit from Submitted

- **Decision**: `submittable` accepts Submitted (activity) and the PR rules of FR-024. Inside the
  submission's single Write, before the new round is stored, the open round ends with `OutcomeSuperseded`
  at its current level and gets an `ActionSuperseded` history entry by the person re-submitting. A decision
  on it afterwards is refused with "The submission was re-submitted by <person>" (409 with time). The
  activity stays Submitted (no status change in the log), or becomes Approved when the new round needs no
  decision. Only the new round's waiting approvers are notified. The `Submission` answer adds `replaces`
  (round number, level, label) so the dialog states the consequence (FR-018).
- **Rationale**: D-323, client 009 FR-017 and US1 AS6; 013 FR-033 "Save and re-submit".
- **Alternatives**: recall then submit (two log entries and a Created moment that never existed in Hive9).

## R10 — Responses stay backward compatible

- **Decision**: additive contract changes only. `ApprovalRow` gains `subject` (kind, label, id, name,
  status, display status) and its `activity` becomes nullable (null only for rows of other record types,
  which cannot exist before 032); `Notification` gains `subject` and its `activity` becomes nullable in the
  same way; `ApprovalRound` gains `kind`; enums widen (`outcome`, `action`, `approverType`, `group`); new
  optional properties elsewhere. The web app reads `subject`.
- **Rationale**: B6 (additive changes stay compatible); every activity answer is unchanged in content.

## R11 — Who may recall and submit purchase requests in the engine

- **Decision**: recall of a PR round: the PR's owner or the round's submitter, or someone allowed to decide
  on behalf in the PR's plan; submit availability is not limited by a 014 permission in the engine (032's
  route table and permissions do that). Approve/decline use 017's authority for every kind.
- **Rationale**: 013 FR-034 ("the owner"); 032 owns PR permissions (gap analysis §7 Access).

## R12 — Seeded purchase-request workflows

- **Decision**: feature 031 seeds **no** purchase-request workflow. Feature 032, which seeds the purchase
  requests, adds "Purchase request approval" (object type purchase request; level 1: Person named on the
  record — MoA approver; backup approver Alex Morgan) and "Purchase request modification approval" (modified
  purchase request; level 1: Record owner; backup Alex Morgan) — recommended values.
- **Rationale**: 032's end-to-end journey needs workflows of both kinds, and Shell's as-is PR form is approved
  by the MoA approver while owners approve their own forms (FA C6, D3 1.4). Seeding them here would change
  `seed.Loaded` in `backend/seed/seed.go`, a file several features of this round edit; leaving it to the
  feature that seeds purchase requests keeps the seed consistent and the merge small. 031's own journey
  creates a purchase-request workflow through Settings.

## R13 — Home keeps activity rows

- **Decision**: Home's "Waiting for me" adapter skips rows of other record types until 032 decides how Home
  shows them (it links rows to activities).
- **Rationale**: no behaviour change for activities; avoids broken links.

## R14 — What the running app showed (T040)

API on 127.0.0.1:18131 with `mapp_f031` freshly seeded, exercised with curl:

- The three seeded workflows answer `objectType: activity`; the two that rounds used are `objectTypeFixed`.
  Rule options list the three object types and the MoA approver field of both purchase-request kinds.
- A purchase-request workflow with a MoA approver level sent with mode `all` was created as
  "Level 1 · MoA approver", mode `any`. The same level on an activity workflow, and changing what the used
  Lubricants EU approval approves, were refused with the messages of FR-011 and FR-005.
- Priya Raman re-submitted the Submitted Trade magazine ads (A-1004): the submission check said `kind:
  resubmit`, round 3, `replaces` round 2 at Level 1; after it, round 3 was open, round 2 `superseded`
  (its last entry "superseded"), the activity still Submitted. Daniel Okoro's approval of round 2 answered
  409 "The submission was re-submitted by Priya Raman" with the time.
- The activity form of I-1001 offered the three activity workflows and not the new purchase-request one;
  approval rows and notifications carried `subject` (kind, label, id).

## Decisions for the log (proposed)

- **D-031-1** (for D-327): rounds, history entries and notifications belong to a subject (record type +
  record id); numbering and the single open round are per record across record types (R1).
- **D-031-2**: `RoundChanged` reports `open` again after a decision that approves a level while the round
  waits at a later level (R3; agreed with the 032 planner).
- **D-031-3**: self-approval of purchase requests is stored as `selfApproval` on the decision (R5).
- **D-031-4**: a workflow's object type is fixed once a record names it or a round used it (R8).
- **D-031-5**: purchase-request workflows use amount conditions only (R7).
