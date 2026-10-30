# Implementation Plan: Bulk Actions and Excel Import of Plans

**Branch**: `feature/012-bulk-actions-and-import` | **Date**: 2026-10-28 (re-planned on constitution 4.6.0; the
plan of 2026-10-19 targeted the JSON store and is replaced) | **Spec**: [spec.md](spec.md)

**Design artifacts**: [research.md](research.md) · [data-model.md](data-model.md) ·
[contracts/routes.md](contracts/routes.md) · [tasks.md](tasks.md) · [quickstart.md](quickstart.md)

## Summary

- **Bulk actions** — the All work grid gets a checkbox on every campaign, initiative and activity row, a header
  checkbox and a selection bar with **Change owner**, **Change field**, **Submit for approval**, **Copy to…**,
  **Move to…**, **Delete** and **Clear**. Every action asks the API for a **preview** (the very run, discarded:
  counts, re-approvals, effects, the delete summary, every skip reason), then runs it and shows "33 changed ·
  2 skipped · 1 failed" with **Select failed items**; a bulk delete offers **Undo**.
- **Plan workbook** — one .xlsx layout per plan (sheets Campaigns, Initiatives, Activities, Line items, Allowed
  values, Instructions), downloaded empty (**Download template**) or filled (**Export to Excel**). An upload is
  checked row by row with every on-screen rule (every problem at once), then **Import** creates the rows without
  ID and updates the rows with an ID in one step; **Undo import** sends what it created to the recycle bin.

**Go API** — two packages, `internal/bulk` and `internal/workbook` (handler → service → store), each reaching the
planning areas through a consumer-defined port implemented in `internal/app` on the owning services
(`campaign`, `activity`, `lineitem`, `approval`), so every record is changed by the same code, permission check
and activity-log entry as its single edit. `platform/sheet` gains the .xlsx and CSV readers (standard library,
fuzz-tested); `history` gains `WithNote` (the bulk or import note on every log entry of the run).

**Web app** — `src/features/bulk/` (selection, bar, dialogs, results) plugged into All work's grid;
`src/features/workbook/` (the import page with its three steps, the results page) and two items in the plan
page's ⋯ menu (`useWorkbookMenu`).

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict, React 19
**Primary Dependencies**: none new (Go `archive/zip`, `encoding/xml`, `encoding/csv`, `encoding/base64`)
**Storage**: PostgreSQL; migration `00023_bulk_and_imports.sql`: `bulk_actions`, `bulk_action_bin_entries`,
`plan_imports`, `plan_import_records`, `plan_import_problems`; counters `BLK`, `IMP`
**Testing**: Go `testing` with fakes (ports and `bulktest`/`workbooktest` stores), `apitest` contract tests on
the demo data, PostgreSQL scope-isolation tests of the two scoped reads, fuzz tests of the readers and the row
parser; Vitest + Testing Library + MSW + axe; Playwright + axe for US1, US2 and US3
**Target Platform / Project Type**: web application (React SPA + Go REST API), `127.0.0.1` only
**Performance Goals**: SC-007 — 500 records or a 1,000-row file within 10 s (one `Write` per run), measured by `BenchmarkBulkAndImport`
**Constraints**: ≤ 1,000 records per action; ≤ 5 MB and 5,000 rows per file; uploads never written to disk or
logged; request body limit raised only for the two upload operations (7 MB)
**Scale/Scope**: 12 operations (11 in the contract already, 1 new), 2 page routes, ~25 web files

## Constitution Check

| Principle | How 012 meets it | Result |
|---|---|---|
| I Spec first | FR-001–FR-036 and the Clarifications; test names carry the FR | Pass |
| II Shell business | Shell's spec 008 and RFP §8; Shell vocabulary; "Plan workbook", "Import from Excel" | Pass |
| III Documented design | this plan, research, data model, routes; decisions D-338–D-341; no ADR (no dependency) | Pass |
| IV Two parts, server as truth | the API computes every count, skip reason, preview and check; the web app shows them | Pass |
| V Simplicity | preview = the same run discarded (D-126); one port per package; no new dependency; files ≤ 300 lines | Pass |
| VI Meaningful tests | a unit test per rule on fakes, contract test per operation, scope-isolation tests of `GetAction` and `ListImports`/`GetImport`, fuzz of the readers, view-state tests, e2e of US1–US3 | Pass |
| VII Security | route rules (`bulkRoutes`, new `workbookRoutes`); every record through its owning service's permission and scope check; unseen = "No longer exists"; uploads size-limited, parsed in memory, zip bombs bounded; bulk actions readable only by their actor | Pass |
| VIII Clean construction | comment budget and 300 lines checked on every touched file | Pass |
| IX–X Experience, palette | one yellow primary per dialog; delete red; Cancel focused; tokens only | Pass |

