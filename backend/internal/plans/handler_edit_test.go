package plans_test

import (
	"net/http"
	"testing"
	"time"

	"mapp/internal/plans"
	"mapp/internal/platform/apitest"
)

// europeForm is the edit form of the seed plan FY2026 Lubricants Europe, unchanged, from version.
func europeForm(version int) map[string]any {
	return map[string]any{
		"version": version, "name": "FY2026 Lubricants Europe", "modelId": "MOD-1001", "teamIds": []string{"T-1001"},
		"ownerId": "U-1002", "aop": false, "description": "Lubricants marketing in Europe for 2026.",
		"budget": map[string]any{"amount": "1200000.00", "currency": "USD"}, "defaultFinanceBudgetId": "FB-1001",
		"defaultSpread": "evenly", "bcpir": false, "goalStage": "inq", "goalAmount": "250000",
		"attributes": map[string]any{"SA-1001": []string{"SO-1002"}},
	}
}

// Saving the edit form answers the plan after the change: still Published, version 2, by Daniel. A save
// from the old version is refused naming Daniel and the time; a missing version and an empty name are
// reported at once; an unknown plan is not found (FR-018, FR-022, FR-023, US4). The seed's global budget
// sets this plan's budget (014 FR-031), so the edit changes the description.
func TestUpdatePlan_200_409_404_FR018_FR023(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(90 * time.Minute)
	edited := europeForm(1)
	edited["description"] = "Lubricants marketing in Europe for 2026, with the spring campaign."
	invalid := europeForm(0)
	invalid["name"] = ""

	saved := daniel.Do(t, http.MethodPatch, "/api/v1/plans/P-1001", edited)
	outOfDate := priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1001", europeForm(1))
	refused := priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1001", invalid)
	gone := priya.Do(t, http.MethodPatch, "/api/v1/plans/P-9999", europeForm(1))

	assertFields(t, saved, http.StatusOK, map[string]any{
		"status": "published", "version": 2, "year": 2026,
		"description": "Lubricants marketing in Europe for 2026, with the spring campaign.", "updatedAt": "2026-03-12T10:30:00Z", "updatedBy": map[string]any{"id": "U-1003", "name": "Daniel Okoro"},
	})
	assertAnswer(t, outOfDate, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this plan",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a plan name."},`+
		`{"field":"version","message":"Reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// The consequences of a new model and business unit are listed before saving, without saving anything:
// removing Lubricants from FY2027 Global Brand hides the Lubricants attributes of its activities (FR-019,
// FR-020, US4 scenarios 2 and 3).
func TestChangeEffects_FR019_FR020(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	effects := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1003/change-effects",
		map[string]any{"modelId": "MOD-1002", "attributes": map[string]any{"SA-1001": []string{"SO-1001"}}})
	unknown := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1003/change-effects",
		map[string]any{"modelId": "MOD-9999", "attributes": map[string]any{}})
	gone := priya.Do(t, http.MethodPost, "/api/v1/plans/P-9999/change-effects", map[string]any{"modelId": nil, "attributes": map[string]any{}})

	assertAnswer(t, effects, http.StatusOK, `{"modelChanged":true,"disallowedActivities":[],"hiddenGroups":[`+
		`{"unit":"Lubricants","levels":["activity"],"attributes":["Brands","Global key accounts","Sector","Sub-sector","Product launch"]}],`+
		`"shownGroups":[]}`)
	assertAnswer(t, unknown, http.StatusBadRequest, validationBody(`{"field":"modelId","message":"Choose a model from the list."}`))
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
	var plan plans.PlanView
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1003", nil).Decode(t, &plan)
	if plan.Version != 1 || plan.Model.ID != "MOD-1001" || len(plan.BusinessUnit) != 2 {
		t.Errorf("FY2027 Global Brand changed: version %d, model %s, business unit %+v", plan.Version, plan.Model.ID, plan.BusinessUnit)
	}
}

// formOf is the edit form of a plan as the plan page shows it, unchanged, as the web app sends it.
func formOf(p plans.PlanView) map[string]any {
	teamIDs := []string{}
	for _, team := range p.Teams {
		teamIDs = append(teamIDs, team.ID)
	}
	form := map[string]any{
		"version": p.Version, "name": p.Name, "modelId": p.Model.ID, "teamIds": teamIDs, "ownerId": nil,
		"aop": p.AOP, "description": p.Description, "budget": p.Budget, "defaultFinanceBudgetId": nil,
		"defaultSpread": p.DefaultSpread, "bcpir": p.BCPIR, "goalStage": p.GoalStage, "goalAmount": p.GoalAmount,
		"attributes": p.Attributes,
	}
	if p.Owner != nil {
		form["ownerId"] = p.Owner.ID
	}
	if p.DefaultFinanceBudget != nil {
		form["defaultFinanceBudgetId"] = p.DefaultFinanceBudget.ID
	}
	return form
}
