# Feature Specification: Bulk Actions and Import

**Feature Branch**: `012-bulk-actions-and-import`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Select many rows: change owner/status fields, copy, move, delete; import activities from a template" (P3, depends on 010).

**References**: Domain model [§1 hierarchy, §2.3 activity status, §2.5 re-approval, §3 required fields, §4.3 unassigned balance, §4.5 goals, §5 rules R2, R3, R5, R7, R9, R10, R13, R14](../../docs/product/domain-model.md) · Field catalogue [§2 campaign, §3 initiative, §4 activity](../../docs/product/field-catalogue.md) · Design system [§5 components, §6.5 delete, §6.6 locked and read-only, §8 words ("Select"), §10 usability targets](../../docs/product/design-system.md) · Glossary *Select* · Decisions D-004, D-009, D-013, D-016, D-019 · Shell as-is document, prerequisite 2 (import template) · FA §B8 (bulk actions, copy, import)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Select many rows and change their owner or a field (Priority: P1)

Priya has 20 activities that move to a new owner. In **All work** she filters to them and ticks the
header checkbox; a bar appears at the bottom: "20 selected · 20 activities". She chooses **Change
owner**, picks Daniel Okoro and confirms; a summary says "20 changed". The same bar offers **Change
field** to set one value — for example *Company code = GB01* — on every selected record.

**Why this priority**: Shell users plan in spreadsheets partly because changing many records in Hive9 is
slow (PRISM). Selecting rows and changing them together is the base every other bulk action reuses.

**Independent Test**: In All work, select 20 seeded activities, change their owner and their company
code, and check each record, its history and the results summary.

**Acceptance Scenarios**:

1. **Given** the All work grid or a plan tree, **Then** every campaign, initiative and activity row shows
   a checkbox at all times, and ticking a campaign selects only the campaign, never its children.
2. **Given** a filtered list of 236 rows showing 50 per page, **When** the user ticks the header checkbox,
   **Then** the 50 rows on the page are selected and the bar offers "Select all 236 matching".
3. **Given** 20 selected activities, **When** the user chooses **Change owner**, picks a person and
   confirms, **Then** all 20 get the new owner, the summary reads "20 changed", and each activity's
   history shows the change marked "bulk change of 20 activities".
4. **Given** 10 selected activities, 4 of them Approved, **When** the user sets a new end date, **Then**
   before confirming the dialog lists the 4 approved activities that would need approval again
   (domain model §2.5) and offers **Leave those unchanged** or **Change and restart approval**.
