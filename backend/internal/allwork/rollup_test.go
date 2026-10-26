package allwork

import (
	"testing"

	"mapp/internal/platform/money"
)

// allColumns asks for every column of the Standard group.
func allColumns() []string {
	keys := make([]string, 0, len(standardColumns))
	for _, c := range standardColumns {
		keys = append(keys, c.Key)
	}
	return keys
}

// Containers add up the matching activities shown below them (FR-013, domain model §4.2–4.5): planned
// cost, line items, the positive balances, the number of activities, INQ, MQL and revenue.
func TestRollup_ContainersSumMatchesShown_FR013(t *testing.T) {
	r := view(seedYear(), Query{Columns: allColumns()})

	tests := []struct{ row, column, want string }{
		{"P-1001", "plannedCost", "460000.00 USD"},
		{"P-1001", "lineItems", "165000.00 USD"},
		{"P-1001", "unassigned", "295000.00 USD"},
		{"P-1001", "activities", "7"},
		{"P-1001", "inq", "310000"},
		{"P-1001", "mql", "6200"},
		{"P-1001", "revenue", "2620000.00 USD"},
		{"P-1002", "plannedCost", "190000.00 USD"},
		{"P-1002", "unassigned", "150000.00 USD"}, // −500.00 of Grand opening Shanghai is not subtracted
		{"C-1001", "plannedCost", "275000.00 USD"},
		{"I-1001", "plannedCost", "200000.00 USD"},
		{"I-1001", "lineItems", "125000.00 USD"},
		{"I-1001", "unassigned", "75000.00 USD"},
		{"I-1003", "lineItems", "—"}, // no line items below
		{"I-1007", "plannedCost", "—"},
		{"I-1007", "activities", "0"},
		{"A-1001", "plannedCost", "120000.00 USD"},
		{"A-1001", "lineItems", "100000.00 USD"},
		{"A-1001", "unassigned", "20000.00 USD"},
		{"A-1001", "inq", "250000"},
		{"A-1008", "unassigned", "-500.00 USD"},
		{"P-1001", "planBudget", "1200000.00 USD"},
		{"P-1001", "remainingBudget", "50000.00 USD"}, // 1,200,000 − 700,000 − 450,000
		{"C-1003", "remainingBudget", "50000.00 USD"}, // 600,000 − 350,000 − 200,000 (Loyalty has no budget)
		{"I-1001", "remainingBudget", ""},             // initiatives have no children with budgets: blank
		{"P-1001", "budgetUsed", "38 %"},              // 460,000 ÷ 1,200,000 = 38.3 %
		{"I-1001", "budgetUsed", "67 %"},              // 200,000 ÷ 300,000
	}
	for _, tt := range tests {
		if got := cell(r, tt.row, tt.column); got != tt.want {
			t.Errorf("%s %s = %s, want %s", tt.row, tt.column, got, tt.want)
		}
	}
	if row := rowOf(r.Rows, "P-1001"); moneyText(row.PlannedCost) != "460000.00 USD" || moneyText(row.Budget) != "1200000.00 USD" || row.Filtered {
		t.Errorf("plan row: planned cost %s, budget %s, filtered %v", moneyText(row.PlannedCost), moneyText(row.Budget), row.Filtered)
	}
}

// When filters hide activities below a container, its roll-ups count only the matching ones shown, are
// marked Filtered, and carry the value without filters (FR-013).
func TestRollup_UnfilteredValueWhenRecordsHidden_FR013(t *testing.T) {
	data := seedYear().data()
	tr := buildTree(data.Nodes)
	matchEverything(tr)
	hidden := tr.byID["A-1001"] // Launch film is hidden by a filter
	hidden.match, hidden.shown = false, false
	for _, li := range hidden.children {
		li.match, li.shown = false, false
	}
	rollUp(tr)

	plan := tr.byID["P-1001"]
	if !plan.filtered() || plan.fig.activities != 6 || plan.all.activities != 7 {
		t.Fatalf("plan: filtered %v, activities %d of %d; want filtered, 6 of 7", plan.filtered(), plan.fig.activities, plan.all.activities)
	}
	c := cellOf(standard("plannedCost", "Planned cost", KindMoney, costLevels), cellRow{level: LevelPlan, t: plan, year: 2026}, usd)
	if got := cellText(c); got != "340000.00 USD [all: 460000.00 USD]" {
		t.Errorf("planned cost cell = %s, want 340000.00 USD with 460000.00 USD without filters", got)
	}
	if tr.byID["C-1002"].filtered() {
		t.Errorf("Rimula fleet has nothing hidden below it")
	}
	budget := cellOf(standard("planBudget", "Plan budget", KindMoney, containerLevels), cellRow{level: LevelPlan, t: plan, year: 2026}, usd)
	if budget.Unfiltered != nil {
		t.Errorf("the plan's own budget is not a roll-up: %s", cellText(budget))
	}
}

