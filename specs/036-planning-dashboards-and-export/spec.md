# Feature Specification: Planning Dashboards and Export to PowerPoint and PDF

**Feature Branch**: `feature/036-planning-dashboards-and-export`

**Created**: 2026-10-28

**Status**: Draft

**Input**: User description: "Executive and regional planning dashboards built only from the plan and finance
data SP&A One holds — budget, planned cost, committed, actual, unspent budget and how much of the budget is
spent, by market, plan and campaign and by month; approvals waiting and how long decisions take; purchase
requests by status and amount; how complete the plan details are — filtered by year, market, plan type and
display currency, and limited to what each person may see. Plans and dashboards can be exported to PowerPoint
and PDF files that download to the person's computer." (feature map row 036, D-333)

**References**: Constitution 4.6 — Current Phase & Scope (036 in scope, Measure's performance reporting out),
Principles IV (the server works out every figure), VII (scope isolation), IX, X, F6 (every chart has a table) ·
RFP §8 ("view all the content from a central dashboard with controls at access level — Global can view all
data for all markets vs local market can view all data for their market"; "executive dashboards, regional and
operating-unit dashboards, campaign, portfolio and budget dashboards"; "export our plans into any format (ppt,
pdf, …)") · Hive9 Measure Module Analysis §1.3 (UC-04, UC-06, UC-07, UC-09), §3.4 Plan Budget, §3.6
Approvals Dashboard, §3.7 PR Dashboard, §3.9 Plan Details · Shell's client specs 017 (Measure financial
reports: FR-005–FR-009, FR-018, FR-019, FR-021) and 018 (Measure operational dashboards: FR-012, FR-018,
FR-019, FR-028, FR-031) — only the parts computable from data SP&A One holds · `docs/product/domain-model.md`
§4 · features 009 (Home), 010 (All work and its Budget & cost view), 014 (access, global budgets), 015
(GSAP transactions), 016 (plan type, plan currency), 017 and 031 (approval rounds), 032 and 033 (purchase
requests).

## Clarifications

### Session 2026-10-28

The product owner is not available; each question is answered with the best evidence and recorded as a
decision (D-342–D-345) or an open question (Q-022, Q-023).

- Q: What are a "region", a "market" and an "operating unit" in SP&A One? → A: a **team** (D-237: "a market
  is a team"). Plans carry their teams, and access scopes are teams, so the regional dashboard is the
  dashboard of one team. Grouping by the Shell attributes Cluster, Region and Country/OU instead is left to
  Shell (Q-022).
- Q: What does "actual" mean? → A: what GSAP reports as **paid** — Shell's "actuals are the payments
  released" (015 spec; the Financials page's "Paid (actuals)"). Committed is the purchase-order value (015).
- Q: How are "remaining" and "utilisation" defined, given that "Remaining budget" already means a parent's
  budget minus its children's? → A: two new words so nothing is defined twice: **Unspent budget** = budget −
  actual (client 017 FR-019 "Remaining Budget") and **Budget spent** = actual ÷ budget as a whole percent
  (client 017 FR-018 "Burn rate"), rounded as Budget used is. Budget used (planned ÷ budget) and Left to plan
  (budget − planned) keep their meaning (D-342).
- Q: Which exchange rate converts actuals by month? → A: the current rate, as every other figure in SP&A One
  (D-258–D-261). Client 017's "Actual @ Monthly FX" is not offered (Q-023).
- Q: Who may open the dashboards and who may export? → A: anyone who may view plans opens them; every figure
  covers only the plans the person may see; financial figures cover only the plans where they may see
  financial data; global budgets appear only to people who may see global budgets. Exporting a plan or a
  dashboard needs the existing export permission, `data.export` (D-343); every built-in role holds it.
- Q: Where are the files made, and are they kept or sent anywhere? → A: SP&A One makes the file when asked
  and hands it to the browser to save; nothing is kept, e-mailed or shared (D-344, ADR-0017). Word and Excel
  are not part of this feature: All work already exports to Excel (010), and Word adds nothing a PDF does not
  give a reader.
- Q: Are dashboard filters saved per person, as Hive9 does? → A: no; the filters live in the page address, so
  a dashboard can be bookmarked and shared as a link without saving anything (constitution F3). Hive9's
  persisted filters are a known source of "why is my dashboard empty?" calls (Measure analysis §3.0.3).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See the planning picture of every market I may see (Priority: P1)

Carlos, a global planner, opens **Dashboards** in the sidebar. The **Executive dashboard** shows, for the
current year and every market he may see: the total budget, the planned cost and how much of the budget it
uses, what is committed and what is actually spent, the unspent budget and the share of the budget spent.
Below, a table shows the same figures **by market** and another **by plan**; a chart shows **planned cost and
actual spend by month** of the year, with a table view; two short lists show the **campaigns most over
budget** and the **campaigns with the most budget left to plan**; panels show **approvals** (how many wait,
since when, how many were decided and how many days a decision takes on average), **purchase requests** (how
many and how much in each status), **plan details** (the share of records with every needed field filled in)
and, for people who may see them, the **global budgets** of the year with what is allocated. He changes the
year, the plan type and the display currency, and every figure follows.

**Why this priority**: the RFP asks for a central dashboard with controls at access level and for executive
dashboards (RFP §8); leaders today have to open every plan to see where the money is.

**Independent Test**: sign in as people with different scopes, open the Executive dashboard for a year and
compare each figure with the plan pages, All work's totals and Budget & cost view, the Financials page, the
Approvals monitoring and the Purchase requests page for the same plans.

**Acceptance Scenarios**:

1. **Given** Carlos may see every market and the demo data, **When** he opens the Executive dashboard for
   2026 in US dollars, **Then** the total budget is the sum of the budgets of FY2026 Lubricants Europe and
   FY2026 Mobility China, the planned cost is the sum of their planned costs as their plan pages show them,
   and Budget used is planned ÷ budget as a whole percent.
2. **Given** Priya may see only Lubricants Europe, **When** she opens the Executive dashboard for 2026,
   **Then** it covers only FY2026 Lubricants Europe: no figure, row, campaign, approval or purchase request of
   Mobility China counts anywhere on it.
3. **Given** a plan whose line items have GSAP purchase orders and payments, **When** the dashboard is shown,
   **Then** Committed and Actual equal the plan's Committed and Paid on the Financials figures (015), Unspent
   budget is budget − actual and Budget spent is actual ÷ budget as a whole percent.
4. **Given** a person who may view plans but not see financial data, **When** they open the dashboard, **Then**
   Committed, Actual, Unspent budget, Budget spent and the actual line of the monthly chart show "—" with
   "You can't see financial data" and every other figure is shown.
5. **Given** a plan shared by two markets, **When** the By market table is shown, **Then** the plan counts in
   both markets' rows and once in the totals, and the table says so.
6. **Given** a year, **When** the monthly chart is shown, **Then** each month's planned cost is that month's
   amount in All work's Budget & cost view for the same plans, amounts not yet spread show as "No dates", each
   month's actual is the payments posted in that month, and the table view lists every month with its
   running totals.
7. **Given** campaigns whose planned cost is above their budget, **When** the dashboard is shown, **Then** the
   "Most over budget" list shows up to five of them, largest overspend first, each with its plan, budget,
   planned cost and how much over; "Most left to plan" shows up to five campaigns with the largest budget not
   yet planned; campaigns without a budget appear in neither.
8. **Given** activities and purchase requests of the plans in view with approval rounds, **When** the
   Approvals panel is shown, **Then** it shows how many wait for a decision and since when the oldest waits,
   how many were approved and declined, how many were approved automatically, and the average number of days
   from submission to decision to one decimal, worked out as the Approvals monitoring works it out (017).
9. **Given** purchase requests on the plans in view, **When** the Purchase requests panel is shown, **Then**
   it shows the count and the amount of each status (Created, Submitted, In review, Declined, Approved,
   Closed), the total, and how many approved ones wait for GSAP, amounts as the Purchase requests page shows
   them.
10. **Given** the plans in view, **When** the Plan details panel is shown, **Then** it shows, for plans,
    campaigns, initiatives, activities and line items, how many there are, how many have every needed field
    filled in and that share as a whole percent.
11. **Given** a person who may see global budgets, **When** the dashboard for a year is shown, **Then** a
    Global budgets panel lists the year's global budgets with their amount, allocated, unallocated and the
    planned cost of the allocated plans, as the Global budgets page shows them; people who may not see global
    budgets do not see the panel.
12. **Given** the dashboard, **When** the person changes the year, the plan type or the display currency,
    **Then** every figure is worked out again for that choice and the page address keeps the choice.
13. **Given** a year with no plans the person may see, **When** the dashboard is shown, **Then** it says "No
    plans to show for 2027" with what to change, and shows no zeros as if they were figures.

---

### User Story 2 - Export a plan to PowerPoint or PDF (Priority: P1)

Priya opens FY2026 Lubricants Europe and chooses **Export › PowerPoint**. A file "SP&A One – FY2026 Lubricants
Europe – 28 Oct 2026.pptx" downloads. It opens with a title slide (plan, year, status, owner, markets, display
currency, when it was made and how current the GSAP data is), a summary slide (budget, planned cost,
committed, actual, unspent budget, budget spent), a campaigns slide (each campaign's budget, planned cost,
committed and actual) with a chart of budget against planned cost and actual, and the activities as tables
(name, campaign, status, dates, planned cost, committed, actual) over as many slides as they need. **Export ›
PDF** gives the same content as a PDF document of the same pages.

**Why this priority**: marketers present plans in PowerPoint today and copy figures by hand; the RFP asks to
export plans to ppt and pdf (RFP §8).

**Independent Test**: export a plan to both formats, open both files, and compare every figure with the plan
page.

**Acceptance Scenarios**:

1. **Given** a plan Priya may see and export, **When** she chooses Export › PowerPoint, **Then** a .pptx file
   named "SP&A One – <plan name> – <today>.pptx" downloads and opens in PowerPoint with the slides above.
2. **Given** the same plan, **When** she chooses Export › PDF, **Then** a .pdf file with the same pages
   downloads and opens in any PDF reader.
3. **Given** the export, **When** it is compared with the plan page, **Then** every amount is the plan page's
   amount in the person's display currency, labelled with the currency.
4. **Given** a plan with more activities than fit on one slide, **When** it is exported, **Then** the
   activities continue on the next slides, each titled "Activities (continued)" with its page of rows.
5. **Given** a person who may not see financial data in the plan, **When** they export it, **Then** the
   committed and actual figures show "—" with the note "You can't see financial data".
6. **Given** a person without the export permission, **When** they open a plan, **Then** no Export choice is
   offered, and asking for the file directly is refused with a plain message.
7. **Given** a plan the person may not see, **When** its export is asked for, **Then** the answer is the same
   "not found" as for a plan that does not exist.
8. **Given** the export is being made, **When** it takes longer than a second, **Then** the menu shows it is
   working and a second click does not start a second export; when it fails, the person sees why.

---

### User Story 3 - Look at one market and drill into its plans and campaigns (Priority: P2)

Daniel, a team leader in Lubricants Europe, opens the **Regional dashboard**. It shows the same panels as the
Executive dashboard for one market — his own by default — with a table of the market's plans. He selects a
plan in the table: a **campaigns** table shows each campaign of the plan with its budget, planned cost,
committed, actual and left to plan, each campaign linking to its page. Executives can open the Regional
dashboard of a market from its row in the Executive dashboard.

**Why this priority**: the RFP asks for regional and operating-unit dashboards; it builds on story 1's figures.

**Independent Test**: open the Regional dashboard for a market, compare its figures with the Executive
dashboard's row for that market, open a plan's campaigns and compare them with the plan page.

**Acceptance Scenarios**:

1. **Given** Daniel belongs to Lubricants Europe, **When** he opens the Regional dashboard without choosing a
   market, **Then** it shows Lubricants Europe, and his other teams are offered in the Market filter.
2. **Given** the Executive dashboard, **When** Carlos selects the row of Mobility China, **Then** the Regional
   dashboard of Mobility China opens with the same year, plan type and currency, and its totals equal that
   row.
3. **Given** the Regional dashboard, **When** a plan is chosen in its plans table, **Then** the campaigns table
   lists every campaign of that plan in the plan's order with budget, planned cost, committed, actual and left
   to plan, and each campaign name opens the campaign.
4. **Given** a market with no plans the person may see in the year, **When** its Regional dashboard is shown,
   **Then** it says "No plans of Mobility China to show for 2026" and offers the other choices.
5. **Given** a market the person's scope does not include, **When** they choose it, **Then** the dashboard
   shows only the plans of that market they may see — none, if they may see none — and never another
   market's figures.

---

### User Story 4 - Export a dashboard to PowerPoint or PDF (Priority: P2)

Carlos chooses **Export › PowerPoint** on the Executive dashboard. The file holds a title slide with the
filters, the summary figures, the by-market and by-plan tables, the monthly chart with its figures, the
campaigns over budget and with the most left to plan, and the approvals, purchase requests, plan details and
global budgets panels — the same figures as on screen. **Export › PDF** gives the same pages as a PDF.

**Why this priority**: leaders take dashboards into meetings; it reuses story 2's files.

**Independent Test**: export a dashboard in both formats and compare every figure with the screen.

**Acceptance Scenarios**:

1. **Given** the Executive or Regional dashboard with its filters, **When** the person exports it, **Then** a
   file named "SP&A One – Executive dashboard – 2026 – 28 Oct 2026.pptx" (or "… – Regional dashboard –
   Lubricants Europe – 2026 – …") downloads, with the filters written on the title page.
2. **Given** the export, **When** it is compared with the screen, **Then** it holds the same figures for the
   same filters and the same scope.
3. **Given** a person without the export permission, **When** the dashboard is shown, **Then** no Export
   choice is offered and the file is refused if asked for directly.

---

### Edge Cases

- A plan without a budget: its budget shows "—"; Budget used, Left to plan, Unspent budget and Budget spent of
  that plan show "—"; the totals add up the plans with a budget and say how many have none.
- No plan in view has a budget: the total budget and the percentages show "—" ("No budget set").
- Nothing planned yet: planned cost shows "—" ("Nothing planned yet"), never 0.00 as if it were a figure.
- Over budget: an amount over budget is written as "Over budget by …" as well as coloured; a percent above 100
  is written in full ("112%").
- Financial data visible in some plans only: the financial totals cover those plans and say "Financial figures
  cover 1 of 2 plans".
- Payments posted outside the chosen year: counted in Actual, shown in the monthly table as "Paid outside 2026".
- A currency whose rate was removed from Settings: its transactions are left out of every sum as on the
  Financials page (015 D-257).
- Purchase requests switched off in Settings: the Purchase requests panel says so instead of showing zeros.
- No approvals in the plans in view: the Approvals panel says "No approvals yet".
- A plan deleted (in the recycle bin): it and everything in it is not counted, as everywhere else.
- A long plan or campaign name in a file: it is shortened with "…" to fit its cell; the full name is on the
  title slide of a plan export.
- Characters a PDF's standard fonts cannot show (for example Chinese): they are replaced by "?" in the PDF
  only; the PowerPoint file keeps them.
- A plan with hundreds of activities: the export continues the table over as many pages as needed.

## Requirements *(mandatory)*

### Functional Requirements

**Dashboards area and access**

- **FR-001**: The sidebar MUST offer **Dashboards** to every person who may view plans; it opens the Executive
  dashboard, and the Regional dashboard is the second tab.
- **FR-002**: Every figure, row and list on a dashboard MUST cover only plans the person may see (014 scope),
  and everything in them; a plan outside the scope MUST NOT count anywhere, and its name MUST NOT appear.
- **FR-003**: Committed, actual, unspent budget, budget spent and monthly actuals MUST cover only the plans
  where the person may see financial data (015 FR-028, D-301). When that is none of the plans in view, they
  show "—" with "You can't see financial data"; when it is some, the dashboard states how many plans the
  financial figures cover.
- **FR-004**: The Global budgets panel MUST be shown only to people who may see global budgets, with the
  figures the Global budgets page shows for the chosen year (014 FR-032).
- **FR-005**: Dashboards MUST be read-only: nothing on them changes a record, and opening them writes nothing.

**Filters**

- **FR-006**: Both dashboards MUST offer **Year** (the plan year; default the current year; the choices are the
  years with plans the person may see, plus the current and the chosen year), **Plan type** (All, Operational, Strategic,
  Cross-market) and **Currency** (the active currencies; default the person's display currency). The Regional
  dashboard also offers **Market** (every team; default the person's first team by name, else the first team).
- **FR-007**: The filters MUST live in the page address, so Back, a bookmark and a shared link show the same
  dashboard; an unknown value falls back to its default and never breaks the page.

**Figures** (every amount worked out by SP&A One in US dollars, then converted once to the chosen currency)

- **FR-008**: **Budget** of a set of plans MUST be the sum of their plan budgets (domain model §4.1; an
  allocated plan's budget is its allocation, 014). Plans without a budget add nothing and are counted.
- **FR-009**: **Planned cost** MUST be the sum of the plans' planned costs, each the sum of its activities'
  planned costs (domain model §4.2); "—" when no activity has one.
- **FR-010**: **Budget used** MUST be planned cost ÷ budget and **Budget spent** actual ÷ budget, each as a whole
  percent rounded half up; **Left to plan** MUST be budget − planned cost and **Unspent budget** budget −
  actual; all four "—" without a budget. Planned cost above budget MUST be marked "Over budget".
- **FR-011**: **Committed** and **Actual** MUST be the purchase-order and payment figures of the plans' line items
  from their GSAP transactions, as feature 015 works them out.
- **FR-012**: The **By market** table MUST list each team with at least one plan in view, with its number of
  plans and the figures of FR-008–FR-011 for its plans; a plan with several teams counts in each of their rows
  and once in the totals.
- **FR-013**: The **By plan** table MUST list each plan in view with its status, type, markets and the figures
  of FR-008–FR-011, by name.
- **FR-014**: The **monthly** chart MUST show, for each month of the year, the planned cost spread into that
  month as All work's Budget & cost view spreads it (010 FR-023: line-item spreads and the unassigned planned
  cost), the amount not yet spread ("No dates") and spread into other years separately, and the actual paid in
  that month by the payments' posting dates; a table view MUST list each month with its running totals.
- **FR-015**: **Most over budget** MUST list up to five campaigns of the plans in view whose planned cost is
  above their budget, largest overspend first; **Most left to plan** up to five whose budget is above their
  planned cost, largest left to plan first; each with its plan, budget, planned cost and the difference.
- **FR-016**: The **Approvals** panel MUST count, over the approval rounds of the activities and purchase
  requests of the plans in view: rounds waiting for a decision, the oldest waiting since, rounds approved and
  declined, rounds approved automatically, and the average days from submission to decision to one decimal,
  as the Approvals monitoring works them out (017); and the activities by approval status.
- **FR-017**: The **Purchase requests** panel MUST show, for the purchase requests on the plans in view that
  the person may see, the count and amount in each status, the total, and how many approved ones wait for
  GSAP; amounts as the Purchase requests page works them out (032).
- **FR-018**: The **Plan details** panel MUST show, for each level from plan to line item, how many records
  there are, how many have every needed field filled in (domain model §2.6) and that share.
- **FR-019**: A dashboard MUST state the date and time GSAP data was last received, when any was (015 FR-008).

**Regional dashboard**

- **FR-020**: The Regional dashboard MUST show the panels of FR-008–FR-019 for the plans of one market.
- **FR-021**: Choosing a plan in its plans table MUST show the plan's campaigns in the plan's order with budget,
  planned cost, committed, actual and left to plan; each campaign links to its page and each plan to its page.
- **FR-022**: A market row of the Executive dashboard MUST open that market's Regional dashboard with the same
  year, plan type and currency.

**Export**

- **FR-023**: A plan's page MUST offer **Export › PowerPoint** and **Export › PDF** to people who hold the export
  permission in its plan; the file MUST hold a title page, a summary page, a campaigns page with a chart of
  budget, planned cost and actual by campaign, and the activities over as many pages as needed.
- **FR-024**: Each dashboard MUST offer **Export › PowerPoint** and **Export › PDF** to people who hold the export
  permission; the file MUST hold a title page with the filters and every panel the screen shows, with the same
  figures.
- **FR-025**: Exported files MUST be made when asked, MUST hold only what the person may see, MUST be saved by the
  browser under the name "SP&A One – <plan or dashboard> – <today>" with the format's extension, and MUST NOT be
  stored, e-mailed or shared by SP&A One.
- **FR-026**: Every chart in a file MUST be accompanied by its figures in a table, and every amount MUST carry its
  currency.
- **FR-027**: Exporting a plan or a dashboard MUST need the existing export permission (014 Table A,
  `data.export`), in the plan for a plan's export.

**Experience**

- **FR-028**: A dashboard MUST show a loading state in reserved space, an error with Retry, a no-access state and
  an empty state (no plans to show); once shown, a panel with nothing to count says why ("No approvals yet",
  "Purchase requests are switched off") instead of zeros.
- **FR-029**: Every chart MUST have an equivalent table (a "Table" view) and a text summary for assistive
  technology; colours are never the only carrier of meaning (constitution F6).

### Key Entities

- **Dashboard**: a read-only view of the plans one person may see for a year, plan type and market, with the
  figures of FR-008–FR-019 in one currency. Not stored.
- **Dashboard figures**: budget, planned cost, committed, actual and the derived budget used, left to plan,
  unspent budget and budget spent of a set of plans, a market, a plan or a campaign.
- **Exported file**: a PowerPoint or PDF document made from a plan or a dashboard for one person at one moment.
  Not stored.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: For every seeded person and year, each dashboard total equals the sum of the same figure on the
  plan pages (budget, planned cost) and the Financials figures (committed, actual) of the plans that person may
  see — in 100% of automated comparisons.
- **SC-002**: A person whose scope is one market never sees a figure, name or count of another market on a
  dashboard or in an exported file (100% of scope tests).
- **SC-003**: The Executive dashboard for one year of the reference volume (feature 028: 50 plans, 5,000
  activities, 20,000 line items) is shown within 3 seconds, and a plan's export is ready within 5 seconds.
- **SC-004**: A first-time user finds the budget, planned cost and spend of their market in under one minute
  without help (design system §10 usability target).
- **SC-005**: Every exported .pptx opens in PowerPoint and every .pdf in a standard PDF reader without a repair
  prompt, and every figure in them matches the screen.

## Assumptions

- The demo data (feature 001 seed, extended by 014–033) is enough to demonstrate every panel; no seed change is
  needed.
- Plans of other years than the chosen one never count, even when payments or approvals are dated in that year.
- Budgets phased by month (016) are not drawn on the monthly chart: the chart compares planned cost with actual
  spend, as client 017's Plan Budget chart does; the phased budget stays on the plan page.
- Finance budgets, forecasts, goals, ROI, revenue and other Measure performance figures are out of scope
  (constitution 4.6.0).
- Word and Excel exports of plans are not part of this feature (All work exports to Excel since 010).
- Exports are not recorded in the activity log: they change no record (as All work's export, 010).
