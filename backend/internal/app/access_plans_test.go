package app_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/platform/apitest"
)

// Priya and Mei see their own market's plans, Sara both of hers, Carlos every plan; the count is over what they see.
func TestPlansList_FilteredByScope_FR006_FR007(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	for _, c := range []struct {
		email string
		want  []string
	}{
		{priyasEmail, []string{"P-1004", "P-1001"}},
		{meisEmail, []string{"P-1002"}},
		{sarasEmail, []string{"P-1004", "P-1001", "P-1002"}},
		{carlosEmail, []string{"P-1003", "P-1004", "P-1001", "P-1002"}},
		{alexsEmail, []string{"P-1003", "P-1004", "P-1001", "P-1002"}},
	} {
		response := server.SignIn(t, c.email).Do(t, http.MethodGet, "/api/v1/plans", nil)
		var list struct {
			Items []struct{ ID string }
			Total int
		}
		response.Decode(t, &list)
		var got []string
		for _, p := range list.Items {
			got = append(got, p.ID)
		}
		if !slices.Equal(got, c.want) || list.Total != len(c.want) {
			t.Errorf("%s sees %q of %d, want %q", c.email, got, list.Total, c.want)
		}
	}
}

// Mei's role now applies to Global Marketing Ops only, yet she still sees the plan she owns.
func TestPlan_OwnerSeesIt_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex, mei, priya := server.SignIn(t, alexsEmail), server.SignIn(t, meisEmail), server.SignIn(t, priyasEmail)
	saved := alex.Do(t, http.MethodPut, "/api/v1/people/U-1004/roles", map[string]any{"version": 1, "roles": []map[string]any{
		{"roleId": "RL-1003", "scope": map[string]any{"type": "teams", "teamIds": []string{"T-1003"}}}}})
	if saved.Status != http.StatusOK {
		t.Fatalf("changing Mei's roles = %d %s", saved.Status, saved.Body)
	}

	if opened := mei.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil); opened.Status != http.StatusOK {
		t.Errorf("Mei, the owner, opening her plan = %d %s, want 200", opened.Status, opened.Body)
	}
	if got := itemIDs(t, mei.Do(t, http.MethodGet, "/api/v1/plans", nil)); !slices.Equal(got, []string{"P-1003", "P-1002"}) {
		t.Errorf("Mei's plans = %q, want Global Marketing Ops's and her own", got)
	}
	hidden(t, "Priya opening it", priya.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil), "plan")
}

func TestRecordOutsideAccess_AnswersLikeAMissingOne_022_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	mei := server.SignIn(t, meisEmail)
	for _, c := range []struct{ method, outside, missing string }{
		{http.MethodGet, "/api/v1/plans/P-1001", "/api/v1/plans/P-9999"},
		{http.MethodGet, "/api/v1/plans/P-1001/campaigns", "/api/v1/plans/P-9999/campaigns"},
		{http.MethodGet, "/api/v1/campaigns/C-1001", "/api/v1/campaigns/C-9999"},
		{http.MethodGet, "/api/v1/initiatives/I-1001", "/api/v1/initiatives/I-9999"},
		{http.MethodGet, "/api/v1/activities/A-1001", "/api/v1/activities/A-9999"},
		{http.MethodGet, "/api/v1/activities/A-1001/approval", "/api/v1/activities/A-9999/approval"},
		{http.MethodGet, "/api/v1/line-items/L-1001", "/api/v1/line-items/L-9999"},
		{http.MethodGet, "/api/v1/all-work/records/A-1001", "/api/v1/all-work/records/A-9999"},
		{http.MethodDelete, "/api/v1/activities/A-1001", "/api/v1/activities/A-9999"},
		{http.MethodDelete, "/api/v1/line-items/L-1001", "/api/v1/line-items/L-9999"},
	} {
		outside, missing := mei.Do(t, c.method, c.outside, nil), mei.Do(t, c.method, c.missing, nil)
		if outside.Status != http.StatusNotFound || missing.Status != outside.Status || string(missing.Body) != string(outside.Body) {
			t.Errorf("Mei: %s %s = %d %s\nwant as %s: %d %s", c.method, c.outside, outside.Status, outside.Body,
				c.missing, missing.Status, missing.Body)
		}
	}
	hidden(t, "Mei: GET the line items of A-1001", mei.Do(t, http.MethodGet, "/api/v1/activities/A-1001/line-items", nil), "activity")
	if own := mei.Do(t, http.MethodGet, "/api/v1/activities/A-1008", nil); own.Status != http.StatusOK {
		t.Errorf("Mei opening her own market's activity = %d %s, want 200", own.Status, own.Body)
	}
}

