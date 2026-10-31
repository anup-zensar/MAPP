# Research: Plan Details Report (043)

Only the real choices are recorded here (build guide §8). Each states the decision, why, and what was rejected.

## R1 — Read the year through All work, not a new query

**Decision**: `plandetails` reads one year through a `Planning` port with All work's `Year`, `Attributes` and
`Choices` (`allwork.Source`), adapted in `app/measure_plandetails.go` exactly as 036's
`dashboardPlanning` does. It asks for the columns `completeness` and `inq`, so the year carries every record's
`Missing` count and the activities' goals. No `queries/measure_plandetails.sql` is written.

**Rationale**: `allwork.Source.Year` already returns every plan, campaign, initiative and activity of the year in
the person's scope (`PlanStore.ListInScope(authz.ScopeOf(ctx))`, covered by 023's scope-isolation test) with
what 043 needs: `Attrs` (choice and yes/no values with shares), `Fields["attr.<id>"]` (text and number values),
`Missing` (domain model §2.6, the figure of 036's completeness panel), `INQ`/`MQL`/`Revenue`, `Status`,
`Timing`, `StartDate`, `ActivityTypeID`, `BusinessUnitIDs` and `PlannedCost` in USD. It was checked against 028
(statements do not grow with the records) and is the read 036 benchmarks. A second query would define planned
cost, completeness and scope a second time (build guide §5: "a report never defines … itself").

**Rejected**: a dedicated sqlc query over `activities`, `attribute_values` and `plans` — faster for one
attribute, but it re-derives planned cost and completeness and needs its own scope-isolation test; calling
`allwork.Service.Breakdown` once per attribute — one year read per attribute (30 at the 028 volume) breaks
SC-001.

## R2 — Where 016's attribution rule lives

**Decision**: `plandetails/attribution.go` holds the rule of 016 FR-041 for one record: the values it carries,
a % split dividing its planned cost by the shares (half up to the cent, the remainder on the last value), a
multiple choice without shares giving each value the whole amount. A test,
`TestAttribution_EqualsAllWorkBreakdown_SC002`, runs `allwork.Service.Breakdown` on the same in-memory year for
every choice and yes/no attribute of the demo data and asserts value-for-value equality (Not filled in plus Does
not apply against Breakdown's Not filled in), so the copy cannot drift.

**Rationale**: the rule's code in `allwork` (`attributeValues`, `split`, `shareOfAmount`) is unexported and
`allwork` is outside lane D's paths (issue #36 "Owns"). The copy is about 40 lines.

**Preferred alternative, for the integrator**: export it once from `allwork` (for example
`allwork.AttributeParts(a Attribute, n Node, units []string, amount money.Amount) []Part`, with Breakdown calling
it — no behaviour change), as 036 exported `allwork.PlannedByActivity`. If the integrator approves this in wave A,
`attribution.go` shrinks to a call and the equality test stays. Raised in the pull request, not done here.

## R3 — Does not apply (043-C2)

**Decision**: worked out in `plandetails` from exported data: an attribute applies to a record when
`Attribute.Condition` is empty or shares an option with its plan's `BusinessUnitIDs` (the rule of
`allwork.Attribute.shownFor`, 002 FR-022). A record where it does not apply goes to `doesNotApply` before the
value is read; otherwise an absent or empty value goes to `notFilledIn`. All work's reader turns an empty
yes/no into No, so a yes/no attribute is never Not filled in: it is Yes, No or, where hidden, Does not apply.

**Rationale**: `allwork.valueOf` folds hidden into not filled in, which is right for the Breakdown and wrong for
043-C2; the condition and the units are exported fields, so no `allwork` change is needed.

## R4 — Filters, period and export come from 039

**Decision**: the handler parses 039's `measure.Filter` and `measure.Periods` once from the query string and
adds only `level`; the service applies the filter to the year's nodes with what 039 provides and never reads the
request. The report implements 039's `measure.Report`; its `Sheets` method turns each widget into one
`platform/sheet` table, which 039's shared export writes as CSV or Excel.

**Rationale**: build guide §5–§6 and D-368; 043 must not redefine the frame (spec FR-015).

**Open until 039 is pushed**: the exact names (`measure.Filter`, its node-matching function, `measure.Report`,
`Sheets`' signature, the shared widget and state schemas). `contracts/api.md` names them as 039's convention
describes them; the integrator aligns them when writing the wave A contract.

## R5 — Benchmark inside the package

**Decision**: `BenchmarkPlanDetails_SC001` in `plandetails` runs the service over an in-memory year at the 028
volume (50 plans, 5,000 activities, 20,000 line items, 30 active attributes) through a fake `Planning`; the
year read itself is already measured by 036's `BenchmarkDashboards` and 028.

**Rationale**: `internal/platform/apibench` is not a lane D path. Target: the calculation adds under 300 ms to
the year read, keeping each tab within SC-001's 3 s.

**Rejected**: an `apibench` file — needs the integrator's agreement; proposed in the pull request if the
reviewers want an end-to-end figure.
