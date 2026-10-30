# Measure metrics — every figure defined once

The figures the Reports area shows (features 039–050), each defined **once** and reused by every report. A report
never redefines a figure; where a figure already exists in the domain model or feature 036, the row names it.
Money follows [domain-model.md](domain-model.md) §4: exact decimals, USD base, converted once to the display
currency at the current rate (D-303); percentages are whole per cent, half up, unless a row says otherwise; a
figure that cannot be worked out shows "—" with the reason, never 0.

## 1. Money and planning figures (wave A)

| Figure | Definition | Already defined in | Shown by |
|---|---|---|---|
| **Budget** | The plan, campaign or initiative budget; an allocated plan's budget is its allocation | 004, 014 (D-342) | 040, 048 |
| **Planned cost** | The activities' planned cost; a line item's is its own | domain model §4.2 | 040, 043, 048, 049 |
| **Planned cost by period** | Line items by their spread, an activity's positive unassigned balance evenly over its months; amounts without dates as *No dates*; months outside the plan's year noted | 010 FR-023, D-266, D-336 | 040 |
| **Budget used** · **Left to plan** | Planned ÷ budget as a whole per cent · budget − planned | plan page (004) | 040 |
| **Committed** | Sum of the purchase-order lines of the plan's year, the simulated GSAP's POs included | domain model §4.6, D-335, D-358 | 040, 042 |
| **Invoiced** | Sum of the invoices less credit notes | domain model §4.6 | 042, 044 |
| **Actual** (paid) | Sum of the payments posted in the plan's year | domain model §4.6, D-342, D-357 | 040, 048, 050 |
| **Actual by period** | Payments by their posting month at the current rate | 036 (D-342) | 040 |
| **Unspent budget** · **Budget spent** | Budget − actual · actual ÷ budget as a whole per cent | 036 (D-342) | 040 |
| **Not yet committed** · **% committed** | Planned − committed · committed ÷ planned | domain model §4.6 | 040 |
| **Approvals waiting / decided / automatic** | Counts over the approval rounds of the plans' activities and purchase requests, and since when the oldest waits | 017, 036 (D-342) | 041 |
| **Average days to a decision** | Mean days from submission to decision, one decimal, as Approvals monitoring works it out | 017 | 041, 048 |
| **Purchase requests by status** | Count and amount per status (Created, Submitted, In review, Approved, Declined, Closed) and how many approved ones wait for GSAP, as the Purchase requests page | 032, 036 (D-342) | 042 |
| **Commitment by vendor / by owner per delivery month** | PR line amounts of Approved and Closed requests bucketed by the line's delivery date; a Closed line is its closed-out amount | 032, D-335 | 042 |
| **Invoiced per PR line** | Invoices linked to the line's line item and purchase order | 015, 033 | 042, 044 |
| **Completeness** | Records with every needed field filled in, per level, as a count and a whole per cent | domain model §2.6, 036 | 043, 048 |
| **Records by attribute value** | For one Shell attribute at one level: how many records carry each value, their planned cost, and how many have it *Not filled in*; % splits count by share (D-270, D-336) | 016 Breakdown | 043 |
| **Planned cost entered** · **Goal entered** | Activities with a planned cost above zero · activities with a goal | 006 | 043 |
| **SWBS not received** | Line items whose SWBS request is sent and not answered, with the request date | 030 | 044 |
| **Waiting for GSAP** | Approved purchase requests routed and without a GSAP PR number | 032, 033 | 044 |
| **Purchase orders per line item** | The PO ledger: line item, purchase request, GSAP PR number, PO number and date, PO amount, invoiced, paid | 033, 015 | 044 |
| **Last sign-in** | The time of the person's latest successful demo sign-in, in UTC | 045 (new) | 045 |

## 2. Goals, performance and projections (wave B)

| Figure | Definition | Already defined in | Shown by |
|---|---|---|---|
| **Goal INQ / MQL / Revenue** (plan) | The plan goal at its goal stage and the other stages from the model's rates and average deal size | domain model §4.5 | 048, 049 |
| **Goal per stage** (activities) | The sum of the activities' goals of the plans in view, per stage | domain model §4.5 | 048, 049 |
| **Goal coverage** | Activities' goals ÷ plan goals per stage, a whole per cent — a planning-completeness check, not performance | 048 (new) | 048, 049 |
| **Actual INQ / MQL / Deals / Revenue** | Performance facts matched to the activities in view, by the fact's date; a fact matched to no activity counts nowhere and is listed to review | 046 (new) | 048, 049 |
| **Projected INQ / MQL / Revenue by period** | From each activity's goal stage forward: the model's conversion rates, placed by its velocity (months between stages) from the activity's start date, to the end of the plan year; revenue = deals × average deal size | 047 (new), domain model §4.5 | 048, 049 |
| **Actual or projected** | Actual to the end of the last period before today, projected from today | 047 | 048, 049 |
| **Performance** | (Actual + projected) ÷ goal as a whole per cent; the variance word is *Above plan* or *Below plan* | 048 (new) | 048, 049 |
| **Conversion rate** | Actual of a stage ÷ actual of the previous stage, one decimal; **conversion goal** is the model's rate | 048 (new), domain model §4.5 | 048 |
| **Attributed revenue** | Revenue facts credited to activities by the chosen attribution model; the model is named next to the figure | 050 (new) | 048, 049, 050 |
| **ROI** | (Attributed revenue − cost) ÷ cost, a whole per cent, with the cost basis named: cost is **Actual** (paid) when the plans in view have payments in the year, otherwise "—"; a report may offer *Use planned cost* as a labelled choice, never silently (Hive9's "Use Planned Cost as Actual Cost" had no on-screen sign) | 050 (new; confirm in its spec) | 048, 049 |
| **Is integrated** | An activity with a data source matched to it (external name or integration ID) | 046 (new) | 043, 046 |

## 3. Hive9 figures MAPP does not show

| Hive9 figure | Why not | Where it is recorded |
|---|---|---|
| Actual at monthly FX | MAPP converts at the current rate everywhere; dated rates exist since 016, so it could be offered later | Q-023 |
| Annual budget, budget YTD, forecast, burn rate of a **finance budget** | The Finance budgets module is out of scope | constitution 4.8.0 |
| Planned − actual in period, recommended accrual, accrual month | Accrual requests are out of scope | constitution 4.8.0 |
| Goods receipt (GRSP), vendor delivery performance | The simulated GSAP sends no receipts | Q-030 |
| Released amount, activity and PO occupied rate | Budget release is a finance-module construct | wave C |
| Configurable summary-card strip | Fixed cards per report until Shell asks | Q-029 |
