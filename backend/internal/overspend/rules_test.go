package overspend

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/money"
)

func usd(t *testing.T, text string) money.Money {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatal(err)
	}
	return money.USD(amount)
}

func budgetOf(t *testing.T, mode Mode, amount, planned string) Budget {
	t.Helper()
	return Budget{Kind: KindCampaign, ID: "C-1001", Name: "Helix launch", Mode: mode, Amount: usd(t, amount),
		Planned: usd(t, planned)}
}

func TestOverages_RefuseRule_037_FR009_FR011(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		before, after []Budget
		refused       bool
	}{
		{name: "FR-009 a raise over a Refuse budget is refused", before: []Budget{budgetOf(t, Refuse, "700000.00", "690000.00")},
			after: []Budget{budgetOf(t, Refuse, "700000.00", "712000.00")}, refused: true},
		{name: "FR-009 exactly at the budget is within it", before: []Budget{budgetOf(t, Refuse, "700000.00", "690000.00")},
			after: []Budget{budgetOf(t, Refuse, "700000.00", "700000.00")}},
		{name: "FR-009 one cent over is refused", before: []Budget{budgetOf(t, Refuse, "700000.00", "690000.00")},
			after: []Budget{budgetOf(t, Refuse, "700000.00", "700000.01")}, refused: true},
		{name: "FR-001 Warn never refuses", before: []Budget{budgetOf(t, Warn, "700000.00", "690000.00")},
			after: []Budget{budgetOf(t, Warn, "700000.00", "990000.00")}},
		{name: "FR-009 lowering a Refuse budget below its planned cost is refused",
			before: []Budget{budgetOf(t, Refuse, "700000.00", "650000.00")},
			after:  []Budget{budgetOf(t, Refuse, "600000.00", "650000.00")}, refused: true},
		{name: "FR-009 a budget typed below the planned cost is refused", after: []Budget{budgetOf(t, Refuse, "600000.00", "650000.00")},
			refused: true},
		{name: "FR-009 a budget of 0.00 refuses any planned cost", after: []Budget{budgetOf(t, Refuse, "0.00", "0.01")}, refused: true},
		{name: "FR-011 choosing Refuse while already over is allowed", before: []Budget{budgetOf(t, Warn, "700000.00", "750000.00")},
			after: []Budget{budgetOf(t, Refuse, "700000.00", "750000.00")}},
		{name: "FR-011 lowering the cost of a budget still over is allowed",
			before: []Budget{budgetOf(t, Refuse, "700000.00", "750000.00")},
			after:  []Budget{budgetOf(t, Refuse, "700000.00", "720000.00")}},
		{name: "FR-011 raising the amount of a budget still over is allowed",
			before: []Budget{budgetOf(t, Refuse, "700000.00", "750000.00")},
			after:  []Budget{budgetOf(t, Refuse, "710000.00", "750000.00")}},
		{name: "FR-009 any raise of a budget already over is refused",
			before: []Budget{budgetOf(t, Refuse, "700000.00", "750000.00")},
			after:  []Budget{budgetOf(t, Refuse, "700000.00", "750000.01")}, refused: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Overages(tt.before, tt.after)
			if (len(got) > 0) != tt.refused {
				t.Errorf("Overages = %+v, refused want %v", got, tt.refused)
			}
		})
	}
}

func TestRefusal_NamesEveryBudgetInOrder_037_FR010(t *testing.T) {
	t.Parallel()
	plan := Budget{Kind: KindPlan, ID: "P-1001", Name: "FY2026 Lubricants Europe", Mode: Refuse,
		Amount: usd(t, "1200000.00"), Planned: usd(t, "1250000.00")}
	global := Budget{Kind: KindGlobalBudget, ID: "GB-1001", Name: "FY2026 Global", Mode: Refuse,
		Amount: usd(t, "2000000.00"), Planned: usd(t, "2000000.50")}
	campaign := budgetOf(t, Refuse, "700000.00", "712000.00")

	one := Refusal(Overages(nil, []Budget{plan}))
	several := Refusal(Overages(nil, []Budget{global, campaign, plan}))

	if one.Status != http.StatusConflict || one.Code != CodeOverBudget || one.Message != "Not saved — the planned cost of plan "+
		"'FY2026 Lubricants Europe' would be 1,250,000.00 USD, over its budget of 1,200,000.00 USD by 50,000.00 USD. "+
		"Its owner refuses overspending." {
		t.Errorf("one = %d %s %q", one.Status, one.Code, one.Message)
	}
	want := "Not saved — the planned cost of plan 'FY2026 Lubricants Europe' would be 1,250,000.00 USD, over its budget of " +
		"1,200,000.00 USD by 50,000.00 USD; the planned cost of campaign 'Helix launch' would be 712,000.00 USD, over its " +
		"budget of 700,000.00 USD by 12,000.00 USD; the planned cost of global budget 'FY2026 Global' would be " +
		"2,000,000.50 USD, over its budget of 2,000,000.00 USD by 0.50 USD. Their owners refuse overspending."
	if diff := cmp.Diff(want, several.Message); diff != "" {
		t.Errorf("several (-want +got):\n%s", diff)
	}
}

func TestLimitsOf_BudgetsAboveTheActivity_037_FR014(t *testing.T) {
	t.Parallel()
	eur := func(text string) money.Money {
		m := usd(t, text)
		m.Currency = "EUR"
		return m
	}
	budgets := []Budget{
		{Kind: KindPlan, ID: "P-1", Name: "Plan", Mode: Refuse, Amount: eur("1000.00"), Planned: eur("900.00")},
		{Kind: KindCampaign, ID: "C-1", Name: "Campaign", Mode: Warn, Amount: eur("500.00"), Planned: eur("100.00")},
		{Kind: KindInitiative, ID: "I-2", Name: "Other initiative", Mode: Refuse, Amount: eur("10.00"), Planned: eur("0.00")},
		{Kind: KindInitiative, ID: "I-1", Name: "Initiative", Mode: Refuse, Amount: eur("50.00"), Planned: eur("60.00")},
		{Kind: KindGlobalBudget, ID: "GB-1", Name: "Global", Mode: Refuse, Amount: usd(t, "200.00"), Planned: usd(t, "100.00"),
			PlanIDs: []string{"P-1"}},
	}
	rates := money.Rates{"EUR": money.RateFromMicros(920000)}

	limits, err := LimitsOf(budgets, Path{PlanID: "P-1", CampaignID: "C-1", InitiativeID: "I-1", PlanCurrency: "EUR"}, rates)

	got := []string{}
	for _, l := range limits {
		got = append(got, l.Budget.Type+" "+l.Budget.Name+" "+l.LeftToPlan.Format())
	}
	want := []string{"plan Plan 100.00 EUR", "initiative Initiative -10.00 EUR", "globalBudget Global 92.00 EUR"}
	if diff := cmp.Diff(want, got); err != nil || diff != "" {
		t.Errorf("LimitsOf (-want +got):\n%s %v", diff, err)
	}
}

func TestParseMode(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]bool{"warn": true, "refuse": true, "": false, "Refuse": false, "block": false} {
		if _, ok := ParseMode(text); ok != want {
			t.Errorf("ParseMode(%q) ok = %v, want %v", text, ok, want)
		}
	}
}
