# Research: SP&A Plan View and BCPIR Marker

Each item gives the decision, why, and what was rejected. Decisions D-334–D-337 record them in the log.

## R1 — Model A: BCPIR is the plan's flag (D-334)

**Decision**: keep "BCPIR plan" as the plan's yes/no (004); add a marker and an Include/Only/Exclude filter
to the Plans list and All work, and a Breakdown grouping. No new record, no sign-off.
**Why**: Shell's spec 006 makes Model A its default (FR-032) and Hive9 has the checkbox (FA §B3); Shell's spec
also asks for the flag as "a filter and grouping value" of the plan workspace (FR-002), which in MAPP is All
work. The filter copies All work's Strategic / AOP filter (include, only, exclude), so it behaves like a
filter people already know.
**Rejected**: Model B (a pair of SP&A and BCPIR plans per campaign with their own approval) — no Shell
evidence beyond the ZenReach reference, and it would duplicate the plan's life cycle.

## R2 — Committed and Actual spend are 015's figures (D-335)

**Decision**: Committed = 015's Committed (purchase-order lines; 033's simulated POs are booked as such);
Actual spend = 015's Paid; Invoiced = 015's Invoiced. Purchase-request amounts are shown separately as
Approved / Pending approval / Draft.
**Why**: every MAPP screen (plan page, All work, Home, Financials) already calls PO lines "Committed"; using
006's PR-based definition would show two committed totals for the same plan. 015's research recorded
"actuals are the payments released".
**Rejected**: 006's Committed from approved PR lines (conflicts with D-243, D-331).

## R3 — Planned cost by period and by year (D-336)

**Decision**: per activity, the monthly planned cost is its line items' USD spreads (007's `USDMonths`, as All
work reads them) plus its positive unassigned balance split evenly over the activity's months (money's
`SplitEvenly`, the rest in the last month) — the rule of All work's Budget & cost (010 FR-023) and the plan's
Budget by month (016 FR-024). The activity's planned cost in the plan's year is its planned cost minus the
months that fall in other years; what cannot be placed (no dates, a negative balance) is Not yet scheduled.
Transactions count in the month of their posting date; those posted in other years are left out, with a note.
**Why**: one planned-cost-by-month rule for every period view of a plan; 006 FR-009 limits a plan's figures to
its fiscal year.
**Rejected**: reusing All work's Budget & cost answer directly — it is in the plan currency and has no
activity contributions or actuals.

## R4 — Dimensions (D-336)

**Decision**: the dimensions are Activity type and every active choice or yes/no Shell attribute of the
activity and line-item levels, in Settings' order, with the activity attribute labelled "Channel / audience"
first and the default. Splitting follows All work's Breakdown (D-270): one value takes the whole amount, a %
split divides by shares rounded half up with the last share taking the rest, several values without shares
each take the whole amount and the answer says so. For a line-item attribute, an activity's planned cost not
on a line item is "Not yet itemised"; an empty value is "Not filled in". Rows are sorted largest first, then
by label.
**Why**: 006 FR-016 asks for Channel / audience (default), activity type, activity category, spend category
and spend type; MAPP's Settings hold Channel / audience, Spend type and Spend category as attributes and has no
activity category, so offering the attributes of those levels covers 006 and whatever Shell adds.
**Rejected**: a fixed list of three attributes by name (breaks when Settings change).

## R5 — Where the inputs come from

**Decision**: `spa` declares three ports: `Planning` (the plan in the actor's scope, its live records and the
dimensions — implemented in `internal/app` over All work's `PlanYear` reader, which already reads the plan
tree once per level with 007's spreads and 016's attributes), `Finance` (`FactsByLineItemMonth` and
`DataAsOf` of `financials`, and 014's `finance.view` check) and `Purchases` (032's new `DrawnOnLineItems`, which
reads `OnLineItems` with the actor's view scope and converts lines to USD at today's rates as 032's own
list does).
**Why**: no third copy of the plan-tree reading; 015 and 032 keep their own conversions and scope rules.
**Rejected**: adding the view to the `allwork` package (it is the largest package already and the view also
needs PRs); calling the HTTP API from the web app several times and combining figures there (constitution IV).

## R6 — Traceability and bars come from the server

**Decision**: every headline figure, period and dimension row carries the list of contributing activities with
their amounts (non-zero only, in table order); bars carry a whole-number length 0–100 relative to the largest
amount of their set; shares carry two decimals.
**Why**: constitution IV — the web app never computes a figure; money is never a float in the browser.

## R7 — Raise purchase request

**Decision**: each activity row has an availability (014's `purchaseRequests.edit` in the plan, Purchase
support on, and at least one line item); the web app lists the activity's line items from the view and opens
032's `/purchase-requests/new?lineItemId=…` for the one chosen.
**Why**: 032's create flow stays the only place that raises a PR (006 FR-012, FR-020) and re-checks access.
