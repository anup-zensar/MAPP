package allwork

import (
	"reflect"
	"strings"
	"testing"

	"mapp/internal/platform/money"
)

// periodText writes the period amounts of a row by period label, "—" for none, and its total:
// "Mar 3333.33 · Apr 3333.33 · = 10000.00".
func periodText(r Result, row Row) string {
	return amountsText(r, row.Periods, row.Total)
}

// amountsText writes amounts by period label, leaving out empty periods, and the total.
func amountsText(r Result, amounts []*money.Money, total *money.Money) string {
	var parts []string
	for i, a := range amounts {
		if a != nil {
			parts = append(parts, r.Periods[i].Label+" "+a.Amount.String())
		}
	}
	parts = append(parts, "= "+moneyText(total))
	return strings.Join(parts, " · ")
}

// budgetYear is one activity from March to May (planned cost of the test) with the line items of the
// test, under one plan with a budget of 50,000.00.
func budgetYear(plannedCost string) (*yearBuilder, string) {
	b := newYear()
	p := b.plan("P-1001", "Plan", budget("50000.00"))
	c := b.campaign(p, "C-1001", "Campaign")
	i := b.initiative(c, "I-1001", "Initiative")
	a := b.activity(i, "A-1001", "Spring", dates("2026-03-01", "2026-05-31"), planned(plannedCost))
	return b, a
}

// A line item's period amounts are its spread (FR-023, domain model §4.4): 10,000.00 Evenly over March to
// May is 3,333.33, 3,333.33 and 3,333.34; All in first month, All in last month and a custom spread put
// their amounts where 007 spread them (US4 scenario 1).
func TestPeriods_EvenlyFirstLastCustom_FR023(t *testing.T) {
	b, a := budgetYear("16000.00")
	b.lineItem(a, "L-1001", "Evenly", "10000.00")
	b.lineItem(a, "L-1002", "First", "1000.00")
	b.months("L-1002", "2026-03", "1000.00")
	b.lineItem(a, "L-1003", "Last", "2000.00")
	b.months("L-1003", "2026-05", "2000.00")
	b.lineItem(a, "L-1004", "Custom", "3000.00")
	b.months("L-1004", "2026-02", "500.00", "2026-04", "2500.00")

	r := view(b, Query{View: ViewBudget})

	tests := map[string]string{
		"L-1001": "Mar 3333.33 · Apr 3333.33 · May 3333.34 · = 10000.00 USD",
		"L-1002": "Mar 1000.00 · = 1000.00 USD",
		"L-1003": "May 2000.00 · = 2000.00 USD",
		"L-1004": "Feb 500.00 · Apr 2500.00 · = 3000.00 USD",
		"A-1001": "Feb 500.00 · Mar 4333.33 · Apr 5833.33 · May 5333.34 · = 16000.00 USD",
	}
	for key, want := range tests {
		if got := periodText(r, rowOf(r.Rows, key)); got != want {
			t.Errorf("%s = %s\nwant  %s", key, got, want)
		}
	}
	if len(r.Periods) != 12 || r.Periods[0] != (Period{Key: "2026-01", Label: "Jan"}) || r.Periods[11].Label != "Dec" {
		t.Errorf("periods = %+v", r.Periods)
	}
	// Other views have no periods.
	if grid := view(b, Query{}); len(grid.Periods) != 0 || grid.Rows[0].Periods != nil || grid.Rows[0].Total != nil {
		t.Errorf("the grid has periods: %+v", grid.Periods)
	}
}

