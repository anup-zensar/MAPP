---

description: "Task list for feature 006 — Activities"
---

# Tasks: Activities

**Input**: Design documents from `specs/006-activities/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–005 merged (002's money, models, activity
types and Shell attributes; 003's person picker and `PersonRef`; 004's plans and plan tree; 005's campaigns,
initiatives and their history kinds).

**Tests**: required — constitution V. In every group, write the tests first and see them fail, then
implement. Test names carry the FR they prove (plan, *Business rules and the tests that prove them*).

**Organization**: Setup, then **Backend** and **Frontend** — independent of each other, both start from
the contract (`backend/api/openapi.yaml` after T001, `frontend/src/api/schema.d.ts` after T002); the web
app's tests use `mockApi()` — then **End-to-end**, which needs both, then **Polish**. Inside Backend and
Frontend, a small foundation first, then one group per user story. Every user-visible text comes from the
plan's *User-visible messages* table.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US5
- Paths are relative to the repository root. Names of other features' files and symbols are the expected
  ones of the plan's *Dependencies* table; use the real names from their merged plans.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [X] TC01 Export `RestartApproval(ctx, a, activityID, reasons)` and use it from `Update`; tests `TestRestartApproval_SendsBackAndLogs_FR027` (D-196).
- [X] TC02 Add `ApprovalHooks.ActivityRestored` and call it after a recycle-bin restore (not Undo) (D-196).
- [X] TC03 `LineItems.CopyAll` and `campaigns.Activities.CopyAll` take `yearShift` (D-195).
- [X] TC04 [P] Add the route `/activities/:activityId` that redirects to the activity under its plan, keeping the hash, with a test; add the activity entry to `recordHref` (D-194).
- [X] TC05 Use 004's `CompletenessChip` `itemsBelow` instead of new `below`/`onOpen` props (D-199).
- [ ] TC99 Check the success criteria SC-002, SC-003, SC-004, SC-005, SC-006, SC-007, SC-008, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup

- [x] T001 Merge the paths, parameters and schemas of `specs/006-activities/contracts/openapi.yaml` into `backend/api/openapi.yaml` (tag Activities; references to 001's responses and to `Money`, `AttributeValues`, `ShellAttribute` (002) and `PersonRef` (003) resolve in the merged file); mention feature 006 in `info.description` *(Done at consolidation, D-192.)*
- [x] T002 [P] Run `npm run api:types` in `frontend/` to regenerate `frontend/src/api/schema.d.ts`; `npm run api:check` passes *(Done at consolidation, D-192.)*
- [X] T003 [P] Create the folders `backend/internal/activities/` and `frontend/src/features/activities/`

**Checkpoint**: the contract is in place; backend (Phase 2) and frontend (Phase 3) can start in parallel.

---

## Phase 2: Backend

### Backend foundation (blocks the backend stories; T004 first, T007 after T004–T006)

- [X] T004 Write `backend/internal/activities/model.go`: package comment; `Activity` (stored, every field of data-model §1 with JSON tags, embeds `record.Meta`); `Status` (`created`, `submitted`, `declined`, `approved`) and `DisplayStatus` (+ `inProgress`, `complete`); API types `View`, `Summary`, `SummaryPage`, `FormView`, `FormContext`, `TypeOption`, `WorkflowOption`, `Defaults`, `Input`, `Update` (`Version *int`, `RestartApproval bool`), `GoalsInput`, `Goals`, `LineItemTotals`, `Completeness`, `SubmitCheck`, `SubmitProblem`, `MoveInput`, `CopyInput`, `Destination`, `DestinationPage`, `Rollup`, `StatusChange` — matching the contract's schemas; lists never nil
- [X] T005 [P] Write `backend/internal/activities/repository.go` (`Repository`: `Get`, `ListByInitiative`, `ListByInitiatives`, `Put`, `Delete`) and `backend/internal/activities/store_json.go` (`NewJSONRepository()`, table `activities`, `ListByInitiative` ordered by id)
- [X] T006 [P] Write `backend/internal/activities/deps.go` (`PlanReader`, `HierarchyReader`, `SettingsReader`, `PeopleReader`, `Workflows`, `WorkflowFacts`, `LineItems`, `LineItemFacts`, `LineItemProblem`, `ApprovalHooks`, `Deps`, unexported `noLineItems` and `noApprovalHooks`) and `backend/internal/activities/workflows_seed.go` (`NewSeededWorkflows()` reading table `approval_workflows`: `GetWorkflow`, `ListWorkflows` sorted by name) as the plan's *What activities reads from other packages*
- [X] T007 Write `backend/internal/activities/fakes_test.go`: in-memory fakes of every `deps.go` interface with the demo shapes (plan P-1001 2026 "FY2026 Lubricants Europe", model "Lubricants 1.0" 2.00 % / 5.00 % / 10,000.00 USD allowing Event (default 25,000.00) and Digital media; inactive type; initiatives "Digital & social" 2026-01-01–2026-06-30, "Print & outdoor" 2026-01-01–2026-12-31, "Retail events" 2026-06-01–2026-12-31 and one in another plan; Shell attributes always shown / Lubricants / Mobility with Country / OU as a split; people Priya (active) and an inactive person; workflows one active one inactive; `LineItems` with settable facts and recorded `CopyAll`/`DatesChanged` calls; `ApprovalHooks` recording calls); `newTestService(t)` on a temporary `store.DB` with a real `history.Service` (kinds `plan`, `campaign`, `initiative` registered as test kinds before `activity`) and the fixed clock 2026-03-12 09:00 UTC
- [X] T008 [P] Write `backend/internal/activities/dates.go` and `dates_test.go`: parse `YYYY-MM-DD`, `dateText` ("01 Mar 2026"), `rangeText`, `checkDates(start, end, ongoing, initiative, latestEnd) []errs.FieldError`, `datesFit(activity, initiative) bool`; tests `TestCheckDates_EndBeforeStart_FR009`, `TestCheckDates_OutsideInitiative_FR009`, `TestCheckDates_LatestEndR5_FR009`, `TestCheckDates_OngoingChecksStartOnly_FR010`, `TestCheckDates_InitiativeWithoutDates_FR009`
- [X] T009 [P] Write `backend/internal/activities/names.go` and `names_test.go`: `sameName` (case and surrounding spaces ignored), `copyName(base, taken)` (" (copy)", " (copy 2)", …, original shortened so the result is ≤ 120 characters), `restoredName`; tests `TestSameName_IgnoresCaseAndSpaces_FR006`, `TestCopyName_Suffixes_FR030`, `TestCopyName_LongNameShortened_EdgeCase`, `TestRestoredName_Suffixes_FR031`
- [X] T010 [P] Write `backend/internal/activities/goals.go` and `goals_test.go` (goals of data-model §4 with `settings.Waterfall`; normalising input: not typed → MQL and revenue not stored); tests `TestGoals_CalculatedFromModel_FR018` (250,000 → 5,000, 250 deals, 2,500,000.00 USD; empty INQ → all null), `TestGoals_TypedValuesKept_FR018`
- [X] T011 [P] Write `backend/internal/activities/status.go` and `status_test.go`: `displayStatus(a Activity, today string)`; `TestDisplayStatus_FR021` (approved before start → approved, on start → inProgress, after end → complete, ongoing never complete, no dates → approved, other statuses unchanged)
- [X] T012 Write `backend/internal/activities/readiness.go` and `readiness_test.go`: pure `check(checkInput) SubmitCheck` and `completeness(SubmitCheck)` per data-model §4.1; tests `TestCheck_ListsEveryProblemAtOnce_FR023` (cost centre empty, split 90.00 %, line item without GL code, custom spread 900.00 of 1,000.00, balance −500.00 → five problems in two groups, in order), `TestCheck_ReadyWhenNothingMissing_FR023`, `TestCheck_DatesOutsideAndTypeNotAllowed_EdgeCase`, `TestCheck_InactiveWorkflow_FR023`, `TestCheck_ZeroPlannedCostIsFilled_FR011`, `TestCheck_HiddenGroupsNotCounted_FR015`, `TestCheck_SplitNot100Listed_FR017`, `TestCheck_PositiveUnassignedIsAdvice_FR024`, `TestCheck_NegativeUnassignedBlocks_FR024`, `TestCompleteness_FR022`
- [X] T013 Write `backend/internal/activities/kinds.go`: `(*Service).Kinds()` with kind `activity` (Singular "activity", Plural "activities", Parent `initiative`, `Snapshot`, `ChildIDs`, `Remove`, `Put`, `Describe` of data-model §5; `PrepareRestore` added in T031); test `TestKinds_RegisterAfterInitiative` in `kinds_test.go`

### User Story 1 — Create an activity (backend)

- [X] T014 [US1] Write `backend/internal/activities/create_test.go`: `TestNewForm_OnlyActiveAllowedTypes_FR007`, `TestNewForm_Defaults_FR008_FR009_FR012` (owner and dates from the initiative; workflow only when exactly one is active), `TestNewForm_ModelWithoutTypes_EdgeCase`, `TestNewForm_AttributesInOrder_FR014`, `TestCreate_OnlyNameAndTypeRequired_FR003`, `TestCreate_CreatedWithReadableIDAndLogged_FR004` (A-1001, `created` entry with actor and path), `TestCreate_NameUniqueInInitiative_FR006`, `TestCreate_ExternalNameOptional_FR006`, `TestCreate_TypeActiveAndAllowed_FR007`, `TestCreate_AllProblemsAtOnce_FR009` (name taken + end before start + outside the initiative's dates in one error), `TestCreate_OngoingClearsEndDate_FR010`, `TestCreate_PlannedCostRules_FR011` (negative refused, 0.00 kept, 12.345 → 12.35, EUR refused), `TestCreate_WorkflowMustBeActive_FR012`, `TestCreate_AttributeValuesChecked_FR016_FR017`, `TestCreate_GoalsMustBeWholeAndNotNegative_FR018`, `TestCreate_InitiativeGone_404`
- [X] T015 [US1] Write `backend/internal/activities/service.go`: `Service`, `NewService`, `UseLineItems`, `UseApprovalHooks`, `NewForm`, `Create` with every rule of the plan's operations table (shape problems and in-Write problems collected into one `validation_failed`), and `backend/internal/activities/view.go` (`newView`, `newSummary`, `newFormView` with `existingNames` and defaults) so T014 passes
- [X] T016 [US1] Write `Rollups` in `backend/internal/activities/provided.go` and `TestRollups_FollowEveryChange_FR005` in `backend/internal/activities/provided_test.go` (create and a planned-cost edit change the sums; extended in T030)
- [X] T017 [US1] Write `backend/internal/activities/handler.go` (`NewHandler`, `Routes` with `GET /api/initiatives/{initiativeId}/activity-form` and `POST /api/initiatives/{initiativeId}/activities`) and `backend/internal/activities/handler_test.go` (`package activities_test`): `TestNewActivityForm_Contract_FR001`, `TestCreateActivity_Contract_FR004` (201; 400 listing every field problem; 404 unknown initiative)
- [X] T018 [US1] Wire activities in `backend/internal/app/app.go`: repository, `NewService` with `Deps` (004's plans repository, 005's campaigns repository, 002's settings, people repository, `NewSeededWorkflows()`), register `Kinds()` after 005's kinds, `Routes`, and hand the service to the activity seams of the plans (004) and campaigns (005) services for `Rollups`; `TestRoutesMatchContract` passes for the routes added so far
- [X] T019 [US1] Add the demo data to `backend/seed/seed.go` (the two workflows WF-1001 and WF-1002 into `approval_workflows`, then activities A-1001 – A-1011 of the plan's *Demo data* table, attribute values by 002's labels) and checks to `backend/seed/seed_test.go` (IDs, statuses, A-1007 ongoing, A-1001 has three empty needed attributes, A-1008 Country / OU totals 90.00 %)

### User Story 2 — See an activity's status and everything it still needs (backend)

- [X] T020 [US2] Write `backend/internal/activities/view_test.go`: `TestView_DerivedValues_FR019_FR020` (path refs, owner/type/workflow with `active`, display status, unassigned planned cost with fake line items, line-item totals, completeness, readiness), `TestListForPlan_Order_FR033` (start date, then name ignoring case, no dates last; every initiative of every campaign of the plan); and in `handler_test.go` `TestGetActivity_Contract_FR019` (seeded A-1001: 3 missing fields, unassigned; 404 text) and `TestListPlanActivities_Contract_FR033` (P-1001 rows; 404 unknown plan)
- [X] T021 [US2] Implement `Get` and `ListForPlan` in `backend/internal/activities/service.go` (one `tx.Read`; line-item facts fetched once per call through `LineItems.Facts`) and the summary builders in `view.go`
- [X] T022 [US2] Add `GET /api/activities/{activityId}` and `GET /api/plans/{planId}/activities` to `backend/internal/activities/handler.go` (`SummaryPage{Items, Total}`)

### User Story 4 — Edit an activity, including one that is submitted or approved (backend)

- [X] T023 [US4] Write `backend/internal/activities/update_test.go`: `TestUpdate_LogsEveryChange_FR026` (one `changed` entry; per-attribute changes `attributes.<id>`), `TestUpdate_NoChangeNoLog_FR026`, `TestUpdate_VersionConflict_FR026`, `TestUpdate_MissingVersion_FR026`, `TestUpdate_ReapprovalNeedsConfirmation_FR027` (Submitted + planned cost change without `restartApproval` → `validation_failed` on `restartApproval`, nothing saved), `TestUpdate_ConfirmedReapprovalReturnsToCreated_FR027` (status in the changes, details "Approval restarted because the planned cost changed.", `ApprovalRestarted` and `DatesChanged` called), `TestUpdate_OtherEditsKeepStatus_FR028` (description, owner, attributes, goals on an Approved activity), `TestUpdate_OngoingCannotBeUndone_FR010`, `TestUpdate_WorkflowLockedWhenSubmittedOrApproved_FR012`, `TestUpdate_InactiveValuesKeptWhenUnchanged_FR013`, `TestUpdate_UnchangedDatesNotChecked_EdgeCase`, `TestEditForm_IncludesCurrentInactiveValues_FR013`
- [X] T024 [P] [US4] Add `TestDescribe_ReadableChanges_FR026` to `backend/internal/activities/kinds_test.go` (owner name, "120,000.00 USD", "01 Mar 2026", type and workflow names, attribute label and option texts, status words)
- [X] T025 [US4] Implement `Update` and `EditForm` in `backend/internal/activities/service.go` (rules of the plan's operations table, re-approval, hooks) and finish `Describe` in `kinds.go` so T023–T024 pass
- [X] T026 [US4] Add `PATCH /api/activities/{activityId}` and `GET /api/activities/{activityId}/form` to `handler.go`; contract tests `TestUpdateActivity_Contract_FR026` (200; 400 `restartApproval`; 409 `version_conflict` with `conflict`) and `TestEditActivityForm_Contract_FR013` in `handler_test.go`

### User Story 3 — Shell attributes, % splits and goals (backend; after T025)

- [X] T027 [US3] Add to `backend/internal/activities/update_test.go` `TestUpdate_HiddenGroupValuesKept_FR015` (Lubricants values survive a save that omits them while the plan is Mobility-only, and count again once Lubricants is back) and to `view_test.go` `TestView_GoalsUseCurrentModelValues_FR018` (changed model rates change the calculated goals; typed goals unchanged)
- [X] T028 [US3] Implement the hidden-group merge in `Update` (`service.go`) and the read-time goals in `view.go` so T027 passes

### User Story 5 — Move, copy and delete an activity (backend)

- [X] T029 [US5] Write `backend/internal/activities/move_copy_test.go`: `TestDestinations_ReasonsPerTarget_FR029_FR030`, `TestMove_Refusals_FR029` (other plan, same initiative, dates not fitting — the exact message —, name taken), `TestMove_KeepsStatusAndLogsParent_FR029`, `TestCopy_Names_FR030`, `TestCopy_CreatedWithLineItems_FR030` (Approved source → Created copy, new ID, all values, `LineItems.CopyAll` called, one `copied` entry with details), `TestCopy_TargetDatesMustFit_FR030`
- [X] T030 [P] [US5] Add to `backend/internal/activities/kinds_test.go` `TestDelete_MovesToBin_FR031` (bin entry, `ActivityDeleting` called), `TestUndo_RestoresExactly_FR031` (same ID, values, status, version), `TestUndo_NameTakenRefused_FR031`, `TestRestore_ResetsToCreatedAndRenames_FR031`; extend `TestRollups_FollowEveryChange_FR005` in `provided_test.go` with move, copy, delete and Undo
- [X] T031 [US5] Implement `Destinations`, `Move` and `Copy` in `backend/internal/activities/move_copy.go`, `Delete` in `service.go` and `PrepareRestore` in `kinds.go` so T029–T030 pass
- [X] T032 [US5] Add `DELETE /api/activities/{activityId}`, `GET /api/activities/{activityId}/destinations`, `POST /api/activities/{activityId}/move` and `POST /api/activities/{activityId}/copy` to `handler.go`; contract tests `TestDeleteActivity_ThenUndo_Contract_FR031`, `TestActivityDestinations_Contract_FR029`, `TestMoveActivity_Contract_FR029` (200; 400 with the dates message; 409), `TestCopyActivity_Contract_FR030` (201) in `handler_test.go`; `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` pass

### Provided to other features (backend)

- [x] T033 Write `TestCopyAll_ShiftsYears` (names kept, dates +1 year, 29 Feb → 28 Feb, ongoing kept, status Created, no log entries, `LineItems.CopyAll` per activity), `TestTypesNotAllowed`, `TestSubmitCheck_SameAsReadiness` and `TestSetStatus_LogsStatusChanged` in `backend/internal/activities/provided_test.go`
- [x] T034 Implement `CopyAll`, `TypesNotAllowed`, `SubmitCheck` and `SetStatus` in `backend/internal/activities/provided.go`; in `backend/internal/app/app.go` hand `CopyAll` and `TypesNotAllowed` to the activity seams of 004 (plan copy, model change) and 005 (campaign and initiative copy)

**Checkpoint**: `gofmt -l .`, `go vet ./...` and `go test ./...` pass in `backend/`.

---

## Phase 3: Frontend

### Frontend foundation (blocks the frontend stories)

- [x] T035 [P] Extend `frontend/src/components/CompletenessChip/CompletenessChip.tsx` with the optional props `below?: string` and `onOpen?: () => void` (plan, *Shared component change*); tests in `CompletenessChip.test.tsx` ("2 fields needed · 1 line item needs attention", `onOpen` called instead of the popover, unchanged behaviour without the props); show both variants in `frontend/src/features/dev/ComponentGalleryPage.tsx`
- [X] T036 [P] Write `frontend/src/features/activities/activitiesApi.ts`: query keys and hooks of the plan's *Data* section (`useActivity`, `useNewActivityForm`, `useEditActivityForm`, `usePlanActivities`, `useActivityDestinations`, `useCreateActivity`, `useUpdateActivity`, `useDeleteActivity`, `useMoveActivity`, `useCopyActivity`), types from `@/api/schema`, invalidation on settle
- [X] T037 [P] Write `frontend/src/features/activities/activityStatus.ts` (+ `activityStatus.test.ts`: word and tone for the six display statuses — Created neutral, Submitted review, Declined problem, Approved positive, In progress active, Complete done; banner text) and `ActivityStatusChip.tsx` (+ `ActivityStatusChip.test.tsx`) — FR-021
- [X] T038 [P] Write `frontend/src/features/activities/activityText.ts` (+ `activityText.test.ts`): `datesText` ("01 Mar 2026 – 31 May 2026", "Ongoing from 01 Mar 2026", "—"), `thingsText` ("1 thing", "5 things"), `lineItemsText`, `activityHref`, `activityEditHref(planId, id, field?)`
- [X] T039 Write `frontend/src/features/activities/activityForm.ts` (+ `activityForm.test.ts`): `FormState`, `fromDefaults(form)`, `fromActivity(activity)`, `isDirty`, `validate(state, form, mode, original?)` repeating every server rule of the plan's *Client checks* with the messages-table texts, `reapprovalChanges(original, state)` (planned cost, start, end, ongoing — old → new texts), `toCreateBody`, `toUpdateBody(state, version, restartApproval)`; tests per rule (FR-006, FR-009, FR-010, FR-011, FR-018, FR-027)
- [x] T040 Add the three activity routes of the plan to `frontend/src/app/router.tsx` (inside `RequireSignIn`, `handle.title` New activity / Activity / Edit activity) and a test in `frontend/src/app/router.test.tsx`

### User Story 1 — Create an activity (frontend)

- [X] T041 [US1] Write `frontend/src/features/activities/ActivityFormPage.test.tsx` (new mode, `mockApi` with the form endpoint and create): fields and markings as words, "From the initiative" labels, Campaign/Initiative read-only with the hint, line-items placeholder (FR-001, FR-002, FR-008, FR-009); type list of allowed types with default costs, choosing Event fills 25,000.00 "From the activity type" (FR-007); Save → toast "Activity 'Launch film' created" and the activity page (FR-003, FR-004); taken name + end before start + outside the initiative's dates → summary and field messages at once, nothing sent (FR-006, FR-009); Ongoing → end cleared + warning, can be unticked (FR-010); negative planned cost (FR-011); single workflow filled with its label (FR-012); Save and add another → fresh form + toast with View; Cancel with typed values → Keep editing / Discard changes; model without types → notice (edge case); 404 and error states
- [X] T042 [US1] Write `frontend/src/features/activities/ActivityForm.tsx` and `ActivityForm.module.css`: sections Basics, Dates & cost, More details and the new-only line-items card (Shell attributes and Goals slots filled by T052), using `TextField`, 003's `PersonPicker`, 005's `DateRangeField`, 002's `MoneyInput`, `InfoTip`; type change fills an empty planned cost; ongoing warning
- [X] T043 [US1] Write `frontend/src/features/activities/ActivityFormPage.tsx` and `ActivityFormPage.module.css` (new mode): load `useNewActivityForm`, `PageHeader` and breadcrumb, `validate` then `useCreateActivity`, server `fields` into `FormErrorSummary`, toasts and navigation, Save and add another (remount with a new key), `useUnsavedChanges`, loading / not found / error states, sticky footer — so T041 passes
- [X] T044 [US1] Add **Add activity** to 005's initiative row actions and initiative details panel (with its Activities list: name links and `ActivityStatusChip`), linking to `/plans/:planId/initiatives/:initiativeId/activities/new`, from `frontend/src/features/activities/activityTreeRows.tsx`; test "Add activity opens the new-activity page" in `activityTreeRows.test.tsx` (FR-001)

### User Story 2 — See an activity's status and everything it still needs (frontend)

- [X] T045 [US2] Write tests: `frontend/src/features/activities/ActivityPage.test.tsx` (header: breadcrumb, level tag, status and completeness chips "3 fields needed", ID, owner, "01 Mar 2026 – 31 May 2026", planned cost, unassigned, created/changed line, actions; banner for Submitted/Approved; Submit with problems → "Submitting needs 5 things — view list" and the panel focused, without problems → the coming-soon toast; sections in order with "—"; From finance collapsed; wrong `planId` redirects; loading, not found, error — FR-019 – FR-021, FR-025), `ReadyToSubmitPanel.test.tsx` (groups, order, links to `…/edit?field=…`, advice, "Ready to submit" — FR-022 – FR-024), `ActivitySections.test.tsx` (Details with "(inactive)" and "Not allowed by the plan's model" — FR-013, FR-020), `activityTreeRows.test.tsx` (row cells, red mark when the unassigned planned cost is negative, actions menu, row opens the page — FR-033)
- [X] T046 [P] [US2] Write `frontend/src/features/activities/ActivityHeader.tsx` and `ActivityHeader.module.css` (breadcrumb with 005's `campaignHref`/`initiativeHref`, 004's `LevelTag`, chips, facts with 003's display-currency formatter, banner, actions slot)
- [X] T047 [P] [US2] Write `frontend/src/features/activities/ReadyToSubmitPanel.tsx` and `ReadyToSubmitPanel.module.css` (collapsible `Card`, groups, links, focus target `#ready`)
- [X] T048 [P] [US2] Write `frontend/src/features/activities/ActivitySections.tsx` and `ActivitySections.module.css` (Details, Shell attributes and Goals read-only slots filled by T052, From finance `<details>`), `ActivityLineItemsSection.tsx` and `ActivityApprovalSection.tsx` (placeholder cards with the messages-table texts)
- [X] T049 [US2] Write `frontend/src/features/activities/ActivityPage.tsx` and `ActivityPage.module.css`: `useActivity`, `usePageTitle(name)`, redirect on a wrong `planId`, header, Submit handler (alert + open panel, or toast), sections in order, states — so the T045 page tests pass
- [X] T050 [US2] Write the rows in `frontend/src/features/activities/activityTreeRows.tsx` (`activityRowsByInitiative(summaries)` → 004's `PlanTreeRow`s under each initiative) and pass them to 004's `PlanTree` from 004's plan page (`usePlanActivities(planId)`) — so the T045 row tests pass

### User Story 3 — Shell attributes, % splits and goals (frontend)

- [X] T051 [US3] Write `frontend/src/features/activities/GoalsFields.test.tsx` (INQ 250,000 → MQL 5,000 and $2,500,000.00 "Calculated" with the ⓘ formula; empty → "—"; "-5" and "12.5" → "Enter a whole number of 0 or more."; Set later stages myself keeps typed values when INQ changes; turning it off lists the values that will be replaced — FR-018) and add to `ActivityFormPage.test.tsx`: Lubricants group with "Shown because the plan's business unit includes Lubricants", Mobility group absent; no business unit → the sentence and **Open the plan**; split three countries → 33.33 / 33.33 / 33.34 "100.00 % — complete"; 50.00 → "116.67 % — 16.67 % too much" and Save still sends; dependent attribute disabled with its hint and the cleared values listed before saving; hidden values sent back unchanged (FR-014 – FR-017)
- [X] T052 [US3] Write `frontend/src/features/activities/GoalsFields.tsx`; fill the Shell attributes section of `ActivityForm.tsx` with 002's `AttributeFields` (`idFor` → `field-attributes.<id>`) and the group / no-business-unit sentences; fill the read-only Shell attributes (002's `AttributeValuesView`) and Goals slots of `ActivitySections.tsx` — so T051 passes

