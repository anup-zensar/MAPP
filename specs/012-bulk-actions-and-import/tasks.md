---

description: "Task list for feature 012 Bulk Actions and Excel Import of Plans"
---

# Tasks: Bulk Actions and Excel Import of Plans

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/routes.md](contracts/routes.md), [quickstart.md](quickstart.md). Re-written 2026-10-28; the task list
of 2026-10-19 (JSON store) is replaced.

**Tests**: required (constitution VI); each test first, seen failing. Names carry the feature and the FR:
`TestChangeOwner_ChangedOrAlreadySet_012FR008_FR021`, `describe('012 FR-001 – FR-005: …')`.

**Environment**: API tests on fakes; scope-isolation tests on `TEST_DATABASE_URL`; e2e on ports 18202/14202,
database `mapp_e2e_b`.

## Format: `[ID] [P?] [Story?] Description`

---

## Phase 1: Setup

- [x] T001 Re-shape the import schemas in `backend/api/openapi.yaml` (`ImportUpload`, `ImportProblem`,
  `ImportLevelCounts`, `ImportCheck`, `ImportSummary`, `ImportRecordRef`, `ImportUndoPreview`, `ImportKept`,
  `ImportDetail`, `PlanImportPage`; `ImportRun`, `ImportNewInitiative`, `ImportWillCreate` removed) and add
  `GET /api/v1/plans/{planId}/workbook`; `pnpm run api:types`.
- [x] T002 [P] Add `PrefixBulkAction = "BLK"` and `PrefixImport = "IMP"` to `backend/internal/platform/ids/ids.go`.
- [x] T003 [P] Migration `00023_bulk_and_imports.sql` (data-model §1), queries `bulk.sql` and `workbook.sql`;
  `sqlc generate && sqlc vet`.
- [x] T004 [P] `access/routes_workbook.go` with the export rule, registered in `access/routes.go`.

## Phase 2: Foundation

- [x] T005 [P] `platform/sheet` readers test first: `read_test.go` (`TestReadXLSX_SharedInlineNumbersDates_012FR028`,
  `TestReadXLSX_RoundTripWithWriter_012FR027`, `TestReadXLSX_Refusals_012FR028`, `TestReadCSV_BOMDelimiterQuotes_012FR028`),
  `FuzzReadXLSX`, `FuzzReadCSV`; then `read.go`, `xlsx_read.go`, `csv_read.go`.
- [x] T006 [P] `history` `note_test.go` `TestWithNote_EndsTheDetailsOfEveryEntry_012FR023` first; then
  `history/note.go` and the note in `appendEntry` (`log.go` brought within the comment budget).
- [x] T007 Stores: `bulk/store_postgres.go`, `workbook/store_postgres.go`, fakes `bulktest`, `workbooktest`,
  `stores` set; scope-isolation tests `TestPostgresStore_GetAction_OnlyTheActor_012FR020`,
  `TestPostgresStore_ImportsInScope_012FR035`.

---

## Phase 3: User Story 1 — Select many rows and change their owner or a field (P1) 🎯 MVP

- [x] T008 [US1] `bulk` service tests on a fake port first (`service_test.go`, fakes in `fixture_test.go`):
  `TestRun_RequestChecks_012FR006_FR007`, `TestChangeOwner_ChangedOrAlreadySet_012FR008_FR021`,
  `TestChangeField_ReapprovalAndRefusals_012FR010_FR011`, `TestRun_GoneFailedOthersKept_UnexpectedErrorSavesNothing_012FR022`,
  `TestPreview_SavesNothing_012D126`, `TestRun_EveryLogEntryCarriesTheNote_012FR023`.
- [x] T009 [US1] `bulk` model, ports, request checks (`request.go`), run, summary, texts, service, handler.
- [x] T010 [US1] App adapters `app/bulk.go`, `bulk_records.go`, `bulk_changes.go`, `bulk_fields.go`; wiring; contract
  tests `bulk/handler_test.go` (`TestFields_Contract_012FR009`, `TestRun_ChangeOwner_Contract_012FR008_FR023`,
  `TestRun_ChangeField_Contract_012FR010_FR011`, `TestRun_ForbiddenSkippedUnseenFailed_Contract_012FR024`); remove
  012's bulk operations from `notYetServed`.
