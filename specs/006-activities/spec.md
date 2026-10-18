# Feature Specification: Activities

**Feature Branch**: `006-activities`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Activities: basics, dates, planned cost, Shell attributes with business-unit groups and % splits, goals, status, completeness"

**References**: Domain model [§1 hierarchy, §2.3 activity status, §2.5 re-approval, §2.6 completeness, §3 required fields, §4.2–4.3 planned cost and unassigned balance, §4.5 goals, §4.6 from finance, §5 rules R3, R5–R7, R9, R10, R13, R14, §6 inactive values](../../docs/product/domain-model.md) · Field catalogue [§4 activity](../../docs/product/field-catalogue.md) · Design system [§4 status colours, §5 split editor, §6 CRUD pattern, §7 forms, §10 usability targets](../../docs/product/design-system.md) · Glossary [planning levels, goals, statuses](../../docs/product/glossary.md) · Decisions D-004, D-009, D-013, D-016, D-017, D-019 · Shell Plan Module doc step 4 · Hive9 functional analysis §B6, §B9–B10 · ZenReach spec §7.4, §8.10 (layout and interaction only)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create an activity (Priority: P1)

On an initiative's row in the plan tree, the user chooses **Add activity**. A full page opens with the
campaign and initiative filled in. Only the name and type are needed to save; owner and dates come from
the initiative, planned cost from the type. After saving, the user lands on the new activity's page,
ready to add its line items (feature 007).

**Why this priority**: The activity carries planned cost and goes through approval; line items, approvals
and every planning view depend on it.

**Independent Test**: In the seeded plan "FY2026 Lubricants Europe", add an activity under "Digital &
social" with only a name and a type; confirm it appears in the plan tree as *Created* and that the
initiative's, campaign's and plan's planned costs went up by its planned cost.

**Acceptance Scenarios**:

1. **Given** the initiative "Digital & social" (owner Priya Raman, 1 Jan – 30 Jun 2026), **When** the user
   chooses **Add activity** on its row, **Then** the page opens with Campaign and Initiative filled in,
   Owner "Priya Raman" and dates 1 Jan – 30 Jun 2026 labelled "From the initiative", and each field's
   marking shown as a word (**Required**, **Needed to submit**).
2. **Given** the new-activity page, **When** the user opens **Activity type**, **Then** only the active
   types the plan's model allows are offered, each with its default planned cost, and choosing "Event"
   (default 25,000.00 USD) fills the planned cost, labelled "From the activity type".
3. **Given** a name and a type, **When** the user chooses **Save**, **Then** the activity is created as
   *Created* with a new ID (e.g. A-1043), the toast says "Activity 'Launch film' created", the activity
   page opens, and the initiative, campaign and plan planned costs include the new amount.
4. **Given** a name already used in the initiative (ignoring case and spaces), an end date before the
   start date and dates outside the initiative's, **When** the user saves, **Then** nothing is saved and
   all problems appear at once, in a summary with links and under each field ("'Digital & social' already
   has an activity called 'Launch film'.", "Dates must be inside the initiative's dates (1 Jan 2026 – 30
   Jun 2026).").
5. **Given** the user ticks **Ongoing (no end date)**, **Then** the end date is removed and a warning says
   "Ongoing can't be undone after saving."; it can still be unticked before saving.
6. **Given** **Save and add another**, **Then** an empty form opens under the same initiative; **Given**
   typed values, **When** the user chooses **Cancel**, **Then** they are asked **Keep editing** or
   **Discard changes**.

---

### User Story 2 - See an activity's status and everything it still needs (Priority: P1)

The activity page header shows status, completeness, ID, owner, dates, planned cost and unassigned
planned cost. A **Ready to submit** panel lists every problem that would stop submission — of the activity
and its line items — each with a link. In the plan tree each activity is a row with the same chips.

**Why this priority**: "Status is always visible" is a core promise (constitution III); Hive9 showed one
error per save, and this panel replaces that with one complete list.

**Independent Test**: Open the seeded, deliberately incomplete Mobility China activities and confirm that
every missing field, the 90 % Country / OU split and the missing GL code are listed at once with links.

