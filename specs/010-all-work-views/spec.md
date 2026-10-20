# Feature Specification: All Work Views

**Feature Branch**: `010-all-work-views`

**Created**: 2026-10-18

**Status**: Draft

**Input**: User description: "All work: grid with roll-ups, filters and search, calendar, budget & cost by month, saved views, export. The sidebar's 'All work' is a cross-plan view of every campaign, initiative and activity."

**References**: Domain model [§1 hierarchy, §2 statuses, §4.2–4.5 money rules, §7 saved view](../../docs/product/domain-model.md) · Field catalogue [§1–§5 fields per level](../../docs/product/field-catalogue.md) · Design system [§2 layout, §4 colours, §5 tree, §6.1 list toolbar, §8 words, §10 usability targets](../../docs/product/design-system.md) · Glossary [All work, Unassigned planned cost, Spread](../../docs/product/glossary.md) · Decisions D-004, D-010, D-011, D-013, D-016 · Open question Q-004 · Hive9 functional analysis §B1 (workspace), §E3.0 (filter panel, saved views) · ZenReach spec §8.1 (view tabs, summary strip, saved views), §8.4–8.6 (grid, Budget & Cost, calendar)

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See all planned work in one grid (Priority: P1)

A marketer, team leader or finance advisor opens **All work** in the sidebar. Below the summary cards
and filter bar are three pill tabs: **Grid · Calendar · Budget & cost**. The Grid shows every plan of the
year as a tree of plans, campaigns, initiatives, activities and line items. Each row has a level bar
and tag, name, ID, status chip, owner, dates and money. Containers add up what is below them, a totals
row closes the grid, and a click opens the record's own page. Nothing is edited here.

**Why this priority**: Hive9 gives no clear cross-plan picture, so Shell users plan in Excel. This grid
is the core of the feature; the other views reuse its rows.

**Independent Test**: On the seed data, expand all, check every roll-up and the totals row against domain
model §4.2–4.3, and open one row of each level.

**Acceptance Scenarios**:

1. **Given** the seed data and a date in 2026, **When** the user opens All work, **Then** the Grid shows
   "FY2026 Lubricants Europe" and "FY2026 Mobility China", with plans, campaigns and initiatives
   expanded and activities collapsed.
2. **Given** any container row, **Then** its planned cost is the sum of the planned cost below it
   (domain model §4.2). The totals row reads "Total · N activities", and its planned cost equals the
   **Planned cost** summary card.
3. **Given** an expanded activity whose line items do not cover its planned cost, **Then** its last row
   reads "Unassigned planned cost (Sys_Gen_Balance)" with the amount, shown in red with a minus sign
   when it is negative. Switching **Unassigned planned cost** off hides only these rows.
4. **Given** any row, **When** the user clicks it or presses Enter, **Then** the record's page opens, and
   coming back restores the filters, expansion and scroll position.

---

### User Story 2 - Narrow down with filters and search (Priority: P1)

