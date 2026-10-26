package allwork

import (
	"time"

	"mapp/internal/platform/date"
)

// The answer of GET /api/v1/all-work as a pure function of the year's nodes (D-114): build takes everything a
// view depends on and returns the rows in their final order with their cells, the summary cards, the count
// and the totals row. Service.Query reads the input; tests call build directly.

// unassignedName is the name of an activity's unassigned planned cost row (FR-012).
const unassignedName = "Unassigned planned cost (Sys_Gen_Balance)"

// input is everything one view is worked out from.
type input struct {
	query      Query
	year       int // the first year of the range
	yearFrom   int // 016: the range (0: year)
	yearTo     int
	attributes []Attribute // 016: every Shell attribute, for the attribute filters and the Breakdown
	rateNotes  []string    // 016: the rates used, for an export's notes (FR-009)
	today      date.Date
	now        time.Time
	currency   Currency
	data       YearData
	choices    Choices  // the teams, business units, activity types and people that exist
	catalogue  []Column // the columns the grid offers
}

// built is a worked-out view: the answer, the tree it came from and the query as it applied, without the
// values left out (exports use all three).
type built struct {
	result Result
	tree   *tree
	query  Query
}

// build works out the view of in (FR-001 – FR-026): it marks the matches and context rows, adds up the
// roll-ups, sorts, and makes the rows — with the period amounts of Budget & cost — the cards, the count and
// the totals row. Filter values and columns that no longer exist are ignored and listed in leftOut (FR-031).
func build(in input) built {
	if in.yearFrom == 0 {
		in.yearFrom = in.year
	}
	if in.yearTo < in.yearFrom {
		in.yearTo = in.yearFrom
	}
	q, leftOut := knownValues(in.query, in.data, in.choices)
	q.attrs = resolveAttrs(q.Attributes, in.attributes, in.data.Nodes) // 016 FR-039
	if q.View == ViewBreakdown {                                       // its rows are the Grid's; the groups come from GET /api/v1/all-work/breakdown
		q.View = ViewGrid
	}
	in.query = q
	t := buildTree(in.data.Nodes)
	if in.choices.SeesFinance {
		t.withFinance(in.choices.FinancePlans) // feature 015 (finance.go)
	}
	applyFilters(t, q)
	rollUp(t)
	columns, sort, columnsLeftOut := resolveColumns(q, in.catalogue)
	leftOut = append(leftOut, columnsLeftOut...)
	if sort != nil {
		sortTree(t, catalogueOf(in.catalogue)[sort.Column], sort.Direction, in.currency, in.year)
	}
	if q.View != ViewGrid {
		columns = []Column{}
	}
	m := matchesOf(t)
	b := rowBuilder{in: in, columns: columns, periodColumns: []Period{}}
	if q.View == ViewBudget {
		p := periodsOf(t, in.yearFrom, in.yearTo)
		b.periods = &p
		b.periodColumns, b.periodBuckets = p.columns(q.Period)
	}
	result := Result{
		Year:            in.yearFrom,
		YearFrom:        in.yearFrom,
		YearTo:          in.yearTo,
		Today:           in.today,
		View:            q.View,
		DisplayCurrency: in.currency.Code,
		GeneratedAt:     in.now.UTC(),
		Columns:         columns,
		Periods:         b.periodColumns,
		Rows:            b.rows(t),
		Summary:         m.summary(in.currency),
		Count:           countOf(t, q, m),
		Totals:          b.totals(m),
		LeftOut:         leftOut,
	}
	return built{result: result, tree: t, query: q}
}

// knownValues returns q without the filter values that no longer exist, and those values as left out
// (FR-031): plans that are not plans of the year, and people, teams, Business unit options and activity
// types that are not among the choices. The rest of the query applies.
func knownValues(q Query, data YearData, c Choices) (Query, []LeftOut) {
	plans := map[string]bool{}
	for _, n := range data.Nodes {
		if n.Level == LevelPlan {
			plans[n.ID] = true
		}
	}
	leftOut := []LeftOut{}
	keep := func(filter string, values []string, known func(id string) bool) []string {
		kept := []string{}
		for _, id := range values {
			if known(id) {
				kept = append(kept, id)
			} else {
				leftOut = append(leftOut, LeftOut{Filter: filter, ID: id})
			}
		}
		return kept
	}
	q.Plans = keep("plan", q.Plans, func(id string) bool { return plans[id] })
	q.Owners = keep("owner", q.Owners, optionIn(c.People))
	q.Teams = keep("team", q.Teams, optionIn(c.Teams))
	q.BusinessUnits = keep("businessUnit", q.BusinessUnits, optionIn(c.BusinessUnits))
	q.ActivityTypes = keep("activityType", q.ActivityTypes, optionIn(c.ActivityTypes))
	return q, leftOut
}

// optionIn returns the test of an ID being one of options.
func optionIn(options []Option) func(id string) bool {
	return func(id string) bool {
		for _, o := range options {
			if o.ID == id {
				return true
			}
		}
		return false
	}
}

// rowBuilder makes the rows of one view.
type rowBuilder struct {
	in            input
	columns       []Column
	periods       *periods // Budget & cost only
	periodColumns []Period
	periodBuckets [][]string // the buckets each period column adds up
}

// withPeriods adds to row the period amounts and total of b (Budget & cost, FR-022).
func (b rowBuilder) withPeriods(row *Row, amounts buckets) {
	if b.periods != nil {
		row.Periods, row.Total = b.periods.amounts(amounts, b.periodBuckets, b.in.currency)
	}
}

