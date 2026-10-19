# Data Model: Bulk Actions and Import

**Feature**: `012-bulk-actions-and-import` · **Plan**: [plan.md](plan.md) · **Contract**: [contracts/openapi.yaml](contracts/openapi.yaml)

This feature creates and changes **no business record type** of its own: campaigns, initiatives and
activities are created, changed, copied, moved and deleted by their owning features (005, 006, 008), with
those features' fields, rules and activity-log entries. It stores two **records about actions** — bulk
actions and imports — so that their deletes and imports can be undone and each plan can list its imports.
Conventions of feature 001's data model apply.

---

## 1. Bulk action — table `bulk_actions`

One run (not a preview) of a bulk action. Never edited except to record an Undo; never deleted; no
version; no history kind.

| Field | Type | Rules |
|---|---|---|
| `id` | string | `BLK-1001`, … (readable-ID counter `BLK`, proposed prefix); taken at the start of the run |
| `action` | enum | `changeOwner`, `changeField`, `submit`, `copy`, `move`, `delete` |
| `note` | string | The log note, e.g. "Bulk change of 20 activities (BLK-1001)" (plan D-129) |
| `selected` | `{campaigns, initiatives, activities}` | counts of the selection after duplicates were removed |
| `done`, `skipped`, `failed` | integer | outcome counts (FR-021) |
| `recycleBinEntryIds` | list of string | `delete` only: one `BIN-…` per top-level deleted record (R14, FR-020) |
| `at` | timestamp | when it ran |
| `by` | ActorRef | who ran it |
| `undo` | `{by, at}` or null | `delete` only: set when Undo succeeded (plan D-130) |

Per-record outcomes are returned to the web app (`BulkActionResult.outcomes`) but not stored; the activity
log holds every change with its note.

### 1.1 Request (`BulkActionRequest`, not stored)

| Field | For | Rules (messages in the plan) |
|---|---|---|
| `action` | all | one of the six |
| `recordIds` | all | 1 – 1,000 IDs of campaigns (`C-`), initiatives (`I-`) or activities (`A-`); the level comes from the prefix; duplicates ignored |
| `ownerId` | `changeOwner` | an active person |
| `field` | `changeField` | `{key, value}`; `key` from `GET /api/bulk-actions/fields?level=`; `value` null = **Clear the value** (not for required fields) |
| `field.value` | | `{date}` for start/end date (`YYYY-MM-DD`); `{optionIds}` for single (one ID) and multiple choices, activity type and approval workflow; `{yesNo}` for yes/no |
| `targetId` | `copy`, `move` | a plan (campaigns), campaign (initiatives) or initiative (activities) |
| `restartApproval` | `changeField` | default false = **Leave those unchanged** |
| `comment` | `submit` | optional, at most 2,000 characters |

`changeField`, `copy` and `move` need one level (FR-007).

### 1.2 Outcome (per selected record, not stored)

| Outcome | Meaning |
|---|---|
| `done` | changed / submitted / copied (with `newRecordId`) / moved / deleted |
| `skipped` | a business rule prevented it; `reasons` lists every reason in plain language |
| `failed` | it could not be saved: the record no longer exists ("No longer exists") |

### 1.3 Delete summary (`deleteSummary`, not stored, FR-019)

| Part | Rule |
|---|---|
| `selected` | the top-level selected records by level (a record under another selected record is counted once, under its ancestor) |
| `below` | campaigns, initiatives, activities and line items that go with them and were not selected themselves |
| `submittedOrApproved` | activities among all of them whose status is Submitted or Approved (incl. In progress, Complete) |

### 1.4 Offered fields (`BulkField`, FR-009)

| Level | Fields |
|---|---|
| Campaign | Start date, End date; active yes/no, single-choice and multiple-choice campaign attributes (Cluster, Region, Insight session completed, …) |
| Initiative | Start date, End date; active yes/no, single- and multiple-choice initiative attributes |
| Activity | Start date, End date, Activity type (required), Approval workflow; active yes/no, single- and multiple-choice activity attributes, never % split ones |

Never offered: names, IDs, money, calculated values, "coming later" values, Ongoing (R6), % split
attributes (R10), text and number attributes; owner has its own action.

