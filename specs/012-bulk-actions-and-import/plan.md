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
page's ⋯ menu.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9 strict, React 19
**Primary Dependencies**: none new (Go `archive/zip`, `encoding/xml`, `encoding/csv`, `encoding/base64`)
**Storage**: PostgreSQL; migration `00023_bulk_and_imports.sql`: `bulk_actions`, `bulk_action_bin_entries`,
`plan_imports`, `plan_import_records`, `plan_import_problems`; counters `BLK`, `IMP`
**Testing**: Go `testing` with fakes (ports and `bulktest`/`workbooktest` stores), `apitest` contract tests on
the demo data, PostgreSQL scope-isolation tests of the two scoped reads, fuzz tests of the readers and the row
parser; Vitest + Testing Library + MSW + axe; Playwright + axe for US1, US2 and US3
**Target Platform / Project Type**: web application (React SPA + Go REST API), `127.0.0.1` only
**Performance Goals**: SC-007 — 500 records or a 1,000-row file within 10 s (one `Write` per run)
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
├── internal/bulk/                         # NEW: model, ports, store(+postgres), service, run, request, texts, handler, bulktest
├── internal/workbook/                     # NEW: model, ports, layout, cells, attributes, read, rows, apply, build,
│                                          #      service, undo, store(+postgres), texts, handler, workbooktest
├── internal/access/routes_workbook.go     # NEW: GET /plans/{planId}/workbook
├── internal/stores/…                      # + Bulk, Workbook stores
└── internal/app/                          # bulk_*.go, workbook_*.go adapters; areas.go wiring; transport.go limits

frontend/src/
├── features/bulk/                         # NEW: useSelection, SelectionBar, BulkBar, dialogs, results, api, texts
├── features/all-work/                     # AllWorkGrid + GridRow: checkbox column; AllWorkPage: selection + bar
├── features/workbook/                     # NEW: ImportPage, CheckResult, ImportResultPage, api, readFile, texts
├── features/plans/PlanPage.tsx            # ⋯ menu: Import from Excel, Export to Excel
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

**Port** (`bulk/ports.go`, implemented in `app/bulk_*.go`): `Records` — `Find`, `Below`, `Fields`, `Targets`,
`Effects`; `Changes` — `SetOwner`, `SetField`, `Submit`, `Copy`, `Move`, `Delete`. The app adapters call the
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
- **Apply** (`apply.go`, `rows.go`): sheets in order Campaigns → Initiatives → Activities → Line items; each row:
  parse cells (problems per cell), resolve its parent (ID or name in the plan, or a record an earlier row of the
  file creates; a parent row with problems → left out, FR-033), then create (no ID) or update (ID of this plan,
  same parent; values compared with the stored ones — equal → unchanged) through the port; port errors map to
  columns by field name. Duplicate names under one parent in the file are problems on both rows.
- **Service**: `Template` / `Export` (workbook built by `build.go`); `Check` = apply in a discarded `Write`;
  `Import` = apply with commit, note "Imported from '<file>' (IMP-…)", stores the import with its records and
  problems; `List`, `Get` (with the undo preview); `Undo` (one `Write`, note "Undo of import (IMP-…)": delete the
  top-level created records still there through the port; refusals kept and listed; records gone counted).

**Port** (`workbook/ports.go`, implemented in `app/workbook_*.go`): `Planning` — `Plan` (facts, the plan's
records with their values, the choices: people, types the model allows, workflows, currencies, attributes and
options per level), `Create`, `Update`, `Delete`, `Exists`.

### Messages

All user-visible texts are in `bulk/texts.go` and `workbook/texts.go` (the spec's sentences; plan of 2026-10-19
"User-visible messages" kept where the behaviour is unchanged).

## Frontend design

- **Selection** (`useSelection`): `Map<id, {id, level, name, planId}>`, `toggle(row, shift)` (Shift selects the
  visible range since the last toggle), `selectRows`, `clear`; cleared with a toast when the year, search or
  filters change (the All work state key) and after an action.
