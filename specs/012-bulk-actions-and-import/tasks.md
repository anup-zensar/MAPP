---

description: "Task list for feature 012 Bulk Actions and Excel Import of Plans"
---

# Tasks: Bulk Actions and Excel Import of Plans

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md),
[contracts/routes.md](contracts/routes.md), [quickstart.md](quickstart.md). Re-written 2026-10-28; the task list
of 2026-10-19 (JSON store) is replaced.

**Tests**: required (constitution VI); each test first, seen failing. Names carry the FR:
`TestChangeOwner_FR008`, `describe('012 FR-004: …')`.

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

- [x] T005 [P] `platform/sheet` readers test first: `read_test.go` (`TestReadXLSX_SharedInlineNumbersDates`,
  `TestReadXLSX_RoundTripWithWriter`, `TestReadXLSX_Limits`, `TestReadCSV_BOMDelimiterQuotes`), `FuzzReadXLSX`,
  `FuzzReadCSV`; then `read.go`, `xlsx_read.go`, `csv_read.go`.
- [x] T006 [P] `history` `TestWithNote_FR023` first; then `history/note.go` and the note in `appendEntry`
  (`log.go` brought within the comment budget).
- [x] T007 Stores: `bulk/store_postgres.go`, `workbook/store_postgres.go`, fakes `bulktest`, `workbooktest`,
  `stores` set; scope-isolation tests `TestPostgresStore_GetAction_OnlyTheActor`,
  `TestPostgresStore_Imports_Scope`.

---

## Phase 3: User Story 1 — Select many rows and change their owner or a field (P1) 🎯 MVP

- [x] T008 [US1] `bulk` service tests on a fake port first (`service_test.go`, `actions_test.go`):
  `TestRun_RequestChecks_FR006_FR007`, `TestChangeOwner_FR008`, `TestChangeField_FR010_FR011`,
  `TestRun_Outcomes_FR021_FR022`, `TestPreview_SavesNothing_D126`, `TestRun_NoteOnTheContext_FR023`.
- [x] T009 [US1] `bulk` model, ports, request checks, run, texts, service, handler.
- [x] T010 [US1] App adapters `app/bulk_records.go`, `bulk_changes.go`, `bulk_fields.go`, `bulk_targets.go`;
  wiring; contract tests `bulk/handler_test.go` (`TestFields_Contract_FR009`, `TestPreview_Contract`,
  `TestRun_ChangeOwner_Contract_FR008_FR023`, `TestRun_ChangeField_Contract_FR010_FR011`,
  `TestRun_ForbiddenSkipped_FR024`); remove 012's bulk operations from `notYetServed`.
- [x] T011 [P] [US1] Web: `features/bulk` `useSelection` (+ test), `BulkBar` (+ test), `ChangeOwnerDialog`,
  `ChangeFieldDialog`, `BulkResultsPanel`, `bulk-text.ts` (+ tests); All work grid checkbox column and header
  checkbox (+ tests); `AllWorkPage` selection and bar.
- [x] T012 [US1] e2e `frontend/e2e/bulk.spec.ts`: select two activities, change owner, results, history note;
  axe on the bar and the dialog.

## Phase 4: User Story 2 — Build a plan's skeleton from a spreadsheet (P1)

- [x] T013 [US2] `workbook` tests on a fake port first: `layout_test.go` (`TestTemplate_FR025_FR026`),
  `cells_test.go` (formats R4), `read_test.go` (`TestRead_FR028`), `apply_test.go` (`TestApply_CreateUpdate_FR029`,
  `TestApply_Parents_FR030`, `TestApply_Problems_FR031`, `TestApply_Defaults_FR032`, `TestApply_LeftOutBelow_FR033`),
  `service_test.go` (`TestImport_FR034`, `TestUndo_FR036`), `FuzzCheck`.
- [x] T014 [US2] `workbook` model, ports, layout, cells, attributes, read, rows, apply, build, service, undo,
  texts, handler.
- [x] T015 [US2] App adapter `app/workbook_planning.go` (+ `workbook_values.go`, `workbook_edits.go`); wiring; body
  limits; contract tests (`TestTemplate_Contract`, `TestCheck_Contract`, `TestImport_Contract_FR034`,
  `TestListAndGet_Contract`, `TestUndoImport_Contract_FR036`); remove the import operations from
  `notYetServed`.
- [x] T016 [P] [US2] Web: `features/workbook` `ImportPage` (+ test: steps, check result, problems, import, states),
  `ImportResultPage` (+ test: counts, undo confirmation, undone state), `readFile.ts`, `problemsCsv.ts`
  (+ tests); routes; plan page ⋯ items.
- [x] T017 [US2] e2e `frontend/e2e/workbook.spec.ts` — skeleton: upload a CSV with new activities (two rows
  wrong), import the ready rows, find them, undo; axe on the steps and the results page.

## Phase 5: User Story 3 — Update a plan through its spreadsheet (P1)

- [x] T018 [US3] `workbook` `TestExportThenCheck_NothingToChange_FR027`, `TestApply_UpdateApprovedNeedsScreen_FR032`;
  contract test `TestExport_Contract_FR027`.
- [x] T019 [US3] e2e in `workbook.spec.ts` — round trip: export a plan, upload it unchanged ("Nothing to
  change").

## Phase 6: User Story 4 — Submit many activities (P2)

- [x] T020 [US4] `bulk` `TestSubmit_FR012_FR013`; adapter `Submit`; contract `TestRun_Submit_Contract_FR012`.
- [x] T021 [P] [US4] Web `SubmitDialog` (+ test).

## Phase 7: User Story 5 — Copy, move or delete many records (P2)

- [x] T022 [US5] `bulk` `TestCopyMove_FR014_FR017`, `TestDelete_FR019`, `TestUndo_FR020`; adapters `Copy`, `Move`,
  `Delete`, `Targets`, `Effects`; contract tests `TestTargets_Contract_FR016`, `TestRun_CopyMoveDelete_Contract`,
  `TestUndoBulkDelete_Contract_FR020`.
- [x] T023 [P] [US5] Web `CopyMoveDialog`, `BulkDeleteDialog`, `MixedLevelDialog` (+ tests), Undo toast.
- [x] T024 [US5] e2e in `bulk.spec.ts`: delete the selection and Undo.

## Phase 8: Polish

- [x] T025 Gates: `golangci-lint fmt` and `run`, `sqlc generate && vet`, `go vet`, `go test -race` with
  `TEST_DATABASE_URL`, `govulncheck`, `pnpm run check`, `pnpm run build`, full e2e.
- [x] T026 Comment budget and 300-line limit on every touched file; feature map row "Built"; decision log
  D-338–D-341, Q-020, Q-021; glossary *Plan workbook*.
