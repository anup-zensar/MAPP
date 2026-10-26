package campaigns_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// The handler tests of 016 on the demo data: P-1004 "FY2027 Lubricants UK & Ireland" is planned in GBP (0.79
// per USD); its campaign C-1005 "Helix UK & Ireland" has 400,000.00 GBP phased 100,000.00 in March, June,
// September and November 2027, and the initiatives I-1009 "UK" (250,000.00 GBP) and I-1010 "Ireland"
// (150,000.00 GBP), not phased.

// phased is a campaign or initiative with its amounts in the plan currency, the display amounts and the
// phasing, as the API answers them.
type phased struct {
	ID      string  `json:"id"`
	Budget  *amount `json:"budget"`
	Display struct {
		Budget *amount `json:"budget"`
	} `json:"display"`
	BudgetPhasing struct {
		Method  string   `json:"method"`
		Monthly []amount `json:"monthly"`
		Total   *amount  `json:"total"`
		Problem *string  `json:"problem"`
	} `json:"budgetPhasing"`
	Version int `json:"version"`
}

// getPhased reads the record at path and stops the test unless it answers 200.
func getPhased(t *testing.T, client *apitest.Client, path string) phased {
	t.Helper()
	response := client.Do(t, http.MethodGet, path, nil)
	assertStatus(t, response, http.StatusOK)
	var record phased
	response.Decode(t, &record)
	return record
}

// A campaign's budget is in the plan currency, its display amount converted from it (FR-011, FR-013).
func TestCampaignBudget_PlanCurrency_Contract_FR011_FR013(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	helix := getPhased(t, priya, "/api/v1/campaigns/C-1005")
	refused := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1004/campaigns", map[string]any{
		"name": "Castrol UK", "budget": usd("1000.00"),
	})

	if *helix.Budget != (amount{"400000.00", "GBP"}) || helix.Display.Budget == nil || *helix.Display.Budget != usd("506329.11") ||
		helix.BudgetPhasing.Method != "custom" || len(helix.BudgetPhasing.Monthly) != 12 ||
		helix.BudgetPhasing.Monthly[2] != (amount{"100000.00", "GBP"}) || helix.BudgetPhasing.Problem != nil {
		t.Errorf("C-1005 = %+v", helix)
	}
	assertBody(t, refused, http.StatusBadRequest, validationBody(`{"field":"budget","message":"Enter the amount in GBP, the plan's currency."}`))
}

// Budget by month of a campaign and an initiative is saved with PUT …/budget-phasing (FR-020 – FR-023): Evenly
// splits the budget over the 12 months; Custom months that add up to the budget leave no problem; an old
// version conflicts; an unknown record is not found.
func TestSetBudgetPhasing_CampaignAndInitiative_Contract_FR020(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	helix := getPhased(t, priya, "/api/v1/campaigns/C-1005")
	uk := getPhased(t, priya, "/api/v1/initiatives/I-1009")

	evenly := priya.Do(t, http.MethodPut, "/api/v1/campaigns/C-1005/budget-phasing",
		map[string]any{"method": "evenly", "months": []any{}, "version": helix.Version})
	custom := map[string]any{"method": "custom", "version": uk.Version, "months": []map[string]any{
		{"month": "2027-03", "amount": "150000.00"}, {"month": "2027-04", "amount": "100000"},
	}}
	ukSaved := priya.Do(t, http.MethodPut, "/api/v1/initiatives/I-1009/budget-phasing", custom)
	stale := priya.Do(t, http.MethodPut, "/api/v1/initiatives/I-1009/budget-phasing", custom)
	gone := priya.Do(t, http.MethodPut, "/api/v1/initiatives/I-9999/budget-phasing", custom)

	assertStatus(t, evenly, http.StatusOK)
	var c phased
	evenly.Decode(t, &c)
	if c.BudgetPhasing.Method != "evenly" || c.BudgetPhasing.Monthly[0] != (amount{"33333.33", "GBP"}) ||
		c.BudgetPhasing.Monthly[11] != (amount{"33333.37", "GBP"}) || c.Version != helix.Version+1 {
		t.Errorf("campaign after Evenly = %+v", c.BudgetPhasing)
	}
	assertStatus(t, ukSaved, http.StatusOK)
	var i phased
	ukSaved.Decode(t, &i)
	if i.BudgetPhasing.Method != "custom" || i.BudgetPhasing.Total == nil || *i.BudgetPhasing.Total != (amount{"250000.00", "GBP"}) ||
		i.BudgetPhasing.Problem != nil || i.BudgetPhasing.Monthly[3] != (amount{"100000.00", "GBP"}) {
		t.Errorf("UK after Custom = %+v", i.BudgetPhasing)
	}
	if stale.Status != http.StatusConflict || gone.Status != http.StatusNotFound {
		t.Errorf("stale = %d %s, unknown = %d; want 409 and 404", stale.Status, stale.Body, gone.Status)
	}
}
