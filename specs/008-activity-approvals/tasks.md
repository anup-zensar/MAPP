---

description: "Task list for feature 008 — Activity Approvals"
---

# Tasks: Activity Approvals

**Input**: Design documents from `specs/008-activity-approvals/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–007 built (their delivered names are
checked in T001).

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Test names carry the FR they prove (plan *Tests*).

**Organization**: Setup (contract), then **Backend** and **Frontend** phases that run in parallel — both
start from the merged contract and never wait for each other — then End-to-end (needs both) and Polish.
Inside each phase, tasks are grouped by user story: US1 submit · US2 decide level by level · US3
approvals page · US4 workflows · US5 re-approval and recall (all P1) · US6 notifications (P2). Every
user-visible text comes from the plan's *User-visible messages* table; every rule from the plan and
data-model.md (when they seem to differ, the plan wins and the difference is reported).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US6
- Paths are relative to the repository root. Toolchain in Git Bash:
  `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [ ] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 Workflow IDs are `WF-` (already renamed); keep 006's seeded `WF-1001`, `WF-1002` (D-201).
- [ ] TC02 Use 006's `RestartApproval` and `ApprovalHooks.ActivityRestored`; no `/activities/:activityId` redirect in T046 (006 serves it) (D-194, D-196).

## Phase 1: Setup (Shared)

**Purpose**: the contract both halves build from.

