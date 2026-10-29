# Feature Specification: Plans

**Feature Branch**: `004-plans`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Plans list and plan page; create, view, edit, copy, publish/unpublish, delete plans; budget, goals, Shell attributes"

**References**: Domain model [§1 hierarchy, §2.1 plan status, §2.6 completeness, §3 required fields, §4.1–4.2 money, §4.5 goals, §5 rules R1, R7, R9, R13, R14, §8 later](../../docs/product/domain-model.md) · Field catalogue [§1 Plan](../../docs/product/field-catalogue.md) · Design system [§2 layout, §4 colours, §5 components, §6 CRUD pattern, §7 forms, §10 targets](../../docs/product/design-system.md) · Glossary [planning, money, goals](../../docs/product/glossary.md) · Demo data [planning seed](../../docs/product/demo-users-and-data.md) · Decisions D-004, D-009, D-011, D-015, D-018, Q-001, Q-004, Q-006 · Hive9 as-is step 1; functional analysis B3, B8, B10

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a plan (Priority: P1)

**New plan** opens a full page with sections in the order fields depend on each other — *Basics*,
*Budget*, *Shell attributes*, *Goals* — and sensible defaults, so a name, a model and a business unit give
a plan that is ready to publish.

**Why this priority**: Everything else in the Plan module lives inside a plan.

**Independent Test**: As Priya Raman, create "FY2026 Lubricants Test" with model Lubricants 1.0 and
business unit Lubricants; its plan page shows Draft and "All filled in".

**Acceptance Scenarios**:

1. **Given** the Plans page, **When** the user chooses **New plan**, **Then** the form opens with Year =
   current year, Teams = the user's teams, Owner = the user, Goal stage = INQ and spread = Evenly, each
   labelled as a default, and every field marked **Required**, **Needed to publish** or nothing.
2. **Given** a name and a model, **When** the user chooses **Save**, **Then** the plan is created as
   **Draft** with a new ID, its page opens and a toast says "Plan '<name>' created — add campaigns next."
3. **Given** an empty name, no model and a plan name already in use (any letter case), **When** the user
   chooses **Save**, **Then** all problems appear at once (summary and under each field); nothing is saved.
4. **Given** no team is chosen, **Then** Owner is disabled with "Choose teams first — the owner must belong
   to one of the plan's teams"; with teams chosen it offers only their members (R1).
5. **Given** model Lubricants 1.0, **When** the user types an INQ goal of 250,000, **Then** MQL 5,000 and
   Revenue 2,500,000.00 USD appear, labelled "Calculated" with the formula; ticking business unit
   Mobility shows SMP with "Shown because the plan's business unit includes Mobility".

---

### User Story 2 - Find and compare plans (Priority: P1)

The **Plans** page shows summary cards and every plan with its key facts, status, completeness and
budget use; filters, a **My teams / All plans** switch and a search narrow the list.

**Why this priority**: The list is the way into every plan and shows which plans are ready and spending.

**Independent Test**: With the demo data, filter to Draft, then to 2026, then search an ID; check cards,
result count and summary cards against the seed plans.

**Acceptance Scenarios**:

1. **Given** the demo data, **When** the user opens Plans, **Then** the three seed plans appear newest
   year first, then by name, and the summary cards show 3 plans, 2 Draft, 1 Published and money totals.
2. **Given** the list, **When** the user selects **Draft** and year 2026, **Then** only FY2026 Mobility
   China remains, removable chips show both filters and the count reads "1 of 3".
3. **Given** the list, **When** the user types "mobility" or a plan ID in "Search name or ID", **Then**
   only matching plans remain, ignoring letter case.
4. **Given** the user switches to **My teams**, **Then** only plans with one of the user's teams remain,
   with the hint "Shows your teams' plans. Everyone can still open every plan in this phase."
5. **Given** filters that hide every plan, **Then** "Nothing matches these filters" offers **Clear
   filters**; with no plans at all, the empty state offers **Create your first plan**.

---

### User Story 3 - Understand a plan at a glance (Priority: P1)

The plan page has a header, summary cards, the **plan tree** (the plan row; features 005 and 006 add the
rows below it), a **Details** section and a **Goals** section.

**Why this priority**: It shows whether the plan is on budget and ready; all lower-level work starts here.

**Independent Test**: Open FY2026 Lubricants Europe and check every header fact, card, detail field and
goal figure against the seed data.

**Acceptance Scenarios**:

1. **Given** a plan, **When** the user selects it in the list, **Then** its page opens (never the edit
   form) with "Plans › <plan name>", the PLAN tag, chips, ID, owner, year and the actions **Edit**,
   **Publish** or **Unpublish**, and **⋯ More** (Copy, View history, Delete).
2. **Given** plan budget 1,200,000 USD and campaign budgets totalling 1,300,000 USD, **Then** Remaining
   budget shows "−100,000.00 USD" in red with "Over-allocated by 100,000.00 USD" (D-015).
3. **Given** planned cost 960,000 USD, **Then** Budget used shows 80 % with a bar that turns red above
   100 %; **Given** no plan budget, **Then** the budget cards show "—" with "No plan budget set".
4. **Given** no business unit, **Then** the completeness chip reads "1 field needed" and links to it.
5. **Given** a goal of INQ 250,000 and activity goals summing to INQ 150,000, **Then** Goals shows the
   plan's INQ, MQL and Revenue beside the activity totals and "60 % of the INQ goal planned".

---

### User Story 4 - Edit a plan (Priority: P1)

**Edit** turns the details into the creation form with the year locked; saving shows every problem at
once and explains model, business-unit and owner effects before anything changes.

**Why this priority**: Plans change all year; edits must be safe and their effects predictable.

**Independent Test**: Edit FY2026 Lubricants Europe: change the budget, the model, the business unit and
the teams; check each message and the saved result.

**Acceptance Scenarios**:

1. **Given** a saved plan, **When** the user edits it, **Then** Year shows a lock and "Year can't change
   once the plan is saved — copy the plan into the next year or create a new plan."
2. **Given** a new model that does not allow the type of 2 activities, **When** the user saves, **Then**
   a confirmation lists them with links: "They keep their type but need an allowed type before they can
   be submitted. Goals will be recalculated with the new model." — **Change model** or **Keep editing**.
3. **Given** the user removes Lubricants from the business unit, **When** they save, **Then** a
   confirmation says the Lubricants attributes will be hidden on the plan's activities with their values
   kept; adding Lubricants back shows the same values (R9).
4. **Given** the user removes the owner's only team, **Then** Owner shows "Priya Raman isn't in any of
   the chosen teams — choose another owner or clear the field", reported on Save until resolved (R1).
5. **Given** a Published plan, **When** the user saves an edit, **Then** it stays Published and the log
   holds each changed field with old and new values.

---

### User Story 5 - Publish and unpublish a plan (Priority: P1)

**Publish** makes a complete Draft plan the agreed plan of record, or lists everything missing with
links; **Unpublish** returns it to Draft with a reason.

**Why this priority**: Publishing is Shell's quality checkpoint for plans.

**Independent Test**: Try to publish FY2026 Mobility China (deliberately incomplete), fix the listed
fields through the links, publish, then unpublish with a reason.

**Acceptance Scenarios**:

1. **Given** a complete Draft plan, **When** the user chooses **Publish**, **Then** it becomes Published
   and a toast says "Plan '<name>' published".
2. **Given** the plan lacks a business unit and its campaign "Retail growth" lacks cluster and region,
   **When** the user chooses **Publish**, **Then** "Publish needs 3 fields" lists them under "Plan" and
   "Campaign 'Retail growth'", each linking to the field, and the plan stays Draft.
3. **Given** a Published plan, **When** the user chooses **Unpublish**, **Then** a dialog asks "Why are
   you unpublishing this plan?"; **Unpublish plan** returns it to Draft and logs the reason.
4. **Given** incomplete activities, an over-allocated budget or a campaign running past the year end,
   **Then** none of these stops publishing.

---

### User Story 6 - Copy or delete a plan (Priority: P2)

**Copy** creates a Draft plan from an existing one, in the same or the next year, with or without
everything below it. **Delete** says what goes with the plan and moves it to the recycle bin, with Undo.

**Why this priority**: It saves re-typing each year and removes mistakes, but is not needed in year one.

**Independent Test**: Copy FY2026 Lubricants Europe into 2027 and check the copy; delete the original,
check the dialog's counts, choose Undo and confirm everything is back with the same IDs.

**Acceptance Scenarios**:

1. **Given** a plan, **When** the user chooses **Copy**, **Then** a dialog offers the name "<name> (copy)"
   (editable; "(copy 2)" if taken), "Same year (2026)" or "Next year (2027)", and "Include campaigns,
   initiatives, activities and line items" (on) with their counts; with it off, only the plan is copied.