---

## 2. Import — table `imports`

One completed import into one plan (FR-033). A check that is never confirmed stores nothing (FR-031).

| Field | Type | Rules |
|---|---|---|
| `id` | string | `IMP-1001`, … (readable-ID counter `IMP`, proposed prefix) |
| `planId` | string | the plan imported into |
| `fileName` | string | as uploaded, at most 255 characters (longer names are cut) |
| `createMissing` | boolean | the "Create missing campaigns and initiatives" choice |
| `rowsRead` | integer | data rows in the file (empty and marking rows not counted) |
| `activityIds` | list of string | activities created, in row order |
| `campaignIds`, `initiativeIds` | list of string | containers created by this import |
| `leftOut` | list of Problem | rows not confirmed (they had problems at the check) with their problems |
| `failed` | list of Problem | confirmed rows that could not be imported, with their reasons |
| `importedAt`, `importedBy` | timestamp, ActorRef | |
| `undoneAt`, `undoneBy` | timestamp, ActorRef or null | set by **Undo import**; then the import cannot be undone again (`already_resolved`) |

**Problem**: `{row, column, value, reason}` — `row` is the spreadsheet's own row number; `column` the
heading as in the template ("" for a whole-row problem); `value` the text found (at most 200 characters).

Status shown: **Imported** (positive) while `undoneAt` is null, then **Undone** (neutral).

Life cycle: check (nothing stored) → import (stored, records created) → undo (records moved to the recycle
bin, `undoneAt/By` set).

**Undo preview** (calculated on read, FR-034 edge case): activities, campaigns and initiatives of the
import that still exist; **changed since the import** = those whose version is above 1 or (activities)
whose status is no longer Created; **added since the import** = records under them not created by this
import (line items, other activities, initiatives); **already deleted** = those no longer stored.

---

## 3. Import template and file

### 3.1 Columns (sheet "Activities", FR-025)

Row 1 = headings, row 2 = markings, data from row 3. Headings are matched ignoring letter case and
surrounding spaces, in any order (FR-026).