- [ ] T001 Check what features 002–007 delivered against plan *Dependencies* (including its *Reconciliation with the sibling plans* table, whose rule applies per row) and note the real names at the top of this file under "Delivered names" (no code): Go — `money.Money`, package `activities` (stored `Activity`, status values, repository, view/check functions, `Kinds()`, re-approval save path, any existing `ApprovalHooks` / `WorkflowLookup`), package `lineitems` (line items of an activity with labels); OpenAPI — `Money`, `ActivityStatus`, `ActivityDisplayStatus`, parameter `ActivityId`; web — `PillTabs`, `TextAreaField`, `PersonPicker`, `ChoiceField`, `DateRangeField`, the display-money helper, `ActivityStatusChip`, `features/activities/ActivityPage.tsx` slots, the activity route, 006's "Submitting needs <n> things — view list" function, 002's Settings section list, any stand-in approval workflows in 006's seed
- [x] T002 Merge `specs/008-activity-approvals/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the three tags, the 19 paths, parameters `WorkflowId` and `NotificationId`, the 36 schemas; add `in_use` to `Error.code` and to the `Conflict` response description; point `$ref`s at the names found in T001 when they differ *(Done at consolidation, D-192.)*
- [x] T003 Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes *(Done at consolidation, D-192.)*

**Checkpoint**: Phases 2 and 3 can start, in parallel.

---

## Phase 2: Backend (Go)

**Purpose**: package `internal/approvals`, its wiring, seed and tests. Depends only on T002.

### Foundation (T004 – T008 before any story task)

- [ ] T004 [P] Only if feature 002 did not deliver it (D-043; then this task is just a check): add `CodeInUse = "in_use"` and `InUse(message string) *Error` (409, empty `fields`) to `backend/internal/platform/errs/errs.go`; test `TestInUse_008P10` in `backend/internal/platform/errs/errs_test.go`
- [ ] T005 [P] Add `DeletedAt time.Time` and `DeletedBy actor.Actor` to `history.RestoreContext` in `backend/internal/history/kinds.go`, filled from the bin entry in `putBack` (`backend/internal/history/service.go`); test `TestRestore_ContextHasDeleteTimeAndPerson_FR023` in `backend/internal/history/service_test.go` (Undo and Restore both carry them)
- [ ] T006 Write `backend/internal/approvals/model.go` (stored `Workflow`, `Level`, `Round`, `RoundLevel`, `Action`, `ChangeText`, `Notification` with the JSON names of data-model.md; the API types of the plan, one per contract schema, slices never nil), `repository.go` (the plan's `Repository`) and `store_json.go` (`NewJSONRepository()`; tables `approval_workflows`, `approval_rounds`, `approval_actions`, `notifications`; `RoundsOf` by number, `ActionsOf` oldest first by id number, `NotificationsOf` newest first)
- [ ] T007 [P] Write `backend/internal/approvals/ports.go` exactly as the plan (`Activities`, `ActivityFacts`, `MissingField`, `ReadyCheck`, `Problem`, `LineItemFacts`, status and display constants) and `backend/internal/approvals/fake_test.go`: `fakeActivities` (in-memory facts, `SetStatus` changes the status and adds 1 to the version, records calls; settable `ReadyCheck` problems and line items) plus a `newTestService(t)` helper (temporary `store.DB`, seeded demo people through `people.NewJSONRepository()`, fixed clock 2026-03-12 09:00 UTC, a `history.Service` with a test `activity` kind registered)
- [ ] T008 Write `backend/internal/approvals/messages.go` (every API text of the plan's messages table as constants / format strings) and `backend/internal/approvals/service.go`: `Service`, `NewService(tx, repo, people, activities, generator, hist, now)`, helpers `personOf`, `levelLabel(level, n)`, `levelsText`, `onBehalfOf(levelApprovers, actor)`, `decidedConflict(message, by, at)` (409 `version_conflict` with `conflict`), `changeText(ChangeText)` (the three sentence forms), and `notify(ctx, a, recipientIDs, kind, activity, title, text)` (drops the actor and duplicates, writes one `NT-` record each)

### User Story 4 — Approval workflows (backend)

- [ ] T009 [US4] Write `backend/internal/approvals/workflows_test.go`: `TestCreateWorkflow_SavesOrderedLevels_FR002`, `TestCreateWorkflow_NameRules_FR002`, `TestCreateWorkflow_AllProblemsAtOnce_FR003`, `TestUpdateWorkflow_InactiveApproverStaysButCannotBeAdded_FR003`, `TestUpdateWorkflow_VersionConflict_FR003`, `TestUpdateWorkflow_NoChangeNoLog_FR026`, `TestUpdateWorkflow_OpenRoundsKeepLevels_FR006`, `TestListWorkflows_UsageAndWarnings_FR005`, `TestGetWorkflow_ActivitiesUsingIt_FR005`, `TestSetWorkflowActive_FR007`, `TestDeleteWorkflow_Unused_FR008`, `TestDeleteWorkflow_InUseRefused_FR008`, `TestWorkflowChanges_Logged_FR026`
- [ ] T010 [US4] Implement `backend/internal/approvals/workflows.go`: `ListWorkflows`, `GetWorkflow`, `CreateWorkflow`, `UpdateWorkflow`, `SetWorkflowActive`, `DeleteWorkflow`, usage counting over stored activities and rounds (data-model §1), as the plan's operations table
- [ ] T011 [US4] Implement `backend/internal/approvals/kinds.go` (`Kinds()` → kind `approvalWorkflow` with `Snapshot`, `Remove`, `Put`, `Describe` for name/description/levels/active, `PrepareRestore` name rule) and `backend/internal/approvals/kinds_test.go`: `TestWorkflowUndo_NameTaken_FR008`, `TestWorkflowRestore_RenamesWhenTaken_FR008`

### User Story 1 — Submit (backend)

- [ ] T012 [US1] Write the submit tests in `backend/internal/approvals/rounds_test.go`: `TestSubmission_ListsLevelsAndNote_FR011`, `TestSubmit_ProblemsChangeNothing_FR010`, `TestSubmit_InactiveWorkflowIsAProblem_FR010`, `TestSubmit_PositiveUnassignedDoesNotBlock_EdgeCase`, `TestSubmit_StartsRound1_FR011` (status, round copy, history entry, `statusChanged` log entry verb `submitted`, level-1 notification), `TestSubmit_OnlyCreatedOrDeclined_FR009`, `TestSubmit_StaleVersionRefused_D030`, `TestSubmit_DraftPlanAllowed_FR012`, `TestResubmit_NewRoundWithCurrentLevels_FR020`
- [ ] T013 [US1] Implement `Submission` and `Submit` in `backend/internal/approvals/rounds.go` (plan operations table, log details of plan *Activity-log entries*)

### User Story 2 — Decide level by level (backend)

- [ ] T014 [US2] Add the decision tests to `backend/internal/approvals/rounds_test.go`: `TestApprove_MovesToNextLevel_FR015`, `TestApprove_LastLevelApproves_FR015`, `TestApprove_EveryLevelAppliesAnyApprover_FR004`, `TestDecline_CommentRequired_FR016`, `TestDecline_EndsRoundAtAnyLevel_FR016`, `TestDecide_OnBehalfOf_FR017`, `TestDecide_OwnerAndTwoLevelsAndInactiveApprover_EdgeCases`, `TestDecide_FirstDecisionCounts_FR018`, `TestDecide_EndedRoundExplains_FR018`, `TestDisplayStatusesNeverNotify_FR019`, `TestApprovalActions_Logged_FR026_SC004`
- [ ] T015 [US2] Implement `Approve` and `Decline` in `backend/internal/approvals/rounds.go` (round/level check with `decidedConflict` messages, on-behalf rule, level moves, status changes, history entries, log verbs `approved level <n> of` / `approved` / `declined`, notifications)
- [ ] T016 [US2] Write `backend/internal/approvals/view_test.go` (`TestApproval_ChainAndHistory_FR024_FR025`, `TestApproval_Actions_FR009_FR014_FR021`) and implement `Approval` in `backend/internal/approvals/view.go` (rounds newest first, chain states, history newest first with change sentences, `upcomingLevels`, `latestDecline`, `viewer`, `actions` with the FR-021 reason)

### User Story 5 — Recall and re-approval (backend)

- [ ] T017 [US5] Add `TestRecall_FR021` and `TestRecall_ApprovedRefused_FR021` to `backend/internal/approvals/rounds_test.go`; implement `Recall` in `backend/internal/approvals/rounds.go`
- [ ] T018 [US5] Write `backend/internal/approvals/hooks_test.go` (`TestApprovalRestarted_Submitted_FR022`, `TestApprovalRestarted_Approved_FR022`, `TestActivityRestored_CancelsOpenRound_FR023`, `TestLookupWorkflow_FR007`, `TestApproverUsage`, `TestLatestDeclines`) and implement `backend/internal/approvals/hooks.go`: `ApprovalRestarted`, `ActivityRestored`, `LookupWorkflow`, `ApproverUsage`, `LatestDeclines`
- [ ] T019 [US5] In `backend/internal/activities/` (feature 006's package), add what T001 found missing, exactly as the plan's *Seams* — where 006 delivered its own names (`Workflows`, `ApprovalHooks` with `ActivityDeleting`, `UseApprovalHooks`), follow the plan's *Reconciliation* table instead and add only `ActivityRestored` to 006's `ApprovalHooks`: `ApprovalHooks`, `WorkflowLookup`, `(*Service).SetApprovalHooks`, `(*Service).SetWorkflowLookup`; call `ApprovalRestarted` (with the described changes of the save's log entry) from the re-approval save path that 006 and 007 use; call `ActivityRestored` from the activity kind's `PrepareRestore` on Restore only and append its notes; use `LookupWorkflow` to validate the Approval workflow field and to add the ready-check problem for an inactive workflow (FR-010 e); the FR-013 lock reason "Recall the activity to choose another workflow." while Submitted. Tests in the activities package with a stub hook: `TestReapprovalChange_CallsApprovalHook_FR022`, `TestLineItemReapprovalChange_CallsApprovalHook_FR022`, `TestRestore_CallsActivityRestored_FR023`, `TestUndo_DoesNotCallActivityRestored_D031`, `TestWorkflowFieldLockReason_FR013`

### User Story 3 — Approvals page (backend)

- [ ] T020 [US3] Write `backend/internal/approvals/inbox_test.go` (`TestInbox_Waiting_FR029`, `TestInbox_Open_FR029`, `TestInbox_Decided_FR029_FR030`, `TestInbox_DeletedActivityAbsent_FR023`, `TestInbox_RowFacts_FR030_EdgeCase`, `TestSummary_MatchesTabs_FR028_SC007`, `TestSummary_MonthBounds_FR028`, `TestPreview_FR032`) and implement `Inbox`, `Summary`, `Preview` in `backend/internal/approvals/inbox.go`

### User Story 6 — Notifications (backend)

- [ ] T021 [US6] Write `backend/internal/approvals/notifications_test.go` (`TestNotify_Recipients_FR035` — one case per row of data-model §4, `TestNotify_NeverOwnAction_FR036`, `TestNotify_OwnerNotSubmitter_Assumption`, `TestNotify_DeclineQuotesComment_FR035`, `TestListNotifications_FR033`, `TestListNotifications_DeletedActivity_EdgeCase`, `TestMarkRead_FR034`, `TestMarkAllRead_FR034`) and implement `ListNotifications`, `MarkRead`, `MarkAllRead` in `backend/internal/approvals/notifications.go`

### HTTP, wiring, seed

- [ ] T022 Write `backend/internal/approvals/handler.go`: `NewHandler`, `Routes` registering the 19 operations of the plan's API table (all wrapped in `signedIn`), query parsing (`tab` default `waiting`; `monthStart`/`monthEnd` RFC 3339), delete answering `history.DeleteResult`
- [ ] T023 Write `backend/internal/app/approvals_adapter.go` (implements `approvals.Activities` over the 006/007 services and repositories found in T001; `SetStatus` with `record.Meta.Touch`; plus, when 006 delivered its own seams, `activityWorkflows` implementing `activities.Workflows` and `activityApprovalHooks` implementing `activities.ApprovalHooks` as the plan's *Reconciliation* table) and wire in `backend/internal/app/app.go`: build the approvals service after activities and line items, register `Kinds()`, `SetApprovalHooks` / `UseApprovalHooks`, `SetWorkflowLookup` or `activities.Deps.Workflows` (replacing `NewSeededWorkflows()`), `Routes`; adapter tests in `backend/internal/app/approvals_adapter_test.go` (`TestAdapter_WorkflowFacts_FR007`, `TestAdapter_ActivityDeletingKeepsRound_008P3`)
- [ ] T024 Extend `backend/seed/seed.go` with `addApprovals` exactly as data-model §8 and the plan's *Demo data* (workflows `WF-1001`/`WF-1002` — or, when 006's seed already wrote `WF-1001`/`WF-1002`, those records kept with their ids and completed (plan 008-P15) — workflow field of FY2026 activities, rounds, history entries, notifications, back-dated `statusChanged` log entries, the complete Mobility activity waiting at level 2; summary line + ", 2 approval workflows"); test `TestSeed_Approvals_FR037` in `backend/seed/seed_test.go`
- [ ] T025 Write `backend/internal/approvals/handler_test.go` (package `approvals_test`, `apitest.New(t)` on the seeded app): every test of the plan's handler table, including `TestPlannedCostChangeRestartsApproval_FR022`, `TestLineItemCurrencyChangeRestartsApproval_FR022`, `TestDeleteSubmittedActivity_UndoKeepsRound_RestoreCancels_FR023` and `TestWorkflowFieldLockedWhileSubmitted_FR013` through 006/007's real endpoints
- [ ] T026 In `backend/`: `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass (incl. 001's `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005`)

**Checkpoint**: the API serves all 19 operations on the seeded data.

---

## Phase 3: Frontend (React)

**Purpose**: pages, panels, dialogs, bell and badge. Depends only on T003 (tests use `mockApi`).

### Foundation (T027 – T033 before any story task)

- [ ] T027 [P] Create `frontend/src/components/CountBadge/CountBadge.tsx`, `CountBadge.module.css`, `CountBadge.test.tsx` (props `count`, `label`, `max` = 99; nothing at ≤ 0; "9+" with `max` 9; visually hidden label), export it in `frontend/src/components/index.ts` (+ `index.test.ts`) and add it to `frontend/src/features/dev/ComponentGalleryPage.tsx`
- [ ] T028 [P] Add `formatRelativeTime(timestamp, now?)` and `formatWaiting(since, now?)` to `frontend/src/lib/format.ts` with cases in `frontend/src/lib/format.test.ts` (each boundary of the plan's texts, fake time)
- [ ] T029 [P] Write `frontend/src/features/approvals/messages.ts` (every web text of the plan's messages table; `alreadyDecidedMessage(error)` adds " at HH:mm." or " on DD MMM YYYY at HH:mm."; `submitConflictMessage(error)`; plural helpers) and `messages.test.ts`
- [ ] T030 [P] Write `frontend/src/features/approvals/approvalsApi.ts` (queries and mutations of the plan's *Server state*, `invalidateAllButSession`, 30 s polling of the summary), `useApprovalSummary.ts`, `useMonthRange.ts` and `useMonthRange.test.ts`
- [ ] T031 [P] Write `frontend/src/features/approval-workflows/workflowsApi.ts` (list, details, create, update with the form's version, deactivate, activate, delete)
- [ ] T032 [P] Write `frontend/src/features/notifications/notificationsApi.ts` (`['notifications']` with `refetchInterval: 30_000` and focus refetch; optimistic mark read; mark all read)
- [ ] T033 For each shared web part T001 found missing (`PillTabs`, `TextAreaField`, `PersonPicker` multiple mode, `ChoiceField`, `DateRangeField`, display-money helper, `ActivityStatusChip`), add it to `frontend/src/components/<Name>/` (or `frontend/src/lib/`) with a test and an export, following design system §5; record the reason in the plan's dependency table

### User Story 4 — Settings › Approval workflows (web)

- [ ] T034 [P] [US4] Write `frontend/src/features/approval-workflows/LevelsEditor.tsx` (+ `.module.css`, `.test.tsx`): one block per level with name, `PersonPicker` (multiple, Required), Move up / Move down / Remove (named per level), **Add level**; field errors per `levels.<n>.*` key (FR-002, FR-003)
- [ ] T035 [US4] Write `frontend/src/features/approval-workflows/WorkflowFormPanel.tsx` (+ `.test.tsx`): create and edit in a `SidePanel`, edit-form rule, `useUnsavedChanges`, `FormErrorSummary` with every API problem, open-rounds note (FR-006), conflict through `conflictMessage(error, 'approval workflow')`, toasts (FR-002, FR-003, FR-006)
- [ ] T036 [P] [US4] Write `frontend/src/features/approval-workflows/WorkflowDetailsPanel.tsx` (+ `.test.tsx`): used-by sentence, levels with "(inactive)", warnings box, activities using it with links, created/changed line, footer actions, 404 text (FR-005, edge case)
- [ ] T037 [US4] Write `frontend/src/features/approval-workflows/ApprovalWorkflowsPage.tsx` (+ `.module.css`, `.test.tsx`): header, search, All/Active/Inactive pills, count, table, `ActionsMenu`, empty states, Set inactive dialog with effects, Set active, used-workflow delete dialog, unused delete with `useUndoableDelete`, `in_use` answer → used dialog (FR-001, FR-005, FR-007, FR-008)
- [ ] T038 [US4] Add the routes `/settings/approval-workflows` and `/settings/approval-workflows/:workflowId` (`handle: { title: 'Approval workflows' }`) to `frontend/src/app/router.tsx` (+ `router.test.tsx`), and the section card "Approval workflows" to feature 002's Settings section list (path from T001)

### User Stories 1, 2, 5 — Activity page parts (web)

- [ ] T039 [P] [US2] Write `frontend/src/features/approvals/ApprovalChain.tsx` (+ `.module.css`, `.test.tsx`): one row per level, badge "L<n>", approvers with "(you)" / "(inactive)", state texts incl. on-behalf (FR-024)
- [ ] T040 [P] [US2] Write `frontend/src/features/approvals/ApprovalHistory.tsx` (+ `.module.css`, `.test.tsx`): rounds newest first with outcome labels, entries newest first, approvers at the time, "<actor> on behalf of <names>", comment, change sentences; `compact` prop for the preview (latest 3) (FR-025)
- [ ] T041 [P] [US1] Write `frontend/src/features/approvals/SubmitDialog.tsx` (+ `.test.tsx`): workflow, levels, re-submit round note, unassigned note, optional comment ≤ 2,000, confirm labels (FR-011, FR-020)
- [ ] T042 [P] [US2] Write `frontend/src/features/approvals/DecisionDialog.tsx` (+ `.test.tsx`): approve / decline modes, level and next/last lines, "You own this activity.", on-behalf sentence, decline comment Required with its message before any request (FR-016, FR-017, edge case)
- [ ] T043 [P] [US5] Write `frontend/src/features/approvals/RecallDialog.tsx` (+ `.test.tsx`): explanation, optional reason, **Recall submission** (FR-021)
- [ ] T044 [US1] Write `frontend/src/features/approvals/ApprovalStatusAction.tsx` (+ `.test.tsx`): button per status; submit flow with the submission fetch, `onShowProblems(n)`, `SubmitDialog`, toast, 400 and conflict handling; Approve opens `DecisionDialog` (FR-009 – FR-011, FR-014)
- [ ] T045 [US2] Write `frontend/src/features/approvals/ApprovalSection.tsx` (+ `.module.css`, `.test.tsx`): decline box on top, status and workflow line, chain or "Not submitted yet" with upcoming levels, actions incl. Recall and the Approved explanation, history; states of the plan (FR-016, FR-021, FR-024, FR-025)
- [ ] T046 [US1] In feature 006's `frontend/src/features/activities/ActivityPage.tsx` render `ApprovalStatusAction` in the header (replacing 006's submit stub, passing 006's problems function) and `ApprovalSection` in the Approval section (`id="approval"`; the body of 006's `ActivityApprovalSection`); make the Approval workflow field show "Recall the activity to choose another workflow." while Submitted (FR-013); if 006 delivered only `/plans/:planId/activities/:activityId`, add the route `/activities/:activityId` to `frontend/src/app/router.tsx` that loads the activity and replaces the URL with its nested address, keeping the `#hash` (plan *Reconciliation*); extend `ActivityPage.test.tsx` (both parts present per status; lock text) and `router.test.tsx` (short link keeps `#approval`)

### User Story 3 — Approvals page (web)

- [ ] T047 [P] [US3] Write `frontend/src/features/approvals/ApprovalsFilters.tsx` (+ `.test.tsx`): Filters popover (plan, workflow, owner from the rows; submitted from/to), chips with remove buttons, **Clear filters** (FR-031)
- [ ] T048 [P] [US3] Write `frontend/src/features/approvals/ApprovalsTable.tsx` (+ `.test.tsx`): columns per tab, level text, waiting duration, "Start date passed", display-currency amounts, decided columns with 006's status chip (FR-030, edge case)
- [ ] T049 [US3] Write `frontend/src/features/approvals/ApprovalPreviewPanel.tsx` (+ `.module.css`, `.test.tsx`): facts, completeness, line items, unassigned, chain and compact history, Approve / Decline / Open activity, move to the next row after a decision, conflict and 404 states (FR-014, FR-018, FR-032)
- [ ] T050 [US3] Write `frontend/src/features/approvals/ApprovalsPage.tsx` (+ `.module.css`, `.test.tsx`): header, four cards with tips, pill tabs with counts (URL `tab`), search, filters, count "<n> of <m>", table, empty states, preview via URL `activity` (FR-027 – FR-032, SC-007)
- [ ] T051 [US3] Route `/approvals` in `frontend/src/app/router.tsx`; in `frontend/src/app/navigation.ts` make Approvals a plain `NavItem` (out of `COMING_SOON_SECTIONS`) with `Badge: ApprovalsNavBadge` and add the optional `Badge` to `NavItem`; render it in `frontend/src/app/Sidebar.tsx`; write `frontend/src/features/approvals/ApprovalsNavBadge.tsx` (+ `.test.tsx`); extend `Sidebar.test.tsx` and `router.test.tsx` (badge equals the waiting count, hidden at 0, link name "Approvals 3 waiting for you") (FR-027)

### User Story 6 — Notification bell (web)

- [ ] T052 [P] [US6] Write `frontend/src/features/notifications/NotificationList.tsx` (+ `.module.css`, `.test.tsx`): header with **Mark all as read**, items with kind icon, title, text, relative time with full date as title, unread marker, deleted-activity line, choosing an item marks it read and navigates (activity `#approval` or `/recycle-bin`), empty, loading and error states (FR-033, FR-034, edge case)
- [ ] T053 [US6] Update `frontend/src/app/NotificationBell.tsx` (+ `.module.css`, `NotificationBell.test.tsx`): `CountBadge` with `max` 9, accessible name "Notifications, <n> unread", popover renders `NotificationList`, closes on choice; polling every 30 s checked with fake timers (FR-033, FR-036)
- [ ] T054 In `frontend/`: `npm run check` passes (types, lint, format, all unit and component tests)

**Checkpoint**: every screen works against `mockApi`; with the API of Phase 2 the whole feature runs.

---

## Phase 4: End-to-end (needs Phases 2 and 3)

- [ ] T055 Write `frontend/e2e/approvals.spec.ts` with the plan's six journeys as separate tests, in this order, each using different seeded activities and asserting only its own rows (never global counts): (1) US1 + US6 submit and bell, (2) US3 inbox approve, decline, Decided by me, Priya's bell, (3) US2 level 2 as Alex and on-behalf approval, (4) US4 workflows CRUD, open-rounds note, set inactive/active, (5) US5 recall and planned-cost restart, (6) axe light mode on Approvals with the preview, the decline dialog with its error, Approval workflows with the form errors, the bell popover, plus the keyboard-only approve; reuse `signInAs` and `expectNoAxeViolations` from `frontend/e2e/fixtures.ts`
- [ ] T056 Run `npm run test:e2e` in `frontend/` until `approvals.spec.ts` and the existing `smoke.spec.ts` pass

---

## Phase 5: Polish

- [ ] T057 [P] Record the outcome in the docs (after consolidation numbers the decisions): `docs/sdd/decision-log.md` (the plan's Proposed decisions; Q-002 and Q-007 defaults applied), `specs/008-activity-approvals/spec.md` FR-004 and FR-012 resolved and `checklists/requirements.md` updated, `docs/sdd/feature-map.md` status, `backend/README.md` package list (`approvals/`)
- [ ] T058 [P] Check every screen against `docs/product/design-system.md` (status colours, chips, CRUD pattern, empty states, locked-state reasons) and run the axe journey once in dark mode (`page.emulateMedia({ colorScheme: 'dark' })`)
- [ ] T059 Demonstrate with fresh seed data (`go run ./cmd/seed -reset`, `go run ./cmd/api`, `npm run dev`): the independent tests of US1–US6 of spec.md, and that no server log line holds a person's name or e-mail

---

## Requirement coverage

| FR | Test tasks |
|---|---|
| FR-001 | T037, T025, T055(4) |
| FR-002 | T009, T034, T035, T025 |
| FR-003 | T009, T034, T035, T025, T055(4) |
| FR-004 | T014 |
| FR-005 | T009, T036, T037, T025 |
| FR-006 | T009, T035, T025, T055(4) |
| FR-007 | T009, T018, T023, T037, T025, T055(4) |
| FR-008 | T009, T011, T037, T025 |
| FR-009 | T012, T016, T044, T046 |
| FR-010 | T012, T019, T044, T025, T055(1) |
| FR-011 | T012, T041, T044, T025, T055(1) |
| FR-012 | T012, T055(1) |
| FR-013 | T019, T046, T025 |
| FR-014 | T016, T044, T049, T025 |
| FR-015 | T014, T025, T055(3) |
| FR-016 | T014, T042, T045, T025, T055(2) |
| FR-017 | T014, T042, T055(3) |
| FR-018 | T014, T029, T049, T025 |
| FR-019 | T014, T048 |
| FR-020 | T012, T041 |
| FR-021 | T016, T017, T043, T045, T025, T055(5) |
| FR-022 | T018, T019, T025, T055(5) |
| FR-023 | T005, T018, T019, T020, T023, T025 |
| FR-024 | T016, T039, T045, T025 |
| FR-025 | T016, T040, T045 |
| FR-026 | T009, T014 |
| FR-027 | T050, T051, T055(2) |
| FR-028 | T020, T050, T025 |
| FR-029 | T020, T050, T025, T055(2) |
| FR-030 | T020, T048, T025 |
| FR-031 | T047, T050 |
| FR-032 | T020, T049, T025, T055(2) |
| FR-033 | T021, T052, T053, T025, T055(1) |
| FR-034 | T021, T052, T025 |
| FR-035 | T021, T025, T055(2) |
| FR-036 | T021, T053, T025 |
| FR-037 | T024, T025 |

SC-001/SC-002: T055(1–2) · SC-004: T014 · SC-005: T018, T019, T025 · SC-006: T021 · SC-007: T020, T051 ·
SC-008: T055(6), T058 · SC-003: usability session (T059 prepares it).

## Dependencies & Execution Order

### Phase dependencies

- **Setup (T001 – T003)**: first. T002 after T001; T003 after T002.
- **Backend (T004 – T026)**: after T002. T004, T005, T007 in parallel; T006 before T008; T008 before every
  story task. Story order inside the phase: US4 (T009 – T011) → US1 (T012 – T013) → US2 (T014 – T016) →
  US5 (T017 – T019) → US3 (T020) → US6 (T021) — US3 and US6 need the actions of US1/US2/US5 to exist,
  US4 can also run in parallel with US1 once T008 is done. T022 after the story tasks; T023 after T022 and
  T019; T024 after T023; T025 after T024; T026 last.
- **Frontend (T027 – T054)**: after T003, never waiting for the backend. T027 – T033 first (T027 – T032 in
  parallel). Then US4 (T034 – T038), activity-page parts (T039 – T046: T039 – T043 in parallel, then
  T044, T045, T046), US3 (T047 – T051: T047, T048 in parallel, then T049, T050, T051), US6 (T052, T053);
  the story groups may run in parallel. T054 last.
- **End-to-end (T055 – T056)**: after T026 and T054.
- **Polish (T057 – T059)**: after T056 (T057, T058 in parallel).

### Parallel opportunities

- Backend and frontend phases run side by side (separate agents), both from the merged contract.
- Backend: T004 ∥ T005 ∥ T007; after T008, US4 (`workflows.go`, `kinds.go`) ∥ US1 (`rounds.go`).
- Frontend: T027 ∥ T028 ∥ T029 ∥ T030 ∥ T031 ∥ T032; T034 ∥ T036 ∥ T039 ∥ T040 ∥ T041 ∥ T042 ∥ T043 ∥
  T047 ∥ T048 ∥ T052.

## Implementation Strategy

1. Setup → contract merged and types generated.
2. Backend and frontend agents work in parallel; each phase ends green on its own (`go test ./...`,
   `npm run check`).
3. MVP inside the feature: US4 + US1 + US2 (workflows, submit, decide on the activity page); then US5,
   US3, US6.
4. End-to-end journeys and the demo close the feature.

## Notes

- Never edit `frontend/src/api/schema.d.ts` by hand; regenerate after any contract change.
- Commits: Conventional Commits with the feature number and FRs, e.g. `feat(008): submit for approval (FR-009 – FR-011)`.
- Report any difference between the delivered 006/007 names and the plan in T001's note rather than
  guessing.