### Threat model (STRIDE)

| Threat | Where | Response |
|---|---|---|
| Spoofing | all operations | session actor; grants from the actor only |
| Tampering: changing records out of scope | bulk run, import | each record changes through its owning service, which checks the permission in the record's plan scope; a plan ID in the path is checked as the import's scope |
| Tampering: malicious workbook | upload | ≤ 5 MB decoded, ≤ 50 MB uncompressed per zip part and ≤ 5,000 rows read; XML decoded with the standard decoder (no external entities); cells are text, never formulas evaluated; export cells written as inline strings (a leading `=` is text) |
| Repudiation | every change | one activity-log entry per record with the actor and the run's note (BLK-…, IMP-…) |
| Information disclosure | export, results | export only of a plan the person sees with `data.export`; skip reasons of unseen records read "No longer exists"; uploaded content never logged |
| Denial of service | runs | 1,000 records / 5,000 rows; one `Write` |
| Elevation of privilege | undo | bulk undo only by the actor who ran it (store filter); undo import needs `activities.delete` in the plan and each delete is checked by its service |

## Project Structure

```text
backend/
├── api/openapi.yaml                       # import schemas re-shaped (not served before), + GET /plans/{planId}/workbook
├── internal/platform/sheet/               # + xlsx_read.go, csv_read.go, read.go (+ tests, fuzz)
├── internal/platform/ids/ids.go           # + BLK, IMP
├── internal/history/note.go               # + WithNote; log.go appends the note (comment budget restored)
├── internal/platform/database/migrations/00023_bulk_and_imports.sql · queries/bulk.sql · queries/workbook.sql
├── internal/bulk/                         # NEW: model, ports, request, run, summary, targets, undo, service, texts,
│                                          #      handler, store(+postgres), bulktest
├── internal/workbook/                     # NEW: model, ports, layout, cells, attributes, read, resolve, apply, build,
│                                          #      service, undo, store(+postgres), texts, handler, workbooktest
├── internal/access/routes_workbook.go     # NEW: GET /plans/{planId}/workbook (the other rules: bulkRoutes in routes_planning.go)
├── internal/stores/…                      # + Bulk, Workbook stores
├── internal/app/                          # bulk.go, bulk_records.go, bulk_changes.go, bulk_fields.go, bulk_moves.go;
│                                          # workbook.go, workbook_planning.go, workbook_choices.go, workbook_create.go,
│                                          # workbook_update.go; areas.go wiring; transport.go limits
└── internal/platform/apibench/bulk_import_test.go   # SC-007: bulk change owner of 500 activities, check of 1,000 rows
                                                     # (+ ActivityIDs, CampaignIDs in apibench/records.go)

frontend/src/
├── features/bulk/                         # NEW: useSelection, BulkBar, BulkDialog(s), ChangeOwnerDialog, ChangeFieldDialog,
│                                          #      FieldValueInput, SubmitDialog, CopyMoveDialog, DeleteDialog, MixedLevelDialog,
│                                          #      ResultsPanel, OutcomeList, api, bulk-text, bulk-types
├── features/all-work/                     # SelectBox (+ grid-selection) in AllWorkGrid and GridRow; AllWorkPage: selection + bar
├── features/workbook/                     # NEW: ImportPage, CheckResult, ImportsList, ImportResultPage, LoadFailed,
│                                          #      useWorkbookMenu, api, files, workbook-text
├── features/plans/PlanPage.tsx            # ⋯ menu (useWorkbookMenu): Import from Excel, Export to Excel
└── app/routes.tsx · pages.tsx             # /plans/:planId/import, /imports/:importId
frontend/e2e/bulk.spec.ts · workbook.spec.ts
```

## Backend design

### Bulk (`internal/bulk`)

