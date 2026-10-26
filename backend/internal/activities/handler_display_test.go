package activities_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// amountOf is an amount as the API answers it.
type amountOf struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// displayedActivity is the part of an Activity or ActivitySummary answer that D-220 is about.
type displayedActivity struct {
	ID                    string    `json:"id"`
	PlannedCost           *amountOf `json:"plannedCost"`
	UnassignedPlannedCost *amountOf `json:"unassignedPlannedCost"`
	Goals                 *struct {
		Revenue *amountOf `json:"revenue"`
	} `json:"goals"`
	Display struct {
		PlannedCost           *amountOf `json:"plannedCost"`
		UnassignedPlannedCost *amountOf `json:"unassignedPlannedCost"`
		Revenue               *amountOf `json:"revenue"`
	} `json:"display"`
}

// A reader whose display currency is EUR gets Launch film's planned cost (120,000.00 USD), its unassigned
// planned cost (20,000.00 USD) and its revenue goal a second time in euros at 0.92, on the activity and on
// its plan-tree row; the USD amounts the form uses stay as they are (D-061, D-220).
func TestActivity_DisplayCurrency_D220(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	assertStatus(t, priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "EUR"}), http.StatusOK)

	var activity displayedActivity
	response := priya.Do(t, http.MethodGet, "/api/v1/activities/A-1001", nil)
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &activity)
	if *activity.PlannedCost != (amountOf{"120000.00", "USD"}) || *activity.UnassignedPlannedCost != (amountOf{"20000.00", "USD"}) {
		t.Fatalf("USD amounts = %+v, %+v", activity.PlannedCost, activity.UnassignedPlannedCost)
	}
	if *activity.Display.PlannedCost != (amountOf{"110400.00", "EUR"}) ||
		*activity.Display.UnassignedPlannedCost != (amountOf{"18400.00", "EUR"}) {
		t.Errorf("display amounts = %+v, %+v", activity.Display.PlannedCost, activity.Display.UnassignedPlannedCost)
	}
	if activity.Goals.Revenue == nil || activity.Display.Revenue == nil || activity.Display.Revenue.Currency != "EUR" ||
		activity.Goals.Revenue.Currency != "USD" {
		t.Errorf("revenue = %+v, display %+v", activity.Goals.Revenue, activity.Display.Revenue)
	}

	var page struct {
		Items []displayedActivity `json:"items"`
	}
	response = priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/activities", nil)
	assertStatus(t, response, http.StatusOK)
	response.Decode(t, &page)
	for _, row := range page.Items {
		if row.ID != "A-1001" {
			continue
		}
		if *row.PlannedCost != (amountOf{"120000.00", "USD"}) || *row.Display.PlannedCost != (amountOf{"110400.00", "EUR"}) ||
			*row.Display.UnassignedPlannedCost != (amountOf{"18400.00", "EUR"}) || row.Display.Revenue != nil {
			t.Errorf("Launch film's row = %+v", row)
		}
		return
	}
	t.Error("the plan's rows have no Launch film")
}
