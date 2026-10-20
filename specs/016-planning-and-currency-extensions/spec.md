# Feature Specification: Planning and Currency Extensions

**Feature Branch**: `016-planning-and-currency-extensions`

**Created**: 2026-10-20

**Status**: Draft (product owner decision of 2026-10-20: build every MVP gap now with documented, reversible
defaults; every default is listed in *Defaults for Shell to confirm*)

**Input**: Close the MVP gaps of BUS.BUD.050, BUS.PLN.020, BUS.PLN.040, BUS.BUD.020 and BUS.UX.020
([BRD](../../docs/reference/brd-requirements.md), [gap analysis](../../docs/reports/brd-gap-analysis.md) §3.2,
§3.4, §3.6, §3.7, §3.9).

**Amends**: 002, 004, 005, 006, 007 and 010. Every amended requirement is named in the FR that changes it
(*amends 00x FR-yyy*) and its new wording is in *Amended requirements of earlier features*. Consequential
changes without new wording in 003, 008, 009, 011 and 012 are listed there too.

**References**: Domain model §1, §2.5, §3, §4 (money rules), §6 · Field catalogue §1–§5 · Design system §4
(money), §5 (InfoTip, MoneyField), §6 (CRUD) · Decisions D-008, D-010, D-017, D-027, D-033, D-041, D-061,
D-079, D-141, D-207, D-213, D-220 · Questions Q-009, Q-012, Q-013, Q-016, Q-017.

## BRD coverage

| BRD | What Shell asks | Stories | Requirements |
|---|---|---|---|
| BUS.BUD.050 | Currency conversion in planning, analysis and reporting; exchange-rate management; daily GSAP rates; plan in local currency and USD | US1, US2 | FR-001 – FR-019 |
| BUS.PLN.040 | Split the budget by month (phasing, ongoing updates); split an activity from its original budget | US3, US4 | FR-020 – FR-031 |
| BUS.PLN.020 | Multi-year, annual, quarterly, monthly; strategic, operational, cross-market planning | US3, US5 | FR-021, FR-024, FR-032 – FR-038 |
| BUS.BUD.020 | Budget categories (region, brand, spend type, business unit, channel); client updates attributes; flexible framework | US6 | FR-039 – FR-044 |
| BUS.UX.020 | Contextual help (ⓘ) on every attribute and field, maintained by Shell administrators | US7 | FR-045 – FR-048 |
| All five | Every change logged; changes guarded by 014's permissions (D-301) | all | FR-049 |

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Keep dated exchange rates and import GSAP daily rates (Priority: P1)