// The six summary cards count the matching campaigns, initiatives and activities and add up their planned
// cost, the matching plans' budgets and the positive balances (FR-003).
func TestSummary_SixCards_FR003(t *testing.T) {
	s := view(seedYear(), Query{}).Summary

	if s.Campaigns != 3 || s.Initiatives != 7 || s.Activities != 10 || moneyText(s.PlannedCost) != "650000.00 USD" ||
		moneyText(s.PlanBudget) != "2000000.00 USD" || moneyText(s.UnassignedPlannedCost) != "445000.00 USD" ||
		s.NegativeBalanceCount != 1 {
		t.Errorf("summary = %+v, planned %s, budget %s, unassigned %s", s, moneyText(s.PlannedCost),
			moneyText(s.PlanBudget), moneyText(s.UnassignedPlannedCost))
	}
}

// The Unassigned planned cost card adds only positive balances; a negative balance is counted apart, so the
// card never shrinks because line items exceed a planned cost (FR-003, D-116).
func TestSummary_UnassignedPositiveOnlyAndNegativeCount_FR003(t *testing.T) {
	b := newYear()
	p := b.plan("P-1001", "Plan")
	c := b.campaign(p, "C-1001", "Campaign")
	i := b.initiative(c, "I-1001", "Initiative")
	over := b.activity(i, "A-1001", "Over", planned("1000.00"))
	b.lineItem(over, "L-1001", "Too much", "1500.00")
	again := b.activity(i, "A-1002", "Over again", planned("100.00"))
	b.lineItem(again, "L-1002", "Too much", "300.00")
	b.activity(i, "A-1003", "Left", planned("2500.00"))

	s := view(b, Query{}).Summary

	if moneyText(s.UnassignedPlannedCost) != "2500.00 USD" || s.NegativeBalanceCount != 2 {
		t.Errorf("unassigned %s with %d negative, want 2500.00 USD with 2", moneyText(s.UnassignedPlannedCost), s.NegativeBalanceCount)
	}
}

// The totals row reads "Total · N activities"; its money sums the matching activities and its Plan budget
// the matching plans, so it equals the summary cards (FR-014, SC-003).
func TestRollup_TotalsEqualSummaryCards_FR014_SC003(t *testing.T) {
	for _, q := range []Query{{}, {Columns: allColumns()}, {View: ViewCalendar}, {View: ViewBudget}} {
		r := view(seedYear(), q)

		assertTotalsEqualCards(t, r)
	}
}

// assertTotalsEqualCards checks the totals row against the summary cards, column by column.
func assertTotalsEqualCards(t *testing.T, r Result) {
	t.Helper()
	totals, s := r.Totals, r.Summary
	shown := map[string]int{"activities": s.Activities, "campaigns": s.Campaigns, "initiatives": s.Initiatives}[r.Count.Noun]
	if totals.Activities != s.Activities || moneyText(totals.PlannedCost) != moneyText(s.PlannedCost) ||
		moneyText(totals.PlanBudget) != moneyText(s.PlanBudget) || r.Count.Shown != shown {
		t.Errorf("totals %d activities, %s planned, %s budget; cards %d, %s, %s; count %d", totals.Activities,
			moneyText(totals.PlannedCost), moneyText(totals.PlanBudget), s.Activities, moneyText(s.PlannedCost),
			moneyText(s.PlanBudget), r.Count.Shown)
	}
	for i, c := range r.Columns {
		got := cellText(totals.Cells[i])
		var want string
		switch c.Key {
		case "plannedCost":
			want = orDash(s.PlannedCost)
		case "planBudget":
			want = orDash(s.PlanBudget)
		case "unassigned":
			want = orDash(s.UnassignedPlannedCost)
		default:
			continue
		}
		if got != want {
			t.Errorf("totals %s = %s, card %s", c.Key, got, want)
		}
	}
}

