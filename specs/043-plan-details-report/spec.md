# Feature Specification: Plan Details Report

**Feature Branch**: `feature/043-plan-details-report`

**Created**: 2026-10-30

**Status**: Draft

**Input**: User description: "Generated from Settings: for each active Shell attribute of each level, a chart of
records per value, a chart of planned cost per value and a count of records with the attribute *Not filled in*;
plus Planned cost entered, Goal entered and, once 046 exists, Is integrated. Level tabs Plan · Campaign ·
Initiative · Activity. Stories: US1 attribute spread and completeness at the activity level (P1) · US2 the other
levels (P2) · US3 export (P2)." (GitHub issue #36, `docs/sdd/measure-issues.md` §#043)

**References**: Constitution 4.8.0 — Current Phase & Scope (Measure wave A, D-366), Principles IV (the server
works out every figure), VII (scope isolation), F3 (state in the address), F6 (every chart has a table) ·
Hive9 functional analysis §E3.9.0 (shared chrome), §E3.9.1 (the self-configuring three-column pattern),
§E3.9.2 (landing widgets), §E3.9.3 (Custom attributes – Tactic), §E3.0.1.1 (*Not Completed*) · Shell's client
spec 019 US2, FR-026–FR-033 (Plan Details), FR-018, FR-020, FR-021 (currency, empty states, table view) ·
`docs/product/measure-metrics.md` §1 (Completeness, Records by attribute value, Planned cost entered · Goal
entered) · `docs/product/domain-model.md` §2.1–§2.3 (statuses), §2.6 (completeness), §3 (needed fields), §4.2
(planned cost), §4.5 (goals), §6 (custom attributes) · features 002 (Shell attributes), 006 (activities),
014 (scope), 016 (Shell-attribute filters and Breakdown rules, FR-039–FR-041; D-270, D-336), 036 (Plan details
completeness panel), 039 (the report frame: filters, Period, Show, Currency, saved views, widget table and
export; D-367, D-368).

## Clarifications

Answers given while specifying are recorded here as `043-Cn` with their reasoning; the integrator moves them
into the decision log at the end of wave A (D-369).

### Session 2026-10-31

