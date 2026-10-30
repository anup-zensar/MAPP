# Feature Specification: SP&A Plan View and BCPIR Marker

**Feature Branch**: `013-bcpir-and-spa`

**Created**: 2026-10-18 · **Re-baselined**: 2026-10-28 on Shell's specification 006 (decision D-333)

**Status**: Built (2026-10-29; documented decisions D-334–D-337)

**Input**: Decision D-333 (constitution 4.6.0): "the SP&A plan view and the BCPIR marker and filter of
Shell's spec 006 (013, re-baselined on it; the BCPIR view's return figures wait for performance data, which
stays out)". Shell's own specification `context/specs/006-bcpir-spa-plans/spec.md` (FR-001–FR-041, Measure
Definitions) is the source; this spec takes its BCPIR identification (FR-001–FR-003), its scope, data and
presentation rules (FR-004–FR-012) and its SP&A view (FR-013–FR-021), and records what MAPP does not build.

**References**: Shell spec 006 (US1, US2, US4; FR-001–FR-021; Measure Definitions) · Hive9 functional
analysis §B3 ("BCPIR Plan?"), §C1, §C4, §E3.0.2, §E3.4.1, §E4.3–§E4.5 · Domain model §2.4a (purchase
requests), §4.1–§4.4 (budgets, roll-ups, line items, spreads), §4.6 · Field catalogue §1 #11 (BCPIR plan) ·
Design system §2 (summary cards), §4 (money, progress, chips) · Glossary *BCPIR*, *SP&A*, *Committed*,
*Paid* · Features 004 (plans), 005 (campaigns), 006 (activities), 007 (line items, spreads, finance budget
link), 010 (All work), 015 (GSAP transactions: committed, invoiced, paid), 016 (Breakdown, plan currency),
032 (purchase requests), 033 (simulated GSAP purchase orders) · Decisions D-015, D-090, D-243, D-257, D-270,
D-303, D-330, D-331, D-333, D-334–D-337.