| Heading | Also accepted | Marking | Value | Activity field (006) | Default when empty (FR-028) |
|---|---|---|---|---|---|
| Campaign | Campaign ID | Required | campaign name or ID in the plan | — (resolves the parent) | — |
| Initiative | Initiative ID | Required | initiative name or ID in that campaign | — (parent `initiativeId`) | — |
| Activity name | Name | Required | text ≤ 120 | `name` | — |
| Activity type | Type | Required | type name allowed by the plan's model | `activityTypeId` | — |
| Owner (e-mail) | Owner, Owner e-mail | Needed to submit | e-mail of an active person | `ownerId` | initiative owner |
| Start date | Start | Needed to submit | date | `startDate` | initiative start |
| End date | End | Needed to submit | date (empty when Ongoing) | `endDate` | initiative end |
| Ongoing | Ongoing (no end date) | Optional | Yes / No | `ongoing` | No |
| Planned cost (USD) | Planned cost | Needed to submit | amount ≥ 0 | `plannedCost` | the type's default planned cost |
| Approval workflow | Workflow | Needed to submit | active workflow name | `approvalWorkflowId` | the only active workflow, if just one |
| Description | — | Optional | text ≤ 2,000 | `description` | — |
| External name | — | Optional | text ≤ 120 | `externalName` | — |
| INQ | INQ (inquiries) | Optional | whole number ≥ 0 | `inq` | — |
| *<attribute label>* (one per active activity-level Shell attribute shown for the plan's business unit, R9, in display order) | — | the attribute's marking (Needed to submit or Optional) | by attribute type (3.2) | `attributes.<attributeId>` | split without shares → even split (R10) |

A column for an attribute of another business-unit group, or any other heading, is **ignored** and listed
("These columns aren't used and were ignored: …").

### 3.2 Cell formats (decision D-138)

| Type | Accepted | Problem |
|---|---|---|
| Date | `2026-02-28`, `28 Feb 2026`, a spreadsheet date cell | "Not a real date" |
| Yes/No | Yes, No, true, false, y, n, 1, 0 (any case) | "Enter Yes or No" |
| Amount | `12500`, `12500.5`, `12,500.00`; more than 2 decimals rounded half up | "Enter an amount, e.g. 12500.00" · "Planned cost can't be negative." |
| Whole number | `250000`, `250,000` | "Enter a whole number of 0 or more." |
| Single choice | one option label | "Not an option of <column>" · "'<value>' is no longer allowed for <column>" |
| Multiple choice | labels separated by `;` | as single choice, per label |
| Multiple choice with % split | `China 60; Hong Kong 40` (shares 0–100, ≤ 2 decimals) or `China; Hong Kong` (even split, remainder on the last) | "Shares must be numbers from 0 to 100"; a total other than 100.00 % is allowed (blocks submitting only) |
| Dependent choice | an option linked to the row's parent value | "'<value>' doesn't fit <parent column> '<parent value>'" |
| Text / long text / number attribute | as typed (002 limits) | 002's messages |

### 3.3 Problems from the activity's own rules

`ValidateNewActivity` (006) answers field errors; they are reported on the column of the field
(table 3.1): `name` → Activity name, `activityTypeId` → Activity type, `ownerId` → Owner (e-mail),
`startDate` / `endDate` → Start date / End date (R3, R5, end before start), `ongoing` → Ongoing,
`plannedCost` → Planned cost (USD), `approvalWorkflowId` → Approval workflow, `description`, `externalName`,
`inq`, `attributes.<id>` → that attribute's column. A taken name reads "Already exists — import only adds
new activities". Fields whose cell already had a format problem are not reported twice.

### 3.4 Sheets "Allowed values" and "Instructions"

- **Allowed values**: one column per choice column — Campaign ("Helix launch (C-1001)"), Initiative
  ("Helix launch › Digital & social (I-1001)"), Activity type, Owner (e-mail) ("mei.lin@northstar.demo —
  Mei Lin"), Approval workflow, Ongoing (Yes, No), and each choice attribute (dependent options as
  "<option> (for <parent option>)").
- **Instructions** (one line each): "Fill one row per activity, starting on row 3. Row 2 says which
  columns are required." · "Required columns must be filled. Needed-to-submit columns may stay empty; the
  activity can then be submitted after you add them." · "Campaign and Initiative: use a name or an ID from
  the Allowed values sheet." · "Dates: 2026-03-01 or 1 Mar 2026. Leave End date empty for an ongoing
  activity." · "Amounts in US dollars, e.g. 12500.00." · "Several values: separate them with a
  semicolon, e.g. External; Internal." · "Shares: write each value with its percentage, e.g. China 60;
  Hong Kong 40. Without percentages the values share equally." · "Import adds new activities only; it
  never changes existing ones, and it doesn't import line items."

### 3.5 File limits

`.xlsx` or `.csv` (CSV: UTF-8, `,` or `;`), at most 5 MB and 1,000 data rows; empty rows and a markings
row are skipped.

---

## 4. Selection (web app only, not stored)

| Field | Rule |
|---|---|
| records | `Map<id, {id, level, name, planId}>`; campaigns, initiatives and activities only |
| scope | All work: year + filters + search; plan tree: plan + its search. A scope change clears the selection with a message (FR-003) |
| kept | while paging, scrolling, expanding, collapsing and sorting |
| cleared | on scope change and after an action completes; **Select failed items** replaces it with the skipped and failed records |
| limit | more than 1,000 → actions refuse with the FR-006 message |

---

## 5. Status effects of bulk actions (domain model §2.3, §2.5)

| Action | Effect on an activity's status |
|---|---|
| Change owner, change of type, workflow or attributes | none (D-013) |
| Change of start or end date on Submitted/Approved | skipped (**Leave those unchanged**) or → Created with the open round ended as "Approval restarted" (**Change and restart approval**, 008 FR-022) |
| Submit | Created → Submitted; Declined → Submitted (new round, 008) |
| Copy | the copy is Created (R13) |
| Move | unchanged (D-023, D-033) |
| Delete | to the recycle bin; a Submitted activity's round ends "Cancelled — activity deleted" (008 FR-023); Undo restores exactly (D-031) |
| Import | new activities are Created, without line items: their whole planned cost is unassigned (FR-032) |