An administrator opens **Settings › Currencies › EUR** and sees the current rate with its date and source
("1 USD = 0.92 EUR · rate of 19 Oct 2026 · GSAP import") and a **Rate history**. Each day's rates arrive as
the GSAP daily rate extract; until GSAP can deliver them automatically the administrator imports the file
(the import is *simulated*: NorthStar reads the file, it connects to no system). A rate dated in the future
(for example next year's opening rate) becomes current on its date by itself.

**Why this priority**: Every other currency feature reads these rates; BUS.BUD.050 asks for rate management
and GSAP daily rates explicitly.

**Independent Test**: Import the sample file, check the review and the rate history; add a future rate and
check it is not used before its date.

**Acceptance Scenarios**:

1. **Given** EUR with rates on 18 and 19 Oct 2026 and today is 20 Oct 2026, **Then** EUR's current rate is the
   19 Oct rate and its page shows "Rate of 19 Oct 2026 · GSAP import"; **When** an administrator adds
   0.95 dated 1 Jan 2027, **Then** the history lists it with "From 01 Jan 2027" and every conversion still uses
   the 19 Oct rate until 1 Jan 2027.
2. **Given** **Import GSAP rates** and a file of 92 rows, **When** it is uploaded, **Then** before anything is
   saved the review lists 87 rates to add, 2 that replace a different rate for the same day (old → new), 1
   already present, 1 not used ("CHF is not set up in Settings › Currencies") and 1 problem ("Row 41: the
   rate must be above 0"), each with its row number; **Import 89 rates** saves them and **Cancel** saves
   nothing.
3. **Given** the import is done, **Then** *Recent imports* shows the file name, who, when, the date range
   and the counts, and the activity log has one entry "Alex Morgan imported 89 exchange rates from
   'gsap_rates_2026-10-20.csv'".
4. **Given** a rate change that alters EUR's current rate, **When** the administrator saves it, **Then** the
   confirmation states the line items and the plans planned in EUR whose USD values change (before → after)
   and that no activity goes back for approval (002 FR-030).
5. **Given** EUR has only one rate dated today or earlier, **When** the administrator deletes it, **Then**
   NorthStar refuses: "EUR needs a rate dated today or earlier." Other rates move to the recycle bin with
   **Undo**.

---

### User Story 2 - Plan in a local currency with USD alongside (Priority: P1)

Priya creates "FY2027 Lubricants UK & Ireland" with **Plan currency GBP**. Every budget and planned cost in
the plan is typed in GBP; every amount in the plan is shown in GBP with its USD value beside it (her display
currency) and the rate used. Line items still use any currency. Cross-plan pages add plans up in USD (or the
reader's display currency) as today.

**Why this priority**: "Ability to plan in local currency and USD" is Shell's own comment on BUS.BUD.050;
today only line items accept a local currency.

**Independent Test**: Create a GBP plan, a campaign with a budget, an activity with a planned cost and a EUR
line item; check every amount, the USD beside it and the unassigned balance.

**Acceptance Scenarios**:

1. **Given** the plan form, **When** Priya chooses Plan currency GBP and Plan budget 600,000, **Then** the
   field reads "600,000.00 GBP" with "≈ 759,493.67 USD at 0.79 GBP per USD (rate of 19 Oct 2026)" beneath.
2. **Given** the GBP plan, **Then** its summary cards, plan tree, campaign and initiative panels and
   activity page show amounts in GBP, each with the USD amount beside it; hovering over the USD amount shows
   the rate used and its date and source.
3. **Given** an activity with planned cost 40,000.00 GBP and line items 15,000.00 GBP and 23,000.00 EUR,
   **Then** the EUR line item shows "= 19,750.00 GBP" and the unassigned planned cost reads "5,250.00 GBP not
   yet assigned to a line item" (≈ 6,645.57 USD).
4. **Given** activity type "Event" with a default planned cost of 5,000.00 USD, **When** it is chosen in the GBP
   plan, **Then** the planned cost is filled with 3,950.00 GBP and the note "Default 5,000.00 USD = 3,950.00 GBP
   at today's rate".
5. **Given** the plan has campaign budgets or activity planned costs, **When** Priya edits the plan, **Then**
   Plan currency is locked with "The plan currency can't change once campaigns, initiatives or activities
   have amounts in it (4 amounts)."
6. **Given** a GBP campaign and the Move dialog, **Then** a 2027 plan planned in EUR is listed but not
   selectable: "Planned in EUR — this campaign's amounts are in GBP."
7. **Given** Plans, Home and All work, **Then** totals over several plans are in the reader's display currency
   and add the GBP plan at today's rate.

---

### User Story 3 - Phase budgets by month and compare them with planned cost (Priority: P2)

Priya splits the plan budget over the months of 2027 (evenly, by quarter or month by month), does the same
for campaigns and initiatives, and keeps it up to date during the year. The **Budget by month** card shows,
month by month or quarter by quarter, the phased budget against the planned cost and flags months where
planned cost runs ahead of budget.

**Why this priority**: BUS.PLN.040 "split budget by month — phasing and ongoing updating"; quarterly views
serve BUS.PLN.020.

**Independent Test**: Phase a plan budget evenly and a campaign budget by quarter; check the card, the
publish check and Budget & cost's comparison.

**Acceptance Scenarios**:

1. **Given** plan budget 600,000.00 GBP, **When** Priya chooses **Evenly**, **Then** each month of 2027 shows
   50,000.00 GBP; changing the budget to 612,000.00 re-phases every month to 51,000.00 at once.
2. **Given** **Custom** in Quarterly mode, **When** Priya types 100,000.00 for Q1, **Then** January and February
   get 33,333.33 and March 33,333.34; the running total reads "100,000.00 of 600,000.00 phased — 500,000.00
   left".
3. **Given** a custom phasing adding up to 590,000.00 of 600,000.00, **When** Priya saves it, **Then** it is
   saved, the plan's completeness shows "Budget by month adds up to 590,000.00 GBP, not the budget
   600,000.00 GBP" and **Publish** lists it as missing.
4. **Given** March has a phased budget of 50,000.00 and planned cost of 64,000.00, **Then** the card shows
   March in red with "Over by 14,000.00 GBP"; nothing is blocked.
5. **Given** **Copy from planned cost**, **Then** each month is filled with the planned cost of that month and
   the note says what was left out ("12,000.00 GBP in 2028 and 3,000.00 GBP without dates are not copied").
6. **Given** the Published plan, **When** Priya changes March's phasing, **Then** it is saved without
   unpublishing and the log shows "Budget by month: Mar 2027 50,000.00 → 60,000.00 GBP".

---

### User Story 4 - Split an activity (Priority: P2)

"UK launch event" turns out to be two pieces of work. Priya chooses **Split activity**, names the new
activity, picks the initiative whose budget it should count against, ticks the line items that go with it
and how much planned cost moves. Both activities remember the split.

**Why this priority**: BUS.PLN.040 "split an activity from its original budget".

**Independent Test**: Split a Created activity into another initiative; split an Approved one and check the
re-approval.

**Acceptance Scenarios**:

1. **Given** "UK launch event" (40,000.00 GBP; "Venue" 15,000.00 GBP; "Agency fee (Ireland)" 23,000.00 EUR =
   19,750.00 GBP), **When** Priya splits off "Agency fee (Ireland)" into the initiative "Ireland" as "Ireland
   roadshow", **Then** the dialog's preview shows the original at 20,250.00 GBP with 5,250.00 GBP unassigned,
   and the new activity at 19,750.00 GBP with 0.00 unassigned.
2. **Given** she confirms, **Then** "Ireland roadshow" is Created with the original's type, owner, dates,
   workflow and Shell attributes, no goals ("Goals stay on 'UK launch event'"), the line item keeps its ID
   L-1010, and the pages show "Split from 'UK launch event' (A-1012)" and "Split into 'Ireland roadshow'
   (A-1014)".
3. **Given** the original is Approved, **Then** the dialog says "This activity will need approval again";
   on confirmation it returns to Created (domain model §2.5).
4. **Given** nothing ticked and 0.00 to move, **When** Priya confirms, **Then** "Choose line items or an
   amount to move" is shown and nothing changes.
5. **Given** an initiative whose dates do not contain the activity's dates, **Then** it is listed but not
   selectable, with Move's reason (006 FR-029).

---

### User Story 5 - Plan types and multi-year views (Priority: P2)

Plans are typed **Operational**, **Strategic** or **Cross-market**. The Plans list and All work look across
several years at once: Plans filters a year range; All work shows a range of up to five years with yearly,
quarterly or monthly budget and cost.

**Why this priority**: BUS.PLN.020 asks for multi-year, quarterly and strategic/operational/cross-market
planning; plans stay one fiscal year each (D-011).

**Independent Test**: Type two plans, filter Plans by 2026–2027 and type, and open All work for 2026–2027 in
the Yearly and Quarterly Budget & cost.

**Acceptance Scenarios**:

1. **Given** the plan form, **Then** Plan type offers Operational (default), Strategic and Cross-market, each
   with its ⓘ meaning, and the plan's header and card show the type chip.
2. **Given** Plans with Years 2026–2027 and Plan type Strategic, **Then** only strategic plans of both years
   are listed and the summary cards cover them.
3. **Given** All work with Years 2026–2027, **Then** the Grid lists the plans of both years with their year,
   Budget & cost opens on **Yearly** with columns 2026 and 2027, and Quarterly shows "Q1 2026" … "Q4 2027".
4. **Given** the Calendar for 2026–2027, **Then** one timeline runs across both years with year and quarter
   bands, and an activity running from Nov 2026 to Feb 2027 is one bar.
5. **Given** **Compare with phased budget** in Budget & cost, **Then** each plan, campaign and initiative row
   gets a "Phased budget" sub-row and periods where planned cost is above it are red.

---

### User Story 6 - Analyse by budget category (Priority: P2)

A global budget owner wants planned cost by brand, by country, by spend type or by Cluster › Region without
exporting to Excel. All work gains filters by any Shell attribute and a **Breakdown** view; the plan page
gains a **Planned cost by** card.

**Why this priority**: BUS.BUD.020 names these categories; they exist as Shell attributes but cannot be
filtered or added up (D-207).

**Independent Test**: Filter All work by Brands; break down by Country/OU then by Activity type; check the
plan page card.

**Acceptance Scenarios**:

1. **Given** **More filters › Brands › Helix, Not filled in**, **Then** activities with Helix or without a
   brand match, with a chip "Brands: Helix, Not filled in".
2. **Given** Breakdown by Country/OU and an activity of 10,000.00 USD split Germany 60 % / France 40 %, **Then**
   Germany gets 6,000.00 and France 4,000.00 of it.
3. **Given** Breakdown by Channel/audience (multiple choice, no split) and an activity with External and
   Internal, **Then** it counts in both groups, the totals row counts it once and the view says "Activities
   with several values count in each group; the total counts them once."
4. **Given** Breakdown by Spend type, **Then** line items' planned cost is grouped by their spend type and each
   activity's unassigned planned cost is in "Not on a line item".
5. **Given** Breakdown by Cluster then by Region, **Then** each cluster group opens into its regions with
   subtotals, and the totals row equals the Planned cost card.
6. **Given** the plan page, **Then** **Planned cost by** shows the plan's planned cost by Spend type in the plan
   currency, with a picker for the other groupings.

---

### User Story 7 - Field help maintained by administrators (Priority: P1)

Every field users fill in, and every calculated value they read, has an ⓘ that explains it. Administrators
keep the texts up to date in **Settings › Field help** without a release; Shell attribute help stays on each
attribute.

**Why this priority**: BUS.UX.020 is MVP; today only Shell attributes and a few fixed tips have help.

**Independent Test**: Open every planning form and check each label has an ⓘ; change one text in Settings
and see it on the next form; reset it.

**Acceptance Scenarios**:

1. **Given** the plan form, **When** the user focuses the ⓘ next to Plan budget, **Then** it reads the text of
   Field help `plan.budget` ("The money this plan may spend in its year, in the plan currency. Campaign
   budgets are allocated from it.").
2. **Given** Settings › Field help, **When** an administrator edits "Planned cost" of activities and saves,
   **Then** the next activity form anyone opens shows the new text, the entry shows "Edited" with who and
   when, and the log shows old → new.
3. **Given** an edited entry, **When** the administrator chooses **Reset to default**, **Then** the default text
   returns and the chip reads "Default".
4. **Given** a text of 301 characters, **Then** "Use 300 characters or fewer." is shown; an empty text is
   allowed and hides the ⓘ of that field.
5. **Given** Field help, **Then** a note says "3 Shell attributes have no help yet — view them" and links to
   the Shell attributes list filtered to them.

---

### Edge Cases

- A currency set inactive keeps its rate history; plans and line items using it keep converting at its
  current rate (002 FR-004).
- A day without a rate (weekend, holiday, a missing GSAP file) uses the latest earlier rate.
- An import row for a date that already has the same rate is skipped as "already present"; a row for USD is
  not used ("USD is always 1").
- A rate deleted to the recycle bin and restored when another rate for that currency and date exists cannot be
  restored: "EUR already has a rate on 19 Oct 2026."
- A plan whose currency is set inactive keeps it; new plans cannot choose it.
- Line items in the plan currency convert exactly (no rounding); others convert once, half up, with the
  rounding hint of D-145 measured in the plan currency.
- Phasing is offered only when the record has a budget; clearing the budget clears the phasing (announced at
  the field, D-151).
- A campaign whose dates leave the plan's year (R4) is still phased over the plan's year only.
- A split of an activity without a planned cost moves line items only; the amount field is disabled with
  "This activity has no planned cost".
- Splitting may leave the original with a negative unassigned balance; it is shown in red and blocks
  submitting, as today (D-016).
- Deleting the original after a split: the new activity's link reads "Split from 'UK launch event' (A-1012,
  deleted)".
- An All work saved view made before this feature (one year) opens as the range year–year.
- Breakdown by an attribute hidden by its business-unit condition puts those records in "Not filled in".
- Field help entries that refer to a field removed later are not shown (the catalogue is part of the code).

## Requirements *(mandatory)*

### Functional Requirements

**A. Exchange rates and the GSAP rate import (BUS.BUD.050)**

- **FR-001**: Every currency except USD MUST keep a **rate history**: dated rates in units per 1 USD (above 0,
  at most six decimals, at most 1,000,000), at most one per currency and date, each with its source
  (**Entered by hand** or **GSAP import**) and who recorded it and when. USD is always 1 and has no history.
  *(BUS.BUD.050; amends 002 FR-028)*
- **FR-002**: A currency's **current rate** MUST be its rate with the latest date on or before today (the API's
  calendar date in UTC, D-091); a future-dated rate MUST become current on its date without anyone saving
  anything. Every conversion in NorthStar MUST use the current rate. *(BUS.BUD.050; amends 002 FR-029,
  007 FR-013; reopens D-027)*
- **FR-003**: The currency's details MUST show the current rate both ways with its date and source, and a
  **Rate history** (newest first: date, rate, inverse, source, recorded by and when, "Current" on the rate in
  use, "From <date>" on future rates) with **Add rate** (date, default today; rate), **Edit** (rate only) and
  **Delete** (recycle bin, Undo) per row. Deleting the last rate dated today or earlier MUST be refused ("EUR
  needs a rate dated today or earlier."). A new currency MUST get its first rate dated today or earlier
  (default today). *(BUS.BUD.050; amends 002 FR-028, US4)*
- **FR-004**: Changing the rate in the currency form MUST set today's rate (add today's row or replace it).
  *(BUS.BUD.050; amends 002 FR-028)*
- **FR-005**: Before saving a rate that changes a currency's current rate NorthStar MUST state the impact of
  002 FR-030 and also the plans planned in that currency: "2 plans are planned in EUR. Their EUR amounts stay
  the same; their USD values change (1,304,347.83 → 1,263,157.89 USD)." No activity goes back for approval.
  *(BUS.BUD.050; amends 002 FR-030)*
- **FR-006**: Settings › Currencies MUST offer **Import GSAP rates**: upload a GSAP daily rate extract (the
  simulated format of data-model §2.4, `.csv`, up to 2 MB and 20,000 rows) and see, before anything is saved,
  every row's outcome with its row number: **to add**, **replaces** (a different rate for the same currency
  and day, old → new), **already present** (skipped), **not used** (another rate type, not quoted against
  USD, USD itself, or a currency not set up — currencies are never created) and **problem** (unreadable
  date or rate). **Import n rates** saves the rows to add and replace in one step; **Cancel** saves nothing.
  A **Download a sample file** link gives a file of today's rates. *(BUS.BUD.050; new)*
- **FR-007**: Every import MUST be recorded (file name, who, when, first and last rate date, counts) and shown
  under **Recent imports** (last 10); imported rates show "GSAP import" and the file name as source; the
  import writes one activity-log entry. *(BUS.BUD.050; new)*
- **FR-008**: Rates MUST reach NorthStar through one replaceable **rate feed** seam, so that an automatic daily
  GSAP feed can later deliver rates under the same rules without changing screens or calculations. In this
  phase the only feed is the file import; NorthStar connects to no external system. *(BUS.BUD.050; new;
  reopens D-008 for rates only)*
- **FR-009**: Every amount converted from another currency MUST say which rates it used: the converted amount
  (and its ⓘ) shows on hover and focus "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import" for each
  currency involved; the line-item form's live conversion includes the rate date; exports list every rate
  used on "About this export". *(BUS.BUD.050; amends 007 FR-011, 010 FR-034)*
- **FR-010**: A currency's "Used by" MUST also count the plans planned in it; a currency that is a plan's
  currency MUST NOT be deleted (Set inactive is offered). *(BUS.BUD.050; amends 002 FR-003, FR-004)*

**B. Plan currency: plan in local currency and USD (BUS.BUD.050)**

- **FR-011**: Every plan MUST have a **plan currency**: an active currency, default USD, chosen in the plan
  form's Budget section. The plan budget, campaign and initiative budgets, budget phasing and activities'
  planned costs of the plan MUST be typed and kept in the plan currency; the API MUST refuse another currency
  ("Enter the amount in GBP, the plan's currency."). *(BUS.BUD.050; amends 004 FR-005, FR-006, FR-010;
  005 FR-002, FR-005; 006 FR-011; reopens D-010, D-079)*
- **FR-012**: The plan currency MUST be changeable only while no campaign or initiative of the plan has a
  budget and no activity has a planned cost; otherwise it shows a lock: "The plan currency can't change once
  campaigns, initiatives or activities have amounts in it (n amounts)." Changing it in the form keeps the
  typed plan budget number and relabels it. *(BUS.BUD.050; amends 004 FR-018)*
- **FR-013**: Inside a plan — plan page and summary cards, plan tree, campaign and initiative panels, activity
  page, line-items section and line-item pages, Budget by month — amounts MUST be shown in the plan currency,
  with the same amount in the reader's display currency beside it when the two differ (USD unless the reader
  chose another in My profile), each labelled with its currency and carrying the rate used (FR-009).
  *(BUS.BUD.050; amends 004 FR-012, FR-013, FR-015; 005 FR-010, FR-011, FR-019, FR-022; 006 FR-019, FR-033;
  007 FR-001, FR-002, FR-003, FR-005; refines D-061, D-220)*
