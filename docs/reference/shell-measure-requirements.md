# Shell's Measure requirements (client specs 016–020, 025) — condensed

Shell wrote six client specifications for the Hive9 Measure module on 17 October 2026 (features 016–020 and the
proposed 025). The originals are kept outside the repository; this file condenses them so that nothing here
depends on them. It is a **source, not MAPP's requirements**: MAPP's requirements are the specs under `specs/`
and the product documents under `docs/product/`. Cite a line as "client 016 FR-012" (spec number, then the
identifier the client spec uses). Every identifier is kept; every sentence is paraphrased. Where the client
specs say Program or Tactic, this file says Initiative or Activity.

## How the six specs fit together

| Spec | Hive9 screens covered | Depends on | MAPP wave |
|---|---|---|---|
| 016 | Summary, Revenue, Waterfall, plus the shared report frame (filter panel, saved views, period, share, export) | Plan hierarchy and goals, line items, actual cost, configuration (Models, attributes, categories), performance data (020) | B — needs performance data |
| 017 | Plan Budget; Finance Budget | 016's frame, ranked list and cards; plan budgets, allocations, actual cost; finance budgets | A for Plan Budget; C for Finance Budget (finance budgets module out of scope) |
| 018 | Approvals Dashboard; PR Dashboard; AR Invoice Details | 016's filter panel and saved views; approvals, purchase requests, simulated GSAP purchase orders and receipts; accruals | A for Approvals and PR Dashboard; C for AR Invoice Details (accruals, invoices) |
| 019 | Plan Details (five pages); China Budget Reports (three pages); nine single-page dashboards | A configuration-driven dashboard engine; plan, finance, people and GSAP data; 020 for the Is integrated flag | A for Plan Details, SWBS Not Received, Pending PR, PR Track Invoices and User Detail Report; C for China Budget Reports and the rest |
| 020 | Data sources, column mappings, uploads, rows to review, the Activity's Performance and Data sources areas, media codes, attribution view | Configuration (Models, external-name rule, media-code attributes), approvals (approved state), plan hierarchy, copy rules | B — the performance data itself |
| 025 | Dashboard builder, read-only viewer, snapshots, dataset upload and joins | Definitions of 016–020, sharing with people and teams, recycle bin, access rules | C — proposed, not planned |

A = buildable from data MAPP already holds; B = needs performance data; C = blocked by out-of-scope modules.

## Client 016 — Summary, Revenue and Waterfall

### User stories

- US1 (P1) — See whether the year is on track: Summary shows projected revenue against goal with a variance label, a trend split at today, and each funnel stage's volume and conversion against goal.
- US2 (P1) — Filter once and keep my filter state: one filter panel with the Plan workspace's attribute set, applied on Update, shared by every standard report, kept across sessions, with an always-visible summary and Reset.
- US3 (P1) — Rank and drill revenue performance: Revenue shows summary cards, a trend chart with its data table, and a two-level ranked list with drill, Back and Open in Plan.
- US4 (P2) — Find the leading segments on Summary: four league tables, each with its own segment selector, refreshing independently.
- US5 (P2) — Diagnose an under-performing funnel stage: Waterfall ranks by a chosen stage's volume, low to high, and drills to the activities behind a shortfall.
- US6 (P2) — Understand why a report is empty or incomplete: the report names the cause in plain words, with counts.
- US7 (P2) — Share and export a report: send the report as configured by e-mail; export one chart or ranked list.
- US8 (P2) — Save, reuse and share report views: named views, a default view, sharing with a team.
- US9 (P3) — Configure the summary cards: show, hide and reorder the cards, kept per user.

### Requirements

**General rules**
- FR-001 — Summary, Revenue and Waterfall are read-only; the only stored things are per-user report preferences.
- FR-002 — Every figure comes from real plan, finance and performance data; no demo or generated figures outside development seed data.
- FR-003 — Shell vocabulary throughout; "Activity category" names the grouping of activity types; stage names come from each plan's Model.
- FR-004 — Behaviour is defined for the acting user over every reachable plan; later access rules must not change any metric definition.

**Shared report frame — filter panel**
- FR-005 — Panel regions in order: Saved views, Year, Plan, Attributes (one tab per level), Line item, Show levels, Update; the attribute set equals the Plan workspace's.
- FR-006 — Year is a multi-select of fiscal years that have plans; it drives the time axis and restricts the plan list.
- FR-007 — Plan is a searchable multi-select with a "selected / available" counter, Select all and Clear all.
- FR-008 — Attribute groups are generated from configuration, one per pick-list attribute per level, identified by level and attribute, never merged by label.
- FR-009 — An Owner list per level and an Activity type list with per-type counts, both limited to the selected plans.
- FR-010 — Plan view is a single choice (All, Strategic/AOP only, Exclude Strategic/AOP), shown only when that organisation setting is on.
- FR-011 — Date sets: an optional from–to range per activity date type, each clearable, none set by default, with a counter of ranges set.
- FR-012 — Status lists the six activity statuses (Created, Submitted, Declined, Approved, In progress, Complete), all ticked by default.
- FR-013 — Show levels is a single choice: Any levels (default), All levels only, No initiatives, No activities, as in the Plan workspace.
- FR-014 — Every optional attribute group ends with a "Not filled in" option matching records without a value; counters include it.
- FR-015 — Select all and Clear all exist globally, per tab and per group; a group with more than ten options gets a search box.
- FR-016 — Selections cascade (years restrict plans; plans restrict owners and types; dependent attributes follow their parent); a newly selected plan selects all its values.
- FR-017 — Filter changes apply only on Update, which refreshes every widget; until then the old results stay and unapplied changes are flagged.
- FR-018 — The applied state is stored per user, shared by all standard reports and kept across sessions; a default saved view applies at session start, otherwise the last state.
- FR-019 — An always-visible filters-applied summary (years, plan count, each restricted group) with one-click Reset.
- FR-020 — Default state: current fiscal year, all its plans, all values, all statuses, Any levels, All plans, no date ranges.

**Shared report frame — controls and time semantics**
- FR-021 — Period choice Monthly or Quarterly (default Monthly) from the fiscal calendar, remembered per user per report.
- FR-022 — An Activity category selector, one category at a time, defaulting to the organisation setting; only activities whose type belongs to it count.
- FR-023 — All money in the header's display currency, always shown with its currency.
- FR-024 — Overlap rule: an object is in scope for a period when its dates overlap it at all; evergreen activities overlap from their start; each object counts once per report.
- FR-025 — Dated values count only in their period; object-level values count in full in every overlapping report; the tip says single-year totals are not additive.
- FR-026 — Period boundaries, today and date-to-period assignment use the acting user's time zone, which every report states.
- FR-027 — One year shows months or quarters; several years show earlier years as one "FY (total)" bucket each and the latest by period, with the change marked on the axis.
- FR-028 — Every time series splits at today: actual before, projected after, the current period marked "in progress" with both; a labelled Today marker.
- FR-029 — Every chart has an accessible data table and an information tip naming its metrics, projection rule, attribution model, cost basis, Models used and the overlap rule.
- FR-030 — A metric without a value shows "--", distinct from zero, and sorts after valued entries whichever order is chosen.
- FR-031 — Update reflects plan and finance changes saved before it; every performance widget states when performance data was last loaded.

