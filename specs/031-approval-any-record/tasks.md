---
description: "Task list for feature 031 — The approval engine approves any record type (amends 008 and 017)"
---

# Tasks: The Approval Engine Approves Any Record Type

**Input**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/](contracts/). **Prerequisites**: features 008, 014, 017 built (on `develop`).

**Tests**: required (constitution VI). In every task write the named tests first and see them fail; test
names carry the FR (`TestX/031 FR-0NN …`, `describe("031 FR-0NN: …")`). When an existing test asserts a
behaviour this feature changes (Re-submit not offered while Submitted), change it to 031's rule and name the
031 FR.

**Organization**: Setup and Foundational (storage and the internal subject), then the stories in priority
order: US1 Re-submit from Submitted (P1), US2 workflow object types (P1), US3 owner and record-field
approvers (P2), US4 purchase requests through the engine (P2), US5 record type on pages (P2); then
end-to-end, docs and polish.

## Format: `[ID] [P?] [Story] Description`

## Phase 1: Setup

- [X] T001 Write migration `backend/internal/platform/database/migrations/00018_approval_subjects.sql` (data-model §10): object type, record field, subject kind and id, self-approval, widened checks, `activity_in_scope` on `subject_id`
- [X] T002 Update queries `approval_rounds.sql`, `approval_actions.sql`, `approval_workflows.sql`, `notifications.sql` in `backend/internal/platform/database/queries/`; run `sqlc generate && sqlc vet`

## Phase 2: Foundational (blocking)

- [X] T003 Add `SubjectKind` with labels, words and record types in `backend/internal/approval/kind.go`; the labels are proven where they show (`inbox_kind_test.go` 031 FR-026/FR-028, `workflows_kind_test.go` 031 FR-009), not by a test that restates the table
- [X] T004 Rename `ActivityID` → `SubjectKind`+`SubjectID` on `Round`, `Action`; `SubjectKind`, `SubjectID`, `SubjectName` on `Notification`; `Workflow.ObjectType`, `Level.RecordField`, `OutcomeSuperseded`, `ActionSuperseded`, `Action.SelfApproval` in `backend/internal/approval/model*.go`
- [X] T005 Update `store_postgres*.go`, `store_fake_test.go` and `approvaltest/store.go` (+ `normalise.go`) for the renamed and new columns (both fakes, constitution VI); the scope-isolation test `store_postgres_test.go` passes on the renamed column (031 FR-001)
- [X] T006 Add the purchase-request port and facts in `backend/internal/approval/ports_purchase.go` (`PurchaseRequests`, `PurchaseRequestFacts`, statuses, `UsePurchaseRequests`)
- [X] T007 Add the internal `subject` and its readers and status hand-off in `backend/internal/approval/subject.go` (`activitySubject`, `purchaseSubject`, `subjectOf`, `moved`, `continued`, `readyCheck`, `subjectsWithRounds`)
- [X] T008 Move the engine onto `subject`: `rounds.go`, `rounds_submission.go`, `rounds_decide.go`, `rounds_recall.go`, `authority.go`, `access_control.go`, `resolve.go`, `view.go`, `inbox*.go` (+ `inbox_rows.go`), `monitoring.go`, `escalations.go`, `notifications.go`, `hooks.go`, `service.go` (notifier, log); every existing test passes (SC-001) except the two that asserted a rule 031 changes (Re-submit while Submitted)
- [X] T009 Update the seed (`backend/seed/approvals.go`, `history_approvals.go`) for the renamed fields; `go test ./seed/...`

## Phase 3: US1 — Re-submit a Submitted activity (P1)

- [X] T010 [US1] Tests first in `backend/internal/approval/resubmit_test.go`: `TestResubmitSubmitted/031 FR-017 FR-020 …` (round superseded at its level, new round open, status stays Submitted, log "re-submitted" with both rounds, new approvers notified, old decisions kept), `…/031 FR-017 a problem changes nothing`, `TestDecisionOnReplacedRound/031 FR-019`, `TestResubmitApprovedAtSubmission/031 FR-020`, `TestSubmissionReplaces/031 FR-018`
- [X] T011 [US1] Implement `backend/internal/approval/resubmit.go` and the submittability rules (`submissionKind`), `Submission.Replaces`, the superseded refusal in `rounds_decide.go`
- [X] T012 [US1] Change `availableActions` (`section.go`) so Re-submit is offered while Submitted; update the 008 tests that asserted the old rule (`view_test.go`, `rounds_submit_test.go`, `handler_test.go`) with 031 FR-017
- [X] T013 [US1] Handler contract test in `backend/internal/approval/handler_subjects_test.go`: `TestResubmitRoute/031 FR-017 FR-018 FR-019` (submission `replaces`, submit 200 with `superseded`, stale approve 409)
- [X] T014 [P] [US1] Frontend `features/approvals`: Re-submit offered with Recall while Submitted (`ApprovalSection`, `SubmitButton`), `SubmitDialog` states the replaced round (`replaces`), history shows "Replaced by a new submission" and the record type of non-activity rounds; tests `ApprovalSection.resubmit.test.tsx` `031 FR-017/FR-018/FR-020` (axe), `ApprovalHistory.kinds.test.tsx` `031 FR-003/FR-016/FR-017`