2. **Given** a next-year copy with everything, **Then** a Draft 2027 plan opens with all records under
   new IDs, every date one year later, activities Created and no approvals, comments or history (R13).
3. **Given** a Published plan with 2 campaigns, 4 initiatives, 7 activities (4 approved) and 15 line
   items, **When** the user chooses **Delete**, **Then** "Delete plan '<name>'?" names the counts, says
   the plan is published, that approved activities restored later return as Created, and that it moves
   to the recycle bin; **Cancel** has focus and the red button says **Delete plan**.
4. **Given** the user confirms, **Then** the Plans list opens with "Plan '<name>' moved to the recycle
   bin" and **Undo** for at least 10 seconds; Undo restores everything exactly as it was.

---

### Edge Cases

- The user belongs to no team: Teams starts empty, Owner explains why it is disabled, and the plan can
  still be saved as Draft.
- The plan's model, a team or an option becomes inactive: the plan keeps it, labelled "(inactive)".
- A model rate is 0 %, so a goal cannot be worked backwards: the stage shows "—" with the reason.
- Someone else saved the plan while the user was editing: Save is refused, naming who and when.
- Two people save plans with the same name at the same moment: the second gets the duplicate-name message.
- A copy name would exceed 120 characters: the original part is shortened so the suffix fits.
- Next-year copy: 29 February becomes 28 February; ongoing activities keep no end date; "Next year" is
  shown but unavailable, with the reason, when it is beyond the permitted plan years.
- A deleted plan's name may be reused; a name clash on a later restore is handled by feature 011.

## Requirements *(mandatory)*

### Functional Requirements

**Plans list**

- **FR-001**: The Plans page MUST follow the standard list pattern (design system §6.1): eyebrow "PLANS",
  title, description, **New plan**, and summary cards Plans, Draft, Published, Total planned cost and Total
  plan budget over the plans in the chosen year and team scope. Selecting a plan opens its plan page; its
  **⋯ Actions** menu offers View, Edit, Copy, Publish or Unpublish, and Delete (last, separated).
- **FR-002**: Each plan MUST appear as a card (or table row, if the user picks Table) with name, ID, year,
  status, completeness and AOP chips, owner, teams, business unit, planned cost against plan budget as a
  bar (red above 100 %, "No budget set" when empty) and "n of m activities approved" (Approved, In
  progress and Complete count as approved).
- **FR-003**: The list MUST offer pill filters All · Draft · Published, a year filter (All years and each
  year with a plan), search on name or ID ignoring letter case, removable filter chips and a result count;
  it is sorted by year (newest first), then name; filters and layout are remembered per user.
- **FR-004**: **My teams** MUST show plans with one of the user's teams; it never stops anyone opening a
  plan (Q-006), and **All plans** is the default.
  *Since feature 014 this rule follows roles: every action needs its permission of 014 Table A in the plan's scope (014 FR-041).*

**Create plan**

