# Data Model: Bulk Actions and Excel Import of Plans (012)

Re-written 2026-10-28 for PostgreSQL (the model of 2026-10-19 targeted the JSON store). Business records are
only ever written by their owning packages; 012 stores records *about* its runs.

## 1. Storage (migration `00023_bulk_and_imports.sql`)

```sql
bulk_actions (
  id            text PRIMARY KEY CHECK (id ~ '^BLK-[0-9]+$'),
  action        text NOT NULL CHECK (action IN ('changeOwner','changeField','submit','copy','move','delete')),
  note          text NOT NULL,
  actor_id      text NOT NULL, actor_name text NOT NULL,
  ran_at        timestamptz NOT NULL,
  undone_at     timestamptz, undone_by_id text, undone_by_name text,
  CHECK ((undone_at IS NULL) = (undone_by_id IS NULL)),
  CHECK (undone_at IS NULL OR action = 'delete'))
bulk_action_bin_entries (
  bulk_action_id text REFERENCES bulk_actions ON DELETE CASCADE, position integer CHECK (position >= 1),
  bin_entry_id   text NOT NULL, PRIMARY KEY (bulk_action_id, position))

plan_imports (
  id            text PRIMARY KEY CHECK (id ~ '^IMP-[0-9]+$'),
  plan_id       text NOT NULL REFERENCES plans ON DELETE CASCADE,
  file_name     text NOT NULL CHECK (length(file_name) BETWEEN 1 AND 255),
  imported_at   timestamptz NOT NULL, imported_by_id text NOT NULL, imported_by_name text NOT NULL,
  rows_read     integer NOT NULL CHECK (rows_read >= 0), rows_left_out integer NOT NULL CHECK (rows_left_out >= 0),
  undone_at     timestamptz, undone_by_id text, undone_by_name text,
  CHECK ((undone_at IS NULL) = (undone_by_id IS NULL)))
plan_import_records (
  import_id text REFERENCES plan_imports ON DELETE CASCADE, position integer CHECK (position >= 1),
  level text NOT NULL CHECK (level IN ('campaign','initiative','activity','lineItem')),
  record_id text NOT NULL, record_name text NOT NULL, parent_id text NOT NULL,
  action text NOT NULL CHECK (action IN ('created','updated')), PRIMARY KEY (import_id, position))
plan_import_problems (
  import_id text REFERENCES plan_imports ON DELETE CASCADE, position integer CHECK (position >= 1),
  sheet text NOT NULL, row_number integer NOT NULL CHECK (row_number >= 0), column_heading text NOT NULL,
  value text NOT NULL, reason text NOT NULL, PRIMARY KEY (import_id, position))
```

