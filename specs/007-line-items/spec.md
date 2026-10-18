# Feature Specification: Line Items

**Feature Branch**: `007-line-items`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "Line items: currency, spread over months, finance coding, finance budget link, SWBS request flag, unassigned balance"

**References**: Domain model [§1 hierarchy, §2.4 line item status, §2.5 re-approval, §2.6 completeness, §3 required fields, §4.3 line items and the unassigned balance, §4.4 spreading, §4.6 from finance, §5 rules R11–R13, §6 currencies and finance budget items](../../docs/product/domain-model.md) · Field catalogue [§5 line item, §1 #9–10 plan defaults](../../docs/product/field-catalogue.md) · Design system [§4 money, §5 money input, §6 CRUD pattern, §6.6 locked and read-only, §7 forms, §8 words, §10 usability targets](../../docs/product/design-system.md) · Glossary [money](../../docs/product/glossary.md) · Decisions D-008, D-009, D-010, D-013, D-016, D-019 · Shell Plan Module doc step 5 · Hive9 functional analysis §B7 · GSAP workshop summary · ZenReach spec §7.4.8, §8.11 (layout and interaction only)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See an activity's costs at a glance (Priority: P1)

On the activity page (feature 006), the Line items section lists each cost with its currency, amounts,
spread, completeness and SWBS request status, then a system row showing how much of the activity's
planned cost is not yet assigned, then a month-by-month grid with totals.

**Why this priority**: Line items are where money is planned and the only level that will later talk to
GSAP; the unassigned balance tells the user when the activity's costs are fully planned.

**Independent Test**: Open the seeded Lubricants activity with a non-zero unassigned balance and check
that the rows, the balance, its meter and the monthly totals agree to the cent.

**Acceptance Scenarios**:

1. **Given** "Launch film" (planned cost 120,000.00 USD, 1 Mar – 31 May 2026) with "Agency fee"
   50,000.00 USD and "Film production" 46,000.00 EUR, **When** the user opens the activity, **Then** each
   row shows name, ID, currency, local amount, USD amount (50,000.00 USD for the EUR line at 0.92), spread
   method, completeness chip and SWBS request chip, and the row "Unassigned planned cost
   (Sys_Gen_Balance)" shows 20,000.00 USD with the meter "100,000.00 of 120,000.00 USD assigned".
2. **Given** the same activity, **Then** the grid shows Mar, Apr and May 2026 with each line item's
   amounts in USD and a totals row; **When** the user switches to **Quarterly**, **Then** the months are
   summed into Q1 and Q2 2026.
3. **Given** line items adding up to 120,500.00 USD, **Then** the system row is red — "Line items exceed
   the activity's planned cost by 500.00 USD" — and the activity's Ready to submit panel lists it.
4. **Given** an activity without line items, **Then** the section says "No line items yet — split this
   activity's planned cost into the costs you will pay" with **Add line item**, and the system row shows
   the whole planned cost as unassigned.
5. **Given** a *Created* activity, **Then** line items with "Request SWBS code from GSAP" on show "Waiting
   for activity approval"; once the activity is *Approved* they show "Ready to send to GSAP"; with the
   request off they show "Not requested".

---

### User Story 2 - Add a line item with sensible defaults (Priority: P1)

The user chooses **Add line item**. A full page opens under the activity, already filled in: name, owner,
currency USD, planned cost equal to what is still unassigned, spread and finance budget from the plan,
SWBS request on. The user picks a spend type (the GL code follows), checks the finance budget item and
saves.

**Why this priority**: Shell's line-item form is the longest in Hive9; good defaults make it a few clicks
while keeping the finance coding complete.

**Independent Test**: On an activity with 20,000.00 USD unassigned, add a line item by choosing only a
spend type and saving; confirm the unassigned balance becomes 0.00 USD.

**Acceptance Scenarios**:

1. **Given** "Launch film" (owner Priya Raman, 20,000.00 USD unassigned) in a 2026 plan whose default
   spread is Evenly and default finance budget "2026 Lubricants Budget", **When** the user chooses **Add
   line item**, **Then** the page opens with name "Launch film – line item", owner Priya Raman, currency
   USD, planned cost 20,000.00, spread Evenly, budget year 2026, the items of "2026 Lubricants Budget"
   listed first and **Request SWBS code from GSAP** on — each labelled with its source — and a summary bar
   "Activity planned cost 120,000.00 USD · Unassigned now 20,000.00 USD · After saving 0.00 USD".
2. **Given** the user selects EUR and types 10,000, **Then** the amount shows "= 10,869.57 USD at 0.92 EUR
   per USD" and the summary bar "After saving 9,130.43 USD".
3. **Given** no spend type, **Then** GL code is disabled with "Choose a spend type first — GL codes depend
   on it"; **When** the user picks a spend type with one GL code, **Then** that code is filled in with
   "The only GL code for this spend type"; with several, only those are offered.
4. **Given** a chosen GL code, **When** the spend type changes to one it does not fit, **Then** before
   saving NorthStar lists "GL code '7250080 – Advertising digital' will be cleared — it doesn't fit 'BTL –
   Event'".
5. **Given** an empty name, a name already used in the activity and planned cost "-50", **When** the user
   saves, **Then** nothing is saved and all three problems appear at once, in a summary and by the fields.
6. **Given** a name and planned cost but no spend type, GL code or finance budget item, **When** the user
   saves, **Then** the line item is created with an ID (e.g. L-2001), the toast says "Line item 'Agency
   fee' created", the activity page shows the new row highlighted with "3 fields needed".
7. **Given** **Save and add another**, **Then** a new form opens whose planned cost is the new unassigned
   balance.

---

### User Story 3 - Spread a line item over time (Priority: P1)

Each line item says when its money will be spent: evenly, all in the first month, all in the last month,
or custom amounts per month — including months before or after the activity, such as a deposit. A
preview shows the result, and spreads follow the activity when its dates change.

**Why this priority**: Budget-by-month views and finance need to know when money is spent; Hive9's
C / E / F / L codes were cryptic.

**Independent Test**: On a three-month activity, set a 10,000.00 line item to each method in turn and
check the preview; then move the activity's dates by one month and check the spreads.

**Acceptance Scenarios**:

1. **Given** an activity of 1 Mar – 31 May 2026 and a line item of 10,000.00 EUR, **When** the user picks
   **Evenly**, **Then** the preview shows "Mar 2026 · 3,333.33 EUR", "Apr 2026 · 3,333.33 EUR", "May 2026
   · 3,333.34 EUR"; **All in first month** puts 10,000.00 in Mar and **All in last month** in May.
2. **Given** **Custom**, **Then** a month grid for Mar–May starts from the current spread, with **Add month
   before** and **Add month after**; **When** the user adds Feb 2026 and types 2,000.00, **Then** Feb is
   marked "Outside the activity's dates — allowed, e.g. a deposit" and the total reads "12,000.00 of
   10,000.00 EUR spread — 2,000.00 EUR too much".
3. **Given** that custom spread, **When** the user saves, **Then** it is saved, the line item's
   completeness lists "Spread doesn't add up to the planned cost" and the activity's Ready to submit panel
   lists it; **Use month total as planned cost** sets the planned cost to 12,000.00 in one step.
4. **Given** an ongoing activity, **When** the user picks **Evenly**, **Then** the note says "This activity
   is ongoing (no end date), so 'Evenly' puts everything in the first month — choose Custom to spread it"
   and the preview shows everything in the start month.
5. **Given** line items spread Evenly, All in first month and All in last month, **When** the activity's
   dates change to 1 Apr – 30 Jun 2026, **Then** they are re-spread over Apr–Jun automatically, while a
   Custom line item keeps its amounts with Mar marked as outside the activity's dates.
6. **Given** a Custom line item, **When** the user switches it to Evenly, **Then** before saving NorthStar
   says the typed month amounts will be replaced.

---

### User Story 4 - View, edit, duplicate and delete a line item (Priority: P1)

Selecting a line item opens its page with **Edit** and **More** (Duplicate, View history, Delete). When the
activity is submitted or approved, changing a line item's cost or currency, or adding or removing a line
item, sends the activity back for approval — after the user confirms.

**Why this priority**: Costs change all year; approval must stay meaningful without blocking coding fixes
(decision D-013).

**Independent Test**: On an approved seed activity, change a line item's GL code (status unchanged), then
its amount (confirmation, activity becomes *Created*), duplicate it and delete the duplicate with Undo.

**Acceptance Scenarios**:

1. **Given** the row "Agency fee", **When** the user selects it, **Then** its page opens with the
   breadcrumb (… › Launch film › Agency fee), name, line-item tag, completeness and SWBS request chips,
   L-2001, owner, "50,000.00 USD", spread, who created and last changed it, **Edit** and **More**; empty
   values show "—", the SWBS code says "Filled by GSAP later" and "From finance (coming later)" is
   collapsed.
2. **Given** an *Approved* activity, **When** the user changes only a line item's GL code, finance budget
   item, spread or SWBS request and saves, **Then** it saves without a dialog, the activity stays
   *Approved*, and the change is logged.
3. **Given** a *Submitted* or *Approved* activity, **When** the user changes a line item's planned cost
   from 50,000.00 to 55,000.00 USD (or its currency), or adds or duplicates a line item, **Then** "This
   activity will need approval again" lists the change with **Save and restart approval** and **Keep
   editing**; on confirmation it is saved and the activity becomes *Created*.
4. **Given** **More › Duplicate**, **Then** "Agency fee (copy)" is created in the same activity with all
   values and a new ID, the unassigned balance updates (in red if it turns negative), and the copy opens.
5. **Given** **More › Delete**, **Then** the dialog asks "Delete line item 'Agency fee'?", says "The
   activity's unassigned planned cost goes up by 50,000.00 USD", "It moves to the recycle bin; you can
   restore it" and, for a submitted or approved activity, "This activity will need approval again", with
   **Cancel** focused and a red **Delete line item** button; afterwards the activity page shows a toast
   with **Undo**, which brings the line item back with the same ID and values.

---

### Edge Cases

- **Rounding:** typed amounts with more than two decimals are rounded half-up to the cent and the field
  says so ("Rounded to 1,234.57"); conversion rounds half-up (domain model §4.3); an even spread puts the
  leftover cent in the last month.
- **Rounding leaves cents unassigned:** when a local-currency line item cannot match the balance exactly,
  the system row adds "If a currency rate leaves a few cents unassigned, adjust the activity's planned
  cost instead."
- **Zero amounts:** a 0.00 line item is allowed and spreads 0.00 to every month. When nothing is left to
  assign, a new line item's planned cost starts empty with "Nothing left to assign — the activity's
  planned cost is fully assigned".
- **Negative balance:** warned in the form's summary bar and on the activity page, never blocks saving,
  blocks submitting (D-016). Negative planned costs or month amounts are refused.
- **Inactive currency or finance budget item:** kept and labelled "(inactive)"; the currency still
  converts at its last rate, while the item is listed as a problem: "Choose an active finance budget item
  — 'Europe – Brand' is no longer active."
- **Budget year without items:** "No finance budget items for 2027 yet — an administrator adds them in
  Settings."
- **Rate change in Settings:** USD amounts update everywhere at once without re-approval (FR-013,
  D-027); an activity whose balance turns negative shows it in red.
- **The activity has no dates yet:** the preview says "Not spread yet — add the activity's dates to see
  the spread", and the activity's grid shows the amount in a "No dates yet" column until dates are set.
- **The activity is deleted while a line item is being edited:** saving says "This activity was deleted —
  restore it from the recycle bin to keep working on its line items."
- **Many months** (long or ongoing activities): the grid scrolls sideways inside its card with the line
  item names kept in view.

## Requirements *(mandatory)*

### Functional Requirements

**Line items section on the activity page**

- **FR-001**: The activity page's Line items section MUST list one row per line item, in creation order,
  with name (link), ID, currency, planned cost in local currency, planned cost in USD, spread method,
  completeness chip and SWBS request chip (domain model §2.4).
- **FR-002**: Below the rows, the system row "Unassigned planned cost (Sys_Gen_Balance)" MUST show the
  balance of domain model §4.3 with a meter "<assigned> of <planned cost> USD assigned", the message "<X>
  not yet assigned to a line item", "All of the planned cost is assigned" at 0.00, or, in red, "Line items
  exceed the activity's planned cost by <X>"; an ⓘ explains the formula.
- **FR-003**: The section MUST show a spread grid with one column per month of the activity plus any month
  used by a custom spread (marked "Outside the activity's dates"), a **Monthly / Quarterly** switch
  (quarters add up their months; display only), and a totals row per period and overall, in USD.
- **FR-004**: With no line items, the section MUST show the empty state with **Add line item**; the system
  row then shows the whole planned cost as unassigned.
- **FR-005**: Money MUST follow design system §4; USD columns MUST be shown in the user's display currency
  (feature 003) when one is chosen, labelled with that currency.

**Creating a line item**

- **FR-006**: **Add line item** MUST open a full page "New line item" under the activity: a line item is
  a long record (design system §6.2) with seven sections and a month grid that needs the page's width. A
  summary bar MUST keep the activity's context: its planned cost, the unassigned balance now, and the
  balance after saving (red when negative).
- **FR-007**: Sections MUST follow design system §7 in this order: Basics (name, owner, description) →
  Cost and currency → Spread over time → Finance coding (spend type, GL code, spend category, contracted
  spend, contracted amount and any other line-item Shell attributes from feature 002) → Link to finance
  budget → GSAP request → "From finance (coming later)", collapsed.
- **FR-008**: The form MUST pre-fill, each labelled with its source: name "<activity name> – line item"
  (with " 2", " 3"… when taken); owner = activity owner; currency USD; planned cost = current unassigned
  balance when positive (otherwise empty, with a note); spread = the plan's default line-item spread;
  budget year = the plan's year; finance budget item as in FR-015; **Request SWBS code from GSAP** on
  (field catalogue §5).
- **FR-009**: Saving MUST require only the fields marked **Required** in field catalogue §5 (name,
  currency, planned cost); "needed to submit" fields MAY stay empty (D-009). Names MUST be unique within
  the activity, ignoring letter case and surrounding spaces; planned cost MUST be 0.00 or more.
- **FR-010**: On save, the line item MUST get a permanent ID (L-…, D-019) and be logged; NorthStar MUST
  show a toast naming it and return to the activity page with the new row highlighted. **Save and add
  another** MUST open a new form whose planned cost is the new unassigned balance.

**Currency and conversion**

- **FR-011**: **Currency** MUST offer the active currencies (feature 002); an inactive currency already on
  a line item MUST be kept, labelled "(inactive)" and still converted at its last rate. The USD amount
  MUST be calculated as in domain model §4.3 and shown live under the amount as "= 10,869.57 USD at 0.92
  EUR per USD", labelled "Calculated".
- **FR-012**: While the planned cost still holds the pre-filled balance, changing the currency MUST
  convert it so its USD value still equals the balance, saying so ("Changed to 18,400.00 EUR = 20,000.00
  USD").
- **FR-013**: USD amounts MUST always be calculated with the currency's current rate in Settings (decision
  D-027). When an administrator changes a rate, the USD amounts, unassigned balances and roll-ups of every
  affected line item MUST update at once; this MUST NOT send any activity back for approval (domain model
  §2.5). If the new rate makes an activity's unassigned balance negative, the activity shows the red
  balance and its Ready to submit panel lists it (D-016).

**Finance coding and finance budget link**

- **FR-014**: Spend type, GL code, spend category and contracted spend MUST appear as line-item Shell
  attributes configured in Settings (feature 002) with their markings. GL code options MUST depend on the
  spend type (R11): disabled with a hint until a spend type is chosen, filled in (saying so) when only one
  fits, and listed for clearing before saving when the spend type changes. Contracted amount MUST be an
  optional amount of 0.00 or more in the line item's currency.
- **FR-015**: **Finance budget year** MUST offer the plan's year and every year with active finance budget
  items. **Finance budget item** MUST offer that year's active items, searchable, grouped by budget, with
  the plan's default finance budget first; the item MUST be pre-selected (saying so) when that budget has
  exactly one active item (R12). Changing the year MUST list the item that will be cleared.
- **FR-016**: An inactive finance budget item already linked MUST be kept and labelled "(inactive)", and
  MUST count as a problem to fix before the activity can be submitted.

**Spread over time**

- **FR-017**: **Spread over time** MUST offer Evenly · All in first month · All in last month · Custom
  (design system §8) with the rules of domain model §4.4, and a read-only preview of the month amounts for
  the first three ("Mar 2026 · 3,333.33 EUR"); without activity dates the preview MUST say "Not spread
  yet — add the activity's dates to see the spread", and spreads apply once dates exist.
- **FR-018**: **Custom** MUST show a month grid of the activity's months, starting from the current spread
  (an even spread for a new line item), with **Add month before** and **Add month after**; months outside
  the activity's dates are marked as allowed. A running total MUST show "<sum> of <planned cost> spread —
  <X> left" or "… too much", with **Use month total as planned cost**. Month amounts MUST be 0.00 or more.
  Switching away from Custom MUST say, before saving, that the typed amounts will be replaced.
- **FR-019**: A custom spread that does not add up MUST NOT block saving but MUST count towards the line
  item's completeness and block submitting the activity (feature 006).
- **FR-020**: Spread amounts MUST be held in the line item's currency. The activity's grid MUST show each
  month in USD, converted at the line item's rate, with any rounding cent placed in the last month that
  has an amount, so the months add up exactly to the line item's USD planned cost.
- **FR-021**: For an ongoing activity, Evenly and All in last month MUST behave like All in first month
  and say so; the custom grid MUST run from the start month to December of the plan's year, extendable.
- **FR-022**: When the activity's dates change (including becoming ongoing), line items spread Evenly, All
  in first month or All in last month MUST be re-spread automatically; custom amounts MUST be kept, with
  months now outside the dates marked. The activity's save confirmation (feature 006) MUST say how many
  line items are re-spread and how many custom spreads are kept. When a line item's planned cost or
  currency changes, non-custom spreads follow it and custom amounts are kept, flagged if they no longer
  add up.
- **FR-023**: The unassigned balance, the meter, the grid and the activity's Ready to submit panel MUST
  reflect every change to a line item or to the activity's planned cost as soon as it is saved.

**GSAP request and finance values**

- **FR-024**: **Request SWBS code from GSAP** MUST be a yes/no choice, on by default and editable in any
  activity status, with the status chip of domain model §2.4 on the row and page; the SWBS code MUST show
  "Filled by GSAP later". NorthStar MUST NOT send anything to GSAP in this phase (D-008).
- **FR-025**: "From finance (coming later)" MUST show total actuals, % of actual cost, actuals at monthly
  rate, total invoiced and total purchase requests as "—" with "Filled when the Finance module and GSAP
  are connected" (domain model §4.6).

**Viewing, editing, duplicating and deleting**

- **FR-026**: Selecting a line item MUST open its details page (design system §6.3): breadcrumb, name,
  line-item tag, completeness and SWBS request chips, ID, owner, planned cost (local and USD), spread, who
  created and last changed it, and the actions **Edit** (primary) and **More** (Duplicate, View history,
  Delete last). Empty values show "—". **View history** opens the line item's history (feature 011;
  "Coming soon" until then).