// A positive unassigned balance is spread evenly over the activity's months, so the activity's Total is its
// planned cost; an ongoing activity's balance is all in its start month (FR-023, US4 scenario 2).
func TestPeriods_PositiveUnassignedSpreadEvenly_OngoingFirstMonth_FR023(t *testing.T) {
	b, a := budgetYear("12000.00")
	b.lineItem(a, "L-1001", "Booked", "9000.00")
	i := b.node(a).ParentID
	b.activity(i, "A-1002", "Fund", ongoingFrom("2026-06-15"), planned("5000.00"))

	r := view(b, Query{View: ViewBudget})

	if got := periodText(r, rowOf(r.Rows, "A-1001:unassigned")); got != "Mar 1000.00 · Apr 1000.00 · May 1000.00 · = 3000.00 USD" {
		t.Errorf("unassigned = %s", got)
	}
	if got := periodText(r, rowOf(r.Rows, "A-1001")); got != "Mar 4000.00 · Apr 4000.00 · May 4000.00 · = 12000.00 USD" {
		t.Errorf("activity = %s", got)
	}
	if got := periodText(r, rowOf(r.Rows, "A-1002:unassigned")); got != "Jun 5000.00 · = 5000.00 USD" {
		t.Errorf("ongoing = %s", got)
	}
}

// A negative balance is not spread (FR-023): its row has no period amounts, and the activity adds up only
// its line items; the activity's balance tells the web app by how much its line items exceed its planned
// cost.
func TestPeriods_NegativeBalanceNotSpread_FR023(t *testing.T) {
	b, a := budgetYear("1000.00")
	b.lineItem(a, "L-1001", "Too much", "1300.00")

	r := view(b, Query{View: ViewBudget})

	unassigned := rowOf(r.Rows, "A-1001:unassigned")
	if got := periodText(r, unassigned); got != "= —" || moneyText(unassigned.PlannedCost) != "-300.00 USD" {
		t.Errorf("negative unassigned = %s, planned cost %s", got, moneyText(unassigned.PlannedCost))
	}
	activity := rowOf(r.Rows, "A-1001")
	if got := periodText(r, activity); got != "Mar 433.33 · Apr 433.33 · May 433.34 · = 1300.00 USD" ||
		moneyText(activity.Balance) != "-300.00 USD" {
		t.Errorf("activity = %s, balance %s", got, moneyText(activity.Balance))
	}
}

// Other years holds Custom amounts outside the plan's year and No dates the amounts of an activity without
// dates; each column is there only when some row has an amount in it (FR-022).
func TestPeriods_OtherYearsAndNoDatesOnlyWhenUsed_FR022(t *testing.T) {
	plain, a := budgetYear("1000.00")
	plain.lineItem(a, "L-1001", "Spring", "1000.00")
	if r := view(plain, Query{View: ViewBudget}); len(r.Periods) != 12 {
		t.Errorf("periods without other years or undated amounts = %+v", r.Periods)
	}

	b, a := budgetYear("6000.00")
	b.lineItem(a, "L-1001", "Deposit", "1000.00")
	b.months("L-1001", "2025-12", "400.00", "2026-03", "600.00")
	undated := b.activity(b.node(a).ParentID, "A-1002", "Undated", planned("5000.00"))
	b.lineItem(undated, "L-1002", "Not spread", "3000.00")

	r := view(b, Query{View: ViewBudget})

	var keys []string
	for _, p := range r.Periods[12:] {
		keys = append(keys, p.Key+" "+p.Label)
	}
	if !reflect.DeepEqual(keys, []string{"other Other years", "noDates No dates"}) {
		t.Fatalf("extra periods = %q", keys)
	}
	tests := map[string]string{
		"L-1001":            "Mar 600.00 · Other years 400.00 · = 1000.00 USD",
		"L-1002":            "No dates 3000.00 · = 3000.00 USD",
		"A-1002:unassigned": "No dates 2000.00 · = 2000.00 USD",
		"A-1002":            "No dates 5000.00 · = 5000.00 USD",
		"P-1001":            "Mar 2266.66 · Apr 1666.66 · May 1666.68 · Other years 400.00 · No dates 5000.00 · = 11000.00 USD",
	}
	for key, want := range tests {
		if got := periodText(r, rowOf(r.Rows, key)); got != want {
			t.Errorf("%s = %s\nwant  %s", key, got, want)
		}
	}
}

