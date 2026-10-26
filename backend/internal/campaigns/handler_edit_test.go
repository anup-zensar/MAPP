package campaigns_test

import (
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

// helixForm is Helix launch's form as the seed stores it, ready to change and send.
func helixForm(version int) map[string]any {
	return map[string]any{
		"name": "Helix launch", "description": "", "ownerId": "U-1002", "startDate": "2026-01-01", "endDate": "2026-12-31",
		"budget": usd("700000.00"), "attributes": map[string]any{
			"SA-1003": []string{"SO-1003"}, "SA-1004": []string{"SO-1010"}, "SA-1005": true, "SA-1006": true, "SA-1007": false,
		},
		"version": version,
	}
}

// digitalForm is Digital & social's form as the seed stores it.
func digitalForm(version int) map[string]any {
	return map[string]any{
		"name": "Digital & social", "description": "", "ownerId": "U-1002", "startDate": "2026-01-01", "endDate": "2026-06-30",
		"budget": usd("300000.00"), "attributes": map[string]any{
			"SA-1008": []string{"SO-1015"}, "SA-1009": "SO-1019", "SA-1010": "SO-1023", "SA-1011": false,
		},
		"version": version,
	}
}

// Saving Helix launch with a new budget and owner answers the campaign with its new version, and the plan's
// remaining budget follows at once (FR-012, FR-018, US3 scenario 2).
func TestUpdateCampaign_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.Advance(time.Hour)
	form := helixForm(1)
	form["budget"], form["ownerId"] = usd("600000.00"), "U-1003"

	response := priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", form)

	assertStatus(t, response, http.StatusOK)
	var helix campaign
	response.Decode(t, &helix)
	if helix.Version != 2 || *helix.OwnerID != "U-1003" || *helix.Budget != usd("600000.00") || helix.UpdatedBy.Name != "Priya Raman" ||
		*helix.ParentRemainingBudget != usd("150000.00") || *helix.BudgetAndCost.RemainingForChildren != usd("50000.00") {
		t.Errorf("Helix launch = %+v", helix)
	}
	var tree planCampaigns
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/campaigns", nil).Decode(t, &tree)
	if *tree.Totals.RemainingPlanBudget != usd("150000.00") {
		t.Errorf("remaining plan budget = %+v, want 150000.00", tree.Totals.RemainingPlanBudget)
	}
}

// An update without a version is refused next to "version" with the other problems; narrowing Helix
// launch's dates past Digital & social names it (FR-007, FR-018).
func TestUpdateCampaign_MissingVersion400(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	form := helixForm(1)
	delete(form, "version")
	form["name"] = ""

	response := priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", form, apitest.SkipRequestValidation())

	assertBody(t, response, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"Enter a name."}`,
		`{"field":"version","message":"Reload the page and try again."}`,
	))
	narrower := helixForm(1)
	narrower["startDate"] = "2026-03-01"
	assertBody(t, priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", narrower), http.StatusBadRequest, validationBody(
		`{"field":"startDate","message":"These initiatives would be outside these dates: `+
			`Digital \u0026 social (01 Jan – 30 Jun 2026), Print \u0026 outdoor (01 Jan – 31 Dec 2026). `+
			`Change their dates first, or choose dates that include them."}`,
	))
	assertBody(t, priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-9999", helixForm(1)), http.StatusNotFound, campaignGoneBody)
}

// A save from an out-of-date copy answers 409 naming who changed the campaign and when (FR-018).
func TestUpdateCampaign_Conflict409(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(time.Hour)
	first := helixForm(1)
	first["description"] = "Premium launch"
	assertStatus(t, priya.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", first), http.StatusOK)

	response := daniel.Do(t, http.MethodPatch, "/api/v1/campaigns/C-1001", helixForm(1))

	assertBody(t, response, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Priya Raman changed this campaign",`+
		`"fields":[],"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T10:00:00Z"}}}`)
}

// Saving Digital & social answers the initiative with its new version; a save from the old version answers
// 409, and dates outside Helix launch are refused with the allowed range (FR-006, FR-018).
func TestUpdateInitiative_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	form := digitalForm(1)
	form["name"], form["endDate"] = "Digital, social & search", "2026-09-30"

	response := priya.Do(t, http.MethodPatch, "/api/v1/initiatives/I-1001", form)

	assertStatus(t, response, http.StatusOK)
	var digital initiative
	response.Decode(t, &digital)
	if digital.Version != 2 || digital.Name != "Digital, social & search" || *digital.EndDate != "2026-09-30" {
		t.Errorf("Digital & social = %+v", digital)
	}
	assertBody(t, priya.Do(t, http.MethodPatch, "/api/v1/initiatives/I-1001", digitalForm(1)), http.StatusConflict,
		`{"error":{"code":"version_conflict","message":"Not saved — Priya Raman changed this initiative",`+
			`"fields":[],"conflict":{"changedBy":"Priya Raman","changedAt":"2026-03-12T09:00:00Z"}}}`)
	outside := digitalForm(2)
	outside["endDate"] = "2027-01-31"
	assertBody(t, priya.Do(t, http.MethodPatch, "/api/v1/initiatives/I-1001", outside), http.StatusBadRequest, validationBody(
		`{"field":"startDate","message":"Dates must be inside the campaign's dates (01 Jan 2026 – 31 Dec 2026)."}`))
	assertBody(t, priya.Do(t, http.MethodPatch, "/api/v1/initiatives/I-9999", digitalForm(1)), http.StatusNotFound, initiativeGoneBody)
}

// Daniel edits a campaign Priya created: there are no role checks (FR-031, D-004).
func TestAnyoneCanChangeAnyCampaign_FR031(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	response := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/campaigns", map[string]any{"name": "Priya's campaign"})
	assertStatus(t, response, http.StatusCreated)
	var created campaign
	response.Decode(t, &created)

	edit := daniel.Do(t, http.MethodPatch, "/api/v1/campaigns/"+created.ID, map[string]any{
		"name": "Daniel's campaign now", "description": "", "ownerId": nil, "startDate": nil, "endDate": nil,
		"budget": nil, "attributes": map[string]any{}, "version": 1,
	})

	assertStatus(t, edit, http.StatusOK)
	var edited campaign
	edit.Decode(t, &edited)
	if edited.Name != "Daniel's campaign now" || edited.UpdatedBy.Name != "Daniel Okoro" {
		t.Errorf("edited = %+v", edited)
	}
}
