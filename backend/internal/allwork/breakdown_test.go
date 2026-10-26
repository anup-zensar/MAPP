package allwork

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The Breakdown of 016 (FR-040 – FR-042, data-model §3.5) over twoYears: A-1 10,000.00 (P-1001, Europe,
// Helix, Germany 60 % / France 40 %, FIT4 yes, Content production, Priya; L-1 4,000.00 Media, L-2 1,000.00
// Event), A-2 6,000.00 (P-1001, Europe, Helix and Rimula, FIT4 no, Digital media, Daniel), A-3 8,000.00
// (P-2001, Asia, Content production, Priya) and A-4 without a planned cost (P-2001, Asia, Digital media, Priya;
// L-3 3,000.00 Media). Planned cost 24,000.00 USD in all.

// breakdownView works out the Breakdown of q over nodes for the years from–to, in USD.
func breakdownView(nodes []Node, q Query, from, to int) (Breakdown, error) {
	if q.Strategic == "" {
		q.Strategic = StrategicInclude
	}
	if q.ShowLevels == "" {
		q.ShowLevels = ShowEverything
	}
	if q.Period == "" {
		q.Period = PeriodMonth
	}
	if q.AmountsIn == "" {
		q.AmountsIn = AmountsInDisplay
	}
	return breakdownOf(input{
		query: q, year: from, yearFrom: from, yearTo: to, today: fixtureToday, now: fixtureNow, currency: usd,
		data: YearData{Nodes: nodes}, choices: fixtureChoices(priya.ID), catalogue: catalogue(rangeAttributes),
		attributes: rangeAttributes,
	})
}

// mustBreakdown is breakdownView that fails the test on an error.
func mustBreakdown(t *testing.T, nodes []Node, q Query) Breakdown {
	t.Helper()
	b, err := breakdownView(nodes, q, 2026, 2027)
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	return b
}

// groupsText writes the groups as "<label> <activities>/<line items> <planned cost> <share>%", with the budget
// and budget used when there are.
func groupsText(groups []BreakdownGroup) []string {
	list := []string{}
	for _, g := range groups {
		list = append(list, groupText(g.Label, g.Activities, g.LineItems, g.PlannedCost, g.Share, g.Budget, g.BudgetUsedPercent))
	}
	return list
}

// groupText writes one group as groupsText does.
func groupText(label string, activities, lineItems int, planned money.Money, share string, budget *money.Money, used *int) string {
	text := fmt.Sprintf("%s %d/%d %s %s%%", label, activities, lineItems, planned.Format(), share)
	if budget != nil {
		text += " of " + budget.Format()
	}
	if used != nil {
		text += fmt.Sprintf(" (%d %%)", *used)
	}
	return text
}

// childrenText writes the "then by" groups of g as groupsText does.
func childrenText(g BreakdownGroup) []string {
	list := []string{}
	for _, c := range g.Children {
		list = append(list, groupText(c.Label, c.Activities, c.LineItems, c.PlannedCost, c.Share, c.Budget, c.BudgetUsedPercent))
	}
	return list
}

// assertGroups compares the groups with want.
func assertGroups(t *testing.T, name string, got []BreakdownGroup, want []string) {
	t.Helper()
	if text := groupsText(got); !slices.Equal(text, want) {
		t.Errorf("%s groups =\n  %s\nwant\n  %s", name, strings.Join(text, "\n  "), strings.Join(want, "\n  "))
	}
}

// A single choice gives each activity's planned cost to the value of its ancestor at the attribute's level, in
// Settings' option order; a campaign attribute's groups have the budgets of their campaigns (FR-041, FR-040).
func TestBreakdown_SingleChoice_FR041(t *testing.T) {
	b := mustBreakdown(t, twoYears(), Query{By: "attr:SA-CL"})

	assertGroups(t, "Cluster", b.Groups, []string{
		"Europe 2/2 16,000.00 USD 66.67% of 60,000.00 USD (27 %)",
		"Asia 2/1 8,000.00 USD 33.33% of 20,000.00 USD (40 %)",
	})
	if b.CountsInEach || len(b.Notes) != 0 || b.Currency != "USD" || b.By.Key != "attr:SA-CL" || b.ThenBy != nil {
		t.Errorf("breakdown = %+v", b)
	}
}

// A multiple choice without split counts the full amount in each chosen group and says so; the totals count
// every activity once (FR-041).
func TestBreakdown_MultipleCountsInEach_FR041(t *testing.T) {
	b := mustBreakdown(t, twoYears(), Query{By: "attr:SA-B"})

	assertGroups(t, "Brands", b.Groups, []string{
		"Helix 2/2 16,000.00 USD 66.67%",
		"Rimula 1/0 6,000.00 USD 25.00%",
		"Not filled in 2/1 8,000.00 USD 33.33%",
	})
	total := groupText(b.Total.Label, b.Total.Activities, b.Total.LineItems, b.Total.PlannedCost, b.Total.Share, b.Total.Budget, b.Total.BudgetUsedPercent)
	if !b.CountsInEach || !slices.Equal(b.Notes, []string{"Activities with several values count in each group; the total counts them once."}) ||
		total != "Total 4/3 24,000.00 USD 100.00%" {
		t.Errorf("counts in each %v, notes %q, total %s", b.CountsInEach, b.Notes, total)
	}
}

// A % split divides the amount by the shares, half up per value, the last value getting the rest (FR-041):
// Germany 60 % / France 40 % of 10,000.00 → 6,000.00 / 4,000.00; 50 % / 50 % of 10,000.01 → 5,000.01 /
// 5,000.00.
func TestBreakdown_SplitRemainderOnLast_FR041(t *testing.T) {
	even := withAttr(twoYears(), "A-1", "SA-C", shares("SO-DE", "50", "SO-FR", "50"))
	for i := range even {
		if even[i].ID == "A-1" {
			a := amountOf("10000.01")
			even[i].PlannedCost = &a
		}
	}

	b := mustBreakdown(t, twoYears(), Query{By: "attr:SA-C"})
	halves := mustBreakdown(t, even, Query{By: "attr:SA-C"})

	assertGroups(t, "Country / OU", b.Groups, []string{
		"Germany 1/2 6,000.00 USD 25.00%",
		"France 1/2 4,000.00 USD 16.67%",
		"Not filled in 3/1 14,000.00 USD 58.33%",
	})
	if b.CountsInEach {
		t.Error("a % split counts each activity once")
	}
	if got := groupsText(halves.Groups); len(got) != 3 || !strings.HasPrefix(got[0], "Germany 1/2 5,000.01 USD") ||
		!strings.HasPrefix(got[1], "France 1/2 5,000.00 USD") {
		t.Errorf("50 / 50 of 10,000.01 = %q", got)
	}
}

// A line-item attribute groups the line items' planned cost and puts each activity's unassigned planned cost
// in Not on a line item, last (FR-041).
func TestBreakdown_LineItemAttributeAndNotOnALineItem_FR041(t *testing.T) {
	b := mustBreakdown(t, twoYears(), Query{By: "attr:SA-S"})

	assertGroups(t, "Spend type", b.Groups, []string{
		"Media 2/2 7,000.00 USD 25.93%",
		"Event 1/1 1,000.00 USD 3.70%",
		"Not on a line item 3/0 19,000.00 USD 70.37%",
	})
}

// Line items of activities without a planned cost are included, and the note names their amount: the totals
// row is then the Planned cost card plus that amount (FR-041).
func TestBreakdown_NoPlannedCostNote_FR041(t *testing.T) {
	b := mustBreakdown(t, twoYears(), Query{By: "attr:SA-S"})

	if !slices.Equal(b.Notes, []string{"Includes 3,000.00 USD on line items of activities without a planned cost."}) ||
		b.Total.PlannedCost.Format() != "27,000.00 USD" {
		t.Errorf("notes %q, total %s; want the 3,000.00 USD note and 27,000.00 USD", b.Notes, b.Total.PlannedCost.Format())
	}
}

// A value hidden by its attribute's Business unit condition is Not filled in.
func TestBreakdown_HiddenIsNotFilled(t *testing.T) {
	nodes := withAttr(withAttr(twoYears(), "A-1", "SA-M", options("SO-X")), "A-3", "SA-M", options("SO-X"))

	b := mustBreakdown(t, nodes, Query{By: "attr:SA-M"})

	assertGroups(t, "Mobility channel", b.Groups, []string{
		"Fleet 1/0 8,000.00 USD 33.33%",
		"Not filled in 3/3 16,000.00 USD 66.67%",
	})
}

// "Then by" groups each group again by a second grouping; it must differ from the first, and the first is
// needed (FR-040, FR-044).
func TestBreakdown_ThenBy_FR040(t *testing.T) {
	b := mustBreakdown(t, twoYears(), Query{By: "plan", ThenBy: "attr:SA-B"})
	_, same := breakdownView(twoYears(), Query{By: "plan", ThenBy: "plan"}, 2026, 2027)
	_, missing := breakdownView(twoYears(), Query{ThenBy: "nothing"}, 2026, 2027)

	if len(b.Groups) != 2 || b.ThenBy == nil || b.ThenBy.Label != "Brands" {
		t.Fatalf("groups = %q, then by %+v", groupsText(b.Groups), b.ThenBy)
	}
	if got := childrenText(b.Groups[0]); !slices.Equal(got, []string{"Helix 2/2 16,000.00 USD 66.67%", "Rimula 1/0 6,000.00 USD 25.00%"}) {
		t.Errorf("FY2026 Lubricants by brand = %q", got)
	}
	if got := childrenText(b.Groups[1]); !slices.Equal(got, []string{"Not filled in 2/1 8,000.00 USD 33.33%"}) {
		t.Errorf("FY2027 UK by brand = %q", got)
	}
	assertProblems(t, same, []errs.FieldError{{Field: "thenBy", Message: "Choose another grouping than the first."}})
	assertProblems(t, missing, []errs.FieldError{
		{Field: "by", Message: "Choose what to break down by."},
		{Field: "thenBy", Message: "Choose one of the listed values."},
	})
}

// Groupings of plans, campaigns and initiatives show the budgets of their records and the budget used; within a
// "then by" group only the records of the group's branch count; groupings of activities have no budget (FR-040).
func TestBreakdown_BudgetGroupings_FR040(t *testing.T) {
	byPlanThenCluster := mustBreakdown(t, twoYears(), Query{By: "plan", ThenBy: "attr:SA-CL"})
	byType := mustBreakdown(t, twoYears(), Query{By: "activityType"})

	assertGroups(t, "Plan", byPlanThenCluster.Groups, []string{
		"FY2026 Lubricants 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)",
		"FY2027 UK 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)",
	})
	if got := childrenText(byPlanThenCluster.Groups[0]); !slices.Equal(got, []string{"Europe 2/2 16,000.00 USD 66.67% of 60,000.00 USD (27 %)"}) {
		t.Errorf("FY2026 Lubricants by cluster = %q", got)
	}
	total := byPlanThenCluster.Total
	if total.Budget == nil || total.Budget.Format() != "150,000.00 USD" || byType.Total.Budget != nil || byType.Groups[0].Budget != nil {
		t.Errorf("plan total budget %v; activity type budgets %v / %v, want none", total.Budget, byType.Total.Budget, byType.Groups[0].Budget)
	}
}

// The built-in groupings: Plan type, Plan currency, Year, Team, Business unit, Activity type and Owner, with
// their labels (FR-040).
func TestBreakdown_BuiltIns_FR040(t *testing.T) {
	cases := map[string][]string{
		"planType":     {"Cross-market 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)", "Operational 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)"},
		"planCurrency": {"GBP 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)", "USD 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)"},
		"year":         {"2026 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)", "2027 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)"},
		"team":         {"Lubricants Europe 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)", "Mobility China 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)"},
		"businessUnit": {"SO-LUB 2/2 16,000.00 USD 66.67% of 100,000.00 USD (16 %)", "SO-MOB 2/1 8,000.00 USD 33.33% of 50,000.00 USD (16 %)"},
		"activityType": {"Content production 2/2 18,000.00 USD 75.00%", "Digital media 2/1 6,000.00 USD 25.00%"},
		"owner":        {"Daniel Okoro 1/0 6,000.00 USD 25.00%", "Priya Raman 3/3 18,000.00 USD 75.00%"},
	}
	for by, want := range cases {
		t.Run(by, func(t *testing.T) {
			assertGroups(t, by, mustBreakdown(t, twoYears(), Query{By: by}).Groups, want)
		})
	}
	b := mustBreakdown(t, twoYears(), Query{By: "plan"})
	var keys []string
	for _, g := range b.Groupings {
		keys = append(keys, g.Key)
	}
	want := []string{"plan", "planType", "planCurrency", "year", "team", "businessUnit", "activityType", "owner",
		"attr:SA-CL", "attr:SA-B", "attr:SA-C", "attr:SA-F", "attr:SA-M", "attr:SA-S"}
	if !slices.Equal(keys, want) {
		t.Errorf("groupings = %q, want %q (no number attribute)", keys, want)
	}
}

// The totals row equals the Planned cost card of the same filters (SC-007), whatever the grouping.
func TestBreakdown_TotalsEqualCard_SC007(t *testing.T) {
	card := rangeView(twoYears(), Query{View: ViewGrid}, 2026, 2027).result.Summary.PlannedCost
	filtered := rangeView(twoYears(), Query{View: ViewGrid, Attributes: []AttrFilter{{AttributeID: "SA-CL", Values: []string{"SO-EU"}}}}, 2026, 2027).result.Summary.PlannedCost

	for _, by := range []string{"plan", "attr:SA-B", "attr:SA-C", "owner", "attr:SA-F"} {
		b := mustBreakdown(t, twoYears(), Query{By: by})
		if card == nil || b.Total.PlannedCost != *card || b.Total.Activities != 4 {
			t.Errorf("by %s: total %s for %d activities, card %v", by, b.Total.PlannedCost.Format(), b.Total.Activities, card)
		}
	}
	europe := mustBreakdown(t, twoYears(), Query{By: "attr:SA-B", Attributes: []AttrFilter{{AttributeID: "SA-CL", Values: []string{"SO-EU"}}}})
	if filtered == nil || europe.Total.PlannedCost != *filtered || europe.Total.PlannedCost.Format() != "16,000.00 USD" {
		t.Errorf("Europe: total %s, card %v", europe.Total.PlannedCost.Format(), filtered)
	}
}

// For one plan, amountsIn=plan gives the Breakdown in the plan currency — the plan page's Planned cost by card
// (FR-042); with several plans it is refused.
func TestBreakdown_PlanCurrencyForOnePlan_FR042(t *testing.T) {
	nodes := twoYears()
	for i := range nodes {
		switch nodes[i].ID {
		case "P-2001":
			nodes[i].BudgetPlan = &money.Money{Amount: amountOf("39500.00"), Currency: "GBP"}
		case "A-3":
			nodes[i].PlannedCostPlan = &money.Money{Amount: amountOf("6320.00"), Currency: "GBP"}
		case "L-3":
			nodes[i].PlannedCostPlan = &money.Money{Amount: amountOf("2370.00"), Currency: "GBP"}
		}
	}

	b := mustBreakdown(t, nodes, Query{By: "planType", Plans: []string{"P-2001"}, AmountsIn: AmountsInPlan})
	bySpend := mustBreakdown(t, nodes, Query{By: "attr:SA-S", Plans: []string{"P-2001"}, AmountsIn: AmountsInPlan})
	_, several := breakdownView(nodes, Query{By: "planType", AmountsIn: AmountsInPlan}, 2026, 2027)

	if b.Currency != "GBP" {
		t.Errorf("currency = %s, want GBP", b.Currency)
	}
	assertGroups(t, "Plan type in GBP", b.Groups, []string{"Cross-market 2/1 6,320.00 GBP 100.00% of 39,500.00 GBP (16 %)"})
	assertGroups(t, "Spend type in GBP", bySpend.Groups, []string{
		"Media 1/1 2,370.00 GBP 27.27%",
		"Not on a line item 1/0 6,320.00 GBP 72.73%",
	})
	assertProblems(t, several, []errs.FieldError{{Field: "amountsIn", Message: "Choose one plan to show amounts in its currency."}})
}
