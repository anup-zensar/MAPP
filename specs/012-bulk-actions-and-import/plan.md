# Implementation Plan: Bulk Actions and Import

**Branch**: `012-bulk-actions-and-import` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/012-bulk-actions-and-import/spec.md`

**Design artifacts**: [data-model.md](data-model.md) · [contracts/openapi.yaml](contracts/openapi.yaml) ·
[tasks.md](tasks.md). Conventions are feature 001's; the All work grid, its pager and `platform/sheet`
come from feature 010's [plan](../010-all-work-views/plan.md).

## Summary

Two capabilities on top of the All work grid (010) and the plan tree (004–006):

- **Bulk actions** — tick campaign, initiative and activity rows (checkbox always visible), then **Change
  owner**, **Change field**, **Submit for approval**, **Copy to…**, **Move to…** or **Delete** up to 1,000
  records at once. Every action first shows a **preview** (counts, re-approval list, effects, every reason
  a record will be skipped), then runs and ends with a summary "33 changed · 2 skipped · 1 failed" with
  **Select failed items**. Bulk delete offers **Undo** for all records in one step.
- **Import activities** — for one plan: download a template made for that plan, upload a filled `.xlsx`
  or `.csv`, see every problem of every row at once, import the ready rows as *Created* activities, and
  **Undo import** later.

**Go API** — two new packages: `internal/bulk` and `internal/imports`. Each record is changed through the
owning feature's own service functions (same rules as a single edit, FR-010, FR-027), reached through a
narrow `Planning` interface per package. A bulk action or an import runs in **one store transaction**; its
preview runs the very same code in a transaction that is thrown away, so the preview equals the result
unless someone changes the data in between. Every record the action touches gets its own activity-log
entry carrying a note such as "Bulk change of 20 activities (BLK-1001)" (`history.WithNote`, new).
`platform/sheet` gains an `.xlsx` and CSV reader; `platform/web` gains `ReadJSONUpTo` for the upload.

**Web app** — `src/features/bulk/` (selection, the bulk-action bar, one dialog per action, the results
panel) plugged into the All work grid and the plan tree; `src/features/imports/` (the three-step import
page, the results page, the plan's Imports card).

## Technical Context

**Language/Version**: Go 1.27; TypeScript 5.9 strict + React 19.3

**Primary Dependencies**: none new (Go standard library `archive/zip`, `encoding/xml`, `encoding/csv`,
`encoding/base64`; web app libraries of feature 001)

**Storage**: JSON file store; new tables `bulk_actions` and `imports`; all business records are written by
their owning packages

**Testing**: Go `testing` + `apitest`; Vitest + Testing Library; Playwright + axe

**Target Platform / Project Type**: as feature 001 — web application (React SPA + Go REST API)

**Performance Goals**: SC-007 — a bulk action on 500 records, or the check of a 1,000-row file, answers in
under 10 seconds (one store commit per action; tested by `TestBulk_500Records_SC007` and
`TestCheck_1000Rows_SC007`)

**Constraints**: `127.0.0.1` only; uploaded files are never written to disk and their content is never
logged; up to 1,000 records per action (FR-006) and 1,000 rows / 5 MB per file (FR-026)

**Scale/Scope**: 2 page routes; 11 API operations; 2 stored record types; ~20 web components and helpers

## Constitution Check

| Principle | How this plan complies | Result |
|---|---|---|
| I. Spec first | Every element traces to an FR, scenario or edge case; the spec has no `[NEEDS CLARIFICATION]`; interpretations of ambiguous points are listed as proposed decisions | Pass |
| II. Plain language | "Select", Shell's levels, the verbs changed/submitted/copied/moved/deleted; every text in *User-visible messages* | Pass |
| III. Simple, friendly | Checkboxes always visible, one bar with every action always shown, mixed selections explained with **Keep only <level>**, every consequence shown before confirming, every skipped record with its plain reason and link, all import problems in one check, Undo for deletes and imports | Pass |
| IV. Layered architecture | Two packages (handler → service → repository); rules stay in their owning packages (bulk and import call them through `Planning`); contract-first; generated web types | Pass |
| V. Tested rules | A Go test per FR rule, contract tests per operation, component tests per dialog and page, end-to-end tests for US1, US2 and US4 (bulk change, import, delete + Undo), performance tests for SC-007 | Pass |
| VI. Safe by default | Every change logged per record with the actor and the action's note; deletes go to the recycle bin; previews never save; uploaded content not logged nor stored; the request limit is raised only for the upload operation | Pass |
| VII. Small and readable | No dependency; the `Planning` interfaces are the only abstraction (justified below); preview = same code + rollback instead of a second "dry-run" implementation | Pass |

**Post-design re-check**: Pass.

## Decisions

The spec has no `[NEEDS CLARIFICATION]` marker. Proposed decisions (numbered at consolidation, 2026-10-19):

| # | Proposed decision | Reason |
|---|---|---|
| D-126 | A bulk action or an import is **one store transaction**. Inside it each record is checked and changed through its owning service (joining the transaction), so every record is either fully changed, skipped (a rule refused it) or failed (it no longer exists); a storage failure saves nothing and answers `internal_error`. The **preview** runs the same code and discards the transaction | FR-022 with the JSON store (one commit per record would rewrite the tables up to 1,000 times — SC-007); preview equals result |
| D-127 | Records are processed **against their latest values**; no version is sent (edge case). A record already holding the value is reported as skipped "Already set — nothing to change" | Edge case; an unchanged record is not a change |
| D-128 | Bulk actions and imports are stored (`bulk_actions` with IDs `BLK-1001`…, `imports` with `IMP-1001`…). Both are new readable-ID prefixes | Undo needs the recycle-bin entries; each plan lists its imports (FR-033) |
| D-129 | Every activity-log entry written during a bulk action or import carries a note in `details`, set once through `history.WithNote(ctx, note)` (new in `history`): "Bulk change of 20 activities (BLK-1001)", "Bulk submit of 15 activities (BLK-1002)", "Bulk copy …", "Bulk move …", "Bulk delete …", "Imported from 'plan.xlsx' (IMP-1001)", "Undo of bulk delete (BLK-1003)", "Undo of import (IMP-1001)" | FR-023 without changing the owning packages' log calls |
| D-130 | **Undo of a bulk delete** restores every recycle-bin entry of the action in one step, all or nothing, within the 2-minute Undo window of D-037; a refusal (too late, a name taken meanwhile) changes nothing and points to the recycle bin | FR-020, D-037 |
| D-131 | **Copy names**: a copy gets " (copy)", " (copy 2)", … only when its name is taken in the target (R13); a **move** whose name is taken in the target is skipped ("'Events' already has an activity called 'Trade fair'") | Spec scenario 1 (copy) and FR-017 (move); scenario 3's message is read as the move case |
| D-132 | Copying campaigns or initiatives into a plan with another model keeps every activity below (with a type the target model does not allow, flagged as in 004 FR-019); **copying activities** skips those whose type the target model does not allow (FR-018) | 004/005 single-record behaviour; FR-018 |
| D-133 | **Move** needs the selection in one plan (initiatives, activities) or one year (campaigns); otherwise the target picker explains "Move works within one plan — select records of one plan" (or "…of one year") and offers no target | FR-015 / D-033 give one valid target set only then |
| D-134 | **Header checkbox** selects the *matching* campaign, initiative and activity rows on the current page; context rows ("Shown for context") are left out and can still be ticked one by one | Prevents changing records the filters did not choose |
| D-135 | **Change field** on a parent attribute (Cluster) clears dependent values that no longer fit (Region) and lists them in the preview's effects; no value is filled in automatically | Single-edit rule (002 FR-025) without guessing in bulk |
| D-136 | **Upload** is JSON `{fileName, content (base64), createMissing}` read with `web.ReadJSONUpTo(w, r, dst, 7 MB)` (new); files up to 5 MB. The check is **stateless**: **Import N activities** sends the file again with the row numbers the user confirmed; a confirmed row that has meanwhile become invalid is reported as *failed*, rows not confirmed as *left out* | Keeps the JSON-only API and contract tests; nothing stored until the user confirms (FR-031) |
| D-137 | **Template**: `.xlsx` with sheets "Activities" (row 1 headings, row 2 markings Required / Needed to submit / Optional, data from row 3), "Allowed values" and "Instructions". Problems name the spreadsheet's own row numbers. A row whose cells are only marking words is skipped, so a CSV saved from the template works | FR-025, FR-027 |
| D-138 | **Import formats**: dates `YYYY-MM-DD`, `12 Mar 2026` or spreadsheet date cells; Yes/No also `true/false`, `y/n`, `1/0`; amounts `12500`, `12,500.00` (more than two decimals rounded half up); multiple choices separated by `;`; splits "China 60; Hong Kong 40" (shares in per cent, no shares → even split, R10) | One documented format per column, shown on the Instructions sheet |
| D-139 | **Undo import** has no time limit and is all or nothing; records of the import that were deleted meanwhile are left alone and counted ("already deleted") | FR-033 "until undone" |

## Project Structure

### Documentation (this feature)

```text
specs/012-bulk-actions-and-import/
├── spec.md · checklists/requirements.md
├── plan.md · data-model.md · contracts/openapi.yaml · tasks.md
```

### Source code

```text
backend/
├── api/openapi.yaml                  # + this feature's fragment
├── internal/app/app.go               # + bulk and imports wiring
└── internal/
    ├── platform/web/web.go           # + ReadJSONUpTo (ReadJSON calls it with 1 MB) (+ test)
    ├── platform/sheet/               # + xlsx_read.go, csv_read.go (+ tests); 010 created the writer
    ├── history/note.go · note_test.go    # + WithNote: details note for every entry logged with the context
    ├── bulk/                         # NEW
    │   ├── model.go                  # Request, Result, Outcome, Effect, DeleteSummary, Field, Target; stored Action
    │   ├── planning.go               # Planning interface + RecordInfo (what bulk needs from 005–008)
    │   ├── planning_adapter.go       # the Planning built on campaigns, activities, approvals, settings, people
    │   ├── repository.go · store_json.go     # table bulk_actions
    │   ├── service.go                # Preview, Run, Undo, Fields, Targets; run(ctx, a, req, commit)
    │   ├── actions.go                # one function per action
    │   ├── messages.go               # reasons, notes, effect texts
    │   ├── fake_test.go              # in-memory Planning for rule tests
    │   ├── service_test.go · actions_test.go · targets_test.go · planning_adapter_test.go
    │   ├── perf_test.go              # SC-007 (skipped with -short)
    │   └── handler.go · handler_test.go
    └── imports/                      # NEW
        ├── model.go                  # Upload, Run, Check, Problem, stored Import, Summary, Detail
        ├── planning.go · planning_adapter.go # what import needs from 002–007
        ├── repository.go · store_json.go     # table imports
        ├── columns.go                # the template's columns for a plan, headings, aliases, markings
        ├── template.go               # template workbook
        ├── parse.go                  # file → table → typed rows (problems per cell)
        ├── service.go                # Template, Check, Import, List, Get, Undo; run(…, commit)
        ├── fake_test.go · columns_test.go · parse_test.go · service_test.go · template_test.go
        ├── planning_adapter_test.go · perf_test.go   # perf: SC-007 (skipped with -short)
        └── handler.go · handler_test.go