- **043-C1** Q: Does the Activity tab show built-in widgets (activity type, status, start month) besides the
  Shell-attribute rows? → A: Yes, as they are planning data MAPP holds (developer's answer: "include them if it
  is in the planning"). *Why*: client 019 US2 and FR-026 and the observed Hive9 landing page (FA §E3.9.2) show
  them; activity type, status and start date are fields of the activity record, so no new data is needed.
- **043-C2** Q: How does a record count when a Shell attribute is hidden on it by its business-unit condition?
  → A: As its own value **Does not apply**, outside Not filled in and the filled-in per cent. *Why*: hidden
  attributes never block publishing or submitting (domain model §3), so counting them as missing would flag
  Mobility records for Lubricants fields; Not filled in plus Does not apply still equals All work's Breakdown
  Not filled in group (016), so the totals reconcile.
- **043-C3** Q: What does the row of a text, long text or number attribute show? → A: Only the filled-in widget.
  *Why*: free values have no groups to count, and All work's Breakdown also offers only choice and yes/no
  attributes (016 FR-040); completeness still answers Shell's compliance question.
- **043-C4** Q: Who adds the **Is integrated** widget (client 019 FR-026, FR-027)? → A: Not 043; 046 adds it in
  wave B as a recorded exception to folder ownership, like its Performance slot on the activity page. *Why*: it
  needs 046's data sources; the integrator confirms the exception with lane B (build guide §4.1).

## User Scenarios & Testing *(mandatory)*

The report is **read-only**: nothing on it changes a record and opening it writes nothing except what 039's
saved views write. Its scenarios are the states a widget can be in — data, empty, no access, no financial
access, filters exclude everything — not create, edit or delete.

Every scenario runs inside 039's report frame. The frame's filter panel, Period, Show, Currency, saved views,
"as of" line, widget table view, widget export and empty-state wording are 039's and are not redefined here;
this spec says only what the Plan details report puts inside the frame.

### User Story 1 - See how activities spread over each attribute and how complete they are (Priority: P1)

A planner opens **Reports › Plan details** for the plan year. The **Activity** tab shows summary cards, a
section of built-in fields (Activity type, Status, Start month, Planned cost entered, Goal entered) and then,
for every active activity-level Shell attribute in its Settings display order, one row of three widgets:
**Activities by <attribute>**, **Planned cost by <attribute>** and **<attribute> filled in**. The planner sees at
a glance where money is concentrated and which attributes are left empty, and opens the activities behind a
value in All work to fix them.

**Why this priority**: Activities are the records that are approved and carry the money; Shell's compliance
question ("do my activities have their segmentation filled in?") and the first diagnostic for empty
performance reports (Goal entered, FA §E3.9.2) are both answered at this level.

**Independent Test**: On the demo data, open Plan details for the current year as a planner with one team in
scope; check the cards, the built-in section and one attribute row of each kind (single choice, multiple
choice, % split, yes/no, text) against counts worked out by hand; open "Not filled in" of one attribute and
land in All work with the matching filter.

**Acceptance Scenarios**:

*Data*

1. **Given** 28 activities in scope, 26 with a planned cost above zero, **When** the planner opens the Activity
   tab, **Then** the cards show Activities 28, Planned cost (their sum in the display currency) and All filled in
   *n* of 28 with its whole per cent, and **Planned cost entered** shows Yes 26 · No 2.
2. **Given** none of those activities has a goal, **Then** **Goal entered** shows Yes 0 · No 28.
3. **Given** the single-choice attribute Company code with 12 activities on GB01, 10 on CN04 and 6 empty,
   **Then** **Activities by Company code** shows GB01 12, CN04 10, Not filled in 6; **Planned cost by Company
   code** shows each value's planned cost and a Not filled in amount; **Company code filled in** shows Filled in
   22 · Not filled in 6 (79 %).
4. **Given** the % split attribute Country / OU and an activity of 10,000.00 USD split Germany 60 % / France
   40 %, **Then** the activity counts once under Germany and once under France, and its planned cost adds
   6,000.00 to Germany and 4,000.00 to France (016 FR-041).
5. **Given** the multiple-choice attribute Channel / audience and an activity with External and Internal,
   **Then** it counts in both values with its whole planned cost in each, the card total counts it once, and the
   row says "Activities with several values count in each group; the total counts them once."
6. **Given** a text or number attribute (for example FIT4 ID), **Then** its row shows only **FIT4 ID filled
   in**, because free values have no groups to count.
7. **Given** **Activity type** and **Status**, **Then** the built-in section shows activities and planned cost
   by activity type and activities by status, with every status of domain model §2.3 that has activities.
8. **Given** Period set to Quarters, **Then** **Activities by start month** shows activities per quarter of the
   plan year by start date, with **No dates** for activities without a start date.
9. **Given** the planner selects **Not filled in** on **Activities by Company code**, **Then** All work opens
   with the filter "Company code: Not filled in" and the frame's other filters, listing the six activities.
10. **Given** a plan in the Chinese yuan and the display currency euro, **Then** every planned-cost figure is in
    euro at the current rate and names its currency.
11. **Given** the Lubricants-only attribute Product launch, 20 activities in Lubricants plans (15 filled in, 5
    empty) and 8 in Mobility plans, **Then** **Activities by Product launch** shows its values, Not filled in 5
    and Does not apply 8, and **Product launch filled in** shows Filled in 15 · Not filled in 5 (75 %), with
    "8 activities where it does not apply are not counted" (043-C2).

*Empty*

12. **Given** the person may see plans for the year but none has an activity, **Then** the Activity tab shows
    039's "No data yet" state saying activities create this report's data, and links to the plans.
13. **Given** no plans in scope for the chosen year, **Then** the tab shows 039's "No plans you may see for
    the year" state.
14. **Given** no active activity-level Shell attribute in Settings, **Then** the built-in section shows as
    usual and the attribute section says "No activity attributes are set up in Settings."

*No access*

15. **Given** a person without "See reports", **Then** Plan details is not offered and its address answers as
    039 answers a refused report.
16. **Given** a plan outside the person's scope, **Then** its activities count nowhere and no value, card or
    export names it.

*No financial access*

17. **Given** a person who may see plans but holds no financial-data permission in any of them, **Then** every
    planned-cost figure still shows, because planned cost is planning data that follows "View plans" (D-237);
    the report shows no financial-access notice.

*Filters exclude everything*

18. **Given** filters that match no activity (for example a team with no plans this year), **Then** every
    widget shows 039's "The filters exclude everything" state with its one-click reset, and the cards show 0.

---

### User Story 2 - The same picture for plans, campaigns and initiatives (Priority: P2)

The planner switches to the **Plan**, **Campaign** or **Initiative** tab. Each shows the same layout for the
records and Shell attributes of that level: cards, a built-in section with records by status, and a
three-widget row per active Shell attribute of that level.

**Why this priority**: Business unit, Cluster, Region and the plan-level attributes sit above activities; the
completeness of plans and campaigns blocks publishing (domain model §3). Valuable, but the activity level
answers Shell's main question first.

**Independent Test**: On the demo data, open the Campaign tab, compare Campaigns by Cluster and Planned cost by
Cluster against All work's Breakdown by Cluster for the same filters; switch tabs and check the address keeps
the tab.

**Acceptance Scenarios**:

1. **Given** the Plan tab and the plan-level attribute Business unit, **Then** **Plans by Business unit**,
   **Planned cost by Business unit** (each plan's planned cost, domain model §4.2) and **Business unit filled
   in** show the plans in scope.
2. **Given** the Campaign tab and Cluster, **Then** each value of **Planned cost by Cluster** and its total equal
   All work's Breakdown by Cluster to the cent for the same filters, Not filled in and Does not apply together
   equalling Breakdown's Not filled in (016 FR-040, 043-C2).
3. **Given** the Plan tab, **Then** the built-in section shows plans by status (Draft, Published); on the
   Campaign and Initiative tabs it shows records by Approval progress and by Timing (domain model §2.2).
4. **Given** the planner switches from Activity to Initiative, **Then** the tab is in the page address, so Back
   returns to Activity and a saved view or shared link opens on Initiative.
5. **Given** a level with records in scope but no active Shell attribute, **Then** the tab says "No
   <level> attributes are set up in Settings" below its built-in section.
6. **Given** filters that match no record at the chosen level, **Then** the tab shows 039's "The filters
   exclude everything" state; the other tabs apply the same filters to their own level.
7. **Given** a person who may see one of two plans, **Then** the Plan tab counts one plan and the other tabs
   count only what lies below it.

---

### User Story 3 - Export a widget (Priority: P2)

The planner exports one widget's table — for example **Planned cost by Country / OU** — as CSV or Excel to work on
it or attach it to a review.

**Why this priority**: Exports are how Shell reviews completeness offline (client 019 FR-014, SC-011); the frame
already gives every widget an export, so the report only supplies the rows.

**Independent Test**: Export one count widget, one planned-cost widget and one completeness widget as CSV and
as Excel and compare each with its table view.

**Acceptance Scenarios**:

1. **Given** a person holding "Export data", **When** they export **Planned cost by Country / OU**, **Then** the
   file holds every value, Not filled in and a total row, the amounts in the display currency with the currency
   named, equal to the table view; the filters and the "as of" time are those 039's export states.
2. **Given** a chart that gathers small values into "Other", **Then** its export still lists every value.
3. **Given** a person without "Export data", **Then** the export action is not offered (039).
4. **Given** a widget in an empty or filtered-to-nothing state, **Then** no export is offered for it.

---

### Edge Cases

- **Inactive attribute**: not shown; its rows return when it is set active again. An inactive *option* still
  carried by records shows as its own value labelled "(inactive)" (domain model §6).
- **Attribute added or renamed in Settings**: the row appears or takes the new label on the next load with no
  change to the report (client 019 FR-028, SC-005); values are identified by attribute and option, never by
  label (FR-030).
- **Hidden by its business-unit condition**: the record counts in **Does not apply**, never in Not filled in or
  the filled-in per cent (043-C2); a row where it applies to no record in view shows Does not apply only.
- **Dependent attribute whose parent value has no options set up**: counts in Not filled in like every empty
  value; its label is unchanged.
- **Yes/no attribute**: values Yes and No; most default to No (field catalogue), so they are rarely Not filled
  in.
- **% split with a 0 % share**: that value does not count the record (016 FR-039).
- **Many values**: a chart shows the ten largest values and gathers the rest into "Other"; Not filled in and
  Does not apply are always shown on their own; the table view and the export list every value.
- **Activity with no planned cost**: counts in every count widget and adds nothing to planned-cost widgets;
  Planned cost entered shows it under No.
- **Activity with a negative unassigned balance**: its planned cost is its activity planned cost (domain model
  §4.2), as on every other screen.
- **Start date outside the plan year**: shown under "Outside <year>" in Activities by start month, so the bars
  add up to the activity count.
- **Goal entered**: an activity has a goal when its INQ goal is above zero, or, with "Set later stages
  myself", any stage goal is above zero (domain model §4.5).
- **Is integrated**: not shown by 043; 046 adds it in wave B (043-C4).

## Requirements *(mandatory)*

### Functional Requirements

**Report and access**

- **FR-001**: The Reports area MUST offer **Plan details** inside 039's report frame, with level tabs
  **Plan · Campaign · Initiative · Activity**, Activity first and the default. The tab MUST live in the page
  address with the frame's filters and be kept by saved views. *(client 019 US2; FA §E3.9.1; D-368)*
- **FR-002**: Plan details MUST be read-only: nothing on it changes a record. *(client 019 FR-022; D-366)*
- **FR-003**: Plan details MUST need "See reports" and MUST count only records of plans in the person's scope;
  a plan outside the scope MUST NOT count or be named anywhere, export included. *(D-367; constitution VII)*
- **FR-004**: Planned cost on this report MUST follow "View plans", not financial-data access, because it is
  planning data. *(D-237, D-301)*

**What each tab shows**

- **FR-005**: Each tab MUST show summary cards for its level: **Records** (the count, named for the level),
  **Planned cost** and **All filled in** — the records whose every needed field is filled in, as a count and a
  whole per cent (domain model §2.6, the figure of 036's completeness panel). *(measure-metrics §1
  Completeness)*
- **FR-006**: The Activity tab MUST show a built-in section: **Activities by activity type**, **Planned cost by
  activity type**, **Activities by status**, **Activities by start month** (per period of 039's Period control,
  with No dates and Outside <year>), **Planned cost entered** (planned cost above zero: Yes · No) and **Goal
  entered** (Yes · No). *(client 019 FR-026, FR-027; FA §E3.9.2)*
- **FR-007**: The Plan tab MUST show plans by status; the Campaign and Initiative tabs MUST show their records by
  Approval progress and by Timing. *(domain model §2.1, §2.2)*
- **FR-008**: Each tab MUST then show one row per **active Shell attribute of its level**, in the Settings
  display order, generated from Settings with no change to the report when attributes change. *(client 019
  FR-028, FR-030; FA §E3.9.1)*
- **FR-009**: For a single choice, multiple choice, multiple choice with % split or yes/no attribute, the row
  MUST show **<Records> by <attribute>** (records per value), **Planned cost by <attribute>** (planned cost per
  value) and **<attribute> filled in** (Filled in · Not filled in, with the whole per cent filled in). Each
  value widget MUST include a **Not filled in** value and, where the attribute is hidden on some records, a
  **Does not apply** value. *(measure-metrics §1 Records by attribute value; 043-C2)*
- **FR-010**: For a text, long text or number attribute the row MUST show only **<attribute> filled in**.
  *(043-C3)*
- **FR-011**: Records and planned cost MUST be attributed by 016's Breakdown rules: a record counts under the
  value it carries; a multiple choice without split counts the record and its whole planned cost under each
  value and the row says "<Records> with several values count in each group; the total counts them once"; a %
  split counts the record under each value with a share above 0 and divides its planned cost by the shares,
  half up per value with the remainder on the last value. *(016 FR-041; D-270, D-336; client 019 FR-029)*
- **FR-012**: A record whose attribute is empty or carries no option MUST count as **Not filled in**. A record on
  which the attribute is hidden by its business-unit condition MUST count as **Does not apply**, and MUST be left
  out of the filled-in count and per cent, which state how many such records they leave out; Not filled in and
  Does not apply together MUST equal All work's Breakdown Not filled in group. *(016 FR-039; FA §E3.0.1.1;
  domain model §3; 043-C2)*
- **FR-013**: A record's planned cost MUST be its planned cost as every other screen shows it (domain model
  §4.2), converted to the display currency at the current rate, each amount with its currency.
  *(measure-metrics §1 Planned cost; D-303; client 019 FR-018)*