- **FR-014**: Pages that add up several plans — Plans list summary cards, Home, All work, Approvals — MUST
  convert each plan's amounts to USD at the current rate before adding and show results in the reader's
  display currency, as today. A Plans list card or row (one plan) MUST show its amounts in the plan currency
  with the display currency beside. *(BUS.BUD.050; amends 004 FR-001, FR-002; 010 FR-017)*
- **FR-015**: A line item MUST keep its own currency (any active one). Its planned cost MUST be shown in its
  currency, in the plan currency ("= 19,750.00 GBP") and in tables also in the display currency. The
  unassigned planned cost (Sys_Gen_Balance) and the activity's spread grid MUST be worked out in the plan
  currency (data-model §3.1). *(BUS.BUD.050; amends 007 FR-001, FR-002, FR-003, FR-011, FR-012, FR-020,
  FR-026; domain model §4.3)*
- **FR-016**: An activity type's default planned cost (kept in USD in Settings) MUST be offered converted into
  the plan currency at the current rate, saying so ("Default 5,000.00 USD = 3,950.00 GBP at today's rate").
  *(BUS.BUD.050; amends 006 FR-007)*
- **FR-017**: Moving or copying a campaign into a plan with another plan currency MUST NOT be offered: the
  destination is listed but not selectable, with "Planned in EUR — this campaign's amounts are in GBP."
  *(BUS.BUD.050; amends 005 FR-023, FR-026)*
