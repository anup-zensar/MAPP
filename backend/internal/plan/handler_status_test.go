package plan_test

import (
	"net/http"
	"strings"
	"testing"

	"mapp/internal/plan"
	"mapp/internal/platform/apitest"
)

// FY2026 Mobility China has no business unit, and its campaign Retail growth has no cluster and region and
// its initiative Fleet cards no dates (005's demo data). Publish lists the plan's field first, then each
// record below that blocks publishing, and the plan stays Draft; once the business unit is filled in and the
// incomplete campaign is gone, Publish succeeds (FR-024, US5 scenarios 1 and 2).
func TestPublish_NotReadyThenPublished_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	refused := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1002/publish", nil)
	var china plan.PlanView
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil).Decode(t, &china)
	form := formOf(china)
	form["attributes"] = map[string]any{"SA-1001": []string{"SO-1002"}}
	saved := priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1002", form)
	stillRefused := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1002/publish", nil)
	deleted := priya.Do(t, http.MethodDelete, "/api/v1/campaigns/C-1003", nil)
	published := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1002/publish", nil)
	gone := priya.Do(t, http.MethodPost, "/api/v1/plans/P-9999/publish", nil)

	below := `{"record":{"type":"campaign","id":"C-1003","name":"Retail growth"},` +
		`"fields":[{"field":"attributes.SA-1003","label":"Cluster"},{"field":"attributes.SA-1004","label":"Region"}]},` +
		`{"record":{"type":"initiative","id":"I-1006","name":"Fleet cards"},"fields":[{"field":"startDate","label":"Dates"}]}`
	assertAnswer(t, refused, http.StatusConflict, `{"error":{"code":"not_ready","message":"Publish needs 4 fields","fields":[],`+
		`"missing":[{"record":{"type":"plan","id":"P-1002","name":"FY2026 Mobility China"},`+
		`"fields":[{"field":"attributes.SA-1001","label":"Business unit"}]},`+below+`]}}`)
	if china.Status != plan.StatusDraft || china.Version != 1 {
		t.Errorf("after the refusal: %s, version %d; want Draft, version 1", china.Status, china.Version)
	}
	assertFields(t, saved, http.StatusOK, map[string]any{"status": "draft", "version": 2})
	assertAnswer(t, stillRefused, http.StatusConflict, `{"error":{"code":"not_ready","message":"Publish needs 3 fields","fields":[],`+
		`"missing":[`+below+`]}}`)
	if deleted.Status != http.StatusOK {
		t.Fatalf("deleting Retail growth = %d %s, want 200", deleted.Status, deleted.Body)
	}
	assertFields(t, published, http.StatusOK, map[string]any{
		"status": "published", "version": 3, "updatedBy": map[string]any{"id": "U-1002", "name": "Priya Raman"},
	})
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// Unpublishing asks for a reason of 1 to 500 characters and returns the plan to Draft; the reason is in
// the plan's activity log (FR-025, US5 scenario 3).
func TestUnpublish_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	empty := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/unpublish", map[string]any{"reason": "  "})
	long := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/unpublish", map[string]any{"reason": strings.Repeat("x", 501)})
	unpublished := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/unpublish", map[string]any{"reason": "Budget under review"})
	log := priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=plan&recordId=P-1001", nil)
	gone := priya.Do(t, http.MethodPost, "/api/v1/plans/P-9999/unpublish", map[string]any{"reason": "Budget under review"})

	assertAnswer(t, empty, http.StatusBadRequest, validationBody(`{"field":"reason","message":"Enter a reason."}`))
	assertAnswer(t, long, http.StatusBadRequest, validationBody(`{"field":"reason","message":"Use 500 characters or fewer."}`))
	assertFields(t, unpublished, http.StatusOK, map[string]any{"status": "draft", "version": 2})
	var page struct {
		Items []struct {
			Verb    string `json:"verb"`
			Comment string `json:"comment"`
		} `json:"items"`
	}
	log.Decode(t, &page)
	if len(page.Items) == 0 || page.Items[0].Verb != "unpublished" || page.Items[0].Comment != "Budget under review" {
		t.Errorf("activity log = %s, want the unpublished entry with the reason first", log.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}