- **FR-014**: The value widgets of a Shell attribute, activity type and status MUST let the person open the
  records behind a value — Not filled in included — in All work with the matching filter and the frame's other
  filters. *(016 FR-039, FR-040; client 019 FR-015)*

**Frame, states and export (039)**

- **FR-015**: The report MUST use 039's filters, Period, Show, Currency, saved views, "as of" line, widget table
  view and widget export as 039 defines them, and MUST NOT add filters or controls of its own other than the
  level tabs. *(D-368)*
- **FR-016**: Each widget MUST render one of 039's states: data; no plans you may see for the year; no data yet,
  naming what creates it (activities, or attributes in Settings); filters exclude everything, with the reset.
  *(D-368; client 019 FR-020)*
- **FR-017**: Every widget's table view and export MUST list every value, Not filled in, Does not apply and a
  total row,
  equal to the chart; a chart MAY gather all but its ten largest values into "Other". *(client 019 FR-012,
  FR-014, FR-021; constitution F6)*

### Key Entities

- **Level**: one of Plan, Campaign, Initiative, Activity; decides which records are counted and which Shell
  attributes are profiled.
- **Attribute profile row** (calculated, never stored): for one Shell attribute at one level, the records per
  value, the planned cost per value and the filled-in count; values identified by attribute and option.