- **FR-018**: A plan copy MUST keep the plan currency and plan type. *(BUS.BUD.050, BUS.PLN.020; amends
  004 FR-027)*
- **FR-019**: Goals MUST stay in USD: the model's average deal size and every revenue goal are USD amounts,
  labelled "USD". *(BUS.BUD.050; clarifies 004 FR-009, 006 FR-018)*

**C. Budget phasing by month (BUS.PLN.040, BUS.PLN.020)**

- **FR-020**: Plans, campaigns and initiatives that have a budget MUST offer **Budget by month** with three
  methods: **Not phased** (default), **Evenly** (the budget over the 12 months of the plan's year by the
  Evenly rule of domain model §4.4, recalculated whenever the budget changes) and **Custom** (an amount of
  0.00 or more per month of the plan's year, in the plan currency). *(BUS.PLN.040; new)*
- **FR-021**: The phasing editor MUST offer a **Monthly · Quarterly** switch — a quarter's amount typed in
  Quarterly is spread over its three months by the Evenly rule — and the tools **Split evenly**, **Copy from
  planned cost** (each month gets the record's planned cost of that month, FR-024; amounts in other years or
  without dates are named as left out) and **Use month total as budget**; a running total reads "<sum> of
  <budget> phased — <X> left" or "… <X> too much". *(BUS.PLN.040, BUS.PLN.020; new)*
- **FR-022**: A Custom phasing that does not add up to the budget MUST NOT block saving; it MUST count as a
  needed-to-publish problem of its record ("Budget by month adds up to X, not the budget Y") and so block
  publishing the plan. A changed budget keeps Custom months and flags them; a removed budget removes the
  phasing, announced at the field. *(BUS.PLN.040; amends 004 FR-017, FR-024; 005 FR-014)*
- **FR-023**: Phasing MUST be changeable at any time, from the record's **Budget by month** card with its own
  **Edit** and **Save**, whatever the plan's status; every change is logged with old and new months; it never
  changes an activity's status. *(BUS.PLN.040 "ongoing updating"; new)*
- **FR-024**: The plan page and the campaign and initiative panels MUST show a **Budget by month** card with a
  Monthly · Quarterly switch and the rows **Budget** (phased; "Not phased" when none), **Planned cost** (the
  record's planned cost by month in the plan currency, by the spread rules of domain model §4.4 and 010
  FR-023), **Difference** (budget − planned cost) and, for plans and campaigns, **Allocated to campaigns /
  initiatives** (the children's phased budgets). Periods where planned cost exceeds the phased budget, or the
  children's phased budgets exceed the record's, MUST be red with "Over by X"; **Other years** and **No
  dates** columns appear when they hold amounts. Nothing here blocks. *(BUS.PLN.040, BUS.PLN.020; amends
  004 FR-012, 005 FR-016)*
- **FR-025**: A next-year plan copy MUST move phasing months one year later; moves and copies of campaigns and
  initiatives keep their phasing. *(BUS.PLN.040; amends 004 FR-028; 005 FR-025, FR-027)*

**D. Split activity (BUS.PLN.040)**

- **FR-026**: The activity page's **More** menu and the activity row's **⋯** menu MUST offer **Split
  activity**, a dialog with: the new activity's name (default "<name> (split)", unique in the chosen
  initiative), its initiative (the current one by default, or another initiative of the same plan, with
  Move's availability rules and reasons, 006 FR-029), the line items to move (ticked list with their amounts
  in the plan currency) and the planned cost to move (0.00 up to the activity's planned cost; defaults to the
  sum of the ticked line items and follows the ticks until typed). *(BUS.PLN.040; amends 006 FR-019,
  FR-033)*
- **FR-027**: Before confirming, the dialog MUST show both activities as they will be — planned cost, line
  items, unassigned planned cost (red when negative) and status — from the API's preview. At least one line
  item or an amount above 0.00 MUST move. *(BUS.PLN.040; new)*
- **FR-028**: **Split** MUST, all or nothing: create the new activity with the original's type, owner, dates,
  ongoing, approval workflow, description, external name and Shell attributes, the chosen name, planned cost =
  the moved amount, no goals ("Goals stay on '<original>'") and status Created; move the ticked line items to
  it keeping their IDs, values and spreads; lower the original's planned cost by the moved amount; and keep
  the link between the two. *(BUS.PLN.040; new)*
- **FR-029**: Splitting a Submitted or Approved activity MUST follow the re-approval rule of domain model §2.5:
  the dialog says "This activity will need approval again" and, on confirmation, the original returns to
  Created (D-013). Moved line items' SWBS request status follows the new activity. *(BUS.PLN.040; amends
  006 FR-027, 007 FR-031)*