- [x] T011 [P] [US1] Web: `features/bulk` `useSelection`, `BulkBar`, `BulkDialogs`, `BulkDialog`, `ChangeOwnerDialog`,
  `ChangeFieldDialog` (+ `FieldValueInput`; test `ChangeFieldDialog.test.tsx`), `ResultsPanel`, `OutcomeList`,
  `bulk-text.ts`; All work grid checkbox column and header checkbox (`all-work/SelectBox.tsx`, `grid-selection.ts`;
  test `AllWorkGrid.selection.test.tsx`, which also proves `useSelection`); `AllWorkPage` selection and bar.
- [x] T012 [US1] e2e `frontend/e2e/bulk.spec.ts`: select two activities, change owner, results, history note;
  axe on the bar and the dialog.

## Phase 4: User Story 2 — Build a plan's skeleton from a spreadsheet (P1)

- [x] T013 [US2] `workbook` tests on a fake port first (fakes in `fixture_test.go`): `cells_test.go`
  (`TestParse_CellFormats_012D138`, `TestParseAttribute_ChoicesAndSplits_012R10`), `read_test.go`
  (`TestRead_HeadingsAnyOrderAndCase_IgnoredColumns_MarkingRow_012FR028`,
  `TestRead_FileRefusals_EveryMissingColumnAtOnce_012FR028`, `FuzzCheck`), `apply_test.go`
  (`TestApply_CreateUpdateUnchanged_012FR029_FR030`, `TestApply_ParentsIDsAndLeftOutBelow_012FR029_FR033`,
  `TestApply_RefusalsUnderTheirColumns_ApprovedOnScreen_012FR031_FR032`, `TestApply_UnexpectedErrorStopsTheRun_012FR022`).
  The layout, import and undo are proven by the contract tests of T015.
- [x] T014 [US2] `workbook` model, ports, layout, cells, attributes, read, resolve, apply, build, service, undo,
  texts, handler.
- [x] T015 [US2] App adapters `app/workbook.go`, `workbook_planning.go`, `workbook_choices.go`, `workbook_create.go`,
  `workbook_update.go`; wiring; body limits; contract tests `workbook/handler_test.go`
  (`TestTemplate_SheetsColumnsMarkings_Contract_012FR025_FR026`, `TestCheck_EveryProblemAtOnce_LeftOutBelow_Contract_012FR031_FR033`,
  `TestCheck_ViewerRefused_Contract_012FR024`, `TestCheck_FileProblems_Contract_012FR028`) and
  `handler_import_test.go` (`TestImport_SkeletonListedAndUndone_Contract_012FR034_FR035_FR036`,
  `TestImport_OtherPeopleSeeNoImportOfAPlanTheyCannotSee_Contract_012FR035`,
  `TestCheck_CSVIsTheActivitiesSheet_Contract_012FR028`); remove the import operations from `notYetServed`.
- [x] T016 [P] [US2] Web: `features/workbook` `ImportPage` with `CheckResult` and `ImportsList` (+ test: steps, check
  result, problems, import, states), `ImportResultPage` (+ test: counts, undo confirmation, undone state;
  `problemsCsv`), `files.ts` (file to base64, `problemsCsv`), `useWorkbookMenu.ts`; routes; plan page ⋯ items.
- [x] T017 [US2] e2e `frontend/e2e/workbook.spec.ts` — skeleton: upload a CSV with new activities (two rows
  wrong), import the ready rows, find them, undo; axe on the steps and the results page.

## Phase 5: User Story 3 — Update a plan through its spreadsheet (P1)

- [x] T018 [US3] `workbook` contract tests `TestExportThenCheck_NothingToChange_Contract_012FR027_SC004` and
  `TestImport_UpdatesByIDApprovedOnScreen_Contract_012FR029_FR032` (the rule on fakes in T013's
  `TestApply_RefusalsUnderTheirColumns_ApprovedOnScreen_012FR031_FR032`).
