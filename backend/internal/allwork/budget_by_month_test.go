package allwork

import (
	"slices"
	"testing"

	"mapp/internal/platform/money"
)

// The Budget by month tests (016 FR-024, data-model §3.4) build one GBP plan of 2027 by hand: P-1 with a
// 120,000.00 GBP budget phased evenly (10,000.00 a month), campaign C-1 (60,000.00 GBP, custom: 30,000.00 in
// March and in June) with initiatives I-1 (phased evenly over its 12,000.00 GBP) and I-2 (not phased).

// gbp is an amount in GBP of the tests.
func gbp(text string) *money.Money {
	return &money.Money{Amount: amountOf(text), Currency: "GBP"}
}

// phasedEvenly returns the 12 evenly phased months of budget.
func phasedEvenly(budget string) []money.Amount {
	return amountOf(budget).SplitEvenly(12)
}

// phasingNodes are the plan, its campaign and initiatives, and the activities of each test.
func phasingNodes(activities ...Node) []Node {
	custom := make([]money.Amount, 12)
	custom[2], custom[5] = amountOf("30000.00"), amountOf("30000.00")
	nodes := []Node{
		{Level: LevelPlan, ID: "P-1", PlanID: "P-1", Name: "FY2027 UK", PlanYear: 2027, PlanCurrency: "GBP",
			BudgetPlan: gbp("120000.00"), PhasingMethod: "evenly", PhasedPlan: phasedEvenly("120000.00")},
		{Level: LevelCampaign, ID: "C-1", ParentID: "P-1", PlanID: "P-1", Name: "Helix", PlanCurrency: "GBP",
			BudgetPlan: gbp("60000.00"), PhasingMethod: "custom", PhasedPlan: custom},
		{Level: LevelInitiative, ID: "I-1", ParentID: "C-1", PlanID: "P-1", Name: "UK", Order: 0, PlanCurrency: "GBP",
			BudgetPlan: gbp("12000.00"), PhasingMethod: "evenly", PhasedPlan: phasedEvenly("12000.00")},
		{Level: LevelInitiative, ID: "I-2", ParentID: "C-1", PlanID: "P-1", Name: "Ireland", Order: 1, PlanCurrency: "GBP",
			BudgetPlan: gbp("8000.00"), PhasingMethod: "none"},
	}
	return append(nodes, activities...)
}

// activityNode is an activity of I-1 with a planned cost in GBP over months.
func activityNode(id, planned string, months ...string) Node {
	n := Node{Level: LevelActivity, ID: id, ParentID: "I-1", PlanID: "P-1", Name: id, PlanCurrency: "GBP", Months: months}
	if planned != "" {
		n.PlannedCostPlan = gbp(planned)
	}
	return n
}

// lineItemNode is a line item of an activity with its months in the plan currency.
func lineItemNode(id, activityID, total string, unspread string, months ...string) Node {
	n := Node{Level: LevelLineItem, ID: id, ParentID: activityID, PlanID: "P-1", Name: id, PlannedCostPlan: gbp(total)}
	for i := 0; i+1 < len(months); i += 2 {
		n.MonthsPlan = append(n.MonthsPlan, PlanMonth{Month: months[i], Amount: amountOf(months[i+1])})
	}
	if unspread != "" {
		n.UnspreadPlan = amountOf(unspread)
	}
	return n
}

// phasedRowOf returns the Budget by month row of the record id.
func phasedRowOf(t *testing.T, b BudgetByMonth, id string) BudgetByMonthRow {
	t.Helper()
	for _, r := range b.Rows {
		if r.RecordID == id {
			return r
		}
	}
	t.Fatalf("no row %s in %+v", id, b.Rows)
	return BudgetByMonthRow{}
}

// texts writes amounts as "1,000.00", "—" for none.
func texts(amounts []*money.Money) []string {
	list := []string{}
	for _, a := range amounts {
		if a == nil {
			list = append(list, "—")
		} else {
			list = append(list, a.Amount.Format())
		}
	}
	return list
}

// plainTexts writes amounts as "1,000.00".
func plainTexts(amounts []money.Money) []string {
	list := []string{}
	for _, a := range amounts {
		list = append(list, a.Amount.Format())
	}
	return list
}