- **FR-030**: A split MUST be logged: on the original (verb "split", what moved, planned cost old → new), on
  the new activity (created, "Split from '<name>' (A-…)") and on each moved line item (moved, old → new
  activity); both initiatives' roll-ups update at once. *(BUS.PLN.040; amends 006 FR-005)*
- **FR-031**: Both activity pages MUST show the link — "Split from 'UK launch event' (A-1012)" and "Split into
  'Ireland roadshow' (A-1014)" — with "deleted" when the other activity is in the recycle bin; copies never
  carry it (R13). *(BUS.PLN.040; new)*

**E. Plan type and multi-year views (BUS.PLN.020)**

- **FR-032**: Every plan MUST have a **Plan type** — **Operational** (default), **Strategic** or
  **Cross-market** — chosen in the plan form's Basics and shown as a chip on plan cards, rows and the plan
  header; the Strategic / AOP flag stays unchanged. *(BUS.PLN.020; amends 004 FR-002, FR-005, FR-006, FR-011)*
- **FR-033**: The Plans list MUST replace its year filter with **Years** — All years (default) or a range
  from–to among the years with plans — and add a **Plan type** filter (several); the summary cards cover the
  range; filters are remembered as today (D-081). *(BUS.PLN.020; amends 004 FR-001, FR-003)*
- **FR-034**: All work MUST replace Year with **Years**, a range from–to of at most 5 years (default the current
  year to the current year); every view, card, count and total covers the plans of every year in the range;
  plan rows show their year; the count reads "Showing N of M activities" over the range. *(BUS.PLN.020;
  amends 010 FR-004, FR-008, FR-009, FR-014)*
- **FR-035**: Budget & cost MUST offer **Monthly · Quarterly · Yearly** over the whole range ("Mar 2026",
  "Q1 2026", "2026"); a range of more than one year opens on Yearly; **Other years** holds amounts outside the
  range. *(BUS.PLN.020; amends 010 FR-022)*
- **FR-036**: The Calendar MUST show the whole range as one timeline with year and quarter bands above the
  months; "Continues from / into <year>" appears only at the range's edges. *(BUS.PLN.020; amends 010
  FR-025, FR-026)*
