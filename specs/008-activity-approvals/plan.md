# Implementation Plan: Activity Approvals

**Branch**: `008-activity-approvals` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/008-activity-approvals/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) (fragment,
19 operations) · [tasks.md](tasks.md). Conventions are feature 001's
([plan](../001-app-shell-and-sign-in/plan.md), [research](../001-app-shell-and-sign-in/research.md),
[data model](../001-app-shell-and-sign-in/data-model.md)); this plan only adds what is new.

## Summary

Feature 008 makes an activity go through approval and tells people when something needs them:

- **Go API** — one new business package `internal/approvals` (handler → service → repository, JSON
  store) holding four record types: **approval workflows** (Settings record, CRUD, set inactive, delete
  to the recycle bin), **approval rounds** (one attempt to get one activity approved, with a frozen copy
  of the workflow's levels), **approval history entries** (append-only steps of a round) and
  **notifications**. The service submits, re-submits, approves, declines and recalls activities level by
  level, answers the approvals page (tabs, summary, preview) and the bell, and reacts to two events of
  features 006/007 through hooks: a re-approval change (domain model §2.5) and a restore from the
  recycle bin. It reaches activities and line items through one narrow interface, `approvals.Activities`,
  implemented by an adapter in `internal/app`, so no package imports another in a circle.
- **React web app** — the **Approvals** page (`/approvals`: summary cards, three pill tabs, search,
  filters, preview side panel with Approve / Decline), **Settings › Approval workflows**
  (`/settings/approval-workflows`, standard CRUD pattern with a levels editor), the activity page's
  **Approval section** and status button (dropped into feature 006's page), the working **notification
  bell** (unread badge, list, Mark all as read) and the **sidebar count** on Approvals. New
  notifications and counts appear within one minute by polling every 30 seconds.

Both `[NEEDS CLARIFICATION]` markers are resolved with the decision-log defaults (Q-002, Q-007).

## Technical Context

**Language/Version**: Go 1.27 (API); TypeScript 5.9 strict + React 19 (web app) — as feature 001

**Primary Dependencies**: no new dependency. API: standard library, `northstar/internal/platform/*`,
`history`, `people`, `platform/money` (feature 002). Web app: the libraries of 001 research R12.

**Storage**: JSON file store, four new tables `approval_workflows`, `approval_rounds`,
`approval_actions`, `notifications` (data-model.md); PostgreSQL later through the same repository
interface

**Testing**: Go `testing` (service tests with a fake `Activities` and a temporary store; handler tests
with `apitest` on the seeded app); Vitest + Testing Library (`mockApi`, fake timers); Playwright + axe
(`frontend/e2e/approvals.spec.ts`)

**Target Platform**: developer laptops, Chromium/Edge — as 001

**Project Type**: web application (React SPA + Go REST API)

**Performance Goals**: every approvals call answers in well under 100 ms with demo data; a new
notification is visible within 60 s (poll every 30 s, FR-036)

**Constraints**: servers on `127.0.0.1` only; no personal data in server logs; no third runtime part
(no push channel); WCAG 2.1 AA (SC-008); deterministic tests (fixed clock 2026-03-12 09:00 UTC)

**Scale/Scope**: 6 demo people, tens of activities, hundreds of rounds and notifications. 3 web routes,
1 page section, 1 bell, 19 API operations.

## Constitution Check

*Checked before design and again after it.*

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario, edge case or SC (traceability table). The two `[NEEDS CLARIFICATION]` markers take the decision-log defaults (Q-002, Q-007); every other choice is a Proposed decision below | Pass |
| II. Shell business, plain language | Plan → … → Activity vocabulary; "approval workflow", "level", "round" only as defined in the spec; every user-visible text is in *User-visible messages* once | Pass |
| III. Simple, friendly experience | Standard CRUD pattern for workflows; submit in 2 steps with every problem at once (006's panel); approve from the inbox in 3 steps; nothing disabled without a reason (Recall on Approved explains; Delete of a used workflow explains); status chips everywhere; all-at-once validation | Pass |
| IV. Simple, layered architecture | One package `approvals` (handler → service → repository); all rules in the service; storage through `approvals.Repository` + `store.Transactor`; REST + OpenAPI; generated web types. Cross-package calls go through `approvals.Activities` (adapter in `internal/app`) and `activities.ApprovalHooks` (implemented by approvals) — no import cycle, `internal/app` the only package that knows all (repository-layout rules) | Pass |
| V. Tested business rules | One Go test per rule named with its FR; a handler test per operation; a component test per page and dialog; Playwright journeys for US1–US6 (P1 stories) plus axe; fixed clock everywhere | Pass |
| VI. Safe by default | Actor on every service call (reads too); every approval action and workflow change logged in the same Write; decisions and submissions refuse stale views (`version_conflict`); deletes to the recycle bin with Undo; notifications hold names and activity names only (no e-mail); nothing personal in server logs | Pass |
| VII. Keep it small | No new dependency; one new shared component (`CountBadge`, reason below); notifications live in `approvals` (only approvals notify in this phase); levels stored inside the workflow record; inbox filtering in the browser; polling instead of a push channel | Pass |

**Post-design re-check**: Pass. The two seams (port + hooks) and the choices that add code are in
*Complexity Tracking*.

## Decisions

### Clarifications resolved with the decision-log defaults

| Spec marker | Decision | Source |
|---|---|---|
| FR-004 — levels by planned-cost threshold? | **No thresholds.** Every level of the workflow applies to every submission, in order; any one designated approver of a level approves it. Hive9's approver types, delegates and amount thresholds are not built. | Q-002 default |
| FR-012 — must the plan be Published first? | **No.** Activities of Draft and Published plans can be submitted alike; the plan's status is not checked. | Q-007 default |

`/speckit-clarify` should record both in spec.md (FR-004, FR-012) and the checklist; the consolidation
step does that.

### Proposed decisions (numbered by the consolidation step)

| Tag | Decision | Reason |
|---|---|---|
| 008-P1 | **D-155**: Approval decisions name the round and level they decide (`roundId`, `level`); the first decision on a level counts. A later Approve / Decline / Recall on a level or round that has moved on answers `409 version_conflict` whose message says what happened ("Level 1 was already approved by Daniel Okoro") and whose `conflict` says who and when; the web app adds the time. Submit uses the activity's `version`. | FR-018 needs "what happened", not "someone changed this"; an unrelated edit (description) must not block an approver |
| 008-P2 | **D-156**: Approval status changes are changes of the activity record: Submit, Approve, Decline and Recall set its `status` with `record.Meta.Touch` (new version, "last changed by") and write a `statusChanged` log entry on the activity. | Status is a stored activity field (domain model §2.3); D-030 must protect it like any other field |
| 008-P3 | **D-157**: Deleting a Submitted activity leaves its open round stored unchanged; the activity simply leaves every approvals list and count because it is no longer stored. **Undo** brings it back Submitted with the same open round (D-031). A later **Restore** from the recycle bin returns it Created (006) and ends the round as "Cancelled — activity deleted", dated at the delete and attributed to the person who deleted it. | FR-023 and D-031 together: Undo must restore open approvals exactly, so the delete itself cannot end the round; the round's end is only visible once the activity exists again |
| 008-P4 | **D-158**: Approval history entries are append-only (no version, never edited); workflows and rounds embed `record.Meta`. | History entries are facts, like 001's log entries; D-030 applies to records people edit |
| 008-P5 | **D-159**: A decline comment is stored once, on its approval history entry; it is "the comment on the activity" of domain model §7. There is no separate comment record in this phase. | No feature lists or adds free comments; one place avoids two copies drifting |
| 008-P6 | **D-160**: Notifications are fetched by the web app every 30 seconds (and when the window regains focus); there is no push channel. | FR-036 asks for one minute; a push channel would be a third runtime part (constitution IV) |
| 008-P7 | **D-161**: Restart notifications (FR-035) go to the owner and — for a Submitted activity — to the approvers of the level that was waiting, or — for an Approved activity — to the people who made the approvals of its last approved round. | The spec's "the people who approved it" names actors, not designated approvers |
| 008-P8 | **D-162**: An approver who becomes inactive may stay in a workflow level (shown "(inactive)", details warn "Level n has no active approver" when nobody active is left); only adding an inactive person to a level is refused. | FR-003 (inactive person refused) and the spec's edge case (inactive designated approver stays) together; otherwise renaming a workflow would fail because of an unrelated level |
| 008-P9 | **D-163**: Notifications are not business records: no version, and opening one or **Mark all as read** writes no activity-log entry. | Same reasoning as D-038 (display state) |
| 008-P10 | **D-043**: New error code `in_use` (409) for deleting a configuration record that is in use ("Used by 3 activities — set it inactive instead."). Shared with feature 002 (models, types, attributes in use). Same decision as D-043, which delivers `errs.InUse` first (build order); consolidation records it once and 008 only reuses it. | The existing codes all mean something else; the web app must recognise the refusal |
| 008-P11 | **D-164**: A workflow's levels are stored inside the workflow record as an ordered list (no level table, no level ids); a round stores a frozen copy of them. PostgreSQL may later map them to a child table inside the repository. | Levels have no identity or history of their own and are always read and saved with their workflow |
| 008-P12 | **D-165**: The approvals page loads all rows of the selected tab and searches/filters them in the browser; counts come from the server's summary. | Demo-sized data; instant filters; one simple endpoint |
| 008-P13 | **D-166**: "This month" on the approvals page is the viewer's local calendar month, sent by the browser as `monthStart` / `monthEnd`. | Spec Assumption ("viewer's local time"); only the browser knows the time zone |
| 008-P14 | **D-167**: Workflow status changes are logged with the verbs "deactivated" and "reactivated" ("Alex Morgan deactivated approval workflow 'China BC approval workflow'"); a level approval that does not finish the round is logged with the verb "approved level <n> of". | 011 FR-006: actions added by later features use their own plain verb in "<person> <verb> <record type> '<name>'" |
| 008-P15 | **D-168**: Approval workflows get the internal readable ids `WF-`; rounds `AR-`; approval history entries `AH-`; notifications `NT-`. If feature 006 was delivered with its seed-only workflows `WF-1001` / `WF-1002` (D-103, same table `approval_workflows`), 008 keeps 006's counter `WF` and those ids instead of `AW`, so no seeded activity changes its workflow id; every `AW` in this feature's documents then reads `WF` (see *Reconciliation*). | Every stored record needs an id from the counters (001 R4); prefixes must not clash |
| 008-P16 | **D-169**: Level names hold at most 60 characters; workflow descriptions and all approval comments at most 2,000 characters; a decline notification quotes the first 80 characters of the comment. | Limits the spec leaves open, chosen like plan descriptions (field catalogue §1 #7) |

## Project Structure

### Documentation (this feature)

```text
specs/008-activity-approvals/
├── spec.md · checklists/requirements.md
├── plan.md              # this file (decisions included; no separate research.md)
├── data-model.md        # records, rules, life cycles, seed
├── contracts/openapi.yaml   # fragment merged into backend/api/openapi.yaml
└── tasks.md
```

### Source code

```text
backend/
├── api/openapi.yaml                       # + the fragment; Error.code + in_use           (shared, extended)
├── internal/
│   ├── approvals/                         # NEW
│   │   ├── model.go          # stored types + API types
│   │   ├── repository.go     # Repository interface
│   │   ├── store_json.go     # JSON implementation (4 tables)
│   │   ├── ports.go          # Activities interface + ActivityFacts, ReadyCheck, Problem, LineItemFacts
│   │   ├── messages.go       # every API message of this package
│   │   ├── service.go        # Service, NewService, shared helpers (people lookups, levels text)
│   │   ├── workflows.go      # workflow CRUD, set (in)active, delete, usage
│   │   ├── kinds.go          # history kind approvalWorkflow (Describe, PrepareRestore)
│   │   ├── rounds.go         # submission preview, submit, approve, decline, recall
│   │   ├── view.go           # ActivityApproval (chain, history, viewer, actions)
│   │   ├── hooks.go          # ApprovalRestarted, ActivityRestored, LookupWorkflow, ApproverUsage, LatestDeclines
│   │   ├── inbox.go          # tabs, summary, preview
│   │   ├── notifications.go  # notify (recipients, texts), list, mark read
│   │   ├── handler.go        # 19 routes
│   │   ├── fake_test.go      # fakeActivities implementing Activities
│   │   ├── workflows_test.go · kinds_test.go · rounds_test.go · view_test.go · hooks_test.go
│   │   ├── inbox_test.go · notifications_test.go
│   │   └── handler_test.go   # package approvals_test, apitest on the seeded app
│   ├── app/
│   │   ├── approvals_adapter.go (+ _test.go)  # NEW: approvals.Activities over 006/007; 006 seam adapters (Reconciliation)
│   │   └── app.go                 # wiring                                                    (shared, extended)
│   ├── history/kinds.go · service.go            # RestoreContext gains DeletedAt, DeletedBy (001 core, additive)
│   ├── platform/errs/errs.go      # CodeInUse, InUse(message)                                  (001, additive)
│   └── activities/                # ApprovalHooks + WorkflowLookup seams, if 006 did not deliver them (006, additive)
└── seed/seed.go · seed_test.go    # workflows, rounds, history, notifications                  (shared, extended)

frontend/src/
├── api/schema.d.ts                         # regenerated
├── components/CountBadge/                  # NEW shared component (+ test, export, gallery entry)
├── lib/format.ts · format.test.ts          # + formatRelativeTime, formatWaiting                  (001, extended)
├── app/
│   ├── navigation.ts                       # Approvals leaves COMING_SOON_SECTIONS; NavItem.Badge  (shared)
│   ├── router.tsx                          # /approvals, /settings/approval-workflows(/:workflowId) (shared)
│   ├── Sidebar.tsx (+ test)                # renders item.Badge                                   (001)
│   └── NotificationBell.tsx (+ css, test)  # real notifications                                   (001)
├── features/approvals/                     # NEW
│   ├── approvalsApi.ts       # queries, mutations, keys, polling, invalidation
│   ├── messages.ts           # web texts + alreadyDecidedMessage, submitConflictMessage, changeText
│   ├── ApprovalsPage.tsx (+ .module.css, .test.tsx)
│   ├── ApprovalsTable.tsx (+ test)
│   ├── ApprovalsFilters.tsx (+ test)
│   ├── ApprovalPreviewPanel.tsx (+ css, test)
│   ├── ApprovalSection.tsx (+ css, test)        # used by 006's activity page
│   ├── ApprovalStatusAction.tsx (+ test)        # header button used by 006's activity page
│   ├── ApprovalChain.tsx (+ css, test)
│   ├── ApprovalHistory.tsx (+ css, test)
│   ├── SubmitDialog.tsx (+ test) · DecisionDialog.tsx (+ test) · RecallDialog.tsx (+ test)
│   ├── useApprovalSummary.ts · ApprovalsNavBadge.tsx (+ test)
│   └── useMonthRange.ts (+ test)
├── features/approval-workflows/            # NEW (Settings › Approval workflows)
│   ├── workflowsApi.ts
│   ├── ApprovalWorkflowsPage.tsx (+ css, test)
│   ├── WorkflowDetailsPanel.tsx (+ test)
│   ├── WorkflowFormPanel.tsx (+ test)
│   └── LevelsEditor.tsx (+ css, test)
├── features/notifications/                 # NEW
│   ├── notificationsApi.ts
│   └── NotificationList.tsx (+ css, test)
├── features/activities/ActivityPage.tsx    # drop in ApprovalSection + ApprovalStatusAction       (006)
└── features/settings/…                     # one section card "Approval workflows"                (002)

frontend/e2e/approvals.spec.ts              # NEW: main journeys US1–US6 + axe
```

**Structure decision**: the web-application layout of `docs/architecture/repository-layout.md`
(`approvals/` is listed there for 008). Workflows get their own web folder because the Settings frame
belongs to feature 002.

## Backend design

### Package `approvals`: dependencies and wiring

```go
// ports.go — what approvals needs from activities (006) and line items (007).
// internal/app implements it (approvals_adapter.go); service tests use fakeActivities.
// Every method runs inside the caller's transaction (the context carries it).
type Activities interface {
    Activity(ctx context.Context, id string) (ActivityFacts, bool, error) // false: not stored
    List(ctx context.Context) ([]ActivityFacts, error)                    // every stored activity
    // SetStatus stores status with record.Meta.Touch(a, now) — a new version — and writes no log entry.
    SetStatus(ctx context.Context, a actor.Actor, id, status string) (ActivityFacts, error)
    // ReadyCheck runs the ready-to-submit check of 006/007 (006 FR-023, 007 FR-028).
    ReadyCheck(ctx context.Context, a actor.Actor, id string) (ReadyCheck, error)
    LineItems(ctx context.Context, a actor.Actor, activityID string) ([]LineItemFacts, error) // in order
}

type ActivityFacts struct {
    ID, Name              string
    Version               int
    Status                string             // StatusCreated | StatusSubmitted | StatusDeclined | StatusApproved
    DisplayStatus         string             // Status, or DisplayInProgress / DisplayComplete (006 FR-021)
    Path                  []history.RecordRef // plan, campaign, initiative — top level first
    OwnerID               string             // "" when empty
    StartDate, EndDate    string             // "YYYY-MM-DD", "" when empty (EndDate "" when ongoing)
    Ongoing               bool
    PlannedCost           *money.Money       // USD; nil when empty
    UnassignedPlannedCost *money.Money       // nil when the planned cost is empty
    TypeName, Description string
    WorkflowID            string             // the Approval workflow field; "" when empty
    Missing               []MissingField     // 006 completeness: empty needed-to-submit fields {ID, Label}
}
type ReadyCheck struct {
    Problems              []Problem    // every blocking problem (FR-010 a–d, dates, type, line items)
    UnassignedPlannedCost *money.Money // set only when positive (note, never blocks)
}
type Problem struct{ Field, Group, LineItemID, Message string } // Group "activity" | "lineItem"
type LineItemFacts struct {
    ID, Name                             string
    PlannedCost                          money.Money  // local currency
    PlannedCostUSD                       *money.Money
    SpendType, GLCode, FinanceBudgetItem string       // readable labels, "" when empty
}
const (
    StatusCreated, StatusSubmitted, StatusDeclined, StatusApproved = "created", "submitted", "declined", "approved"
    DisplayInProgress, DisplayComplete = "inProgress", "complete"
)
```

`NewService(tx store.Transactor, repo Repository, people people.Repository, activities Activities,
generator *ids.Generator, hist *history.Service, now clock.Clock) *Service`; `(*Service).Kinds()
[]history.Kind`; `NewHandler(s *Service) *Handler`; `(*Handler).Routes(r *web.Router, signedIn web.Wrap)`.

**Wiring in `app.New`** (after activities and line items): build `approvalsAdapter{activities service +
repository, line-items service, now}`; `approvals.NewService(db, approvals.NewJSONRepository(),
peopleRepo, adapter, generator, historyService, now)`; register `Kinds()` with history; call
`activitiesService.SetApprovalHooks(approvalsService)` and `activitiesService.SetWorkflowLookup(approvalsService)`;
`approvals.NewHandler(…).Routes(router, signedIn)`.

**Seams with features 006/007** (declared in package `activities`; if 006 did not deliver them, task
T019 adds them exactly so):

```go
// activities.ApprovalHooks — implemented by *approvals.Service.
type ApprovalHooks interface {
    // Inside the Write that saved a re-approval change (domain model §2.5) on a Submitted or Approved
    // activity, after its status was set to Created and its log entry written; changes are that log
    // entry's described changes (label, oldText, newText). 007's line-item changes come through 006.
    ApprovalRestarted(ctx context.Context, a actor.Actor, activityID, previousStatus string, changes []history.Change) error
    // Inside a Restore from the recycle bin (never an Undo), from the activity kind's PrepareRestore, after
    // it reset the status to Created. Returns notes for the "restored" log entry.
    ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) ([]string, error)
}
// activities.WorkflowLookup — implemented by *approvals.Service; 006 uses it to validate and label the
// Approval workflow field and for the ready-check problem "approval workflow inactive" (FR-010 e).
type WorkflowLookup interface {
    LookupWorkflow(ctx context.Context, id string) (name string, active, found bool, err error)
}
```

**001 core changes (additive)**: `history.RestoreContext` gains `DeletedAt time.Time` and `DeletedBy
actor.Actor`, filled by `putBack` from the bin entry (008-P3 needs the delete's time and person);
`errs.CodeInUse = "in_use"` and `errs.InUse(message string) *Error` (409, empty `fields`).

### Repository (`repository.go`)

| Method | Meaning |
|---|---|
| `GetWorkflow(ctx, id) (Workflow, bool, error)` · `ListWorkflows(ctx) ([]Workflow, error)` · `PutWorkflow(ctx, Workflow) error` · `DeleteWorkflow(ctx, id) error` | table `approval_workflows` |
| `GetRound(ctx, id) (Round, bool, error)` · `ListRounds(ctx) ([]Round, error)` · `RoundsOf(ctx, activityID) ([]Round, error)` (by number) · `PutRound(ctx, Round) error` | table `approval_rounds` |
| `PutAction(ctx, Action) error` · `ListActions(ctx) ([]Action, error)` · `ActionsOf(ctx, activityID) ([]Action, error)` (oldest first, by id number) | table `approval_actions` |
| `GetNotification(ctx, id) (Notification, bool, error)` · `NotificationsOf(ctx, personID) ([]Notification, error)` (newest first) · `PutNotification(ctx, Notification) error` | table `notifications` |

Stored types (`model.go`): `Workflow{ID, Name, Description, Active, Levels []Level, record.Meta}`,
`Level{Name, ApproverIDs []string}`, `Round{ID, ActivityID, Number, WorkflowID, WorkflowName, Levels
[]RoundLevel, CurrentLevel, Outcome, SubmittedAt, SubmittedBy, Comment, EndedAt *time.Time, record.Meta}`,
`RoundLevel{Number, Name, Approvers []actor.Actor}`, `Action{ID, RoundID, RoundNumber, ActivityID,
Action, Level, LevelName, Approvers []actor.Actor, Actor, OnBehalfOf []actor.Actor, At, Comment,
Changes []ChangeText}`, `ChangeText{Label, OldText, NewText}`, `Notification{ID, RecipientID, Kind,
Title, Text, ActivityID, ActivityName, Actor, At, Read, ReadAt *time.Time}` — JSON names as in
data-model.md. API types (separate, as 001 R10): `WorkflowView`, `WorkflowPage`, `WorkflowDetails`,
`WorkflowInput`, `WorkflowUpdate{Name, Description *string; Levels *[]LevelInput; Version *int}`,
`StatusChange{Version *int}`, `ActivityApproval`, `RoundView`, `ChainLevel`, `HistoryEntryView`,
`Submission`, `SubmitInput{Version *int; Comment string}`, `DecisionInput{RoundID string; Level *int;
Comment string}`, `RecallInput{RoundID, Reason string}`, `Row`, `RowPage`, `Summary`, `Preview`,
`NotificationView`, `NotificationPage`, `ReadAll` — one per contract schema.

### Service operations and their rules

Every method takes `(ctx, a actor.Actor, …)`; writes run in one `s.tx.Write` with their log entries and
notifications; reads in `s.tx.Read`. "Gone" = `404 not_found` with the messages-table text.

| Operation | Rules (FR) | Errors |
|---|---|---|
| `ListWorkflows(ctx, a) ([]WorkflowView, error)` | Sorted by name ignoring case; each with levels (labels, approvers with live name/initials/active), levels text, usage counts, warnings (FR-005, edge case) | — |
| `GetWorkflow(ctx, a, id) (WorkflowDetails, error)` | + the activities counted in `usedByCount` with path, display status, waiting flag; sorted by plan name, then activity name (FR-005) | gone |
| `CreateWorkflow(ctx, a, WorkflowInput) (WorkflowView, error)` | Trim; validate all at once (FR-002, FR-003, data-model §1); unique name; id `WF`; active; log `created` | `validation_failed` |
| `UpdateWorkflow(ctx, a, id, WorkflowUpdate)` | Version required ("Reload the page and try again.") and checked; same validation on the merged values; inactive approvers already in that level allowed (008-P8); no change → no version, no log; else `Touch`, `changed` entry with `history.Diff`. Rounds are never touched (FR-006) | `validation_failed`, `version_conflict`, gone |
| `SetWorkflowActive(ctx, a, id, active bool, version *int)` | Version checked; same value → unchanged, no log; else `Touch`, `statusChanged` verb `deactivated` / `reactivated`, change `active` (FR-007). Open rounds untouched | `validation_failed` (version), `version_conflict`, gone |
| `DeleteWorkflow(ctx, a, id) (history.BinEntry, error)` | Refused when `usedByCount > 0` (FR-008); else `history.MoveToBin(kind approvalWorkflow)` | `in_use`, gone |
| `Submission(ctx, a, activityID) (Submission, error)` | Changes nothing. `kind` = `resubmit` when Declined; `available` only for Created/Declined (else reason "Only Created or Declined activities can be submitted for approval."); problems = `ReadyCheck` problems + **workflow problem** when the activity's workflow is not found ("Approval workflow — needed to submit", field `approvalWorkflowId`, unless 006 already listed that field) or inactive ("Approval workflow '<name>' is inactive — choose an active one.") (FR-010 e); levels = the workflow's current levels; positive unassigned planned cost as note (FR-011, edge case); next round number | gone |
| `Submit(ctx, a, activityID, SubmitInput) (ActivityApproval, error)` | Comment ≤ 2,000; version required and must equal the activity's (D-030); status Created or Declined (FR-009) else `bad_request`; re-run the check of `Submission` — any problem → `validation_failed` with one field per problem (`field` = problem field, message = problem message) and **nothing changes** (FR-010); plan status not checked (FR-012, Q-007). Then: new `Round` (number = max + 1, `workflowName` and level copy with approver ids and names now, current level 1, `open`, submitter, time, comment; FR-011, FR-020); `SetStatus(submitted)`; history entry `submitted` / `resubmitted` (level 1, its approvers, comment); log `statusChanged` verb `submitted` / `re-submitted` (FR-026); notify level-1 approvers (FR-035) | `validation_failed`, `bad_request`, `version_conflict`, gone |
| `Approve(ctx, a, activityID, DecisionInput)` | Comment ≤ 2,000; `roundId` and `level` required ("Reload the page and try again."); the round must belong to the activity (else 404 "This approval round doesn't belong to this activity. Reload the page and try again."); it must be `open` with `currentLevel == level`, else `version_conflict` built from the decisive history entry: "Level <n> was already approved by <actor>" · "Level <n> was already declined by <actor>" · "The submission was recalled by <actor>" · "Approval was restarted by <actor>" · "The approval round was cancelled because the activity was deleted", with `conflict{changedBy: actor name, changedAt: entry time}` (FR-018). `onBehalfOf` = the level's approvers when `a` is not one of them (FR-017, D-020). Level < m: `currentLevel++`, `Touch`; entry `approved`; log verb `approved level <n> of` (no status change); notify owner `levelApproved`, level n+1 approvers `approvalNeeded` (FR-015, FR-035). Level = m: round `approved`, `endedAt`; `SetStatus(approved)`; entry; log verb `approved` with status change; notify owner `approved` | as listed |
| `Decline(ctx, a, activityID, DecisionInput)` | As Approve for round/level checks and on-behalf; comment trimmed must not be blank ("Add a comment explaining why the activity is declined.") and ≤ 2,000; round `declined` at any level; `SetStatus(declined)`; entry `declined` with comment; log verb `declined`, comment = the comment; notify owner `declined` (FR-016, FR-035) | as Approve |
| `Recall(ctx, a, activityID, RecallInput)` | Reason ≤ 2,000; `roundId` required; Approved activity → `bad_request` with the FR-021 explanation; round must be `open` (else `version_conflict` as above); round `recalled`; `SetStatus(created)`; entry `recalled` (waiting level, reason); log verb `recalled`, comment = reason; notify owner + waiting-level approvers `recalled` (FR-021, FR-035). Anyone may recall (D-020) | as listed |
| `Approval(ctx, a, activityID) (ActivityApproval, error)` | Rounds newest first with chain levels (state: `approved` / `declined` from the level's decision entry with its `decision`; `waiting` = the current level of an open round; `notDecided` = the level that was waiting when the round was recalled, restarted or cancelled; `notReached` = every level after the current one) and history newest first; `upcomingLevels` from the current workflow when there is no round; `latestDecline` while Declined (FR-016, FR-024, FR-025); `viewer` (owner, current approver, on-behalf names); `actions`: submit (Created), resubmit (Declined), approve/decline/recall (Submitted), recall on Approved → `available false` + FR-021 reason | gone |
| `Inbox(ctx, a, tab) ([]Row, error)` | `waiting`: Submitted, open round's current level lists `a`, oldest `submittedAt` first; `open`: every Submitted, oldest first; `decided`: activities with an `approved`/`declined` entry by `a`, latest decision first, with `myDecision` and the latest round (FR-029, FR-030). Only stored activities (FR-023). `waitingSince` = the round's submission time for level 1, else the time of the previous level's approval; `startDatePassed` = Submitted and start date < today (clock date, UTC; edge case) | `validation_failed` (tab) |
| `Summary(ctx, a, monthStart, monthEnd *time.Time) (Summary, error)` | Counts of data-model §6; both or neither bound ("Reload the page and try again." on `monthStart`), `monthEnd` after `monthStart`; default the clock's UTC month (FR-027, FR-028, SC-007) | `validation_failed` |
| `Preview(ctx, a, activityID) (Preview, error)` | Activity facts, `LineItems`, unassigned planned cost, `Approval` (FR-032) | gone |
| `ListNotifications(ctx, a) (NotificationPage, error)` | Recipient = `a`; newest first (time, then id number), at most 50; `total`, `unread` over all; `activity.exists` from `Activities.Activity` (FR-033, edge case) | — |
| `MarkRead(ctx, a, id) (NotificationView, error)` | Only `a`'s own (else 404 "This notification no longer exists."); already read → unchanged (FR-034) | gone |
| `MarkAllRead(ctx, a) (ReadAll, error)` | Every unread of `a` → read, `readAt` now (FR-034) | — |
| `ApprovalRestarted(ctx, a, activityID, previousStatus, changes)` (hook) | Submitted: the open round → `restarted`, `endedAt`; entry `restarted` at the waiting level with `changes` (label/oldText/newText). Approved: entry `restarted` (level 0) on the last `approved` round, which keeps its outcome. No log entry (006/007's save logged it). Notify per 008-P7 (FR-022, FR-035). No round → nothing | — |
| `ActivityRestored(ctx, rc, activityID) ([]string, error)` (hook) | An open round → `cancelled`, `endedAt = rc.DeletedAt`; entry `cancelled` by `rc.DeletedBy` at `rc.DeletedAt` at the waiting level; returns the note "Approval round <n> was cancelled because the activity was deleted — submit it for approval again." No notifications (FR-023, 008-P3) | — |
| `LookupWorkflow(ctx, id)` | Name, active, found — for 006 (FR-007, FR-013) | — |
| `ApproverUsage(ctx, personID) (int, error)` | Number of workflows with the person in any level — for 003's person details and Deactivate dialog (003 FR-004/005) | — |
| `LatestDeclines(ctx, a) (map[string]DeclineNote, error)` | Latest decline per Declined activity — for 009's "needs attention" | — |

Notifications are written by one helper `notify(ctx, a, recipients []string, kind, activity ActivityFacts,
text string)` that removes `a.ID` and duplicates (FR-036) and creates one `NT` record per recipient in the
caller's Write.

### Activity-log entries (FR-026, SC-004)

All on kind `activity` (registered by 006), action `statusChanged`, in the action's Write. Status changes
carry `changes: [{field: "status", old, new}]` (labels and texts from 006's `Describe`).

| Action | Verb | Status change | Comment | Details |
|---|---|---|---|---|
| Submit | `submitted` | Created → Submitted | submit comment | "Round <r> · <workflow> · waiting for level 1 (<approvers>)" |
| Re-submit | `re-submitted` | Declined → Submitted | submit comment | same |
| Approve level n < m | `approved level <n> of` | none | comment | "Round <r> · level <n> of <m> · now waiting for level <n+1> (<approvers>)" + "; on behalf of <names>" when so |
| Approve level m | `approved` | Submitted → Approved | comment | "Round <r> · level <m> of <m>" (+ on behalf) |
| Decline | `declined` | Submitted → Declined | decline comment | "Round <r> · level <n> of <m>" (+ on behalf) |
| Recall | `recalled` | Submitted → Created | reason | "Round <r> · was waiting for level <n> (<approvers>)" |
| Restart | — (006/007's `changed` entry includes Status → Created) | | | |
| Cancel on restore | — (note in the `restored` entry's details) | | | |

Workflow changes: data-model §1 table.

### Errors

Shared format and codes of 001 R9, plus `in_use` (008-P10). Approval actions on a moved-on round use
`version_conflict` with a specific message (008-P1). Bodies that cannot be read: `bad_request` (001).

## API operations

All require a session (`401 sign_in_required` / `session_expired` otherwise). Schemas are in
[contracts/openapi.yaml](contracts/openapi.yaml).

| # | Method and path | Request | Response | Errors |
|---|---|---|---|---|
| 1 | `GET /api/approval-workflows` | — | 200 `ApprovalWorkflowPage` | — |
| 2 | `POST /api/approval-workflows` | `ApprovalWorkflowCreate` | 201 `ApprovalWorkflow` | 400 |
| 3 | `GET /api/approval-workflows/{workflowId}` | — | 200 `ApprovalWorkflowDetails` | 404 |
| 4 | `PATCH /api/approval-workflows/{workflowId}` | `ApprovalWorkflowUpdate` | 200 `ApprovalWorkflow` | 400, 404, 409 `version_conflict` |
| 5 | `DELETE /api/approval-workflows/{workflowId}` | — | 200 `DeleteResult` | 404, 409 `in_use` |
| 6 | `POST /api/approval-workflows/{workflowId}/deactivate` | `ApprovalWorkflowStatusChange` | 200 `ApprovalWorkflow` | 400, 404, 409 |
| 7 | `POST /api/approval-workflows/{workflowId}/activate` | `ApprovalWorkflowStatusChange` | 200 `ApprovalWorkflow` | 400, 404, 409 |
| 8 | `GET /api/activities/{activityId}/approval` | — | 200 `ActivityApproval` | 404 |
| 9 | `GET /api/activities/{activityId}/approval/submission` | — | 200 `ApprovalSubmission` | 404 |
| 10 | `POST /api/activities/{activityId}/approval/submit` | `SubmitRequest` | 200 `ActivityApproval` | 400 `validation_failed` (problems) / `bad_request` (status), 404, 409 |
| 11 | `POST /api/activities/{activityId}/approval/approve` | `DecisionRequest` | 200 `ActivityApproval` | 400, 404, 409 |
| 12 | `POST /api/activities/{activityId}/approval/decline` | `DecisionRequest` | 200 `ActivityApproval` | 400, 404, 409 |
| 13 | `POST /api/activities/{activityId}/approval/recall` | `RecallRequest` | 200 `ActivityApproval` | 400, 404, 409 |
| 14 | `GET /api/approvals?tab=waiting\|open\|decided` | — | 200 `ApprovalRowPage` | 400 |
| 15 | `GET /api/approvals/summary?monthStart&monthEnd` | — | 200 `ApprovalCounts` | 400 |
| 16 | `GET /api/approvals/activities/{activityId}` | — | 200 `ApprovalPreview` | 404 |
| 17 | `GET /api/notifications` | — | 200 `NotificationPage` | — |
| 18 | `POST /api/notifications/{notificationId}/read` | — | 200 `Notification` | 404 |
| 19 | `POST /api/notifications/read-all` | — | 200 `NotificationReadAll` | — |

Handlers follow `samples/handler.go`: `web.ReadJSON` → `actor.From` → service → `web.WriteJSON` /
`web.WriteError`; the delete answers `history.DeleteResult{RecycleBinEntry: entry.View()}`; query
parameters are read with `r.URL.Query()` (`tab` default `waiting`; `monthStart`/`monthEnd` parsed with
`time.Parse(time.RFC3339, …)`, a bad value → `validation_failed` on that field).

## Demo data (seed)

`backend/seed/seed.go` gains `addApprovals(ctx, generator, now)` after the activities and line items of
006/007, exactly as [data-model.md §8](data-model.md#8-demo-data-seed): the two workflows (`WF-1001`,
`WF-1002`), the workflow field of the FY2026 activities, rounds, history entries and notifications per
seeded status (including the Declined activity with Daniel Okoro's comment, the round-2 Lubricants
activity waiting for Daniel, and the complete Mobility China activity waiting at level 2 for Alex Morgan
after Carlos Mendes approved level 1 at now − 2 h, i.e. "on 12 Mar 2026" in tests). Rounds, entries and
notifications are written through `approvals.NewJSONRepository()`; ids from the generator. For each
seeded history entry the seed writes the matching back-dated `statusChanged` activity-log entry through a
`history.Service` built on a clock the seed moves to the entry's time (with the planning kinds registered
as `app.New` does — reuse the seed's back-dated-history helper if 004–007 created one). If 006's seed
already created the two workflows (D-103), 008 keeps their records and ids (`WF-1001`,
`WF-1002`, 008-P15) and only fills in the descriptions and bookkeeping of data-model §8; otherwise it
creates them as `WF-1001` and `WF-1002`.
If no complete Mobility China activity exists in 006/007's seed, 008's seed task completes the one named
there and sets it Submitted. The seed summary line adds ", 2 approval workflows".

## Frontend design

### Routes and navigation

| Path | Page | Tab title |
|---|---|---|
| `/approvals` (`?tab=waiting\|open\|decided`, default `waiting`; `?activity=A-1042` opens the preview) | `ApprovalsPage` | Approvals · NorthStar |
| `/settings/approval-workflows` | `ApprovalWorkflowsPage` | Approval workflows · NorthStar |
| `/settings/approval-workflows/:workflowId` | `ApprovalWorkflowsPage` with the details panel open | Approval workflows · NorthStar |

- `navigation.ts`: `APPROVALS` becomes a plain `NavItem` (out of `COMING_SOON_SECTIONS`); `NavItem` gains
  optional `Badge?: ComponentType<{ collapsed: boolean }>`; `APPROVALS.Badge = ApprovalsNavBadge`.
  `Sidebar.tsx` renders `<item.Badge collapsed={collapsed} />` inside the link after the label (in the
  rail: a small badge on the icon). The badge text is inside the link, so its name becomes "Approvals 3
  waiting for you".
- Feature 002's Settings page gets one section card: "Approval workflows" · "Who approves activities,
  level by level." → `/settings/approval-workflows`.
- Feature 009's Home links use `/approvals?tab=waiting&activity=<id>` (provided).

### Server state (`approvalsApi.ts`, `workflowsApi.ts`, `notificationsApi.ts`)

| Query key | Call | Notes |
|---|---|---|
| `['approvals', tab]` | `GET /api/approvals?tab=` | |
| `['approvals-summary', monthStart]` | `GET /api/approvals/summary` with `useMonthRange()` | `refetchInterval: 30_000`, shared by cards and sidebar badge |
| `['approval-preview', activityId]` | `GET /api/approvals/activities/{id}` | |
| `['approval', activityId]` | `GET /api/activities/{id}/approval` | Approval section |
| — (`fetchQuery`, `staleTime: 0`) | `GET /api/activities/{id}/approval/submission` | on **Submit for approval** click |
| `['approval-workflows']` / `['approval-workflows', id]` | list / details | |
| `['notifications']` | `GET /api/notifications` | `refetchInterval: 30_000`, `refetchOnWindowFocus: true` (FR-036) |

After every approval action (submit, approve, decline, recall) and every workflow change the app calls
`invalidateAllButSession(queryClient)` = `queryClient.invalidateQueries({ predicate: q => q.queryKey[0]
!== 'session' })`: statuses change plan, campaign and initiative approval progress, activity pages, lists
and counts of several features, whose keys this feature does not own. `useMonthRange()` returns the local
calendar month's first instant and the next month's first instant as ISO strings.

### Approvals page (US3, FR-027 – FR-032)

- `PageHeader`: eyebrow "Approvals", title "Approvals", description from the messages table; no primary
  action.
- Four `SummaryCard`s (with tips): Waiting for me (amber `tone` when above zero), All open, Approved this
  month, Declined this month — from the summary query; tabs and filters never change them (FR-028).
- Toolbar: pill tabs (004's `PillTabs`) **Waiting for me** · **All open** · **Decided by me**, the first
  two with a `CountBadge`; search field "Search activity name or ID" (matches name, ignoring case, or ID,
  ignoring case); **Filters** button (popover: Plan, Approval workflow, Owner — choices built from the
  tab's rows — and Submitted from / to dates); removable chips "Plan: FY2026 Lubricants Europe" …;
  **Clear filters**; count "<n> of <m>". Changing tab keeps search and filters.
- `ApprovalsTable` in a `Card` (scrolls sideways inside the card on narrow screens): columns Activity
  (name as a button that opens the preview + ID; red `StatusChip` "Start date passed" when so), Plan ›
  campaign, Owner (avatar + name), Planned cost (right-aligned, display currency through 003's helper),
  Workflow ("(inactive)" when so), Level ("Level 1 of 2 · Carlos Mendes"), Submitted (date + "waiting 3
  days"); **Decided by me** adds My decision ("Approved · 12 Mar 2026") and Status (006's status chip) and
  shows Level only for open rounds. Order as served.
- **Preview** (`ApprovalPreviewPanel`, a `SidePanel`; URL `?activity=`): header name, description "<ID> ·
  Waiting at level <n> of <m>" (or the status); body: status chip + `CompletenessChip`
  (`missing`), Plan › campaign › initiative, Owner, Dates (or "Ongoing from <start>"), Planned cost,
  Activity type, Description ("—" when empty); **Line items** table (Line item, Planned cost (local),
  Planned cost (USD), Spend type, GL code, Finance budget item; "—" for empty; "No line items yet.");
  "Unassigned planned cost" row (red when negative); `ApprovalChain` and the latest 3 history entries
  (`ApprovalHistory` compact). Footer: **Approve** (primary), **Decline** (secondary), **Open activity**
  (link to 006's activity page `#approval`) — Approve/Decline only while Submitted.
- After a decision: success toast (messages table); `invalidateAllButSession`; the preview moves to the
  next row of the current filtered list after the decided one (or the first row); when none is left it
  closes (FR-032). On `version_conflict`: toast `alreadyDecidedMessage(error)`, refetch, dialog closes.

### Activity page parts (US1, US2, US5, FR-009, FR-013, FR-014, FR-024, FR-025)

`ApprovalStatusAction` (in 006's header, replacing 006's Submit stub; props `activity: {id, name, status,
version}`, `onShowProblems(count: number)` = 006's function that shows "Submitting needs <n> things —
view list" and opens the Ready to submit panel): Created → **Submit for approval**; Declined →
**Re-submit for approval**; Submitted → **Approve** (opens `DecisionDialog`); Approved (incl. In progress,
Complete) → nothing. Submit flow: `fetchQuery` the submission → not `ready` → `onShowProblems(problems.length)`
(006's panel lists them; FR-010) → else `SubmitDialog` → POST → toast; a `validation_failed` answer →
`onShowProblems(fields.length)` after invalidating; `version_conflict` → `submitConflictMessage`.

`ApprovalSection` (in 006's Approval section, element `id="approval"`, replacing 006's placeholder):
`Card` "Approval" with subtitle "Round <n> · <workflow name>" when a round exists:
1. Declined: the latest decline box (problem tint): "Declined by <actor>[ on behalf of <names>] on
   <date, time>" and the comment (FR-016).
2. Status chip and "Approval workflow: <name>" ("(inactive)") or "No approval workflow chosen yet." (The
   Submitted banner of design system §6.6 is 006's, FR-027 of 006.)
3. `ApprovalChain` of `rounds[0]`, or "Not submitted yet" with `upcomingLevels`: one row per level —
   badge "L1", approvers (avatar + name, "(you)" for the signed-in person, "(inactive)"), state text.
4. Actions (from `actions`): Submit / Re-submit (same flow as the header), Approve, Decline, **Recall
   submission** (`RecallDialog`); Approved: the recall `reason` as text (FR-021).
5. `ApprovalHistory`: "Approval history", rounds newest first ("Round 2 · Waiting at level 1 of 1 ·
   Lubricants EU approval"), entries newest first: time, action label, level label, "Approver(s):
   <names at the time>", "<actor> on behalf of <names>", comment in quotes, change sentences (FR-025).

Dialogs (all `ConfirmDialog`, focus on Cancel):
- `SubmitDialog`: workflow, ordered levels, unassigned note, round note for re-submit, optional comment
  (`TextAreaField`, max 2,000), confirm **Submit for approval** / **Re-submit for approval** (FR-011).
- `DecisionDialog` (`mode: 'approve' | 'decline'`): level line, next-level or last-level line (approve),
  "You own this activity." when `viewer.isOwner`, the on-behalf sentence when `!viewer.isCurrentApprover`
  (FR-017), comment optional (approve) or **Required** (decline; empty → field error, nothing sent;
  FR-016); confirm **Approve** / **Decline activity** (danger).
- `RecallDialog`: explanation, optional reason, confirm **Recall submission**.

### Settings › Approval workflows (US4, FR-001 – FR-008)

- `ApprovalWorkflowsPage`: `PageHeader` eyebrow "Settings", title "Approval workflows", description,
  primary **New approval workflow**; toolbar: search "Search name", pill filters **All** · **Active** ·
  **Inactive** (default All), count "<n> of <m>"; table: Name (link → details), Levels (levels text),
  Used by, Status (Active green / Inactive grey chip), ⓘ warning icon with tooltip when `warnings`,
  `ActionsMenu`: View · Edit · Set inactive / Set active · Delete (last, separated).
- `WorkflowDetailsPanel` (`SidePanel`, URL `/settings/approval-workflows/:workflowId`): name, status
  chip, used-by sentence, description, levels (badge, label, approvers with "(inactive)"), warnings box,
  "Activities using it" (name link to the activity page, ID, plan › campaign, status chip, "Waiting for
  approval" chip) or "No activities use it yet.", created/changed line; footer **Edit** (primary), **Set
  inactive** / **Set active**, **Delete**.
- `WorkflowFormPanel` (`SidePanel` "New approval workflow" / "Edit approval workflow"): open-rounds note
  when editing and `waitingCount > 0` (FR-006); Name (**Required**, max 120), Description (max 2,000),
  `LevelsEditor`: one bordered block per level — heading "Level <n>", Level name (hint), Approvers
  (003's `PersonPicker`, multiple, **Required**), icon buttons Move up / Move down / Remove (named
  "Move level 2 up", …; Remove hidden when only one level), **Add level**. Validation: the web checks
  nothing it cannot show at once — it sends the save and shows every `fields` problem (keys mapped to
  inputs `levels.<n>.approverIds` → that level's picker) in `FormErrorSummary` and under the fields
  (FR-003). Edit-form rule and `useUnsavedChanges` of 001; conflicts through `conflictMessage(error,
  'approval workflow')`.
- **Set inactive**: `ConfirmDialog` with the effects (FR-007); **Set active**: immediate, toast.
- **Delete**: used (`usedByCount > 0`) → dialog "Can't delete '<name>'" with "Used by <n> activities —
  set it inactive instead." and confirm **Set inactive** (active) or **Close** (inactive); unused → the
  standard delete dialog and `useUndoableDelete` (FR-008). An `in_use` answer shows the same dialog.

### Notification bell (US6, FR-033 – FR-036)

`NotificationBell` (001) keeps its trigger and popover shell; the trigger gets a `CountBadge` (max 9 →
"9+") and the name "Notifications, <n> unread"; the popover renders `NotificationList`: header
"Notifications" + **Mark all as read** (when unread > 0); items newest first as buttons: kind icon
(`approvalNeeded` Bell, `levelApproved`/`approved` CircleCheckBig, `declined` CircleX, `recalled` Undo2,
`restarted` RotateCcw), title, text, relative time (`<time dateTime title=full date and time>`), unread
dot + visually hidden "Unread"; a deleted activity adds "This activity was deleted — find it in the
recycle bin." Choosing an item: optimistic read + `POST …/read`, close the popover, navigate to 006's
activity page `#approval` (or `/recycle-bin` when `exists` is false). Empty list: "You're all caught
up." Loading: `Skeleton`; error: "Notifications couldn't be loaded." + **Try again**.

### New shared component and helpers

- **`CountBadge`** (`@/components`): props `count: number`, `label: string` (visually hidden text read
  with the number), `max?: number` (default 99, shows "<max>+" above); renders nothing for `count <= 0`;
  pill, 11 px semibold, `--status-review` tint (amber, "needs action"), tabular figures. **Reason**: the
  sidebar item, the bell and the approvals tabs need the same counter (three places, two folders); added
  to the gallery.
- `lib/format.ts`: `formatRelativeTime(timestamp, now = new Date())` → "just now" (< 1 min), "<n>
  minutes ago", "<n> hours ago", "yesterday", "<n> days ago" (< 7 days), then `formatDate`;
  `formatWaiting(since, now)` → "less than an hour", "1 hour", "<n> hours", "1 day", "<n> days" (009
  reuses both).
- Reused from earlier features (expected names): `PillTabs` (004), `TextAreaField` (004/006),
  `PersonPicker` (003), `ChoiceField` (002), `DateRangeField` (006), 003's display-money helper, 006's
  `ActivityStatusChip`. If one is missing when 008 is built, 008 adds it to `@/components` with a test
  (task T033).

### States

| Screen | Loading | Empty | Error | Validation | Conflict | Not found |
|---|---|---|---|---|---|---|
| Approvals page | `Skeleton` cards and rows; old rows stay while refreshing | per tab (messages) / "Nothing matches these filters" + **Clear filters** | `ErrorState` + Try again | — | — | — |
| Preview | `Skeleton` | "No line items yet." | `ErrorState` | — | decision: `alreadyDecidedMessage` toast | "This activity no longer exists — someone may have deleted it." + **Close** |
| Approval section | `Skeleton` | "Not submitted yet" / "No approval history yet." | `ErrorState` | dialogs: comment errors under the field | `alreadyDecidedMessage` / `submitConflictMessage` | (006's page handles a gone activity) |
| Workflows page | `Skeleton` rows | "No approval workflows yet — create your first approval workflow" / filtered empty | `ErrorState` | form: `FormErrorSummary` + field messages | `conflictMessage(…, 'approval workflow')` | details: "This approval workflow no longer exists — someone may have deleted it." |
| Bell | `Skeleton` lines | "You're all caught up." | "Notifications couldn't be loaded." + Try again | — | — | deleted activity line |

## User-visible messages

Every text appears here once; code and tests quote it. `<…>` is filled at run time. Plural forms: "1
activity" / "<n> activities"; names joined with ", ".

| Where | Text |
|---|---|
| Approvals page header | Approvals · Approvals · Approve or decline the activities waiting for you, and follow the ones you sent. |
| Summary cards (label · tip) | Waiting for me · Submitted activities whose current approval level lists you. — All open · Every activity waiting for approval, at any level. — Approved this month · Activities that got their final approval this calendar month. — Declined this month · Activities declined this calendar month. |
| Tabs | Waiting for me · All open · Decided by me |
| Toolbar | Search activity name or ID · Filters · Filters (<n>) · Plan · Approval workflow · Owner · Submitted from · Submitted to · <filter>: <value> · Remove filter <filter>: <value> · Clear filters · <n> of <m> |
| Table headers | Activity · Plan › campaign · Owner · Planned cost · Workflow · Level · Submitted · My decision · Status |
| Cells | Level <n> of <m> · <approvers> · waiting <duration> · Start date passed · Approved · <date> / Declined · <date> · (inactive) |
| Empty tabs | Nothing is waiting for you. You're all caught up. · No activities are waiting for approval. · You haven't approved or declined any activities yet. · Nothing matches these filters |
| Sidebar badge (hidden text) | <n> waiting for you |
| Preview | <ID> · Waiting at level <n> of <m> · Details · Plan › campaign › initiative · Owner · Dates · Ongoing from <date> · Planned cost · Activity type · Description · Line items · Line item · Planned cost (USD) · Spend type · GL code · Finance budget item · No line items yet. · Unassigned planned cost · Approve · Decline · Open activity |
| Activity gone (API and web) | This activity no longer exists — someone may have deleted it. |
| Approval section | Approval · Round <n> · <workflow name> · Approval workflow: <name> · No approval workflow chosen yet. · Not submitted yet · Approval history · No approval history yet. · (you) · (inactive) |
| Chain states | Approved by <actor> on <date> · Approved by <actor> on behalf of <names> on <date> · Declined by <actor> on <date> (on-behalf form likewise) · Waiting · Not reached yet · Not decided — submission recalled · Not decided — approval restarted · Not decided — activity deleted |
| Round outcomes | Waiting at level <n> of <m> · Approved · Declined · Recalled · Approval restarted · Cancelled — activity deleted |
| History actions | Submitted · Re-submitted · Approved · Declined · Recalled · Approval restarted · Cancelled — activity deleted · Approver: <names> / Approvers: <names> · <actor> on behalf of <names> |
| Change sentences (API `text`) | <label> changed from <old> to <new> · <label> added: <new> · <label> removed (was <old>) |
| Latest decline box | Declined by <actor>[ on behalf of <names>] on <date>, <time> |
| Status buttons | Submit for approval · Re-submit for approval · Approve · Decline · Recall submission |
| Approved, recall not offered (API reason and web) | Approved activities can't be recalled. Changing its planned cost, dates or line-item amounts sends it back for approval. |
| Submit dialog | Submit '<name>' for approval? · Re-submit '<name>' for approval? · Approval workflow: <name> · Level <n> · <approvers> · Round <n> starts at level 1. · <amount> of the planned cost is not yet assigned to a line item. · Comment for the approvers (optional) · Submit for approval / Re-submit for approval · Cancel |
| Approve dialog | Approve '<name>'? · Level <n> of <m> · <approvers> · This is the last level — the activity becomes Approved. · Next: level <n+1> · <approvers>. · You own this activity. · You are not an approver of level <n>. Your decision will be recorded as made on behalf of <names>. · Comment (optional) · Approve · Cancel |
| Decline dialog | Decline '<name>'? · The activity goes back to <owner name>, who can fix it and re-submit. (no owner: "The activity goes back to its owner, who can fix it and re-submit.") · Why are you declining it? (Required) · Add a comment explaining why the activity is declined. · Decline activity · Cancel |
| Recall dialog | Recall the submission of '<name>'? · It goes back to Created and leaves the approvers' lists. You can submit it again later. · Reason (optional) · Recall submission · Cancel |
| Toasts | Activity '<name>' submitted for approval · Activity '<name>' re-submitted for approval · Level <n> approved — '<name>' now waits for <approvers> (level <n+1>). · Activity '<name>' approved · Activity '<name>' declined · Submission of '<name>' recalled |
| Already decided (API message → web) | Level <n> was already approved by <actor> · Level <n> was already declined by <actor> · The submission was recalled by <actor> · Approval was restarted by <actor> · The approval round was cancelled because the activity was deleted — web: "<message> at <HH:mm>." (not today: "<message> on <DD MMM YYYY> at <HH:mm>.") |
| Submit conflict (web) | Not submitted — <person> changed this activity at <HH:mm>. Check the changes and submit again. |
| Submit problems | (006) Submitting needs <n> things — view list |
| API: submit / recall refusals | Only Created or Declined activities can be submitted for approval. · (recall on Approved: the "Approved activities can't be recalled…" text) |
| API: workflow problems in the check | Approval workflow — needed to submit · Approval workflow '<name>' is inactive — choose an active one. |
| API: comments | Add a comment explaining why the activity is declined. · Use 2,000 characters or fewer. |
| API: round | This approval round doesn't belong to this activity. Reload the page and try again. · (missing roundId / level / version, bad month) Reload the page and try again. · (bad tab) Choose waiting, open or decided. |
| API: restore note | Approval round <n> was cancelled because the activity was deleted — submit it for approval again. |
| Workflows page | Settings · Approval workflows · Who approves activities, level by level. · New approval workflow · Search name · All · Active · Inactive · Name · Levels · Used by · Status · Used by 1 activity / Used by <n> activities · (<w> waiting for approval) · Not used yet · View · Edit · Set inactive · Set active · Delete · No approval workflows yet — create your first approval workflow |
| Workflow form | New approval workflow · Edit approval workflow · Name · Description · Levels · Every level applies to every submission, in order. Any one approver of a level approves it. · Level <n> · Level name · Shown as 'Level <n>' when empty · Approvers · Move level <n> up · Move level <n> down · Remove level <n> · Add level · Save · Cancel · Changes apply to activities submitted from now on. <n> activities waiting for approval keep the levels they were submitted with. (1: "1 activity waiting for approval keeps the levels it was submitted with.") |
| Workflow validation (API) | Enter a name. · Use 120 characters or fewer. · There is already an approval workflow called '<name>'. Choose another name. · Use 2,000 characters or fewer. · Add at least one level. · Level <n>: use 60 characters or fewer. · Level <n>: choose at least one approver. · Level <n>: choose approvers from the list. · Level <n>: <name> is listed twice. · Level <n>: <name> is inactive — choose an active person. |
| Workflow details | Used by … · Levels · Level <n> has no active approver · Anyone can still decide on their behalf. Edit the workflow to choose an active approver. · Activities using it · Waiting for approval · No activities use it yet. · Inactive — not offered on activities. Activities that use it keep it. · Created by <person> on <date>, last changed by <person> on <date> |
| Workflow toasts | Approval workflow '<name>' created — **View** · Approval workflow '<name>' saved · Approval workflow '<name>' set inactive · Approval workflow '<name>' is active again · (delete/undo: 001's texts with record type "Approval workflow") |
| Set inactive dialog | Set '<name>' inactive? · It will no longer be offered on activities. · The <n> activities that use it keep it, shown as '(inactive)'. (omitted at 0; 1: "The 1 activity that uses it keeps it, …") · <w> activities waiting for approval finish with the levels they were submitted with. (omitted at 0; 1: "1 activity waiting for approval finishes …") · Set inactive · Cancel |
| Delete, used | Can't delete '<name>' · Used by 1 activity — set it inactive instead. / Used by <n> activities — set it inactive instead. (also the API `in_use` message) · Set inactive · Close |
| Delete, unused | Delete approval workflow '<name>'? · It moves to the recycle bin; you can restore it. · Delete approval workflow · Cancel |
| Workflow gone (API and web) | This approval workflow no longer exists — someone may have deleted it. |
| Undo refused (API) / restore note | Can't undo — there is now another approval workflow called '<name>'. Restore it from the recycle bin instead. · Renamed to '<new name>' because the name was taken. |
| Settings card | Approval workflows · Who approves activities, level by level. |
| Bell | Notifications · Notifications, <n> unread · Mark all as read · Unread · You're all caught up. · This activity was deleted — find it in the recycle bin. · Notifications couldn't be loaded. · Try again · just now · <n> minutes ago · <n> hours ago · yesterday · <n> days ago |
| Notification gone (API) | This notification no longer exists. |
| Notifications (title · text, stored) | Approval needed · <actor> submitted '<activity>' (<ID>) — Approval needed · <actor> re-submitted '<activity>' (<ID>) — Approval needed · <actor> approved level <n> of '<activity>' (<ID>) — level <n+1> waits for you — Level <n> approved · <actor> approved level <n> of <m> of '<activity>' (<ID>) — Activity approved · <actor> approved '<activity>' (<ID>) — Activity declined · <actor> declined '<activity>' (<ID>): "<first 80 characters of the comment>[…]" — Submission recalled · <actor> recalled '<activity>' (<ID>) — Approval restarted · <actor> changed '<activity>' (<ID>) — it needs approval again |
| Waiting durations | less than an hour · 1 hour · <n> hours · 1 day · <n> days |

## Tests

### Business rules and the Go tests that prove them (package `approvals`, fixed clock, fake `Activities`)

| Rule | Test |
|---|---|
| Workflow saved with ordered levels, default label "Level n", id `WF-` | `workflows_test.go` `TestCreateWorkflow_SavesOrderedLevels_FR002` |
| Name required, ≤ 120, unique ignoring case and spaces (also against inactive ones) | `TestCreateWorkflow_NameRules_FR002` |
| Every problem at once: no name, no level, level without approvers, person twice, inactive person added, unknown person, too-long description / level name | `TestCreateWorkflow_AllProblemsAtOnce_FR003` |
| Inactive approver may stay, may not be added | `TestUpdateWorkflow_InactiveApproverStaysButCannotBeAdded_FR003` |
| Version required and checked; no change → no version, no log | `TestUpdateWorkflow_VersionConflict_FR003`, `TestUpdateWorkflow_NoChangeNoLog_FR026` |
| Edit leaves open rounds' levels unchanged; the next submission uses the new levels | `TestUpdateWorkflow_OpenRoundsKeepLevels_FR006` |
| Usage counts, waiting count, used-by list, warnings | `TestListWorkflows_UsageAndWarnings_FR005`, `TestGetWorkflow_ActivitiesUsingIt_FR005` |
| Set inactive / active: logged, open rounds continue, lookup reports inactive | `TestSetWorkflowActive_FR007` |
| Delete only when unused; refused with `in_use` and the count | `TestDeleteWorkflow_Unused_FR008`, `TestDeleteWorkflow_InUseRefused_FR008` |
| Undo refused when the name is taken; Restore renames | `kinds_test.go` `TestWorkflowUndo_NameTaken_FR008`, `TestWorkflowRestore_RenamesWhenTaken_FR008` |
| Every workflow change logged with readable levels text | `TestWorkflowChanges_Logged_FR026` |
| Preview: problems listed, workflow levels, positive unassigned note, next round number | `rounds_test.go` `TestSubmission_ListsLevelsAndNote_FR011` |
| Problems block and change nothing (each kind a–d from the check, e inactive workflow, missing workflow) | `TestSubmit_ProblemsChangeNothing_FR010`, `TestSubmit_InactiveWorkflowIsAProblem_FR010` |
| Positive unassigned balance and no line items do not block | `TestSubmit_PositiveUnassignedDoesNotBlock_EdgeCase` |
| Submit → Submitted, round 1 at level 1, copy of levels, submitter, comment, log, history | `TestSubmit_StartsRound1_FR011` |
| Only Created / Declined; stale version refused | `TestSubmit_OnlyCreatedOrDeclined_FR009`, `TestSubmit_StaleVersionRefused_D030` |
| Plan status not checked | `TestSubmit_DraftPlanAllowed_FR012` (fake path with a Draft plan; the service never asks) |
| Re-submit: next round number, level 1, current levels | `TestResubmit_NewRoundWithCurrentLevels_FR020` |
| Approve level < m moves on; level m approves | `TestApprove_MovesToNextLevel_FR015`, `TestApprove_LastLevelApproves_FR015` |
| Every level applies; any one approver approves the level | `TestApprove_EveryLevelAppliesAnyApprover_FR004` |
| Decline: comment required (blank refused), ≤ 2,000; ends round at any level; latest decline on top | `TestDecline_CommentRequired_FR016`, `TestDecline_EndsRoundAtAnyLevel_FR016` |
| Non-approver recorded on behalf of the level's approvers | `TestDecide_OnBehalfOf_FR017` |
| Owner may approve; one person on two levels decides each; inactive approver → others act on behalf | `TestDecide_OwnerAndTwoLevelsAndInactiveApprover_EdgeCases` |
| First decision counts; message says what happened, who and when | `TestDecide_FirstDecisionCounts_FR018`, `TestDecide_EndedRoundExplains_FR018` |
| Displayed statuses never notify | `TestDisplayStatusesNeverNotify_FR019` |
| Recall → Created, round Recalled with reason; Approved refused with the explanation | `TestRecall_FR021`, `TestRecall_ApprovedRefused_FR021` |
| Restart: open round ended with changes; approved round gets a restart entry; nothing for other edits (no hook call) | `hooks_test.go` `TestApprovalRestarted_Submitted_FR022`, `TestApprovalRestarted_Approved_FR022` |
| Restore (not Undo) cancels an open round, dated at the delete, with a note | `TestActivityRestored_CancelsOpenRound_FR023` |
| Chain states, history newest first, viewer and actions | `view_test.go` `TestApproval_ChainAndHistory_FR024_FR025`, `TestApproval_Actions_FR009_FR014_FR021` |
| Every action logged with actor, round, level, on behalf | `TestApprovalActions_Logged_FR026_SC004` |
| Tabs: membership and order; deleted activities absent | `inbox_test.go` `TestInbox_Waiting_FR029`, `TestInbox_Open_FR029`, `TestInbox_Decided_FR029_FR030`, `TestInbox_DeletedActivityAbsent_FR023` |
| Row facts: level text, waiting since, start date passed | `TestInbox_RowFacts_FR030_EdgeCase` |
| Summary equals tab counts; month boundaries; bad bounds | `TestSummary_MatchesTabs_FR028_SC007`, `TestSummary_MonthBounds_FR028` |
| Preview facts and line items | `TestPreview_FR032` |
| Recipients per action; never the actor; owner not submitter | `notifications_test.go` `TestNotify_Recipients_FR035`, `TestNotify_NeverOwnAction_FR036`, `TestNotify_OwnerNotSubmitter_Assumption` |
| Decline text quotes the first 80 characters | `TestNotify_DeclineQuotesComment_FR035` |
| 50 newest, total, unread count; deleted activity flagged | `TestListNotifications_FR033`, `TestListNotifications_DeletedActivity_EdgeCase` |
| Mark read (own only), mark all read; not logged | `TestMarkRead_FR034`, `TestMarkAllRead_FR034` |
| History: RestoreContext carries the delete's time and person | `history/service_test.go` `TestRestore_ContextHasDeleteTimeAndPerson_FR023` |
| `errs.InUse` | `platform/errs/errs_test.go` `TestInUse_008P10` |
| Seed: two workflows, rounds/entries/notifications consistent with every seeded status, Mobility waiting at level 2 | `seed/seed_test.go` `TestSeed_Approvals_FR037` |
| Adapters: 006's `Workflows` sees stored workflows (active flag, levels); `ActivityDeleting` leaves the open round stored | `app/approvals_adapter_test.go` `TestAdapter_WorkflowFacts_FR007`, `TestAdapter_ActivityDeletingKeepsRound_008P3` |

### Handler and contract tests (`approvals/handler_test.go`, package `approvals_test`, seeded app)

One test per operation (every request and response validated against the contract), plus the journeys
that need the real 006/007 code:

| Operation | Test |
|---|---|
| 1 | `TestListApprovalWorkflows_SeededTwo_FR005_FR037` |
| 2 | `TestCreateApprovalWorkflow_201_FR002`, `TestCreateApprovalWorkflow_AllProblems400_FR003` |
| 3 | `TestGetApprovalWorkflow_UsedBy_FR005`, `TestGetApprovalWorkflow_404` |
| 4 | `TestUpdateApprovalWorkflow_OpenRoundUnchanged_FR006`, `TestUpdateApprovalWorkflow_409` |
| 5 | `TestDeleteApprovalWorkflow_InUse409_FR008`, `TestDeleteApprovalWorkflow_UnusedUndo_FR008` |
| 6, 7 | `TestDeactivateActivate_FR007` |
| 8 | `TestGetActivityApproval_MobilityWaitingAtLevel2_FR024_FR037` |
| 9 | `TestGetSubmission_IncompleteMobilityListsAll_FR010` |
| 10 | `TestSubmit_ReadyActivity_FR011`, `TestSubmit_Problems400_FR010` |
| 11 | `TestApprove_Level2AsAlex_US2` (Approved, both levels in history), `TestApprove_Twice409_FR018` |
| 12 | `TestDecline_WithoutComment400_FR016` |
| 13 | `TestRecall_FR021` |
| 14 | `TestListApprovals_TabsForDaniel_FR029` |
| 15 | `TestApprovalSummary_FR028` |
| 16 | `TestApprovalPreview_FR032` |
| 17 – 19 | `TestNotifications_SubmitThenDecline_US6` (Priya submits, Daniel declines, both lists checked; nobody gets their own), `TestMarkReadAndAll_FR034`, `TestMarkRead_OtherPersons404_FR034` |
| Restart through 006 | `TestPlannedCostChangeRestartsApproval_FR022` (006's PATCH with its re-approval confirmation on a Submitted and an Approved seed activity) |
| Line item through 007 | `TestLineItemCurrencyChangeRestartsApproval_FR022` |
| Delete through 006 | `TestDeleteSubmittedActivity_UndoKeepsRound_RestoreCancels_FR023` |
| Workflow field lock (006) | `TestWorkflowFieldLockedWhileSubmitted_FR013` |

001's `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` cover the 19 routes.

### Component tests (Vitest, `mockApi`, `renderWithProviders`)

| File | Covers |
|---|---|
| `ApprovalsPage.test.tsx` | Cards from the summary; tab switching via URL; empty texts per tab; search by name and ID; filters, chips, count "3 of 12", Clear filters; row opens the preview; `?activity=` opens it (FR-027 – FR-031) |
| `ApprovalsTable.test.tsx` | Columns per tab, "Start date passed", display currency helper, decided columns (FR-030) |
| `ApprovalsFilters.test.tsx` | Options built from rows; date range; chips remove one filter (FR-031) |
| `ApprovalPreviewPanel.test.tsx` | Facts, line items, unassigned (negative red), chain; Approve → next row and toast; Decline needs a comment; conflict message with time; 404 text (FR-032, FR-018) |
| `ApprovalSection.test.tsx` | Decline box on top; chain states and "(you)"/"(inactive)"; history entries with on-behalf and change sentences; Approved recall explanation; "Not submitted yet" (FR-016, FR-021, FR-024, FR-025) |
| `ApprovalStatusAction.test.tsx` | Button per status; problems → `onShowProblems(n)`; ready → dialog → POST → toast; 400 → `onShowProblems`; conflict → submit conflict text (FR-009 – FR-011) |
| `SubmitDialog.test.tsx` · `DecisionDialog.test.tsx` · `RecallDialog.test.tsx` | Texts, on-behalf and owner notes, comment rules, Cancel focused (FR-011, FR-016, FR-017, FR-021) |
| `ApprovalsNavBadge.test.tsx`, `Sidebar.test.tsx` (extended) | Badge equals waiting count, hidden at 0, link name (FR-027) |
| `ApprovalWorkflowsPage.test.tsx` | List with levels text, used by, status; search; pills; actions menu; set inactive dialog effects; used-workflow delete dialog; unused delete with Undo (FR-001, FR-005, FR-007, FR-008) |
| `WorkflowFormPanel.test.tsx`, `LevelsEditor.test.tsx` | Add/remove/reorder levels; every API problem shown at once and linked; open-rounds note; unsaved changes; conflict (FR-002, FR-003, FR-006) |
| `WorkflowDetailsPanel.test.tsx` | Levels, warnings, activities with links, 404 (FR-005, edge case) |
| `NotificationBell.test.tsx` (extended), `NotificationList.test.tsx` | Badge "9+", name with count, items, relative time, choosing marks read and navigates, deleted activity line, Mark all as read keeps items, empty text, polling interval (FR-033, FR-034, FR-036) |
| `CountBadge.test.tsx`, `format.test.ts` (extended), `useMonthRange.test.ts` | Component and helpers |

### End-to-end journeys (`frontend/e2e/approvals.spec.ts`, real API with fresh seed, workers 1)

1. **US1 + US6 (submit, bell)** — Priya opens an incomplete Mobility China activity, **Submit for
   approval** → "Submitting needs … — view list" and the panel lists every problem; she opens a ready
   Created Lubricants activity → confirmation names "Lubricants EU approval" and "Level 1 · Daniel Okoro" →
   confirm → Submitted, chain "Waiting". Daniel signs in: the bell shows unread, "Approval needed — Priya
   Raman submitted '<name>' (<ID>)"; choosing it opens the Approval section and the count drops.
2. **US3 (inbox)** — Daniel on Approvals: Waiting for me holds both waiting activities, oldest first;
   sidebar badge equals the card; selects one → **Approve** → confirm → toast, row gone, preview on the
   next one → **Decline** with a comment → Declined; **Decided by me** lists both. Priya's bell: "Activity
   declined" with the comment start; Daniel received nothing for his own actions.
3. **US2 (levels, on behalf)** — Alex approves the seeded Mobility activity at level 2 → Approved; its
   history shows Carlos Mendes (level 1) and Alex Morgan (level 2). Then Alex opens a Lubricants activity
   waiting at level 1 → Approve dialog shows "You are not an approver of level 1. … on behalf of Daniel
   Okoro." → confirm → history "Alex Morgan on behalf of Daniel Okoro".
4. **US4 (workflows)** — Alex in Settings › Approval workflows: both seeded workflows with levels text and
   "Used by"; **New approval workflow**, Save with no name and an empty level → both problems; fix →
   created; edit "China BC approval workflow" → open-rounds note; Set inactive → Inactive chip; Set active.
5. **US5 (re-approval, recall)** — Priya recalls a Submitted activity with a reason → Created, history
   "Recalled"; changes the planned cost of a Submitted activity (006's edit, **Save and restart
   approval**) → Created, history "Approval restarted" with "Planned cost changed from … to …".
6. **Accessibility** — axe (WCAG 2.1 A/AA) on Approvals with the preview open, the decline dialog with its
   error, Approval workflows with the form panel and its errors, the bell popover; keyboard only: open a
   row, approve, close.

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 | `ApprovalWorkflowsPage`, ops 1–7 | `ApprovalWorkflowsPage.test.tsx`, handler 1–7, e2e 4 |
| FR-002, FR-003 | `CreateWorkflow`, `UpdateWorkflow`, `WorkflowFormPanel`, `LevelsEditor` | workflows tests, handler 2/4, form tests, e2e 4 |
| FR-004 | `Approve` (all levels, any approver) | `TestApprove_EveryLevelAppliesAnyApprover_FR004` |
| FR-005 | usage in `ListWorkflows` / `GetWorkflow`, list and details | workflows tests, handler 1/3, page/details tests |
| FR-006 | round level copy; form note | `TestUpdateWorkflow_OpenRoundsKeepLevels_FR006`, handler 4, form test |
| FR-007 | `SetWorkflowActive`, `LookupWorkflow` (006 field) | `TestSetWorkflowActive_FR007`, handler 6/7, page test, e2e 4 |
| FR-008 | `DeleteWorkflow`, kind, delete dialogs | workflows/kinds tests, handler 5, page test |
| FR-009 | `ApprovalStatusAction`, `actions` | `TestSubmit_OnlyCreatedOrDeclined_FR009`, status action test |
| FR-010 | `Submission`, `Submit` + 006/007 check via `ReadyCheck` | rounds tests, handler 9/10, e2e 1 |
| FR-011 | `Submit`, `SubmitDialog` | rounds tests, dialog test, e2e 1 |
| FR-012 | no plan-status check | `TestSubmit_DraftPlanAllowed_FR012`, e2e 1 (Mobility plan is Draft) |
| FR-013 | 006 lock + `LookupWorkflow` | `TestWorkflowFieldLockedWhileSubmitted_FR013` |
| FR-014 | `Approve`/`Decline`, section, preview | handler 11/12, preview/section tests |
| FR-015 | `Approve` | rounds tests, handler 11, e2e 3 |
| FR-016 | `Decline`, latest decline | rounds tests, handler 12, section/dialog tests, e2e 2 |
| FR-017 | on-behalf rule, `viewer`, `DecisionDialog` | `TestDecide_OnBehalfOf_FR017`, dialog test, e2e 3 |
| FR-018 | round/level check, `alreadyDecidedMessage` | `TestDecide_FirstDecisionCounts_FR018`, handler 11, preview test |
| FR-019 | display status from 006; no notification on time | `TestDisplayStatusesNeverNotify_FR019`, table test |
| FR-020 | `Submit` on Declined | `TestResubmit_NewRoundWithCurrentLevels_FR020` |
| FR-021 | `Recall`, `RecallDialog`, reason text | rounds tests, handler 13, section test, e2e 5 |
| FR-022 | `ApprovalRestarted` hook (006/007 call it) | hooks tests, handler restart tests, e2e 5 |
| FR-023 | 008-P3: inbox filters stored activities; `ActivityRestored` | inbox + hooks tests, `TestDeleteSubmittedActivity_UndoKeepsRound_RestoreCancels_FR023` |
| FR-024, FR-025 | `Approval` view, `ApprovalSection`, `ApprovalChain`, `ApprovalHistory` | view tests, handler 8, section tests |
| FR-026 | log entries in every action and workflow change | `TestApprovalActions_Logged_FR026_SC004`, `TestWorkflowChanges_Logged_FR026` |
| FR-027 | `ApprovalsNavBadge`, `navigation.ts`, `Sidebar` | badge/sidebar tests, e2e 2 |
| FR-028 | `Summary`, cards | summary tests, handler 15, page test |
| FR-029, FR-030 | `Inbox`, `ApprovalsTable` | inbox tests, handler 14, table test, e2e 2 |
| FR-031 | `ApprovalsFilters`, page | page/filters tests |
| FR-032 | `Preview`, `ApprovalPreviewPanel` | `TestPreview_FR032`, handler 16, preview test, e2e 2 |
| FR-033, FR-034 | notifications ops, `NotificationBell`, `NotificationList` | notifications tests, handler 17–19, bell/list tests, e2e 1 |
| FR-035, FR-036 | `notify`, polling | notifications tests, `TestNotifications_SubmitThenDecline_US6`, list test (interval), e2e 1–2 |
| FR-037 | `seed.go` | `TestSeed_Approvals_FR037`, handler 1 and 8 |

SC-001/SC-002 step counts are checked in e2e 1–2; SC-007 by `TestSummary_MatchesTabs_FR028_SC007` and
e2e 2; SC-008 by e2e 6; SC-003 by the usability session (not automated).

## Dependencies

### Used from other features (by name; adapt to the names delivered, keep the meaning)

| From | What | Used for |
|---|---|---|
| 001 | `history.Service` (`Log`, `MoveToBin`, `Register`, `Diff`), `history.Kind`, `history.RestoreContext`, `history.DeleteResult`, `errs.*`, `record.Meta`, `ids.Generator`, `store.Transactor`, `web.Router`, `actor`, `clock`, `apitest`; `people.Repository` (`Get`, `List`), `people.Person` (`Name`, `Active`, `Initials`); OpenAPI `BadRequest`, `Unauthorized`, `NotFound`, `Conflict`, `Error`, `DeleteResult`, `RecordRef`, `ActorRef`; web `@/components` (Button, PageHeader, SummaryCard, Card, StatusChip, CompletenessChip, InfoTip, Tooltip, ActionsMenu, Avatar, SidePanel, ConfirmDialog, useToast, EmptyState, Skeleton, FormErrorSummary, TextField, ErrorState), `api`/`call`/`ApiError`/`conflictMessage`, `useUndoableDelete`, `useUnsavedChanges`, `usePageTitle`, `formatDate`/`formatDateTime`/`formatTime`, `mockApi`, `renderWithProviders`, e2e `signInAs`, `expectNoAxeViolations` | everything |
| 002 | `backend/internal/platform/money` (`money.Money`), OpenAPI schema `Money`; web `ChoiceField`; the Settings page frame and its section list | money in facts and API; filter choices; Settings card |
| 003 | Person picker `PersonPicker` (multiple mode, active people only); the display-currency money helper (web); people's `active` flag rules; person details usage mechanism (for `ApproverUsage`) | levels editor; planned cost in display currency; "(inactive)" |
| 004 | `PillTabs` (list toolbar), `TextAreaField` (or 006's) | tabs, comments, description |
| 006 | Package `activities`: stored `Activity` (`Status` with the values `created/submitted/declined/approved`, `ApprovalWorkflowID`, `OwnerID`, dates, `PlannedCost`, name, version), its repository (`Get`, `List`, `Put`) and service (view with path, display status, completeness, unassigned planned cost; ready-to-submit check with problems keyed by field; `Kinds()` with `PrepareRestore` resetting to Created on Restore; a `Describe` that names statuses "Created", "Submitted", …); the re-approval save path (006 FR-027) and a `RestartApproval`-style function 007 calls; `ApprovalHooks` and `WorkflowLookup` seams + `SetApprovalHooks`, `SetWorkflowLookup` (added by T019 if missing); the history kind name `activity`. OpenAPI `ActivityStatus`, `ActivityDisplayStatus`, parameter `ActivityId`. Web: `features/activities/ActivityPage.tsx` with an Approval slot (`id="approval"`) and a header action slot, its `onShowProblems`-style "Submitting needs <n> things — view list" + Ready to submit panel (006 FR-025, including the inactive-workflow problem through `WorkflowLookup`), the workflow field lock (FR-013 text for Submitted), `ActivityStatusChip`, `DateRangeField`, activity route (expected `/activities/:activityId`) | facts, status changes, check, hooks, page integration |
| 007 | Package `lineitems`: line items of an activity with labels (spend type, GL code, finance budget item), local and USD planned cost; its re-approval changes routed through 006's restart path with labels naming the line item ("Line item 'Agency fee' (L-1003) — Planned cost"); its ready-check problems inside 006's check | preview; FR-022; FR-010 |

### Reconciliation with the sibling plans (002, 006, 007)

Features 002, 006 and 007 were planned in parallel with this one and name some seams differently. Where
their delivered code has the name below, 008 uses it and adds nothing; this plan's own names apply only
to what is missing (T001 records the outcome, T019 and T023 act on it).

| This plan says | Sibling plan says | Rule for the build |
|---|---|---|
| `activities.WorkflowLookup` + `SetWorkflowLookup`; `(*approvals.Service).LookupWorkflow` | 006: `activities.Workflows` (`GetWorkflow`, `ListWorkflows` → `WorkflowFacts{ID, Name, Active, Levels}`) in `activities.Deps`, `NewSeededWorkflows()` until 008 | Implement 006's `Workflows` in `internal/app/approvals_adapter.go` as `activityWorkflows{repo approvals.Repository}`, mapping stored `approvals.Workflow` to `activities.WorkflowFacts` (reads join the caller's transaction); `app.New` passes it in `activities.Deps.Workflows` instead of `NewSeededWorkflows()`. `WorkflowLookup` is then not added; `LookupWorkflow` stays as the service method the adapter and tests use. (An adapter is needed anyway: `approvals.Service` already has `GetWorkflow` / `ListWorkflows` with other signatures.) |
| `activities.ApprovalHooks.ApprovalRestarted(ctx, a, activityID, previousStatus, changes)`; `SetApprovalHooks` | 006: `ApprovalHooks{ApprovalRestarted(ctx, a, before, after Activity, changes []history.Change); ActivityDeleting(ctx, a, act Activity)}`; `UseApprovalHooks` | `activityApprovalHooks{svc *approvals.Service}` in `approvals_adapter.go` implements 006's interface: `ApprovalRestarted` → `svc.ApprovalRestarted(ctx, a, after.ID, before.Status, changes)`; `ActivityDeleting` → `nil` (008-P3: the delete leaves the open round stored). Wired with `UseApprovalHooks`. `approvals` never imports `activities` |
| `ApprovalHooks.ActivityRestored(ctx, rc, activityID)` called from the activity kind's `PrepareRestore` on Restore | 006: no such hook (its `PrepareRestore` resets the status to Created with a note); 007: optional `history.Kind.AfterRestore(ctx, rc, top)` | T019 adds `ActivityRestored` to 006's `ApprovalHooks` (additive; the adapter forwards it) and calls it on Restore only (`!rc.Undo`) — from the activity kind's `AfterRestore` when 007 delivered that field, else from `PrepareRestore`; its notes join the `restored` entry |
| Restart changes arrive as `history.Change` (label, oldText, newText) | 007: `RestartApproval(ctx, a, activityID, reasons []string)` with ready-made sentences ("Line item 'Agency fee' (L-1003) added: 1,200.00 USD") | 006's restart path passes each reason as `history.Change{Label: reason}` with empty texts; 008 stores it as `ChangeText{Label: reason}` and its `text` is the label unchanged (data-model §3, fourth case) |
| Workflow ids `WF-` (008-P15) | D-103: seed-only `WF-1001`, `WF-1002` (counter `WF`) in the same table | Keep 006's `WF` ids and counter when delivered (008-P15, *Demo data*) |
| Error code `in_use`, `errs.InUse` (008-P10, T004) | D-043: the same code and helper (plus `cannot_restore`) | Reuse 002's; T004 only adds it if 002 did not |
| Workflow field locked while Submitted: "Recall the activity to choose another workflow." (FR-013) | 006: field `approvalWorkflowId` "The approval workflow can't change while the activity is submitted or approved." | The spec wins for Submitted: T019 uses the FR-013 text for Submitted activities and keeps 006's text for Approved ones |
| Activity page link `/activities/:activityId` (`#approval`) | 006: `/plans/:planId/activities/:activityId`; 007 also expects `/activities/:activityId` | If only the nested route exists, T046 adds the route `/activities/:activityId` that loads the activity and replaces the URL with its nested address (keeping the `#hash`); notification and row links stay plan-free |
| 006's Submit stub and Approval placeholder | 006: Submit handler on `ActivityPage` ("Sending activities for approval is coming soon.") and `ActivityApprovalSection.tsx` ("feature 008 replaces its body") | T046 replaces that handler with `ApprovalStatusAction` and the body of `ActivityApprovalSection` with `ApprovalSection` |
| Ready-to-submit check `Activities.ReadyCheck` | 007: `lineitems.Summaries(ctx, a, facts)` feeds 006's check | The adapter's `ReadyCheck` calls 006's check, which already includes 007's problems; it never calls `Summaries` itself |

### Provided to later features

| To | What |
|---|---|
| 009 | `GET /api/approvals?tab=waiting` rows (FR-013 of 009), `GET /api/approvals/summary`, `/approvals?tab=waiting&activity=<id>` deep link, `(*approvals.Service).LatestDeclines`, `formatRelativeTime`, `formatWaiting`, `CountBadge` |
| 010, 011 | Log entries with verbs `submitted`, `re-submitted`, `approved level <n> of`, `approved`, `declined`, `recalled`, `deactivated`, `reactivated`; the `approvalWorkflow` history kind; the restore note |
| 012 | `(*approvals.Service).Submission` and `Submit` (bulk submit runs them per activity, FR-012 of 012) |
| 003 | `(*approvals.Service).ApproverUsage` ("approver in 1 approval workflow") |
| 006 | `LookupWorkflow`, `ApprovalRestarted`, `ActivityRestored`; `ApprovalSection`, `ApprovalStatusAction` components; `GET /api/approval-workflows` for the workflow field |
| all | `errs.InUse` / code `in_use`, `CountBadge` |

### Shared files this feature changes

`backend/api/openapi.yaml`, `backend/internal/app/app.go` (+ new `approvals_adapter.go`),
`backend/seed/seed.go`, `backend/internal/history/{kinds,service}.go` (RestoreContext fields),
`backend/internal/platform/errs/errs.go` (`in_use`), `backend/internal/activities/*` (seams, only if
missing), `frontend/src/app/{navigation.ts,router.tsx,Sidebar.tsx,NotificationBell.tsx}`,
`frontend/src/components/index.ts` (+ `CountBadge`), `frontend/src/lib/format.ts`,
`frontend/src/features/activities/ActivityPage.tsx` (two slots), 002's Settings section list,
`frontend/src/features/dev/ComponentGalleryPage.tsx` (CountBadge).

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| `approvals.Activities` interface + adapter in `internal/app` | approvals must read and update activities, and activities must call approvals back (restart, restore) — a direct import both ways is a cycle; the port also lets the service tests run without building plans, campaigns and line items, and lets this feature be built from the contract while 006/007 names settle | Importing `activities` directly and having activities import approvals: import cycle. Moving approval rules into `activities`: one huge package mixing two features |
| `activities.ApprovalHooks` / `WorkflowLookup` implemented by approvals | 006/007 own the edit that restarts approval and the restore that cancels a round; 008 owns what happens to the round | Duplicating round logic in 006/007; a generic event bus (more concepts) |
| Rounds stay stored while their activity is in the recycle bin (008-P3) | Exact Undo (D-031) of open rounds; cancellation dated at the delete | Ending the round on delete would make Undo inexact; making rounds child kinds of the activity would list "approval rounds" in every delete message and bin entry |
| Polling every 30 s | FR-036's one minute without a push channel | Server-sent events or WebSockets add a long-lived connection type and its tests; a third runtime part is forbidden without an ADR |
| `CountBadge` shared component | Three places (sidebar, bell, tabs) in two folders | Three copies would drift in size, colour and accessible text |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-155–D-169 (008-P10 is D-043, recorded with 002) in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **IDs (D-201).** Approval workflows use `WF-` (006's seed and counter); rounds `AR-`, history entries `AH-`, notifications `NT-`.
  This plan, data model, tasks and fragment were updated from `AW-`.
- **Contract (D-193).** The approvals page summary schema is `ApprovalCounts` (005 owns `ApprovalSummary`).
- **006 seams (D-196).** 006 exports `RestartApproval` and calls `ApprovalHooks.ApprovalRestarted` and the new
  `ApprovalHooks.ActivityRestored`; the *Reconciliation* table holds with these names. 006 serves `/activities/:activityId`
  (D-194), so T046 adds no redirect.
- **Error code.** `in_use` is 002's (D-043); T004 only checks it exists.
- **Web parts (D-197).** 008 owns `CountBadge`, `formatRelativeTime` and `formatWaitingTime` (009 reuses them); it uses 004's
  `PillTabs`, 002's `TextAreaField`, 003's `PersonPicker`, 005's `DateRangeField`.
- **Spec.** FR-004 and FR-012 markers are resolved in spec.md (D-205, D-206); T057's spec update is done.
- **Bulk submit (012).** Submissions check the activity version; 012's adapter passes the version it reads in the same transaction.
