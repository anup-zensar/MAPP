package overspend_test

import (
	"net/http"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/overspend"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

func TestCampaignRefuse_CreateActivityOverTheBudget_Contract_037_US1(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	set := refuseHelix(t, priya, "100.00")
	within := createActivity(t, priya, digital, newActivity("Within", "100.00"))
	over := createActivity(t, priya, printOutdoor, newActivity("One cent over", "0.01"))
	warn := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodPatch, "/api/v1/campaigns/"+helix,
		helixForm(set.Version, set.Budget.Amount.String(), "warn")), http.StatusOK)
	afterWarn := createActivity(t, priya, printOutdoor, newActivity("Over on Warn", "5000.00"))

	if set.Overspend.Mode != overspend.Refuse || !set.Overspend.CanChange.Available {
		t.Errorf("037 FR-001 FR-002 set = %+v, want Refuse that Priya may change", set.Overspend)
	}
	created(t, within)
	refusedNaming(t, over, "the planned cost of campaign 'Helix launch' would be ")
	if warn.Overspend.Mode != overspend.Warn {
		t.Errorf("037 FR-001 back to Warn = %+v", warn.Overspend)
	}
	created(t, afterWarn)
}

func TestCampaignRefuse_UpdateActivityAndLowerTheBudget_Contract_037_US1(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	a := created(t, createActivity(t, priya, digital, newActivity("Raised later", "1000.00")))
	set := refuseHelix(t, priya, "0.00")
	update := func(cost string) map[string]any {
		return map[string]any{"name": a.Name, "externalName": "", "activityTypeId": "AT-1002", "ownerId": nil,
			"startDate": nil, "endDate": nil, "ongoing": false, "plannedCost": money2(cost), "approvalWorkflowId": nil,
			"description": "", "attributes": map[string]any{}, "goals": map[string]any{"inq": nil, "setLaterStagesMyself": false, "mql": nil, "revenue": nil},
			"version": a.Version, "restartApproval": false}
	}

	raised := priya.Do(t, http.MethodPatch, "/api/v1/activities/"+a.ID, update("1000.01"))
	lowered := priya.Do(t, http.MethodPatch, "/api/v1/activities/"+a.ID, update("900.00"))
	cut := priya.Do(t, http.MethodPatch, "/api/v1/campaigns/"+helix, helixForm(set.Version, "1.00", "refuse"))

	refusedNaming(t, raised, "campaign 'Helix launch'")
	decoded[activity.View](t, lowered, http.StatusOK)
	over := set.BudgetAndCost.PlannedCost.Amount.Sub(money.FromCents(100*100 + 100))
	refusedNaming(t, cut, "over its budget of 1.00 USD by "+over.Format()+" USD")
}

func TestOverspendSetting_OnlyOwnerOrBudgetManager_Contract_037_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, alex := server.SignIn(t, priyasEmail), server.SignIn(t, alexsEmail)
	rimula := decoded[campaign.CampaignView](t, priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1002", nil), http.StatusOK)
	form := map[string]any{"name": rimula.Name, "description": rimula.Description, "ownerId": "U-1003",
		"startDate": "2026-02-01", "endDate": "2026-12-31", "budget": money2("450000.00"), "attributes": rimula.Attributes,
		"overspend": "refuse", "version": rimula.Version}

	refused := priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1002", form)
	allowed := alex.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1002", form)

	e := decoded[apiError](t, refused, http.StatusForbidden)
	if e.Error.Message != "Only the budget's owner or someone who manages budgets can change this." ||
		rimula.Overspend.CanChange.Available {
		t.Errorf("037 FR-003 Priya: %+v, control %+v", e, rimula.Overspend)
	}
	if saved := decoded[campaign.CampaignView](t, allowed, http.StatusOK); saved.Overspend.Mode != overspend.Refuse {
		t.Errorf("037 FR-003 Alex manages budgets: %+v", saved.Overspend)
	}
}

func TestOverspendSetting_LoggedOnTheRecord_Contract_037_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	refuseHelix(t, priya, "0.00")

	var log struct {
		Items []struct {
			Changes []struct {
				Label, OldText, NewText string
			} `json:"changes"`
		} `json:"items"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=campaign&recordId="+helix, nil).Decode(t, &log)
	found := false
	for _, entry := range log.Items {
		for _, c := range entry.Changes {
			found = found || (c.Label == "Overspend" && c.OldText == "Warn" && c.NewText == "Refuse")
		}
	}
	if !found {
		t.Errorf("037 FR-005 log = %+v, want Overspend: Warn → Refuse", log)
	}
}