**Metric definitions**
- FR-032 — Funnel stages are each in-scope plan's Model stages in order; Revenue is the money value of the final stage.
- FR-033 — Goal per stage: the entered goal-stage value, later stages derived by the Model; Goal revenue = final-stage goal × average deal size; report goals are activity roll-ups.
- FR-034 — Goals are time-phased evenly over the activity's months, later stages shifted by the Model's velocity; an evergreen goal runs to the plan year end.
- FR-035 — Actual stage volumes and revenue come from loaded performance facts credited by the designated attribution model, in the period of the fact date, up to today.
- FR-036 — Projected volumes and revenue follow 020's projection rules after today, from the goal stage forward, never past the plan year; no goal, no projection. Open: Hive9's own projection algorithm is undocumented.
- FR-037 — Actual/Projected over a span = actual to today plus projected after; this is Summary's "Projected" figure.
- FR-038 — Performance = Actual/Projected ÷ Goal as a percentage; "--" when the goal is zero or absent.
- FR-039 — Variance = (Actual/Projected − Goal) ÷ Goal, labelled Above plan, Below plan or On plan; "No goal" when the goal is absent, never "0% above".
- FR-040 — Stage conversion = stage n volume ÷ stage n−1 volume; conversion goal and conversion variance likewise; "--" on a zero denominator.
- FR-041 — ROI = (Revenue − Cost) ÷ Cost as a percentage; cost basis is actual cost (planned cost when the setting says so); "--" on zero cost; every display names model and basis.
- FR-042 — Actual cost converts at the annual planned rate; Actual at monthly FX uses the month's rate where enabled, else the annual rate with a notice; both in the transaction's period.
- FR-043 — Multi-select attributes with splits: cost measures use the cost split, goals, volumes and revenue the performance split; no recorded split means equal shares.
- FR-044 — Additive measures are summed; ratios are recomputed from the aggregated numerator and denominator, never averaged.
- FR-045 — Conversion at the annual planned rate of the relevant year; amounts without a rate are excluded and the widget says how many and which currencies.
- FR-046 — Actual total = running total of the actual value from the first in-scope period to today.

