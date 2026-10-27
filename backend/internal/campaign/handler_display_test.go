package campaign_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// displayed is the part of a Campaign or Initiative answer that D-220 adds: its amounts in the reader's
// display currency, beside the USD ones.
type displayed struct {
	Name          string  `json:"name"`
	Budget        *amount `json:"budget"`
	BudgetAndCost struct {
		RemainingForChildren *amount `json:"remainingForChildren"`
		PlannedCost          *amount `json:"plannedCost"`
	} `json:"budgetAndCost"`
	Display struct {
		Budget                *amount `json:"budget"`
		ParentRemainingBudget *amount `json:"parentRemainingBudget"`
		BudgetAndCost         struct {
			AllocatedToChildren  amount  `json:"allocatedToChildren"`
			RemainingForChildren *amount `json:"remainingForChildren"`
			PlannedCost          *amount `json:"plannedCost"`
			BudgetUsedPercent    *int    `json:"budgetUsedPercent"`
		} `json:"budgetAndCost"`
	} `json:"display"`
	Initiatives []displayed `json:"initiatives"`
}

// eur is a EUR amount as the API answers it.
func eur(text string) amount { return amount{Amount: text, Currency: "EUR"} }

// useEuros makes EUR (0.92 per USD in the demo data) the display currency of the signed-in person.
func useEuros(t *testing.T, c *apitest.Client) {
	t.Helper()
	response := c.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "EUR"})
	assertStatus(t, response, http.StatusOK)
}

// A reader whose display currency is EUR gets the tree's amounts a second time in euros — every campaign's
// and initiative's budget, budget and cost and the totals row — converted once from USD at 0.92, while the
// USD amounts the forms use stay as they are (D-061, D-220).
func TestListPlanCampaigns_DisplayCurrency_D220(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	useEuros(t, priya)

	response := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/campaigns", nil)

	assertStatus(t, response, http.StatusOK)
	var tree struct {
		Items  []displayed `json:"items"`
		Totals struct {
			PlannedCost *amount `json:"plannedCost"`
		} `json:"totals"`
		DisplayTotals struct {
			Campaigns            int     `json:"campaigns"`
			AllocatedToCampaigns amount  `json:"allocatedToCampaigns"`
			PlanBudget           *amount `json:"planBudget"`
			RemainingPlanBudget  *amount `json:"remainingPlanBudget"`
			PlannedCost          *amount `json:"plannedCost"`
		} `json:"displayTotals"`
	}
	response.Decode(t, &tree)
	helix := tree.Items[0]
	if helix.Name != "Helix launch" || *helix.Budget != usd("700000.00") || *helix.BudgetAndCost.PlannedCost != usd("275000.00") {
		t.Fatalf("Helix launch in USD = %+v", helix)
	}
	shown := helix.Display
	if *shown.Budget != eur("644000.00") || *shown.ParentRemainingBudget != eur("46000.00") ||
		shown.BudgetAndCost.AllocatedToChildren != eur("506000.00") || *shown.BudgetAndCost.RemainingForChildren != eur("138000.00") ||
		*shown.BudgetAndCost.PlannedCost != eur("253000.00") || *shown.BudgetAndCost.BudgetUsedPercent != 39 {
		t.Errorf("Helix launch in EUR = %+v", shown)
	}
	digital := helix.Initiatives[0]
	if digital.Name != "Digital & social" || *digital.Display.Budget != eur("276000.00") ||
		*digital.Display.ParentRemainingBudget != eur("138000.00") || *digital.Display.BudgetAndCost.PlannedCost != eur("184000.00") {
		t.Errorf("Digital & social in EUR = %+v", digital.Display)
	}
	totals := tree.DisplayTotals
	if totals.Campaigns != 2 || totals.AllocatedToCampaigns != eur("1058000.00") || *totals.PlanBudget != eur("1104000.00") ||
		*totals.RemainingPlanBudget != eur("46000.00") || *totals.PlannedCost != eur("423200.00") || *tree.Totals.PlannedCost != usd("460000.00") {
		t.Errorf("display totals = %+v, totals planned cost %+v", totals, tree.Totals.PlannedCost)
	}
}

// A campaign and an initiative read on their own carry the same display amounts; a reader in USD sees the
// USD amounts twice (D-220).
func TestGetCampaignAndInitiative_DisplayCurrency_D220(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	useEuros(t, priya)

	var campaign displayed
	response := priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001", nil)
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &campaign)
	if *campaign.Display.Budget != eur("644000.00") || *campaign.Budget != usd("700000.00") {
		t.Errorf("campaign budget = %+v, display %+v", campaign.Budget, campaign.Display.Budget)
	}

	var initiative displayed
	response = priya.Do(t, http.MethodGet, "/api/v1/initiatives/I-1001", nil)
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &initiative)
	if *initiative.Display.Budget != eur("276000.00") || *initiative.Budget != usd("300000.00") {
		t.Errorf("initiative budget = %+v, display %+v", initiative.Budget, initiative.Display.Budget)
	}

	response = daniel.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001", nil)
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &campaign)
	if *campaign.Display.Budget != usd("700000.00") || *campaign.Display.BudgetAndCost.PlannedCost != usd("275000.00") {
		t.Errorf("Daniel's display amounts = %+v, want USD", campaign.Display)
	}
}