// The planned cost by month is in the plan currency: line items by their months (already converted with
// ConvertParts by the source), a positive unassigned balance spread evenly over its activity's months, a
// negative one never; an activity without dates and amounts that cannot be spread go to No dates, months of
// other years to Other years (FR-024, data-model §3.4).
func TestPlannedByMonth_PlanCurrency_FR024(t *testing.T) {
	t.Parallel()
	nodes := phasingNodes(
		// 9,000.00 planned, 6,000.00 on a line item over Mar–May: 3,000.00 left, 1,000.00 a month.
		activityNode("A-1", "9000.00", "2027-03", "2027-04", "2027-05"),
		lineItemNode("L-1", "A-1", "6000.00", "", "2027-03", "2000.00", "2027-04", "2000.00", "2027-05", "2000.00"),
		// 1,000.00 planned, 1,500.00 on a line item (one month in 2028): negative balance, never spread.
		activityNode("A-2", "1000.00", "2027-12"),
		lineItemNode("L-2", "A-2", "1500.00", "", "2027-12", "1000.00", "2028-01", "500.00"),
		// No dates: 700.00 planned, 200.00 on a line item that cannot be spread → 700.00 without dates.
		activityNode("A-3", "700.00"),
		lineItemNode("L-3", "A-3", "200.00", "200.00"),
	)

	result := budgetByMonth(buildTree(nodes), 2027)

	plan := phasedRowOf(t, result, "P-1")
	want := []string{"0.00", "0.00", "3,000.00", "3,000.00", "3,000.00", "0.00", "0.00", "0.00", "0.00", "0.00", "0.00", "1,000.00"}
	if got := plainTexts(plan.Months.PlannedCost); !slices.Equal(got, want) {
		t.Errorf("planned by month = %q, want %q", got, want)
	}
	if plan.OtherYears == nil || plan.OtherYears.Format() != "500.00 GBP" || plan.NoDates == nil || plan.NoDates.Format() != "700.00 GBP" {
		t.Errorf("other years %v, no dates %v; want 500.00 GBP and 700.00 GBP", plan.OtherYears, plan.NoDates)
	}
	if result.Currency != "GBP" || result.Year != 2027 || len(result.Months) != 12 || result.Months[0] != "2027-01" {
		t.Errorf("result = %s %d %v", result.Currency, result.Year, result.Months)
	}
}

// The rows: Budget (phased, "—" when not phased), Planned cost, Difference (only when phased), Allocated to the
// children (plans and campaigns; null for initiatives) and the over flags; "2 campaigns are not phased" is
// the count of children without phasing (FR-024).
func TestBudgetByMonth_Rows_FR024(t *testing.T) {
	t.Parallel()
	nodes := phasingNodes(
		activityNode("A-1", "15000.00", "2027-03"), // 15,000.00 in March in I-1
	)

	result := budgetByMonth(buildTree(nodes), 2027)

	ids := []string{}
	for _, r := range result.Rows {
		ids = append(ids, r.RecordID)
	}
	if !slices.Equal(ids, []string{"P-1", "C-1", "I-1", "I-2"}) {
		t.Fatalf("rows = %v, want the plan, then the campaign followed by its initiatives", ids)
	}
	plan, campaign, uk, ireland := result.Rows[0], result.Rows[1], result.Rows[2], result.Rows[3]
	if plan.Method != "evenly" || texts(plan.Months.Budget)[0] != "10,000.00" || *plan.Months.Difference[2] != *gbp("-5000.00") ||
		!plan.Months.Over[2] || plan.Months.Over[0] {
		t.Errorf("plan = %+v", plan.Months)
	}
	// The campaign's phased budget allocates 30,000.00 in March and June, above the plan's 10,000.00.
	if plan.Months.Allocated == nil || texts(*plan.Months.Allocated)[2] != "30,000.00" || !plan.Months.Over[5] || plan.ChildrenNotPhased != 0 {
		t.Errorf("plan allocated = %v, over %v", plan.Months.Allocated, plan.Months.Over)
	}
	// Of the campaign's initiatives, UK is phased (1,000.00 a month) and Ireland is not: "1 initiative is not phased".
	if campaign.Method != "custom" || campaign.ChildrenNotPhased != 1 || texts(*campaign.Months.Allocated)[0] != "1,000.00" ||
		!campaign.Months.Over[0] || campaign.Months.Over[2] {
		t.Errorf("campaign = %+v (allocated %v)", campaign.Months, texts(*campaign.Months.Allocated))
	}
	if uk.Months.Allocated != nil || uk.ParentID == nil || *uk.ParentID != "C-1" || !uk.Months.Over[2] {
		t.Errorf("UK = %+v", uk)
	}
	if ireland.Method != "none" || texts(ireland.Months.Budget)[0] != "—" || ireland.Months.Difference[0] != nil || slices.Contains(ireland.Months.Over, true) {
		t.Errorf("Ireland, not phased = %+v", ireland.Months)
	}
	// Quarterly adds three months: Q1 budget 30,000.00, planned 15,000.00, allocated 30,000.00 — not over.
	if q := plan.Quarters; len(q.Budget) != 4 || q.Budget[0].Amount.Format() != "30,000.00" || q.PlannedCost[0].Amount.Format() != "15,000.00" ||
		q.Over[0] || (*q.Allocated)[1].Amount.Format() != "30,000.00" {
		t.Errorf("plan quarters = %+v", q)
	}
}

// The planned cost by month, with Other years and No dates, adds up to the plan's planned cost (SC-004).
func TestBudgetByMonth_EqualsBudgetAndCost_SC004(t *testing.T) {
	t.Parallel()
	nodes := phasingNodes(
		activityNode("A-1", "9000.00", "2027-03", "2027-04", "2027-05"),
		lineItemNode("L-1", "A-1", "6000.00", "", "2027-03", "2000.00", "2027-04", "2000.00", "2027-05", "2000.00"),
		activityNode("A-3", "700.00"),
	)

	plan := phasedRowOf(t, budgetByMonth(buildTree(nodes), 2027), "P-1")

	var total money.Amount
	for _, m := range plan.Months.PlannedCost {
		total = total.Add(m.Amount)
	}
	for _, extra := range []*money.Money{plan.OtherYears, plan.NoDates} {
		if extra != nil {
			total = total.Add(extra.Amount)
		}
	}
	if total.Format() != "9,700.00" {
		t.Errorf("months + other years + no dates = %s, want the planned cost 9,700.00", total.Format())
	}
}
