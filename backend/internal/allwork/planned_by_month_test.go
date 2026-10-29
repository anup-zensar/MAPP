package allwork_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
)

func usdOf(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return a
}

// Line-item spreads and the positive unassigned planned cost spread evenly by month; a negative balance is not
// spread; months of other years and amounts without dates are kept apart (010 FR-023, 036 FR-014).
func TestPlannedByActivity_010_FR023(t *testing.T) {
	t.Parallel()
	amount := func(text string) *money.Amount { a := usdOf(t, text); return &a }
	nodes := []allwork.Node{
		{Level: allwork.LevelPlan, ID: "P-1", PlanID: "P-1"},
		{Level: allwork.LevelCampaign, ID: "C-1", ParentID: "P-1", PlanID: "P-1"},
		{Level: allwork.LevelInitiative, ID: "I-1", ParentID: "C-1", PlanID: "P-1"},
		{Level: allwork.LevelActivity, ID: "A-1", ParentID: "I-1", PlanID: "P-1", PlannedCost: amount("1000.00"),
			Months: []string{"2026-11", "2026-12", "2027-01"}},
		{Level: allwork.LevelLineItem, ID: "L-1", ParentID: "A-1", PlanID: "P-1", PlannedCost: amount("300.00"),
			MonthsUSD: []allwork.MonthAmount{{Month: "2026-11", USD: usdOf(t, "100.00")}, {Month: "2027-01", USD: usdOf(t, "200.00")}}},
		{Level: allwork.LevelActivity, ID: "A-2", ParentID: "I-1", PlanID: "P-1", PlannedCost: amount("50.00")},
		{Level: allwork.LevelLineItem, ID: "L-2", ParentID: "A-2", PlanID: "P-1", PlannedCost: amount("80.00"),
			UnspreadUSD: usdOf(t, "80.00")},
		{Level: allwork.LevelActivity, ID: "A-3", ParentID: "I-1", PlanID: "P-1", PlannedCost: amount("40.00")},
	}

	got := allwork.PlannedByActivity(nodes, 2026)

	want := map[string]allwork.Months{
		"A-1": {ByMonth: map[string]money.Amount{"2026-11": usdOf(t, "333.33"), "2026-12": usdOf(t, "233.33")},
			OtherYears: usdOf(t, "433.34")},
		"A-2": {ByMonth: map[string]money.Amount{}, NoDates: usdOf(t, "80.00")},
		"A-3": {ByMonth: map[string]money.Amount{}, NoDates: usdOf(t, "40.00")},
	}
	if diff := cmp.Diff(want, got, cmp.Comparer(func(x, y money.Amount) bool { return x.Cmp(y) == 0 })); diff != "" {
		t.Errorf("PlannedByActivity mismatch (-want +got):\n%s", diff)
	}
}