## Phase 4: US2 — Approval workflows have an object type (P1)

- [X] T015 [US2] Tests first in `backend/internal/approval/workflows_kind_test.go`: `TestCreateWorkflowObjectType/031 FR-004` (default, each kind, unknown refused), `TestUpdateWorkflowObjectType/031 FR-005` (unused changes, used refused with message, logged "Approves"), `TestWorkflowUsageOfPurchaseRequests/031 FR-009`
- [X] T016 [US2] Implement object type in `workflows.go`, `workflows_create.go`, `workflows_list.go`, `workflows_view.go`, `kinds.go` (describe "Approves"); `objectTypeFixed`
- [X] T017 [US2] Test and implement the submit problem for a workflow of another kind: `subject_api_test.go` `TestSubmitSubjectGates/031 FR-008` (in `rounds_submission.go`)
- [X] T018 [US2] `backend/internal/app/activity_workflows.go` lists and finds activity workflows only; test `activity_workflows_test.go` `TestActivityWorkflowsOnlyActivity/031 FR-007`
- [X] T019 [US2] Leave purchase-request workflows to feature 032's seed (research R12); the e2e journey creates one through Settings
- [X] T020 [US2] Handler contract tests in `handler_subjects_test.go`: `TestWorkflowObjectTypeRoutes/031 FR-004 FR-005 FR-011` (create with objectType and a record-field level, patch refused when used, rule options carry object types and record fields)
- [X] T021 [P] [US2] Frontend `features/approval-workflows`: **Approves** field (fixed with reason when `objectTypeFixed`), Approves column and filter (`?approves=`) in `WorkflowList`, details line and records using it; texts per kind (`workflow-texts.ts`); tests `WorkflowFormPanel.objectType.test.tsx` `031 FR-004/FR-005` (axe), `ApprovalWorkflowsPage.kinds.test.tsx` `031 FR-006/FR-009/FR-011` (axe)

## Phase 5: US3 — Owner and record-field approvers (P2)

- [X] T022 [US3] Tests first in `backend/internal/approval/sources_test.go`: table `TestResolveOwnerAndRecordField/031 FR-010 FR-011 FR-012 FR-015` (owner, MoA approver, empty owner/field problem, inactive/cannot approve left out, activity owner left out), `TestResolveRecordFieldBackup/031 FR-012`
- [X] T023 [US3] Implement sources in `resolve.go`, `workflows_levels.go` (normalisation, record field allowlist per kind), `workflows_view.go` (`recordField`, summaries); `TestSinglePersonLevels/031 FR-013`
- [X] T024 [US3] Tests and implementation of conditions per object type: `workflows_kind_test.go` `TestConditionsPerObjectType/031 FR-014` (refusal and the "amount" wording of `rules.go`), `TestRecordFieldOnActivityWorkflow/031 FR-011`
- [X] T025 [US3] Rule options carry object types and record fields (`options.go`); proven by `TestWorkflowObjectTypeRoutes/031 FR-011`
- [X] T026 [P] [US3] Frontend `LevelsEditor`, `level-drafts.ts`, `RuleConditionFields`, `chain-messages.ts`: sources Record owner and Person named on the record (field select, not offered on activity workflows), no mode for single-person sources, kind-aware conditions; tests `LevelsEditor.sources.test.tsx` `031 FR-010/FR-011/FR-013` (axe), `RuleConditionFields.test.tsx` `031 FR-014`

## Phase 6: US4 — Purchase requests through the engine (P2)