- **Built-in distribution** (calculated): records per activity type, status, approval progress, timing or start
  period; Planned cost entered and Goal entered.
- Read from existing records only: plans, campaigns, initiatives, activities and their Shell-attribute values
  (002, 004–006), custom attribute definitions (002), currencies and rates (016). No new stored data.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For a person whose scope holds 50 plans, 5,000 activities and 20,000 line items with 30 active Shell
  attributes, each tab shows all its widgets within 3 seconds for 95 % of opens and filter changes (028 volumes).
- **SC-002**: For every attribute and every filter tested, the planned cost per value equals All work's Breakdown
  by the same attribute to the cent (Not filled in plus Does not apply against Breakdown's Not filled in), and
  the record counts equal hand-counted values, for 100 % of cases.
- **SC-003**: An attribute added in Settings appears on the report on the next load with no other change.
- **SC-004**: A planner finds the activities missing a needed Shell attribute or a goal, and opens them in All
  work, within 3 minutes on the first attempt (client 019 SC-008).
- **SC-005**: Every export equals its table view, value for value; no plan outside the person's scope appears in
  any figure or file.
- **SC-006**: Every tab passes WCAG 2.1 AA with no critical or serious finding, and every chart has a table view
  (client 019 SC-010).

## Assumptions

- **Levels**: four tabs as issue #36 says; line-item attributes (Spend type, GL code) are not profiled here —
  they are in All work's Breakdown and 040's ranked list.
- **Year**: a record belongs to the year of its plan (D-368); Hive9's rolling six-month window and its
  date-overlap rule are not adopted.
- **No financial-access notice**: planned cost needs only "View plans" (D-237); the report shows no committed,
  invoiced or actual figure.
- **"Other" after ten values** is this report's chart rule; 039 does not define one.
- 039's frame, its route rules and its export are in place before 043's web part is built (build guide §3).