**Words used here**: *SP&A* (Sales, Promotion & Advertising) is Shell's spend view of a plan. *BCPIR*
(Business Case & Post-Investment Review) marks a plan whose business case is reviewed after the spend. The
*scope* of the view is the whole plan or one of its campaigns. *Actual spend* is what 015 calls **Paid**
(Shell's actuals: payments released). *Committed* is what 015 calls Committed: GSAP purchase-order lines,
including the simulated GSAP's purchase orders of 033.

## Clarifications

### Session 2026-10-28

No product owner was in the loop (D-333: built on our reading of Shell's specs). Each point was answered from
Shell's spec 006 first, then the Hive9 functional analysis, then MAPP's existing decisions, and is recorded as a
decision for the product owner to confirm.

- Q: Q-001 asked what BCPIR and SP&A plans are and whether they need their own sign-off. → A: Answered by
  Shell's spec 006 and D-334: **Model A** (006 FR-032) — BCPIR stays the plan's "BCPIR Plan?" yes/no, as in
  Hive9; SP&A is a read-only view of every plan and campaign; no pair of plans per campaign (Model B), no
  sign-off of its own: the plan's Publish (004) and activity approval (008) remain the agreement. Q-001 is
  closed. The baseline's planning gate is superseded by D-333.
- Q: Which figures are "Committed" and "Actual spend" in MAPP, where 006 defines Committed from approved
  purchase requests? → A: D-335 — MAPP already defines Committed as purchase-order lines from GSAP (015,
  D-243; the simulated GSAP's POs since 033, D-331) and the plan page, All work and Home show it; the SP&A
  view uses the **same** figures so one plan never shows two committed totals. Actual spend is 015's Paid.
  Purchase-request amounts are shown in the procurement panel under their own names: **Approved**
  (Approved and Closed requests — Closed at their closed-out amount), **Pending approval** (Submitted, In
  review) and **Draft** (Created); Declined requests are listed but count nowhere.
- Q: How is planned cost allocated to periods and to the plan's year? → A: D-336 — by the rule All work's
  Budget & cost and the plan's Budget by month already use (010 FR-023, 016 FR-024): each line item by its
  spread, and an activity's positive unassigned balance evenly over the activity's months. Only months in
  the plan's year count; amounts in other years are left out of every SP&A figure and a note gives them.
  Transactions count in the month of their posting date, and only in the plan's year.
- Q: Which dimensions does "Spend by channel or category" offer? → A: D-336 — Activity type and every
  active choice or yes/no Shell attribute of the activity and line-item levels (the groupings All work's
  Breakdown offers at those levels, D-270), with Channel / audience first and the default. A % split
  divides the amount by its shares; several values without shares each count the whole amount, as in the
  Breakdown, and the view says so. Line-item dimensions show an activity's unassigned planned cost as "Not
  yet itemised". MAPP has no "Activity category" attribute (Hive9's tactic category); it appears by itself
  if Shell adds one in Settings.
- Q: What does the switch "Actual / Actual @ Monthly FX" (006 FR-007) do in MAPP? → A: D-337 — not built:
  MAPP converts every amount at one current rate per currency (D-258–D-261, D-303), so there is no monthly
  rate to switch to. Recorded as deferred.
- Q: Are the BCPIR view, the yearly register, the BCPIR publication warning and export built? → A: D-337 —
  no. The BCPIR view (006 US3, FR-022–FR-029) needs performance data (attributed revenue, leads, pipeline),
  which stays out with Measure (D-333). The register (FR-030–FR-031) shows return figures per plan and is
  deferred with it; finding BCPIR plans is covered by the marker and filter on the Plans list and in All
  work. The publication warning (FR-035) belongs to the business case and waits with the BCPIR view. Export
  (FR-039) waits for feature 036's export of plans and dashboards. The finance-budget link of the budget
  panel (FR-018, Finance module) and the Plan Budget dashboard link (FR-021, Measure) are out of scope.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review SP&A spend for a plan or campaign (Priority: P1)

A team leader or local finance advisor opens the SP&A view of a plan, or of one of its campaigns. Headline
figures show planned cost, committed, actual spend, remaining, utilisation and variance; spend by quarter
(or by month) compares planned and actual; spend by a chosen dimension shows where the planned cost goes;
a table lists every activity with its planned, committed and actual amounts. Every figure can be traced to
the activities behind it (006 US1).

**Why this priority**: SP&A is the spend side of Shell's marketing plans; the project plan names "Plan
module – BCPIR & SP&A" as a deliverable, and spend pacing against plan is reviewed every month.

**Independent Test**: Open the SP&A view of a seeded plan and of one of its campaigns and compare every
headline figure, period, dimension row and table row with a hand calculation from its activities, line-item
spreads and GSAP transactions.

**Acceptance Scenarios**:

1. **Given** a campaign with Activity A (planned cost 340,000.00 USD) and Activity B (60,000.00 USD), where
   A's line items have purchase-order lines of 200,000.00 and payments of 150,000.00 in the plan's year and
   B has none, **When** the user opens the campaign's SP&A view, **Then** it shows Planned cost 400,000.00,
   Committed 200,000.00, Actual spend 150,000.00, Remaining 250,000.00, Utilisation 37.5 % and Variance
   −62.5 %.
2. **Given** the same campaign with payments of 450,000.00 on A, **Then** Remaining shows −50,000.00 and
   Utilisation 112.5 %, both marked as overspend, and Variance shows +12.5 %, marked as overspend.
3. **Given** a scope whose activities all have planned cost 0.00, **Then** Utilisation and Variance show "—"
   with the reason "No planned cost".
4. **Given** A's line items spread 200,000.00 to Q3 and 140,000.00 to Q4, B's spread 60,000.00 to Q4, and
   150,000.00 of payments posted in Q3, **When** the user looks at Spend by quarter, **Then** Q3 shows planned
   200,000.00 and actual 150,000.00, and Q4 planned 200,000.00 and actual 0.00; **When** the user switches to
   months, **Then** each month shows its own planned and actual amounts and the months add up to the
   quarters.
5. **Given** A's Country / OU is split China 60 % / India 40 % and B's is China 100 %, **When** the user
   chooses the dimension Country / OU, **Then** China shows 264,000.00 and India 136,000.00, totalling the
   400,000.00 planned cost.
6. **Given** the dimension Spend type and A has 300,000.00 on line items of spend type "Brand" and 40,000.00
   not on a line item, **Then** the view shows Brand 300,000.00 and "Not yet itemised" 40,000.00, and B's
   amounts by its own line items.
7. **Given** the activity table, **When** the user sorts by Actual spend, **Then** the rows are ordered by it;
   **When** the user opens a row, **Then** its line items appear with their own planned, committed and actual
   amounts; **When** the user selects the activity's name, **Then** the activity's page opens.
8. **Given** the headline figure Committed of 200,000.00, **When** the user selects it, **Then** the
   activities that make it up are listed with their committed amounts, which total 200,000.00; the same
   holds for every headline figure, every period and every dimension row.
9. **Given** the view, **Then** it states when GSAP data was last received ("Finance data as of 21 Oct 2026,
   14:05"), or that none was received yet.
10. **Given** the user's display currency is EUR, **Then** every amount is shown in EUR with its code, each
    figure converted once from its USD value.
11. **Given** the SP&A view of the plan, **When** the user chooses one of its campaigns in the scope picker,
    **Then** the view shows that campaign's figures and the address keeps the choice, so Back returns to the
    whole plan.

---

### User Story 2 - Mark and find BCPIR plans (Priority: P1)

A business champion ticks "BCPIR plan" on a plan (feature 004's form). From then on the plan carries a
BCPIR marker wherever plans are listed or shown and can be found with one filter (006 US2).

**Why this priority**: The flag is on every Shell plan form and is a filter in Hive9's Plan and Measure
screens; without the marker BCPIR plans cannot be told apart.

**Independent Test**: Tick and untick the flag on two plans and check the marker and the filter on the
Plans list, the plan page and All work, and the activity log entry.

**Acceptance Scenarios**:

1. **Given** a plan with "BCPIR plan" ticked and another without it, **When** the user filters the Plans
   list to "BCPIR plans: Only", **Then** only the first plan is listed, with a BCPIR marker.
2. **Given** the filter "BCPIR plans: Exclude", **Then** only plans without the flag are listed, including
   plans on which it was never set.
3. **Given** the flag is removed from the first plan, **When** the list is refreshed, **Then** the plan has no
   marker, is no longer listed under "Only", and the plan's history shows the change "BCPIR plan: Yes → No".
4. **Given** a BCPIR plan, **When** its page opens, **Then** the header shows the BCPIR marker beside the
   plan's status.
5. **Given** All work, **When** the user filters "BCPIR plans: Only", **Then** only BCPIR plans and the
   records under them are shown, their plan rows carry the marker, and the filter appears as a chip that
   can be removed; **When** the user chooses the Breakdown "BCPIR plan", **Then** planned cost is grouped
   into Yes and No.

---

### User Story 3 - Reconcile spend with the budget and purchase requests (Priority: P2)

From the SP&A view a finance advisor compares planned cost with the budget, sees how much planned cost is
linked to a finance budget item, follows the purchase requests raised on the scope's line items and starts
a purchase request from an activity (006 US4).

**Why this priority**: Keeping planning and funding in step, and seeing commitment before it becomes
actual spend, are the main finance uses of SP&A.

**Independent Test**: With a campaign budget, line items linked and not linked to finance budget items and
purchase requests in known statuses, open the SP&A view and compare both panels with a hand calculation;
open a purchase request and raise a new one from an activity row.

**Acceptance Scenarios**:

1. **Given** a campaign with budget 2,200,000.00 and planned cost 400,000.00, **Then** the budget panel
   shows Budget 2,200,000.00, Planned cost 400,000.00, Committed 200,000.00 and Planned vs budget −81.8 %.
2. **Given** A has 300,000.00 on line items linked to a finance budget item and 40,000.00 not on a line
   item, and B's 60,000.00 is on a line item without one, **Then** the panel shows Linked to a finance budget
   item 300,000.00 and Not linked 100,000.00.
3. **Given** a scope without a budget, **Then** Budget shows "No budget set" and Planned vs budget "—".
4. **Given** three purchase requests on the scope's line items — Approved for 200,000.00, Submitted for
   50,000.00 and Created for 10,000.00 — and 120,000.00 invoiced, **Then** the procurement panel shows 3
   purchase requests, Approved 200,000.00, Pending approval 50,000.00, Draft 10,000.00 and Invoiced
   120,000.00, and lists each request with its name, vendor, status and amount.
5. **Given** a Declined purchase request on the scope, **Then** it is listed with its status and counts in
   none of the amounts.
6. **Given** the procurement panel, **When** the user selects a purchase request, **Then** its page opens.
7. **Given** Activity A with two line items, **When** the user chooses "Raise purchase request" on A's row,
   **Then** A's line items are offered; choosing one starts feature 032's New purchase request form for it.
8. **Given** an activity without line items, or a person without the right to raise purchase requests in
   the plan, **Then** "Raise purchase request" is unavailable and says why.

### Edge Cases

- **Scope with no activities**: every amount is 0.00, Utilisation and Variance "—", and the view says "This
  plan has no activities yet" (or "This campaign …") with a link to the plan tree.
- **Division by zero**: Utilisation and Variance show "—" when planned cost is 0.00; Planned vs budget shows
  "—" with "No budget set" when the budget is empty or 0.00.
- **Activity without dates**: its unassigned balance cannot be placed in a month; it is shown as "Not yet
  scheduled" beside the periods, so periods plus Not yet scheduled equal the planned cost.
- **Line items above the activity's planned cost** (negative unassigned balance, §4.3): Not yet scheduled is
  negative and shown in red; the totals still reconcile.
- **Multi-year activities** (006 FR-009): only planned cost spread to months of the plan's year counts, and
  only transactions posted in that year; a note gives the planned cost and the transactions left out.
- **Activity attribute not filled in**: its amounts appear under "Not filled in".
- **Several currencies**: every amount is converted to USD at the current rate of its currency (015 FR-010,
  D-303), added up, then converted once to the display currency.
- **Transactions in a currency no longer in Settings** (D-257): counted nowhere, as on the plan page.
- **Records in the recycle bin** are left out until restored; their transactions and purchase requests too.
- **A person without access to financial data in the plan** (014 `finance.view`): Committed, Actual spend,
  Remaining, Utilisation, Variance and Invoiced show "—" with "You don't have access to financial data in
  this plan"; planned figures are shown.
- **Purchase requests the person cannot see** (032 FR-050) are left out of the procurement panel.
- **A campaign that is not in the plan, or a plan outside the person's access**: the view answers "not
  found", like a plan that does not exist.
- **BCPIR flag removed**: the marker disappears at once everywhere.

## Requirements *(mandatory)*

### Functional Requirements

**BCPIR identification** (006 FR-001–FR-003)

- **FR-001**: The plan's "BCPIR plan" yes/no (field catalogue §1 #11, owned by 004) MUST mark the plan as a
  BCPIR plan when Yes; No or never set is not a BCPIR plan.
- **FR-002**: A BCPIR plan MUST carry a BCPIR marker on the Plans list, in the plan page's header and on its
  plan row in All work.
- **FR-003**: The Plans list and All work MUST offer the filter "BCPIR plans" with Include (default), Only and
  Exclude; All work's filter MUST appear as a chip, be kept in the address and in saved views, and be named
  in its export notes. All work's Breakdown MUST offer the grouping "BCPIR plan" (Yes / No).
- **FR-004**: Changing the flag MUST take effect at once in every list and view, and MUST be recorded in the
  activity log (already done by 004).

**Scope, data and presentation** (006 FR-004–FR-012)

- **FR-005**: The SP&A view MUST be available for a plan and for each of its campaigns; a scope picker MUST
  switch between "Whole plan" and each campaign, keeping the choice in the address.
- **FR-006**: Every figure MUST be computed on the server from the live plan tree (004–007), the GSAP
  transactions (015, 033) and the purchase requests (032). Nothing is estimated; where data is missing the
  view MUST show "—" with the reason.
- **FR-007**: Every headline figure, period and dimension row MUST be traceable: selecting it MUST list the
  contributing activities with their amounts, which sum to the figure (in a display currency other than USD each
  amount is converted once, so a list may differ from its figure by a cent, as everywhere in the app, D-061).
- **FR-008**: Money MUST be shown in the person's display currency with its code; each figure is worked out
  in USD and converted once (D-010, D-220).
- **FR-009**: The view MUST show when GSAP data was last received ("Finance data as of …"), or "No finance
  data received yet".
- **FR-010**: Only planned cost spread to months of the plan's year and only transactions posted in that year
  MUST count (D-336); a note MUST give what was left out when it is not zero.
- **FR-011**: Records in the recycle bin MUST be left out.
- **FR-012**: The view MUST be shown only for plans the person can see (014); any other plan, or a campaign
  that is not in the plan, MUST answer "not found". Financial figures need `finance.view` in the plan.
- **FR-013**: The view MUST be read-only; activities, line items and purchase requests are changed in their
  own pages, reached from the view by link.
- **FR-014**: The view MUST show the plan's status (Draft or Published) and BCPIR marker in its header and
  link to the plan and to All work filtered to the plan.

**SP&A figures** (006 FR-013–FR-017, Measure Definitions)

- **FR-015**: Headline figures MUST be Planned cost, Committed, Actual spend, Remaining = Planned cost −
  Actual spend, Utilisation % = Actual spend ÷ Planned cost × 100 and Variance % = (Actual spend − Planned
  cost) ÷ Planned cost × 100, percentages with one decimal rounded half up; Utilisation and Variance are
  "—" when planned cost is 0.00.
- **FR-016**: Remaining below zero, Utilisation above 100 % and Variance above 0 % MUST be marked as
  overspend (in red with the word "Overspend", never colour alone).
- **FR-017**: "Spend by quarter" MUST show, for each quarter of the plan's year, the planned cost allocated
  to it (D-336) and the actual spend posted in it, with a switch to months, and "Not yet scheduled" when not
  zero; periods plus Not yet scheduled MUST equal the planned cost.
  *Amended by [038 FR-023](../038-qa-findings/spec.md) (D-363, QA M-2): Not yet scheduled is worked out per activity
  and is never negative; line items above an activity's planned cost show as a row "Line items above planned cost",
  so periods + Not yet scheduled − that row = planned cost.*
- **FR-018**: "Spend by dimension" MUST show the planned cost by the dimension the person chooses (D-336),
  largest first, each with its share of the planned cost, "Not filled in" and "Not yet itemised" rows when
  not zero, and say when activities count in several rows.
- **FR-019**: The activity table MUST list every activity in scope with its campaign, initiative, owner,
  activity type, Channel / audience, start quarter, status, planned cost, committed, actual spend and
  variance %; it MUST be sortable by every column, each row MUST open to its line items with their planned
  cost, committed and actual spend, and the activity's name MUST link to its page.

**Budget and procurement** (006 FR-018–FR-020)

- **FR-020**: The budget panel MUST show the scope's budget (the plan's budget, or the campaign's), Planned
  cost, Committed and Planned vs budget % = (Planned cost − Budget) ÷ Budget × 100, and the planned cost on
  line items linked to a finance budget item (007) and the rest, "Not linked" (which includes unassigned
  planned cost).
- **FR-021**: The procurement panel MUST show the number of purchase requests on the scope's line items that
  the person can see, their Approved, Pending approval and Draft amounts (D-335) — counting only their lines
  on the scope's line items — and the Invoiced amount (015); it MUST list each request with its ID, name,
  vendor, status and amount, and open a selected request on its page.
- **FR-022**: Each activity row MUST offer "Raise purchase request", which lists the activity's line items
  and opens feature 032's New purchase request form for the one chosen; when unavailable it MUST say why.

### Not built in this feature (D-337)

006 FR-007 (Actual @ Monthly FX), FR-018's link to the finance budget (Finance module), FR-021's link to the
Plan Budget dashboard (Measure), FR-022–FR-029 (BCPIR view), FR-030–FR-031 (register), FR-033's BCPIR-view
offer, FR-034's publication date (MAPP stores no publication date), FR-035 (BCPIR publication warning),
FR-036–FR-038 (Model B), FR-039 (export, with 036) and FR-041 (financial-information restriction, already
covered by 014's `finance.view`).

### Key Entities

- **BCPIR flag**: the plan's existing "BCPIR plan" yes/no (004); nothing new is stored.
- **SP&A view**: a read-only, computed view of a plan or campaign — headline figures, periods, dimension
  rows, activity table, budget panel and procurement panel. It stores nothing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every seeded plan and campaign, 100 % of SP&A figures equal a hand calculation to the cent:
  periods plus Not yet scheduled, and dimension rows without multi-value activities, add up to the planned
  cost.
- **SC-002**: 100 % of headline figures, periods and dimension rows list contributing activities whose
  amounts sum exactly to the figure.
- **SC-003**: The plan's Planned cost and Committed in the SP&A view equal those on the plan page whenever no
  planned cost or transaction falls outside the plan's year.
- **SC-004**: From the Plans list, a user finds all BCPIR plans of a year in at most two actions.
- **SC-005**: A first-time user finds a campaign's actual spend in Q3 within 60 seconds of opening its plan,
  in at least 90 % of attempts (design system §10).
- **SC-006**: The SP&A view of a plan with 2,000 activities answers within 2 seconds at p95 with one read
  transaction and no statement per activity.

## Assumptions

- Model A (D-334); SP&A is offered for every plan and campaign, whatever the plan's status.
- Quarters are calendar quarters of the plan's year (D-011, Q-004).
- The default dimension is the activity attribute "Channel / audience" when Settings has it active.
- The SP&A view needs only `plans.view` in the plan (as the plan page); financial figures need
  `finance.view`, and the procurement panel shows only purchase requests the person can see (032 FR-050).
- Demo data: "FY2026 Lubricants Europe" is marked as a BCPIR plan, so the marker and the filter show in a
  demo; its GSAP transactions and PR-1001 fill the SP&A view.