- **FR-037**: All work MUST add the filters **Plan type** and **Plan currency** (several each, tested on
  plans) and the Standard columns **Year**, **Plan type**, **Plan currency**, **Plan budget (plan
  currency)** and **Planned cost (plan currency)**. *(BUS.PLN.020, BUS.BUD.050; amends 010 FR-004, FR-018)*
- **FR-038**: Budget & cost MUST offer **Compare with phased budget** (off by default): under each plan,
  campaign and initiative row a "Phased budget" sub-row gives its phased budget per period in the display
  currency ("Not phased" when none), and planned-cost cells above the phased budget of their period are red.
  *(BUS.PLN.040, BUS.PLN.020; amends 010 FR-022, FR-024)*

**F. Budget categories (BUS.BUD.020)**

- **FR-039**: All work MUST offer **Shell-attribute filters** under **More filters**: every active choice or
  yes/no Shell attribute, grouped by level; values (several) and **Not filled in** can be chosen; each filter is
  tested on the attribute's level with the matching rules of 010 FR-005 (D-113); a % split value matches when
  its share is above 0; a value hidden by its business-unit condition counts as not filled in; each filter is a
  removable chip. *(BUS.BUD.020; amends 010 FR-004, FR-008, FR-010; reopens D-207, answers Q-009)*
- **FR-040**: All work MUST add a fourth view, **Breakdown**: "Break down by" one grouping — any choice or yes/no
  Shell attribute of any level, or Plan, Plan type, Plan currency, Year, Team, Business unit, Activity type,
  Owner — and optionally "then by" a second. Each group shows Activities, Line items, Planned cost and Share of
  planned cost and, for groupings of plans, campaigns or initiatives, Budget and Budget used; a "Not filled in"
  group; a totals row equal to the summary cards. Rows open the group in the Grid with the matching filter.
  *(BUS.BUD.020; amends 010 FR-001)*
- **FR-041**: Breakdown MUST attribute amounts by these rules (data-model §3.5): an activity's planned cost goes
  to the value of the attribute on the activity or on its nearest ancestor at the attribute's level; a % split
  divides it by the shares (half up per value, the remainder on the last value); a multiple choice without
  split counts it fully in each chosen group, and the view says "Activities with several values count in each
  group; the total counts them once"; a line-item attribute groups line items' planned cost and puts each
  activity's unassigned planned cost in "Not on a line item". *(BUS.BUD.020; new)*
- **FR-042**: The plan page MUST show a **Planned cost by** card — the Breakdown of that plan in the plan
  currency, Spend type by default, with the same groupings to choose. *(BUS.BUD.020; amends 004 FR-012)*
- **FR-043**: Saved views MUST store the year range, Shell-attribute, plan type and plan currency filters,
  the Breakdown groupings, the Yearly period and the compare switch; an export of the Breakdown view MUST hold
  the group table. *(BUS.BUD.020, BUS.PLN.020; amends 010 FR-029, FR-032, FR-033, FR-034)*
- **FR-044**: The planning levels stay Plan → Campaign → Initiative → Activity → Line item (constitution II).
  Shell's own hierarchies (Cluster › Region, Spend type › GL code, any chain of dependent attributes, 002
  FR-022) are analysed with "Break down by … then by …". Budgets are reported by category, not set per
  category. *(BUS.BUD.020; default for Q-017)*

**G. Field help (BUS.UX.020)**

- **FR-045**: Settings MUST have a **Field help** section listing every built-in field and calculated value of
  the plan, campaign, initiative, activity and line-item levels and of the planning pages (the catalogue of
  data-model §6), grouped by level, searchable, each with its help text (a default supplied by NorthStar) and a
  "Default" or "Edited" chip. Shell attributes' help stays on each attribute's page, linked from here.
  *(BUS.UX.020; amends 002 FR-001)*
- **FR-046**: An administrator MUST be able to edit a help text (0–300 characters, plain text) and **Reset to
  default**; an empty text hides that ⓘ. Every change is logged with old and new text; a save over a newer
  change is refused, naming who and when (001 FR-029); the next page anyone opens shows the change (002
  FR-007). *(BUS.UX.020; new)*
- **FR-047**: Every form field and every labelled value of the details pages, panels and summary cards of
  features 004–007, and All work's summary cards and column chooser (010), MUST show an ⓘ with its Field help
  text — including the tips written in code today (summary-card formulas, unassigned planned cost, budget
  used, spread), which become Field help entries. *(BUS.UX.020; amends 002 FR-024; 004 FR-012, FR-015;
  005 FR-011; 006 FR-018, FR-019; 007 FR-002, FR-011; 010 FR-003)*
- **FR-048**: The Shell attributes list MUST mark attributes without help ("No help yet") and Field help MUST
  say how many there are, with a link; the seed MUST give every seeded Shell attribute a help text.
  *(BUS.UX.020; amends 002 FR-016, FR-032)*

**H. Shared rules**

- **FR-049**: Every change this feature adds (rates, imports, plan currency and type, phasing, split, field
  help) MUST be logged with the actor. Access follows feature 014 (consolidation D-301): rate maintenance and
  rate import need "Add currencies, change exchange rates and import rate files" (exchangeRates.manage), field
  help needs "Change Settings lists" (settings.manage), phasing needs plans.edit or campaigns.edit, split needs
  activities.edit; refusals answer 403 forbidden. With 014's default roles only administrators maintain field
  help and rates. *(all five BRD rows; follows 002 FR-008)*

### Amended requirements of earlier features

The consolidation step writes these words into the earlier specs (or links them to this one). "Add" means a
sentence added to the existing requirement; "Now reads" replaces it.