One filter bar serves all three views: year, plan, statuses, owner, team, business unit, activity type,
Strategic / AOP plans, **Show levels**, and a search box ("Search name, ID, external name or
description"). Active filters show as removable chips, with **Clear all** and a count ("Showing 12 of 40
activities"). The summary cards follow the filters.

**Why this priority**: Plans hold hundreds of activities, and Hive9's hidden, remembered filters left
users facing empty screens with no idea why (functional analysis §E3.0).

**Independent Test**: On the seed data, apply each filter alone and combined, checking the rows, chips,
count and cards, then search for an activity by ID.

**Acceptance Scenarios**:

1. **Given** the Grid, **When** the user types an activity's ID or part of its name, **Then** only the
   matches remain, their path is expanded, the matching text is highlighted, and the count reads
   "Showing 1 of N activities".
2. **Given** "Activity status: Declined", **Then** only branches that lead to a declined activity are
   shown, the chip "Activity status: Declined ×" appears, and the cards count only the matches.
3. **Given** the owner "Daniel Okoro", **Then** Daniel's records are shown. Their parents appear in
   lighter text, marked "Shown for context", and are not counted.
4. **Given** **Show levels: Initiatives without activities**, **Then** only initiatives with no
   activities are listed, under their plan and campaign, and the count reads "Showing N of M
   initiatives".

---

### User Story 3 - Choose, order and sort columns (Priority: P2)

A **Columns** side panel offers searchable groups: **Standard, Plan, Campaign, Initiative, Activity,
Line item**. Clicking a heading sorts rows within each parent, so the hierarchy stays intact.

**Why this priority**: People need different facts, but the default columns serve most users.

**Independent Test**: Add a column per group, reorder, sort, sign in again; the choice is kept.

**Acceptance Scenarios**:

1. **Given** the chooser, **When** the user ticks **Cluster** under Campaign, **Then** a Cluster column
   appears. It has values on campaign rows only and shows "—" for a campaign with no cluster. Shell
   attributes added later in Settings are offered automatically.
2. **Given** the Grid, **When** the user clicks the **Planned cost** heading, **Then** the children of
   every parent are ordered from lowest to highest (FR-015, D-218), while totals and unassigned rows stay
   in place. A second click reverses the order, and a third restores the default.

---

### User Story 4 - See planned cost by month (Priority: P2)

The **Budget & cost** tab shows the same rows with **Plan budget**, the months (or quarters) of the year
and **Total**. Line items show their spread, unassigned planned cost is spread evenly over the
activity's months, and containers add up. The view is read-only.

**Why this priority**: Finance needs to know *when* money is spent; it replaces Excel phasing sheets.

**Independent Test**: Check every cell of an activity with Evenly, Custom and All-in-first-month line
items plus an unassigned balance against domain model §4.4.

**Acceptance Scenarios**:

1. **Given** an activity from March to May with a 10,000.00 USD line item spread Evenly, **Then** the
   line item shows 3,333.33, 3,333.33 and 3,333.34 under Mar, Apr and May, and 10,000.00 as its Total.
2. **Given** an activity with a planned cost of 12,000.00 and line items of 9,000.00, **Then** an
   unassigned row spreads 3,000.00 evenly over its months, and the activity's Total is 12,000.00.
3. **Given** **Quarterly**, **Then** each quarter is the sum of its months. The footer's grand total
   equals the **Planned cost** card when no activity has a negative unassigned balance.

---

### User Story 5 - See work on a timeline (Priority: P2)

The **Calendar** tab shows the year as a timeline, with months across the top. Campaigns, initiatives
and activities are bars from their start date to their end date, coloured by status with a legend. A
**Today** line marks the date, and clicking a bar opens the record.

**Why this priority**: Team leaders look for overlaps and gaps over time.

**Independent Test**: With the date fixed at 18 Oct 2026, check bars, colours and the Today line.

**Acceptance Scenarios**:

1. **Given** an activity running from 1 Mar to 31 May, **Then** its bar spans March to May in its
   status colour and shows its name and status word when they fit; the name column always shows the
   status chip.
2. **Given** today is 18 Oct 2026, **Then** a line labelled "Today" crosses the timeline at that date.
3. **Given** an ongoing activity, **Then** its bar runs to the end of the year with an open end and the
   label "Ongoing". A record that ends after the year stops at December with "Continues into 2027".
4. **Given** records without dates, **Then** a section "No dates yet (N)" lists them with links.

---

### User Story 6 - Save and reuse my views (Priority: P2)

A **Views** button lists the user's saved views and offers **Save current view…**. A view keeps the
filters, columns, view type and expansion under a name. One view can be the default, and views can be
renamed and deleted.

**Why this priority**: Users return to the same few views, but the page works without them.

**Independent Test**: Save two views, set a default, sign in again; then rename and delete one.

**Acceptance Scenarios**:

1. **Given** filters, columns and the Budget & cost tab, **When** the user saves "Lubricants by month",
   **Then** the toast says "View 'Lubricants by month' saved", and the view is listed as active.
2. **Given** an applied view, **When** the user changes a filter, **Then** the Views button shows
   "Modified" and offers **Update view** and **Save as new view**.
3. **Given** Priya's default view, **When** she next opens All work, **Then** it is applied, and
   **When** Daniel signs in, **Then** he does not see Priya's views.

---

### User Story 7 - Export what I see (Priority: P2)

An **Export** button offers a **Spreadsheet** or a **CSV** file. It holds the view's rows and columns as
filtered, its totals, money in its currency, and a note of every filter used.

**Why this priority**: Finance works in spreadsheets; stating the filters avoids disputes.

**Independent Test**: Export each view with filters on and compare the file with the screen.

**Acceptance Scenarios**:

1. **Given** a filtered Grid, **When** the user exports a spreadsheet, **Then** it has the same rows
   (including rows in collapsed branches), the same columns in order, and the totals. An "About this
   export" sheet lists the view, year, filters, search, Show levels, currency, exporter and time.
2. **Given** the Calendar or Budget & cost view, **When** the user exports a CSV, **Then** the file
   lists the records with their dates and status, or their period amounts, after the same notes.

---

### Edge Cases

- No plans in the year: "No plans in 2026 yet", with a link to the Plans page. Nothing matches the
  filters: "Nothing matches these filters" with **Clear filters**; Export says "Nothing to export —
  clear a filter first."
- An activity without a planned cost shows "—" and counts as zero. A sum of only empty values shows "—".
- Line items in EUR or GBP are summed in USD and then converted, so totals do not drift through
  rounding.
- A saved view refers to a plan, person or option that no longer exists: the rest is applied, and a
  message names what was left out.
- A record that someone else deleted is clicked: "This activity was deleted — it is in the recycle bin",
  and the view refreshes. Other changes appear on the next filter or view change, or through
  **Refresh** ("Updated at 14:05").
- Inactive types, options and people still appear, labelled "(inactive)". Long names are cut with an
  ellipsis and shown in full on hover and focus.

## Requirements *(mandatory)*

### Functional Requirements

**Page and summary cards**

- **FR-001**: NorthStar MUST provide an **All work** page in the sidebar, with the page header of design
  system §2 and the pill tabs **Grid · Calendar · Budget & cost**. The views MUST show the same records
  for the same filters, and switching views MUST keep the filters, search and expansion.
- **FR-002**: The page MUST NOT create, edit or delete records. Changes are made on each record's page
  (features 004–007), and selecting rows belongs to feature 012.
- **FR-003**: Six summary cards MUST follow every filter and search change. Each has an ⓘ tip giving its
  rule and shows money in compact form, with the full value on hover and focus:
  - **Campaigns**, **Initiatives** and **Activities**: the number of matching records of that level;
  - **Planned cost**: the sum of the matching activities' planned cost;
  - **Plan budget**: the sum of the budgets of the plans shown;
  - **Unassigned planned cost**: the sum of the matching activities' positive balances, plus "N
    activities have line items above their planned cost" in red when any balance is negative.

**Filters and search**

- **FR-004**: The filter bar MUST offer the filters below, each tested on the level shown.

  | Filter | Choices | Tested on |
  |---|---|---|
  | Year | One year: years with plans, plus the current year (default) | Plans |
  | Plan · Plan status | Plans of the year (several) · Draft, Published | Plans |
  | Campaign / initiative status | Approval progress and Timing values (domain model §2.2) | Campaigns, initiatives |
  | Activity status | The six activity statuses (domain model §2.3) | Activities |
  | Owner | People who own something in the year's plans (several) | Every level |
  | Team · Business unit | Teams, with **My teams** first · options of the plan's Business unit | Plans |
  | Activity type | Activity types (several) | Activities |
  | Strategic / AOP plans | Include (default) · Only · Exclude | Plans |
  | Show levels | Everything (default) · Complete branches only · Campaigns without initiatives · Initiatives without activities | Branches |

- **FR-005**: A record **matches** when it meets every filter tested on its level; a filter tested only
  on lower levels counts as met when a record below it matches. NorthStar MUST show every match and its
  parents. Parents that do not match MUST appear as context rows (lighter text, "Shown for context")
  and MUST NOT be counted in the cards, the count or the totals.
- **FR-006**: Search MUST match the typed text, ignoring letter case, anywhere in the name, ID, external
  name or description of a record at any level, line items included. Matches MUST be highlighted with
  their path expanded, and a matching container shows everything inside it, collapsed.
- **FR-007**: **Complete branches only** MUST keep only branches that reach at least one activity.
  **Campaigns without initiatives** and **Initiatives without activities** MUST list only those
  records, under their parents.
- **FR-008**: Every active filter except the year MUST appear as a removable chip. **Clear all** MUST
  clear every filter but the year. The count MUST read "Showing N of M activities", where M is all
  activities in the year's plans; under the last two Show levels choices it counts campaigns or
  initiatives instead.
- **FR-009**: Filters MUST apply as soon as they change, without an Apply button, and are kept while
  the user moves around NorthStar in a session. A new visit MUST start from the user's default saved
  view or the standard view (current year, no filters, Grid, default columns).
- **FR-010**: The filter bar MUST offer only the filters of FR-004. Resolved: no Shell-attribute
  filters and no "Not filled in" choice in this phase (D-207).

**Grid view**

- **FR-011**: The Grid MUST follow the tree of design system §5. Plans are sorted by name, and children
  appear in their parent's order. Each row has a level bar and tag, its name as a link, its ID, an
  "AOP" tag on strategic plans, and a status chip: plan status, campaign or initiative approval
  progress, activity status, or line-item completeness (domain model §2). Rows open and close one at a
  time (keyboard included) and through **Expand all** and **Collapse all**. Plans, campaigns and
  initiatives start expanded, and activities collapsed.
- **FR-012**: An **Unassigned planned cost** switch, on by default, MUST show the "Unassigned planned
  cost (Sys_Gen_Balance)" row under each expanded activity whose balance is not zero (domain model
  §4.3).