// rows returns the shown records in tree order (FR-011): each plan, then its campaigns, initiatives and
// activities; under an activity its line items and, when its balance is not zero, its unassigned planned
// cost row (FR-012). The Calendar has no line-item or unassigned rows (FR-025).
func (b rowBuilder) rows(t *tree) []Row {
	rows := []Row{}
	var add func(n *tnode)
	add = func(n *tnode) {
		if !n.shown {
			return
		}
		rows = append(rows, b.recordRow(n))
		if n.n.Level != LevelActivity {
			for _, c := range n.children {
				add(c)
			}
			return
		}
		if b.in.query.View == ViewCalendar {
			return
		}
		for _, li := range n.children {
			rows = append(rows, b.recordRow(li))
		}
		if n.balance != nil && !n.balance.IsZero() {
			rows = append(rows, b.unassignedRow(n))
		}
	}
	for _, p := range t.plans {
		add(p)
	}
	parents := map[string]bool{}
	for _, r := range rows {
		if r.ParentKey != nil {
			parents[*r.ParentKey] = true
		}
	}
	for i := range rows {
		rows[i].HasChildren = parents[rows[i].Key]
	}
	return rows
}

// recordRow returns the row of a plan, campaign, initiative, activity or line item.
func (b rowBuilder) recordRow(t *tnode) Row {
	n, cur := t.n, b.in.currency
	// A line item is never tested by filters: it takes the role of its activity (D-113).
	roleOf := t
	if n.Level == LevelLineItem && t.parent != nil {
		roleOf = t.parent
	}
	row := Row{
		Key: n.ID, Level: n.Level, RecordID: n.ID, PlanID: n.PlanID, Depth: t.depth, Name: n.Name,
		Role: roleOf.role(), SearchHit: t.hit, Status: statusOf(n), Owner: n.Owner, PlanType: n.PlanType,
		Cells: b.cells(cellRow{level: n.Level, t: t, year: yearOf(planOf(t).n, b.in.year)}),
	}
	if t.parent != nil {
		parent := t.parent.n.ID
		row.ParentKey = &parent
	}
	switch n.Level {
	case LevelPlan:
		strategic, year := n.Strategic, yearOf(n, b.in.year)
		row.Strategic, row.Year = &strategic, &year
	case LevelCampaign, LevelInitiative:
		row.Timing = timingOf(n)
	case LevelActivity:
		row.Balance = cur.optional(t.balance)
	case LevelLineItem:
		row.ActivityID = n.ParentID
	}
	if n.StartDate != nil {
		row.StartDate = *n.StartDate
	}
	if n.EndDate != nil {
		row.EndDate = *n.EndDate
	}
	row.Ongoing = n.Ongoing
	if t.isContainer() {
		row.PlannedCost = cur.optional(t.fig.planned)
		row.Budget = cur.optional(n.Budget)
		row.Filtered = t.filtered()
	} else {
		row.PlannedCost = cur.optional(n.PlannedCost)
	}
	if b.periods != nil {
		b.withPeriods(&row, b.periods.of[t])
		if b.in.query.ComparePhased && t.isContainer() { // 016 FR-038
			row.PhasedBudget, row.OverPeriods = b.periods.phased(t, b.periods.of[t], b.periodBuckets, b.in.currency)
		}
	}
	return row
}

// yearOf is the year of the record's plan: its PlanYear, or year when the source did not say.
func yearOf(n *Node, year int) int {
	if n.PlanYear != 0 {
		return n.PlanYear
	}
	return year
}

// unassignedRow returns the unassigned planned cost row of the activity a (FR-012): it opens its activity
// and takes its role.
func (b rowBuilder) unassignedRow(a *tnode) Row {
	parent := a.n.ID
	row := Row{
		Key: a.n.ID + ":unassigned", ParentKey: &parent, Level: LevelUnassigned, RecordID: a.n.ID,
		PlanID: a.n.PlanID, ActivityID: a.n.ID, Depth: a.depth + 1, Name: unassignedName, Role: a.role(),
		PlannedCost: b.in.currency.optional(a.balance),
		Cells:       b.cells(cellRow{level: LevelUnassigned, t: a, year: yearOf(planOf(a).n, b.in.year)}),
	}
	if b.periods != nil {
		b.withPeriods(&row, b.periods.unassigned[a])
	}
	return row
}

// cells returns the cells of a row, one per column.
func (b rowBuilder) cells(r cellRow) []*Cell {
	cells := make([]*Cell, len(b.columns))
	for i, c := range b.columns {
		cells[i] = cellOf(c, r, b.in.currency)
	}
	return cells
}

// totals returns the totals row (FR-014): "Total · N activities", money summed over the matching
// activities and the Plan budget over the matching plans, so it equals the summary cards.
func (b rowBuilder) totals(m matches) Totals {
	cur := b.in.currency
	totals := Totals{
		Activities:  m.activities.activities,
		Cells:       make([]*Cell, len(b.columns)),
		PlannedCost: cur.optional(m.activities.planned),
		PlanBudget:  cur.optional(m.planBudget),
	}
	for i, c := range b.columns {
		switch c.Key {
		case "planBudget":
			totals.Cells[i] = moneyCell(cur, m.planBudget)
		case "plannedCost", "lineItems", "unassigned", "activities", "inq", "mql", "revenue",
			columnCommitted, columnInvoiced, columnPaid, columnAccrued, columnNotYetCommitted:
			totals.Cells[i] = rolledCell(c.Key, &tnode{n: &Node{}}, m.activities, cur)
		}
	}
	if b.periods != nil { // the footer of Budget & cost: the matching activities by period (FR-024)
		totals.Periods, totals.Total = b.periods.amounts(b.periods.footer, b.periodBuckets, cur)
	}
	return totals
}
