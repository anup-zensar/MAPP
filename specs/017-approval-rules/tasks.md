---

description: "Task list for feature 017 — Approval rules (amends 008, 014 and 003)"
---

# Tasks: Approval Rules

**Input**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml). **Prerequisites**: feature 008 built; **feature 014
(Access control) merged** — 017 uses its `platform/authz`, `internal/access` hooks, Table C and approve/
decline check order.

**Tests**: required (constitution V). In every task, write the named tests first and see them fail.
Test names carry the FR. When an 008 or 014 test asserts behaviour 017 changes (the submitter may decide,
no delegates, no backup), change it to 017's rule and rename it with the 017 FR.

**Organization**: Setup, then **Backend** and **Frontend** in parallel from the merged contract, then
End-to-end and Polish. Stories: US1 automatic vs manager approval · US2 conditional and "all" levels ·
US3 who may decide · US4 delegation (P1) · US5 reminders and escalation · US6 monitoring · US7 audit
trail (P2). Texts come from plan *User-visible messages* and data-model; when they seem to differ the
plan wins and the difference is reported.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- Paths from the repository root. Git Bash: `export PATH="$HOME/tools/node:/c/Program Files/Go/bin:$PATH"`;
  write files with the editor tools (heredocs with apostrophes break).

---

## Phase 1: Setup

- [ ] T001 Check 014 is merged; if not, stop and report. Note under "Delivered names" at the top of this file (no code): 014's `authz` names (grants, `PlanFacts`, the permission constants for `approvals.decide`, `approvals.decideOnBehalf`, `workflows.manage`, the access-rule field, `Require`, `errs.Forbidden`), its `internal/access` hooks (`CanApprove`, role listing with permissions, assignments with scopes, any `RoleHolders`), where its approve/decline check order lives in `internal/approvals`, its `activities.Approvers.NamedIn` adapter; whether 011 is merged; the `people` service functions for create/update/validation and usage; 002's readers `ActivityTypes`, `BusinessUnit`, `Options`
- [x] T002 (done by the consolidation, D-298) Merge `specs/017-approval-rules/contracts/openapi.yaml` into `backend/api/openapi.yaml` following its "How to merge" rules (new paths and schemas; `x-merge: extend` properties, required entries and `x-add-enum` values); add the 7 new operations to `notYetServed` in `backend/internal/app/app_test.go`
- [x] T003 (done by the consolidation) Regenerate `frontend/src/api/schema.d.ts` (`npm run api:types` in `frontend/`); `npm run api:check` passes

**Checkpoint**: Phases 2 and 3 start in parallel.

---

## Phase 2: Backend (Go)

### Foundation (before any story task)

- [ ] T004 [P] Add `PrefixApprovalDelegation = "DLG"` to `backend/internal/platform/ids/ids.go` and `Prefixes`; extend its test
- [ ] T005 Extend `backend/internal/approvals/model.go` with data-model §1–§8: `Condition`, the new `Level`, `Workflow`, `RoundLevel` (incl. `noApprover`), `Round`, `Action` fields, `Delegation`, new action and notification kinds, and the API types of the fragment (`ConditionView`, `SubmissionLevel`, `ExcludedPerson`, `RuleFacts`, `RuleOptions`, `DelegationView`, `DelegationPage`, `DelegationInput`, `DelegationUpdate`, `Monitoring`, `AgeingBucket`, `LevelWaiting`, `EscalationRun`, `EscalationEvent`, and the new fields of `ChainLevel`, `Viewer`, `Row`, `RowRound`, `Decision`, `DeclineNote`, `HistoryEntryView`, `RoundView`, `Submission`, `WorkflowView`, `LevelView`); a `normalise()` on read applying data-model §10 defaults
- [ ] T006 Add table `approval_delegations` to `backend/internal/approvals/repository.go` and `store_json.go` (`GetDelegation`, `ListDelegations`, `PutDelegation`); tests in `store_json_test.go` including `TestLegacyRecords_ReadWithDefaults_017P19` (008-shaped JSON reads with §10 defaults)
- [ ] T007 Extend `backend/internal/approvals/ports.go`: `ActivityFacts.Plan` (`authz.PlanFacts`), `ActivityTypeID`, `BusinessUnits`; `Choice`; `Access` (`CanApprove`, `ApprovingRoles`, `RoleHolders`); `RuleChoices`; `(*Service).UseAccess`, `UseRuleChoices` with permissive defaults in `NewService` (`service.go`); extend `fake_test.go` (`fakeActivities` new fields, `fakeAccess`, `fakeChoices`, a helper putting `authz` grants in the context)
- [ ] T008 `backend/internal/app/approvals_adapter.go`: fill the new `ActivityFacts` fields (plan facts from 004, activity type ID, the plan's business units via 002's `BusinessUnit` with labels and active flags) and add `ruleChoices` over 002's readers; new `backend/internal/app/approvals_access.go` implementing `approvals.Access` over `internal/access` (add `RoleHolders` to `internal/access/hooks.go` if T001 found none: active holders of the role whose scope covers the plan); wire both in `app.go`; add the six Table C rows of plan *Access rules for the new operations* to `internal/access/routes.go`; tests `TestAdapter_RuleFacts_FR010`, `TestApprovalsAccess_RoleHolders_FR001`, 014's every-operation-has-a-rule test green

