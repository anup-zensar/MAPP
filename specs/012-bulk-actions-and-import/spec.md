# Feature Specification: Bulk Actions and Excel Import of Plans

**Feature Branch**: `feature/012-bulk-actions-and-import`

**Created**: 2026-10-18 · **Re-baselined**: 2026-10-28 (constitution 4.6.0, D-333)

**Status**: Clarified (documented assumptions, see Clarifications) — building

**Input**: Baseline: "Select many rows: change owner/status fields, copy, move, delete; import activities from a
template" (P3, depends on 010). Re-baselined on Shell's spec 008 (*plan bulk import and export*, US1 and US3)
and RFP §8: "Flexibility to import the skeleton of plan and upload the updated plan using excels", "The tool
should allow users to have an excel integration or bulk plan uploading functionality".

**References**: Domain model [§1 hierarchy, §2.3 activity status, §2.5 re-approval, §3 required fields, §4.3
unassigned balance, §5 rules R2, R3, R5, R7, R9, R10, R13, R14](../../docs/product/domain-model.md) · Field
catalogue [§2–§5](../../docs/product/field-catalogue.md) · Design system [§5 components, §6.5 delete, §8
words, §10 usability targets](../../docs/product/design-system.md) · Glossary *Select* · Decisions D-004,
D-009, D-033, D-037, D-126–D-139, D-338–D-341 · Shell spec 008 FR-001–FR-010, FR-044–FR-051 · Shell as-is
document, prerequisite 2 · FA §B8 (bulk actions, import) · 014 Table A (permissions)

## Clarifications

### Session 2026-10-28

No product owner was available (the brief asks the agent to decide); each answer is a documented assumption
recorded in the decision log, with questions Q-020 and Q-021 for Shell.

- Q: The RFP asks to "import the skeleton of a plan and upload the updated plan". Is import only for new
  activities, as Hive9 does? → A: No. One spreadsheet layout, the **plan workbook**, holds a plan's campaigns,
  initiatives, activities and line items, one sheet per level. MAPP gives it empty (the template) or filled with
  the plan (the export). A row without an ID creates a record; a row with an ID updates that record (D-339,
  D-340). This supersedes the baseline's "import only adds activities, never line items".
- Q: Does an import ever delete or move records? → A: No. Rows that are missing from the file leave their
  records alone, and a row that names another parent than its record's own is a problem ("To move it, use
  Move to… in All work"). Deleting and moving stay deliberate, one confirmation each (D-340).
- Q: Shell's prerequisite 2 refuses the template for a plan without a campaign and an initiative. Does it
  still hold? → A: No, because the workbook can create campaigns and initiatives; the prerequisite existed
  because Hive9's template only held activities. Q-020 asks Shell to confirm (D-340).
- Q: Partial import or all or nothing? → A: Partial, as Shell's spec 008 FR-050 says: rows with problems are
  left out, and so are the rows below them in the file (an initiative whose campaign row has a problem). The
  check lists every problem at once; **Import** applies the ready rows in one step (D-340).
- Q: How does an update treat a Submitted or Approved activity? → A: As the single edit does, but without a
  question in the middle of an import: a row that would send it back for approval (planned cost, dates or
  Ongoing changed) is a problem "Approved — changing its planned cost or dates needs approval again; change it
  on screen". Other changes are applied (D-340).
- Q: What does **Undo import** undo? → A: The records the import created move to the recycle bin with
  anything added under them since; updates are not reverted (the activity log keeps the values before and
  after). A record that cannot be deleted (for example a line item on a purchase request) is kept and listed
  with the reason. No time limit (D-139, D-340).
- Q: Where can rows be selected? → A: In the All work grid, which shows one plan's tree when filtered to it.
  The plan page links to it; the plan tree itself gets no checkboxes in this feature (Q-021, D-338).