- **FR-027**: **Edit** MUST follow design system §6.4: same layout, **Save** never disabled, all problems
  at once, unsaved changes protected, every change logged with old and new values.
- **FR-028**: The completeness chip MUST show "All filled in" or "<n> fields needed", counting the empty visible
  needed-to-submit fields (field catalogue §5), a custom spread that does not add up and an inactive
  finance budget item; choosing it MUST list them with links. These problems MUST also appear in the
  activity's Ready to submit panel (feature 006).
- **FR-029**: **Duplicate** MUST create a copy in the same activity with all values, including the spread,
  a name made unique with " (copy)", " (copy 2)"…, a new ID and no GSAP or finance values (R13), then open
  it.
- **FR-030**: **Delete** MUST follow design system §6.5 (dialog naming the line item and the effect on the
  unassigned balance, **Cancel** focused, red **Delete line item**), return to the activity page and offer
  **Undo** for at least 10 seconds; Undo restores the same ID and values.

**Changes that restart approval**

- **FR-031**: When the activity is *Submitted* or *Approved*, adding, duplicating or deleting a line item,
  or changing its planned cost or currency, MUST first show "This
  activity will need approval again" with the change, and **Save and restart approval** / **Keep
  editing** (in the delete dialog, as an extra line). On confirmation the activity MUST return to
  *Created* as defined in feature 006 and domain model §2.5 (D-013).