- **FR-013**: Container rows MUST show these roll-ups, calculated as in domain model §4.2–4.5 from the
  matches shown under them: planned cost, line items, unassigned planned cost, activity count, INQ,
  MQL and revenue. When filters hide records below a container, its roll-up cells MUST carry a
  "Filtered" mark whose tip gives the unfiltered value.
- **FR-014**: A totals row MUST read "Total · N activities". Its money sums the matching activities, and
  its Plan budget sums plan rows only, so the totals equal the summary cards.
- **FR-015**: Clicking a column heading MUST sort the children of every parent: ascending, then
  descending, then default. Empty values sort last, and statuses sort in lifecycle order.
- **FR-016**: Clicking a row or pressing Enter MUST open the record's page; an unassigned row opens its
  activity. Returning MUST restore the filters, expansion and scroll position.
- **FR-017**: Money MUST be shown in the user's display currency and labelled with it (decision D-010).
  Line items' local amounts appear only in their own column, with their currency.

**Columns**

- **FR-018**: The column chooser MUST offer a **Standard** group (ID, Level, Owner, Start date, End date,
  Status, Timing, Completeness, Plan budget, Remaining budget, Budget used, Planned cost, Line items,
  Unassigned planned cost, INQ, MQL, Revenue, Created, Last changed). It MUST also offer one group per
  level with that level's fields from field catalogue §1–§5, including every active Shell attribute.
  Fields that Finance or GSAP will fill later are not offered in this phase.