frontend/
├── e2e/bulk.spec.ts · import.spec.ts    # NEW
└── src/
    ├── app/router.tsx                    # + /plans/:planId/import, /plans/:planId/imports/:importId
    ├── components/SelectionBar/          # NEW shared component (+ test, export in index.ts)
    ├── features/all-work/AllWorkGrid.tsx · GridRow.tsx · AllWorkPage.tsx   # + checkboxes, bar, Import activities
    ├── features/plans/…                  # + checkboxes in the plan tree, bar, Imports card, Import activities (004–006 files)
    ├── features/bulk/                    # NEW
    │   ├── useSelection.ts · useSelection.test.ts
    │   ├── RowCheckbox.tsx · HeaderCheckbox.tsx (+ tests)
    │   ├── BulkActionBar.tsx (+ css, test)
    │   ├── ChangeOwnerDialog.tsx · ChangeFieldDialog.tsx · SubmitDialog.tsx · CopyMoveDialog.tsx
    │   ├── TargetPicker.tsx · BulkDeleteDialog.tsx · MixedLevelDialog.tsx · BulkResultsPanel.tsx
    │   │   (+ css, one test each)
    │   ├── bulkApi.ts · bulkText.ts · bulkText.test.ts
    └── features/imports/                 # NEW
        ├── ImportPage.tsx · ImportCheckResult.tsx · ImportResultsPage.tsx (+ css, tests)
        ├── PlanImportsCard.tsx · ChoosePlanDialog.tsx (+ tests)
        ├── importApi.ts · readFile.ts · problemsCsv.ts (+ tests)