- Q: Who may run bulk actions and imports? → A: Each record changes only when the person may make that change
  on it alone (014 Table A in the record's plan scope); otherwise it is skipped with the reason. A record the
  person cannot see reads "No longer exists" (constitution VII). Template, check and import need *Edit
  activities* in the plan; the export needs *Export data* and a plan the person sees (D-341).
- Q: Copy of activities into another plan? → A: No: bulk copy and move follow the single-record rules of
  005 and 006 (D-033): activities copy and move within their plan, initiatives within their plan, campaigns
  copy to any plan and move to a plan of the same year (D-338).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Select many rows and change their owner or a field (Priority: P1)

Priya has 20 activities that move to a new owner. In **All work** she filters to them and ticks the header
checkbox; a bar appears at the bottom: "20 selected · 20 activities". She chooses **Change owner**, picks
Daniel Okoro and confirms; a summary says "20 changed". The same bar offers **Change field** to set one value —
for example *Company code = GB01* — on every selected record.

**Why this priority**: Shell users plan in spreadsheets partly because changing many records in Hive9 is slow
(PRISM). Selecting rows and changing them together is the base every other bulk action reuses.

**Independent Test**: In All work, select seeded activities, change their owner and their company code, and
check each record, its history and the results summary.

**Acceptance Scenarios**:

1. **Given** the All work grid, **Then** every campaign, initiative and activity row shows a checkbox at all
   times, and ticking a campaign selects only the campaign, never its children.
2. **Given** a filtered list showing 50 rows per page out of more, **When** the user ticks the header checkbox,
   **Then** the matching rows on the page are selected and the bar offers "Select all N matching".
3. **Given** 20 selected activities, **When** the user chooses **Change owner**, picks a person and confirms,
   **Then** all 20 get the new owner, the summary reads "20 changed", and each activity's history shows the
   change marked "Bulk change of 20 activities (BLK-…)".
4. **Given** 10 selected activities, 4 of them Approved, **When** the user sets a new end date, **Then**
   before confirming the dialog lists the 4 approved activities that would need approval again (domain model
   §2.5) and offers **Leave those unchanged** or **Change and restart approval**.
5. **Given** selected activities where some cannot take the value (a type their plan's model does not allow,
   R7), or that the person may not edit, **When** the user confirms, **Then** the others are changed and the
   summary reads "8 changed · 2 skipped" with each skipped activity's ID, name and reason.

---

### User Story 2 - Build a plan's skeleton from a spreadsheet (Priority: P1)

Mei has planned "FY2026 Mobility China" in a spreadsheet: 3 campaigns, 8 initiatives, 60 activities and their
line items. On the plan page she chooses **Import from Excel**, downloads the template made for the plan, fills
it in and uploads it. SP&A One lists every problem at once — "Activities · Row 14 · End date · 31 Feb 2026 ·
Not a real date" — and she imports the ready rows, with **Undo import** if needed.

**Why this priority**: RFP §8 asks to import the skeleton of a plan from Excel; Shell's users already plan in
spreadsheets (PRISM) and Hive9 offers an import (FA §B8).

**Independent Test**: Download the template of a seeded plan, fill campaigns, initiatives, activities and line
items (two rows wrong), upload, import the ready rows, find them in the plan, then undo the import.

**Acceptance Scenarios**:

1. **Given** a plan, **When** the user downloads the template, **Then** it holds one sheet per level
   (Campaigns, Initiatives, Activities, Line items) with one column per importable field marked Required,
   Needed to publish/submit or Optional, an **Allowed values** sheet (the plan's campaigns and initiatives,
   the types its model allows, active workflows, currencies, people, every option of every choice column) and
   an **Instructions** sheet.
2. **Given** an uploaded file with problems, **When** the check runs, **Then** every problem in every row is
   listed at once by sheet, row, column, value found and reason, and nothing has been created yet.
3. **Given** a checked file, **Then** the check shows per level how many records will be created, updated,
   left unchanged and left out, and **Import** applies exactly that in one step.
4. **Given** an initiative row whose campaign row has a problem, **Then** the initiative and everything below
   it in the file are left out with "Its campaign on Campaigns row 4 has problems".
5. **Given** a completed import, **When** the user chooses **Undo import** and confirms, **Then** every record
   that import created moves to the recycle bin and plan totals return to their earlier values.

---

### User Story 3 - Update a plan through its spreadsheet (Priority: P1)

Before the budget review Daniel exports "FY2026 Lubricants Europe" to Excel, changes the planned cost of
twelve activities, adds two line items and corrects three owners, and uploads the file. The check reads
"12 activities to update · 2 line items to create · 3 campaigns to update · 120 unchanged"; he imports it.

**Why this priority**: RFP §8 asks to "upload the updated plan using excels"; round trips through Excel are
how Shell's planners review plans today.

**Independent Test**: Export a seeded plan, upload it unchanged (nothing to change), change some values and
add rows, upload again and compare every record and its history.

**Acceptance Scenarios**:

1. **Given** a plan, **When** the user chooses **Export to Excel**, **Then** the file has the template's layout
   with one row per campaign, initiative, activity and line item the person sees, with IDs and values.
2. **Given** an exported file uploaded unchanged, **Then** the check reads "Nothing to change" and every row is
   unchanged.
3. **Given** a row with an ID and changed values, **Then** that record is updated with the same rules as
   editing it on screen; its history shows the change marked "Imported from '<file>' (IMP-…)".
4. **Given** a row that names another campaign than its initiative's own, or an ID of another plan, **Then**
   it is a problem and nothing is moved.
5. **Given** an Approved activity whose planned cost changed in the file, **Then** that row is a problem
   saying it needs approval again and must be changed on screen.

---

### User Story 4 - Submit many activities for approval (Priority: P2)

Before the approval deadline Priya selects 15 activities and chooses **Submit for approval**. The dialog says
"11 will be submitted · 4 are not ready". The summary lists the 4 with everything missing, and **Select failed
items** lets her fix and retry just those.

**Why this priority**: Every activity must be submitted; doing it one by one is one of Hive9's slowest tasks
(design system §10).

**Independent Test**: Select seeded activities in mixed statuses, submit, and compare the summary with each
activity's completeness.

**Acceptance Scenarios**:

1. **Given** selected activities that are Created or Declined and complete, **When** the user submits,
   **Then** each goes to Submitted under feature 008's rules and its approvers are notified.
2. **Given** a selected activity missing fields, **Then** it is skipped with all reasons at once.
3. **Given** a selection that also contains campaigns or already-approved activities, **Then** they are
   skipped with "Only activities go through approval" or "Already approved"; **Select failed items** then
   selects exactly the skipped and failed records.

---

### User Story 5 - Copy, move or delete many records (Priority: P2)

Daniel reuses last year's event activities: he selects 6 and chooses **Copy to…**, picks the initiative
"Rimula fleet › Events", and gets 6 copies in Created status. He moves 3 initiatives into another campaign with
**Move to…**, then deletes 2 initiatives he no longer needs with one confirmation, and undoes the delete.

**Why this priority**: Reusing proven work and tidying a plan are frequent. Single-record copy, move and delete
exist (features 004–006); this extends them to many rows, safely.

**Independent Test**: Copy seeded activities into another initiative, move initiatives to another campaign,
delete a selection and Undo; verify names, statuses, IDs, the recycle bin and totals.

**Acceptance Scenarios**:

1. **Given** selected activities, **When** the user copies them into an initiative, **Then** each copy includes
   its line items, starts as Created, and is named as the single copy names it (R13).
2. **Given** a selection mixing campaigns and activities, **When** the user chooses **Copy to…**, **Then** it
   explains "Copy works on one level at a time" and offers **Keep only activities** or **Keep only campaigns**.
3. **Given** campaigns moved to a plan whose model or business unit differs, **Then** before confirming the
   dialog lists the effects with counts (FR-018).
4. **Given** 2 initiatives and 5 activities selected, **When** the user chooses **Delete**, **Then** one dialog
   says what goes with them, how many are submitted or approved, that plan totals update and that they move to
   the recycle bin, with **Cancel** focused and a red **Delete 7 records** button; an activity under a selected
   initiative is counted once.
5. **Given** a bulk delete has just happened, **When** the user chooses **Undo**, **Then** every deleted record
   comes back with its original IDs and values.

---

### Edge Cases

- A selected record is changed or deleted by someone else before the action runs: it is processed against its
  latest values; one that no longer exists is reported as failed ("No longer exists").
- The same name appears twice for one parent in the file (two activities called "Trade fair" in one
  initiative), or a new row's name already exists there: both are problems.
- An option was deactivated after the template was downloaded: the file is checked against current Settings and
  the value is reported as no longer allowed.
- A workbook saved as .csv holds one sheet: SP&A One reads it as the Activities sheet.
- Undo import after imported records were edited or had records added under them: the confirmation says how
  many records go and which are kept, with the reason.

## Requirements *(mandatory)*

### Functional Requirements

**Selecting rows**

- **FR-001**: Every campaign, initiative and activity row of the All work grid (feature 010) MUST show a
  checkbox at all times (no separate selection mode); selecting a row MUST select only that record.
- **FR-002**: The header checkbox MUST select every matching selectable row on the page (D-134); the bar MUST
  then offer "Select all N matching" (every matching record of the current search and filters) and **Clear**.
- **FR-003**: The selection MUST be kept while paging and expanding or collapsing rows, and MUST be cleared,
  with a short message, when the year, search or filters change, and after an action completes.
- **FR-004**: While anything is selected, a bar MUST stay visible under the list showing the count by level
  ("12 selected · 2 campaigns, 10 activities") and the actions **Change owner**, **Change field**, **Submit
  for approval**, **Copy to…**, **Move to…**, **Delete** and **Clear**.
- **FR-005**: Selection MUST work by keyboard (Space toggles a checkbox; Shift with a click or Space selects
  the range from the last ticked row); each checkbox MUST have an accessible name ("Select activity 'Launch
  film'"); the ticked box, not colour alone, shows the selection.
- **FR-006**: One bulk action MUST handle up to 1,000 records; a larger selection is refused with "Bulk actions
  handle up to 1,000 records at a time — narrow the filters".
- **FR-007**: **Change field**, **Copy to…** and **Move to…** need one level; with a mixed selection they MUST
  explain why and offer **Keep only <level>**. Other actions MUST run for the records they apply to and list the
  rest as skipped.

**Change owner and change field**

- **FR-008**: **Change owner** MUST set one active person (person picker, feature 003) as owner of every
  selected campaign, initiative and activity; line items keep their own owners.
- **FR-009**: **Change field** MUST offer start date and end date; for activities the activity type and the
  approval workflow; and the level's active yes/no, single-choice and multiple-choice Shell attributes (chosen
  values replace the old ones), plus **Clear the value** for fields that are not Required. It MUST NOT offer
  names, IDs, money, calculated fields, Ongoing (R6), text, number or % split fields; the dialog says these are
  edited one record at a time.
- **FR-010**: Each record MUST be changed under the same rules as editing it alone (features 004–008); a record
  that the single edit would refuse MUST be skipped with that edit's reasons. Changing an activity's type MUST
  NOT change its planned cost.
- **FR-011**: The confirmation MUST state the change and count ("Set Company code to 'GB01' on 20 activities")
  and list Submitted or Approved activities whose change needs approval again (domain model §2.5), offering
  **Leave those unchanged** (reported as skipped) or **Change and restart approval**.

**Submit for approval**

- **FR-012**: **Submit for approval** MUST submit every selected Created or Declined activity that meets
  feature 008's submit rules, notifying approvers as 008 does.
- **FR-013**: Before confirming, the dialog MUST show how many will be submitted and how many skipped; each
  skipped activity MUST list all its reasons at once with a link. Campaigns and initiatives are skipped with
  "Only activities go through approval".

**Copy to, move to and delete**

- **FR-014**: **Copy to…** MUST copy every selected record, with everything below it, into one target of the
  right level, following the single copy (R13, D-033): campaigns into any plan, initiatives into a campaign of
  their plan, activities into an initiative of their plan; copies start Created.
- **FR-015**: **Move to…** MUST follow the single moves (D-033): campaigns to another plan of the same year,
  initiatives to another campaign of their plan, activities to another initiative of their plan — keeping IDs,
  status, line items and history.
- **FR-016**: The target picker MUST be searchable, show the full path (Plan › Campaign › Initiative), offer
  only targets the single action allows for the selection, and leave out the selected records and anything
  below them; when the selection spans plans (or years, for campaign moves) it MUST say so and offer none.
- **FR-017**: A copied or moved record that the single action refuses there (dates outside the target, R3; a
  name already taken by a move; already in the target) MUST be skipped with the reason; dates are never changed
  silently.
- **FR-018**: When campaigns are moved or copied into a plan with another model or business unit, the
  confirmation MUST list with counts the activities whose type the target model does not allow (kept, to change
  before submitting), the attribute groups hidden (values kept) and shown, and that both plans' totals update.
- **FR-019**: **Delete** MUST ask once, following design system §6.5: counts by level of the selected records,
  what goes with them, how many are Submitted or Approved, the effect on totals, and that everything moves to
  the recycle bin. A record under another selected record is counted once.
- **FR-020**: After a bulk delete, the message MUST offer **Undo** for at least 10 seconds, restoring all of
  them in one step; each top-level deleted record MUST also be its own recycle-bin entry (R14).

**Results, access and history**

- **FR-021**: Every bulk action MUST end with a summary such as "33 changed · 2 skipped · 1 failed" (verb per
  action: changed, submitted, copied, moved, deleted). *Skipped* means a rule or a permission prevents the
  change; *failed* means the record no longer exists. Each skipped or failed record MUST show its ID, name,
  reason and a link, and **Select failed items** MUST make them the new selection.
- **FR-022**: Records MUST be processed independently: successful changes are kept when others are skipped or
  fail, and a single record is never left half-changed.
- **FR-023**: Every record changed, created or deleted by a bulk action or an import MUST get its own
  activity-log entry naming the actor and the bulk action or import it was part of.
- **FR-024**: A bulk action or an import MUST change a record only when the person holds the permission the
  single change needs on it (014 Table A, in the record's plan scope); a record they can see but may not change
  is skipped with the reason; a record they cannot see is reported exactly like a missing one.

**Plan workbook: template and export**

- **FR-025**: The **plan workbook** MUST have the sheets Campaigns, Initiatives, Activities and Line items, then
  Allowed values and Instructions. On each level sheet row 1 holds the headings, row 2 each column's marking
  (Required, Needed to publish, Needed to submit, Optional; the ID column "Empty for a new record"), and data
  starts on row 3.
- **FR-026**: The level sheets MUST hold these columns, then one column per active Shell attribute of the
  level that the plan's business unit uses (R9; types yes/no, single choice, multiple choice, multiple choice
  with % split, text, long text, number): *Campaigns* — ID, Campaign name, Owner (e-mail), Start date, End
  date, Budget, Description; *Initiatives* — ID, Campaign, Initiative name, Owner, Start date, End date, Budget,
  Description; *Activities* — ID, Campaign, Initiative, Activity name, External name, Activity type, Owner,
  Start date, End date, Ongoing, Planned cost, Approval workflow, Description; *Line items* — ID, Campaign,
  Initiative, Activity, Line item name, Owner, Currency, Planned cost, Description. Budgets and activity costs
  are in the plan currency.
- **FR-027**: **Download template** (plan page) MUST give the workbook without data rows; **Export to Excel**
  MUST give it with one row per record of the plan the person sees, written exactly as the import reads it, so
  that the exported file uploaded unchanged changes nothing.

**Plan workbook: upload, check and import**

- **FR-028**: Upload MUST accept one .xlsx workbook (any of the level sheets; missing sheets mean nothing for
  that level) or one .csv file (read as the Activities sheet), up to 5 MB and 5,000 data rows. Columns are
  matched by heading in any order and letter case; unrecognised columns are listed and ignored; a missing
  Required column of a sheet that has data stops the check with "The Activities sheet has no 'Activity type'
  column — download a fresh template".
- **FR-029**: A row without an ID MUST create a record; a row with an ID MUST update that record of this plan
  with the row's values in the columns present (an empty cell clears an optional value). An import MUST NOT
  delete or move records: a row naming another parent than its record's own, or an ID of another plan or one
  the person cannot see, is a problem.
- **FR-030**: Parents MUST be found by ID or by name within the plan (ignoring letter case and surrounding
  spaces), including records the same file creates on an earlier sheet.
- **FR-031**: The check MUST apply the same rules as creating or editing the record on screen (features
  005–007; domain model §3, R3, R5, R7, R9, R10; names unique under their parent, including within the file) and
  list every problem at once as sheet, row, column, value found and reason, with per level the counts to
  create, to update, unchanged and left out, and **Download problem list**.
- **FR-032**: Only missing Required values and invalid values make a row a problem; empty needed-to-publish or
  needed-to-submit values are allowed (D-009) and counted ("42 records will need more fields before submitting
  or publishing"). Empty cells of a new record take the on-screen defaults (field catalogue §2–§5). A row that
  would send a Submitted or Approved activity back for approval is a problem (D-340).
- **FR-033**: A row whose parent row in the file has problems MUST be left out with "Its <level> on <sheet>
  row <n> has problems".
- **FR-034**: Nothing MUST change until the user confirms **Import**; **Import** MUST apply the ready rows of
  the same check in one step, new records starting as on screen (activities Created); the user may instead
  upload a corrected file, which replaces the check.

**Import results and undo**

- **FR-035**: The results page MUST show records created and updated per level, rows left out with reasons,
  links to the plan, and **Undo import**. Each plan MUST keep a list of its imports (when, who, file name,
  counts), each with **Undo import** until undone.
- **FR-036**: **Undo import** MUST, after one confirmation stating the counts, move every record the import
  created — with anything added under them since — to the recycle bin in one step, keep and list any record
  that cannot be deleted with the reason, count records already deleted, and record who undid it and when.
  Updates made by the import are not reverted.

### Key Entities *(include if feature involves data)*

- **Selection**: the records ticked in one list, with their levels; kept only while the user stays there.
- **Bulk action**: one run of an action on a selection — action, actor, time, its note, and for a delete the
  recycle-bin entries Undo restores.
- **Plan workbook**: a spreadsheet generated for one plan — level sheets, markings, allowed values,
  instructions; empty (template) or filled (export).
- **Import**: one upload into a plan — file name, actor, time, rows read, counts created and updated per level,
  rows left out with their problems, the records created and updated, and whether and by whom it was undone.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: From a filtered list of 20 activities, a user changes the owner of all 20 in at most 6 steps
  (design system §10 step counting; one at a time this takes about 100 steps).
- **SC-002**: A user submits 20 ready activities in at most 4 steps, and every reason for a skipped activity is
  visible in the one summary.
- **SC-003**: 100 % of skipped or failed records show a plain-language reason; 0 records are left half-changed;
  100 % of changed records have an activity-log entry naming the actor and the action.
- **SC-004**: A template filled only with its allowed values imports with 0 problems, and an export uploaded
  unchanged changes 0 records, in 100 % of test runs.
- **SC-005**: Every problem of a file is reported in a single check — 100 % of seeded errors appear with sheet,
  row and column.
- **SC-006**: **Undo** after a bulk delete and **Undo import** each take 1 step and restore or remove 100 % of
  the affected records that can be.
- **SC-007**: For 500 selected records or a 1,000-row file, the summary or check appears within 10 seconds.

## Assumptions

- Plans (handled one at a time on the Plans list) and line items (edited inside their activity) have no
  checkbox. Exporting selected rows belongs to feature 010. Hive9's "Change status" becomes **Submit for
  approval**; approving or declining many activities belongs to the approvals inbox (feature 008).
- The plan itself (name, year, model, business unit, currency, teams) is created on screen; the workbook fills
  and updates what is below it. Goals, line-item spreads, finance-budget links, contracted amounts and SWBS
  requests are edited on screen; new line items take the on-screen defaults for them.
- Owner changes, field changes, copies, moves and import updates have no Undo (the activity log shows before
  and after values); deletes and imports have (D-130, D-139).
- The 1,000-record limit per action and the 5 MB / 5,000-row limit per file suit this phase's volumes.
- Hive9 bulk items for Finance (purchase requests), templates, linking and packaging are out of scope.
- Depends on: 010 (grid, filters, paging), 004–007 (records, single-record rules, copy and move), 008 (submit
  rules), 003 (person picker), 002 (Settings), 011 (log, recycle bin), 014 (permissions and scopes).