- [X] T027 [US4] Fake `fakePurchases` in `backend/internal/approval/fakes_purchase_test.go` (records RoundChanged events, applies a 013-like status mapping, scope through `ListInScope`)
- [X] T028 [US4] Tests first in `backend/internal/approval/subject_api_test.go`: `TestPurchaseRequestsNotWired/031 FR-025`, `TestSubmitSubjectGates/031 FR-021 FR-008`, `TestRoundChangedLevelByLevel/031 FR-023 FR-030`, `TestRoundChangedEnds/031 FR-023 FR-017`, `TestModificationRounds/031 FR-002 FR-022`, `TestPurchaseRequestSelfApproval/031 FR-015 FR-016`, `TestActivitySelfApprovalStillBarred/031 FR-015`, `TestSectionActions/031 FR-024 FR-017`, `TestRecallSubjectRights/031 FR-024 FR-029`
- [X] T029 [US4] Implement `backend/internal/approval/subject_api.go` (`SubmitSubject`, `RecallSubject`, `DecideSubject`, `SectionOf`), `section.go` (`Section`), `SubjectDecision`, and self-approval (`SelfApproval`, D-302 by kind) in `authority.go`, `resolve.go`, `escalations.go`
- [X] T030 [US4] Contract schemas `ApprovalSubject`, `ApprovalSection` in `backend/api/openapi.yaml`; contract test that a `Section` validates against `ApprovalSection` (`subject_contract_test.go` `TestSectionMatchesContract`)

## Phase 7: US5 — Record type on the Approvals page, Monitoring and notifications (P2)

- [X] T031 [US5] Tests first `backend/internal/approval/inbox_kind_test.go`: `TestInboxCarriesKind/031 FR-026` (activity and PR rows, `activity` null for PR, scope), `TestMonitoringKind/031 FR-027`, `TestNotificationSubject/031 FR-028` (titles, exists via the port); `handler_subjects_test.go` `TestRowsAndNotificationsCarrySubject/031 FR-026 FR-028`
- [X] T032 [US5] Implement row `subject`, `WaitingByLevel.Kind`, notification `subject`, titles per kind; `internal/app/home_sources.go` keeps activity rows
- [X] T033 [US5] Contract: update `backend/api/openapi.yaml` (rows, notifications, monitoring, rounds, history, submission, rule options, workflows); `pnpm run api:types`; existing contract tests pass
- [X] T034 [P] [US5] Frontend `features/approvals`: Record column and Record type column (`ApprovalsTable`), Record type filter (`approval-filters.ts`, `ApprovalsFilters`), Monitoring by-level "Approves"; rows of other kinds open their page (`ui/subject-links.ts`); status chip of any record (`ui/SubjectStatusChip`); tests `ApprovalsFilters.test.tsx` `031 FR-026`, `ApprovalsTable.test.tsx` `031 FR-026`, `MonitoringView.test.tsx` `031 FR-027`
- [X] T035 [P] [US5] Frontend `features/notifications`: read `subject`, link per record type, deleted text per kind; test `NotificationList.test.tsx` `031 FR-028`

## Phase 8: End-to-end, docs, polish

- [X] T036 E2E `frontend/e2e/approval-any-record.spec.ts`: US1 (re-submit a Submitted activity, see "Replaced by a new submission") and US2 (create a purchase-request workflow, filter the list, the activity form does not offer it), with axe
- [X] T037 Docs: `docs/product/domain-model.md` §2.3 (re-submit from Submitted, record types), §6 approval workflow row (object type, sources), §7 notification; no new design-system pattern
- [X] T038 Comment budget and file length of every touched file; `gofmt`, `golangci-lint run --new ./...`, `go vet ./...`, `govulncheck ./...`
- [X] T039 Full checks of brief §9: `go test -race ./...` with `TEST_DATABASE_URL`; frontend `api:check`, `typecheck`, `lint`, `format:check`, `test`, `build`; e2e on ports 18131/14131 and `mapp_e2e_031`
- [X] T040 Run the app on `mapp_f031` and port 18131; exercise re-submit and the workflow object type through the API (curl) and record what was seen (research R14)

## Dependencies

T001 → T002 → T004 → T005; T003 → T004; T006 → T007 → T008 → every story. US1 (T010–T014) and US2
(T015–T021) are independent after Phase 2; US3 depends on US2 (object type of a workflow); US4 depends on
T007–T008 and US3 (record-field source); US5 depends on T008. Frontend tasks [P] follow T033 (generated
types).

## Requirement coverage

FR-001 T001–T005, T009 · FR-002 T001, T028 · FR-003 T014, T033 · FR-004 T015–T016, T020–T021 · FR-005
T015–T016, T020–T021 · FR-006 T021 · FR-007 T018 · FR-008 T017 · FR-009 T015 · FR-010 – FR-013 T022–T026 ·
FR-014 T024, T026 · FR-015, FR-016 T028–T029 · FR-017 – FR-020 T010–T014 · FR-021 – FR-025 T027–T030 ·
FR-026 – FR-028 T031–T035 · FR-029 T028 · FR-030 T010, T028 · SC-001 T008, T039 · SC-002 T036 · SC-004 T018,
T036 · SC-005 T028.
