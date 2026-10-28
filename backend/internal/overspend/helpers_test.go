package overspend_test

import (
	"net/http"
	"strings"
	"testing"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	helix        = "C-1001"
	digital      = "I-1001"
	printOutdoor = "I-1002"
)

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decoded[T any](t *testing.T, r *apitest.Response, status int) T {
	t.Helper()
	if r.Status != status {
		t.Fatalf("status %d %s, want %d", r.Status, r.Body, status)
	}
	var v T
	r.Decode(t, &v)
	return v
}

// refusedNaming checks a 409 over_budget whose message names the budget.
func refusedNaming(t *testing.T, r *apitest.Response, budget string) {
	t.Helper()
	e := decoded[apiError](t, r, http.StatusConflict)
	if e.Error.Code != "over_budget" || !strings.Contains(e.Error.Message, budget) {
		t.Errorf("refusal = %+v, want over_budget naming %s", e, budget)
	}
}

func money2(amount string) map[string]string {
	return map[string]string{"amount": amount, "currency": "USD"}
}

// helixForm is Helix launch's form as the seed stores it, with budget and overspend setting.
func helixForm(version int, budget, mode string) map[string]any {
	return map[string]any{
		"name": "Helix launch", "description": "", "ownerId": "U-1002", "startDate": "2026-01-01", "endDate": "2026-12-31",
		"budget": money2(budget), "overspend": mode, "version": version, "attributes": map[string]any{
			"SA-1003": []string{"SO-1003"}, "SA-1004": []string{"SO-1010"}, "SA-1005": true, "SA-1006": true, "SA-1007": false,
		},
	}
}

// refuseHelix sets Helix launch to Refuse with its budget at exactly its planned cost plus room.
func refuseHelix(t *testing.T, c *apitest.Client, room string) campaign.CampaignView {
	t.Helper()
	view := decoded[campaign.CampaignView](t, c.Do(t, http.MethodGet, "/api/v1/campaigns/"+helix, nil), http.StatusOK)
	extra, err := money.ParseAmount(room)
	if err != nil {
		t.Fatal(err)
	}
	budget := view.BudgetAndCost.PlannedCost.Amount.Add(extra)
	return decoded[campaign.CampaignView](t, c.Do(t, http.MethodPatch, "/api/v1/campaigns/"+helix,
		helixForm(view.Version, budget.String(), "refuse")), http.StatusOK)
}

func newActivity(name, cost string) map[string]any {
	return map[string]any{"name": name, "activityTypeId": "AT-1002", "plannedCost": money2(cost)}
}

func createActivity(t *testing.T, c *apitest.Client, initiativeID string, body map[string]any) *apitest.Response {
	t.Helper()
	return c.Do(t, http.MethodPost, "/api/v1/initiatives/"+initiativeID+"/activities", body)
}

func created(t *testing.T, r *apitest.Response) activity.View {
	t.Helper()
	return decoded[activity.View](t, r, http.StatusCreated)
}
