package dashboard

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
)

func amount(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return a
}

func ptr(t *testing.T, text string) *money.Amount {
	t.Helper()
	a := amount(t, text)
	return &a
}

func facts(t *testing.T, committed, paid string) *allwork.FinanceFacts {
	t.Helper()
	return &allwork.FinanceFacts{Committed: amount(t, committed), Paid: amount(t, paid)}
}

// sampleNodes are two plans of 2026: Beta (teams T-1 and T-2, budget 1,000.00, its finance visible) with a campaign
// over budget and one under, and Alpha (team T-2, no budget, finance not visible).
func sampleNodes(t *testing.T) []allwork.Node {
	t.Helper()
	return []allwork.Node{
		{Level: allwork.LevelPlan, ID: "P-1", PlanID: "P-1", Name: "Beta plan", TeamIDs: []string{"T-1", "T-2"},
			Budget: ptr(t, "1000.00"), PlanType: "operational", Status: "published"},
		{Level: allwork.LevelCampaign, ID: "C-1", ParentID: "P-1", PlanID: "P-1", Name: "Over", Budget: ptr(t, "600.00")},
		{Level: allwork.LevelInitiative, ID: "I-1", ParentID: "C-1", PlanID: "P-1"},
		{Level: allwork.LevelActivity, ID: "A-1", ParentID: "I-1", PlanID: "P-1", PlannedCost: ptr(t, "700.00"), Status: "approved"},
		{Level: allwork.LevelActivity, ID: "A-2", ParentID: "I-1", PlanID: "P-1", Status: "created", Missing: 1},
		{Level: allwork.LevelLineItem, ID: "L-1", ParentID: "A-1", PlanID: "P-1", Finance: facts(t, "300.00", "200.00")},
		{Level: allwork.LevelLineItem, ID: "L-2", ParentID: "A-1", PlanID: "P-1"},
		{Level: allwork.LevelCampaign, ID: "C-2", ParentID: "P-1", PlanID: "P-1", Name: "Under", Budget: ptr(t, "500.00"), Order: 1},
		{Level: allwork.LevelInitiative, ID: "I-2", ParentID: "C-2", PlanID: "P-1"},
		{Level: allwork.LevelActivity, ID: "A-3", ParentID: "I-2", PlanID: "P-1", PlannedCost: ptr(t, "100.00"), Status: "submitted"},
		{Level: allwork.LevelPlan, ID: "P-2", PlanID: "P-2", Name: "Alpha plan", TeamIDs: []string{"T-2"}, PlanType: "strategic",
			Status: "draft", Missing: 2},
		{Level: allwork.LevelCampaign, ID: "C-3", ParentID: "P-2", PlanID: "P-2", Name: "No budget"},
		{Level: allwork.LevelInitiative, ID: "I-3", ParentID: "C-3", PlanID: "P-2"},
		{Level: allwork.LevelActivity, ID: "A-4", ParentID: "I-3", PlanID: "P-2", PlannedCost: ptr(t, "50.00"), Status: "submitted", Missing: 2},
		{Level: allwork.LevelLineItem, ID: "L-3", ParentID: "A-4", PlanID: "P-2", Finance: facts(t, "10.00", "5.00")},
	}
}

func samplePaid(t *testing.T) map[string]map[string]money.Amount {
	t.Helper()
	return map[string]map[string]money.Amount{
		"L-1": {"2026-03": amount(t, "150.00"), "2025-12": amount(t, "50.00")},
		"L-3": {"2026-03": amount(t, "5.00")},
	}
}

func sampleBoard(t *testing.T, keep func(*allwork.Node) bool) *board {
	t.Helper()
	if keep == nil {
		keep = func(*allwork.Node) bool { return true }
	}
	b := newBoard(sampleNodes(t), keep, true, []string{"P-1"})
	b.countPaid(samplePaid(t), 2026)
	return b
}

func text(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return m.Amount.String()
}

func percent(p *int) int {
	if p == nil {
		return -1
	}
	return *p
}

func texts(values ...any) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = fmt.Sprint(v)
	}
	return out
}

func same(t *testing.T, what string, want, got []string) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("%s mismatch (-want +got):\n%s", what, diff)
	}
}