**Acceptance Scenarios**:

1. **Given** activity "Launch film" (*Created*, three needed-to-submit fields empty), **When** the user
   opens it, **Then** the header shows the grey chip *Created*, the red chip "3 fields needed", A-1042, the
   owner, "1 Mar 2026 – 31 May 2026", "120,000.00 USD" and "Unassigned 20,000.00 USD".
2. **Given** an activity with an empty cost centre, a Country / OU split at 90.00 %, a line item without a
   GL code, a custom spread adding up to 900.00 of 1,000.00, and line items exceeding the planned cost by
   500.00 USD, **Then** the Ready to submit panel lists all five at once, grouped "This activity" and
   "Line items", each with a link that opens the field; with nothing missing it says "Ready to submit".
3. **Given** an approved activity, **When** today reaches its start date, **Then** the chip shows *In
   progress*, and after its end date *Complete*; an ongoing activity never shows *Complete*.
4. **Given** problems remain, **When** the user chooses **Submit for approval**, **Then** the message says
   "Submitting needs 5 things — view list" and the panel opens.
5. **Given** the plan tree with an initiative expanded, **Then** each activity shows as a row with level
   bar and tag, name, ID, status and completeness chips, owner, dates and planned cost; selecting the row
   opens the activity page.

---

### User Story 3 - Fill in Shell attributes, % splits and goals (Priority: P1)

The Shell attributes section shows the attributes every activity needs and, by the plan's business unit,
a Lubricants and a Mobility group saying why they are shown; Country / OU and Brands use the split editor.
In Goals, the user types INQ and NorthStar calculates MQL and revenue from the plan's model, unless the
user chooses **Set later stages myself**.

**Why this priority**: These attributes carry the finance keys (company code, cost centre) and Shell's
reporting dimensions; Hive9 silently hid whole groups, which users could not understand.

**Independent Test**: On a Lubricants plan ("Lubricants 1.0": 2 %, 5 %, 10,000 USD), fill every attribute,
split Country / OU over three countries, type INQ 250,000, then change the plan's business unit to
Mobility and back; confirm the values survive and the goals read MQL 5,000 and 2,500,000.00 USD.

**Acceptance Scenarios**:

1. **Given** a plan whose business unit is Lubricants, **When** the user edits an activity, **Then** the
   always-shown attributes and the Lubricants group appear, the group saying "Shown because the plan's
   business unit includes Lubricants", and the Mobility group is not shown.
2. **Given** a plan with no business unit, **Then** only the always-shown attributes appear, with
   "Lubricants and Mobility attributes appear once the plan's business unit is set" and a link to the plan.
3. **Given** Country / OU, **When** the user selects Germany, France and Netherlands, **Then** the split
   editor shows 33.33 %, 33.33 % and 33.34 % and "100.00 % — complete".
4. **Given** the user changes Germany to 50.00 %, **Then** the total reads "116.67 % — 16.67 % too much";
   **Save** still works, the Ready to submit panel lists "Country / OU shares add up to 116.67 % — they
   must add up to 100.00 %", and **Split evenly** restores an even split in one step.
5. **Given** an activity with Lubricants values, **When** the plan's business unit changes to Mobility only,
   **Then** the Lubricants values are kept but hidden and not counted as needed, and they reappear when
   the business unit includes Lubricants again.
6. **Given** the model "Lubricants 1.0", **When** the user types INQ 250,000, **Then** MQL shows 5,000 and
   revenue 2,500,000.00 USD, labelled "Calculated" with an ⓘ showing the formula and the model's values;
   with INQ empty they show "—", and INQ "-5" or "12.5" gives "Enter a whole number of 0 or more."
7. **Given** **Set later stages myself** is on, **Then** MQL and revenue are editable and keep their typed
   values when INQ changes; **When** it is turned off, **Then** NorthStar lists the typed values that will
   be replaced before saving.

---

### User Story 4 - Edit an activity, including one that is submitted or approved (Priority: P1)

