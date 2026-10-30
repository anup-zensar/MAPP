package overspend_test

import (
	"net/http"
	"testing"

	"mapp/internal/campaign"
	"mapp/internal/overspend"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

func TestInitiativeRefuse_CreateActivityOverTheBudget_Contract_037_US1(t *testing.T) {
	t.Parallel()
	priya := apitest.New(t).SignIn(t, priyasEmail)
	path := "/api/v1/initiatives/" + printOutdoor
	view := decoded[campaign.InitiativeView](t, priya.Do(t, http.MethodGet, path, nil), http.StatusOK)
	budget := view.BudgetAndCost.PlannedCost.Amount.Add(money.FromCents(100 * 100))
	set := decoded[campaign.InitiativeView](t, priya.Do(t, http.MethodPatch, path, map[string]any{
		"name": view.Name, "description": view.Description, "ownerId": view.OwnerID, "startDate": view.StartDate,
		"endDate": view.EndDate, "budget": money2(budget.String()), "attributes": view.Attributes,
		"overspend": "refuse", "version": view.Version,
	}), http.StatusOK)

	within := createActivity(t, priya, printOutdoor, newActivity("Within", "100.00"))
	over := createActivity(t, priya, printOutdoor, newActivity("One cent over", "0.01"))

	if set.Overspend.Mode != overspend.Refuse {
		t.Errorf("037 US1 set = %+v, want Refuse", set.Overspend)
	}
	created(t, within)
	refusedNaming(t, over, "initiative 'Print & outdoor'")
}
