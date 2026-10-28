# Data Model: The Approval Engine Approves Any Record Type

Amends 008 data-model §1–§4 and 017 data-model §1–§6. Field names are the Go names; JSON names are
camelCase as the contract shows them.

## §1 Record type (`SubjectKind`)

| Value | Label (screens) | Word (messages) | Plural | Record type of the log | Self-approval |
|---|---|---|---|---|---|
| `activity` | Activity | activity | activities | `activity` | barred while D-302 is on |
| `purchaseRequest` | Purchase request | purchase request | purchase requests | `purchaseRequest` | allowed (013 FR-037) |
| `modifiedPurchaseRequest` | Modified purchase request | purchase request | modified purchase requests | `purchaseRequest` | allowed |

A purchase request is one record with two record types of rounds: `purchaseRequest` (its approval) and
`modifiedPurchaseRequest` (a modification of the approved request).

## §2 Approval workflow (table `approval_workflows`) — amended

| Field | Type | Rule |
|---|---|---|
| `ObjectType` (`objectType`) | `SubjectKind` | Required; default `activity` on create; existing rows `activity`. Changeable only while unused (§7) — else problem on `objectType` "It approves <plural> and is in use, so this can't change. Create a new workflow instead." |

`WorkflowView` adds `objectType` and `objectTypeFixed` (true once used). `WorkflowInput.ObjectType`
(optional, default `activity`; an unknown value is a problem on `objectType`: "Choose what the workflow
approves."). `WorkflowUpdate.ObjectType` (optional pointer). The log labels the field "Approves" with the
label of §1.

## §3 Approval level (table `approval_workflow_levels`) — amended

| Field | Type | Rule |
|---|---|---|
| `ApproverType` | text | adds `owner` ("Record owner") and `recordField` ("Person named on the record") |
| `RecordField` (`recordField`) | text | `""` unless `recordField`; then one of the object type's record fields: `moaApprover` ("MoA approver") for `purchaseRequest` and `modifiedPurchaseRequest`; activity workflows have none |

Normalisation: `owner` and `recordField` levels keep no approver ids and no role, and their mode is always
`any`. Problems (with every other problem):

- `levels.<n>.recordField` — activity workflow: "Level <n>: activities have no field that names an approver.
  Choose another option."; unknown or empty field: "Level <n>: choose the field that names the approver."
- `levels.<n>.appliesWhen` / `autoApproveWhen` on a non-activity workflow with business-unit or activity-type
  parts: "Level <n>: only activity workflows can use business unit and activity type conditions."

`LevelView` adds `recordField` (`{id, label}` or null). Level summary sources: "Record owner",
"<field label>" (e.g. "MoA approver"). Condition texts on non-activity workflows say "amount" instead of
"planned cost" ("amount is at least $100,000.00").

## §4 Approval round (table `approval_rounds`) — amended

| Field | Change |
|---|---|
| `SubjectKind` (`subjectKind`) | new, `SubjectKind`; existing rows `activity` |
| `SubjectID` (`subjectId`) | renamed from `ActivityID` / `activity_id` |
| `Outcome` | adds `superseded` ("Replaced by a new submission") |

Constraints: `UNIQUE (subject_id, number)`; at most one `open` round per `subject_id` (exclusion). A round
level's `Source` text for the new sources: "record owner", "<field label>" (e.g. "MoA approver"); its
`Excluded` reasons `owner`/`submitter` occur only on activity rounds.

`RoundView` adds `kind` (the round's `SubjectKind`).

## §5 Approval history entry (table `approval_actions`) — amended

| Field | Change |
|---|---|
| `SubjectKind`, `SubjectID` | as §4 (renamed `activity_id`) |
| `Action` | adds `superseded`: the round ended because the record was re-submitted (actor: who re-submitted, level: the level it waited at) |
| `SelfApproval` (`selfApproval`) | new bool: a decision on a purchase request by its owner or the round's submitter |

`HistoryEntryView` and `Decision` add `selfApproval`.

## §6 Notification (table `notifications`) — amended

`SubjectKind`, `SubjectID`, `SubjectName` (renamed from `ActivityID`, `ActivityName`). `NotificationView`
adds `subject` (`{kind, label, id, name, exists}`); `activity` stays for activity notifications and is null
for the others. Titles that named the activity name the record type: "<Label> approved", "<Label> declined",
"<Label> approved automatically" (activity titles unchanged).

## §7 Usage of a workflow

A workflow is **used** when an activity names it in its Approval workflow field, or any round (of any
record type) was started with it. `usedByCount` counts those records; `waitingCount` those with an open
round of it; the details list them (`activities` keeps its name: each item's `activity.type` is the record
type `activity` or `purchaseRequest`, and `displayStatus` is the record's status).

## §8 Subject (engine-internal) and the purchase request port

`subject` (unexported) — the record as every engine step sees it:

| Field | From an activity | From a purchase request |
|---|---|---|
| `Kind` | `activity` | the round's kind |
| `ID`, `Name`, `Path`, `OwnerID`, `Meta`, `Access`, `Plan` | the activity's | the PR's |
| `Status`, `DisplayStatus` | stored / display status | PR status (both) |
| `Amount` (USD) | planned cost | PR total (`Amount`) |
| `WorkflowID`, `ModificationWorkflowID` | Approval workflow / — | Approval workflow / for modification |
| `MoAApproverID` | — | MoA approver |
| `StartDate`, `ActivityTypeID`, `TypeName`, `BusinessUnits` | the activity's | — |

`PurchaseRequests` (port) and `PurchaseRequestFacts`: see [contracts/go-api.md](contracts/go-api.md).

### Events reported to a purchase request (`RoundChanged`)

| Engine event | `outcome` | 013 mapping (feature 032) |
|---|---|---|
| A round starts (submit, re-submit, modification) | `open` | Submitted (first round) or In-Review (modification) |
| A level is approved and the round waits at a later level | `open` (again) | In-Review |
| The round is approved (also at submission) | `approved` | Approved, then routing |
| The round is declined | `declined` | Declined (first round) or Approved with last approved values (modification) |
| The round is recalled | `recalled` | Created (first round) or Approved with last approved values (modification) |
| The round is replaced by a re-submission | `superseded` (followed at once by `open`) | keep the record's values |

## §9 Actions offered (`ActionSet`)

| Record type | Submit | Re-submit | Recall |
|---|---|---|---|
| Activity | Created | Declined, Submitted | an open round (owner or submitter with `activities.submit`, or on behalf) |
| Purchase request | Created | Declined; an open round of kind `purchaseRequest` | an open round (owner or submitter, or on behalf) |
| Modified purchase request | Approved and no open round | an open round of kind `modifiedPurchaseRequest` | as above |

Approve and Decline: the open round, for the people of 017 data-model §6; the owner and submitter are
refused only on activities while D-302 is on.

## §10 Migration `00018_approval_subjects.sql`

1. `approval_workflows.object_type` (`'activity'` for existing rows; check of §1 values).
2. `approval_workflow_levels.record_field` (`''`), approver-type check adds `owner`, `recordField`; check that
   `record_field <> ''` exactly when the type is `recordField`.
3. `approval_rounds`: rename `activity_id` → `subject_id`; add `subject_kind`; outcome check adds `superseded`.
4. `approval_actions`: rename `activity_id` → `subject_id`; add `subject_kind`, `self_approval` (`false`);
   action check adds `superseded`; index renamed.
5. `notifications`: rename `activity_id`, `activity_name` → `subject_id`, `subject_name`; add `subject_kind`.
6. Defaults used for the backfill are dropped, so every new row names its kind.
7. `activity_in_scope` recreated on `subject_id` and `subject_kind = 'activity'`.