// orDash writes a card's amount as a cell would.
func orDash(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return moneyText(m)
}

// Money is shown in the person's display currency (FR-017): every sum is made in USD and converted once, so
// the total of EUR amounts does not drift by the rounding of each converted value.
func TestMoney_DisplayCurrency_SumInUSDThenConvert_FR017(t *testing.T) {
	b := newYear()
	p := b.plan("P-1001", "Plan", budget("0.01"))
	c := b.campaign(p, "C-1001", "Campaign")
	i := b.initiative(c, "I-1001", "Initiative")
	for _, id := range []string{"A-1001", "A-1002", "A-1003"} {
		b.activity(i, id, id, planned("0.01")) // 0.01 USD × 0.5 = 0.005 → 0.01 EUR each, 0.02 EUR in all
	}
	eur := Currency{Code: "EUR", Rate: mustRate("0.5")}

	r := view(b, Query{Columns: []string{"plannedCost"}}, eur)

	if r.DisplayCurrency != "EUR" || moneyText(r.Summary.PlannedCost) != "0.02 EUR" || cell(r, "P-1001", "plannedCost") != "0.02 EUR" ||
		cell(r, "A-1001", "plannedCost") != "0.01 EUR" || moneyText(r.Totals.PlannedCost) != "0.02 EUR" {
		t.Errorf("display EUR: card %s, plan %s, activity %s, totals %s; want 0.02, 0.02, 0.01, 0.02 EUR",
			moneyText(r.Summary.PlannedCost), cell(r, "P-1001", "plannedCost"), cell(r, "A-1001", "plannedCost"),
			moneyText(r.Totals.PlannedCost))
	}
	// A line item shows its USD planned cost converted; its own currency stays in its own column (FR-017).
	r = view(seedYear(), Query{Columns: []string{"plannedCost"}}, Currency{Code: "EUR", Rate: mustRate("0.92")})
	if got := cell(r, "L-1002", "plannedCost"); got != "46000.00 EUR" {
		t.Errorf("line item in EUR = %s, want 46000.00 EUR", got)
	}
}

// An activity without a planned cost shows "—" and counts as zero; a sum of only empty values shows "—"
// (edge case).
func TestRollup_EmptyPlannedCostCountsZeroAllEmptyIsDash_EdgeCase(t *testing.T) {
	b := newYear()
	p := b.plan("P-1001", "Plan")
	c := b.campaign(p, "C-1001", "Campaign")
	empty := b.initiative(c, "I-1001", "Only empty")
	b.activity(empty, "A-1001", "No cost")
	mixed := b.initiative(c, "I-1002", "Mixed")
	b.activity(mixed, "A-1002", "No cost either")
	b.activity(mixed, "A-1003", "With cost", planned("250.00"))

	r := view(b, Query{Columns: []string{"plannedCost", "unassigned"}})

	for row, want := range map[string]string{"A-1001": "—", "I-1001": "—", "I-1002": "250.00 USD", "P-1001": "250.00 USD"} {
		if got := cell(r, row, "plannedCost"); got != want {
			t.Errorf("%s planned cost = %s, want %s", row, got, want)
		}
	}
	if got := cell(r, "I-1001", "unassigned"); got != "—" {
		t.Errorf("unassigned of activities with neither cost nor line items = %s, want —", got)
	}
}

// mustRate parses a rate of the tests.
func mustRate(text string) money.Rate {
	r, err := money.ParseRate(text)
	if err != nil {
		panic(err)
	}
	return r
}