`Service.Preview(ctx, a, req)` and `Service.Run(ctx, a, req)` call `run(ctx, a, req, commit)`: the request checks
(all problems at once, `validation_failed`), then one `Write` in which every selected ID (duplicates removed;
for copy, move and delete a record under another selected record is dropped and counted once) goes through the
port, and the result is built. A preview ends the `Write` with `errDiscard` (as `financials` does, D-126), so
nothing is saved; a run takes `BLK-…` first, puts `history.WithNote(ctx, note)` on the context, and stores the
action (with the bin entries of a delete).

Errors from the port become outcomes: `*errs.Error` `not_found` → **failed** "No longer exists"; any other
`*errs.Error` (validation, conflict, not ready, forbidden) → **skipped** with its field messages (or message);
anything else ends the run with `internal_error` and saves nothing.

| Action | Per record | Extra result |
|---|---|---|
| changeOwner | campaigns, initiatives, activities: owner ← person; same owner → skipped "Already set — nothing to change" | — |
| changeField | the single edit with one field changed; dates of a Submitted/Approved activity: listed in `reapproval`, skipped "<status> — left unchanged …" unless `restartApproval` | `reapproval` |
| submit | activities only (others skipped "Only activities go through approval"); Approved → "Already approved", Submitted → "Already submitted"; Created/Declined → 008's submit with the version read now | — |
| copy | single copy into `targetId`; campaign and initiative names get " (copy)", " (copy 2)", … when taken (D-131) | `newRecordId` |
| move | single move to `targetId`; already there → skipped "Already in '<target>'" | `effects` (campaigns to another plan) |
| delete | single delete; each top-level record its own bin entry | `deleteSummary`, `recycleBinEntries`, `undoAvailableUntil` |

