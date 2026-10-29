package campaign_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

func dateProblem(field string) string {
	return validationBody(`{"field":"` + field + `","message":"Enter a valid date."}`)
}

func TestImpossibleDateIsAFieldProblem_038_FR054_L5(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	skip := apitest.SkipRequestValidation()
	createCampaign := map[string]any{"name": "Leap year", "startDate": "2026-02-30", "endDate": "2026-12-31"}
	createInitiative := map[string]any{"name": "Leap year", "startDate": "2026-03-01", "endDate": "31/12/2026"}
	updateCampaign := helixForm(1)
	updateCampaign["startDate"] = "2026-02-30"
	updateInitiative := digitalForm(1)
	updateInitiative["endDate"] = "2026-13-01"
	tests := []struct {
		name, method, path string
		body               map[string]any
		field              string
	}{
		{"create a campaign", http.MethodPost, "/api/v1/plans/P-1001/campaigns", createCampaign, "startDate"},
		{"create an initiative", http.MethodPost, "/api/v1/campaigns/C-1001/initiatives", createInitiative, "endDate"},
		{"update a campaign", http.MethodPatch, "/api/v1/campaigns/C-1001", updateCampaign, "startDate"},
		{"update an initiative", http.MethodPatch, "/api/v1/initiatives/I-1001", updateInitiative, "endDate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := priya.Do(t, tt.method, tt.path, tt.body, skip)

			assertBody(t, response, http.StatusBadRequest, dateProblem(tt.field))
		})
	}
}

func TestDateThatIsNotTextIsUnreadable_038_FR054(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/campaigns",
		map[string]any{"name": "Numbers", "startDate": 20260101, "endDate": "2026-12-31"}, apitest.SkipRequestValidation())

	if response.Status != http.StatusBadRequest {
		t.Errorf("status %d %s, want 400", response.Status, response.Body)
	}
}

func TestUnknownFieldNextToDatesIsStillRefused_038_FR054(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/campaigns",
		map[string]any{"name": "Colour", "colour": "red"}, apitest.SkipRequestValidation())

	assertBody(t, response, http.StatusBadRequest,
		`{"error":{"code":"bad_request","message":"SP&A One couldn't read this request. Reload the page and try again.","fields":[]}}`)
}