| Feature FR | Change | By |
|---|---|---|
| 002 FR-001 | Add: the cards **Field help** ("What each built-in field means, shown as ⓘ tips") and, on Currencies, "with their rate history and GSAP rate import". | FR-045 |
| 002 FR-003, FR-004 | Add: currencies count the plans planned in them; a plan's currency cannot be deleted. | FR-010 |
| 002 FR-016, FR-032 | Add: attributes without help are marked "No help yet"; seeded attributes all have help. | FR-048 |
| 002 FR-024 | Add: built-in fields show their Field help tip the same way. | FR-047 |
| 002 FR-028 | Now reads: "A currency MUST have a three-letter code (unique, locked once saved), a name, an optional symbol and a rate history (016 FR-001, FR-003); its current rate is shown both ways. USD MUST always exist, be active and have rate 1; its locks MUST say why." | FR-001, FR-003, FR-004 |
| 002 FR-029 | Now reads: "Conversions MUST use each currency's current rate (016 FR-002); a new current rate MUST recalculate every converted amount at once." | FR-002 |
| 002 FR-030 | Add: and the plans planned in that currency, with their USD values before and after. | FR-005 |
| 002 Assumptions | Remove "One current rate per currency…"; add "Rates are dated; the latest rate on or before today converts (016)." | FR-002 |
| 004 FR-001, FR-002 | Add: summary cards cover the Years range in the display currency; cards and rows show the plan-type chip and their amounts in the plan currency with the display currency beside. | FR-014, FR-032, FR-033 |
| 004 FR-003 | Now reads: "…pill filters All · Draft · Published, **Years** (All years or a from–to range) and **Plan type**, search…" | FR-033 |
| 004 FR-005, FR-006 | Add the fields **Plan type** (Basics, default Operational) and **Plan currency** (Budget, default USD, active currencies). | FR-011, FR-032 |
| 004 FR-010 | Add: plan currency must be an active currency (or the one already kept); the budget is in the plan currency. | FR-011 |
| 004 FR-011 | Add: the header shows the plan type chip and "Planned in GBP". | FR-013, FR-032 |
| 004 FR-012 | Add: amounts in the plan currency with the display currency beside; the cards **Budget by month** and **Planned cost by** follow the summary cards; every ⓘ text comes from Field help. | FR-013, FR-024, FR-042, FR-047 |
| 004 FR-013, FR-015 | "All amounts in this feature appear in the user's display currency" now reads "Amounts appear in the plan currency with the user's display currency beside when it differs (016 FR-013)." | FR-013 |
| 004 FR-017, FR-024 | Add: a Budget by month that does not add up is a needed-to-publish problem of its record. | FR-022 |
| 004 FR-018 | Add: Plan currency follows 016 FR-012 (lock once amounts exist). | FR-012 |
| 004 FR-027, FR-028 | Add: copies keep plan currency, plan type and phasing; next-year copies move phasing months one year. | FR-018, FR-025 |
| 005 FR-002, FR-005 | Budgets are "in the plan currency" (was USD). | FR-011 |
| 005 FR-010, FR-011, FR-019, FR-022 | Amounts in the plan currency with the display currency beside. | FR-013 |
| 005 FR-014 | Add: a Budget by month that does not add up counts as a needed-to-publish problem. | FR-022 |
| 005 FR-016 | Add: details show the **Budget by month** card. | FR-024 |
| 005 FR-023, FR-026 | Add: a plan with another plan currency is listed but not selectable. | FR-017 |
| 005 FR-025, FR-027 | Add: moves and copies keep phasing. | FR-025 |
| 006 FR-005 | Add: and split. | FR-030 |
| 006 FR-007 | Add: the type's default is converted into the plan currency, saying so. | FR-016 |
| 006 FR-011 | Now reads: "**Planned cost** MUST be an amount in the plan currency of 0.00 or more…" | FR-011 |
| 006 FR-018 | Add: revenue stays in USD. | FR-019 |
| 006 FR-019, FR-033 | Add: **More** and **⋯** offer Split activity; the header shows "Split from / Split into" links; amounts in the plan currency with the display currency beside. | FR-013, FR-026, FR-031 |
| 006 FR-027 | Add: a split counts as a re-approval change. | FR-029 |
| 006 Assumptions | "The planned cost is entered in USD" now reads "in the plan currency (016)"; "Single-activity copy and move stay inside one plan" also holds for split. | FR-011, FR-026 |
| 007 FR-001 | The column "planned cost in USD" now reads "planned cost in the plan currency". | FR-015 |
| 007 FR-002, FR-003, FR-020 | "USD" now reads "the plan currency" (balance, meter, grid, rounding into the last month). | FR-015 |
| 007 FR-005 | Amounts in the plan currency with the display currency beside. | FR-013 |
| 007 FR-011, FR-012 | The live conversion reads "= 19,750.00 GBP at 0.92 EUR and 0.79 GBP per USD (rates of 19 Oct 2026)"; the pre-filled balance keeps its plan-currency value. | FR-009, FR-015 |
| 007 FR-013 | "the currency's current rate in Settings (decision D-027)" now reads "the current rate of 016 FR-002". | FR-002 |
| 007 FR-026 | Add: planned cost also in the plan currency. | FR-015 |
| 007 FR-031 | Add: moving line items by a split counts as adding/deleting them for re-approval of the original. | FR-029 |
| 007 Key entities | "Belongs to exactly one activity" adds "; it changes activity only by Split (016)". | FR-028 |
| 010 FR-001 | Add the pill tab **Breakdown**. | FR-040 |
| 010 FR-003 | Add: card tips come from Field help; cards cover the range. | FR-034, FR-047 |
| 010 FR-004 | Year → **Years** (range); add **Plan type**, **Plan currency** and **More filters** (Shell attributes). | FR-034, FR-037, FR-039 |
| 010 FR-008, FR-009, FR-014 | Counts, the standard view and the totals cover the range. | FR-034 |
| 010 FR-010 | Now reads: "The filter bar MUST offer the filters of FR-004 and the Shell-attribute filters of 016 FR-039." (D-207 reopened) | FR-039 |
| 010 FR-017 | Add: converted amounts carry the rate used. | FR-009 |
| 010 FR-018 | Add the Standard columns of 016 FR-037. | FR-037 |
| 010 FR-022, FR-024 | Add Yearly and periods across the range; Compare with phased budget. | FR-035, FR-038 |
| 010 FR-025, FR-026 | The timeline covers the range with year and quarter bands. | FR-036 |
| 010 FR-029, FR-032 – FR-034 | Views store and exports state the range, new filters, Breakdown and rates used. | FR-009, FR-043 |
| 010 Assumptions | "The year filter selects plans of that year" → "of the years in the range"; "There are no historical rates" → "Rates are the current rates of 016". | FR-002, FR-034 |