- **FR-005**: **New plan** MUST open a full-page form with the fields of field catalogue §1 (#1–#16) in
  the sections Basics, Budget, Shell attributes and Goals, each field after the fields it depends on (year
  before default finance budget, teams before owner, model before goals, business unit before SMP), with
  markings shown as words (domain model §3).
- **FR-006**: The form MUST pre-fill and label the defaults of field catalogue §1 (year = current year,
  choices current year − 1 to + 5; the user's teams; the user as owner when eligible; Strategic / AOP and
  BCPIR plan = No; spread = Evenly; goal stage = INQ) and the model when only one is active. *Amended by
  feature 034 (D-326): the year choices are the current year to + 10, no past year.*
- **FR-007**: Model MUST offer only active models and show the chosen model's rates and deal size; Owner
  MUST offer only members of the chosen teams and, with no team chosen, stay visible but disabled with the
  reason (R1); Default finance budget MUST offer only finance budgets of the plan's year, and a year
  change clears one of another year with a note.
- **FR-008**: The Shell attributes section MUST show every active plan-level attribute from Settings
  (feature 002) in display order with its marking; business-unit attributes appear only when the business
  unit includes that unit, saying why (R9).
- **FR-009**: The goal amount MUST be a whole number for INQ and MQL and a USD amount for Revenue; the
  other stages and deals MUST be calculated live from the model (domain model §4.5), forwards or
  backwards, rounded half up to whole numbers (revenue to the cent), labelled "Calculated" with the
  formula, or "Choose a model to calculate" without a model.
- **FR-010**: Saving MUST require only name, year and model, and MUST check: name of at most 120
  characters, unique across MAPP ignoring letter case and surrounding spaces; owner in a chosen team
  (R1); description of at most 2,000 characters; budget and goal amount not negative. Every problem is
  reported in one attempt (design system §6.4). A new plan is Draft, gets a permanent ID and opens.

**Plan page**

- **FR-011**: The plan page MUST show the details header of design system §6.3: breadcrumb, PLAN tag,
  status, completeness and AOP chips, ID, owner, year, teams, business unit, created and last changed by
  and on, and **Edit** (primary), **Publish** or **Unpublish**, **⋯ More** (Copy, View history, Delete).
- **FR-012**: Summary cards MUST show, each with an ⓘ formula: Plan budget; Planned cost (domain model
  §4.2); Remaining budget = plan budget − sum of campaign budgets (§4.1), red with "Over-allocated by X"
  when negative; Budget used = planned cost ÷ plan budget with a bar, red above 100 %; Activities approved
  "n of m". Budget cards show "—" with the reason when no budget is set.
- **FR-013**: The plan page MUST hold the **plan tree**: a card with "Search name or ID in this plan",
  **Expand all** / **Collapse all**, columns Name, Owner, Dates, Status, Budget, Planned cost and Budget
  used, and the plan row — plan level bar and PLAN tag (design system §4), name, owner, "1 Jan – 31 Dec
  <year>", status and completeness chips, budget with remaining budget beneath, planned cost and budget
  used. Features 005 and 006 add their rows, add actions and the totals row.
- **FR-014**: Tree rows MUST expand and collapse by mouse and keyboard; search MUST match name or ID
  ignoring letter case, keep ancestors of matches visible and open the path, or say "Nothing in this plan
  matches '<text>'"; a plan without campaigns says "No campaigns yet — a campaign is a theme that runs for
  most of the year."
- **FR-015**: Details MUST show every field of field catalogue §1 read-only, "—" when empty, calculated
  values labelled "Calculated", business-unit attributes only while shown, and inactive values
  "(inactive)". All amounts in this feature appear in the user's display currency (feature 003), labelled.
- **FR-016**: Goals MUST show the plan goal (INQ, MQL, Revenue) beside the sum of all the plan's activity
  goals (domain model §4.5) and the share of the goal stage already planned.
- **FR-017**: The completeness chip MUST follow domain model §2.6: "All filled in" or "n fields needed" for the
  plan's own needed-to-publish fields, each linking to the field, plus "n items below need attention"
  listing every campaign, initiative, activity and line item below that is not complete.

**Edit plan**

- **FR-018**: **Edit** MUST open the plan form with current values and Year read-only with the lock and
  reason (design system §6.6); Save is never disabled, every problem is reported in one attempt, and
  leaving with unsaved changes asks **Keep editing** or **Discard changes**.
- **FR-019**: Before saving a model change, MAPP MUST list the activities whose type the new model
  does not allow (R7) with links; once confirmed, they keep their type and show "Type not allowed by the
  plan's model" until changed, and the plan goal and model-calculated activity goals are recalculated.
- **FR-020**: Before saving a business-unit change, MAPP MUST say which attribute groups will be
  hidden or shown on the plan and its activities; hidden values are kept and reappear when shown again
  (R9), and only shown attributes count towards completeness.
- **FR-021**: When the owner belongs to none of the chosen teams, Owner MUST show a problem naming the
  person, and Save MUST report it until an eligible owner is chosen or the field is cleared (R1).
- **FR-022**: A Published plan MUST stay Published when edited, saying that changes are logged; a changed
  default finance budget or default spread applies only to line items created afterwards.
- **FR-023**: If the plan changed after the user opened the form, Save MUST be refused, naming who changed
  it and when, and keeping the typed values on screen.

**Publish and unpublish**

- **FR-024**: Draft plans MUST offer **Publish**, which succeeds only when the plan and every campaign
  and initiative in it have all needed-to-publish fields filled (domain model §2.1); otherwise it lists
  every missing field at once, grouped by record with links, and changes nothing. Activities, line items
  and warnings (negative remaining budget, campaign dates outside the year) MUST NOT block publishing.
- **FR-025**: Published plans MUST offer **Unpublish**, which asks for a reason (1–500 characters),
  returns the plan to Draft and records the reason in the activity log.

**Copy plan**

- **FR-026**: **Copy** MUST open a dialog with the new name pre-filled per R13 and editable (checked for
  uniqueness), "Same year" or "Next year", and "Include campaigns, initiatives, activities and line
  items" (on by default) with their counts.
- **FR-027**: The copy MUST be a Draft plan with a new ID and all the source plan's fields; with the
  include option, everything below is copied per R13 (new IDs, activities Created, no approvals, comments,
  history or GSAP values). It is made completely or not at all, then its plan page opens.
- **FR-028**: A next-year copy MUST use the source year + 1, move every copied date one year later
  (29 February → 28 February), clear the default finance budget with a note, and move line items' finance
  budget links to the new year as defined by feature 007; "Next year" is offered only when it is a
  permitted plan year, otherwise shown with the reason. *Amended by feature 034 (D-326): permitted up to the
  current year + 10.*

**Delete plan**

- **FR-029**: Delete MUST follow design system §6.5, naming the counts of campaigns, initiatives,
  activities and line items, saying when the plan is published or holds approved activities (which return
  as Created if restored later, R14), and moving everything to the recycle bin.
- **FR-030**: After deleting, the Plans list MUST open with a toast offering **Undo** for at least
  10 seconds; Undo restores the plan and everything below it exactly as it was (feature 001).

**History and later rules**

- **FR-031**: **View history** MUST open the plan's history (feature 011; "Coming soon" until then), and
  every create, edit, publish, unpublish, copy and delete MUST be logged with the actor (feature 001).
- **FR-032**: MAPP MUST NOT restrict these actions by role or team (D-004). Shell's rules — plans
  visible only to their teams' members; creating, publishing, unpublishing and deleting plans limited to
  permitted roles — are enforced when roles arrive.
  *Since feature 014 this rule follows roles: every action needs its permission of 014 Table A in the plan's scope (014 FR-041).*

### Key Entities *(include if feature involves data)*

- **Plan**: fields per field catalogue §1, status, ID, created and last changed by and on; refers to a
  **Model**, **Teams**, an owner (**Person**) and a **Finance budget** (features 002, 003).
- **Plan goal**: goal stage and amount; other stages calculated from the plan's model.
- **Plan attribute value**: the plan's value for a plan-level Shell attribute; kept while hidden.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Starting on the Plans page, a first-time user creates a plan that is ready to publish in at
  most 6 steps using the defaults (design system §10).
- **SC-002**: In a test with at least 5 first-time users, at least 90 % create a plan within 5 minutes
  and edit, copy and delete one within 2 minutes each, without help.
- **SC-003**: 100 % of form problems and of fields missing for publishing are shown in a single attempt
  with working links, and 100 % of read-only fields and unavailable actions say why.
- **SC-004**: A user finds any plan by name or ID from the Plans page in at most 2 steps.
- **SC-005**: Without opening a plan, test users correctly state its status, what is missing and whether
  it is over budget in at least 90 % of questions.
- **SC-006**: Totals, remaining budgets, budget used and calculated goals equal hand-calculated values in
  100 % of test cases, including negative remaining budgets.
- **SC-007**: A copy matches its source in 100 % of copied fields and records, with statuses reset per
  R13 and, for next-year copies, every date exactly one year later.
- **SC-008**: 100 % of plan deletes can be undone within 10 seconds with nothing lost.
- **SC-009**: With 200 plans, search and filter results appear in under 1 second; a plan with 500
  activities opens in under 2 seconds.

## Assumptions

- Builds on feature 001 (layout, components, activity log, recycle bin, Undo), 002 (models, finance
  budget items, plan attributes) and 003 (people, teams, display currency). Features 005 and 006 add the
  tree rows, **Add campaign**, the totals row and activity counts (until then counts show 0 and planned
  cost "—"); feature 007 defines how copied line items' finance budget links move to a new year.
- **My teams** is off by default so nothing is hidden in the demo.
- After a model change, activities with a disallowed type keep their type and status; feature 006 blocks
  submitting them until the type changes (R7).
- A copy keeps the source's owner, teams, budget, goal and Shell attributes; the copier is its creator.
- Undo straight after a delete restores exactly (feature 001); a later restore from the recycle bin
  follows R14 (feature 011).
- Publishing needs no campaign and no approved activity; BCPIR plan is only a yes/no flag (Q-001); the
  fiscal year is the calendar year (D-011, Q-004).
- A save over someone else's newer change is refused rather than merged, to keep the rule simple.