**Summary screen**
- FR-047 — Two stacked sections, Revenue above Waterfall, with Period, Activity category and Share controls and a Details button per section.
- FR-048 — The revenue section shows the period label, "Projected vs Goal", the variance with its label, and a trend chart of both series split at today.
- FR-049 — Four league tables (top revenue, performance, cost and ROI by segment), each with its own segment selector, columns Name, metric and Trend, the five highest values, and a Total row over all values.
- FR-050 — Segment options are the pick-list attributes at Plan, Campaign and Activity level marked as report segments; records without a value show as "Not filled in".
- FR-051 — Changing one table's segment refreshes only that table; each table's segment is remembered per user.
- FR-052 — The waterfall section shows a volume block per stage, a conversion card per later stage, and a conversion trend chart for the selected stage, first conversion stage by default.
- FR-053 — Details opens Revenue or Waterfall (with the chart's stage selected), keeping filters, period and category.

**Revenue screen**
- FR-054 — Layout top to bottom: summary cards, trend chart with data table, control bar, ranked list.
- FR-055 — Control bar: Period, Activity category, View by (four levels), Group by (None, Plan, Owner, campaign-level attributes), Sort by (actual revenue, projected revenue, ROI), Compare, Per page (default 10), Order, Share.
- FR-056 — The chart plots actual (to today), projected (from today) and goal revenue; the table rows are actual, projected, goal, performance, actual at monthly FX, actual cost, ROI and actual total, with a total column.
- FR-057 — Each ranked entry shows name, level, owner, goal, actual and projected revenue, performance, actual cost and ROI.
- FR-058 — An information tip explains in Shell words that the Model defines stages, rates and velocity, how a goal projects later stages within the plan year, and what the list compares.

**Waterfall screen**
- FR-059 — A stage selector with one "<stage> to plan" entry per Model stage, default the first; with differing Models every stage appears once, uncovered plans show "--" and their count is stated.
- FR-060 — The same controls as Revenue, except Sort by lists the stage volumes.
- FR-061 — Chart and table series are named from the selected stage (actual, projected, goal), with performance and actual total rows added.
- FR-062 — Each ranked entry shows name, level, owner, the stage's goal, actual, projected and performance, and conversion from the previous stage against its goal.

**Ranked list (shared with 017)**
- FR-063 — Two hierarchy levels at once: the View by level ranked by Sort by and Order, each entry listing its children; no lower layer below Activity.
- FR-064 — Selecting a child makes it the upper layer; a drill trail shows the path; Back goes up one level; drilling never changes the filter panel.
- FR-065 — Every entry offers "Open in Plan", opening the object in the Plan workspace details panel.
- FR-066 — Group by places entries under headings with group totals, groups ordered by their total of the sort metric; campaign attributes apply to descendants, splits as FR-043.
- FR-067 — Compare narrows the list to entries under one chosen plan, campaign, initiative, activity, type, owner or attribute value; default All; the filter panel is untouched.
- FR-068 — Per page pages the upper layer; the list offers Expand all, Collapse all, a name search and highlights the sorted column.
- FR-069 — With Group by None and Compare All, the totals of all upper entries across pages equal the chart totals, except flagged multi-year activities under different parents.

**Summary cards**
- FR-070 — Cards are generated per stage and category: "<stage> plan goal", "<stage> goal coverage by category", "<stage> <category> activity roll-up goal" and "ROI <category>".
- FR-071 — Card sets follow the configured stages and categories without a product change; defaults are the revenue and ROI cards on Revenue and the selected stage's cards on Waterfall.
- FR-072 — A settings action shows, hides and reorders cards (drag and keyboard); the strip scrolls; each card has a small trend; the layout is stored per user per screen and resettable.
- FR-073 — Cards respect the filter panel; category cards show for every category whatever category is selected, so coverage can be compared.

**Empty states and diagnosis**
- FR-074 — An empty widget names its cause: filters exclude everything (with Reset), unapplied changes, no goals, activities not approved, none integrated, a Model without stages, or no performance data loaded.
- FR-075 — Partial data brings a completeness note with the same counts, from the Goal entered and Is integrated flags of 020.

**Sharing and export**
- FR-076 — The share dialog takes active directory people, an optional message and an attachment chip with the report name; Share and Cancel; all validation problems at once.
- FR-077 — A shared report reaches recipients by e-mail with a rendering as displayed, filter summary, currency, model, cost basis, time zone, generation time, sender and a link; the sender learns the outcome.
- FR-078 — Charts export as image, PDF and CSV data; ranked lists as spreadsheet or CSV with every entry of the current drill level; exports carry name, filters, currency, model, basis and time.
- FR-079 — Exports and shared renderings carry exactly the screen's values at the moment of generation.

**Saved views**
- FR-080 — Saved views follow the Plan workspace lifecycle: save with a unique name, load, set default, update with confirmation, delete, reset, share with teams or users who are notified.
- FR-081 — A view captures the whole filter state and every report selection (period, category, View by, Group by, Sort by, Order, Per page, Compare, stage, segments), never data.
- FR-082 — Report views are listed apart from Plan workspace views; sharing shares configuration only and each recipient sees their own permitted data.

**Audit and accessibility**
- FR-083 — Audit events for view create, update, delete and share, for report sharing with its recipients, and for exports (actor, time, report, widget).
- FR-084 — Every control is keyboard operable with visible focus and names; status never by colour alone; the Today marker is labelled; the user's date and number formats apply.

### Success criteria

- SC-001 — At the reference volume, the three reports complete within 5 seconds of Update at the 95th percentile.
- SC-002 — A segment change, drill, page, sort or card layout change completes within 2 seconds (95th); a segment change never touches the other tables.
- SC-003 — The filter panel, including a search over about 1,200 cost centres, responds within 2 seconds (95th).
- SC-004 — SC-001 to SC-003 hold with 200 concurrent Measure users.
- SC-005 — Every table-driven calculation case (goal, actual, projection, performance, variance, conversion, ROI, currency, overlap, time zone, quarters, splits, "--") passes.
- SC-006 — Ranked-list totals equal chart totals, and Summary's figures equal Revenue's and Waterfall's, in every reconciliation test.
- SC-007 — Each of the seven empty causes is named in every test; 90% of UAT participants can explain a deliberately empty report unaided.
- SC-008 — The applied filter state is identical on every standard report and restored in a new session, every time.
- SC-009 — Nine of ten first-time UAT participants finish the scripted first session within 5 minutes.
- SC-010 — 99% of shared e-mails arrive within 5 minutes; a 10,000-entry export finishes within 30 seconds; exports always match the screen.
- SC-011 — Automated and manual WCAG 2.1 AA audits of the three screens find no critical or serious issue.
- SC-012 — Where Shell supplies Hive9 figures, goals, actuals and costs match exactly and projections match within an agreed tolerance.

### Assumptions and open points worth knowing

- Written for a proof of concept: local file store, seeded demo data with no real Shell data, an acting-user selector instead of sign-in; the rules target the real product.
- The projection rule belongs to 020; the default comes from Hive9 help text and Model fields, and a straight-line run-rate forecast is not adopted. Still open with the vendor.
- Charts and ranked lists use activity roll-up goals; plan-level goals appear only on the Plan goal and Goal coverage cards, whose gap shows goals not yet cascaded.
- ROI is a percentage on actual cost at the annual planned rate; a three-band on-track / at-risk / off-track status is not adopted but could be added.
- Fiscal year = calendar year unless the fiscal calendar says otherwise; the time zone comes from the device until a profile preference exists.
- Share recipients are limited to active directory people (Hive9 accepts any address) — a reversible deviation; a shared rendering holds what the sender sees.
- Reference volume for performance tests: 100 plans, 30,000 activities, 150,000 line items, 5 million performance facts a year; 200 of about 2,000 users at once.
- Hive9 behaviours deliberately not reproduced: off-by-one "Not filled in" counters, "0% above" on 0/0, an unlabelled mixed-granularity axis, vague empty messages, a hidden evergreen-only flag.

## Client 017 — Plan Budget and Finance Budget

### User stories

- US1 (P1) — Review marketing spend pacing: Plan Budget's cards and time chart show budget and planned cost running into the future while actual cost stops at today.
- US2 (P1) — Review consumption of a finance budget: choose a budget (its year sets the period) and read the overview cards, including burn rate, and the overview chart.
- US3 (P2) — Rank and drill spend by plan object: rank by allocated, actual, actual at monthly FX or budget, drill down to line items, see over-allocation, open the object in Plan.
- US4 (P2) — Compare a finance budget with its forecast and with another budget: a Forecast series, a compare-to chart and four comparison charts.
- US5 (P2) — Explore a finance budget by level and distribution: one tab per configured level, element selection narrows the screen, distribution charts.
- US6 (P2) — Share and export financial reports: share Plan Budget by e-mail; export any chart, the ranked list or a level table.
- US7 (P2) — Understand an empty or incomplete financial report: the screen names the cause with counts.

### Requirements

**General rules**
- FR-001 — Both screens are read-only; only per-user report preferences are stored.
- FR-002 — Every figure comes from real plan budgets, line items and allocations, finance budgets and actual cost; no demo figures outside seed data.
- FR-003 — Shell vocabulary; all money in the header's display currency, always with its currency.
- FR-004 — Behaviour is for the acting user over every plan and budget; later access rules must not change any metric definition.

**Plan Budget metrics**
- FR-005 — Time-phased plan budget from plans, campaigns and initiatives in the in-scope periods, counted once per branch: a parent's budget when all its children are in scope, otherwise the children's.
- FR-006 — Plan budget: Annual = the entered annual amounts; Plan budget: Total = their time-phased values in the periods; the gap is budget not yet spread.
- FR-007 — Planned cost: Annual = planned cost of in-scope activities, once each; Allocated planned cost: Total = their line-item allocations (including the system balance row) in the periods.
- FR-008 — Actual cost at the annual planned rate; Actual at monthly FX at the month's rate where enabled, otherwise annual with a notice; both in the transaction's period, so they stop at today.
- FR-009 — Unallocated budget = plan budget − allocated cost, a negative value labelled "Over-allocated"; Unspent allocation to date = allocated − actual through today's period.
- FR-010 — When "Use planned cost as actual cost" is on, actual-cost measures follow that rule and every display states the cost basis.
- FR-011 — The Plan Budget series are exactly plan budget, allocated cost, actual cost and actual at monthly FX. Open: whether a forecast series (for example from latest estimates) should be added.

**Finance Budget metrics**
- FR-012 — Scope is the selected finance budget's elements, or the elements chosen on a level tab with their descendants; the period is the budget's fiscal year.
- FR-013 — Annual budget = entered annual amounts counted once per branch; Total budget = time-phased values over the year.
- FR-014 — Budget YTD = time-phased values from the year start through the period containing today (user's time zone); the full year for past years, zero for future ones.
- FR-015 — Total planned cost = allocations within the year of line items linked to in-scope elements; Planned YTD the same through today's period.
- FR-016 — Total actual and Total actual at monthly FX = actual cost of the linked line items dated within the year, converted as FR-008.
- FR-017 — Forecast = time-phased forecast values on in-scope elements; Annual forecast is their sum; both appear only when Forecast is on.
- FR-018 — Burn rate = actual at monthly FX to date ÷ annual budget, "--" on zero, with a warning above 100%. Open: whether finance divides by Budget YTD instead.
- FR-019 — Remaining budget = annual budget − total actual at monthly FX; Unplanned budget = annual budget − total planned cost; negatives carry a written over-budget warning, naming a hard limit where set.
- FR-020 — Budget amounts convert at the annual planned rate of the budget year; line items and actuals as FR-007 and FR-008; rateless amounts are excluded and announced.
- FR-021 — Monthly and quarterly values are sums; burn rate is recomputed from aggregates, never averaged.

**Plan Budget screen**
- FR-022 — Uses 016's frame: the filter panel and shared state, saved views, period, overlap, time zone, today split, multi-year axis, tips and "--"; no Activity category selector.
- FR-023 — Layout: summary cards, money chart with data table, control bar, ranked list.
- FR-024 — The chart plots budget and allocated cost across the axis and the two actual series to today, with a Today marker; the table adds Unallocated budget and a total column; the tip explains the two actual rows.
- FR-025 — Default cards in order: Planned cost: Annual, Allocated planned cost: Total, Plan budget: Annual, Plan budget: Total, Actual cost: Total, Actual at monthly FX: Total; 016's goal and ROI cards are available through the same settings.
- FR-026 — Control bar: Period, View by, Group by (as Revenue), Sort by (allocated, actual at monthly FX, actual, plan budget), Compare, Per page, Order, Share.
- FR-027 — The ranked list follows 016's ranking rules with line items below Activity; columns name, level, owner, budget, allocated, actual, actual at monthly FX, unallocated; activities and line items show "--" for budget.
- FR-028 — Line-item filters narrow the cost measures only, never plan budget; the chart says so while one is active.
- FR-029 — With plan budgets switched off in configuration, the budget series, cards, columns and sort option are hidden and the screen says so.
- FR-030 — Empty or zero-actual causes named: filters exclude all, unapplied changes, planned cost on N activities not allocated to periods, budget not spread, no actual cost (with the latest date recorded), rateless amounts excluded.

**Finance Budget — controls and period**
- FR-031 — No plan filter panel; its panel holds the budget selector, a Forecast option (off by default), the level-tab element selection and Update.
- FR-032 — The budget selector lists finance budgets with name and year, newest first; the last choice is preselected, otherwise the screen asks for one.
- FR-033 — The reporting period is the budget's fiscal year, stated on screen and restated when the budget changes.
- FR-034 — A Monthly or Quarterly period, defaulting to the budget's own default.
- FR-035 — A header shows the budget's name, year, owner and team, and whether its values and forecast are locked and it is a hard limit.
- FR-036 — The last budget, Forecast option, period, comparison choices, chart views and element selection are remembered per user.

**Finance Budget — overview cards and charts**
- FR-037 — Overview cards: Annual budget, Total budget, Budget YTD, Total planned cost, Planned YTD, Total actual, Total actual at monthly FX, Burn rate, plus Annual forecast when on.
- FR-038 — A full-width overview chart of budget, planned, actual and actual at monthly FX by period (plus forecast), with a Today marker, bar, line and table views and export.
- FR-039 — Four comparison charts in a grid: Budget vs planned cost and Budget vs actual cost (bar or table); Actual cost vs budget and Actual cost vs planned cost (line or table).
- FR-040 — A "Compare this budget to" chart with a measure selector and a comparison budget (never the selected one), paired by month of the fiscal year, with bar, line and table views and export.
- FR-041 — Distribution charts showing the share of budget, planned and actual cost by element of the first level (or of the open tab), with values, legend and export.
- FR-042 — Each chart offers only its listed views; the view is remembered per user per chart; the table view is the accessible equivalent.

**Finance Budget — level tabs**
- FR-043 — The tab strip shows Overview plus one tab per level the budget uses, named from configuration; a single-level budget shows Overview only; tabs are never fixed.
- FR-044 — A level tab lists elements with parent path, the overview measures and Remaining budget, sortable with totals; selecting elements and pressing Update narrows cards and charts, with a selection summary and Reset.
- FR-045 — Empty causes named: no budget selected, no elements, no linked line items, no actual cost, year not started, Forecast on without values, rateless amounts, allocations outside the year.

**Sharing, export, audit and accessibility**
- FR-046 — Plan Budget offers Share with 016's dialog and e-mail content; Finance Budget offers export only.
- FR-047 — Every chart exports as image, PDF and CSV with named files; the ranked list and level tables export as spreadsheet or CSV; exports carry name, period, filters or elements, currency, basis and time.
- FR-048 — Shares and exports are audited with actor, time, report or budget and widget.
- FR-049 — Every control (tabs, element selection, chart views) is keyboard operable with names and focus; over-allocation and over-budget are written, never colour alone.

### Success criteria

- SC-001 — Plan Budget completes within 5 seconds of Update at the 95th percentile at the reference volume.
- SC-002 — A budget with 5,000 elements and 100,000 linked line items shows its overview within 5 seconds; a tab, view or Forecast switch within 2 seconds (95th).
- SC-003 — Every table-driven calculation case (budgets, once-per-branch counting, allocations, rates and fallbacks, YTD boundaries, burn rate, remaining and unallocated, quarters, cross-year alignment) passes.
- SC-004 — Finance Budget totals reconcile to the finance budget screens, actuals to attributed transactions, and Plan Budget's allocated cost to the Plan workspace cost view.
- SC-005 — Plan Budget ranked-list totals equal the chart totals in every test.
- SC-006 — Nine of ten finance UAT participants state a budget's spend, pace and remainder within 1 minute.
- SC-007 — Every empty or zero cause in FR-030 and FR-045 is named in every test.
- SC-008 — Exports match the screen every time; 99% of shared Plan Budget e-mails arrive within 5 minutes.
- SC-009 — WCAG 2.1 AA audits of both screens find no critical or serious issue.
- SC-010 — Every money value carries its currency; no amount is dropped for a missing rate without an announcement.
- SC-011 — With at least 8 first-time users, 90% answer how much of a budget is left and was spent last quarter, and export it, unaided in under 2 minutes.

### Assumptions and open points worth knowing

- Proof-of-concept framing as in 016; Plan Budget reuses 016's layout, ranked list and cards, so 016 is a build prerequisite.
- "Annual" cards are entered amounts and "Total" cards their time-phased values, matching a Hive9 observation of an annual budget with a zero spread total.
- Budgets count once per branch so flowed-down budgets are never doubled; over-allocation is allowed and highlighted.
- YTD includes the whole period containing today; actuals, being dated postings, never pass today.
- Level tabs, element selection and distribution charts are interpreted from the analysis — the sandbox showed only an Overview tab — and are confirmed in UAT.
- Finance Budget has no Share and no saved views in Hive9; remembering the last selection is a convenience addition.
- The two "Actual cost vs …" line charts show per-period values, not cumulative ones.

## Client 018 — Approvals, PR Dashboard, AR Invoice Details

### User stories

- US1 (P1) — See approval workload and approval cycle time: counts per approval status for one object type and the mean days to sign off, overall and per approver.
- US2 (P1) — Track forward commitment to vendors: PR Dashboard cards, committed value per vendor and per owner by delivery month, and a grid by vendor.
- US3 (P1) — Reconcile accruals down to purchase orders and invoices: each accrual line shows planned, actual, difference and recommendation and expands to its POs and invoices.
- US4 (P2) — Find, inspect and export individual approval records: a grid with its own object type, search, sort, full comments and export.
- US5 (P2) — Judge vendor delivery reliability: per vendor, the share of due lines received on time, or an explicit "receipt data required" state.
- US6 (P3) — Tailor the AR Invoice Details grid and reuse views: column management, saved views, sharing with a team.

### Requirements

**Common to all three dashboards**
- FR-001 — Measure navigation offers Approvals Dashboard, PR Dashboard and AR Invoice Details as three separate screens.
- FR-002 — Read-only for business data; only personal preferences (filters, views, cards, columns) are saved, each change recorded with who and when.
- FR-003 — Filter and date changes apply only on Update.
- FR-004 — Each dashboard restores the user's last filters, always shows a filters-applied summary and offers a one-click reset.
- FR-005 — Every amount in the display currency with its currency, converted by the rate rule (monthly where enabled, else annual); switching currency re-expresses the page.
- FR-006 — Absent values show "--", never zero.
- FR-007 — Empty widgets name the cause: no matching records, feature not enabled or configured, or source data not received.
- FR-008 — Every chart has a table view; every control is keyboard and screen-reader operable.
- FR-009 — Wherever figures come from GSAP, the page shows when GSAP data was last received.
- FR-010 — Each grid and each chart's data export to CSV or spreadsheet: every matching row, visible columns in order, display currency named, no helper columns.
- FR-011 — Hive9's access rules here (Measure enablement, per-role report rights, own-team rows and amounts) are recorded for later and not enforced; the acting user sees everything.

**Approvals Dashboard**
- FR-012 — Three widgets: Current approval status (upper left), Average (upper right) and the Approval activities grid (full width below).
- FR-013 — Uses 016's standard filter panel and shares its persisted state; no Monthly or Quarterly selector.
- FR-014 — Extra filters: Approver, Vendor, purchase request and PR line sections (owners, statuses, PR attribute groups), accrual request and line sections, and an approval status filter.
- FR-015 — The approval status filter offers each object type's own status vocabulary (including In review and the accrual statuses), not one reduced list.
- FR-016 — Year limits the dashboard to approvals of objects in the selected fiscal years (the plan's year, or the budget's year for accruals).
- FR-017 — The status widget and the grid each have an independent object-type selector listing only types with an enabled workflow; default Activity.
- FR-018 — Current approval status shows, for the selected type, the count per current status and each status's share.
- FR-019 — Average shows mean calendar days from submission to sign-off over completed steps (approved or declined), to one decimal, with the count and a per-approver list, slowest first; pending, cancelled and not-required steps are excluded.
- FR-020 — The grid shows one row per approver step per submission, with the columns of FR-024.
- FR-021 — The grid offers free-text search over text columns, sort by any column, page sizes 5 to 100 (default 50) and the total count.
- FR-022 — Long comments are readable in full without leaving the dashboard.
- FR-023 — No approving, declining or recalling here; that happens in the approvals queue.
- FR-024 — Grid columns in default order: item ID, item name, approver, sign-off date, sign-off status, submitted date, level, PO number, amount, owner, comments, attribute group name.

**PR Dashboard**
- FR-025 — Filters: saved views with PR presets, date range (default current fiscal year), vendors, PR and PR-line attribute tabs, PR owner and line owner, PR status (six values), Update; no Year or period selector.
- FR-026 — Each PR attribute group includes "Not filled in", ticked by default, so unclassified PRs are included.
- FR-027 — The date range selects PR lines by delivery date; a PR is in scope when any of its lines is.
- FR-028 — Six summary cards: PR count, PR line count, distinct vendors, GRSP amount received, invoice amount, and PR line amount committed.
- FR-029 — The three count cards offer a Distribution view: PRs and lines by status, vendors by PR count and line value.
- FR-030 — Users choose which cards show and in what order, remembered per user.
- FR-031 — Chart "PR total by vendor by period": committed line value per vendor by calendar month of delivery date.
- FR-032 — Chart "PR total by owner by period": the same value grouped by the PR's owner.
- FR-033 — A chart with more than ten vendors or owners shows the ten largest and groups the rest as "Other"; table view and export list all.
- FR-034 — Chart "Vendor performance percentage": per vendor, the share of lines due within the range and on or before today that were received by their delivery date.
- FR-035 — Without any receipt data in scope, that chart states that receipt data is required and not yet received, instead of a percentage.
- FR-036 — Below the charts, the read-only PR overview grid (vendor › PR › line › line item, expandable) limited to the filters, with row detail and the FR-037 columns.
- FR-037 — PR grid columns: name, actions, ID, level, status, PR cost, owner, created date, delivery date, sent date, PO number, PO line number, PO line status, invoice amount, GRSP amount, approval workflow.

**AR Invoice Details**
- FR-038 — Filters: date range with a "Filters applied" indicator and Clear (default current fiscal year), plan, budget name and budget item, accrual request tab (statuses, owner, attributes), accrual line tab, saved views, Update; no Year, period or Share.
- FR-039 — Request status and line status are two distinct lists.
- FR-040 — The date range selects accrual requests by accrual month.
- FR-041 — The grid nests accrual request › line › PO › invoice; POs come from GSAP against the line's line item, invoices against each PO; expand only where rows exist.
- FR-042 — Planned and actual amount in period = the line item's allocated and actual cost in the accrual period; their difference is highlighted when negative.
- FR-043 — Recommended accrual shows the proposed accrual per line (and PO or invoice where meaningful). Open: the formula, and how accrual method, cut-off day and threshold change it.
- FR-044 — Default columns are the standard attributes plus Recommended accrual, so variance and recommendation show without configuration.
- FR-045 — Column management shows, hides, reorders and resizes columns across five sections, remembers the layout per user and offers a reset; custom-attribute sections follow configuration.
- FR-046 — Each line row offers Line item cost allocation, a read-only view of allocated planned cost by period, closed with one action.
- FR-047 — The Approval workflow column names the governing workflow when accrual workflows are enabled, otherwise it is empty.
- FR-048 — Saved views support save, load, default, update, delete, reset and share with teams and users, as in 016; sharing never widens data access.
- FR-049 — Standard columns in default order: name, accrual type, actions, ID, level, status, owner, AR date, line amount, line item ID, budget ID, planned, actual and difference in period, workflow, accrual month, PO count.
- FR-050 — PO attributes: PO number, PO amount, invoice count.
- FR-051 — Invoice attributes: invoice number, date, amount, GR date, SE date, accruals already recorded, recommended accrual.

### Success criteria

- SC-001 — Each dashboard shows its complete first and refreshed view within 5 seconds for 95% of requests at production scale.
- SC-002 — Grid interactions (paging, search, sort, expand, cost allocation view) complete within 2 seconds for 95% of requests.
- SC-003 — SC-001 and SC-002 hold with 200 concurrent Measure users.
- SC-004 — An export of up to 50,000 rows is ready within 60 seconds; exported values always equal the screen.
- SC-005 — On a reference dataset, every status count, card total, chart value, average (to 0.1 day), vendor percentage and difference (to the cent) matches expected values.
- SC-006 — 90% of governance users find the longest-waiting pending approval of a type within 2 minutes at the first attempt.
- SC-007 — A finance user reaches the invoices under an accrual line in at most 3 interactions and reconciles one request in under 5 minutes.
- SC-008 — Every empty-state case shows its specific cause rather than a generic message.
- SC-009 — All three dashboards pass WCAG 2.1 AA with no critical or serious finding, and every chart has a table view.
- SC-010 — With at least 8 first-time users, 90% find a named approver's pending items, next month's lines for a vendor and a request's accrual lines unaided, each in under 2 minutes.

### Assumptions and open points worth knowing

- Proof-of-concept framing as in 016; production scale is up to 50,000 sign-off steps, 20,000 PR lines across about 150 vendors and 5,000 accrual lines a year, five years kept.
- Cycle time is calendar days from submission to sign-off per step; approved and declined both count; the Average widget follows the status widget's object type.
- Hive9's reduced approval status list is a known inconsistency not reproduced; each object's own statuses are offered.
- Object types are whatever workflows are enabled, which also covers modified PRs.
- Date ranges default to the current fiscal year; the accrual range uses accrual month because Hive9's basis was not observable; PR charts bucket by calendar month.
- The vendor performance definition is a default: Hive9's formula was not visible, and a service-entry date may understate on-time delivery because it follows invoice approval.
- Exports of the accrual grid and of PR chart data go beyond Hive9 and can be dropped without affecting anything else.

## Client 019 — Plan Details and tenant custom dashboards

### User stories

- US1 (P1) — Open and work with a custom dashboard: landing page, breadcrumb, scope banner, a date range defaulting to a rolling six months, filter groups, View by, Update, export and View data.
- US2 (P1) — Audit plan data completeness and balance with Plan Details: distribution by type, start month and status, three completeness charts, and attribute profile pages for Activity and Initiative.
- US3 (P2) — Monitor the GSAP link: SWBS Not Received, Pending PR and PR Track Invoices list waiting items with how long they have waited.
- US4 (P2) — Run the China budget control pack: release, occupancy, PO commitment and receipt per initiative, with Activity Summary and PO Report.
- US5 (P2) — Control actuals and accruals: Actuals at monthly FX, Transactions Report, Overlapping Actuals and Accrual Finance Report.
- US6 (P2) — Review SP&A spend for Lubes and Mobility: planned, committed, actual, variance and utilisation by quarter and channel, equal to the plan's SP&A views.
- US7 (P3) — Audit user access: User Detail Report and User Details list users with role, teams and last sign-in, exportable.
- US8 (P3) — Personalise: pin a dashboard as home page, save filter presets, switch on comparison display.

### Requirements

**Custom dashboard engine**
- FR-001 — A custom dashboard is one or more pages, one of them the landing page.
- FR-002 — The Measure menu lists each available dashboard after the standard reports with its pages as sub-items; opening one shows its landing page.
- FR-003 — Every page shows a "Dashboard/Page" breadcrumb and a collapsible banner naming the records in scope.
- FR-004 — Dashboards, pages, widgets, filter groups, defaults and comparison allowance are configuration data; an end-user authoring screen is out of scope (025).
- FR-005 — A date-range picker with From and To, presets (This quarter, This year, Year to date, Last 12 months, Last 4 quarters, Last calendar year, Custom), Apply and Cancel; default a rolling six months; always visible; From after To rejected.
- FR-006 — Each widget declares which date decides range membership; records with a span count when they overlap it.
- FR-007 — Each dashboard has its own filter groups, independent of 016's state, each with collapse, a counter, Select all and Clear all, search, a checkbox list, "View all" and a "Not filled in" value.
- FR-008 — Changes apply on Update; Reset restores defaults; the filter menu offers Save current filters, Load saved filters and Clear all.
- FR-009 — Presets are personal, named, per dashboard and include the date range; the last-applied filters and range are restored per user, with the summary always visible.
- FR-010 — View by offers Years, Quarters, Months and Weeks and regroups every time-based widget; the default comes from the definition.
- FR-011 — Widget types: data highlight (a key figure with optional goal and comparison), bar (horizontal or vertical, optional second axis), pie, line and table.
- FR-012 — Each widget definition sets title, help text, value format, rows or slices before "Other" (5 for bars, 20 for pies), legend and labels, an optional totals row and highlight rules.
- FR-013 — Table widgets offer page sizes from 5 to 500 (default 20) and sort by any column.
- FR-014 — Tables export as CSV and spreadsheet; charts export the same plus an image; exports hold every matching row, the currency named, no helper columns.
- FR-015 — Every chart offers View data: a sortable, exportable table of the records behind it or behind a selected bar or slice.
- FR-016 — A user can pin one dashboard page as home page, and unpin it.
- FR-017 — Where allowed, comparison display shows highlights and time charts against the preceding period of equal length, with the difference.
- FR-018 — Every amount follows the display-currency switch and the rate rule, always with its currency.
- FR-019 — Each page shows when its data was last refreshed and a notice while a refresh runs.
- FR-020 — An empty page or widget names the cause: the range excludes all data (offering wider presets), no matching records, no widgets configured, or source data not received.
- FR-021 — Every chart has a table view; every control is keyboard and screen-reader operable.
- FR-022 — Read-only for business data; presets, pins and comparison choices are personal preferences recorded with who and when.
- FR-023 — Each dashboard carries separate View and Export rights, recorded for later and not enforced here; the acting user sees and exports everything.

**Dashboard catalogue**
- FR-024 — Eleven Shell dashboards run on the engine: Plan Details (five pages), China Budget Reports (three pages), Actuals at monthly FX, PR Track Invoices, Transactions Report, SWBS Not Received, Pending PR, Accrual Finance Report, Overlapping Actuals, SP&A Lubes and SP&A Mobility. Open: which are used in production, and the exact content of the nine not observed.
- FR-025 — Each catalogue dashboard can be switched off in configuration without affecting the others.

**Plan Details**
- FR-026 — Landing widgets: Activities by type (bar), Planned cost by type (bar), Activities by start date (line), Activities by status (pie), and Planned cost entered, Goal entered and Is integrated (pies).
- FR-027 — Planned cost entered = planned cost above zero; Goal entered = any stage or custom goal set; Is integrated = linked to an integrated performance source as 020 defines it.
- FR-028 — A self-configuring attribute profile per level: per custom attribute a count widget, a planned-cost widget and a completeness chart, following configuration changes without dashboard changes.
- FR-029 — Multi-select attributes: a record counts once under each value; planned cost is divided by the cost split so totals reconcile.
- FR-030 — Attributes are identified by level and identity, never by display name.
- FR-031 — The Custom attributes – Activity page profiles every Activity-level attribute as pie charts.
- FR-032 — The Custom attributes – Initiative page profiles Initiative-level attributes; its filters include Initiative status with six values.
- FR-033 — Filter groups: plan, business unit, campaign, activity type, activity status and activity owner on the landing and Activity pages; plan, business unit, initiative status and initiative owner on the Initiative page; overlap on start–end dates.
- FR-034 — The User Detail Report table: first name, last name, e-mail, job title, last sign-in, role and teams (comma-separated), filterable by team and status; role and sign-in show "--" until access rules exist.
- FR-035 — The User Details page: users by team, by role and by last sign-in band (30 days, 90 days, over 90, never), stating that role and sign-in figures are unavailable until then.

**China Budget Reports**
- FR-036 — A budget name filter group and a date range defaulting to the current fiscal year.
- FR-037 — The main table has one row per budget, team and initiative with budget, team, initiative, FY budget, YTD released amount and %, activity occupied amount and rate, PO occupied amount and rate, GR/SE amount and actuals.
- FR-038 — The Activity Summary table has one row per line item with year, budget, team, initiative, activity, line item, owner, SWBS code, budget and release figures, occupancy, PO occupied, remained amount and GR/SE amount.
- FR-039 — The PO Report table has one row per PR line and PO line with plan, activity, line item, PR, PO, SWBS, cost centre, vendor and outline agreement details, GR amount, to be delivered, earliest accrual, most recent amount and requested action.
- FR-040 — Derived columns: YTD released % = released ÷ FY budget; occupied rates = occupied ÷ released; remained = released − activity occupied; to be delivered = PR/PO line amount − GR; "--" on zero, highlighted above 100%.

**Further catalogue dashboards**
- FR-041 — Actuals at monthly FX: highlights for actual at the planned rate, at the monthly rate and the difference; a chart of both; a table by hierarchy and line item with local-currency figures; filters plan, business unit, currency, company code, cost centre, GL code.
- FR-042 — PR Track Invoices: each returned invoice with PR, key, vendor, PO, invoice details, marketer approval status, service entry date and working days waiting, highlighted beyond ten working days; highlights for awaiting approval and approved without service entry.
- FR-043 — Transactions Report: transactions with the actuals feature's columns, highlights for total, attributed and remaining, a chart by period split linked, partly linked and unlinked; filters link status, cost centre, vendor, document type, plan.
- FR-044 — SWBS Not Received: line items flagged for an SWBS code on approved activities still without one after the turnaround (default 48 hours, configurable), with hierarchy, approval date, owner, codes, request status, rejection reason and hours waiting; a count highlight and a waiting-band chart.
- FR-045 — Pending PR: two groups — awaiting approval, and approved without a GSAP PR or PO after the expected time (default two working days per step) — with identifiers, vendor, owner, status, dates, numbers, amount and days pending; highlights per group; filters owner, vendor, status, type, e-mail group, plan.
- FR-046 — Accrual Finance Report: accrual lines by month, budget, row, cost centre, GL, company code, business and country with amount, statuses, line item, owner and approval date; highlights per status; a chart by month and status; filters status, business, country, budget.
- FR-047 — Overlapping Actuals: actuals the overlap rule flags as counted more than once, with document, amount, linked line items and activities, attributed total and overlapping amount; a count and amount highlight.
- FR-048 — SP&A Lubes and SP&A Mobility: for their business unit only, planned, committed, actual, variance and utilisation by quarter and channel as highlights, charts and a table, using exactly the plan's SP&A definitions; filters plan, cluster, region, channel, brand, GL code, spend type.

### Success criteria

- SC-001 — A page with up to 12 widgets shows all of them within 5 seconds for 95% of opens and Updates at production scale.
- SC-002 — Paging, sorting, View by changes and View data complete within 2 seconds for 95% of requests.
- SC-003 — SC-001 and SC-002 hold with 200 concurrent Measure users.
- SC-004 — A widget export of up to 50,000 rows is ready within 60 seconds; values always equal the screen.
- SC-005 — Adding an Activity- or Initiative-level attribute makes its widgets appear on the next load with no dashboard change.
- SC-006 — Every completeness count, distribution (with splits), China rate and catalogue total matches independent values; SWBS Not Received and Pending PR list exactly the qualifying records.
- SC-007 — At the start of UAT, every dashboard Shell confirms for production is in the menu with all its pages.
- SC-008 — 90% of marketers identify their Activities lacking a goal or a required attribute within 3 minutes at the first attempt.
- SC-009 — Every empty-state case names its cause; the range and filters are visible on every page without scrolling.
- SC-010 — Every dashboard passes WCAG 2.1 AA with no critical or serious finding, and every chart has a table view.
- SC-011 — With at least 8 first-time users, 90% open Plan Details for a plan, apply two filters and export unaided in under 2 minutes.

### Assumptions and open points worth knowing

- Proof-of-concept framing as in 016; production scale is about 5,000 activities, 15,000 line items, 20,000 PR lines, 100,000 transactions, 5,000 accrual lines and 2,000 users a year, five years kept.
- The rolling six-month default is kept for parity and overridable per dashboard (China Budget Reports uses the fiscal year); weeks start on Monday.
- Only Plan Details and China Budget Reports were observed; the nine others are inferred from their names and Shell's process, to be confirmed or replaced.
- Plan Details filter groups and the User Details page are defaults: Hive9 showed the groups without naming them and the page was undocumented.
- Profiling Initiative-level attributes and adding a completeness chart to every attribute row go beyond what the sandbox showed.
- Chart data export to CSV and spreadsheet goes beyond Hive9's image-only export.
- Turnaround defaults come from the GSAP workshop: 48 hours for SWBS codes, two working days per PR step, ten working days for invoices; GSAP status and corrective actions belong to the GSAP feature, which these dashboards only read.
- Nothing shows personal data beyond user names, except the User Detail Report, which lists e-mail addresses because it is an access audit.

## Client 020 — Performance data, media codes, integrations, attribution

### User stories

- US1 (P1) — Set up a data source and its column mappings: name, category, Models and cadence; each mapping says what one file column means; full find, view, edit and delete.
- US2 (P1) — Upload performance data and match it to Activities: rows are validated, matched on one of four keys and summarised as matched, held, unattributed, rejected or unmatched; uploads can be undone.
- US3 (P1) — See performance on an approved Activity: per stage goal, actual, projected and conversion; revenue with its model's name; the facts list; flags and freshness; a gate until approval.
- US4 (P1) — External names and integration IDs: generated on approval from the Model's rule, settable by hand; one integration ID per source; both are matching keys.
- US5 (P2) — Review and fix rows that could not be loaded or matched: search, correct and re-check, dismiss with a comment, reopen.
- US6 (P2) — Media codes for campaign links: generated per Activity where the type requires them; copy the code or a tagged link; edit, deactivate or delete.
- US7 (P2) — Projected stage volumes and revenue: from the goal stage forward with the Model's rates, velocity and deal size, within the plan year.
- US8 (P2) — Credit revenue to Activities with an attribution model: the designated model credits journeys, every figure names it, and models can be compared without changing it.

### Requirements

**Behaviour of every screen**
- FR-001 — Every screen follows the product principles: plain labels, a create action on each creatable list, search and filters, a details view, all validation errors at once, unsaved-change confirmation, confirmations stating what changes, reasons for unavailable actions.

**Data sources**
- FR-002 — Data sources (name, description, owner, category, status, delivery route, cadence, Models) can be created, found, viewed, edited and deleted only while they hold no facts; otherwise Deactivate is offered.
- FR-003 — Within a Model each stage and revenue is supplied by at most one active source; a conflicting mapping or assignment is refused, naming the source.
- FR-004 — Deactivating keeps facts, refuses uploads and updates Is integrated at once; a source can be reactivated.
- FR-005 — Each source shows freshness: last successful upload, "data through" date, cadence, and a status of Current, Stale, Failed or Inactive.

**Column mappings**
- FR-006 — Mappings (one column meaning a matching key, source reference, fact date, stage, revenue with currency, web metric or journey element) can be created, found, viewed, edited with validation and deleted; they affect future uploads only.
- FR-007 — A blank file template can be downloaded from a source's mappings.

**Uploading performance data**
- FR-008 — Upload a CSV or spreadsheet file for a chosen source; any future automated feed passes through the same rules.
- FR-009 — Every row is validated with all its problems reported at once: empty required column, invalid or future date, negative or fractional count, invalid amount or unconfigured currency, unknown stage, missing or repeated reference.
- FR-010 — A file holding personal-data columns is refused in full; journeys carry only a pseudonymous reference.
- FR-011 — Each fact matches exactly one Activity through activity ID, media code, integration ID or external name, in that order; disagreeing, ambiguous or unknown keys leave it unmatched with a stated reason.
- FR-012 — Facts for an Activity not counting as approved are held, excluded from every figure and released automatically on approval.
- FR-013 — Facts for a binned Activity are held until restore and approval; facts for a permanently deleted one go to Rows to review.
- FR-014 — Each upload yields a summary of rows read and rows matched, held, unattributed, rejected and unmatched by stage or metric; the five add up; rejected and unmatched rows go to review.
- FR-015 — Uploads are idempotent: the same source and reference replaces the earlier fact; an identical file changes nothing; replacements are audited.
- FR-016 — Uploads can be found (file, source, uploader, status, date) and viewed with their summary; there is no edit, and the reason is explained.
- FR-017 — Administrators can undo a whole upload after a confirmation stating the counts and the recalculation; replaced versions are restored; the upload stays listed as Undone.
- FR-018 — Revenue facts are stored exactly in their currency; without a rate they are marked, excluded from converted totals with a count, and included once a rate exists.
- FR-019 — A fact counts in the fiscal year of its fact date, also for a multi-year Activity.

**Rows to review**
- FR-020 — Rows to review hold every rejected or unmatched row: no create; find by key, source, reason, upload, date and status; view with history; Correct; Dismiss with a required comment; reopen.
- FR-021 — Open rows and held facts can be re-checked on demand and are re-checked automatically at each upload from the same source; a match makes the row Resolved.
- FR-022 — A later upload with the same source and reference replaces a corrected row; the source wins.

**Performance on an Activity**
- FR-023 — Performance information lives in the Activity details panel's Performance and Data sources areas.
- FR-024 — An Activity not counting as approved shows "Approve the activity to see performance data" and no figures.
- FR-025 — An Activity counts as approved only while Approved, In progress or Complete, not while Created, Submitted or Declined, including after re-approval or restore.
- FR-026 — For an approved Activity: per stage goal, actual to date, projected for the plan year and conversion; revenue goal, attributed actual (model named) and projected; web metrics by month; freshness per source.
- FR-027 — Its facts can be listed (date, source, stage or metric, value, currency, key used, upload), searched and exported.
- FR-028 — The Data sources area shows external name and aliases, media-code count, the two flags, and per source the integration ID, fact count and last fact date; a notice when the Model has no source.
- FR-029 — Every empty performance view states its reason (not approved, no source, no goal, no data yet, stale or failed) and never shows sample figures.

**External names and integration IDs**
- FR-030 — The Model's external-name rule applies; without one, "{Activity name}_{Activity ID}".
- FR-031 — The external name is generated when the Activity first reaches Approved; before that it reads "Generated on approval".
- FR-032 — With "Regenerate on edit" on, changes to the fields the rule uses regenerate the name; off, it stays; previous names remain as matching aliases.
- FR-033 — A name set by hand is not regenerated until the user switches back to the generated one.
- FR-034 — Names and aliases are unique and within the rule's length: a generated clash gets the Activity ID appended, an overlong name is shortened; a hand-set clash or overlong name is refused with the reason.
- FR-035 — Integration IDs: at most one per source per Activity, unique within the source, removable after a confirmation about matching, every change audited.
- FR-036 — Copying an Activity copies no external name, aliases, integration IDs, media codes or facts.

**Media codes**
- FR-037 — Whether a type requires media codes, and which code attributes exist and are required, come from configuration.
- FR-038 — The Media codes area appears only on Activities whose type requires them; codes never attach to line items or any other level.
- FR-039 — Codes can be generated at any status (all required attributes, errors at once), found, viewed, edited (attributes and landing page only), and deleted only while unused, otherwise deactivated.
- FR-040 — Codes are system-generated numbers, unique and never reused; the starting number is configurable so new codes avoid Hive9's.
- FR-041 — One-action copy of the code, and of a tagged link appending "mediacode=<code>" to the landing page, each confirmed.
- FR-042 — A deactivated code keeps matching but is not offered for copying; it can be reactivated.
- FR-043 — When a type stops requiring codes, existing codes are kept, keep matching and show read-only with the reason.
- FR-044 — A code-requiring Activity without a code shows a non-blocking notice that web data cannot be matched.

**Is integrated and Goal entered**
- FR-045 — Is integrated = Yes when the Activity counts as approved and its Model has at least one active source.
- FR-046 — Goal entered = Yes when any stage or custom goal on the Activity is above zero; the same definition Plan Details uses.
- FR-047 — Both flags are always current and offered wherever Activity fields are: grid columns, filters, exports and completeness charts.

**Projections (the same default as 016)**
- FR-048 — Later-stage goals = the previous goal × the conversion rate; revenue goal = final-stage goal × average deal size.
- FR-049 — Goals are time-phased evenly across the Activity's months and the days within them; an evergreen goal runs to the plan year end.
- FR-050 — Projection runs forward from the goal stage only; after today the goal stage follows its phased goal, later stages follow prior actual or projected volume × rate, shifted by velocity; revenue = final volume × deal size.
- FR-051 — The plan-year figure = actual to today + projected after; nothing projects past the plan year.
- FR-052 — Projections recalculate when goals, dates, Model parameters or facts change; every projected figure states its Model and the date its rates last changed.
- FR-053 — Projections are available to Measure by day, month and quarter.

**Attribution**
- FR-054 — Each conversion's revenue, and the conversion itself, are credited to the Activities of its journey's touches; weights sum to 100%; only touches on or before the conversion count.
- FR-055 — A revenue fact without a journey reference is a single-touch journey credited 100% to its Activity under every model.
- FR-056 — Eleven attribution models with exact weight rules: first touch, last touch, last non-direct, equally distributed, interaction weighted, linear time decay, half-life time decay, U-shaped, W-shaped, full-path; custom statistically inferred is a later capability.
- FR-057 — Unmatched touches report as "Unattributed"; credited plus unattributed equals converted revenue to the minor unit, the residual going to the largest weight, ties to the newest touch.
- FR-058 — Administrators view and edit the designated model (seed: equally distributed), the half-life and touch-type weights; every actual revenue and ROI figure uses the designated model and names it.
- FR-059 — Changing the model or its parameters is audited and restates every attributed figure, after a confirmation saying so.
- FR-060 — An Attribution view lets users pick any model, see credited revenue and conversions by Activity (groupable), the unattributed amount, a cross-model comparison and freshness, without changing the designated model.
- FR-061 — Grouping the view by a multi-select attribute splits credit by each Activity's performance split.

**Audit, events, permissions and later capabilities**
- FR-062 — Audit records for every change to sources, mappings, uploads (completed, failed, undone), fact replacements, review rows, external names, integration IDs, media codes and attribution settings.
- FR-063 — Events for notifications: an upload failed, a source became stale, new performance arrived for an owned Activity.
- FR-064 — Managing sources, mappings, uploads, review rows and attribution settings follows an integrations right; external names, IDs and codes follow the Activity edit right; enforced later.
- FR-065 — The first release never writes to an external system; pushing Activities to marketing-automation platforms (folders, push mappings, a sync action, name updates) is a later capability pending Shell's confirmation.

### Success criteria

- SC-001 — The upload category counts add up to the rows read in every upload; no row lacks a category.
- SC-002 — Every fact whose keys identify one approved Activity matches it; every conflicting or unknown fact reaches review with a reason.
- SC-003 — Uploading an identical file twice changes no figure in any run.
- SC-004 — A 50,000-row file uploads and shows its summary within 5 minutes.
- SC-005 — For each of the ten available models, credited plus unattributed equals converted revenue to the cent on every test journey.
- SC-006 — Projections match a hand-calculated table and 016's rules in every table-driven case.
- SC-007 — Every approved Activity under a rule has an external name immediately; no two Activities share a name or alias.
- SC-008 — A marketer generates a media code and pastes its link in under 30 seconds with at most three actions.
- SC-009 — Every empty performance view shows a reason; sample figures appear in none.
- SC-010 — Both flags agree with their definitions for every Activity in a test plan of at least 30 Activities.
- SC-011 — Every revenue, ROI or projected figure names its model or Model and its "data through" date so a reviewer can reproduce it.
- SC-012 — With five operations users and five marketers, 90% complete the source, upload, review-row and media-code tasks unaided in under 2 minutes each; usability score at least 75.
- SC-013 — Every form shows all errors at once and guards unsaved changes; every confirmation names what is removed; every unavailable action states why.

### Assumptions and open points worth knowing

- Proof-of-concept framing as in 016; performance data arrives by file upload only, connectors are deferred.
- Which production systems feed Hive9, which stages they supply and at what cadence is open; the sandbox shows no integrated Activity, so Shell may use little performance data today.
- The projection rule is a default carried once, in 016 FR-036; when answered it changes in both specs together.
- The attribution catalogue is the union of Hive9's nine models and two reference-only rules that may be dropped; the custom inferred model is a later capability.
- Multi-touch attribution applies to revenue and closed-won conversions; stage volumes count for the Activity they matched.
- The approval gate is the approvals feature's approved state, so performance disappears while an Activity is re-approved.
- External-name fallback, clash and shortening rules are defaults; names and codes already issued in Hive9 arrive by data migration, not by this feature.
- Media codes are numeric after Hive9's only example; realtime analytics, traffic generators and demo-data fallbacks are not reproduced.

## Client 025 — Self-service dashboard builder (Proposed, not planned for MAPP)

### User stories

- US1 (P1) — Build and save a dashboard from Shell data: pick widget types, a governed source, dimensions and measures with a live preview, widths and order, and a unique name.
- US2 (P1) — View and share dashboards read-only: a viewer separate from the builder, pinned widgets first, hidden ones excluded, per-session filters, save a copy.
- US3 (P2) — Show targets, reference lines and display options: a target with a direction, reference lines, aggregation, sort, top N, colours, pin and hide.
- US4 (P2) — Capture snapshots and export widget data: stored images with label and time; View data and export per widget.
- US5 (P3) — Upload my own data, join it and chart it: CSV or spreadsheet datasets with inferred types, joins on a key, suggested charts, extended chart types.
- US6 (P3) — Start from a starter layout or from the assistant: focus and layout presets; an assistant-proposed dashboard is accepted only when every widget is valid.

### Requirements

- FR-001–FR-006 — dashboards and layout: create, rename, copy and delete with unique names and suggested alternatives; a three-column grid with S, M and L widths; reorder, pin and hide; an unsaved-change warning; delete to the recycle bin with restore naming; concurrent saves detected.
- FR-007–FR-014 — widgets and binding: eight core and nine extended widget types; typed slots offering only fitting columns; valid before save with a live preview; custom axes, aggregation, sort, top N and stacking; generated titles never overwrite typed ones; an accessible palette with distinct series; View data, export and image per chart; gauges, funnels, reference and target lines wherever supported.
- FR-015–FR-020 — Shell data sources: governed sources (plan activities, plan budget, finance budget, purchase and accrual requests, approvals, performance and funnel) with time, hierarchy and attribute dimensions and Measure-consistent measures; split percentages; the shared filter panel with per-viewer session filters; display currency; viewer-scoped visibility; no demo data.
- FR-021–FR-024 — targets and reference lines: a value with a direction on single-measure widgets; labelled lines, zones and on- or off-target marks not by colour alone; suggested values and K, M and B suffixes; a reference line at average, median or a custom value.
- FR-025–FR-027 — viewer and sharing: a read-only viewer of owned and shared dashboards with Edit for the owner and Save a copy for all; view-only sharing with people or teams, prompting to share datasets too; distinct empty states.
- FR-028–FR-029 — snapshots: stored images with label, time and capturer under a configurable limit; listed newest first, visible only to the capturer and the owner.
- FR-030–FR-038 — dataset upload and joins: accepted file types and size limits (open: which data classifications may be uploaded, and for how long); phased progress and sheet choice; header and type inference rules; overridable types with currency and scale; specific upload errors; replace while bound; delete to the recycle bin listing the dashboards affected; inner or left joins on a key with a row cap; datasets private unless shared.
- FR-039–FR-041 — suggestions, starters and the assistant: up to six suggested charts with reasons; starter layouts per focus; assistant definitions accepted only when valid.
- FR-042–FR-043 — general: audit of every dashboard, snapshot and dataset action; WCAG 2.1 AA with keyboard alternatives and table views.

### Success criteria

- SC-001 — 80% of untrained business users build and save a four-widget dashboard in under 10 minutes.
- SC-002 — Widget figures match the standard Measure screen for the same scope in every one of at least 30 comparisons.
- SC-003 — Dashboards with up to 12 widgets display within 5 seconds in 95% of loads; the preview updates within 2 seconds.
- SC-004 — A 50 MB, 500,000-row file is ready to chart within 2 minutes, with progress at least every 5 seconds.
- SC-005 — No unsaved work is lost in usability tests; every deleted dashboard and dataset can be restored.
- SC-006 — All 17 widget types render with a table alternative and pass WCAG 2.1 AA.
- SC-007 — In a pilot, at least half of new dashboard requests are met by self-service rather than a change request.

### Assumptions and open points worth knowing

- Proposed beyond Hive9 parity, modelled on a reference builder; it needs Shell's approval before planning. In Hive9 only the vendor can create or change custom dashboards.
- Shell's tenant dashboards stay with 019; self-built dashboards are separate and read the same reporting data as Measure, with no separate BI tool.
- Until clarified, uploads are limited to internal business data without personal data and kept until deleted; uploaded data never changes plan, finance or performance records.
- Limits (file size, rows, caps, snapshot count) are configurable defaults from the reference, tuned in a pilot.

## Terms the client specs use

| Client-spec term | MAPP word |
|---|---|
| Tactic | Activity |
| Program | Initiative |
| HUD, heads-up display | Summary cards |
| (Not Completed) | Not filled in |
| Inspection Window | Details |
| Show Hierarchies | Show levels |
| Tactic Category | Activity category |
| Tactic Type | Activity type |
| Waterfall | Funnel stages |
| My Saved Views | Saved views |
| Share Report (e-mail) | Not offered (e-mail is out of scope) |
| Monthly / Quarterly aggregation | Period |