**Edit** turns the activity page into the same form. Most edits simply save; on a submitted or approved
activity, changing planned cost or dates sends it back for approval, and NorthStar says so first.

**Why this priority**: Plans change all year; approval must stay meaningful without blocking small fixes
(decision D-013).

**Independent Test**: Change the description of an approved seed activity (it stays *Approved*), then its
planned cost (confirmation, it becomes *Created*); check both changes in the activity log.

**Acceptance Scenarios**:

1. **Given** a *Created* activity, **When** the user edits fields and saves, **Then** the toast says
   "Activity 'Launch film' saved" and the log records each field's old and new value.
2. **Given** a *Submitted* activity (banner: "Submitted for approval — changing planned cost or dates will
   restart approval."), **When** the user changes the planned cost from 120,000.00 to 150,000.00 USD and
   saves, **Then** a dialog "This activity will need approval again" lists the change with **Save and
   restart approval** and **Keep editing**; on confirmation it is saved and the status becomes *Created*.
3. **Given** an *Approved* activity, **When** the user changes only the description, owner or attributes,
   **Then** it saves without a dialog and stays *Approved*.
4. **Given** a saved ongoing activity, **Then** the Ongoing choice shows a lock and "Ongoing can't be undone
   — to plan with an end date, create a new activity."
5. **Given** an activity with a planned cost, **When** the user changes its type to "Event" (default
   25,000.00 USD), **Then** NorthStar asks "Use the default planned cost of 25,000.00 USD for 'Event'?"
   with **Use default** and **Keep 120,000.00 USD**.

---

### User Story 5 - Move, copy and delete an activity (Priority: P2)

From **More**, the user moves an activity to another initiative of the same plan, copies it (with its
line items) as a new *Created* activity, or deletes it to the recycle bin with **Undo**.

**Why this priority**: Needed to reorganise a plan, but less frequent than creating and editing.

**Independent Test**: Copy a seeded approved activity, move the copy to another initiative, delete it and
undo the delete; check totals, statuses and the activity log at each step.

**Acceptance Scenarios**:

1. **Given** "Launch film" (1 Mar – 31 May 2026), **When** the user chooses **More › Move** and picks
   "Print & outdoor" (1 Jan – 31 Dec 2026), **Then** it moves with its line items and status, both
   initiatives' totals update and the move is logged; a target running 1 Jun – 31 Dec 2026 is refused
   with "'Launch film' runs 1 Mar – 31 May 2026, but 'Retail events' runs 1 Jun – 31 Dec 2026. Change the
   activity's dates first, or pick another initiative."
2. **Given** an *Approved* activity, **When** the user chooses **More › Copy**, **Then** "Launch film
   (copy)" is created as *Created* with all values and copies of its line items (new IDs), without
   approvals, comments, history or GSAP values, and its page opens.
3. **Given** "Launch film" with 3 line items, **When** the user chooses **More › Delete**, **Then** the
   dialog asks "Delete activity 'Launch film'?", says "3 line items go with it", "The initiative, campaign
   and plan planned costs go down by 120,000.00 USD" and "It moves to the recycle bin; you can restore it",
   with **Cancel** focused and a red **Delete activity** button.
4. **Given** the activity has just been deleted, **Then** the plan page shows "Activity 'Launch film' moved
   to the recycle bin" with **Undo**, which within 10 seconds returns the activity and its line items with
   the same IDs, values and status.

---

### Edge Cases

- The plan's model allows no activity types: **Add activity** stays visible and explains "The model
  'Lubricants 1.0' has no activity types yet — an administrator adds them in Settings."
- The type is later removed from the model or made inactive: the value is kept and labelled; a type the
  model no longer allows is listed in the Ready to submit panel.
- The initiative's dates change (feature 005) so the activity falls outside them: other edits still save;
  the Ready to submit panel lists "Dates are outside the initiative's dates (…)".
- A negative planned cost is refused ("Planned cost can't be negative."); 0.00 is a real value. Dates
  crossing the plan's year end are fine inside the initiative; multi-year linking is out of scope (D-017).
- Two people save the same activity at once: the second save is refused with "Not saved — Priya Raman
  changed this activity at 10:42", keeping the typed values on screen (shared behaviour of feature 001,
  FR-029).