Plans are deleted to the recycle bin (their rows are removed), so a plan's imports go with it (`ON DELETE
CASCADE`); a restored plan starts with no import list (accepted: Undo import needs the records it created, which
the plan delete took too).

**Scoped reads** (scope-isolation tests): `GetBulkAction(id, actor_id)` (only the actor who ran it);
`ListPlanImports(plan_id, scope…)` and `GetPlanImport(id, scope…)` filtered with `plan_in_scope(…)`.

## 2. Bulk types (`internal/bulk/model.go`, contract `Bulk*`)

| Type | Fields |
|---|---|
| `Request` | `Action`, `RecordIDs` (1–1,000), `OwnerID`, `Field *FieldChange{Key, Value *FieldValue{Date, OptionIDs, YesNo}}`, `TargetID`, `RestartApproval`, `Comment` (≤ 2,000) |
| `Record` (port) | `ID`, `Level` (campaign · initiative · activity), `Name`, `PlanID`, `PlanName`, `Year`, `ParentID`, `Status` (activities), `Version` |
| `Outcome` | `RecordID`, `Level`, `Name`, `PlanID`, `Outcome` (done · skipped · failed), `Reasons`, `NewRecordID` |
| `Result` | `ID *string`, `Preview`, `Action`, `Verb`, `Note`, `Selected LevelCounts`, `Counts{Done,Skipped,Failed}`, `Outcomes`, `Reapproval []RecordRef`, `Effects []Effect{Kind,Count,Text}`, `DeleteSummary *{Selected, Below{…,LineItems}, SubmittedOrApproved}`, `RecycleBinEntries`, `UndoAvailableUntil *time.Time` |
| `Action` (stored) | `ID`, `Kind`, `Note`, `Actor`, `RanAt`, `BinEntryIDs`, `UndoneAt`, `UndoneBy` |
| `Field`, `FieldList` | key (`startDate`, `endDate`, `activityType`, `approvalWorkflow`, `attr.<SA-id>`), label, kind (date · singleChoice · multipleChoice · yesNo), required, options (id, label, parentOptionIds), dependsOn, condition |
| `Target`, `TargetPage` | id, level (plan · campaign · initiative), name, path "Plan › Campaign › Initiative", planId, year, dates; `Note` when none |

Verbs: changed, submitted, copied, moved, deleted. Notes: "Bulk change of 20 activities (BLK-1001)", "Bulk submit
of …", "Bulk copy of …", "Bulk move of …", "Bulk delete of …", "Undo of bulk delete (BLK-1003)".

## 3. Workbook (`internal/workbook`)

### 3.1 Sheets and columns (FR-025, FR-026)

| Sheet | Columns (Required = R; publish/submit markings from Settings and the field catalogue) |
|---|---|
| Campaigns | ID · Campaign name (R) · Owner · Start date · End date · Budget · Description · campaign attributes |
| Initiatives | ID · Campaign (R) · Initiative name (R) · Owner · Start date · End date · Budget · Description · initiative attributes |
| Activities | ID · Campaign (R) · Initiative (R) · Activity name (R) · External name · Activity type (R) · Owner · Start date · End date · Ongoing · Planned cost · Approval workflow · Description · activity attributes |
| Line items | ID · Campaign (R) · Initiative (R) · Activity (R) · Line item name (R) · Owner · Currency · Planned cost · Description · line-item attributes |
| Allowed values | one column per choice: Campaigns (`Name (C-…)`), Initiatives, Activity types (allowed by the model), Approval workflows, Currencies, People (e-mails), each attribute's options |
| Instructions | the formats of research R4, the markings, "empty ID = new record", "rows missing from the file are left alone" |

Row 1 headings, row 2 markings, data from row 3; a CSV file is the Activities sheet (its row 2 may hold the
markings). Attribute headings are the attribute labels; a label equal to a base heading gets " (attribute)".

### 3.2 Values (`workbook.Values`)

`Fields map[string]Value` keyed `name`, `ownerId`, `startDate`, `endDate`, `budget`, `description`,
`externalName`, `activityTypeId`, `ongoing`, `plannedCost`, `approvalWorkflowId`, `currency`, `attr.<id>` — each
`Value` the normalised cell (text for comparison and the typed value for the port). `Present` = the columns the
sheet has: an update changes only those.

### 3.3 Import types (contract `Import*`)

| Type | Fields |
|---|---|
| `Upload` | `FileName` (1–255), `Content` (base64, ≤ 5 MB decoded) |
| `Problem` | `Sheet`, `Row`, `Column`, `Value`, `Reason` |
| `LevelCounts` | `Level`, `Create`, `Update`, `Unchanged`, `LeftOut` |
| `Check` | `FileName`, `Rows`, `Ready`, `WithProblems`, `Levels [4]`, `Problems`, `IgnoredColumns`, `NeedMoreFields`, `NothingToChange` |
| `Import` (stored) | `ID`, `PlanID`, `FileName`, `ImportedAt/By`, `RowsRead`, `RowsLeftOut`, `Records []{Level, RecordID, Name, ParentID, Action}`, `Problems`, `UndoneAt/By` |
| `Summary` | id, planId, fileName, status (imported · undone), importedAt/By, rowsRead, created and updated counts per level, rowsLeftOut, undoneAt/By |
| `Detail` | `Import Summary`, `Problems`, `Records []{id, level, name, action, exists}`, `Undo *{campaigns, initiatives, activities, lineItems, alreadyDeleted}`, `Kept []{id, level, name, reason}` |

Notes: "Imported from 'plan.xlsx' (IMP-1001)", "Undo of import (IMP-1001)".