### User Story 4 — Edit an activity (frontend)

- [X] T053 [US4] Add edit-mode tests to `frontend/src/features/activities/ActivityFormPage.test.tsx` (toast "Activity 'Launch film' saved"; the version sent is the one the form opened with even after a background refetch; `version_conflict` → `conflictMessage` at the top with values kept; Submitted + planned cost 120,000 → 150,000 → `ReapprovalDialog` listing the change, **Save and restart approval** sends `restartApproval: true`, **Keep editing** sends nothing; Approved + description only → no dialog; saved ongoing locked with its reason; workflow locked with its reason; type change with a planned cost → "Use the default planned cost of 25,000.00 USD for 'Event'?" with both answers; `?field=ownerId` focuses the owner — FR-007, FR-010, FR-012, FR-026 – FR-028) and write `ReapprovalDialog.test.tsx` (changes old → new, line-item spread line only when dates change and counts > 0)
- [X] T054 [P] [US4] Write `frontend/src/features/activities/ReapprovalDialog.tsx` (`ConfirmDialog`, texts of the messages table)
- [X] T055 [US4] Add edit mode to `frontend/src/features/activities/ActivityFormPage.tsx` and the locks and type question to `ActivityForm.tsx` (001's edit-form rule, `useUpdateActivity`, conflict handling, re-approval dialog, `?field=` focus) — so T053 passes

### User Story 5 — Move, copy and delete (frontend)

- [X] T056 [US5] Write `frontend/src/features/activities/MoveActivityDialog.test.tsx` (grouped destinations, search, unavailable ones visible with their reason, no-destination text, API refusal shown, toast — FR-029), `CopyActivityDialog.test.tsx` (current initiative preselected, text with the line-item count, opens the copy — FR-030), `DeleteActivityDialog.test.tsx` (every line of the dialog by status and counts, Cancel focused, **Delete activity** — FR-031) and `useActivityActions.test.tsx` (delete → plan page and "Activity 'Launch film' moved to the recycle bin" with Undo calling the undo endpoint; View history → `/activity-log?recordType=activity&recordId=…` — FR-031, FR-032)
- [X] T057 [P] [US5] Write `frontend/src/features/activities/DestinationList.tsx` (+ `.module.css`), `MoveActivityDialog.tsx` and `CopyActivityDialog.tsx`
- [X] T058 [P] [US5] Write `frontend/src/features/activities/DeleteActivityDialog.tsx`
- [X] T059 [US5] Write `frontend/src/features/activities/useActivityActions.tsx` (Copy, Move, View history, Delete with `useUndoableDelete`) and use it in the **More** menu of `ActivityHeader.tsx` and the row menu of `activityTreeRows.tsx` — so T056 passes

**Checkpoint**: `npm run check` passes in `frontend/`.

---

## Phase 4: End-to-end (needs Phases 2 and 3; one file, so in order)

- [ ] T060 [US1] Write `frontend/e2e/activities.spec.ts` test "US1: add an activity with only a name and a type" (as the plan's end-to-end table; a unique name from `uniqueName`; the initiative's planned cost on the plan tree before and after)
- [ ] T061 [US2] Add "US2: see status and everything missing" (A-1008: chips, panel items with links, Submit message, a link opens the form at the field)
- [ ] T062 [US3] Add "US3: attributes, splits and goals" (on a copy of A-1001 made through the API, so the seed stays untouched)
- [ ] T063 [US4] Add "US4: edit an approved activity" (on seeded A-1006, Approved and used by no other test: description keeps Approved; planned cost shows "This activity will need approval again" and returns it to Created; both changes in `GET /api/activity-log?recordId=A-1006`)
- [ ] T064 [US5] Add "US5: copy, move, delete and Undo" (copy A-1002, move the copy, refused move message, delete dialog texts, Undo restores the same ID)
- [ ] T065 Add the accessibility checks to `frontend/e2e/activities.spec.ts`: axe (WCAG 2.2 A/AA) on the activity page, the form with its error summary, and the Move, Copy, Delete and re-approval dialogs; a keyboard-only journey (Add activity → type → Save → More › Delete → Undo)

---

## Phase 5: Polish

- [ ] T066 [P] Record the plan's D-102 – D-112 in `docs/sdd/decision-log.md` (done at consolidation), close Q-008 for 006 (default applied), set 006's status in `docs/sdd/feature-map.md`, and add `activities/` to the package list of `backend/README.md`
- [ ] T067 [P] Check every screen against `docs/product/design-system.md` (§4 chip colours, §6 CRUD pattern, §6.6 locks) at 1280 × 720 and 390 px (no horizontal page scroll; the tree scrolls inside its card)
- [ ] T068 Run `gofmt -l .`, `go vet ./...`, `go test ./...` (backend), `npm run check` and `npm run test:e2e` (frontend); all green
- [ ] T069 Demonstrate each user story's *Independent Test* of the spec on freshly seeded data (`go run ./cmd/seed -reset`), including SC-001 (activity with one line item ≤ 15 steps, once 007 is merged)

---

## Requirement coverage

Every FR has at least one test task (T-numbers) and is implemented by the tasks of its story.

| FR | Test tasks | Implementation tasks |
|---|---|---|
| FR-001 | T014, T017, T041, T044, T060 | T015, T017, T043, T044 |
| FR-002 | T041 | T042, T043 |
| FR-003 | T014, T041 | T015, T043 |
| FR-004 | T014, T017, T041, T060 | T015, T017, T043 |
| FR-005 | T016, T030, T060 | T016, T018 |
| FR-006 | T009, T014, T039, T041 | T015, T039 |
| FR-007 | T014, T041, T053 | T015, T042, T055 |
| FR-008 | T014, T041 | T015, T042 |
| FR-009 | T008, T014, T039, T041 | T008, T015, T039 |
| FR-010 | T008, T014, T023, T039, T041, T053 | T015, T025, T042, T055 |
| FR-011 | T012, T014, T039, T041 | T015, T042 |
| FR-012 | T014, T023, T041, T053 | T015, T025, T042, T055 |
| FR-013 | T023, T045, T053 | T025, T048, T055 |
| FR-014 | T014, T051 | T015, T052 |
| FR-015 | T012, T027, T051 | T012, T028, T052 |
| FR-016 | T014, T051 | T015, T052 |
| FR-017 | T012, T014, T051, T062 | T012, T015, T052 |
| FR-018 | T010, T014, T027, T039, T051, T062 | T010, T028, T052 |
| FR-019 | T020, T045 | T021, T022, T046, T049 |
| FR-020 | T020, T045 | T021, T048, T049 |
| FR-021 | T011, T037, T045 | T011, T037 |
| FR-022 | T012, T035, T045 | T012, T035, T046 |
| FR-023 | T012, T045, T061 | T012, T047 |
| FR-024 | T012, T045 | T012, T047 |
| FR-025 | T045, T061 | T049 |
| FR-026 | T023, T024, T026, T053 | T025, T026, T055 |
| FR-027 | T023, T026, T039, T053, T063 | T025, T054, T055 |
| FR-028 | T023, T053, T063 | T025 |
| FR-029 | T029, T032, T056, T064 | T031, T032, T057 |
| FR-030 | T009, T029, T032, T056, T064 | T031, T032, T057 |
| FR-031 | T030, T032, T056, T064 | T031, T032, T058, T059 |
| FR-032 | T056 | T059 |
| FR-033 | T020, T045 | T021, T022, T050 |

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup**: T001 first (needs 002/003's schemas in the merged contract); T002 and T003 after it.
- **Backend (Phase 2)**: T004 → T005, T006 (parallel) → T007; T008 – T011 in parallel after T004; T012
  after T008, T010; T013 after T004–T006. Stories: US1 (T014 → T015 → T016 → T017 → T018 → T019) → US2
  (T020 → T021 → T022) → US4 (T023, T024 → T025 → T026) → US3 (T027 → T028) → US5 (T029, T030 → T031 →
  T032) → provided (T033 → T034).
- **Frontend (Phase 3)**: T035 – T038 in parallel; T039 after T038; T040 any time. Stories: US1 (T041 →
  T042 → T043; T044) → US2 (T045 → T046, T047, T048 → T049; T050) → US3 (T051 → T052) → US4 (T053 → T054 →
  T055) → US5 (T056 → T057, T058 → T059).
- **End-to-end (Phase 4)**: after both Phase 2 and Phase 3; T060 → T065 in order (one file).
- **Polish (Phase 5)**: T066, T067 any time after Phase 3; T068, T069 last.

### Parallel opportunities

- Backend and frontend run in parallel from T001/T002: the frontend tests use `mockApi()` with bodies shaped
  like the contract's examples; the backend tests use `fakes_test.go` and `apitest`.
- Inside each side, tasks marked [P] touch different files.
- Feature 007 can start against `activities.LineItems` once T006 is merged; feature 008 against
  `activities.Workflows`, `ApprovalHooks`, `SubmitCheck` and `SetStatus` once T034 is merged.

### Parallel example

```text
Backend:  T004 → (T005 + T006) → T007 → (T008 + T009 + T010 + T011) → T012 → T013 → US1 …
Frontend: (T035 + T036 + T037 + T038) → T039 → T040 → US1 …
Both:     T060 … T065
```

---

## Implementation Strategy

### MVP first (User Story 1)

1. Setup, backend foundation, US1 backend; frontend foundation, US1 frontend.
2. Stop and validate: an activity can be added from the plan tree with a name and a type and the plan's
   planned cost goes up (US1 independent test).

### Incremental delivery

1. + US2 → the activity page and tree rows; + US4 → editing with re-approval; + US3 → attributes, splits,
   goals; + US5 → move, copy, delete, Undo.
2. After each story: its tests and its end-to-end test; commit with the feature number, e.g.
   `feat(006): create activity (FR-001–FR-012)`.

### Notes

- Never edit `frontend/src/api/schema.d.ts` by hand: change `backend/api/openapi.yaml`, then
  `npm run api:types`.
- Keep user-visible messages exactly as in the plan's table; tests quote them.
- Mark each task `[X]` here when it is done.