- **FR-032**: All other line-item edits (name, owner, description, spread, finance coding, finance budget
  link, SWBS request) MUST NOT change the activity's status.
- **FR-033**: Undoing a line-item delete MUST NOT restore an approval the delete ended; restoring a line
  item into a *Submitted* or *Approved* activity counts as adding one.

### Key Entities *(include if feature involves data)*

- **Line item**: one cost of an activity — the fields of field catalogue §5, the rate used for its USD
  amount, readable ID, who created and last changed it. Belongs to exactly one activity.
- **Month amount**: the part of a line item's planned cost in one calendar month, in the line item's
  currency; months may lie outside the activity's dates (custom spreads only).
- **Unassigned planned cost** (derived): the activity's planned cost minus its line items' USD planned
  costs (domain model §4.3).
- **SWBS request status** (derived): Not requested · Waiting for activity approval · Ready to send to GSAP.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With its defaults accepted, a line item is added in at most 5 steps (Add line item, spend
  type, GL code when not filled in, finance budget item when not pre-selected, Save), so an activity with
  one line item takes at most 15 steps (design system §10).
- **SC-002**: For 100 % of tested amounts, methods and month counts, the month amounts of a line item spread
  Evenly, All in first month or All in last month add up exactly, to the cent, to its planned cost — in
  local currency and in USD.