- **Grid**: a first column of checkboxes on campaign, initiative and activity rows ("Select activity '<name>'"),
  a header checkbox ("Select all rows on this page") with mixed state; the Calendar and Budget views keep the
  selection but show no checkboxes.
- **Bar** (`BulkBar`): sticky under the grid, `role="region"` "Selection"; "<n> selected · <levels>", "Select all
  <n> matching" (from the result's rows), the actions and **Clear**; over 1,000 the limit message.
- **Dialogs**: `ChangeOwnerDialog`, `ChangeFieldDialog`, `SubmitDialog`, `CopyMoveDialog` (+ target list),
  `BulkDeleteDialog`, `MixedLevelDialog`; each previews first, then runs; `BulkResultsPanel` (SidePanel) with
  **Select failed items**; a delete toast with **Undo**.
- **Import page** `/plans/:planId/import`: steps *1 Get the template* (**Download template**, **Export to
  Excel**), *2 Upload* (file input, **Check file**), *3 Check and import* (counts per level, ignored columns,
  problems table with **Download problem list**, **Import**, **Upload a corrected file**); "Earlier imports" list.
- **Import results** `/imports/:importId`: counts, records created/updated (links), rows left out, **Undo
  import** with the confirmation of FR-036.

### States

| View | Loading | Empty | Error | Forbidden | Success |
|---|---|---|---|---|---|
| Bulk dialogs | Skeleton lines, confirm busy | "Nothing can be <verb> — …" | API sentence + Try again | the API's refusal | preview, then summary |
| Import page | Skeleton | no earlier imports: "No imports yet" | ErrorState | NoAccess (403) | steps |
| Import results | Skeleton | — | ErrorState / 404 "This import no longer exists." | NoAccess | counts and lists |

## Rules and their tests (Go)

| Rule | Test |
|---|---|
| FR-006, FR-007 request checks | `bulk` `TestRun_RequestChecks_FR006_FR007` |
| FR-008 owner, line items untouched | `TestChangeOwner_FR008` |
| FR-009 fields per level | `TestFields_FR009` (app adapter, apitest) |
| FR-010, FR-011 single-edit rules, re-approval | `TestChangeField_FR010_FR011` |
| FR-012, FR-013 submit | `TestSubmit_FR012_FR013` |
| FR-014–FR-017 copy, move, targets | `TestCopyMove_FR014_FR017`, `TestTargets_FR016` |
| FR-019, FR-020 delete summary, undo | `TestDelete_FR019`, `TestUndo_FR020` |
| FR-021, FR-022 outcomes, independence, preview saves nothing | `TestRun_Outcomes_FR021_FR022`, `TestPreview_SavesNothing_D126` |
| FR-023 note on every entry | `history` `TestWithNote_FR023`; contract tests read the log |
| FR-024 permissions per record | `TestRun_ForbiddenSkipped_FR024` (apitest with a view-only role) |
| FR-025–FR-027 template, export, round trip | `workbook` `TestTemplate_FR025_FR026`, `TestExportThenCheck_NothingToChange_FR027` |
| FR-028 reading | `sheet` reader tests + `FuzzReadXLSX`, `FuzzReadCSV`; `workbook` `TestRead_FR028`, `FuzzCheck` |
| FR-029–FR-033 rows | `TestApply_CreateUpdate_FR029`, `TestApply_Parents_FR030`, `TestApply_Problems_FR031`, `TestApply_Defaults_FR032`, `TestApply_LeftOutBelow_FR033` |
| FR-034–FR-036 import, list, undo | `TestImport_FR034`, `TestUndo_FR036` |
| Scope isolation | `bulk` `TestPostgresStore_GetAction_OnlyTheActor`, `workbook` `TestPostgresStore_Imports_Scope` |

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