### US1 — automatic vs manager approval, roles (P1)

- [ ] T009 [P] [US1] `backend/internal/approvals/rules.go`: `Condition.Matches`, `conditionText`, `levelsText`, `evaluate` (data-model §2, §3 steps 1–3); tests first in `rules_test.go`: `TestConditionMatches_Table_FR005`, `TestConditionText_FR009`, `TestLevelsText_FR009`, `TestEvaluate_Table_FR003_FR004_SC001`
- [ ] T010 [US1] `backend/internal/approvals/resolve.go`: named people, line manager and role holders; exclusions (inactive, cannotApprove, owner and submitter while 014's rule is on); usable backup; `noApprover` with advice; the three problems (data-model §3); tests `resolve_test.go` `TestResolve_Sources_FR001_FR011`, `TestResolve_NoApprover_017P8`, `TestResolve_Problems_FR012` (depends on T009)
- [ ] T011 [US1] `backend/internal/approvals/workflows.go` + `messages.go`: keep 014's approver check; add data-model §1–§2 validation all at once (backup approver rules FR-006, reminder/escalation days FR-007, backup holds `approvals.decide`, role includes it, conditions); normalisation; views with `summary`, conditions with texts, `backupApprover`, `role`, timings, new warnings; `kinds.go` `Describe` for levels text and timings; tests `TestCreateWorkflow_RuleProblemsAtOnce_FR008`, `TestWorkflow_BackupAndTimings_FR006_FR007`, `TestUpdateWorkflow_Normalises_FR001_FR002`, `TestWorkflow_BackupAndRoleMustApprove_FR025`, `TestWorkflowChanges_LoggedWithRules_FR035`
- [ ] T012 [P] [US1] `backend/internal/approvals/options.go` `RuleOptions` + route `GET /api/approval-rule-options` in `handler.go`; test `TestRuleOptions_FR001` (only roles that approve)
- [ ] T013 [US1] `backend/internal/approvals/rounds.go`: `submitCheck` keeps every 008 problem and adds resolution problems after the workflow problem (FR-012, FR-036); `Submission` adds `outcomes` (incl. `noApprover`, `advice`) and `approvedAutomatically` (FR-013); `Submit` stores `facts`, timings, resolved levels, `startedAt`, writes `submitted` + `autoApproved`/`skipped` entries, approves at submission when no level waits with the two log entries, notifies approvers and their current delegates or the owner (`autoApproved`, only when owner ≠ submitter) (FR-010, FR-014, FR-032); tests `TestSubmission_OutcomesAndProblems_FR012_FR013`, `TestSubmit_PartlyAutomatic_FR010_FR014`, `TestSubmit_ApprovedAtSubmission_FR014_017P4` (depends on T010)
- [ ] T014 [P] [US1] `backend/internal/people`: `Person.ManagerID`, create/update validation (unknown, inactive when added, self, loop) and messages, views `manager` and `reportCount`, usage role `manager` (details and Deactivate), `Describe` label "Line manager"; tests `TestManager_Rules_FR016`, `TestUsage_Manager_FR016`, handler `TestPeople_ManagerField_FR016`
- [ ] T015 [US1] Handler tests in `backend/internal/approvals/handler_rules_test.go`: `TestSubmit_WF1003Automatic_FR014` (needs T029's seed, or creates the workflow in the test), `TestCreateWorkflow_RuleFields201_FR001`

### US2 — conditional and "all must approve" levels (P1)

- [ ] T016 [US2] `rounds.go` `decide`: counted approvals per data-model §4–§5, level done per mode, backup/on-behalf complete the level, next `decide` level with `startedAt`, partial approvals notify nobody and log "<a> of <b> approved", decline ends the round (FR-015); test `TestDecide_AllMustApprove_FR015`
- [ ] T017 [US2] `view.go`: chain levels with `mode`, `source`, `ruleText`, `decisions`, counts, `excluded`, `backup`, `backupUsed`, `waitingSince`, `remindedAt`, `escalatedAt`, `escalatedTo`, `noApprover` (derived, data-model §4), `advice`, states `skipped`/`autoApproved`; history entries with the new fields; round `facts`, timings, `autoApproved`; `DeclineNote` capacity (FR-033, FR-034); tests `TestApproval_ChainRuleStates_FR033`, `TestApproval_HistoryRuleEntries_FR034`

### US3 — who may decide, segregation of duties (P1)

- [ ] T018 [US3] `backend/internal/approvals/authority.go`: move 014's check order here and extend it exactly as data-model §6 (steps 3a–3f, submitter in step 4, `Shown` reasons); `NamedIn` extended for current delegates and escalated backups, used by 014's `activities.Approvers.NamedIn` adapter (FR-021); table test `authority_test.go` `TestAuthority_Table_FR017_FR018_FR020_SC002` (approver, approver without `approvals.decide`, delegate, backup before/after escalation, on behalf, already approved, other; owner and submitter with the rule on and off; delegate who owns); `TestNamedIn_DelegatesAndBackup_FR021`
- [ ] T019 [US3] Use `authority` in `decide` (403 refusals), record `capacity`, `asDelegateOf`, `onBehalfOf` (on behalf only); `view.go` viewer fields and `actions.approve`/`decline` from `Authority.Shown`; tests `TestDecide_RecordsCapacity_FR019`, `TestApproval_ViewerAuthority_FR017`; update 008/014 tests that 017 changes (depends on T018)
- [ ] T020 [US3] Handler tests: `TestApprove_BySubmitter403_FR018`, `TestApprove_AlreadyApprovedAllLevel403_FR020`

### US4 — delegation (P1)

- [ ] T021 [US4] `backend/internal/approvals/delegations.go`: list (`mine`/`all`, order, `state`, `canChange`, `canManageOthers`), create, update (changed fields + version), cancel; rules, permissions and messages of data-model §7 (403 for permissions, 409 for ended/cancelled); history kind `approvalDelegation` (in `Kinds()`), log `created`/`changed`/`cancelled`; `currentDelegations(today)`; tests `TestCreateDelegation_Rules_FR022`, `TestDelegation_States_FR024`, `TestUpdateCancelDelegation_FR024`, `TestDelegations_ForOthersNeedWorkflowsManage_FR022`
- [ ] T022 [US4] Delegation effect: `authority` step 3b, `Inbox` waiting membership and `asDelegateOf`, `Summary` count, approval-needed and reminder notifications to current delegates with the delegate texts; tests `TestDelegation_Effect_FR023`, `TestInbox_DelegateAndBackup_FR031`, `TestSummary_MatchesWaitingWithDelegates_SC007` (depends on T018, T021)
- [ ] T023 [US4] Routes `GET/POST /api/approval-delegations`, `PATCH /api/approval-delegations/{delegationId}`, `POST …/{delegationId}/cancel` in `handler.go`; handler tests `TestDelegations_CRUD_FR022_FR024`, `TestDelegationsAll_NeedsWorkflowsManage_FR024`

### US5 — reminders and escalation (P2)

- [ ] T024 [US5] `backend/internal/approvals/escalations.go` `RunEscalations` (plan *Service operations*; data-model §4, §8): escalation before reminder, usable backup, `notEscalated`, entries and log by NorthStar with "run by <person>", notifications; tests `escalations_test.go` `TestRunEscalations_FR026_FR027_FR028_SC005` (reminder once; escalation once; escalation replaces a due reminder; inactive / non-approving / owner / submitter / waiting-approver backup not usable; second run sends nothing; legacy round uses 008's waiting-since rule)
- [ ] T025 [US5] Route `POST /api/approvals/escalations/run` (route rule Need `workflows.manage`, T008); handler tests `TestRunEscalations_SeededEscalatesToAlex_FR027`, `TestRunEscalations_WithoutPermission403_FR028`

### US6 — monitoring (P2)

- [ ] T026 [US6] `backend/internal/approvals/monitoring.go` `Monitoring` (data-model §11) and the row flags in `inbox.go` (`overdue`, `waitingDays`, `escalatedTo`, `noApprover`, round `mode`/counts); tests `monitoring_test.go` `TestMonitoring_FR029_FR030_SC006`, `TestInbox_RowFlags_FR031`
- [ ] T027 [US6] Route `GET /api/approvals/monitoring`; handler test `TestMonitoring_Seeded_TradeMagazineOverdue_FR030`

### US7 — audit trail (P2)

- [ ] T028 [US7] One test that drives every new event through the service and checks both the approval history and the activity log (actor or NorthStar, verb, details with capacity, rule or waited days): `TestApprovalRuleEvents_Logged_FR034_FR035_SC004`; fix any gap in T013/T016/T019/T021/T024

### Seed and finish

- [ ] T029 [P] Seed (data-model §13): the people step sets line managers; `backend/seed/approvals.go` adds timings and backups to WF-1001/WF-1002, creates WF-1003, fills the new fields of the seeded rounds, writes DLG-1001 (after 014's roles); 011's demo-history step (if merged) logs its `created`; summary line; tests `backend/seed/approvals_test.go` `TestSeed_ApprovalRules_FR037` and update 008's seed/handler tests that count two workflows
- [ ] T030 Remove the 7 operations from `notYetServed`; `gofmt -l .` prints nothing, `go vet ./...`, `go test ./...` green in `backend/`

---

## Phase 3: Frontend (React)

### Foundation

- [ ] T031 [P] Extend `frontend/src/features/approvals/approvalTypes.ts`, `messages.ts` (every text of plan *User-visible messages*), `approvalsApi.ts` (monitoring query, run mutation), new `delegationsApi.ts`, `features/approval-workflows/workflowsApi.ts` (rule options query) and the `testData.ts` fixtures of both folders with the new fields

### US1

- [ ] T032 [P] [US1] `features/approval-workflows/RuleConditionFields.tsx` (+ css, test): the four parts, USD `MoneyField`s, multiple `ChoiceField`s with "(inactive)", errors per field
- [ ] T033 [US1] `LevelsEditor.tsx` (+ test) and `levelDrafts.ts`: Who approves, approvers or role (roles that approve), mode (not for line manager), Only needed when / Approve automatically when toggles with `RuleConditionFields`, Backup approver; problem keys `levels.<n>.*` mapped to fields (depends on T032)
- [ ] T034 [US1] `WorkflowFormPanel.tsx` (+ test): Reminders and escalation section (new workflows 3 and 5), new hint text, every API problem in `FormErrorSummary`; `WorkflowDetailsPanel.tsx`, `ApprovalWorkflowsPage.tsx`, `workflowTexts.ts`: level summaries and timings (tests updated)
- [ ] T035 [US1] `features/approvals/SubmitDialog.tsx` (+ test): outcomes per level, left-out notes (incl. "can't approve — no approving role"), backup line, no-approver advice, "This activity will be approved automatically.", toast "Activity '<name>' approved automatically"
- [ ] T036 [P] [US1] `features/people/…` people form and details (+ tests): Line manager picker with hint, details lines, Deactivate usage line (FR-016)

### US2 and US7

- [ ] T037 [US2] `ApprovalChain.tsx` (+ css, test): Not needed / Approved automatically states, "<a> of <b> approved" with each approver's state, escalation and reminder lines, source text, left-out people, "No approver" with advice
- [ ] T038 [P] [US7] `ApprovalHistory.tsx` (+ test): new actions, capacity sentences, "Rules applied to: …" per round, waited days

### US3

- [ ] T039 [US3] `ApprovalSection.tsx`, `ApprovalPreviewPanel.tsx`, `DecisionButton.tsx`, `DecisionDialog.tsx` (+ tests): buttons per `actions` (shown / reason as text / hidden on empty reason, 014); capacity sentences; "all" level lines; 008's "You own this activity." note removed

### US4

- [ ] T040 [US4] `DelegationPanel.tsx`, `DelegationForm.tsx` (+ css, tests): groups, Everyone switch and Delegating person only when `canManageOthers`, add/edit/cancel with confirm, all API problems shown, conflict message
- [ ] T041 [US4] `DelegationBanner.tsx` (+ test); `ApprovalsPage.tsx` header action **Delegate my approvals** with `?panel=delegations`; banners above the cards (depends on T040)

### US5 and US6

- [ ] T042 [US6] `MonitoringView.tsx`, `AgeingBars.tsx` (+ css, tests): four cards with tips, ageing list with text counts, Waiting by level table, Overdue list reusing `ApprovalsTable`, empty texts; Monitoring as the fourth pill tab in `ApprovalsPage.tsx` (toolbar hidden on it)
- [ ] T043 [US5] `RunEscalationsButton.tsx` (+ test): shown when `canRunEscalations`; confirm dialog; result toast and not-escalated toast; invalidation
- [ ] T044 [US6] `ApprovalsTable.tsx` (+ test): chips Overdue, Escalated to, as delegate of, No approver; "<a> of <b> approved" in the Level cell
- [ ] T045 [P] `features/notifications/NotificationList.tsx` (+ test): icons and texts of `autoApproved`, `reminder`, `escalated`
- [ ] T046 `npm run check` green in `frontend/` (api:check, typecheck, lint, format, tests)

---

## Phase 4: End-to-end

- [ ] T047 `frontend/e2e/approval-rules.spec.ts` journeys 1–5 of plan *Tests* (journey 5 in its own `test.describe` with a fresh seed); planning helpers from `frontend/e2e/planning.ts`
- [ ] T048 Journey 6: axe on the workflow form with conditions and errors, Monitoring, the delegation panel and form; keyboard-only delegation
- [ ] T049 Run `core-flow.spec.ts`, `module-rest.spec.ts`, `smoke.spec.ts` and 014's e2e file; fix fallout only where 017 changed behaviour (e.g. the third seeded workflow in the Approval workflow select), and report anything else

---

## Phase 5: Polish

- [ ] T050 Check every FR of spec.md against plan *Requirement traceability*: each has a passing test named with it; tick the tasks; list the Proposed decisions 017-P1 – 017-P22 and the 16 client defaults in the lane report for the consolidation step (decision log, domain model §2.3, §6, §8, 014 FR-019 label, feature map, demo data page — not edited by build agents)

---

## Dependencies and parallel work

- 014 merged → T001. T002 → everything. T004–T008 before story tasks. T009 → T010 → T013. T018 → T019, T022.
  T021 → T022 → T023. T024, T026 need T013, T016, T018. T029 can start after T005. T030 last in Phase 2.
- Frontend needs only T003: T031 first; then T032–T036, T037–T038, T039, T040–T041, T042–T045 in parallel
  groups by file. T046 last in Phase 3.
- Phase 4 needs Phases 2 and 3.
