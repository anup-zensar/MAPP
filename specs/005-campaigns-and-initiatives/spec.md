# Feature Specification: Campaigns and Initiatives

**Feature Branch**: `005-campaigns-and-initiatives`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Campaigns and initiatives inside a plan; budgets flow down, costs roll up; move, copy, delete"

**References**: Domain model [§1 hierarchy, §2.2 derived statuses, §2.6 completeness, §3 required fields, §4.1–4.2 money, §4.5 goals, §5 rules R2, R3, R4, R7, R8, R9, R13, R14](../../docs/product/domain-model.md) · Field catalogue [§2 Campaign, §3 Initiative](../../docs/product/field-catalogue.md) · Design system [§4 colours, §5 tree table and side panel, §6 CRUD pattern, §7 forms, §10 targets](../../docs/product/design-system.md) · Glossary [planning, money](../../docs/product/glossary.md) · Demo data [planning seed](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-012, D-015, D-017 · Hive9 as-is steps 2–3; functional analysis A3, B4, B5

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Add campaigns to a plan (Priority: P1)

**Add campaign** on the plan page opens a side panel with the plan already chosen, owner and dates taken
from the plan, the remaining plan budget shown while a budget is typed, and a cluster that fills in its
region.

**Why this priority**: Campaigns are a plan's first level and hold the cluster and region needed to publish.

**Independent Test**: In FY2026 Mobility China, add "Fleet growth" with cluster Fleet – China and a
budget; check the region is filled in, the row appears and the plan's remaining budget changes.

**Acceptance Scenarios**:

1. **Given** the plan page, **When** the user chooses **Add campaign**, **Then** "New campaign — Inside
   plan '<plan>'" opens with Owner = the plan's owner ("From the plan") and dates 1 Jan – 31 Dec of the
   plan's year ("The plan's year").
2. **Given** no cluster, **Then** Region is disabled with "Choose a cluster first"; **When** the user picks
   "Retail – China", which has one region, **Then** Region becomes "RETAIL – CHINA" with "Region set to
   'RETAIL – CHINA', the only region for this cluster" (R8).
3. **Given** a plan budget of 800,000 USD with 600,000 USD given to campaigns, **When** the user types
   300,000, **Then** the remaining plan budget shows "−100,000.00 USD" in red with "Over-allocated by
   100,000.00 USD", and saving is still allowed (D-015).
4. **Given** a name, **When** the user chooses **Save**, **Then** the campaign gets a new ID, its row
   appears highlighted and a toast says "Campaign '<name>' created — View"; **Save and add another**
   instead saves and opens a fresh panel for the same plan.
5. **Given** a name already used in this plan and an end date before the start date, **When** the user
   saves, **Then** both problems appear at once; **Given** an end date of 31 Mar 2027 in a 2026 plan,
   **Then** the panel warns "This campaign runs outside 2026, the plan's year" and saving is allowed (R4).

---

### User Story 2 - Add initiatives to a campaign (Priority: P1)

**Add initiative** on a campaign row, or in a campaign's details, opens a side panel with the campaign
fixed and owner and dates taken from it; a name and **Save** are enough.

**Why this priority**: Initiatives group the activities; no activity can be added without one.

**Independent Test**: Under "Helix launch", add "Digital & social" in three steps; then try dates outside
the campaign and a budget larger than what remains.

**Acceptance Scenarios**:

1. **Given** a campaign row, **When** the user chooses **Add initiative**, **Then** the panel shows
   Campaign read-only ("Change it with Move") and Owner and dates from the campaign; after a name and
   **Save**, the initiative appears highlighted under its expanded campaign.
2. **Given** campaign dates 1 Mar – 31 Dec 2026, **When** the user saves a start date of 1 Feb 2026,
   **Then** Save reports "Dates must be inside the campaign's dates (1 Mar 2026 – 31 Dec 2026)", the range
   the date picker also shows (R3).
3. **Given** a campaign budget of 400,000 USD with 300,000 USD given to initiatives, **When** the user
   types 150,000, **Then** the remaining campaign budget shows "−50,000.00 USD" in red; saving is allowed.

---

### User Story 3 - View and edit a campaign or initiative (Priority: P1)

Selecting a campaign or initiative opens its details in a side panel; **Edit** turns the panel into the
form, and saving shows every problem at once.

**Why this priority**: Campaigns and initiatives are checked and adjusted far more often than created.

**Independent Test**: Open "Helix launch", change its owner, budget, dates and clusters; check the
messages, the panel, the row and the plan totals.

**Acceptance Scenarios**:

1. **Given** the plan tree, **When** the user selects a campaign, **Then** its details panel opens (not
   the form) with "Plans › <plan> › <campaign>", the CAMPAIGN tag, Approval progress, Timing and
   completeness chips, ID, owner, dates, money, goals, Shell attributes, initiatives, **Edit** and
   **⋯ More** (Add initiative, Copy, Move to…, View history, Delete).
2. **Given** the user changes budget and owner and saves, **Then** the panel, the row, the plan's summary
   cards and remaining budgets update at once, and the log holds old and new values.
3. **Given** initiative "Digital & social" runs 1 Feb – 30 Nov 2026, **When** the user saves campaign
   dates of 1 Mar – 31 Dec 2026, **Then** the problem names "Digital & social (1 Feb – 30 Nov 2026)" as
   outside them and nothing is saved (R3).
4. **Given** clusters Retail – China and Fleet – China, **When** the user removes Fleet – China, **Then**
   its regions are removed with a note naming them; **When** the user then closes the panel, **Then**
   they are asked **Keep editing** or **Discard changes**.

---

### User Story 4 - See budget, cost and progress in the plan tree (Priority: P1)

Campaign and initiative rows show level, owner, dates, Approval progress, Timing, completeness, budget
with what remains, planned cost and budget used; a totals row sums the plan.

**Why this priority**: The tree is where planners see at a glance whether money and work line up.

**Independent Test**: In FY2026 Lubricants Europe, check every row and the totals row against
hand-calculated values, then change one budget and watch every affected number update.

**Acceptance Scenarios**:

1. **Given** initiatives with planned costs of 100,000 and 50,000 USD under a campaign budget of
   120,000 USD, **Then** the campaign shows planned cost 150,000.00 USD ("Calculated") and budget used
   125 % with a red bar, and the plan's planned cost includes it (domain model §4.2).
2. **Given** 5 activities of which 3 are Approved, In progress or Complete, **Then** Approval progress
   reads "In progress · 3 of 5 activities approved" (or "0 of 5" before any approval); with all 5, "All
   approved"; with no activities below, "No activities".
3. **Given** today is before, within or after a campaign's dates, **Then** Timing reads Not started,
   Running or Finished.
4. **Given** a campaign without cluster and region, **Then** its completeness chip reads "2 fields needed"
   with links, and the plan's chip shows "1 item below needs attention".
5. **Given** the tree, **Then** the totals row reads "Total · 2 campaigns · 4 initiatives · 7 activities"
   with the sum of campaign budgets ("Allocated to campaigns") and the plan's planned cost.

---

### User Story 5 - Delete a campaign or initiative and undo it (Priority: P2)

**Delete** says what goes with the record and that it moves to the recycle bin; Undo brings it back.

**Why this priority**: Mistakes must be removable safely, but planning can start without deleting.

**Independent Test**: Delete "Helix launch", check the counts, choose Undo and confirm everything is back
with the same IDs and the plan totals restored.

**Acceptance Scenarios**:

1. **Given** "Helix launch" holds 3 initiatives, 12 activities (2 approved) and 20 line items, **When**
   the user chooses **Delete**, **Then** "Delete campaign 'Helix launch'?" names those counts, says "Plan
   totals will update", that approved activities restored later return as Created, and that it moves to
   the recycle bin; **Cancel** has focus and the red button says **Delete campaign**.
2. **Given** the user confirms, **Then** the panel closes, the rows disappear, totals update and the toast
   offers **Undo** for at least 10 seconds; Undo restores everything exactly as it was.

---

### User Story 6 - Move or copy a campaign or initiative (Priority: P2)

**Move to…** puts an initiative under another campaign of the same plan, or a campaign under another plan
of the same year, with everything below it. **Copy** duplicates a record with everything below it.

**Why this priority**: Plans get reorganised and campaigns repeat, but neither is needed for a first plan.

**Independent Test**: Move "Digital & social" from "Helix launch" to "Rimula fleet" and check both
campaigns' totals; copy "Helix launch" and compare the copy with the original.

**Acceptance Scenarios**:

1. **Given** "Digital & social" in "Helix launch", **When** the user moves it to "Rimula fleet", **Then**
   it moves with its activities and line items, keeps its ID, dates and statuses, and both campaigns'
   planned cost and remaining budgets update.
2. **Given** a campaign whose dates do not contain the initiative's dates, or that already has an
   initiative with that name, **Then** it is listed but cannot be chosen, with the reason and the fix, e.g.
   "Dates don't fit (1 Mar – 31 Dec 2026) — change the initiative's dates first".
3. **Given** a campaign, **When** the user chooses **Move to…**, **Then** only other plans of the same year
   are offered, with "Campaigns move only between plans of the same year"; a plan with another model or
   business unit shows the consequences before the move is confirmed.
4. **Given** "Helix launch", **When** the user chooses **Copy**, **Then** a dialog offers "Helix launch
   (copy)" (editable; "(copy 2)" if taken), the destination (this plan by default) and the counts of what
   will be copied, and warns if the copy's budget would over-allocate the destination.
5. **Given** the user confirms the copy, **Then** a new campaign with all its initiatives, activities and
   line items appears highlighted, with new IDs, activities Created and no approvals, comments or history
   (R13).

---

### Edge Cases

- A campaign without dates: its initiatives' dates are not checked against it, Timing shows "—" with
  "Timing appears once dates are set", and the missing dates count as needed to publish.
- A parent without a budget: remaining budget shows "—" with "No budget to allocate from".
- The chosen clusters have no regions set up: Region says so and is not counted as missing, so a field
  nobody can fill never blocks publishing.
- An option, person or cluster becomes inactive: the value is kept and labelled "(inactive)".
- Ongoing activities have no end date: only their start date is checked when an initiative's dates change.
- No possible destination exists: **Move to…** says so instead of showing an empty list.
- Two people add the same name under the same parent at once: the second gets the duplicate-name message.
- A copy name would exceed 120 characters: the original part is shortened so the suffix fits.

## Requirements *(mandatory)*

### Functional Requirements

**Adding campaigns and initiatives**

- **FR-001**: **Add campaign** MUST be offered in the plan tree's toolbar and on the plan row, and **Add
  initiative** on each campaign row and in each campaign's details; each opens a side panel (design system
  §6.2) naming the parent, with the footer **Save**, **Save and add another** (saves, then opens an empty
  panel for the same parent) and **Cancel** (asks before discarding typed values).