// Sara (Viewer) is told her role does not include it; Priya may change only her market's records.
func TestChanges_NeedThePermissionInThePlan_FR009_FR010(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	sara, priya := server.SignIn(t, sarasEmail), server.SignIn(t, priyasEmail)
	rename := map[string]any{"name": "Renamed", "version": 1}

	refused(t, "Sara renaming a campaign", sara.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", rename, apitest.SkipRequestValidation()),
		"Changing campaigns and initiatives isn't part of your role (Viewer). Ask an administrator if you need it.")
	refused(t, "Sara deleting an activity", sara.Do(t, http.MethodDelete, "/api/v1/activities/A-1001", nil),
		"Deleting activities and line items isn't part of your role (Viewer). Ask an administrator if you need it.")
	refused(t, "Priya deleting a plan", priya.Do(t, http.MethodDelete, "/api/v1/plans/P-1001", nil),
		"Deleting plans isn't part of your role (Planner). Ask an administrator if you need it.")
	hidden(t, "Priya adding a campaign to the Mobility plan",
		priya.Do(t, http.MethodPost, "/api/v1/plans/P-1002/campaigns", map[string]any{"name": "Spring"}, apitest.SkipRequestValidation()),
		"plan")

	// Given the Planner role for Mobility China too, Priya sees the Mobility plan, and still may not delete there.
	server.GrantRole(t, priyasEmail, "viewer", "teams", "Mobility China")
	refused(t, "Priya, a Viewer of Mobility China, changing its campaign",
		priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1003", rename, apitest.SkipRequestValidation()),
		"Changing campaigns and initiatives is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here.")

	var campaign struct {
		Name    string
		Version int
	}
	sara.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001", nil).Decode(t, &campaign)
	if campaign.Name != "Helix launch" || campaign.Version != 1 {
		t.Errorf("after the refusals the campaign is %q version %d, want unchanged", campaign.Name, campaign.Version)
	}
}

// Carlos, for All markets, may give a plan any team (messages 20 and 21).
func TestPlanTeams_OwnTeamsOnly_FR013(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya, carlos := server.SignIn(t, priyasEmail), server.SignIn(t, carlosEmail)

	other := priya.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Mobility extra", "T-1002"))
	mine := priya.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Lubricants extra", "T-1001"))
	carlosPlan := carlos.Do(t, http.MethodPost, "/api/v1/plans", planForm("FY2026 Global extra", "T-1001", "T-1002"))
	var shared struct{ ID string }
	carlosPlan.Decode(t, &shared)
	edit := planForm("FY2026 Global extra", "T-1002") // Priya takes Lubricants Europe off Carlos's plan
	delete(edit, "year")
	edit["version"] = 1
	dropped := priya.Do(t, http.MethodPatch, "/api/v1/plans/"+shared.ID, edit)

	assertProblem := func(what string, r *apitest.Response, message string) {
		t.Helper()
		var e errorAnswer
		r.Decode(t, &e)
		if r.Status != http.StatusBadRequest || !slices.ContainsFunc(e.Error.Fields, func(f struct{ Field, Message string }) bool {
			return f.Field == "teamIds" && f.Message == message
		}) {
			t.Errorf("%s = %d %s\nwant teamIds: %s", what, r.Status, r.Body, message)
		}
	}
	assertProblem("Priya's plan for Mobility China", other, "You can only choose your own teams: Lubricants Europe.")
	assertProblem("Priya dropping her team", dropped,
		"Keep at least one of your teams (Lubricants Europe) on the plan, or you won't be able to see it.")
	if mine.Status != http.StatusCreated || carlosPlan.Status != http.StatusCreated {
		t.Errorf("Priya's own plan = %d %s; Carlos's = %d %s; want both created", mine.Status, mine.Body, carlosPlan.Status, carlosPlan.Body)
	}
}

// Destination lists offer only plans where the person may change campaigns.
func TestCampaignMoves_DestinationInScope_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "viewer", "teams", "Mobility China") // she sees the Mobility plan, may not change it

	var destinations struct{ Items []struct{ ID string } }
	priya.Do(t, http.MethodGet, "/api/v1/campaigns/C-1001/destinations?purpose=copy", nil).Decode(t, &destinations)
	moved := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/move", map[string]any{"planId": "P-1002", "version": 1})
	copied := priya.Do(t, http.MethodPost, "/api/v1/campaigns/C-1001/copy", map[string]any{"planId": "P-1002", "name": "Helix copy"})

	if len(destinations.Items) != 1 || destinations.Items[0].ID != "P-1001" {
		t.Errorf("copy destinations = %+v, want her own plan only", destinations.Items)
	}
	message := "Changing campaigns and initiatives is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."
	refused(t, "moving to the Mobility plan", moved, message)
	refused(t, "copying to the Mobility plan", copied, message)
}
