---

description: "Task list for feature 005 — Campaigns and Initiatives"
---

# Tasks: Campaigns and Initiatives

**Input**: Design documents from `specs/005-campaigns-and-initiatives/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–004 built (002's money and attribute rules,
003's people and person picker, 004's plans, plan page and plan tree).

**Tests**: required — constitution V. Within each group write the tests first and see them fail, then
implement. Every user-visible text comes from the plan's message tables; every rule from the plan and
data-model.md (when they seem to differ, the plan wins and the difference is reported).

**Organization**: Phase 1 (Setup) merges the contract, which both tracks start from. Phase 2 (Backend) and
Phase 3 (Frontend) are **independent tracks** that can run in parallel: the frontend works against
`mockApi` with contract-shaped answers, the backend against `apitest`. Inside each track: a foundation,
then one group per user story (US1–US4 are P1, US5–US6 P2). Phase 4 (End-to-end) needs both tracks.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: US1 … US6
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [X] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 `platform/date` comes from the shared foundation; T-tasks that create it only add what is missing (D-198).
- [ ] TC02 Add campaign and initiative entries to `recordHref` (D-194).
- [ ] TC99 Check the success criteria SC-002, SC-003, SC-005, SC-006, SC-007, SC-008, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup (shared by both tracks)

- [x] T001 Merge `specs/005-campaigns-and-initiatives/contracts/openapi.yaml` into `backend/api/openapi.yaml`: the `Campaigns` tag, the 10 paths (15 operations), the parameters `CampaignId`, `InitiativeId`, `Purpose` and the 24 schemas; check that `Money`, `AttributeValues` (002), `PersonRef` (003), `MissingField` and `ApprovalProgress` (004) exist there under these names and that no 005 schema name is already taken (report any difference instead of renaming) *(Done at consolidation, D-192.)*
- [x] T002 [P] Regenerate `frontend/src/api/schema.d.ts` with `npm run api:types` in `frontend/`; `npm run api:check` passes *(Done at consolidation, D-192.)*
- [X] T003 [P] Create the folders `backend/internal/campaigns`, `backend/internal/rollup`, `backend/internal/platform/date`, `frontend/src/features/campaigns`, `frontend/src/components/DateRangeField`

---

## Phase 2: Backend track

### Backend foundation (blocks the backend stories; order T004/T005 → T006 → T007 → T008 → T009 → T010 → T011 → T012 → T013)

- [ ] T004 [P] Write `backend/internal/platform/date/date.go` and `date_test.go` as the plan's *Shared date package* table: `Date` with JSON `YYYY-MM-DD` / `null` via pointer, `Parse`, `MustParse`, `New`, `String`, `Before`, `After`, `Compare`, `Year`, `Today(now)` (UTC), `YearStart`, `YearEnd`, `AddYears` (29 Feb → 28 Feb), `Format` ("01 Mar 2026"), `FormatRange`, `FormatRangeFull`; tests `TestParse_AndString`, `TestJSON_DateAndNull`, `TestAddYears_LeapDayBecomes28Feb`, `TestFormat_TwoDigitDayEnglishMonths`, `TestFormatRange_SameYearShortOtherwiseFull`, `TestToday_UsesUTC_DecisionB`
- [X] T005 [P] Write `backend/internal/rollup/rollup.go` and `rollup_test.go` as the plan's *Roll-ups* section (`Sum`, `Total`, `Remaining`, `OverBy`, `BudgetUsedPercent` with the integer half-up formula on cents, `OverBudget`, `LeftToPlan`, `ApprovalProgress`, `Timing`, `Goals`/`AddGoals`, `BudgetAndCost`/`Container`) on 002's `platform/money`; tests `TestSum_SkipsEmpty_NilWhenNothing_FR011`, `TestTotal_ZeroWhenNothing_FR022`, `TestRemaining_NegativeAllowed_FR010`, `TestRemaining_NilWithoutBudget_FR010`, `TestBudgetUsedPercent_RoundsHalfUp_FR011`, `TestOverBudget_ExactComparison_FR011`, `TestLeftToPlan_FR011`, `TestApprovalProgress_States_FR013`, `TestTiming_BeforeOnStartWithinOnEndAfter_FR013`, `TestAddGoals_NilWhenNothing_FR016`, `TestContainer_HandCalculated_SC004` (10 table cases to the cent)
- [X] T006 Write `backend/internal/campaigns/model.go`: stored `Campaign` and `Initiative` (data-model §1–§2, JSON names exact, `record.Meta` embedded, `Attributes` never nil), API types `CampaignView`, `InitiativeView`, `PlanCampaigns`, `PlanCampaignTotals`, `BudgetAndCost`, `ApprovalSummary`, `RecordCompleteness`, `MissingField` (shape of 004's schema), `GoalSums`, `CampaignCounts`, `InitiativeCounts`, `PlanRef`, `CampaignRef`, `Destination`, `DestinationList`, `MoveConsequences`, `TypeNotAllowed`, inputs `CampaignInput`, `CampaignUpdate` (`Version *int`), `InitiativeInput`, `InitiativeUpdate`, `CampaignMove`, `InitiativeMove`, `CampaignCopy`, `InitiativeCopy`, and the provided types `PlanRollup`, `RecordProblems`, `CopyCounts`, `InitiativeFacts` — every slice non-nil when encoded
- [X] T007 Write `backend/internal/campaigns/repository.go` (`Repository` with `GetCampaign`, `PutCampaign`, `DeleteCampaign`, `CampaignsOfPlan`, `GetInitiative`, `PutInitiative`, `DeleteInitiative`, `InitiativesOfCampaigns`) and `store_json.go` (`NewJSONRepository()`, tables `campaigns` and `initiatives`, results ordered by id)
- [X] T008 Write `backend/internal/campaigns/deps.go` exactly as the plan's *Dependencies through small interfaces*: `Deps`, `PlanFacts`, `PlanSource`, `PersonFacts`, `PersonSource`, `Settings`, `ActivitySummary`, `ActivityCopyCounts`, `Activities`, `NoActivities` (no summaries; `CopyAll` returns zero counts)
- [X] T009 Write `backend/internal/campaigns/fakes_test.go` (package `campaigns`): in-memory fakes of the four interfaces (settings fake: configurable attribute problems, missing attributes, group changes, allowed types), a fake `plan` kind and a fake `activity` kind (child of `initiative`) for history, and `newTestService(t)` with a JSON store in `t.TempDir()`, `ids.NewGenerator(ids.NewJSONCounters())`, a real `history.Service`, the fixed clock 2026-03-12 09:00 UTC and seeded fake plans P-1001 (2026, budget 1,200,000.00, Priya), P-1002 (2026, budget 800,000.00, other model and business unit), P-1003 (2027, no budget)
- [X] T010 Write `backend/internal/campaigns/validate.go`: every API message of the plan's *API validation and error messages* table as constants; `checkFields` (name, description, owner, both-or-neither dates, start ≤ end, budget USD ≤ 2 decimals ≥ 0) adding to an `errs.Problems`; `nameTaken` helpers for both levels (ignoring case, trimmed); `checkInsideCampaign` (R3, D-095); `checkChildrenInside` for initiatives and for activities (FR-007, ongoing start-only, undated ignored, one or several records message)
- [X] T011 Write `backend/internal/campaigns/views.go` (builders of `CampaignView`, `InitiativeView`, `PlanCampaigns` per the plan's *Views* table, using `rollup`, `PersonSource`, `Settings.MissingAttributes`, `Activities.Summaries`, ordering of FR-020) and `views_test.go`: `TestCampaignView_PlannedCostRollsUp_FR011`, `TestViews_NothingBelow_DashAndNoActivities_FR011_FR013`, `TestCampaignView_RemainingForInitiatives_FR010`, `TestViews_ParentRemainingIncludesOwnBudget_FR010`, `TestViews_ParentWithoutBudget_Nil_EdgeCase`, `TestViews_ApprovalProgressCountsApproved_FR013`, `TestViews_TimingAgainstToday_FR013`, `TestCampaignView_CompletenessOwnFields_FR014`, `TestCampaignView_RegionWithoutOptionsNotMissing_EdgeCase`, `TestViews_ItemsBelowNeedingAttention_FR014`, `TestViews_GoalSums_FR016`
- [X] T012 Write `backend/internal/campaigns/kinds.go`: `(*Service).Kinds()` returning `campaign` (parent `plan`) and `initiative` (parent `campaign`) with `Snapshot`, `ChildIDs`, `Remove`, `Put`, `Describe` (data-model §7 table) and `PrepareRestore` (Undo refused with `name_taken`, Restore suffix " (restored)", " (restored 2)", … with `Touch` and the note); `NewService(tx, repo, generator, hist, now, deps)` and the `Service` struct in `service.go`
- [X] T013 Write `backend/internal/app/campaign_adapters.go` (adapters from 004's plans, 001/003's people and 002's settings to `campaigns.PlanSource`, `PersonSource`, `Settings`) and wire `backend/internal/app/app.go`: build the campaigns service with `Activities: campaigns.NoActivities{}`, register its kinds right after 004's `plan` kind, call `campaigns.NewHandler(svc).Routes(router, signedIn)`; register the `campaign` and `initiative` levels with 004's `planService.RegisterLevel` as the plan's *Alignment with feature 004 as planned* table (`AddFacts` from `svc.PlanRollups`, `Copy` from `svc.CopyPlanContents`), with a test in `backend/internal/app/campaign_adapters_test.go` that a plan's facts count its campaigns, initiatives, campaign budgets and incomplete campaigns (`TestCampaignLevels_AddFacts_For004`) (`TestRoutesMatchContract` passes only once T036 has registered the last routes; checked in T037)

**Checkpoint**: `go build ./...` and `go test ./internal/rollup/ ./internal/platform/date/ ./internal/campaigns/` pass.

### US1 — Add campaigns to a plan (P1)

- [X] T014 [US1] Write create tests in `backend/internal/campaigns/service_test.go`: `TestCreateCampaign_OnlyNameRequired_FR005`, `TestCreateCampaign_EveryProblemAtOnce_FR005`, `TestCreateCampaign_NameUniqueInPlanIgnoringCaseAndSpaces_FR005`, `TestCreateCampaign_ReadableIDAndCreatedEntry_FR004_FR030`, `TestCreateCampaign_StoresWhatItReceives_DecisionD`, `TestCreateCampaign_NewOwnerMustBeActive_FR002`, `TestCreateCampaign_AttributeProblemsReportedWithOthers_FR009`, `TestCreateCampaign_OutsidePlanYearWarnsOnly_FR008`, `TestCreateCampaign_OverAllocationAllowed_FR010`, `TestCreateCampaign_PlanGone_NotFound`, `TestCreateCampaign_SameNameTwice_SecondRefused_EdgeCase`
- [X] T015 [US1] Implement `CreateCampaign`, `ListPlanCampaigns` (order and totals) and `GetCampaign` in `backend/internal/campaigns/service.go` (one Write/Read each; all problems collected inside the Write; log `created`)
- [X] T016 [US1] Write `backend/internal/campaigns/handler.go` (`NewHandler`, `Routes`) with `GET` and `POST /api/plans/{planId}/campaigns` and `GET /api/campaigns/{campaignId}`, and `handler_test.go` (package `campaigns_test`): `TestListPlanCampaigns_Contract`, `TestListPlanCampaigns_UnknownPlan404`, `TestCreateCampaign_Contract201`, `TestCreateCampaign_AllProblems400`, `TestGetCampaign_Contract`, `TestGetCampaign_404`
- [ ] T017 [US1] Add `addCampaigns` to `backend/seed/seed.go` (data-model §9: 4 campaigns, 8 initiatives in table order, attribute values by label from 002's seed, summary line ", 4 campaigns, 8 initiatives") and `TestSeed_CampaignsAndInitiatives` to `backend/seed/seed_test.go`

### US2 — Add initiatives to a campaign (P1)

- [X] T018 [US2] Write tests in `backend/internal/campaigns/service_test.go`: `TestCreateInitiative_DatesInsideCampaign_FR006`, `TestCreateInitiative_CampaignWithoutDates_NotChecked_EdgeCase`, `TestCreateInitiative_NameUniqueInCampaign_FR005`
- [X] T019 [US2] Implement `CreateInitiative` and `GetInitiative` in `backend/internal/campaigns/service.go`
- [X] T020 [US2] Add `POST /api/campaigns/{campaignId}/initiatives` and `GET /api/initiatives/{initiativeId}` to `handler.go`; tests `TestCreateInitiative_Contract201`, `TestCreateInitiative_OutsideCampaign400`, `TestGetInitiative_Contract`, `TestGetInitiative_404` in `handler_test.go`

### US3 — View and edit a campaign or initiative (P1)

- [X] T021 [US3] Write tests in `backend/internal/campaigns/service_test.go`: `TestUpdateCampaign_MissingVersion_FR018`, `TestUpdateCampaign_VersionConflict_FR018`, `TestUpdateCampaign_NoChangeNoVersionNoLog_FR030`, `TestUpdateCampaign_LogsEachChangedFieldWithTexts_FR030`, `TestUpdateCampaign_DatesLeavingInitiativesOutside_Refused_FR007`, `TestUpdateCampaign_UnchangedDatesNeverRefused_DecisionF`, `TestUpdateCampaign_AllowedInPublishedPlan_FR018`, `TestUpdateCampaign_InactiveOwnerKept_EdgeCase`, `TestUpdateInitiative_DatesLeavingActivitiesOutside_Refused_FR007`, `TestUpdateInitiative_OutsideCampaign_Refused_FR006`, `TestNoRoleChecks_AnyoneChangesAnyRecord_FR031`
- [X] T022 [US3] Implement `UpdateCampaign` and `UpdateInitiative` in `backend/internal/campaigns/service.go` (404 → version → all problems → `history.Diff` without attributes + `Settings.AttributeChanges` → no change, no log)
- [X] T023 [US3] Add `PATCH /api/campaigns/{campaignId}` and `PATCH /api/initiatives/{initiativeId}` to `handler.go`; tests `TestUpdateCampaign_Contract`, `TestUpdateCampaign_MissingVersion400`, `TestUpdateCampaign_Conflict409`, `TestUpdateInitiative_Contract`, `TestUpdateInitiative_Conflict409`, `TestAnyoneCanChangeAnyCampaign_FR031` in `handler_test.go`

### US4 — Budget, cost and progress in the plan tree (P1)

- [X] T024 [US4] Write tests in `backend/internal/campaigns/views_test.go`: `TestListPlanCampaigns_OrderByStartThenNameUndatedLast_FR020`, `TestListPlanCampaigns_Totals_FR022`, `TestViews_FollowChangesBelow_FR012_FR015` (the fake's activity planned cost and status change → the next list shows new planned cost, budget used and progress)
- [X] T025 [US4] Complete ordering and totals in `ListPlanCampaigns` (`backend/internal/campaigns/service.go`, `views.go`) until T024 passes
- [X] T026 [US4] Write `backend/internal/campaigns/provide.go` (`PlanRollups`, `InitiativeFacts`, `InitiativesOfPlan`) with tests `TestPlanRollups_CountsMoneyAndPublishProblems_For004` and `TestInitiativeFacts_For006` in `moves_test.go`

### US5 — Delete a campaign or initiative and undo it (P2)

- [X] T027 [US5] Write `backend/internal/campaigns/kinds_test.go`: `TestDeleteCampaign_EverythingInOneBinEntry_FR028`, `TestDeleteInitiative_FR028`, `TestUndoDelete_RestoresExactly_FR029`, `TestUndo_NameTaken_Refused_FR029`, `TestRestore_NameTaken_GetsRestoredSuffix_FR029`, `TestDescribe_ReadableValues_FR030`, `TestDelete_Gone_NotFound`
- [X] T028 [US5] Implement `DeleteCampaign` and `DeleteInitiative` (404 check, `history.MoveToBin`) in `backend/internal/campaigns/service.go`
- [X] T029 [US5] Add `DELETE /api/campaigns/{campaignId}` and `DELETE /api/initiatives/{initiativeId}` to `handler.go` (answer `history.DeleteResult`); tests `TestDeleteCampaign_ContractAndUndo`, `TestDeleteInitiative_Contract` in `handler_test.go`

### US6 — Move or copy a campaign or initiative (P2)

- [ ] T030 [US6] Write destination tests in `backend/internal/campaigns/moves_test.go`: `TestInitiativeDestinations_Move_UnavailableWithReasonAndFix_FR023`, `TestInitiativeDestinations_Copy_CurrentFirstSuggestedNames_FR026`, `TestCampaignDestinations_Move_SameYearOnly_FR023`, `TestCampaignDestinations_Consequences_FR024`, `TestDestinations_OverAllocationWarning_FR026`, `TestDestinations_NoneAvailable_EmptyList_EdgeCase`, `TestDestinations_BadPurpose_Refused`
- [ ] T031 [US6] Implement `CampaignDestinations`, `InitiativeDestinations` and `suggestCopyName` in `backend/internal/campaigns/moves.go`
- [ ] T032 [US6] Write move tests in `moves_test.go`: `TestMoveInitiative_KeepsIDDatesValuesAndChildren_FR025`, `TestMoveInitiative_RefusedWithEveryReason_FR023`, `TestMoveCampaign_SameYearOnly_FR023`, `TestMoveCampaign_KeepsOutsideYearWarning_FR023`, `TestMove_LogsParentChange_FR030`, `TestMove_VersionConflict_FR025`
- [ ] T033 [US6] Implement `MoveCampaign` and `MoveInitiative` in `moves.go` (log `moved` with the `parent` change)
- [ ] T034 [US6] Write copy tests in `moves_test.go`: `TestCopyCampaign_CopiesEverythingBelow_FR027`, `TestCopyCampaign_AllOrNothing_FR027`, `TestCopyCampaign_NameCheckedInDestination_FR026`, `TestCopyInitiative_DatesMustFit_FR026`, `TestCopy_OneCopiedEntryWithDetails_DecisionH`, `TestSuggestCopyName_CopyThenNumbers_R13`, `TestSuggestCopyName_ShortensLongNames_EdgeCase`, `TestCopyPlanContents_ShiftsDatesKeepsNames_For004`
- [ ] T035 [US6] Implement `CopyCampaign`, `CopyInitiative` (one Write; `Activities.CopyAll` per initiative; log `copied` on the new record only) and `CopyPlanContents` (campaigns and initiatives only, returning the old → new IDs for 004's `CopyJob`) in `moves.go`
- [ ] T036 [US6] Add the six move/copy/destination routes to `handler.go`; tests `TestCampaignDestinations_Contract`, `TestMoveCampaign_Contract`, `TestMoveCampaign_Refused400`, `TestCopyCampaign_Contract201`, `TestInitiativeDestinations_Contract`, `TestMoveInitiative_Contract`, `TestCopyInitiative_Contract201` in `handler_test.go`
- [ ] T037 [US6] Run `go test ./internal/app/`: `TestRoutesMatchContract` and `TestEveryProtectedOperationRequiresSignIn_FR005` pass with the 15 new routes
- [ ] T038 [US6] Backend done: `gofmt -l .` prints nothing, `go vet ./...` and `go test ./...` pass in `backend/`

---

## Phase 3: Frontend track

### Frontend foundation (blocks the frontend stories)

- [ ] T039 [P] Write `frontend/src/components/DateRangeField/DateRangeField.tsx`, `DateRangeField.module.css`, `DateRangeField.test.tsx` (props and behaviour of the plan's *New shared component*) and export it from `frontend/src/components/index.ts`
- [ ] T040 [P] Add `formatDateRange(start, end)` to `frontend/src/lib/format.ts` with cases in `frontend/src/lib/format.test.ts` ("01 Feb – 30 Nov 2026", "01 Dec 2025 – 31 Jan 2026", "—")
- [X] T041 [P] Write `frontend/src/features/campaigns/texts.ts`: every text of the plan's *User-visible messages (web)* table as constants and small functions (counts with singular forms, contents sentences, toasts)
- [X] T042 Write `frontend/src/features/campaigns/campaignApi.ts`: query keys of the plan's *Data and refresh* table, `usePlanCampaigns`, `useCampaign`, `useInitiative`, `useDestinations(level, id, purpose)` (`staleTime: 0`), mutations for create, update, delete, move and copy of both levels (each awaiting `refreshPlanning` in `onSettled`), `refreshPlanning(queryClient)` (invalidates `['plans']`, `['plan']`, `['campaigns']`, `['initiatives']`, `['recycle-bin']`, `['activity-log']`)
- [X] T043 [P] Write `frontend/src/features/campaigns/StatusChips.tsx` (`ApprovalProgressChip`, `TimingChip`, `OutsideYearNote`, `ItemsBelowLink`) with `StatusChips.test.tsx` (FR-013, FR-015: words, tones, "0 of 5", Timing "—" tip)
- [X] T044 [P] Write `frontend/src/features/campaigns/BudgetCost.tsx` (`RemainingBudgetLine` with exact cent arithmetic from `@/lib/money`, `BudgetCostSection`, `GoalsSection`) with `BudgetCost.test.tsx` (FR-010, FR-011)
- [ ] T045 [P] Write `frontend/src/features/campaigns/extensions.ts`: `initiativeExtraActions(initiative, navigate): ActionsMenuItem[]` and `useRowsBelowInitiatives(planId)` (004's `TreeNode` children per initiative id), both empty until feature 006
- [X] T046 Write `frontend/src/features/campaigns/useRecordActions.ts`: state and handlers that open the Move, Copy and Delete dialogs for a campaign or initiative from a row or a panel (dialog components arrive in US5–US6; until then the items are not listed)

**Checkpoint**: `npm run check` passes.

### US1 — Add campaigns to a plan (P1)

- [X] T047 [US1] Write `frontend/src/features/campaigns/CampaignForm.tsx` (+ `.module.css`): sections Basics, Budget, Shell attributes per the plan's *Forms* table, field ids `campaign-<field>`, default hints while unchanged, R4 warning, `RemainingBudgetLine`, 002's `AttributeFields level="campaign"`
- [X] T048 [US1] Write `frontend/src/features/campaigns/NewCampaignPanel.tsx` and `NewCampaignPanel.test.tsx`: US1 scenarios 1–5, FR-001, FR-003, FR-004 (toast with **View**, navigate with `state.highlight`), FR-005 (400 problems all shown, summary focused), FR-008, FR-009 (region disabled, single region filled with its message), **Save and add another**, **Cancel** with typed values asks **Keep editing** / **Discard changes**, parent-gone and loading states
- [X] T049 [US1] Write `frontend/src/features/campaigns/campaignRows.ts` (004's `TreeNode`s: campaign nodes with their initiative nodes as `children` — name, href, child count, owner, dates, chips, budget/remaining, planned cost, budget used, actions, highlighted; campaigns start expanded — see the plan's *Alignment with feature 004 as planned*) and `frontend/src/features/campaigns/useCampaignTreeContent.tsx` (rows, **Add campaign** in toolbar, plan row ⋯ and empty tree, totals row, dialogs slot, loading and error) with `useCampaignTreeContent.test.tsx` (FR-001, FR-012 refresh after a create)
- [ ] T050 [US1] Add the route `campaigns/new` under `/plans/:planId` in `frontend/src/app/router.tsx`; plug into 004's plan page: fill `planNode.children` in `frontend/src/features/plans/tree/usePlanTreeNodes.ts` from `useCampaignTreeContent(planId)`, pass **Add campaign** as `PlanTree` `toolbarActions` and the totals row as `footer` in `PlanPage.tsx`; add the plan row ⋯ / empty-tree **Add campaign** and highlight props to `PlanTree` only if 004 has not built them (report it)

### US2 — Add initiatives to a campaign (P1)

- [X] T051 [US2] Write `frontend/src/features/campaigns/InitiativeForm.tsx` (+ css): read-only Campaign with "Change it with Move", `DateRangeField` with `allowed` = the campaign's dates and the range hint, "Remaining campaign budget"
- [X] T052 [US2] Write `frontend/src/features/campaigns/NewInitiativePanel.tsx` and `NewInitiativePanel.test.tsx`: US2 scenarios 1–3, FR-006 (allowed range, the API's R3 message under the dates), defaults "From the campaign" / "The campaign's dates" (empty when the campaign has none), highlight under the expanded campaign
- [X] T053 [US2] Add the route `campaigns/:campaignId/initiatives/new` to `router.tsx` and the campaign row's **Add initiative** quick action (tooltip "Add initiative to <name>") and ⋯ item in `campaignRows.ts`

### US3 — View and edit a campaign or initiative (P1)

- [X] T054 [US3] Write `frontend/src/features/campaigns/CampaignPanel.tsx` (+ css; details and edit modes) and `CampaignPanel.test.tsx`: US3 scenarios 1–4, FR-016 (header, chips, facts, sections, initiatives list and empty text, ⋯ More items), FR-017 (opened by address; wrong plan address replaced), FR-018 (edit form, version kept, save refreshes, 409 `conflictMessage` keeps values, Discard guard), FR-007 message under the dates, cluster removal note, `?focus=` from the completeness chip, View history link, loading, error + Try again, not found + **Back to the plan**
- [X] T055 [US3] Write `frontend/src/features/campaigns/InitiativePanel.tsx` (+ css) and `InitiativePanel.test.tsx`: details (campaign link, remaining campaign budget, activities line), edit with R3/FR-007 messages, More items with `initiativeExtraActions`, not found
- [ ] T056 [US3] Add the routes `campaigns/:campaignId`, `campaigns/:campaignId/edit`, `initiatives/:initiativeId`, `initiatives/:initiativeId/edit` to `router.tsx`; row **View** / **Edit** items navigate to them; add `campaign` and `initiative` entries (record and `…/edit?focus=<field>` links) to 004's `RECORD_LINKS` in `frontend/src/features/plans/recordLinks.ts`, with cases in its test

### US4 — Budget, cost and progress in the plan tree (P1)

- [X] T057 [US4] Write `frontend/src/features/campaigns/campaignRows.test.ts`: FR-019 (every cell per level, "(inactive)" owner, R4 flag), FR-020 (API order kept, campaigns start expanded, initiatives collapsed), FR-021 (⋯ items per level, separated Delete, extension actions), FR-022 (totals text with singular forms, "Allocated to campaigns", planned cost), "Over-allocated by" in red beneath a campaign budget; complete `campaignRows.ts` until it passes

### US5 — Delete a campaign or initiative and undo it (P2)

- [X] T058 [US5] Write `frontend/src/features/campaigns/DeleteRecordDialog.tsx` and `DeleteRecordDialog.test.tsx`: US5 scenarios 1–2, FR-028 (title, contents from counts, "Plan totals will update.", R14 sentence only when approved > 0 with singular form, recycle-bin sentence, Cancel focused, red **Delete campaign**), FR-029 (panel closes, `useUndoableDelete` Undo message for 10 s with fake timers, refresh), 404 toast; wire it into `useRecordActions.ts`, row ⋯ and panel More

### US6 — Move or copy a campaign or initiative (P2)

- [ ] T059 [P] [US6] Write `frontend/src/features/campaigns/MoveDialog.tsx` (on 004's `FormDialog`) and `MoveDialog.test.tsx`: US6 scenarios 1–3, FR-023 (rule, disabled destinations with reasons, empty text), FR-024 (consequences block), FR-025 (version sent, toast with **View**, highlight), budget warning, "Choose where to move it.", 400/409 shown in the dialog
- [ ] T060 [P] [US6] Write `frontend/src/features/campaigns/CopyDialog.tsx` (on 004's `FormDialog`) and `CopyDialog.test.tsx`: US6 scenarios 4–5, FR-026 (suggested name follows the destination until edited, current first with "Current", counts and R13 sentences, warning, name error from a 400), FR-027 (toast "… created" with **View**, new row highlighted)
- [ ] T061 [US6] Wire **Copy** and **Move to…** into `useRecordActions.ts`, the row ⋯ menus (`campaignRows.ts`) and both panels' More menus; extend `useCampaignTreeContent.test.tsx` to open each dialog from a row

**Checkpoint**: `npm run check` passes in `frontend/`.

---

## Phase 4: End-to-end (needs both tracks)

- [ ] T062 [US1] Write `frontend/e2e/campaigns.spec.ts` journey 1 (US1 on FY2026 Mobility China as Mei Lin: defaults, cluster → region message, over-allocation in red, save, toast, highlighted row, totals row up by the new budget; SC-001 steps)
- [ ] T063 [US2] Add journey 2 (US2 under Helix launch: **Add initiative**, name, **Save**; the R3 message for a start before the campaign)
- [ ] T064 [US3] Add journey 3 (US3/US4: open `/plans/P-1001/campaigns/C-1001`, check details, edit budget and owner, row and plan summary card update, FR-007 problem when narrowing dates past "Digital & social", row chips and budget used)
- [ ] T065 [US5] Add journey 4 (US5: create a campaign with an initiative, delete with the dialog's counts, **Undo** within 10 s, same IDs and totals back)
- [ ] T066 [US6] Add journey 5 (US6: move an initiative, an unavailable destination shows its reason; copy a campaign → "(copy)" highlighted with new IDs)
- [ ] T067 Add journey 6 (axe WCAG 2.1 AA on the create panel with errors, a details panel, the move dialog and the delete dialog; keyboard-only: a row's ⋯ menu and **Add initiative**); `npm run test:e2e` passes

---

## Phase 5: Polish

- [ ] T068 [P] Document the `rollup` and `platform/date` packages in `backend/README.md` (*Packages*) and `docs/architecture/repository-layout.md`; add `features/campaigns` and `DateRangeField` to `frontend/README.md` where the folders are listed
- [ ] T069 [P] Update `docs/sdd/feature-map.md` (005 status) and add the numbered decisions of the plan (D-090 … D-101, as numbered by the consolidation step) to `docs/sdd/decision-log.md`
- [ ] T070 Demo check on fresh seed data (`go run ./cmd/seed -reset`, API and web app): FY2026 Lubricants Europe shows 2 campaigns, 4 initiatives, "Allocated to campaigns" 1,150,000.00 and remaining 50,000.00; Retail growth "2 fields needed"; Fleet cards Timing "—"; Brand platform R4 warning; Sponsorships' campaign over-allocated by 50,000.00; every message matches the plan's tables

---

## Dependencies & Execution Order

- **Setup (T001–T003)** first; it blocks both tracks.
- **Backend track**: foundation T004–T013 (T004 and T005 in parallel), then US1 → US2 → US3 → US4 → US5 → US6
  (each story's tests before its implementation; the handler tasks of one story touch `handler.go` and
  `handler_test.go`, so stories run in order within the track). T037–T038 last.
- **Frontend track**: foundation T039–T046 (T039, T040, T041, T043, T044, T045 in parallel; T042 before
  T046), then US1 → US2 → US3 → US4 → US5 → US6; T059 and T060 in parallel.
- **The two tracks never wait for each other** until Phase 4; both follow the merged contract.
- **End-to-end (T062–T067)** after both tracks; **Polish** last.
- Outside this feature: 002, 003 and 004 must be built (money, attribute rules and fields, people and
  picker, plans, plan page and tree). Feature 006 later replaces `NoActivities{}` and fills `extensions.ts`.

## Parallel example

```text
# Backend foundation
T004 platform/date        T005 rollup
# Frontend foundation (another developer, same time)
T039 DateRangeField   T040 formatDateRange   T041 texts.ts   T043 StatusChips   T044 BudgetCost   T045 extensions.ts
# US6 frontend
T059 MoveDialog       T060 CopyDialog
```

## Implementation Strategy

### MVP first (User Stories 1–4, all P1)

1. Setup (T001–T003), then the backend foundation and US1–US4 (T004–T026) alongside the web foundation and
   US1–US4 (T039–T057).
2. Stop and validate with journeys 1–3 (T062–T064): add a campaign and an initiative in FY2026 Mobility
   China and under Helix launch, open and edit Helix launch, and check every row, the totals row and the
   plan's summary cards against hand-calculated values (SC-004).

### Incremental delivery

1. + US5 (delete with Undo: T027–T029, T058, T065) → + US6 (move and copy: T030–T037, T059–T061, T066).
2. After each story: its tests, its e2e journey, then commit with the feature number and FRs, e.g.
   `feat(005): move and copy campaigns and initiatives (FR-023 – FR-027)`.
3. Then accessibility (T067) and Polish (T068–T070).

### Parallel team strategy

- One backend agent (Phase 2) and one frontend agent (Phase 3) start together from the merged contract
  (T001–T002); neither waits for the other until Phase 4.
- Feature 006 can start its design against `campaigns.Activities`, `InitiativeFacts` and `extensions.ts`
  as soon as T008, T026 and T045 are done.

## FR coverage (every FR has a test task)

| FR | Test tasks | FR | Test tasks |
|---|---|---|---|
| FR-001 | T048, T049, T062 | FR-017 | T054, T064 |
| FR-002 | T014, T048 | FR-018 | T021, T023, T054 |
| FR-003 | T014, T048 | FR-019 | T057, T064 |
| FR-004 | T014, T016, T048, T062 | FR-020 | T024, T057 |
| FR-005 | T014, T016, T018 | FR-021 | T057, T067 |
| FR-006 | T018, T020, T052, T063 | FR-022 | T005, T024, T057, T062 |
| FR-007 | T010, T021, T054, T064 | FR-023 | T030, T032, T059, T066 |
| FR-008 | T014, T057 | FR-024 | T030, T059 |
| FR-009 | T011, T014, T048, T062 | FR-025 | T032, T059, T066 |
| FR-010 | T005, T011, T044, T062 | FR-026 | T030, T034, T060 |
| FR-011 | T005, T011, T044 | FR-027 | T034, T060, T066 |
| FR-012 | T024, T049, T064 | FR-028 | T027, T029, T058, T065 |
| FR-013 | T005, T011, T043 | FR-029 | T027, T029, T058, T065 |
| FR-014 | T011, T013, T026 | FR-030 | T014, T021, T027, T032, T034 |
| FR-015 | T024, T043 | FR-031 | T021, T023 |
| FR-016 | T005, T011, T054 | | |

## Notes

- Commit after each task group, e.g. `feat(005): add campaigns to a plan (US1)`.
- Test names end with the requirement they prove; texts are quoted from the plan's tables, never invented.
- When an expected name from 002, 003 or 004 differs (plan *Expects from other features*), adapt in
  `internal/app/campaign_adapters.go` or at the import, and report it; do not change other features' code
  beyond the shared files listed in the plan.