```

**Structure decision**: feature 001's layout; two business packages that write only through the owning
packages, plus the shared helpers named above.

## Backend design

### Shared additions

| Where | Addition |
|---|---|
| `platform/web` | `ReadJSONUpTo(w, r, dst any, maxBytes int64) error` — `ReadJSON` becomes `ReadJSONUpTo(…, 1 MB)`; same rules and messages; a body over the limit → `bad_request` "The file is larger than 5 MB — remove unused rows or columns, or split it." is answered by the import handler (it checks `http.MaxBytesError`) |
| `history` | `WithNote(ctx, note string) context.Context`; `Log` appends the note as its own sentence to `Details` ("with 3 line items. Bulk delete of 7 records (BLK-1003)") for every entry logged with that context; nested notes keep the innermost |
| `platform/sheet` | `ReadXLSX(data []byte) (Table, error)` — first sheet named "Activities" or else the first sheet; shared and inline strings, numbers as written, date cells (built-in formats 14–22, 45–47 and custom formats with d/m/y) as `YYYY-MM-DD`; `ReadCSV(data []byte) (Table, error)` — UTF-8 with or without BOM, delimiter `,` or `;` (the one in the first line), quoted fields. `Table{Rows [][]string}` with the spreadsheet's row numbers kept (`Row{Number int; Cells []string}`) |

### Package `bulk`

**`Planning`** (implemented in `planning_adapter.go`, faked in tests):

```go
type Planning interface {
    Record(ctx context.Context, id string) (RecordInfo, bool, error) // level, name, plan, parent, status, dates, owner, has-set-later-stages…
    Below(ctx context.Context, id string) (Contents, error)            // campaigns, initiatives, activities, line items below; submitted/approved count
    SetOwner(ctx context.Context, a actor.Actor, id, ownerID string) error
    SetField(ctx context.Context, a actor.Actor, id string, change FieldChange, restartApproval bool) (cleared []string, err error)
    SubmitProblems(ctx context.Context, a actor.Actor, activityID string) ([]Problem, error)
    Submit(ctx context.Context, a actor.Actor, activityID, comment string) error   // submit or re-submit (008)
    Copy(ctx context.Context, a actor.Actor, id, targetID string) (newID string, err error)
    Move(ctx context.Context, a actor.Actor, id, targetID string) error
    Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error)
    PlanChangeEffects(ctx context.Context, activityIDs []string, targetPlanID string) (PlanEffects, error)
    Fields(ctx context.Context, level Level) ([]Field, error)
    Targets(ctx context.Context, level Level) ([]Target, error)
    PersonActive(ctx context.Context, personID string) (name string, active bool, err error)
}
```

Errors returned by the owning services (`*errs.Error`, validation or conflict) become **skip** reasons
(their field messages, in order); `not_found` becomes **failed** "No longer exists"; any other error stops
the action (nothing saved, `internal_error`).

**Service**

| Operation | Rules (FR) |
|---|---|
| `Preview(ctx, a, req) (Result, error)` | `run(ctx, a, req, false)`: everything below, then the transaction is discarded; `id` null; nothing logged or stored |
| `Run(ctx, a, req) (Result, error)` | `run(ctx, a, req, true)`: one `tx.Write`; takes `BLK-…` first, sets `history.WithNote`, processes, stores the `Action` record |
| `Undo(ctx, a, bulkActionID) (UndoResult, error)` | Only `delete` actions; one Write: `history.Undo` for each bin entry (note "Undo of bulk delete (BLK-…)"); first refusal → whole Undo refused with that error (`undo_expired` message replaced by "Undo is no longer available — restore them from the recycle bin"); stores the resolution (FR-020, D-130) |
| `Fields(ctx, a, level) (FieldList, error)` | FR-009: start date, end date; activities also Activity type (required, no clear) and Approval workflow; every active yes/no, single-choice and multiple-choice Shell attribute of the level (not text, number or % split), with active options, parent attribute and business-unit condition |
| `Targets(ctx, a, q TargetQuery) (TargetPage, error)` | FR-016: copy → every plan (campaigns), every campaign (initiatives), every initiative (activities) of any plan (D-033); move → FR-015 (D-133); never a selected record or anything below one; search on name, ID and path ignoring case; sorted by path; first 100 with `total` |

**`run`** — request checks first (all problems at once, `validation_failed`):

| Check | Message |
|---|---|
| no record | Select at least one record. |
| more than 1,000 (FR-006) | Bulk actions handle up to 1,000 records at a time — narrow the filters |
| unknown action | Choose one of the listed actions. |
| changeOwner without an active person | Choose a person. · <name> is inactive — choose an active person. |
| changeField without a field / value | Choose a field. · Choose a value, or choose Clear the value. · <field> is required — it can't be cleared. |
| field not offered for the level | <field> can't be changed for <level plural> here. |
| copy/move without a target / wrong level | Choose where to copy them. · Choose where to move them. · Choose a <plan / campaign / initiative>. |
| changeField, copy, move with mixed levels (FR-007) | <Action> works on one level at a time — keep only one level. |
| comment over 2,000 characters | Use 2,000 characters or fewer. |

Then, in the transaction, for each selected ID (duplicates removed; for copy, move and delete a record
under another selected record is dropped and counted once, FR-019):

| Action | Per record (FR) | Skip / fail reasons |
|---|---|---|
| `changeOwner` | campaigns, initiatives, activities → `SetOwner`; line items keep their owners (FR-008) | Already set — nothing to change · owning-service messages |
| `changeField` | `SetField` with the same rules as a single edit (FR-010); dates on Submitted or Approved activities (incl. In progress / Complete): listed in `reapproval`; with `restartApproval` false → skipped, true → changed and sent back to *Created* (FR-011, §2.5); a type change never touches planned cost; attribute not used by the plan's business unit (R9) and option not fitting the parent (R8/R11) checked first | Approved — left unchanged so it keeps its approval · Submitted — left unchanged so it stays with its approvers · Not used by plan '<plan>' — its business unit doesn't include <unit> · '<option>' doesn't fit the <parent label> of this <level> · The model '<model>' doesn't allow '<type>' · It can't change while the activity is submitted or approved. · Dates must be inside the <parent level>'s dates (<range>). · Already set — nothing to change |
| `submit` | activities only (FR-012, FR-013): *Created* → submit, *Declined* → re-submit, through 008 (notifications included); `SubmitProblems` first, every reason at once | Only activities go through approval · Already submitted · Already approved · "<n> fields needed: Owner, Company code; line item L-1004: GL code" and each other problem text of 008's check |
| `copy` | `Copy` into the target (FR-014, R13: everything below, *Created*, no approvals/comments/history/GSAP, names " (copy)" on a clash — D-131); activities whose type the target model does not allow are skipped (D-132); `newRecordId` in the outcome | The model '<model>' doesn't allow '<type>' · '<name>' runs <dates>, but '<target>' runs <dates>. · owning-service messages |
| `move` | `Move` (FR-015, D-033): keeps IDs, status, line items, history; dates never change (FR-017) | Already in '<target>' · '<target>' already has a <level> called '<name>' · '<name>' runs <dates>, but '<target>' runs <dates>. · Activities move only within their plan · Initiatives move only within their plan · Campaigns move only between plans of the same year |
| `delete` | top-level selected records → `Delete` (each its own recycle-bin entry, R14; submitted activities end their round as 008 says); `deleteSummary` before deleting (FR-019) | owning-service messages |
| all | record gone → failed "No longer exists" (edge case) | |

**Effects** (preview and result, FR-018) for copy and move into another plan, from `PlanChangeEffects`
over the activities copied or moved (with everything below them): activities skipped because the target
model does not allow their type; activities kept with a type the target model does not allow (D-132);
activities whose MQL and revenue are recalculated with the target model (those without "Set later stages
myself"); attribute groups hidden (values kept) with their activity count; attribute groups added with the
number of needed-to-submit fields that will be empty; "Totals and budgets of '<plan A>' and '<plan B>'
update". For `changeField` on a parent attribute: dependent values cleared (D-135).

**Summary** (FR-021): `counts {done, skipped, failed}` and `verb` (changed, submitted, copied, moved,
deleted); every record's outcome with ID, name, plan and reasons.

### Package `imports`

**`Planning`** (adapter on 002–007): `Plan(ctx, planID)` (name, year, model, business unit, owner,
campaigns with initiatives — IDs, names, owners, dates), `PlanOf(ctx, recordID)`, `People(ctx)` (active
e-mails → IDs), `ActivityTypes(ctx, modelID)` (active, allowed, default planned cost), `Workflows(ctx)`
(active), `ActivityAttributes(ctx, businessUnitIDs)` (active activity-level attributes shown for that
business unit, with options, parents, markings, types), `CreateCampaign`, `CreateInitiative`,
`ValidateNewActivity(ctx, a, initiativeID, input) []errs.FieldError`, `CreateActivity`,
`ActivityState(ctx, id)` (exists, version, missing needed fields), `RecordsBelow(ctx, id)`, `Delete(ctx, a,
id)`.

**Service**

| Operation | Rules (FR) |
|---|---|
| `List(ctx, a, planID) (PlanImportPage, error)` | the plan's imports, newest first; `canImport` and the reason (FR-024, FR-033) |
| `Template(ctx, a, planID) (File, error)` | refused without a campaign and an initiative (FR-024); columns of data-model §3 for the plan's business unit (R9) with markings; Allowed values (the plan's campaigns "Name (C-…)", initiatives "Campaign › Initiative (I-…)", types allowed by its model, active workflows, active people's e-mails, every option of every choice column); Instructions (D-138); file name "NorthStar import template – <plan>.xlsx" (FR-025) |
| `Check(ctx, a, planID, Upload) (Check, error)` | `run(…, commit=false, rows=all)` (FR-026 – FR-031) |
| `Import(ctx, a, planID, Run) (Detail, error)` | `run(…, commit=true, rows=confirmed)`; stores the `Import` (FR-032, FR-033) |
| `Get(ctx, a, importID) (Detail, error)` | the import with its undo preview: records still there, changed since the import (version > 1 or status no longer *Created*), records added under them since, already deleted (edge case) |
| `Undo(ctx, a, importID) (Detail, error)` | one Write, note "Undo of import (IMP-…)": `Delete` every top-level record the import created (created campaigns; created initiatives under existing campaigns; imported activities under existing initiatives) with everything below; records already gone are skipped; sets `undoneAt/By`; a second Undo → `already_resolved` with `{how: undone, by, at}` (FR-034, D-139) |

**`run`** (one Write, discarded when `commit` is false):

1. Decode base64 (≤ 5 MB), read `.xlsx` (zip signature) or CSV, else "Choose an .xlsx or .csv file." /
   "NorthStar can't read this file — save it as .xlsx or .csv and try again." / "The file is empty — use
   the template.".
2. Headings: matched by name ignoring case and surrounding spaces, plus the aliases of data-model §3;
   every missing **Required** column → `validation_failed` on `file`, one message each: "The file has no
   '<column>' column — download a fresh template"; unrecognised columns → `ignoredColumns` (FR-026).
   Rows: skip empty rows and marking rows; more than 1,000 data rows → "The file has <n> activity rows —
   import up to 1,000 at a time.".
3. Per row (only confirmed rows when committing): parse each cell (D-138) → problems "Not a real date",
   "Enter a whole number of 0 or more.", "Planned cost can't be negative.", "No active person has this
   e-mail", "Not an option of <column>", "'<value>' is no longer allowed for <column>" (inactive option,
   edge case), "'<value>' doesn't fit <parent column> '<parent value>'", "Shares must be numbers from 0 to
   100"; resolve Campaign (ID or name in the plan; "C-1004 belongs to another plan ('<plan>')"; unknown →
   with Create missing on: created with owner = plan owner and dates 1 Jan – 31 Dec of the plan's year;
   off: "No campaign called '<name>' in this plan — turn on 'Create missing campaigns and initiatives' or
   use a name from the template") and Initiative (ID or name within that campaign; new ones take the
   campaign's owner and dates, R2) (FR-029, FR-030).
4. File-level: the same activity name twice for one initiative → both rows "Appears twice in the file for
   '<initiative>' (rows <a> and <b>)".
5. Defaults for empty cells (FR-028, field catalogue §4.1): owner and dates from the initiative, planned
   cost from the type's default, the approval workflow when only one is active, Ongoing = No, split values
   without shares split evenly (R10).
6. `ValidateNewActivity` with the row's values (cells with a parse problem left out) → its field problems
   mapped to columns (data-model §3.3), so every problem of the row is listed at once (FR-027); "Already
   exists — import only adds new activities" for a name taken in the initiative (edge case).
7. A row without problems → `CreateActivity` (*Created*, readable ID, no line items, FR-032; its `created`
   entry carries the import note). A confirmed row that now has problems → *failed* with its reasons; rows
   not confirmed → *left out*.
8. Result: counts "N rows · N ready · N with problems", every problem (row, column, value found, reason),
   `willCreate` (containers created for ready rows only), `needMoreFields` (ready rows whose activity still
   misses a needed-to-submit field — "42 activities will need more fields before submitting").

### Stored records, history kinds, activity log

Stored: `bulk.Action` (table `bulk_actions`) and `imports.Import` (table `imports`), data-model §1–§2. They
are records about an action, not business records: no version, no history kind (they are never deleted).
Activity-log entries are written by the owning packages for every record created, changed, copied, moved,
submitted or deleted, each with the note of D-129 (FR-023). Deletes go to the recycle bin through the owning
packages (one bin entry per top-level record, R14).

### API operations

| Method and path | Request | Response | Errors |
|---|---|---|---|
| `GET /api/bulk-actions/fields?level=` | `level` = campaign \| initiative \| activity | `200 BulkFieldList` | 400, 401 |
| `POST /api/bulk-actions/targets` | `BulkTargetQuery` | `200 BulkTargetPage` | 400, 401 |
| `POST /api/bulk-actions/preview` | `BulkActionRequest` | `200 BulkActionResult` (`preview: true`, `id: null`) | 400, 401 |
| `POST /api/bulk-actions` | `BulkActionRequest` | `201 BulkActionResult` | 400, 401 |
| `POST /api/bulk-actions/{bulkActionId}/undo` | — | `200 BulkUndoResult` | 400 (not a delete), 401, 404, 409 (`undo_expired`, `already_resolved`, `name_taken`, `restore_parent_first`) |
| `GET /api/plans/{planId}/imports` | — | `200 PlanImportPage` | 401, 404 |
| `GET /api/plans/{planId}/import-template` | — | `200` `.xlsx` with `Content-Disposition` | 400 (not ready), 401, 404 |
| `POST /api/plans/{planId}/imports/check` | `ImportUpload` | `200 ImportCheck` | 400 (file problems), 401, 404 |
| `POST /api/plans/{planId}/imports` | `ImportRun` | `201 ImportDetail` | 400, 401, 404 |
| `GET /api/imports/{importId}` | — | `200 ImportDetail` | 401, 404 |
| `POST /api/imports/{importId}/undo` | — | `200 ImportDetail` | 401, 404, 409 (`already_resolved`) |

`404` texts: "This plan no longer exists — someone may have deleted it." (004's), "This import no longer
exists.", "This bulk action no longer exists.". `400` "Only a bulk delete can be undone.".

### Wiring and demo data

`app.go`: `bulk.NewService(tx, bulk.NewJSONRepository(), bulk.NewPlanningAdapter(…), generator, hist,
now)` and `imports.NewService(tx, imports.NewJSONRepository(), imports.NewPlanningAdapter(…), generator,
hist, now)`; their handlers' `Routes`. No kinds to register.

Demo data: **none**. A seeded import would offer **Undo import** on the demo story's own activities; the
end-to-end tests create their own records (`frontend/e2e/bulk.spec.ts`, `import.spec.ts`).

## Frontend design

### Routes

| Path | Page | Tab title |
|---|---|---|
| `/plans/:planId/import` | `ImportPage` | Import activities · NorthStar |
| `/plans/:planId/imports/:importId` | `ImportResultsPage` | Import results · NorthStar |

No sidebar change. Entry points: plan page **⋯ More › Import activities** and the plan page's **Imports**
card (004's `PlanPage`); All work toolbar **Import activities** → `ChoosePlanDialog` → `/plans/:planId/import`.

### Selection (FR-001 – FR-007)

- `useSelection({ scopeKey, scopeName })` (`features/bulk/useSelection.ts`): `Map<id, {id, level, name,
  planId}>`; `toggle(row, { shift, orderedRows })` (Shift selects the range since the last toggle in the
  visible order), `selectRows(rows)`, `selectAllMatching(rows)`, `remove(ids)`, `clear()`, `count`,
  `byLevel`. When `scopeKey` changes (All work: year + filters + search; plan tree: plan ID + its search) it
  clears and shows "Selection cleared because the <scopeName> changed" (filters / search / plan). It also
  clears after an action completes (FR-003). Expanding, collapsing, paging and sorting keep it.
- `RowCheckbox`: native checkbox, accessible name "Select <level> '<name>'" (FR-005), Space toggles,
  Shift+Space / Shift+click range; on every campaign, initiative and activity row (context rows too), none
  on plans, line items and unassigned rows.
- `HeaderCheckbox`: "Select all rows on this page" — ticks the **matching** selectable rows of the page
  (D-134); checked / mixed / unchecked. When every such row of the page is ticked and more match, the bar
  offers **Select all <n> matching** (all matching selectable rows of the result) and **Clear** (FR-002).
- `SelectionBar` (**new shared component** `@/components/SelectionBar` — reason: the same bar sits under
  the All work grid and the plan tree, and design system §8 names "a bulk-action bar"): sticky at the
  bottom of its list, `role="region"` `aria-label="Selection"`, props `summary`, `children` (actions),
  `onClear`, `extra` (the Select all link). Hidden while nothing is selected.
- `BulkActionBar`: text "<n> selected · <counts by level>" ("12 selected · 2 campaigns, 10 activities");
  buttons **Change owner**, **Change field**, **Submit for approval**, **Copy to…**, **Move to…**,
  **Delete** (danger), **Clear** — always visible while something is selected (FR-004, FR-007). Over 1,000
  selected: every action shows "Bulk actions handle up to 1,000 records at a time — narrow the filters"
  (FR-006). **Change field**, **Copy to…**, **Move to…** with mixed levels open `MixedLevelDialog`
  ("<Action> works on one level at a time" + **Keep only campaigns** / **Keep only initiatives** / **Keep
  only activities** for the levels present, then the action's dialog opens).

### Action dialogs (all call `POST /api/bulk-actions/preview` first, then `POST /api/bulk-actions`)

| Dialog | Content |
|---|---|
| `ChangeOwnerDialog` | `PersonPicker` (003, single, active people); preview line "Set owner to <name> on <n> <records>"; skipped list if any; **Change owner** (primary) / **Cancel** (FR-008) |
| `ChangeFieldDialog` | field select from `GET /api/bulk-actions/fields?level=` (with the note "Names, IDs, money, calculated fields, Ongoing and % split fields are edited one record at a time."); value control by kind (date input, choice list searchable above 10 options, multiple choice chips, Yes/No radios) and **Clear the value** (not for required fields); preview "Set <field> to '<value>' on <n> <records>", the re-approval list with **Leave those unchanged** / **Change and restart approval** (radio, default Leave), skipped list, effects (dependent values cleared); **Change field** (FR-009 – FR-011) |
| `SubmitDialog` | preview "<n> will be submitted · <m> are not ready"; not-ready list with every reason and a link; optional comment (max 2,000); **Submit <n> activities**; with n = 0 the dialog says "Nothing can be submitted — fix the problems first" and offers only **Close** (FR-012, FR-013) |
| `CopyMoveDialog` + `TargetPicker` | `TargetPicker`: search box "Search name, ID or path", list of targets with full path and dates, the note when there is none ("No place to <copy / move> them to" or D-133's sentence); after choosing, preview with counts, effects (FR-018) and skipped list; **Copy <n> <records>** / **Move <n> <records>** (FR-014 – FR-018) |
| `BulkDeleteDialog` | from `deleteSummary`: "Delete <counts>?" · "With them go <below counts>." · "<k> of them are submitted or approved." (when k > 0) · "Plan totals will update. They move to the recycle bin; you can restore them."; **Cancel** focused; red **Delete <n> records** (FR-019) |
| `BulkResultsPanel` | `SidePanel` "Results": "<done> <verb> · <skipped> skipped · <failed> failed"; sections **Skipped** and **Failed** with ID, name (link), reasons; **Select failed items** makes the skipped and failed records the selection and closes the panel; for copies, links to the new records (FR-021) |

After a run: the selection clears; All work and plan queries are invalidated; the summary toast
"<summary>" with **View details** opens `BulkResultsPanel` (opened directly when something was skipped or
failed). After a delete: toast "<n> records moved to the recycle bin" with **Undo** for 10 seconds (alt text
"Restore them from the recycle bin") → `POST /api/bulk-actions/{id}/undo` → "<n> records restored"; refusals
show the API message (FR-020).

### Import pages (FR-024 – FR-034)

- `ImportPage`: breadcrumb Plans › <plan> › Import activities; `PageHeader` eyebrow "Import", title
  "Import activities", description "Add many activities to '<plan>' from a spreadsheet."; a three-step
  list (the current step marked with `aria-current="step"`):
  1. **Get the template** — text "The template lists this plan's campaigns, initiatives and every allowed
     value." and **Download template** (`downloadFromApi`, 010's `lib/download.ts`).
  2. **Upload** — file input "Choose a file" (`accept=".xlsx,.csv"`), checkbox "Create missing campaigns
     and initiatives" (off) with hint "New ones get their owner and dates from the plan or campaign.",
     **Check file** (primary). `readFile.ts` reads the file as base64 (FileReader).
  3. **Check and import** — `ImportCheckResult`: "<n> rows · <r> ready · <p> with problems"; ignored
     columns note; "Will create <c> campaigns and <i> initiatives" with names; "<k> activities will need
     more fields before submitting"; the problems table (Row · Column · Value found · Reason) with
     **Download problem list** (`problemsCsv.ts`, client-side CSV); **Import <r> activities** (primary;
     with r = 0: "No rows are ready to import — fix the file and upload it again.") and **Upload a
     corrected file** (back to step 2, the check is replaced).
  Not ready: `EmptyState` "Add at least one campaign and one initiative to this plan before importing
  activities" with **Add a campaign** (to the plan page).
- `ImportResultsPage`: summary cards Activities imported · Campaigns created · Initiatives created · Rows
  left out · Rows failed; card "Imported activities" (name, ID, link; first 50 + **Show all**); cards
  "Rows left out" and "Rows that failed" (row, column, reason); **Undo import** (danger) → `ConfirmDialog`
  "Undo import of '<file>'?" with "<a> activities, <c> campaigns and <i> initiatives it created move to the
  recycle bin, with anything added under them since." plus, when non-zero, "<x> of them were changed
  since the import." and "<y> records were added under them since the import." and "<z> were already
  deleted."; **Cancel** focused; red **Undo import** → toast "Import undone — the records are in the
  recycle bin". After undo the page shows "Undone by <person> on <date> at <time>".
- `PlanImportsCard` (rendered on 004's plan page): title "Imports"; rows "<date time> · <person> ·
  <file> · <n> activities imported" with status chip *Imported* (positive) / *Undone* (neutral), **View**
  and **Undo import**; empty: "No imports yet — import activities from a spreadsheet." with **Import
  activities**.
- `ChoosePlanDialog` (All work): "Import activities into which plan?" — the year's plans; a plan that is
  not ready is listed with its reason and cannot be chosen; **Continue**.

### States

| State | Shown as |
|---|---|
| Preview loading | the dialog shows `Skeleton` lines; the confirm button waits (`busy`) |
| Running | the dialog's confirm button busy ("Working on <n> records…") |
| Validation (request) | the API message under the dialog's field / `FormErrorSummary` |
| Error (unreachable, internal) | the API sentence in the dialog with **Try again**; nothing was saved |
| Record gone | outcome *failed* "No longer exists" in the results |
| Undo refused | toast with the API message ("Undo is no longer available — restore them from the recycle bin") |
| Import: plan not ready | the empty state above |
| Import: file problems (400) | messages under the file input |
| Import: plan or import gone (404) | `ErrorState` with the API message and **Go to Plans** |
| Import: already undone (409) | toast "This import was already undone by <person> at <HH:mm>." and the page refreshes |
| Loading pages | `Skeleton`; errors `ErrorState` with **Try again** |

### User-visible messages

| Where | Text |
|---|---|
| Checkboxes | Select <level> '<name>' · Select all rows on this page |
| Bar | <n> selected · <a> campaigns, <b> initiatives, <c> activities (only levels present; singular when 1) · Select all <n> matching · Clear · Change owner · Change field · Submit for approval · Copy to… · Move to… · Delete · Selection |
| Selection cleared | Selection cleared because the filters changed · Selection cleared because the search changed · Selection cleared because the plan changed |
| Limit | Bulk actions handle up to 1,000 records at a time — narrow the filters |
| Mixed levels | <Change field / Copy / Move> works on one level at a time · Keep only campaigns · Keep only initiatives · Keep only activities |
| Change owner | Change owner · New owner · Set owner to <name> on <n> <records> · Change owner |
| Change field | Change field · Field · Value · Clear the value · Names, IDs, money, calculated fields, Ongoing and % split fields are edited one record at a time. · Set <field> to '<value>' on <n> <records> · Clear <field> on <n> <records> · These <k> activities are submitted or approved — changing their dates sends them back for approval: · Leave those unchanged · Change and restart approval · Change field |
| Submit | Submit for approval · <n> will be submitted · <m> are not ready · Comment for the approvers (optional) · Submit <n> activities · Nothing can be submitted — fix the problems first · Close |
| Copy / move | Copy to… · Move to… · Search name, ID or path · Copy <n> <records> into '<target>' · Move <n> <records> to '<target>' · Copy <n> <records> · Move <n> <records> · No place to copy them to · No place to move them to · Move works within one plan — select records of one plan · Move works within one year — select campaigns of plans of one year |
| Effects | <n> activities skipped — the model '<model>' doesn't allow their type · <n> activities keep a type the model '<model>' doesn't allow — change it before submitting · MQL and revenue recalculated with the model '<model>' for <n> activities · <group> attributes hidden on <n> activities — their values are kept · <group> attributes added on <n> activities — <f> needed-to-submit fields will be empty · Totals and budgets of '<plan A>' and '<plan B>' update · <dependent field> cleared on <n> <records> because it no longer fits the new <parent field> |
| Delete | Delete <counts>? · With them go <counts below>. · <k> of them are submitted or approved. · Plan totals will update. They move to the recycle bin; you can restore them. · Cancel · Delete <n> records |
| Results | Results · <done> <verb> · <s> skipped · <f> failed · Skipped · Failed · Select failed items · View details · New: <ID> |
| Verbs | changed · submitted · copied · moved · deleted |
| Skip / fail reasons | the table of *run* above · No longer exists · Already set — nothing to change |
| Delete toasts | <n> records moved to the recycle bin · **Undo** (alt text: Restore them from the recycle bin) · <n> records restored · Undo is no longer available — restore them from the recycle bin |
| Log notes | Bulk change of <n> <records> (BLK-…) · Bulk submit of <n> activities (BLK-…) · Bulk copy of <n> <records> (BLK-…) · Bulk move of <n> <records> (BLK-…) · Bulk delete of <n> <records> (BLK-…) · Undo of bulk delete (BLK-…) · Imported from '<file>' (IMP-…) · Undo of import (IMP-…) |
| Request validation | Select at least one record. · Choose one of the listed actions. · Choose a person. · <name> is inactive — choose an active person. · Choose a field. · Choose a value, or choose Clear the value. · <field> is required — it can't be cleared. · <field> can't be changed for <level plural> here. · Choose where to copy them. · Choose where to move them. · Choose a plan. · Choose a campaign. · Choose an initiative. · <Action> works on one level at a time — keep only one level. · Use 2,000 characters or fewer. · Only a bulk delete can be undone. · This bulk action no longer exists. |
| Import page | Import · Import activities · Add many activities to '<plan>' from a spreadsheet. · 1 Get the template · 2 Upload · 3 Check and import · The template lists this plan's campaigns, initiatives and every allowed value. · Download template · Choose a file · Create missing campaigns and initiatives · New ones get their owner and dates from the plan or campaign. · Check file · Checking your file… |
| Not ready | Add at least one campaign and one initiative to this plan before importing activities · Add a campaign |
| Check result | <n> rows · <r> ready · <p> with problems · These columns aren't used and were ignored: <list> · Will create <c> campaigns and <i> initiatives · <k> activities will need more fields before submitting · Row · Column · Value found · Reason · Download problem list · Import <r> activities · Upload a corrected file · No rows are ready to import — fix the file and upload it again. |
| File problems | Choose an .xlsx or .csv file. · NorthStar can't read this file — save it as .xlsx or .csv and try again. · The file is empty — use the template. · The file is larger than 5 MB — remove unused rows or columns, or split it. · The file has <n> activity rows — import up to 1,000 at a time. · The file has no '<column>' column — download a fresh template |
| Row problems | Missing — this column is required · Not a real date · Enter a whole number of 0 or more. · Planned cost can't be negative. · Enter an amount, e.g. 12500.00 · Enter Yes or No · No active person has this e-mail · Not an option of <column> · '<value>' is no longer allowed for <column> · '<value>' doesn't fit <parent column> '<parent value>' · Shares must be numbers from 0 to 100 · <ID> belongs to another plan ('<plan>') · No campaign called '<name>' in this plan — turn on 'Create missing campaigns and initiatives' or use a name from the template · No initiative called '<name>' in '<campaign>' — turn on 'Create missing campaigns and initiatives' or use a name from the template · Appears twice in the file for '<initiative>' (rows <a> and <b>) · Already exists — import only adds new activities · Ongoing activities have no end date — leave End date empty · and 006's field messages (dates, R5, R7, lengths) |
| Template | Activities · Allowed values · Instructions · Required · Needed to submit · Optional · the instruction lines of data-model §3.4 |
| Results page | Import results · Activities imported · Campaigns created · Initiatives created · Rows left out · Rows failed · Imported activities · Show all · Rows left out · Rows that failed · Undo import · Undone by <person> on <date> at <HH:mm> |
| Undo import | Undo import of '<file>'? · <a> activities, <c> campaigns and <i> initiatives it created move to the recycle bin, with anything added under them since. · <x> of them were changed since the import. · <y> records were added under them since the import. · <z> were already deleted. · Cancel · Undo import · Import undone — the records are in the recycle bin · This import was already undone by <person> at <HH:mm>. |
| Imports card | Imports · Imported · Undone · View · Undo import · No imports yet — import activities from a spreadsheet. · Import activities · <n> activities imported |
| Choose plan | Import activities into which plan? · Continue |
| Gone | This plan no longer exists — someone may have deleted it. · This import no longer exists. · Go to Plans |

## Tests

### Go — rules (`bulk` with `fakePlanning`; `imports` with `fakePlanning` and real `sheet` files built in the test)

| Test | Proves |
|---|---|
| `bulk/service_test.go` `TestRun_LimitOf1000_FR006`, `TestRun_MixedLevelsRefusedForFieldCopyMove_FR007`, `TestRun_OtherActionsSkipRecordsTheyDontApplyTo_FR007`, `TestRun_RequestProblemsAllAtOnce` | FR-006, FR-007 |
| `TestPreview_EqualsRunAndSavesNothing_D_n` (store files, log and counters unchanged after preview) | D-126, FR-011, FR-013 |
| `actions_test.go` `TestChangeOwner_CampaignsInitiativesActivities_LineItemsKeepOwners_FR008`, `TestChangeOwner_AlreadySetIsSkipped_D_o` | FR-008 |
| `TestFields_OfferedPerLevel_NoNamesMoneySplitsOngoing_FR009` | FR-009 |
| `TestChangeField_SkipsTypeNotAllowedByModel_R7_FR010`, `TestChangeField_SkipsDatesOutsideParentOrChildren_FR010`, `TestChangeField_SkipsAttributeNotUsedByBusinessUnit_R9_FR010`, `TestChangeField_SkipsOptionNotFittingParent_FR010`, `TestChangeField_TypeChangeKeepsPlannedCost_FR010`, `TestChangeField_ParentAttributeClearsDependentValues_D_w`, `TestChangeField_MultipleChoiceReplacesValues_FR009`, `TestChangeField_ClearNotForRequired_FR009` | FR-009, FR-010 |
| `TestChangeField_DatesOnApproved_LeaveUnchangedSkipped_FR011`, `TestChangeField_DatesOnApproved_RestartApprovalToCreated_FR011`, `TestPreview_ListsReapproval_FR011` | FR-011 |
| `TestSubmit_CreatedAndDeclined_FR012`, `TestSubmit_SkipsWithEveryReasonAtOnce_FR013`, `TestSubmit_CampaignsAndApprovedSkipped_FR013` | FR-012, FR-013 |
| `TestCopy_EverythingBelow_CreatedNoHistory_R13_FR014`, `TestCopy_NameMadeUniqueOnlyOnClash_D_s`, `TestCopy_ActivitiesWithTypeNotAllowedSkipped_D_t`, `TestCopy_DatesOutsideTargetSkipped_FR017` | FR-014, FR-017 |
| `TestMove_RulesOfD033_FR015`, `TestMove_KeepsIdsStatusLineItems_FR015`, `TestMove_NameTakenOrAlreadyThereSkipped_FR017`, `TestMove_DatesNeverChange_FR017` | FR-015, FR-017 |
| `targets_test.go` `TestTargets_CopyAnyPlan_MoveRules_ExcludesSelectionAndBelow_FR016`, `TestTargets_SelectionAcrossPlansHasNoMoveTarget_D_u`, `TestTargets_SearchAndPath_FR016` | FR-016 |
| `TestEffects_OtherModelAndBusinessUnit_FR018` | FR-018 |
| `TestDelete_SummaryCountsNestedOnce_FR019`, `TestDelete_OneBinEntryPerTopLevelRecord_FR020`, `TestUndo_RestoresAllInOneStep_FR020`, `TestUndo_RefusedAfterTwoMinutesChangesNothing_D_r`, `TestUndo_OnlyForDeletes` | FR-019, FR-020 |
| `TestResult_CountsVerbAndReasons_FR021` | FR-021 |
| `TestRun_RecordsIndependent_GoneIsFailed_OthersKept_FR022`, `TestRun_UnexpectedErrorSavesNothing_D_n` | FR-022 |
| `TestRun_OneLogEntryPerRecordWithNote_FR023` (with the real `history` service) | FR-023 |
| `TestBulk_500Records_SC007` (skipped with `-short`) | SC-007 |
| `history/note_test.go` `TestWithNote_AppendedToDetails_FR023`, `TestWithNote_InnermostWins` | D-129 |
| `imports/template_test.go` `TestTemplate_RefusedWithoutCampaignAndInitiative_FR024`, `TestList_CanImportAndReason_FR024` | FR-024 |
| `template_test.go` `TestTemplate_ColumnsMarkingsAllowedValuesInstructions_FR025`; `columns_test.go` `TestColumns_OnlyAttributesOfThePlansBusinessUnit_R9_FR025`; `service_test.go` `TestTemplate_FilledWithAllowedValuesImportsWithoutProblems_SC005` (round trip: template → filled → check → 0 problems) | FR-025, SC-005 |
| `columns_test.go` `TestHeadings_AnyOrderAndCase_Aliases_Unrecognised_FR026`, `TestHeadings_MissingRequiredStopsCheck_FR026`; `parse_test.go` `TestParse_LimitsAndFormats_FR026`, `TestParse_CellFormats_D_z` | FR-026 |
| `TestCheck_EveryProblemOfEveryRowAtOnce_FR027`, `TestCheck_NamesUniqueInInitiativeAndInFile_FR027`, `TestCheck_DeactivatedOptionReported_EdgeCase` | FR-027 |
| `TestCheck_NeededToSubmitEmptyAllowed_DefaultsAndEvenSplit_FR028` | FR-028 |
| `TestCheck_CampaignAndInitiativeByIdOrName_IdFromOtherPlan_FR029` | FR-029 |
| `TestCheck_CreateMissingOnAndOff_FR030` | FR-030 |
| `TestCheck_CreatesNothing_FR031` | FR-031 |
| `TestImport_CreatedNoLineItemsWholeCostUnassigned_FR032`, `TestImport_OnlyConfirmedRows_FailedAndLeftOut_D_x` | FR-032 |
| `TestImport_StoredAndListedForThePlan_FR033` | FR-033 |
| `TestUndoImport_MovesEverythingCreatedAndAddedSinceToBin_RecordsWho_FR034`, `TestUndoImport_PreviewCountsChangedAndAdded_EdgeCase`, `TestUndoImport_Twice_AlreadyResolved` | FR-034 |
| `TestImport_LogEntriesCarryImportNote_FR023` | FR-023 |
| `TestCheck_1000Rows_SC007` (skipped with `-short`) | SC-007 |
| `platform/sheet` `TestReadXLSX_SharedInlineNumbersDates`, `TestReadXLSX_FirstOrActivitiesSheet`, `TestReadXLSX_RoundTripWithWriter`, `TestReadCSV_BOMCommaSemicolonQuotes` | D-120, D-138 |
| `platform/web` `TestReadJSONUpTo_LimitAndRules` | D-136 |

### Go — handler and contract (`package bulk_test`, `package imports_test`, `apitest` with the seed)

`TestFields_Contract`, `TestTargets_Contract`, `TestPreview_Contract`, `TestRun_Contract` (201, each
action once on seed records), `TestUndoBulkDelete_Contract` (200, 409 `undo_expired` after
`Advance(3 min)`), `TestListImports_Contract`, `TestTemplate_Contract` (content type, disposition, opens with
`ReadXLSX`), `TestCheck_Contract` (200; 400 with every missing column), `TestImport_Contract` (201),
`TestGetImport_Contract`, `TestUndoImport_Contract` (200, then 409), `TestEveryBulkAndImportOperationRequiresSignIn`.

### Web — component and unit tests

| File | Proves |
|---|---|
| `useSelection.test.ts` | toggle, Shift range, select rows, select all matching, kept on page/expand/sort, cleared with message on scope change and after an action (FR-001 – FR-003, FR-005) |
| `RowCheckbox.test.tsx`, `HeaderCheckbox.test.tsx` | accessible names, Space, mixed state, header ticks matching rows of the page only (FR-002, FR-005, D-134) |
| `components/SelectionBar/SelectionBar.test.tsx` | region name, summary, actions, Clear, hidden when empty |
| `BulkActionBar.test.tsx` | count by level text, every action visible, Select all N matching, limit message over 1,000, mixed-level dialog with Keep only (FR-002, FR-004, FR-006, FR-007) |
| `ChangeOwnerDialog.test.tsx` | person picker, preview line, run, summary toast, selection cleared (FR-008, FR-021) |
| `ChangeFieldDialog.test.tsx` | offered fields + note, value controls, Clear the value, re-approval choice sent as `restartApproval`, skipped reasons (FR-009 – FR-011) |
| `SubmitDialog.test.tsx` | "11 will be submitted · 4 are not ready", reasons with links, comment, nothing-ready case (FR-012, FR-013) |
| `CopyMoveDialog.test.tsx` / `TargetPicker.test.tsx` | search, paths, no-target notes, effects, confirm labels (FR-014 – FR-018) |
| `BulkDeleteDialog.test.tsx` | counts text, submitted/approved line, Cancel focused, red button label; Undo toast restores (FR-019, FR-020) |
| `BulkResultsPanel.test.tsx` | summary text per verb, skipped/failed lists with links, Select failed items (FR-021) |
| `bulkText.test.ts` | "12 selected · 2 campaigns, 10 activities", summary sentence, singular forms |
| `AllWorkGrid.test.tsx` (010, extended) / plan tree test (004–006, extended) | checkboxes on campaign/initiative/activity rows only; ticking a campaign selects only it (FR-001) |
| `ImportPage.test.tsx` | not-ready state with link (FR-024), template download (FR-025), upload + Create missing + check, problems table + Download problem list, Import N, Upload a corrected file replaces the check (FR-026 – FR-031) |
| `ImportResultsPage.test.tsx` | counts, lists, Undo import confirmation texts incl. changed/added counts, undone state (FR-032 – FR-034) |
| `PlanImportsCard.test.tsx`, `ChoosePlanDialog.test.tsx` | list per plan with Undo until undone; empty state; plan chooser with unavailable plans (FR-024, FR-033) |
| `readFile.test.ts`, `problemsCsv.test.ts` | base64 of a file; CSV with header and quoted values |

### End-to-end

- `frontend/e2e/bulk.spec.ts` — **US4 then US1**: as Priya in All work, select two activities of FY2026
  Lubricants Europe, **Copy to…** another initiative → "2 copied"; search " (copy)", select both copies,
  **Change owner** to Daniel Okoro → "2 changed"; open one copy → owner Daniel, history entry with "Bulk
  change of 2 activities"; select both copies, **Delete** → dialog texts, **Undo** → back; delete again.
  Also: header checkbox selects the page's matching rows; a mixed selection with **Copy to…** offers
  **Keep only activities**. axe on the bar and each dialog.
- `frontend/e2e/import.spec.ts` — **US2**: as Mei Lin open FY2026 Mobility China › **Import activities**;
  download the template (file name checked); upload a CSV written by the test (10 rows, a unique campaign
  and initiative name, type "Event", owner mei.lin@northstar.demo; row 5 date "31 Feb 2026", row 9 type
  "Teleportation") with **Create missing campaigns and initiatives** on → "10 rows · 8 ready · 2 with
  problems", both problems listed; **Import 8 activities** → results page; the activities appear in the
  plan tree as *Created*; **Undo import** → confirmation → gone; the plan's Imports card shows *Undone*.
  axe on the three steps and the results page.

## Dependencies

### Uses (expected names; the consolidation step maps them)

| From | What |
|---|---|
| 001 | `store.Transactor`, `ids.Generator` (prefixes `BLK`, `IMP`), `clock`, `actor`, `errs`, `web`, `history.Service` (`Undo`, `Log`) + `history.BinEntry`, `RecycleBinEntry`/`RestoreResult`/`ActorRef`/`Resolution` schemas, `apitest`; web: `SidePanel`, `ConfirmDialog`, `useToast`, `Button`, `Card`, `PageHeader`, `SummaryCard`, `EmptyState`, `ErrorState`, `Skeleton`, `FormErrorSummary`, `TextField`, `StatusChip`, `Tooltip`, `InfoTip`, `lib/format` (`useUndoableDelete` is not used: it undoes one entry, a bulk delete undoes many through `POST /api/bulk-actions/{id}/undo`) |
| 002 | `settings`: activity-level Shell attributes (type, marking, options with active flags and parent options, business-unit condition), activity types (active, models allowing them, default planned cost), models; `money.Amount`/`money.Parse`; R10 even split helper (`settings.EvenSplit(n)` or 006's) |
| 003 | active people by e-mail; web `PersonPicker` (`@/components/PersonPicker`, single mode) |
| 004 | `plans` (name, year, model, business unit, owner); `PlanId` OpenAPI parameter and the plan's 404 message; web `PlanPage` (adds **Import activities** to ⋯ More and renders `PlanImportsCard`), route `/plans/:planId` |
| 005 | `campaigns.Service`: `CreateCampaign`, `CreateInitiative`, `UpdateCampaign` / `UpdateInitiative` (PATCH with pointer fields: owner, dates, attributes), `CopyCampaign(ctx, a, id, targetPlanID)`, `CopyInitiative(ctx, a, id, targetCampaignID)`, `MoveCampaign`, `MoveInitiative`, `DeleteCampaign`, `DeleteInitiative` (→ `history.BinEntry`), `PlanChangeEffects(ctx, activityIDs, targetPlanID)` (the consequences 005 FR-024 lists); records with dates and parents; web plan-tree rows (`PlanTree` with a `selection` prop added here) |
| 006 | `activities.Service`: `ValidateNew(ctx, a, initiativeID, ActivityInput) []errs.FieldError` (every check of `CreateActivity` without saving), `CreateActivity`, `UpdateActivity` (PATCH: owner, dates, type, workflow, attributes; `RestartApproval bool` for §2.5 changes), `CopyActivity(ctx, a, id, targetInitiativeID)` **into an initiative of any plan**, `MoveActivity`, `DeleteActivity`; `ActivityInput` JSON field names `name`, `externalName`, `activityTypeId`, `ownerId`, `startDate`, `endDate`, `ongoing`, `plannedCost`, `approvalWorkflowId`, `description`, `inq`, `attributes.<attributeId>`; display status; missing needed-to-submit fields |
| 007 | line items move, copy and delete with their activity (inside 005/006's functions) |
| 008 | `approvals.Service.CheckSubmit(ctx, a, activityID) ([]approvals.Problem, error)` (every problem of the ready-to-submit check, missing fields grouped per record), `Submit(ctx, a, activityID, comment)` and `Resubmit(ctx, a, activityID, comment)` (notifications included) |
| 010 | All work grid (`AllWorkGrid`, `GridRow`, pager, `AllWorkResult` rows with `role`, `level`, `recordId`, `planId`), `AllWorkPage` toolbar, `platform/sheet` writer, `lib/download.ts`, `useAllWorkOptions` (plans of the year for `ChoosePlanDialog`) |
| 011 | recycle-bin page for "restore them from the recycle bin" links |

### Provides

- `history.WithNote` (any later bulk-style operation), `web.ReadJSONUpTo`, `sheet.ReadXLSX` / `ReadCSV`.
- `@/components/SelectionBar`; `features/bulk` selection and bar, reusable by any future list (e.g. the
  approvals inbox when bulk approval is specified).

## Requirement traceability

| FR | Implemented in | Tested by |
|---|---|---|
| FR-001 – FR-005 | `useSelection`, `RowCheckbox`, `HeaderCheckbox`, `SelectionBar`, `BulkActionBar`; grid and tree integration | selection and bar tests, grid/tree tests, e2e bulk |
| FR-006, FR-007 | `bulk.run` checks, `BulkActionBar`, `MixedLevelDialog` | `TestRun_*_FR006/FR007`, `BulkActionBar.test.tsx` |
| FR-008 – FR-011 | `bulk/actions.go`, `Fields`, owner and field dialogs | action tests, dialog tests, e2e bulk |
| FR-012, FR-013 | submit action, `SubmitDialog` | submit tests, `SubmitDialog.test.tsx` |
| FR-014 – FR-018 | copy and move actions, `Targets`, effects, `CopyMoveDialog` | copy/move/targets/effects tests, dialog tests, e2e bulk |
| FR-019, FR-020 | delete action, `Undo`, `BulkDeleteDialog` | delete/undo tests, dialog test, e2e bulk |
| FR-021 – FR-023 | `Result`, one transaction, `history.WithNote` | result, independence and log tests |
| FR-024 – FR-031 | `imports` template, check, parse; `ImportPage` | imports tests, `ImportPage.test.tsx`, e2e import |
| FR-032 – FR-034 | `imports` import, list, undo; results page and card | imports tests, page/card tests, e2e import |

## Complexity Tracking

| Choice | Why needed | Simpler alternative rejected because |
|---|---|---|
| `bulk.Planning` and `imports.Planning` interfaces with one adapter each | Each package reaches five owning packages; the interface keeps the coupling in one file, documents exactly what is reused, and lets every rule test run on a fake | Calling five services everywhere couples every function and makes rule tests need the whole application |
| Preview by running the action and discarding the transaction | Guarantees the preview shows exactly what the run will do (skips, re-approvals, effects) | A separate dry-run path would duplicate every rule and drift |
| Own `.xlsx` reader in `platform/sheet` | FR-026 accepts spreadsheets; standard library only | A spreadsheet library is a new dependency (constitution VII) |
| `history.WithNote` (context value) | FR-023 needs the bulk/import note on entries written by five packages | Passing a note through every owning service's signature changes all of them |

## Consolidation (2026-10-19, binding)

The Plan-module consolidation (the cross-feature analyze step) settled the points below. They and the build guide,
[docs/sdd/plan-module-build.md](../../docs/sdd/plan-module-build.md), win where this plan, its data model or its tasks
say otherwise (D-200).

- **Contract (D-192).** The fragment is merged into `backend/api/openapi.yaml`; the merge task (T001) is done. The
  feature's operations are listed in `notYetServed` in `backend/internal/app/app_test.go`; the backend removes each one in
  the change that serves it. Agents do not change the contract; a problem with it is reported.
- **Decisions.** This plan's proposed decisions are numbered D-126–D-139 in `docs/sdd/decision-log.md` (labels kept in the
  table above); the consolidation decisions are D-189–D-207.

- **Shared parts (D-197).** 012 owns `SelectionBar` and adds the selection prop to 004's `TreeTable` and 010's `AllWorkGrid`; it uses
  003's `PersonPicker`, 005's `DateRangeField`, 002's `ChoiceField`, `ShellAttributeFields`.
- **Versions.** 008 refuses stale submissions: the bulk adapter passes the version it reads inside the same transaction (D-127).
- **Copies (D-195).** Bulk copies use the single-copy seams with `yearShift`; names follow D-131.
- **Money.** 002's names (`ParseAmount`, `money.PercentSplitEvenly` for even splits).