`Undo(ctx, a, id)`: the stored delete action of the same actor (others → 404), one `Write` running
`history.Undo` for each bin entry; the first refusal refuses all (D-130, message "Undo is no longer available —
restore them from the recycle bin" for `undo_expired`).

`Fields(ctx, a, level)` and `Targets(ctx, a, query)` read through the port (FR-009, FR-016).

**Port** (`bulk/ports.go`, implemented in `app/bulk_records.go`, `bulk_fields.go`, `bulk_changes.go` and
`bulk_moves.go`): `Records` — `Find`, `Below`, `Fields`, `Targets`, `Effects`; `Changes` — `SetOwner`, `SetField`,
`Submit`, `Copy`, `Move`, `Delete`. The app adapters call the
owning services with the actor (their checks, logs and hooks included).

### Workbook (`internal/workbook`)

- **Layout** (`layout.go`): the four level sheets, their base columns (heading, marking, field) and aliases;
  attribute columns from the plan's active attributes per level visible for its business unit (R9).
- **Cells** (`cells.go`, `attributes.go`): one format per kind (D-138): dates `YYYY-MM-DD` (also `12 Mar 2026`
  and date cells), Yes/No (also true/false, y/n, 1/0), amounts `12500.00` (also `12,500.00`), people by e-mail,
  references by name or ID, multiple choices `A; B`, splits `China 60; Hong Kong 40` (no shares → even split,
  R10). Export writes exactly the import format.
- **Read** (`read.go`): base64 → bytes (≤ 5 MB) → `sheet.ReadXLSX` (zip signature) or `sheet.ReadCSV` (the
  Activities sheet) → headings matched (case, spaces, aliases) → rows (empty and marking rows skipped, ≤ 5,000).
- **Apply** (`apply.go`, parents in `resolve.go`): sheets in order Campaigns → Initiatives → Activities → Line items; each row:
  parse cells (problems per cell), resolve its parent (ID or name in the plan, or a record an earlier row of the
  file creates; a parent row with problems → left out, FR-033), then create (no ID) or update (ID of this plan,
  same parent; values compared with the stored ones — equal → unchanged) through the port; port errors map to
  columns by field name. Duplicate names under one parent in the file are problems on both rows.
- **Service**: `Template` / `Export` (workbook built by `build.go`); `Check` = apply in a discarded `Write`;
  `Import` = apply with commit, note "Imported from '<file>' (IMP-…)", stores the import with its records and
  problems; `List`, `Get` (with the undo preview); `Undo` (one `Write`, note "Undo of import (IMP-…)": delete the
  top-level created records still there through the port; refusals kept and listed; records gone counted).

**Port** (`workbook/ports.go`, implemented in `app/workbook_planning.go`, `workbook_choices.go`,
`workbook_create.go` and `workbook_update.go`): `Planning` — `Plan` (facts, the plan's
records with their values, the choices: people, types the model allows, workflows, currencies, attributes and
options per level), `Create`, `Update`, `Delete`, `Exists`.

### Messages

All user-visible texts are in `bulk/texts.go` and `workbook/texts.go` (the spec's sentences; plan of 2026-10-19
"User-visible messages" kept where the behaviour is unchanged).

## Frontend design

- **Selection** (`useSelection`): `Map<id, {id, level, name, planId}>`, `toggle(row, shift)` (Shift selects the
  visible range since the last toggle), `selectRows`, `clear`; cleared with a toast when the year, search or
  filters change (the All work state key) and after an action.
- **Grid** (`all-work/SelectBox.tsx`): a first column of checkboxes on campaign, initiative and activity rows ("Select activity '<name>'"),
  a header checkbox ("Select all rows on this page") with mixed state; the Calendar and Budget views keep the
  selection but show no checkboxes.
- **Bar** (`BulkBar`): sticky under the grid, `role="region"` "Selection"; "<n> selected · <levels>", "Select all
  <n> matching" (from the result's rows), the actions and **Clear**; over 1,000 the limit message.
- **Dialogs** (opened by `BulkDialogs`, framed by `BulkDialog`): `ChangeOwnerDialog`, `ChangeFieldDialog`
  (+ `FieldValueInput`), `SubmitDialog`, `CopyMoveDialog` (+ target list), `DeleteDialog`, `MixedLevelDialog`;
  each previews first, then runs; `ResultsPanel` (SidePanel, outcomes in `OutcomeList`) with **Select failed
  items**; a delete toast with **Undo**.
- **Import page** `/plans/:planId/import`: steps *1 Get the template* (**Download template**, **Export to
  Excel**), *2 Upload* (file input, **Check file**; the file read to base64 in `files.ts`), *3 Check and import*
  (`CheckResult`: counts per level, ignored columns, problems table with **Download problem list**, written by
  `problemsCsv` in `files.ts`; **Import**); "Earlier imports" list (`ImportsList`).
- **Import results** `/imports/:importId`: counts, records created/updated (links), rows left out, **Undo
  import** with the confirmation of FR-036.

### States

| View | Loading | Empty | Error | Forbidden | Success |
|---|---|---|---|---|---|
| Bulk dialogs | Skeleton lines, confirm busy | "Nothing can be <verb> — …" | API sentence + Try again | the API's refusal | preview, then summary |
| Import page | Skeleton | no earlier imports: "No imports yet" | ErrorState (`LoadFailed`) | NoAccess (403, `LoadFailed`) | steps |
| Import results | Skeleton | — | ErrorState / 404 "This import no longer exists." (`LoadFailed`) | NoAccess (403, `LoadFailed`) | counts and lists |

Each state has its test: `ImportPage.test.tsx` and `ImportResultPage.test.tsx` (loading, empty, error, no access,
success); the bulk dialogs' in `BulkBar.test.tsx` and `ChangeFieldDialog.test.tsx`.

## Rules and their tests (Go)

Test names end with the feature and requirement IDs (`_012FR008_FR021`); contract tests run through `apitest`.

| Rule | Test |
|---|---|
| FR-006, FR-007 request checks | `bulk/service_test.go` `TestRun_RequestChecks_012FR006_FR007` |
| FR-008, FR-021 owner, changed or already set | `TestChangeOwner_ChangedOrAlreadySet_012FR008_FR021`; contract `TestRun_ChangeOwner_Contract_012FR008_FR023` |
| FR-009 fields per level | `bulk/handler_test.go` `TestFields_Contract_012FR009` |
| FR-010, FR-011 single-edit rules, re-approval | `TestChangeField_ReapprovalAndRefusals_012FR010_FR011`; contract `TestRun_ChangeField_Contract_012FR010_FR011` |
| FR-012, FR-013 submit | `TestSubmit_ActivitiesOnlyEveryReasonAtOnce_012FR012_FR013`; contract `TestRun_Submit_Contract_012FR012_FR013` |
| FR-014–FR-018 copy, move, targets, effects | contract `TestRun_CopyMoveDeleteUndo_Contract_012FR014_FR020`, `TestTargets_Contract_012FR016` |
| FR-019, FR-020 delete summary, undo | `TestDelete_NestedCountedOnceAndSummarised_012FR019_FR020`, `TestUndo_Refusals_012FR020` |
| FR-022 independence, preview saves nothing (D-126) | `TestRun_GoneFailedOthersKept_UnexpectedErrorSavesNothing_012FR022`, `TestPreview_SavesNothing_012D126`; `workbook` `TestApply_UnexpectedErrorStopsTheRun_012FR022` |
| FR-023 note on every entry | `history/note_test.go` `TestWithNote_EndsTheDetailsOfEveryEntry_012FR023`; `TestRun_EveryLogEntryCarriesTheNote_012FR023` |
| FR-024 permissions per record | `TestRun_ForbiddenSkippedUnseenFailed_Contract_012FR024`; `workbook` `TestCheck_ViewerRefused_Contract_012FR024` |
| FR-025–FR-027 template, export, round trip | `workbook/handler_test.go` `TestTemplate_SheetsColumnsMarkings_Contract_012FR025_FR026`, `TestExportThenCheck_NothingToChange_Contract_012FR027_SC004`; `sheet` `TestReadXLSX_RoundTripWithWriter_012FR027` |
| Cell formats (D-138, R10) | `workbook/cells_test.go` `TestParse_CellFormats_012D138`, `TestParseAttribute_ChoicesAndSplits_012R10` |
| FR-028 reading | `sheet/read_test.go` `TestReadXLSX_SharedInlineNumbersDates_012FR028`, `TestReadXLSX_Refusals_012FR028`, `TestReadCSV_BOMDelimiterQuotes_012FR028`, `FuzzReadXLSX`, `FuzzReadCSV`; `workbook/read_test.go` `TestRead_HeadingsAnyOrderAndCase_IgnoredColumns_MarkingRow_012FR028`, `TestRead_FileRefusals_EveryMissingColumnAtOnce_012FR028`, `FuzzCheck`; contract `TestCheck_FileProblems_Contract_012FR028`, `TestCheck_CSVIsTheActivitiesSheet_Contract_012FR028` |
| FR-029–FR-033 rows | `workbook/apply_test.go` `TestApply_CreateUpdateUnchanged_012FR029_FR030`, `TestApply_ParentsIDsAndLeftOutBelow_012FR029_FR033`, `TestApply_RefusalsUnderTheirColumns_ApprovedOnScreen_012FR031_FR032`; contract `TestCheck_EveryProblemAtOnce_LeftOutBelow_Contract_012FR031_FR033`, `TestImport_UpdatesByIDApprovedOnScreen_Contract_012FR029_FR032` |
| FR-034–FR-036 import, list, undo | `workbook/handler_import_test.go` `TestImport_SkeletonListedAndUndone_Contract_012FR034_FR035_FR036`, `TestImport_OtherPeopleSeeNoImportOfAPlanTheyCannotSee_Contract_012FR035` |
| Scope isolation | `bulk` `TestPostgresStore_GetAction_OnlyTheActor_012FR020`, `workbook` `TestPostgresStore_ImportsInScope_012FR035` |
| SC-007 time and statements | `platform/apibench/bulk_import_test.go` `BenchmarkBulkAndImport` (500 activities of the grown demo plan, a 1,000-row file); on the demo data the bulk run takes 0.5 s, the 1,000-row check 50.1 s / 18,034 statements before T030, 17.5 s / 9,043 after reading each parent's form once — not yet met (tasks T030) |

### Shared files touched

`backend/api/openapi.yaml` (import schemas, the export path), `frontend/src/api/generated/schema.d.ts`
(regenerated), `backend/internal/access/routes.go` (+1 line), `backend/internal/platform/ids/ids.go`,
`backend/internal/stores/{stores,postgres}.go`, `stores/storestest/storestest.go`, `backend/internal/app/areas.go`,
`app/transport.go`, `app/contract_routes_test.go` (notYetServed emptied of 012), `history/log.go`,
`frontend/src/app/routes.tsx`, `pages.tsx`, `features/plans/PlanPage.tsx`, `features/all-work/*`,
`docs/sdd/decision-log.md`, `docs/sdd/feature-map.md`, `docs/product/glossary.md`.

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| Ports implemented in `app` | each package reaches four owning services; the port keeps the rules testable on fakes | calling the services directly couples every rule test to the whole API |
| Preview by a discarded run | the preview shows exactly what the run does | a dry-run path would duplicate every rule |
| Own .xlsx reader | the standard library suffices and 010 already writes .xlsx this way | a spreadsheet library is a new dependency |