- **FR-019**: The default columns MUST be Name, ID, Status, Owner, Start date, End date, Plan budget,
  Planned cost, Line items and Unassigned planned cost, with **Reset to default columns**. Name is
  always shown, and its tick is locked with "The name column is always shown so you can tell rows
  apart".
- **FR-020**: A level-specific column MUST be blank on other levels and show "—" for an empty value on
  its own level. Split values show their shares ("Germany 60 %, France 40 %"), and long text is cut to
  one line.
- **FR-021**: Columns MUST be reorderable by dragging and with the keyboard. The choice and order MUST
  be remembered for each user.

**Budget & cost view**

- **FR-022**: Budget & cost MUST show the Grid's rows, including line items and unassigned rows, with
  these columns, in order:
  - **Plan budget** (containers only);
  - the months of the year, or its quarters through a **Monthly · Quarterly** switch;
  - **Other years** for Custom amounts outside the year, and **No dates** for the amounts of an
    activity without dates, each shown only when it holds an amount;
  - **Total**.
- **FR-023**: A line item's period amounts MUST come from its spread (domain model §4.4). A positive
  unassigned balance MUST be spread over the activity's months with the Evenly rule, or put in the first
  month of an ongoing activity. A negative balance MUST NOT be spread; the activity row shows "Line
  items exceed the planned cost by X" in red instead.
- **FR-024**: Containers MUST add up their children in each period. A footer shows the period totals,
  the grand total and the Plan budget of plan rows only, and empty cells show "—". The view MUST be
  read-only, with the note "To change how a cost is spread, open its line item"; clicking a row opens
  the record.

**Calendar view**

- **FR-025**: The Calendar MUST show the year as a timeline with months across the top and the Grid's
  rows without line items. Each campaign, initiative and activity is a bar from its start date to its
  end date. Activities are coloured by status, and campaigns and initiatives by approval progress
  (design system §4), with a legend; each row's name shows its status chip.
- **FR-026**: The Calendar MUST show a **Today** line when today falls in the year; an open end and
  "Ongoing" on ongoing activities; "Continues from / into <year>" on bars cut by the year's edges; and a
  "No dates yet (N)" list. Hovering over or focusing a bar MUST show its name, ID, level, status, dates,
  owner and planned cost, and clicking it opens the record.