- **FR-002**: The campaign panel MUST show the fields of field catalogue §2 (#1–#12) and the initiative
  panel those of §3 (#1–#12), in the sections Basics, Budget and Shell attributes, with every other active
  attribute of that level from Settings (feature 002) in display order and markings shown as words; the
  initiative's campaign is read-only with "Change it with Move".
- **FR-003**: The panels MUST pre-fill and label: owner = the parent's owner (R2); campaign dates = 1 Jan
  – 31 Dec of the plan's year; initiative dates = the campaign's dates (empty if it has none); yes/no
  attributes = No.
- **FR-004**: A saved record MUST get a permanent ID (C-… or I-…), appear highlighted under its expanded
  parent, and be announced by a toast with **View**.

**Rules checked on save**

- **FR-005**: Saving MUST require only the name, and MUST check: name of at most 120 characters, unique
  under the same parent ignoring letter case and surrounding spaces; description of at most 2,000
  characters; budget not negative; start on or before end; both dates or neither. Every problem is
  reported in one attempt (design system §6.4).
- **FR-006**: An initiative's dates MUST lie inside its campaign's dates whenever the campaign has both
  dates (R3); the date picker shows the allowed range and the message names it.
- **FR-007**: Saving campaign dates that would leave initiatives outside them, or initiative dates that
  would leave activities outside them (only the start date for ongoing activities), MUST be reported as a
  problem naming each such record with its dates.
- **FR-008**: Campaign dates before 1 Jan or after 31 Dec of the plan's year MUST show a warning in the
  panel and on the row, and MUST NOT block saving or publishing (R4).
- **FR-009**: Region MUST stay disabled with the reason until a cluster is chosen, offer only the regions
  of the chosen clusters, add a chosen cluster's only region automatically saying so, remove a removed
  cluster's regions with a note naming them (R8), and not count as missing when the chosen clusters have
  no regions set up.

**Budget and cost**

- **FR-010**: The campaign panel MUST show the remaining plan budget and the initiative panel the remaining
  campaign budget (domain model §4.1), including the amount being typed; campaign details also show what
  remains for its initiatives. A negative remaining budget is shown in red with "Over-allocated by X" and
  never blocks saving (D-015); without a parent budget, remaining shows "—" with the reason.
- **FR-011**: Planned cost MUST be the read-only roll-up of domain model §4.2, labelled "Calculated" with
  an ⓘ formula, "—" when nothing below has a planned cost; budget used = planned cost ÷ budget as a bar,
  red above 100 %, "—" without a budget.
- **FR-012**: Every saved change to a budget, or to a planned cost below, MUST update all affected rows,
  panels, plan summary cards and the totals row at once, without reloading.

**Statuses and completeness**

- **FR-013**: Campaigns and initiatives MUST show **Approval progress** (domain model §2.2) — No
  activities · In progress with "n of m activities approved" · All approved, counting Approved, In
  progress and Complete activities as approved — and **Timing** (Not started, Running, Finished against
  today's date, or "—" with the reason while dates are missing).
- **FR-014**: They MUST show a completeness chip (domain model §2.6) counting their own needed-to-publish
  fields (shown attributes only), each linking to the field, plus "n items below need attention"; this
  completeness feeds the plan's publish check (feature 004).
- **FR-015**: These statuses MUST be recalculated whenever something below changes and shown as a word
  with a colour (design system §4) on rows and in panels.

**Viewing and editing**

- **FR-016**: Selecting a campaign or initiative MUST open its details panel, never the form, with the
  header of design system §6.3 (breadcrumb, level tag, status and completeness chips, ID, owner, dates,
  created and last changed by and on), budget and cost, goals (sum of the activities' INQ, MQL and
  Revenue, domain model §4.5), Shell attributes ("—" when empty), children with counts, **Edit** (primary)
  and **⋯ More** (Add initiative for campaigns, Copy, Move to…, View history, Delete).
- **FR-017**: Each campaign and initiative MUST have its own address that opens the plan page with its
  details panel open.
- **FR-018**: **Edit** MUST turn the panel into the form; Save is never disabled, every problem is shown in
  one attempt, values a change will clear are listed before saving, and leaving with unsaved changes asks
  **Keep editing** or **Discard changes**. Editing is allowed whatever the plan's status.

**Rows in the plan tree**

- **FR-019**: The plan tree (feature 004) MUST show campaign rows under the plan row and initiative rows
  under their campaign, each with the level bar and tag (design system §4), the name as a link, the number
  of children, owner, dates, Approval progress, Timing and completeness chips, budget with what remains
  for its children beneath (red when over-allocated), planned cost and a budget-used bar.
- **FR-020**: Rows MUST be ordered by start date, then name, records without dates last; plan and campaign
  rows start expanded; the tree's search, **Expand all** and **Collapse all** include these rows.
- **FR-021**: Row actions MUST be reachable on hover, by keyboard and in a **⋯** menu: campaign — Add
  initiative, View, Edit, Copy, Move to…, Delete; initiative — View, Edit, Copy, Move to…, Delete
  (feature 006 adds Add activity).
- **FR-022**: A totals row MUST show the numbers of campaigns, initiatives and activities, the sum of
  campaign budgets labelled "Allocated to campaigns", and the plan's planned cost.

**Move and copy**

- **FR-023**: **Move to…** MUST list, for an initiative, the other campaigns of the same plan with their
  dates and, for a campaign, the other plans of the same year (Draft or Published). A destination that
  already has a child with the same name, or a campaign whose dates do not contain the initiative's
  dates, is listed but not selectable, with the reason and the fix; other years are not offered, and the
  dialog says why (D-017). A moved campaign keeps the R4 warning if its dates leave the year.
- **FR-024**: When a campaign's target plan has another model or business unit, the dialog MUST list,
  before confirming, the activities whose type the target model does not allow (as feature 004 does for a
  model change), say that goals will be recalculated, and say which attribute groups will be hidden or
  shown with their values kept (R9).
- **FR-025**: A move MUST take everything below the record, keep IDs, dates, values and statuses
  (including activity approvals), update the roll-ups and remaining budgets of the old and new parents
  (negative allowed with the warning), and confirm with a toast with **View**.
- **FR-026**: **Copy** MUST open a dialog with the name pre-filled per R13 and editable (checked for
  uniqueness in the destination), the destination (the current parent by default, or another under the
  rules of FR-023), the counts of everything that will be copied, and a warning when the copy's budget
  will over-allocate the destination.
- **FR-027**: The copy MUST include everything below the record per R13 (new IDs, activities Created, no
  approvals, comments, history or GSAP values), be made completely or not at all, and appear highlighted.

**Delete**

- **FR-028**: Delete MUST follow design system §6.5: the dialog names the counts of initiatives (for a
  campaign), activities and line items, says "Plan totals will update" and that approved activities
  restored later return as Created (R14), and moves everything to the recycle bin.
- **FR-029**: After deleting, the panel MUST close and a toast offer **Undo** for at least 10 seconds;
  Undo restores the record and everything below it exactly as it was (feature 001).

**History and later rules**

- **FR-030**: Every create, edit, move (with old and new parent), copy and delete MUST be logged with the
  actor (feature 001); **View history** opens the record's history (feature 011).
- **FR-031**: NorthStar MUST NOT restrict these actions by role (D-004); Shell's role rules for creating,
  editing, moving and deleting campaigns and initiatives are enforced when roles arrive.

### Key Entities *(include if feature involves data)*

- **Campaign**: fields per field catalogue §2, readable ID, exactly one plan, created and last changed by
  and on.
- **Initiative**: fields per field catalogue §3, readable ID, exactly one campaign.
- **Attribute value**: a campaign's or initiative's value for a Shell attribute; kept while hidden.
- **Derived values** (never typed): remaining budgets, planned cost, budget used, Approval progress,
  Timing, completeness and goal sums.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user adds a campaign in at most 5 steps and an initiative in at most 3 steps
  from the plan page, using the defaults (design system §10).
- **SC-002**: In a test with at least 5 first-time users, at least 90 % create, view, edit, move, copy and
  delete a campaign and an initiative without help, each task within 2 minutes.
- **SC-003**: 100 % of problems with a campaign or initiative form are shown in a single save attempt, and
  100 % of read-only fields and unavailable choices (such as move destinations) say why.
- **SC-004**: Planned costs, remaining budgets and budget used equal hand-calculated values to the cent in
  100 % of test cases and appear on every affected row, panel and card straight after saving.
- **SC-005**: From the plan tree alone, test users correctly state each campaign's and initiative's
  approval progress, timing, completeness and budget position in at least 90 % of questions.
- **SC-006**: 100 % of moves and copies keep every record and value below them; copies follow R13.
- **SC-007**: 100 % of deletes can be undone within 10 seconds with nothing lost.
- **SC-008**: A plan with 20 campaigns and 100 initiatives expands fully, and shows updated totals after a
  save, in under 1 second.

## Assumptions

- Builds on feature 004 (plan page, plan tree, summary cards, publish check), 002 (Cluster → Region
  dependency, campaign and initiative attributes), 003 (person picker, display currency) and 001 (side
  panel, toasts, activity log, recycle bin, Undo). Feature 006 adds activity rows, **Add activity** and
  activity statuses; until then planned cost shows "—" and Approval progress "No activities".
- Moves and copies never change dates: a destination whose dates do not fit is not offered, so nothing
  changes silently and no activity needs re-approval; a move is not a re-approval change (§2.5).
- A campaign or initiative copy always includes everything below it (R13); only plan copy (feature 004)
  offers to leave children out.
- Destinations stay within the same plan (initiatives) or year (campaigns); multi-year work is out of
  scope (D-017) and moving many records at once belongs to feature 012.
- Campaigns and initiatives have no approval of their own (D-012).
- Undo straight after a delete restores exactly (feature 001); a later restore follows R14 (feature 011).