// Budget and planned cost add up every plan; committed and actual only the plans whose finance is visible; the
// percents and differences follow from them; a plan without a budget adds nothing to the budget (FR-008 – FR-011).
func TestFigures_FR008_FR011(t *testing.T) {
	t.Parallel()
	b := sampleBoard(t, nil)
	total := b.total().figures(money.DisplayUSD)

	alpha := b.own["P-2"].figures(money.DisplayIn("EUR", "0.92"))
	over := b.own["C-1"].figures(money.DisplayUSD)

	same(t, "totals", texts("1000.00", "850.00", "300.00", "150.00", "150.00", "850.00", 85, 15, false),
		texts(text(total.Budget), text(total.PlannedCost), text(total.Committed), text(total.Actual), text(total.LeftToPlan),
			text(total.UnspentBudget), percent(total.BudgetUsedPercent), percent(total.BudgetSpentPercent), total.OverBudget))
	same(t, "Alpha in euros", texts("—", "—", -1, "—", "46.00", "EUR"), texts(text(alpha.Budget), text(alpha.Committed),
		percent(alpha.BudgetUsedPercent), text(alpha.LeftToPlan), text(alpha.PlannedCost), alpha.PlannedCost.Currency))
	same(t, "campaign over budget", texts(true, "-100.00"), texts(over.OverBudget, text(over.LeftToPlan)))
}

// Actual is what was paid in the plan's year (013's Actual spend): L-1's payment of 2025 counts nowhere (D-357).
func TestActual_PaidInThePlansYearOnly_D357(t *testing.T) {
	t.Parallel()
	b := sampleBoard(t, nil)

	activity := b.activityFinance()["A-1"]
	same(t, "actuals", texts("150.00", "150.00", "150.00", "—"), texts(text(usd(b.own["P-1"].actual)),
		text(usd(b.own["C-1"].actual)), text(usd(activity.actual)), text(usd(b.own["P-2"].actual))))
}

func TestPlans_ByNameWithTheirFinance_FR013(t *testing.T) {
	t.Parallel()
	rows := sampleBoard(t, nil).planRows(nil, money.DisplayUSD)

	if len(rows) != 2 || rows[0].Plan.Name != "Alpha plan" || rows[0].FinanceShown || !rows[1].FinanceShown ||
		len(rows[1].Teams) != 2 || rows[1].Plan.PlanType != "operational" {
		t.Errorf("rows = %+v", rows)
	}
}

// A plan of two teams counts in both teams' rows; each row adds up only its own plans (FR-012).
func TestMarkets_PlanInTwoTeams_FR012(t *testing.T) {
	t.Parallel()
	rows := sampleBoard(t, nil).markets(nil, money.DisplayUSD)

	if len(rows) != 2 || rows[0].Team.ID != "T-1" || rows[0].PlanCount != 1 || text(rows[0].Figures.PlannedCost) != "800.00" ||
		rows[1].PlanCount != 2 || text(rows[1].Figures.PlannedCost) != "850.00" {
		t.Errorf("markets = %+v", rows)
	}
}

// Campaigns over budget and with budget left to plan, largest first, and none without a budget (FR-015).
func TestCampaignLists_FR015(t *testing.T) {
	t.Parallel()
	over, left := sampleBoard(t, nil).campaignLists(money.DisplayUSD)

	if len(over) != 1 || over[0].Campaign.ID != "C-1" || text(over[0].Difference) != "100.00" || over[0].Plan.Name != "Beta plan" {
		t.Errorf("over budget = %+v", over)
	}
	if len(left) != 1 || left[0].Campaign.ID != "C-2" || text(left[0].Difference) != "400.00" {
		t.Errorf("left to plan = %+v", left)
	}
}

// A plan's campaigns in its order; a plan not on the board has none (FR-021).
func TestPlanCampaigns_FR021(t *testing.T) {
	t.Parallel()
	b := sampleBoard(t, func(p *allwork.Node) bool { return p.ID == "P-1" })

	got := b.planCampaigns("P-1", money.DisplayUSD)
	if got == nil || got.Plan.Name != "Beta plan" || len(got.Items) != 2 || got.Items[1].Campaign.Name != "Under" {
		t.Errorf("campaigns of P-1 = %+v", got)
	}
	if other := b.planCampaigns("P-2", money.DisplayUSD); other != nil {
		t.Errorf("a plan not on the board shows %+v, want nothing", other)
	}
}

// Each level's records with every needed field filled in, and their share (FR-018).
func TestCompleteness_FR018(t *testing.T) {
	t.Parallel()
	got := sampleBoard(t, nil).completeness()

	want := map[string][3]int{"plan": {2, 1, 50}, "campaign": {3, 3, 100}, "initiative": {3, 3, 100}, "activity": {4, 2, 50},
		"lineItem": {3, 3, 100}}
	for _, c := range got {
		if w := want[c.Level]; c.Total != w[0] || c.Complete != w[1] || percent(c.Percent) != w[2] {
			t.Errorf("%s = %d of %d (%d%%), want %v", c.Level, c.Complete, c.Total, percent(c.Percent), w)
		}
	}
}