- **SC-003**: 100 % of currency conversions match domain model §4.3 to the cent, and the converted amount
  is visible before saving.
- **SC-004**: The unassigned planned cost shown always equals the activity's planned cost minus its line
  items' USD amounts, to the cent, and users read it on the activity page without opening another screen.
- **SC-005**: Every problem with a line-item form is shown in a single save attempt, and 100 % of line-item
  problems that block submitting appear in the activity's Ready to submit panel.
- **SC-006**: In a test with at least 5 first-time users, at least 90 % create, edit, duplicate and delete
  a line item without help, each task within 5 minutes.
- **SC-007**: 100 % of the line-item changes listed in domain model §2.5 on a submitted or approved
  activity are confirmed first and return it to *Created*; 0 % of other line-item edits change its status.
- **SC-008**: 100 % of read-only and locked line-item fields say where their value comes from or why.

## Assumptions

- Activities, the activity page and its Ready to submit panel are feature 006; this feature fills the Line
  items section and supplies the line-item checks.
- A full page is used for creating, viewing and editing a line item (design system §6.2 lists line items
  as long records); the summary bar keeps the activity's context visible.
- Quarterly is a display choice; amounts are always entered per month. The unassigned balance has no
  monthly spread; how it appears in budget-by-month views is decided by feature 010.
- Line items are not rows in the plan tree (features 004–006); they appear on the activity page and in
  the All work views (feature 010).
- The plan's default finance budget (field catalogue §1 #9) is a whole budget, so it narrows and orders
  the item list rather than choosing an item, unless the budget has a single active item.
- All currencies use two decimals in this phase.
- Sending SWBS requests to GSAP and receiving codes, purchase requests and related primary activities
  are future features (D-008).
- No permissions in this phase: every signed-in user can do everything here (D-004); every change records
  the actor.