**Consequential changes (no new requirement wording):** 003 — the display currency list also shows each
currency's rate date. 008 — approval rows, preview and submit dialog convert planned costs and balances from
the plan currency (amounts still in the display currency, D-213). 009 — Home converts each plan's amounts
to USD before adding (FR-014). 011 — the activity log shows the new entries (exchange rates, rate imports,
field help, split) with their own verbs (D-167); exchange rates are restorable from the recycle bin. 012
(deferred) — imported planned costs are in the plan currency; bulk copy and move follow FR-017.

### Key Entities *(include if feature involves data)*

- **Exchange rate**: a currency's rate per 1 USD on a date, its source (entered by hand or GSAP import, with
  the import) and who recorded it; the latest on or before today is the current rate.
- **Rate import**: one GSAP rate file read into NorthStar — file name, feed, who, when, rate dates, counts.
- **Plan currency** and **Plan type**: two new fields of the plan.
- **Budget phasing**: a plan's, campaign's or initiative's budget by month of the plan's year (method and,
  for Custom, the months).
- **Split link**: the activity an activity was split from.
- **Field help entry**: the help text of one built-in field or calculated value, identified by a stable key;
  only edited texts are stored.
- **Breakdown** (calculated, never stored): planned cost and budget grouped by one or two groupings.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100 % of converted amounts in the worked examples of data-model §3 match to the cent in the API
  and on screen, and every converted amount on the pages of FR-013 shows its rate and rate date.
- **SC-002**: An administrator imports a 1,000-row GSAP rate file and sees the full review in under 10 seconds
  and in at most 3 steps from Settings.
- **SC-003**: At least 90 % of first-time planners create a plan in GBP with a budget, a campaign budget and an
  activity planned cost, and find the USD value of each, within 4 minutes without help.
- **SC-004**: For one plan, the Budget by month card's planned cost per month equals All work Budget & cost
  for that plan converted to the plan currency, to the cent, for every month.
- **SC-005**: After 100 % of splits, the two activities' planned costs add up to the original's planned cost
  before the split, and the line items of both add up to the original's line items.
- **SC-006**: All work for a 5-year range with 25,000 activities shows its first page within the time 010
  SC-004 allows for one year.
- **SC-007**: In the Breakdown view the totals row equals the Planned cost card to the cent for 100 % of
  groupings.
- **SC-008**: 100 % of built-in fields on the five planning forms show an ⓘ; an edited help text is visible on
  the next form any user opens.
- **SC-009**: Every new screen and dialog passes the automated WCAG 2.1 AA check and works with the keyboard
  alone.

## Defaults for Shell to confirm

Each default is reversible and is a decision of [plan.md](plan.md) (D-258–D-275).

| # | Default | Question |
|---|---|---|
| 1 | Every conversion uses the latest daily rate on or before today; there is no separate annual planning (budget) rate. Approved activities keep their plan-currency amounts; their USD values follow the rates. | Q-012 |
| 2 | The simulated GSAP extract is a CSV with the columns `rate_type, from_currency, to_currency, valid_from, rate`; only rate type `M` quoted from USD is used; one rate per currency and day; a differing rate for an existing day replaces it. | Q-012 |
| 3 | Days without a rate (weekends, holidays) use the latest earlier rate. | Q-012 |
| 4 | One plan currency per plan, default USD, chosen at creation, locked once amounts are planned in it. Line items may use any currency. | Q-012 |
| 5 | Inside a plan amounts show in the plan currency with the reader's display currency (USD by default) beside; pages over several plans use the display currency. | Q-012 |
| 6 | Goals (average deal size, revenue) stay in USD; activity types' default planned cost stays in USD and is converted. | Q-012 |
| 7 | Campaigns cannot be moved or copied between plans of different currencies. | Q-012 |
| 8 | Budget phasing is monthly over the plan's year (quarters spread evenly over their months); a phasing that does not add up blocks publishing. | — |
| 9 | The planned cost compared with the phased budget uses the line items' spreads and spreads a positive unassigned balance evenly (as All work's Budget & cost). | — |
| 10 | "Split an activity from its original budget" means dividing one activity into two within the same plan (the new one may count against another initiative's budget); whole line items move; goals stay on the original; there is no undo (the log keeps the record). | Q-013 |
| 11 | Plan type is a fixed list (Operational, Strategic, Cross-market) beside the existing Strategic / AOP flag; a strategic plan is still one fiscal year and does not link annual plans. | Q-016 |
| 12 | Multi-year means views across plans: a year range on Plans and up to 5 years in All work. | Q-016 |
| 13 | Budgets are reported by category (filters, Breakdown), not set per category. Multiple choices count in each group; % splits divide. | Q-017 |
| 14 | Level names stay fixed (constitution II); hierarchies are modelled with dependent Shell attributes. | — |
| 15 | Field help texts are plain text up to 300 characters, defaults written by NorthStar; empty hides the ⓘ. | — |

## Assumptions

- Builds on 002–010 as built and integrated on `plan-module`, and on 011 once merged (history pages show the
  new entries; nothing here waits for 011).
- The demo data adds, without changing existing IDs or 2026 totals, a GBP plan in 2027 (data-model §8) and 30
  days of simulated GSAP rates whose latest values equal today's seeded rates, so existing tests keep their
  numbers.
- Fiscal year = calendar year (D-011); phasing months are the plan year's months.
- No external system is called; GSAP, SSO and e-mail stay simulated or later (product owner, 2026-10-20).
- Features 014 (access control), 015 (financial transactions) and 017 (approval rules) are specified in
  parallel: 014 restricts who maintains rates and field help (FR-049); 015 converts transactions at the
  current rate, as every other amount (consolidation D-303).