- [x] T019 [US3] e2e in `workbook.spec.ts` — round trip: export a plan, upload it unchanged ("Nothing to
  change").

## Phase 6: User Story 4 — Submit many activities (P2)

- [x] T020 [US4] `bulk` `TestSubmit_ActivitiesOnlyEveryReasonAtOnce_012FR012_FR013`; adapter `Submit`; contract
  `TestRun_Submit_Contract_012FR012_FR013`.
- [x] T021 [P] [US4] Web `SubmitDialog` (test in `BulkBar.test.tsx`, "012 FR-012, FR-013, FR-021").

## Phase 7: User Story 5 — Copy, move or delete many records (P2)

- [x] T022 [US5] `bulk` `TestDelete_NestedCountedOnceAndSummarised_012FR019_FR020`, `TestUndo_Refusals_012FR020`;
  `targets.go`, `undo.go`; adapters `Copy`, `Move`, `Delete`, `Targets`, `Effects` (`app/bulk_moves.go`,
  `bulk_changes.go`, `bulk_records.go`); contract tests `TestTargets_Contract_012FR016`,
  `TestRun_CopyMoveDeleteUndo_Contract_012FR014_FR020` (copy, move, delete and its undo).
- [x] T023 [P] [US5] Web `CopyMoveDialog`, `DeleteDialog`, `MixedLevelDialog` (tests in `BulkBar.test.tsx`: "012
  FR-007, FR-014, FR-016", "012 FR-019, FR-020"), Undo toast.
- [x] T024 [US5] e2e in `bulk.spec.ts`: delete the selection and Undo.

## Phase 8: Polish

- [x] T025 Gates: `golangci-lint fmt` and `run`, `sqlc generate && vet`, `go vet`, `go test -race` with
  `TEST_DATABASE_URL`, `govulncheck`, `pnpm run check`, `pnpm run build`, full e2e.
- [x] T026 Comment budget and 300-line limit on every touched file; feature map row "Built"; decision log
  D-338–D-341, Q-020, Q-021; glossary *Plan workbook*.

## Phase 9: Analyze findings (2026-10-29)

- [x] T027 [US2] Web: the loading (Skeleton) and no-access (403) states of `ImportPage` and `ImportResultPage`,
  tests first in `ImportPage.test.tsx` and `ImportResultPage.test.tsx`; both pages show the no-access page for a
  403 through `LoadFailed.tsx`. Done: the two no-access tests failed first (the pages showed Try again).
- [x] T028 SC-007 benchmark `backend/internal/platform/apibench/bulk_import_test.go` `BenchmarkBulkAndImport`:
  the demo plan grown to 500 activities by copying its campaigns, preview and run of **Change owner** on them, and
  the check of a 1,000-row activities file; one sub-benchmark per data set (`demo`, `volume`). Done, demo data
  (Apple M5, 2026-10-29): preview 0.52 s / 2,461 statements, run 0.48 s / 2,463 statements (within 10 s); import
  check 50.1 s / 18,034 statements — **SC-007 not met** (T030). Volume half not run yet: `TEST_DATABASE_URL=… go test -run='^$' -bench='BenchmarkBulkAndImport/volume' -benchtime=1x -timeout=6h ./internal/platform/apibench/` (from `backend/`; the first run builds the volume template, which took more than 1.5 hours on 2026-10-29 and was stopped).
- [ ] T030 SC-007: the check of a 1,000-row file takes 50 s on the demo data, five times the 10 s allowed; it sends
  about 18 statements per row and its time grows faster than the rows (250 rows: 7.8 s, 4,534 statements). Find
  where the per-row reads go (the `app/workbook_*` adapters run each row through the owning service's create in the
  discarded `Write`) and read once per check, then rerun T028. Open: found by T028, not fixed here.
  Progress 2026-10-29: `app/workbook_create.go` keeps each parent's activity form and line-item form options for
  the rest of the Write (memo area `workbookForms:`): `NewForm` listed every attribute with its usage (every
  campaign, initiative, activity, line item and purchase request) and the overspend limits for every row. 1,000-row
  check on the demo data: before 50.1 s / 18,034 statements, after 17.5 s / 9,043 statements (9 per row). Still
  open, still over 10 s. Per created activity: 4 writes (PutActivity, AppendLogEntry, two AdvanceCounter) and 5
  reads — the sibling list of the name check (re-read after every write because a write forgets the memo; it
  grows with the file, which is where the time goes), the log's path read of the record just written, and the 3
  reads of the view the adapter discards. Next: a create that returns only the ID (no view), and keeping the
  written row in the memo instead of forgetting the area. Statements cannot become independent of the row count
  while the check is the import discarded (D-126): every record keeps its own insert and log entry, and 037's
  overspend guard refuses each create on its own.
- [x] T029 Docs: plan.md structure, the rules→tests table and these task texts name the real files and tests;
  FR-027 puts **Download template** in step 1 of the Import page; the stale "deferred" comment on `bulkRoutes`
  removed from `access/routes_planning.go` (file brought within the comment budget).