## Requirements *(mandatory)*

### Functional Requirements

**Creating an activity**

- **FR-001**: Users MUST be able to start a new activity from **Add activity** on an initiative's row in
  the plan tree and on the initiative's details (feature 005), with campaign and initiative pre-filled.
- **FR-002**: The new-activity form MUST open on a full page (design system §6.2) with the fields of field
  catalogue §4.1–4.5 in the section order of design system §7, defaults pre-filled and labelled with their
  source, markings shown as words, and "Save the activity to add its line items" in place of line items.
- **FR-003**: Saving MUST require only the fields marked **Required** in field catalogue §4 (name, type,
  initiative); "needed to submit" fields MAY stay empty (decision D-009).
- **FR-004**: On save, the activity MUST be created as *Created* with a permanent readable ID (A-…, D-019),
  logged with the actor, announced by a toast, and its page opened. **Save and add another** MUST save and
  open an empty form under the same initiative.
- **FR-005**: The planned costs of the initiative, campaign and plan MUST reflect the activity whenever it
  is saved, moved, copied, deleted or restored (domain model §4.2).

**Fields and their rules**

- **FR-006**: Activity names MUST be unique within their initiative, ignoring letter case and surrounding
  spaces (domain model §1). The external name is optional; when empty, the activity name is used.
- **FR-007**: **Activity type** MUST offer only the active types the plan's model allows (R7). A type's
  default planned cost MUST fill an empty planned cost; when one is already set, NorthStar MUST ask "Use
  the default planned cost of <amount> for '<type>'?" (**Use default** · **Keep <current amount>**).
- **FR-008**: **Owner** MUST default to the initiative's owner and use the person picker (feature 003).
- **FR-009**: Start and end dates MUST default to the initiative's dates. When the user enters or changes
  dates, saving MUST be blocked unless the end is on or after the start, both lie inside the initiative's
  dates (R3), and the end is no later than 31 December of the plan's year + 10 (R5).
- **FR-010**: **Ongoing (no end date)** MUST remove the end date and warn before the first save that it
  cannot be undone; once saved it MUST be locked with the reason (R6). For ongoing activities only the
  start date is checked against the initiative's dates.
- **FR-011**: **Planned cost** MUST be a USD amount of 0.00 or more (design system §5 money input); 0.00
  counts as filled; more than two decimals are rounded half-up to the cent with a note.
- **FR-012**: **Approval workflow** MUST offer the active approval workflows (feature 008; seeded until
  then), filled in (saying so) when only one is active, and be changeable only while the activity is
  *Created* or *Declined*; otherwise it shows a lock and "It can't change while the activity is submitted
  or approved."
- **FR-013**: Inactive values already chosen (type, owner, workflow, attribute options) MUST be kept,
  labelled "(inactive)" and not offered for new choices (domain model §6).

**Shell attributes**

- **FR-014**: The Shell attributes section MUST show the activity-level Shell attributes configured in
  Settings (feature 002; seeded as field catalogue §4.2–4.4), each with the control for its type (design
  system §5) and its marking.
- **FR-015**: Groups with a business-unit condition MUST appear only when the plan's business unit
  includes that unit (R9), saying "Shown because the plan's business unit includes <unit>"; with no
  business unit, the section MUST explain how to make them appear, with a link to the plan. Values in
  hidden groups MUST be kept, MUST NOT count towards completeness, and MUST reappear with the group.
- **FR-016**: A dependent attribute MUST stay visible but disabled with a hint until its parent is set, be
  filled in (saying so) when only one option fits, and list values that will be cleared when the parent
  changes, before saving (design system §7).
- **FR-017**: Multiple-choice attributes with a % split MUST use the split editor (design system §5): new
  selections start evenly split to two decimals, remainder on the last value (R10); a running total shows
  "100.00 % — complete", "… left" or "… too much"; **Split evenly** re-splits. While shares are still
  even, adding or removing a value re-splits them; otherwise a new value starts with the share still left
  (or 0.00 %). Each share MUST be 0.00–100.00 with at most two decimals (blocks saving); a total other
  than exactly 100.00 % MUST NOT block saving but MUST block submitting.