5. **Given** selected activities where some cannot take the value (type not allowed by their plan's
   model, R7, or an attribute their plan's business unit does not use, R9), **When** the user confirms,
   **Then** the others are changed and the summary reads "8 changed · 2 skipped" with each skipped
   activity's ID, name and reason.

---

### User Story 2 - Import activities from a template (Priority: P1)

Mei has planned 60 activities for "FY2026 Mobility China" in a spreadsheet. She gets a template made
for the plan, fills it in and uploads it. NorthStar lists every problem at once — "Row 14 · End date ·
31 Feb 2026 · Not a real date" — and she imports the 57 ready rows, with **Undo import** if needed.

**Why this priority**: Shell's users already plan in spreadsheets (PRISM); import turns that work into
NorthStar records instead of retyping it. Hive9 offers the same import (FA §B8).

**Independent Test**: Download the template for a seeded plan, fill 10 rows (2 deliberately wrong),
upload, import the 8 valid rows, find them in the plan tree, then undo the import.

**Acceptance Scenarios**:

1. **Given** a plan with no campaign or no initiative, **When** the user chooses **Import activities**,
   **Then** it explains "Add at least one campaign and one initiative to this plan before importing
   activities" with a link to add a campaign (Shell prerequisite 2).
2. **Given** a plan with campaigns and initiatives, **When** the user gets the template, **Then** it holds
   one column per importable field, each marked Required, Needed to submit or Optional, and the allowed
   values of every choice column, including the plan's own campaigns and initiatives.
3. **Given** an uploaded file with problems, **When** the check runs, **Then** every problem in every
   row is listed at once by row, column, value found and reason, and nothing has been created yet.
4. **Given** a checked file with 57 ready rows, **When** the user chooses **Import 57 activities**,
   **Then** 57 activities are created with status Created and the rows with problems are left out.
5. **Given** a row naming a campaign that does not exist, **When** "Create missing campaigns and
   initiatives" is off, **Then** that row is a problem; **When** it is on, **Then** the check lists
   "Will create 1 campaign and 2 initiatives" and they are created with the import.
6. **Given** a completed import, **When** the user chooses **Undo import** and confirms, **Then** every
   record that import created moves to the recycle bin and plan totals return to their earlier values.

---

### User Story 3 - Submit many activities for approval (Priority: P2)

Before the approval deadline Priya selects 15 activities and chooses **Submit for approval**. The dialog
says "11 will be submitted · 4 are not ready". The summary lists the 4 with everything missing, and
**Select failed items** lets her fix and retry just those.

**Why this priority**: Every activity must be submitted; doing it one by one, discovering one problem per
attempt, is one of Hive9's slowest tasks (design system §10).

**Independent Test**: Select seeded activities in mixed statuses, submit, and compare the summary with
each activity's completeness.

**Acceptance Scenarios**:

1. **Given** selected activities that are Created or Declined and complete, **When** the user submits,
   **Then** each goes to Submitted under feature 008's rules and its approvers are notified.
2. **Given** a selected activity missing fields, **Then** it is skipped with all reasons at once, e.g.
   "3 fields needed: Owner, Company code; line item L-1004: GL code", linking to the activity.
3. **Given** a selection that also contains campaigns or already-approved activities, **Then** they are
   skipped with "Only activities go through approval" or "Already approved"; **Select failed items**
   then selects exactly the skipped and failed records.

---

### User Story 4 - Copy, move or delete many records (Priority: P2)

Daniel reuses last year's event activities: he selects 6 and chooses **Copy to…**, picks the initiative
"Rimula fleet › Events", and gets 6 copies in Created status. He moves 3 initiatives into another
campaign with **Move to…**, then deletes 2 initiatives he no longer needs with one confirmation.

**Why this priority**: Reusing proven work and tidying a plan are frequent. Single-record copy, move and
delete exist (features 004–006); this extends them to many rows, safely.

**Independent Test**: Copy 6 seeded activities into another initiative, move 2 initiatives to another
campaign, delete a selection and Undo; verify names, statuses, IDs, the recycle bin and totals.

**Acceptance Scenarios**:

1. **Given** selected activities, **When** the user copies them into an initiative, **Then** each copy
   includes its line items, starts as Created, and gets " (copy)" when the name already exists (R13).
2. **Given** a selection mixing campaigns and activities, **When** the user chooses **Copy to…**, **Then**
   it explains "Copy works on one level at a time" and offers **Keep only activities** or **Keep only
   campaigns**.
3. **Given** activities copied into an initiative of another plan, or campaigns moved to another plan, whose
   model or business unit differs, **Then**
   before confirming the dialog lists the effects with counts (FR-018); an activity whose name already
   exists there is skipped with "'Events' already has an activity called 'Trade fair'".
4. **Given** 2 initiatives and 5 activities selected, **When** the user chooses **Delete**, **Then** one
   dialog says "Delete 2 initiatives and 5 activities? With them go 6 more activities and 14 line items.
   Plan totals will update. They move to the recycle bin; you can restore them." with **Cancel** focused
   and a red **Delete 7 records** button; an activity under a selected initiative is counted once.
5. **Given** a bulk delete has just happened, **When** the user chooses **Undo**, **Then** every deleted
   record comes back with its original IDs and values.

---

### Edge Cases

- A selected record is changed or deleted by someone else before the action runs: it is processed
  against its latest values; one that no longer exists is reported as failed ("No longer exists").
- The same activity name appears twice for one initiative in the file, or already exists there: both
  are problems ("Already exists — import only adds new activities").
- An option was deactivated after the template was downloaded: the file is checked against current
  Settings and the value is reported as no longer allowed.
- Undo import after imported activities were edited or submitted: the confirmation says how many were
  changed since the import and how many records were added under campaigns or initiatives it created.

## Requirements *(mandatory)*

### Functional Requirements

**Selecting rows**

- **FR-001**: Every campaign, initiative and activity row in the All work grid (feature 010) and in the
  plan tree (features 004–006) MUST show a checkbox at all times (no separate selection mode); selecting
  a row MUST select only that record, never its children.
- **FR-002**: The header checkbox MUST select every row currently visible on the page; the bar MUST then
  offer "Select all N matching" (every record matching the current search and filters) and **Clear**.
- **FR-003**: The selection MUST be kept while paging, scrolling and expanding or collapsing rows, and MUST
  be cleared, with a short message, when the search, filters or plan change or after an action completes.
- **FR-004**: While anything is selected, a bar MUST stay visible at the bottom of the list showing the
  count by level ("12 selected · 2 campaigns, 10 activities"), the actions **Change owner**, **Change
  field**, **Submit for approval**, **Copy to…**, **Move to…**, **Delete**, and **Clear**.
- **FR-005**: Selection MUST work by keyboard (Space toggles, Shift selects a range); each checkbox MUST
  have an accessible name ("Select activity 'Launch film'"); the ticked box, not colour alone, shows it.
- **FR-006**: One bulk action MUST handle up to 1,000 records; a larger selection is refused with "Bulk
  actions handle up to 1,000 records at a time — narrow the filters".
- **FR-007**: Actions MUST stay visible whenever something is selected. **Change field**, **Copy to…** and
  **Move to…** need one level; with a mixed selection they MUST explain why and offer **Keep only
  <level>**. Other actions MUST run for the records they apply to and list the rest as skipped.

**Change owner and change field**

- **FR-008**: **Change owner** MUST set one chosen person (person picker, feature 003) as owner of every
  selected campaign, initiative and activity; line items keep their own owners.
- **FR-009**: **Change field** MUST offer the level's yes/no, single-choice and multiple-choice fields and
  Shell attributes (field catalogue §2–§4; chosen values replace the old ones), start and end dates, and
  for activities the type and approval workflow, plus **Clear the value** for fields not Required. It MUST
  NOT offer names, IDs, money, calculated or "coming later" fields, Ongoing (R6) or % split fields (R10);
  the list says these are edited one record at a time.
- **FR-010**: Each record MUST be changed under the same rules as editing it alone (features 004–008):
  a record that the single edit would refuse — activity type not allowed by its plan's model (R7), dates
  outside its parent's or children's dates (R3–R5), an attribute not used by its plan's business unit
  (R9), an option that does not fit a parent attribute — MUST be skipped with that reason. Changing an
  activity's type MUST NOT change its planned cost.
- **FR-011**: The confirmation MUST state the change and count ("Set Company code to 'GB01' on 20
  activities") and list Submitted or Approved activities whose change needs re-approval (domain model
  §2.5), offering **Leave those unchanged** (reported as skipped) or **Change and restart approval**.

**Submit for approval**

- **FR-012**: **Submit for approval** MUST submit every selected Created or Declined activity that meets
  feature 008's submit rules (domain model §2.3, §4.3, R10, D-016), notifying approvers as 008 does.
- **FR-013**: Before confirming, the dialog MUST show how many will be submitted and how many skipped;
  each skipped activity MUST list all its reasons at once with a link, including fields missing on its
  line items. Campaigns and initiatives are skipped with "Only activities go through approval".

**Copy to, move to and delete**

- **FR-014**: **Copy to…** MUST copy every selected record, with everything below it, into one target of
  the right level — campaigns into a plan, initiatives into a campaign, activities into an initiative —
  following R13 (copied activities start Created; no approvals, comments, history or GSAP values; names
  made unique with " (copy)", " (copy 2)", …).
- **FR-015**: **Move to…** MUST follow the same rules as single moves (decision D-033): selected campaigns
  move to another plan of the same year, selected initiatives to another campaign of the same plan, and
  selected activities to another initiative of the same plan — keeping IDs, status (domain model §2.5),
  line items and history.
- **FR-016**: The target picker MUST be searchable, show the full path (Plan › Campaign › Initiative),
  offer only targets allowed by FR-015, and leave out the selected records and anything below them.
- **FR-017**: A copied or moved record whose dates fall outside the target's dates (R3) MUST be skipped
  with the reason — dates are never changed silently; a moved record whose name already exists under the
  target, or that is already there, MUST be skipped. Moving MUST NOT change an activity's status.
- **FR-018**: When copied activities, or activities under a moved campaign, land in a plan with another
  model or business unit, the confirmation MUST list with counts: activities skipped because the target
  model does not allow their type (R7); MQL and revenue recalculated with the target model unless "Set
  later stages myself" (§4.5); attribute groups hidden there, which keep their values, and groups added
  whose needed-to-submit fields will be empty (R9); and that both plans' totals and budgets update.
- **FR-019**: **Delete** MUST ask once, following design system §6.5: counts by level of the selected
  records, what goes with them, how many are Submitted or Approved, the effect on totals, and that
  everything moves to the recycle bin. A record under another selected record is counted once.
- **FR-020**: After a bulk delete, the message MUST offer **Undo** for at least 10 seconds, restoring all
  of them in one step; each top-level deleted record MUST also be its own recycle-bin entry (R14).

**Results summary and history**

- **FR-021**: Every bulk action MUST end with a summary such as "33 changed · 2 skipped · 1 failed" (verb
  per action: changed, submitted, copied, moved, deleted). *Skipped* means a business rule prevents the
  change; *failed* means it could not be saved. Each skipped or failed record MUST show its ID, name,
  reason in plain language and a link, and **Select failed items** MUST make them the new selection.
- **FR-022**: Records MUST be processed independently: successful changes are kept when others are
  skipped or fail, and a single record is never left half-changed.
- **FR-023**: Every record changed, created or deleted by a bulk action or an import MUST get its own
  activity-log entry (feature 001) naming the actor and the bulk action or import it was part of.

**Import: template, upload and check**

- **FR-024**: **Import activities** MUST be available on the plan page and in All work (after choosing a
  plan) as three steps — *1 Get the template · 2 Upload · 3 Check and import* — and MUST be refused, with
  the reason and a link, until the plan has a campaign and an initiative (Shell prerequisite 2).
- **FR-025**: The template MUST be a spreadsheet generated for the chosen plan with columns Campaign and
  Initiative (name or ID), Activity name, Activity type, Owner (e-mail), Start date, End date, Ongoing,
  Planned cost (USD), Approval workflow, Description, External name, INQ, and one column per active
  activity-level Shell attribute used by the plan's business unit (R9); each column marked Required,
  Needed to submit or Optional (field catalogue §4); the allowed values of every choice column (the plan's
  campaigns and initiatives, types allowed by its model, active workflows and options, people); and short
  instructions on formats, including splits written as "China 60; Hong Kong 40".
- **FR-026**: Upload MUST accept one spreadsheet or CSV file of up to 1,000 activity rows. Columns are
  matched by heading in any order and letter case; unrecognised columns are listed and ignored; a missing
  Required column stops the check with "The file has no 'Activity type' column — download a fresh
  template".
- **FR-027**: The check MUST apply the same rules as creating the activity on screen (feature 006;
  domain model §3, R3, R5, R7, R9, R10; names unique in the initiative, including within the file) and
  list every problem at once as row, column, value found and reason, with the counts "N rows · N ready ·
  N with problems" and **Download problem list**.
- **FR-028**: Only missing Required values and invalid values MUST make a row a problem. Empty
  needed-to-submit columns are allowed (D-009); the check notes "42 activities will need more fields
  before submitting". Empty columns take the same defaults as creating on screen (field catalogue §4.1),
  and split values without shares are split evenly (R10).
- **FR-029**: Campaigns MUST be matched by ID or name within the plan, and initiatives by ID or name
  within that campaign, ignoring letter case and surrounding spaces; an ID from another plan is a problem.
- **FR-030**: With **Create missing campaigns and initiatives** on (off by default), unknown names MUST be
  created with on-screen defaults (owner and dates from the parent, R2) and listed in the check; with it
  off, they are problems.
- **FR-031**: Nothing MUST be created until the user confirms **Import N activities**; the user may
  instead upload a corrected file, which replaces the previous check.

**Import: results and undo**

- **FR-032**: Imported activities MUST start as Created, get readable IDs (D-019) and have no line items,
  so their whole planned cost is unassigned (domain model §4.3). Import MUST only add records, never
  change existing ones, and MUST NOT import line items.
- **FR-033**: The results page MUST show activities imported, campaigns and initiatives created, rows
  left out, rows that failed with reasons, a link to the imported activities, and **Undo import**. Each
  plan MUST keep a list of its imports (when, who, file name, counts), each with **Undo import** until
  undone.
- **FR-034**: **Undo import** MUST, after one confirmation stating the counts, move every record that import
  created — with anything added under them since — to the recycle bin in one step, and record who undid
  it and when.

### Key Entities *(include if feature involves data)*

- **Selection**: the records ticked in one list, with their levels; kept only while the user stays there.
- **Bulk action**: one run of an action on a selection — action, value or target, actor, time, and per
  record the outcome (changed, skipped or failed) with a reason.
- **Import template**: a file generated for one plan — columns, markings, allowed values, instructions.
- **Import**: one upload into a plan — file name, actor, time, rows read, problems (row, column, value,
  reason), records created, and whether and by whom it was undone.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a filtered list of 20 activities, a user changes the owner of all 20 in at most 6 steps
  (design system §10 step counting; one at a time this takes about 100 steps).
- **SC-002**: A user submits 20 ready activities in at most 4 steps, and every reason for a skipped
  activity is visible in the one summary — no retry is needed to discover a problem.
- **SC-003**: 100 % of skipped or failed records show a plain-language reason and a link; 0 records are
  left half-changed; 100 % of changed records have an activity-log entry naming the actor and action.
- **SC-004**: In a test with at least 5 first-time users, at least 90 % import a 50-row file containing 3
  errors without help within 10 minutes, seeing all 3 problems in a single check.
- **SC-005**: A template downloaded for a plan and filled only with its allowed values imports with 0
  problems in 100 % of test runs.
- **SC-006**: **Undo** after a bulk delete and **Undo import** each take 1 step and restore or remove 100 %
  of the affected records.
- **SC-007**: For 500 selected records or a 1,000-row file, the summary or check appears within 10 seconds.

## Assumptions

- Plans (handled one at a time on the Plans list) and line items (edited inside their activity) have no
  checkbox. Exporting selected rows belongs to feature 010. Hive9's "Change status" becomes **Submit for
  approval**; approving or declining many activities belongs to the approvals inbox (feature 008).
- Bulk moves follow exactly the single-move rules of features 005 and 006 (decision D-033). Moving keeps
  an activity's status unless its dates must change (domain model §2.5).
- **Line-item import is not part of this feature.** Hive9 imports activities only (FA §B8), and line items
  need currency, spread and finance coding that are quicker to add on screen; a later feature may add it.
- Owner changes, field changes, copies and moves have no Undo (the activity log shows before and after
  values); deletes and imports do. The 1,000-record and 1,000-row limits suit this phase's volumes.
- Shell's import prerequisite is kept even with "Create missing campaigns and initiatives", because the
  template lists the plan's campaigns and initiatives as allowed values.
- Hive9 bulk items for Finance (purchase requests), templates, linking and packaging are out of scope.
- No permissions in this phase (D-004). When roles arrive, each record will need the same permission as
  its single edit, and records without it will be skipped with the reason.
- Depends on: 010 (grid, filters, paging), 004–006 (plan tree, single-record rules, copy and move), 007
  (line items), 008 (submit rules), 003 (person picker), 002 (Settings), 001/011 (log, recycle bin).
