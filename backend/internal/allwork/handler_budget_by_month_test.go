package allwork_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/allwork"
	"mapp/internal/platform/apitest"
)

// Budget by month of the demo plan P-1004 "FY2027 Lubricants UK & Ireland" (016 data-model §8): 600,000.00
// GBP phased evenly; campaign C-1005 "Helix UK & Ireland" 400,000.00 GBP phased 100,000.00 in March, June,
// September and November; its initiatives UK and Ireland not phased. UK launch event (Mar – May) costs
// 40,000.00 GBP — Venue 15,000.00 GBP and Agency fee 23,000.00 EUR = 19,750.00 GBP spread evenly, 5,250.00
// GBP unassigned spread evenly — and Dealer conference 25,000.00 GBP in September (FR-024).
func TestGetBudgetByMonth_Contract_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1004/budget-by-month", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/plans/P-9999/budget-by-month", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s", response.Status, response.Body)
	}
	var result allwork.BudgetByMonth
	response.Decode(t, &result)
	if result.PlanID != "P-1004" || result.Year != 2027 || result.Currency != "GBP" || len(result.Rows) != 4 {
		t.Fatalf("result = %s", response.Body)
	}
	// The plan tree's order: the plan, the campaign, then its initiatives by start date and name — Ireland, UK.
	plan, campaign, ireland, uk := result.Rows[0], result.Rows[1], result.Rows[2], result.Rows[3]
	wantPlanned := []string{"0.00", "0.00", "13,333.33", "13,333.33", "13,333.34", "0.00", "0.00", "0.00", "25,000.00", "0.00", "0.00", "0.00"}
	var planned []string
	for _, m := range plan.Months.PlannedCost {
		planned = append(planned, m.Amount.Format())
	}
	if !slices.Equal(planned, wantPlanned) || plan.Method != "evenly" || plan.Months.Budget[0].Format() != "50,000.00 GBP" {
		t.Errorf("plan planned = %q, method %s, budget %v", planned, plan.Method, plan.Months.Budget[0])
	}
	wantOver := []bool{false, false, true, false, false, true, false, false, true, false, true, false}
	if plan.Months.Allocated == nil || (*plan.Months.Allocated)[2].Format() != "100,000.00 GBP" || !slices.Equal(plan.Months.Over, wantOver) {
		t.Errorf("plan allocated %v, over %v; want 100,000.00 GBP allocated over 50,000.00 in Mar, Jun, Sep, Nov", plan.Months.Allocated, plan.Months.Over)
	}
	if campaign.Method != "custom" || campaign.ChildrenNotPhased != 2 || !campaign.Months.Over[3] || campaign.Months.Over[2] ||
		campaign.Months.Difference[2].Format() != "86,666.67 GBP" {
		t.Errorf("campaign = %+v", campaign)
	}
	if uk.RecordID != "I-1009" || uk.Method != "none" || uk.Months.Allocated != nil || uk.Months.Budget[2] != nil ||
		uk.Months.PlannedCost[4].Amount.Format() != "13,333.34" {
		t.Errorf("UK = %+v", uk)
	}
	if ireland.RecordID != "I-1010" || ireland.Months.PlannedCost[8].Amount.Format() != "25,000.00" {
		t.Errorf("Ireland = %+v", ireland)
	}
	if plan.NegativeBalanceCount != 0 || uk.NegativeBalanceCount != 0 { // 038 FR-024: no line items above planned cost
		t.Errorf("negative balance counts = %d, %d; want 0", plan.NegativeBalanceCount, uk.NegativeBalanceCount)
	}
	if gone.Status != http.StatusNotFound {
		t.Errorf("unknown plan = %d %s, want 404", gone.Status, gone.Body)
	}
}