- **FR-027**: Bars MUST NOT be draggable; their tip says "Change dates on the record's page". The
  reason: changing a submitted or approved activity's dates sends it back for approval (domain model
  §2.5, D-013), and a drag would do that silently, without the warning the record's page gives.
- **FR-028** *(P3, may follow later)*: The Calendar MUST offer a **Timeline · Month** switch. The Month
  layout is a Monday-to-Sunday grid of one month with ‹ › and **Today**, showing each activity on the
  days it runs.

**Saved views**

- **FR-029**: Users MUST be able to save the current state under a name of at most 60 characters,
  unique for that user (ignoring letter case), and are asked before an existing name is replaced. A
  view stores the view type, year, filters, search, Show levels, columns and their order, sort,
  expanded rows, the unassigned switch, and the monthly or quarterly choice.
- **FR-030**: Saved views MUST be personal; nobody else sees them. The Views menu marks the active view,
  and shows "Modified" with **Update view** and **Save as new view** when the state differs from it.
- **FR-031**: Users MUST be able to mark one view as their default (FR-009), unmark it, and rename and
  delete views. Deleting asks for confirmation, and the toast offers **Undo**. A view that refers to
  something that no longer exists MUST apply the rest and name what was left out.

**Export**

- **FR-032**: **Export** MUST produce a spreadsheet or CSV file of the current view. The file holds
  every matching and context row, including rows in collapsed branches, with Level and Path columns,
  the visible columns in order, and the totals.
- **FR-033**: Money MUST be exported as numbers in the display currency, with the currency in the column
  heading; a local amount has a Currency column beside it. Dates MUST be exported as dates and statuses
  as words.
- **FR-034**: Every export MUST state, before the data, the view, year, active filters, search, Show
  levels, display currency, exporter, and date and time; a spreadsheet puts these on an "About this
  export" sheet. The file name MUST give the view, year and date, e.g. "NorthStar All work – Grid –
  2026 – 18 Oct 2026".

### Key Entities *(include if feature involves data)*

- **Saved view** (domain model §7): owner (person), name, default flag, and the state listed in FR-029.
- **Filter set**: year, filter choices, search and Show levels. It decides which records match and
  which are shown as context.
- **Period amount** (calculated, never typed): the planned cost of a line item or an unassigned balance
  in one month, derived with the spread rules of domain model §4.4.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user finds any activity by name or ID in at most 2 steps: type in the search box, then
  open the result (design system §10).
- **SC-002**: On the seed data, 100 % of roll-ups, period amounts, totals and summary cards equal values
  calculated by hand from domain model §4, to the cent.
- **SC-003**: For every combination of filters tested, the totals of each view equal the summary cards.
- **SC-004**: With 5,000 activities and 20,000 line items in one year, every filter, search or view
  change shows its result in under 2 seconds.
- **SC-005**: In a test with at least 5 first-time users, at least 90 % find "the planned cost of
  declined activities in FY2026 Lubricants Europe, by quarter" within 3 minutes without help. Every user
  can say which filters are active without opening a menu.
- **SC-006**: 100 % of exported files contain exactly the view's rows, columns and totals, and name
  every active filter.
- **SC-007**: Saving the current view takes at most 3 steps, and applying a saved view at most 2.
- **SC-008**: Every view passes an automated WCAG 2.2 AA check with no failures, can be used with the
  keyboard alone, and never shows status by colour alone.

## Assumptions

- The year is the calendar year (decision D-011, open question Q-004). The year filter selects plans of
  that year, so work in another year's plan that runs into this year is not shown.
- Typical volume per year is up to 20 plans, 5,000 activities and 20,000 line items.
- The display currency comes from the user's profile (feature 003), using today's rates in Settings.
  There are no historical rates.
- Statuses, completeness, roll-ups and spreads are calculated as the domain model says (features
  004–008); this feature only shows them.
- An export includes rows in collapsed branches, so expansion never silently drops data.
- Saved views are personal settings, not business records. They are not written to the activity log or
  sent to the recycle bin, and they are not shared in this phase.
- Out of scope: inline editing, drag-to-reschedule, printing or PDF, calendar feeds, snapshots,
  templates, actuals and other Finance columns, Measure dashboards and AI. Selecting rows for bulk
  actions, and import, come with feature 012.
- Every user sees every plan (decision D-004); **My teams** only narrows the view.