// Quarterly shows each quarter as the sum of its three months (FR-022, US4 scenario 3); since 016 a quarter's
// key names its year ("Q1-2026", FR-035) and its label does so only over several years.
func TestPeriods_QuarterIsSumOfMonths_FR022(t *testing.T) {
	b, a := budgetYear("10000.00")
	b.lineItem(a, "L-1001", "Evenly", "10000.00")

	r := view(b, Query{View: ViewBudget, Period: PeriodQuarter})

	var labels []string
	for _, p := range r.Periods {
		labels = append(labels, p.Key)
	}
	if !reflect.DeepEqual(labels, []string{"Q1-2026", "Q2-2026", "Q3-2026", "Q4-2026"}) {
		t.Fatalf("periods = %q", labels)
	}
	if got := periodText(r, rowOf(r.Rows, "L-1001")); got != "Q1 3333.33 · Q2 6666.67 · = 10000.00 USD" {
		t.Errorf("by quarter = %s", got)
	}
}

// Containers add up the matching activities below them in each period; the footer gives the period
// totals, the grand total — equal to the Planned cost card when no activity's line items exceed its planned
// cost — and the matching plans' budget (FR-024, US4 scenario 3).
func TestPeriods_ContainersAndFooter_GrandTotalEqualsPlannedCostCard_FR024(t *testing.T) {
	b := seedYear()
	b.node("L-1007").MonthsUSD = []MonthAmount{{Month: "2026-04", USD: amountOf("1000.00")}} // its custom spread adds up
	b.node("L-1008").PlannedCost = ptr(amountOf("39000.00"))                                 // and no line items exceed

	for _, period := range []string{PeriodMonth, PeriodQuarter} {
		r := view(b, Query{View: ViewBudget, Period: period})

		if moneyText(r.Totals.Total) != moneyText(r.Summary.PlannedCost) || moneyText(r.Totals.PlanBudget) != "2000000.00 USD" {
			t.Errorf("%s: grand total %s, Planned cost card %s, plan budget %s", period, moneyText(r.Totals.Total),
				moneyText(r.Summary.PlannedCost), moneyText(r.Totals.PlanBudget))
		}
		if r.Summary.NegativeBalanceCount != 0 {
			t.Errorf("the year still has a negative balance")
		}
	}
	r := view(b, Query{View: ViewBudget})
	// Rimula fleet: Partner co-marketing fund (ongoing from March: all 100,000.00 then), Fleet roadshow
	// (Jul–Sep) and Christmas trade gifting (Nov–Dec), all unassigned.
	if got := periodText(r, rowOf(r.Rows, "C-1002")); got != "Mar 100000.00 · Jul 20000.00 · Aug 20000.00 · Sep 20000.00 · "+
		"Nov 12500.00 · Dec 12500.00 · = 185000.00 USD" {
		t.Errorf("Rimula fleet = %s", got)
	}
	// With a filter, containers and footer count only the matching activities.
	declined := view(b, Query{View: ViewBudget, ActivityStatuses: []string{"declined"}})
	if got := periodText(declined, rowOf(declined.Rows, "P-1001")); got != "Jul 20000.00 · Aug 20000.00 · Sep 20000.00 · = 60000.00 USD" {
		t.Errorf("declined plan = %s", got)
	}
	if got := amountsText(declined, declined.Totals.Periods, declined.Totals.Total); got != "Jul 20000.00 · Aug 20000.00 · Sep 20000.00 · = 60000.00 USD" {
		t.Errorf("declined footer = %s", got)
	}
	if rowOf(r.Rows, "P-1001").Budget == nil || moneyText(rowOf(r.Rows, "P-1001").Budget) != "1200000.00 USD" {
		t.Errorf("the plan row carries its Plan budget")
	}
}
