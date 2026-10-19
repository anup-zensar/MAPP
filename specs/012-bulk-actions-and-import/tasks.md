---

description: "Task list for feature 012 — Bulk Actions and Import"
---

# Tasks: Bulk Actions and Import

**Input**: Design documents from `specs/012-bulk-actions-and-import/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [data-model.md](data-model.md),
[contracts/openapi.yaml](contracts/openapi.yaml); features 001–011 built — in particular 010's All work
grid, pager, `platform/sheet` writer and `lib/download.ts`, 004's `PlanId` parameter and plan page,
005/006's single-record create/update/copy/move/delete and `ValidateNew`, 008's submit check, 003's
`PersonPicker`.

**Tests**: required (constitution V). Write the tests of each task first and see them fail. Test names
carry the FR they prove; texts come from the plan's *User-visible messages*.

**Organization**: Setup, then **Backend** and **Frontend** phases run by two agents in parallel from the
contract (T001), then End-to-end and Polish. Story tags: US1 Select and change owner or a field (P1) ·
US2 Import (P1) · US3 Submit many (P2) · US4 Copy, move, delete (P2).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- Paths are relative to the repository root.

---

## Consolidation changes (2026-10-19)

Read the *Consolidation* section at the end of [plan.md](plan.md) and the build guide [docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md) first; they win over the tasks below.

- The Phase 1 tasks that merge the contract and regenerate `schema.d.ts` are done (marked below); verify with `npm run api:check` (D-192).
- [ ] TC00 When an operation is served, remove it from `notYetServed` in `backend/internal/app/app_test.go` (D-192).
- [ ] TC01 The bulk submit adapter passes the activity version read in the same transaction (008 refuses stale versions).
- [ ] TC99 Check the success criteria SC-001, SC-002, SC-003, SC-004, SC-006, which no task above names, in the final review (timed end-to-end run or moderated session, as each criterion says) and record the result in `checklists/`.

## Phase 1: Setup

- [x] T001 Merge `specs/012-bulk-actions-and-import/contracts/openapi.yaml` into `backend/api/openapi.yaml` (tags, 10 paths / 11 operations, parameters `BulkActionId`, `ImportId`, schemas `Bulk*`, `Import*`, `PlanImportPage`; reuses 001's `RecycleBinEntry`, `RestoreResult`, `ActorRef` and 004's `PlanId`) and run `npm run api:types` in `frontend/`; `npm run api:check` passes *(Done at consolidation, D-192.)*
- [ ] T002 [P] Create the folders `backend/internal/bulk/`, `backend/internal/imports/`, `frontend/src/features/bulk/`, `frontend/src/features/imports/`, `frontend/src/components/SelectionBar/`

**Checkpoint**: contract final; backend and frontend start in parallel.

---

## Phase 2: Backend (Go)

### Foundation

- [ ] T003 [P] Write `backend/internal/history/note.go` (`WithNote(ctx, note) context.Context`; `Log` appends the note to `Details` as its own sentence) and `note_test.go`: `TestWithNote_AppendedToDetails_FR023`, `TestWithNote_InnermostWins`; existing history tests stay green
- [ ] T004 [P] Add `ReadJSONUpTo(w, r, dst, maxBytes)` to `backend/internal/platform/web/web.go` (`ReadJSON` = `ReadJSONUpTo` with 1 MB) and `TestReadJSONUpTo_LimitAndRules` to `web_test.go`
- [ ] T005 [P] Write `backend/internal/platform/sheet/xlsx_read.go` (`ReadXLSX`: "Activities" or first sheet, shared/inline strings, numbers, date cells by built-in and custom formats, row numbers kept) and `csv_read.go` (`ReadCSV`: BOM, `,` or `;`, quotes) with `TestReadXLSX_SharedInlineNumbersDates`, `TestReadXLSX_FirstOrActivitiesSheet`, `TestReadXLSX_RoundTripWithWriter`, `TestReadCSV_BOMCommaSemicolonQuotes`
- [ ] T006 Write `backend/internal/bulk/model.go` (request, result, outcome, effect, delete summary, field, target types per the contract; stored `Action`), `planning.go` (`Planning` interface and `RecordInfo`, plan *Package bulk*), `repository.go` + `store_json.go` (table `bulk_actions`), and the test helper `fake_test.go` (in-memory `Planning` with plans, campaigns, initiatives, activities, line items, statuses, models, business units and rule failures, writing real `history` entries)
- [ ] T007 [P] Write `backend/internal/imports/model.go` (upload, run, check, problem, stored `Import`, summary, detail, undo preview), `planning.go` (`Planning` interface, plan *Package imports*), `repository.go` + `store_json.go` (table `imports`) and `fake_test.go`

### US1 — Select many rows and change their owner or a field (P1)

- [ ] T008 [US1] Write `backend/internal/bulk/service.go` (`NewService`, `run(ctx, a, req, commit)` with request checks, duplicate and nested-record handling, `history.WithNote`, one Write, rollback for previews, outcome mapping: `*errs.Error` → skipped, `not_found` → failed "No longer exists", other errors → nothing saved) and `messages.go`; `service_test.go`: `TestRun_LimitOf1000_FR006`, `TestRun_MixedLevelsRefusedForFieldCopyMove_FR007`, `TestRun_OtherActionsSkipRecordsTheyDontApplyTo_FR007`, `TestRun_RequestProblemsAllAtOnce`, `TestPreview_EqualsRunAndSavesNothing_D_n`, `TestResult_CountsVerbAndReasons_FR021`, `TestRun_RecordsIndependent_GoneIsFailed_OthersKept_FR022`, `TestRun_UnexpectedErrorSavesNothing_D_n`, `TestRun_OneLogEntryPerRecordWithNote_FR023`
- [ ] T009 [US1] Write `backend/internal/bulk/actions.go` for `changeOwner` and `changeField` and `Service.Fields`; `actions_test.go`: `TestChangeOwner_CampaignsInitiativesActivities_LineItemsKeepOwners_FR008`, `TestChangeOwner_AlreadySetIsSkipped_D_o`, `TestFields_OfferedPerLevel_NoNamesMoneySplitsOngoing_FR009`, `TestChangeField_MultipleChoiceReplacesValues_FR009`, `TestChangeField_ClearNotForRequired_FR009`, `TestChangeField_SkipsTypeNotAllowedByModel_R7_FR010`, `TestChangeField_SkipsDatesOutsideParentOrChildren_FR010`, `TestChangeField_SkipsAttributeNotUsedByBusinessUnit_R9_FR010`, `TestChangeField_SkipsOptionNotFittingParent_FR010`, `TestChangeField_TypeChangeKeepsPlannedCost_FR010`, `TestChangeField_ParentAttributeClearsDependentValues_D_w`, `TestPreview_ListsReapproval_FR011`, `TestChangeField_DatesOnApproved_LeaveUnchangedSkipped_FR011`, `TestChangeField_DatesOnApproved_RestartApprovalToCreated_FR011`
- [ ] T010 [US1] Write `backend/internal/bulk/planning_adapter.go` (`NewPlanningAdapter(…)` on the services listed in the plan's *Dependencies*: 002 settings, 003 people, 005 campaigns, 006 activities, 008 approvals) and `planning_adapter_test.go` (`TestPlanningAdapter_SeedRecordsAndOwnerChange_FR008` on `apitest` seed data)
- [ ] T011 [US1] Write `backend/internal/bulk/handler.go` (routes `GET /api/bulk-actions/fields`, `POST /api/bulk-actions/preview`, `POST /api/bulk-actions`) and wire `bulk` in `backend/internal/app/app.go`; `handler_test.go` (`package bulk_test`): `TestFields_Contract`, `TestPreview_Contract`, `TestRun_Contract` (changeOwner and changeField on seed activities; activity-log entries carry the note)
- [ ] T012 [US1] Write `backend/internal/bulk/perf_test.go`: `TestBulk_500Records_SC007` (500 activities, change owner, under 10 s; skipped with `-short`)

### US2 — Import activities from a template (P1)

- [ ] T013 [P] [US2] Write `backend/internal/imports/columns.go` (the template columns of data-model §3.1 for a plan's business unit, headings, aliases, markings, column ↔ activity field map of §3.3) and `columns_test.go`: `TestHeadings_AnyOrderAndCase_Aliases_Unrecognised_FR026`, `TestHeadings_MissingRequiredStopsCheck_FR026`, `TestColumns_OnlyAttributesOfThePlansBusinessUnit_R9_FR025`
- [ ] T014 [P] [US2] Write `backend/internal/imports/parse.go` (file → `sheet.Table` → typed rows: formats of data-model §3.2, empty and marking rows skipped, limits of §3.5) and `parse_test.go`: `TestParse_LimitsAndFormats_FR026`, `TestParse_CellFormats_D_z`
- [ ] T015 [US2] Write `backend/internal/imports/template.go` and `Service.Template` / `Service.List` in `service.go`; `template_test.go`: `TestTemplate_ColumnsMarkingsAllowedValuesInstructions_FR025`, `TestTemplate_RefusedWithoutCampaignAndInitiative_FR024`, `TestList_CanImportAndReason_FR024`
- [ ] T016 [US2] Write `Service.Check`, `Service.Import`, `Service.Get`, `Service.Undo` and `run` in `backend/internal/imports/service.go` (plan *Package imports* steps 1–8); `service_test.go`: `TestCheck_EveryProblemOfEveryRowAtOnce_FR027`, `TestCheck_NamesUniqueInInitiativeAndInFile_FR027`, `TestCheck_DeactivatedOptionReported_EdgeCase`, `TestCheck_NeededToSubmitEmptyAllowed_DefaultsAndEvenSplit_FR028`, `TestCheck_CampaignAndInitiativeByIdOrName_IdFromOtherPlan_FR029`, `TestCheck_CreateMissingOnAndOff_FR030`, `TestCheck_CreatesNothing_FR031`, `TestImport_CreatedNoLineItemsWholeCostUnassigned_FR032`, `TestImport_OnlyConfirmedRows_FailedAndLeftOut_D_x`, `TestImport_StoredAndListedForThePlan_FR033`, `TestImport_LogEntriesCarryImportNote_FR023`, `TestUndoImport_MovesEverythingCreatedAndAddedSinceToBin_RecordsWho_FR034`, `TestUndoImport_PreviewCountsChangedAndAdded_EdgeCase`, `TestUndoImport_Twice_AlreadyResolved`, `TestTemplate_FilledWithAllowedValuesImportsWithoutProblems_SC005`
- [ ] T017 [US2] Write `backend/internal/imports/planning_adapter.go` (on 002, 003, 004, 005, 006) and `planning_adapter_test.go` (`TestPlanningAdapter_SeedPlanTemplateAndCheck_FR025` on `apitest` seed data)
- [ ] T018 [US2] Write `backend/internal/imports/handler.go` (the six import routes; `ReadJSONUpTo` with 7 MB for check and import; file too large → the plan's message) and wire `imports` in `app.go`; `handler_test.go` (`package imports_test`): `TestListImports_Contract`, `TestTemplate_Contract` (opens with `sheet.ReadXLSX`), `TestCheck_Contract` (200; 400 listing every missing column), `TestImport_Contract`, `TestGetImport_Contract`, `TestUndoImport_Contract` (200 then 409); `perf_test.go` `TestCheck_1000Rows_SC007`

### US3 — Submit many activities for approval (P2)

- [ ] T019 [US3] Add `submit` to `backend/internal/bulk/actions.go` (008's check first, every reason grouped as "n fields needed: …; line item L-…: …", then submit or re-submit); tests `TestSubmit_CreatedAndDeclined_FR012`, `TestSubmit_SkipsWithEveryReasonAtOnce_FR013`, `TestSubmit_CampaignsAndApprovedSkipped_FR013`; extend `TestRun_Contract` with a submit on seed activities (008's notification created)

### US4 — Copy, move or delete many records (P2)

- [ ] T020 [US4] Add `copy` and `move` with their effects to `backend/internal/bulk/actions.go`; tests `TestCopy_EverythingBelow_CreatedNoHistory_R13_FR014`, `TestCopy_NameMadeUniqueOnlyOnClash_D_s`, `TestCopy_ActivitiesWithTypeNotAllowedSkipped_D_t`, `TestCopy_DatesOutsideTargetSkipped_FR017`, `TestMove_RulesOfD033_FR015`, `TestMove_KeepsIdsStatusLineItems_FR015`, `TestMove_NameTakenOrAlreadyThereSkipped_FR017`, `TestMove_DatesNeverChange_FR017`, `TestEffects_OtherModelAndBusinessUnit_FR018`
- [ ] T021 [US4] Add `Service.Targets` and `POST /api/bulk-actions/targets`; `targets_test.go`: `TestTargets_CopyAnyPlan_MoveRules_ExcludesSelectionAndBelow_FR016`, `TestTargets_SelectionAcrossPlansHasNoMoveTarget_D_u`, `TestTargets_SearchAndPath_FR016`; `TestTargets_Contract` in `handler_test.go`
- [ ] T022 [US4] Add `delete` (top-level records only, `deleteSummary`, bin entries stored on the `Action`) and `Service.Undo` + `POST /api/bulk-actions/{bulkActionId}/undo`; tests `TestDelete_SummaryCountsNestedOnce_FR019`, `TestDelete_OneBinEntryPerTopLevelRecord_FR020`, `TestUndo_RestoresAllInOneStep_FR020`, `TestUndo_RefusedAfterTwoMinutesChangesNothing_D_r`, `TestUndo_OnlyForDeletes`; `TestUndoBulkDelete_Contract` (200; 409 after `Advance(3 min)`), `TestEveryBulkAndImportOperationRequiresSignIn` in `handler_test.go`

**Checkpoint**: `go test ./...` green; `TestRoutesMatchContract` green.

---

## Phase 3: Frontend (web app)

### Foundation

- [ ] T023 [P] Write the shared component `frontend/src/components/SelectionBar/SelectionBar.tsx` (+ `.module.css`, `SelectionBar.test.tsx`) and export it from `frontend/src/components/index.ts` (update `index.test.ts`); add it to the gallery `frontend/src/features/dev/ComponentGalleryPage.tsx`
- [ ] T024 [P] Write `frontend/src/features/bulk/bulkText.ts` (count by level, summary sentence per verb, singular/plural, delete dialog sentences) and `bulkText.test.ts`
- [ ] T025 [P] Write `frontend/src/features/bulk/useSelection.ts` and `useSelection.test.ts`: toggle, Shift range, select rows, select all matching, kept on page/expand/sort, cleared with the message on scope change and after an action, Select failed items replaces the selection (FR-001 – FR-003, FR-005, FR-021)
- [ ] T026 [P] Write `frontend/src/features/bulk/RowCheckbox.tsx` and `HeaderCheckbox.tsx` with tests: accessible names "Select <level> '<name>'" and "Select all rows on this page", Space, mixed state, header ticks the matching rows of the page only (FR-002, FR-005, D-134)
- [ ] T027 Write `frontend/src/features/bulk/bulkApi.ts`: `previewBulkAction`, `useRunBulkAction` (invalidates `['all-work']` and the plan queries), `useUndoBulkDelete`, `useBulkFields(level)`, `useBulkTargets(query)`

### US1 — Select and change owner or a field (P1)

- [ ] T028 [US1] Write `frontend/src/features/bulk/BulkActionBar.tsx` (+ css) and `MixedLevelDialog.tsx`; tests: count text by level, every action always visible, Select all N matching and Clear, the 1,000 limit message, mixed selection → "works on one level at a time" + Keep only <level> (FR-002, FR-004, FR-006, FR-007)
- [ ] T029 [US1] Write `ChangeOwnerDialog.tsx` and `ChangeFieldDialog.tsx` in `frontend/src/features/bulk/` with tests: person picker, offered fields and the not-offered note, value controls, Clear the value, preview line, re-approval list with Leave those unchanged / Change and restart approval sent as `restartApproval`, skipped reasons, run and summary toast, selection cleared (FR-008 – FR-011)
- [ ] T030 [US1] Write `frontend/src/features/bulk/BulkResultsPanel.tsx` and its test: "<done> <verb> · <s> skipped · <f> failed", Skipped and Failed lists with ID, linked name and reasons, new IDs for copies, Select failed items (FR-021)
- [ ] T031 [US1] Add checkboxes to `frontend/src/features/all-work/GridRow.tsx` and `AllWorkGrid.tsx` (campaign, initiative and activity rows; header checkbox per page) and the `BulkActionBar` to `AllWorkPage.tsx` on the Grid tab, with the scope key year + filters + search; extend `AllWorkGrid.test.tsx` and `AllWorkPage.test.tsx`: no checkbox on plans, line items and unassigned rows; ticking a campaign selects only it; selection kept across pages and cleared when a filter changes (FR-001 – FR-003)
- [ ] T032 [US1] Add the same checkboxes and bar to the plan tree of features 004–006 (`frontend/src/features/plans/PlanTree.tsx` and the plan page; scope key = plan + tree search) and extend their tests (FR-001)

### US2 — Import (P1)

- [ ] T033 [P] [US2] Write `frontend/src/features/imports/readFile.ts` (file → base64) and `problemsCsv.ts` (problem list → CSV with BOM) with `readFile.test.ts`, `problemsCsv.test.ts`
- [ ] T034 [US2] Write `frontend/src/features/imports/importApi.ts`: `usePlanImports(planId)`, `downloadTemplate(planId)` (010's `downloadFromApi`), `checkImport`, `useImport`, `useImportDetail(importId)`, `useUndoImport`
- [ ] T035 [US2] Write `ImportPage.tsx` and `ImportCheckResult.tsx` (+ css) in `frontend/src/features/imports/` and the route `/plans/:planId/import` in `frontend/src/app/router.tsx` (`handle: { title: 'Import activities' }`); `ImportPage.test.tsx`: not-ready empty state with Add a campaign (FR-024), Download template (FR-025), upload + Create missing + Check file, counts, ignored columns, will-create list, needs-more-fields note, problems table, Download problem list, Import N activities sends the ready rows, Upload a corrected file replaces the check, file problems under the input (FR-026 – FR-031)
- [ ] T036 [US2] Write `frontend/src/features/imports/ImportResultsPage.tsx` (+ css) and the route `/plans/:planId/imports/:importId` (`handle: { title: 'Import results' }`); `ImportResultsPage.test.tsx`: cards, imported activities with links and Show all, left-out and failed rows, Undo import confirmation texts with changed/added/already-deleted counts, undone state, already-undone 409 toast (FR-032 – FR-034)
- [ ] T037 [US2] Write `frontend/src/features/imports/PlanImportsCard.tsx` and render it on 004's plan page with **⋯ More › Import activities**; `PlanImportsCard.test.tsx`: list with status chips, View, Undo import until undone, empty state with Import activities (FR-024, FR-033)
- [ ] T038 [US2] Write `frontend/src/features/imports/ChoosePlanDialog.tsx` and the **Import activities** button in 010's All work toolbar; `ChoosePlanDialog.test.tsx`: the year's plans, not-ready plans listed with their reason and not selectable, Continue navigates (FR-024)

### US3 — Submit many (P2)

- [ ] T039 [US3] Write `frontend/src/features/bulk/SubmitDialog.tsx` and its test, and open it from the bar: "<n> will be submitted · <m> are not ready", every reason with a link, optional comment, Submit <n> activities, nothing-ready case (FR-012, FR-013)

### US4 — Copy, move, delete (P2)

- [ ] T040 [US4] Write `TargetPicker.tsx` and `CopyMoveDialog.tsx` in `frontend/src/features/bulk/` with tests: search, paths and dates, no-target notes (D-133), preview counts, effects, skipped list, Copy/Move <n> <records> (FR-014 – FR-018)
- [ ] T041 [US4] Write `frontend/src/features/bulk/BulkDeleteDialog.tsx` and its test, and the delete toast with **Undo** (10 s, `useUndoBulkDelete`): counts sentence, "With them go …", submitted/approved line, Cancel focused, red Delete <n> records, Undo restores and refused Undo shows the API message (FR-019, FR-020)

**Checkpoint**: `npm run check` green.

---

## Phase 4: End-to-end

- [ ] T042 Write `frontend/e2e/bulk.spec.ts` (plan *End-to-end*: copy two activities, change the copies' owner and see the history note, delete + Undo, delete again; header checkbox; mixed selection with Keep only activities; axe on the bar and dialogs) — covers US1 and US4
- [ ] T043 Write `frontend/e2e/import.spec.ts` (plan *End-to-end*: template download, a CSV written by the test with two wrong rows, check, import 8, see them Created in the plan tree, Undo import, Imports card shows Undone; axe on the steps and results) — covers US2

---

## Phase 5: Polish

- [ ] T044 [P] Update `backend/README.md` (packages `bulk`, `imports`; `history.WithNote`, `web.ReadJSONUpTo`, `sheet` reader) and `frontend/README.md` (`features/bulk`, `features/imports`, `SelectionBar`); set 012 to Done in `docs/sdd/feature-map.md`; add the proposed decisions to `docs/sdd/decision-log.md` as numbered by the consolidation step
- [ ] T045 Run `gofmt -l .`, `go vet ./...`, `go test ./...` (and without `-short` once for T012 and T018) in `backend/`, and `npm run check`, `npm run test:e2e` in `frontend/`; fix what fails
- [ ] T046 Demonstrate every spec scenario on the seed data (Priya's 20-activity owner change in at most 6 steps, Mei's import with problems) and check the screens against `docs/product/design-system.md`

---

## Dependencies & Execution Order

- T001 → everything; T002 any time.
- **Backend**: T003, T004, T005, T007 [P] and T006 → T008 → T009 → T010 → T011 → T012; T013, T014 [P] → T015 → T016 → T017 → T018; T019 after T011; T020 → T021 → T022 after T011.
- **Frontend**: T023 – T026 [P] → T027 → T028 → T029 → T030 → T031 → T032; T033 [P] → T034 → T035 → T036 → T037 → T038; T039 after T028; T040, T041 after T028.
- Backend and frontend never wait for each other (frontend tests use `mockApi` with contract-shaped answers).
- End-to-end after both phases; Polish last.

## FR coverage (every FR has a test task)

| FR | Tasks |
|---|---|
| FR-001 | T025, T031, T032, T042 |
| FR-002 | T025, T026, T028, T031 |
| FR-003 | T025, T031 |
| FR-004 | T028 |
| FR-005 | T025, T026 |
| FR-006 | T008, T028 |
| FR-007 | T008, T028, T042 |
| FR-008 | T009, T010, T029, T042 |
| FR-009 – FR-011 | T009, T011, T029 |
| FR-012, FR-013 | T019, T039 |
| FR-014, FR-015, FR-017, FR-018 | T020, T040, T042 |
| FR-016 | T021, T040 |
| FR-019, FR-020 | T022, T041, T042 |
| FR-021 | T008, T030 |
| FR-022 | T008 |
| FR-023 | T003, T008, T011, T016, T042 |
| FR-024 | T015, T035, T037, T038 |
| FR-025 | T013, T015, T017, T035, T043 |
| FR-026 | T005, T013, T014, T018, T035 |
| FR-027 – FR-031 | T016, T018, T035, T043 |
| FR-032 – FR-034 | T016, T018, T036, T037, T043 |
| SC-005 | T016 |
| SC-007 | T012, T018 |

## Implementation Strategy

### MVP first (User Stories 1 and 2)

1. Setup; backend foundation, US1 and US2 (T003 – T018); frontend foundation, US1 and US2 (T023 – T038).
2. Stop and validate: in All work, select 20 seeded activities, change their owner and a field, and see
   "20 changed" and the note in each activity's history (US1 independent test); download the template of
   a seeded plan, upload it with 2 wrong rows, import the 8 ready rows and undo the import (US2
   independent test).

### Incremental delivery

1. + US3 (submit many) → + US4 (copy, move, delete with Undo).
2. After each story: its tests and its part of `bulk.spec.ts` or `import.spec.ts`; commit with the
   feature number and FRs, e.g. `feat(012): bulk submit for approval (FR-012, FR-013)`.

### Notes

- Never edit `frontend/src/api/schema.d.ts` by hand: change `backend/api/openapi.yaml`, then
  `npm run api:types`.
- Keep every user-visible text exactly as in the plan's *User-visible messages* table; tests quote it.
- `bulk` and `imports` never write business records themselves: every create, change, copy, move,
  submit and delete goes through the owning package's service (plan, D-126), so the single-edit
  rules and activity-log entries are reused.
- Previews must never save: `TestPreview_EqualsRunAndSavesNothing_D_n` and `TestCheck_CreatesNothing_FR031`
  guard this; keep them green.
- Uploaded files are never written to disk and their content is never logged.
- Mark each task `[X]` here when it is done.
