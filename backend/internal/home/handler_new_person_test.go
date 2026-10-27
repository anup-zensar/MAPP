package home_test

import (
	"net/http"
	"testing"

	"mapp/internal/home"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

// User story 5 — Get started as a new user.

// A new person in no team gets the Getting started card with the no-team reason and empty cards; once they
// join a team that has a plan, the card is gone (FR-020, FR-021, US5 scenarios 1 and 2).
func TestHomeSummary_GettingStartedForANewPerson_FR020(t *testing.T) { //nolint:gocyclo // 12: the new person before and after joining a team, each card checked in turn
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	created := alex.Do(t, http.MethodPost, "/api/v1/people", map[string]any{"name": "Nina Berg", "email": "nina.berg@shell.com"})
	var nina struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	}
	created.Decode(t, &nina)
	if created.Status != http.StatusCreated && created.Status != http.StatusOK {
		t.Fatalf("POST /api/v1/people = %d %s", created.Status, created.Body)
	}
	newcomer := server.SignIn(t, "nina.berg@shell.com")

	alone := get[home.SummaryView](t, newcomer, "/api/v1/home/summary?scope=myTeams")
	joined := alex.Do(t, http.MethodPatch, "/api/v1/people/"+nina.ID, map[string]any{"name": "Nina Berg",
		"email": "nina.berg@shell.com", "jobTitle": "", "teamIds": []string{"T-1003"}, "version": nina.Version})
	inTeam := get[home.SummaryView](t, newcomer, "/api/v1/home/summary?scope=myTeams")

	if alone.GettingStarted == nil || *alone.GettingStarted != "notInTeam" {
		t.Errorf("getting started = %v, want notInTeam", alone.GettingStarted)
	}
	empty := home.PlannedCostTotal{PlannedCost: money.USD(money.Amount{})}
	if alone.Plans != (home.PlanCounts{}) || alone.ActivitiesOwned != (home.ActivityCounts{}) ||
		alone.WaitingForMe != (home.WaitingCounts{}) || alone.PlannedCost != empty {
		t.Errorf("cards = %+v, want every card empty", alone)
	}
	if joined.Status != http.StatusOK || inTeam.GettingStarted != nil || inTeam.Plans.Count != 1 {
		t.Errorf("after joining Global Marketing Ops (%d %s): getting started %v, plans %+v; want no card and FY2027 Global Brand",
			joined.Status, joined.Body, inTeam.GettingStarted, inTeam.Plans)
	}
}

// A scope other than myTeams or everything, or none, is refused on the field scope.
func TestHome_InvalidScope_400(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	refused := `{"error":{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"scope","message":"Choose My teams or Everything."}]}}`

	for _, path := range []string{"/api/v1/home/summary?scope=all", "/api/v1/home/summary", "/api/v1/home/plans?scope=mine", "/api/v1/home/plans"} {
		response := priya.Do(t, http.MethodGet, path, nil, apitest.SkipRequestValidation()) // the contract allows only the two

		if response.Status != http.StatusBadRequest || string(response.Body) != refused {
			t.Errorf("GET %s = %d %s\nwant 400 %s", path, response.Status, response.Body, refused)
		}
	}
}