**Goals**

- **FR-018**: The Goals section MUST offer INQ (whole number, 0 or more) and show MQL and revenue
  calculated from the plan's model's current values (domain model §4.5), labelled "Calculated" with an ⓘ
  giving the formula; with INQ empty they show "—". **Set later stages myself** MUST make MQL (whole
  number) and revenue (USD) editable; turning it off MUST list the typed values that will be replaced.

**Activity page, status and completeness**

- **FR-019**: The activity page MUST be the activity's details view (design system §6.3). Its header MUST
  show the breadcrumb (Plans › plan › campaign › initiative), name, level tag, status chip, completeness
  chip, ID, owner, dates (or "Ongoing from <start>"), planned cost, unassigned planned cost (domain model
  §4.3; red when negative), who created and last changed it and when, and the actions **Edit** (primary),
  **Submit for approval** and **More** (Copy, Move, View history, Delete last).
- **FR-020**: Below the header, in order: the Ready to submit panel, Details, Shell attributes, Goals, Line
  items (content: feature 007), Approval (content: feature 008; until then the chosen workflow and
  "Approval steps and decisions will appear here"), and "From finance (coming later)" — collapsed, showing
  field catalogue §4.6 as "—" with the explanation of domain model §4.6. Empty values show "—".
- **FR-021**: The status chip MUST show the status of domain model §2.3 as word and colour (design system
  §4): *Created*, *Submitted*, *Declined*, *Approved* as stored, and *In progress* / *Complete* derived
  from today's date for approved activities (never *Complete* when ongoing).
- **FR-022**: The completeness chip MUST show "All filled in" or "<n> fields needed" for the activity's own
  visible needed-to-submit fields, plus "<n> line items need attention" when any line item is incomplete
  (domain model §2.6); choosing it MUST open the Ready to submit panel.
- **FR-023**: The Ready to submit panel MUST list, all at once, grouped "This activity" and "Line items",
  each with a link that opens the form at the field or line item: every empty visible needed-to-submit
  field of the activity and of its line items; every % split not totalling 100.00 %; every custom spread
  not adding up; a negative unassigned planned cost (D-016); dates outside the initiative's dates; a type
  the model no longer allows; and the line-item problems feature 007 adds. With nothing listed it MUST say
  "Ready to submit".
- **FR-024**: A positive unassigned planned cost MUST appear in the panel as advice that does not block
  ("2,000.00 USD not yet assigned to a line item"). [NEEDS CLARIFICATION: Must an activity's planned cost
  be fully assigned to line items (unassigned planned cost exactly 0.00, so at least one line item when
  the planned cost is above 0) before it can be submitted, or is a positive balance only advice? Default
  until answered: advice only, as domain model §4.3 blocks only a negative balance.]
- **FR-025**: **Submit for approval** MUST always be visible; while problems remain, choosing it MUST say
  "Submitting needs <n> things — view list" and open the panel. Submitting itself is feature 008.

**Editing and re-approval**

- **FR-026**: **Edit** MUST turn the page into a form with the same layout (design system §6.4): **Save**
  is never disabled, all problems show at once, leaving with unsaved changes asks **Keep editing** or
  **Discard changes**, and every saved change is logged with old and new values.
- **FR-027**: While an activity is *Submitted* or *Approved* (including shown *In progress* / *Complete*),
  its page MUST show "<status> — changing planned cost or dates will restart approval." When a save
  changes its planned cost, start date or end date (including making it ongoing), NorthStar MUST first
  show "This activity will need approval again", listing each change (old → new) and its effect on
  line-item spreads (feature 007), with **Save and restart approval** and **Keep editing**. On
  confirmation the activity MUST return to *Created* with the reason logged; ending the pending or given
  approval is defined by feature 008 (domain model §2.5, D-013).
- **FR-028**: Other edits (names, description, owner, attributes, goals) MUST NOT change the status.

**Move, copy and delete**

- **FR-029**: **Move** MUST offer the other initiatives of the same plan (searchable, grouped by campaign)
  and MUST refuse, saying why and what to do, when the activity's dates are not inside the target's dates
  or the target already has an activity with that name. A move keeps status, line items and values,
  updates both initiatives' totals and is logged.
- **FR-030**: **Copy** MUST create, in the same or another initiative of the same plan (same checks as
  Move), a copy with all values and copies of its line items (new IDs), named " (copy)", " (copy 2)"…,
  status *Created*, without approvals, comments, history or GSAP values (R13), and open it.
- **FR-031**: **Delete** MUST follow design system §6.5: the dialog names the activity, its line-item
  count, the effect on planned costs (and, when *Submitted*, that it leaves the approvers' inbox) and the
  recycle bin; the plan page then shows a toast with **Undo** for at least 10 seconds. **Undo** MUST
  restore the activity and its line items with the same IDs, values and status; a later restore from the
  recycle bin (feature 011) brings it back as *Created* (R14).
- **FR-032**: **View history** MUST open the activity's history (feature 011; "Coming soon" until then).

**Plan tree rows**

- **FR-033**: In the plan tree (features 004–005), each initiative's activities MUST appear as rows with
  level bar and tag, name as a link, ID, status chip, completeness chip, owner, dates and planned cost
  (with a red mark when the unassigned planned cost is negative), and a **⋯ Actions** menu (View · Edit ·
  Copy · Move · Delete last). Selecting a row MUST open the activity page.

### Key Entities *(include if feature involves data)*

- **Activity**: one piece of marketing work under an initiative — the fields of field catalogue §4
  (including its goals: INQ, the "set later stages myself" choice, MQL and revenue), status (domain model
  §2.3), readable ID, who created and last changed it. Owns its line items (feature 007).
- **Attribute value**: the activity's value for one Shell attribute; for split attributes, each chosen
  option with its share (two decimals).
- **Submit check** (derived, not stored): the list of problems shown in the Ready to submit panel.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user creates an activity with one line item (name, type, planned cost, one line
  item with its defaults accepted) in at most 15 steps (design system §10).
- **SC-002**: For 100 % of the seeded incomplete activities, every problem that would stop submission is
  shown in one view, in one attempt, and each is reached in one click.
- **SC-003**: In a test with at least 5 first-time users, at least 90 % create, edit and delete an
  activity without help, each task within 5 minutes.
- **SC-004**: Users can tell an activity's status and what is missing from its plan-tree row or page
  header, without opening another screen, for 100 % of activities.
- **SC-005**: 100 % of the changes listed in domain model §2.5 on a submitted or approved activity are
  confirmed before saving and return it to *Created*; 0 % of other edits change its status.
- **SC-006**: Calculated MQL and revenue match domain model §4.5 for 100 % of tested values, and planned
  cost roll-ups match the sum of activities to the cent after every create, move, copy and delete.
- **SC-007**: A user fixes a % split that does not total 100.00 % in at most 2 steps.
- **SC-008**: 100 % of locked fields and unavailable actions on the activity page explain why.

## Assumptions

- Plans (feature 004) provide business unit, model and year; feature 005 provides the plan tree, the
  initiative's dates and owner, and the **Add activity** entry points.
- Line items, the spread grid and the unassigned-balance row are feature 007; submitting, approving,
  declining and ending an approval when an activity returns to *Created* are feature 008, whose approval
  workflows are seeded from the demo data until it is delivered.
- The planned cost is entered in USD (base currency, D-010); a display currency (feature 003) only changes
  how amounts are shown. Moving an activity does not trigger re-approval (not listed in domain model §2.5).
- Single-activity copy and move stay inside one plan; copying into another plan happens through plan copy
  (feature 004) or bulk actions (feature 012).
- Undo right after a delete restores the status as it was; only a later recycle-bin restore resets the
  activity to *Created* (R14).
- No permissions in this phase: every signed-in user can do everything here (D-004); every change records
  the actor.
