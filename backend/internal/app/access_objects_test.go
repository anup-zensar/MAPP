package app_test

import (
	"net/http"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
)

// availability is one ActionAvailability of an answer.
type availability struct {
	Available bool
	Reason    string
}

// The planning answers carry what the reader may do with the record, and why not, so the web app shows
// actions without working out a permission itself (FR-011, FR-015). Sara (Viewer) may change nothing in
// the Lubricants plan, with her role named as the reason; Priya (Planner of Lubricants Europe) may change
// everything but may not delete the plan; both see the "From finance" sections. Every answer is validated
// against the contract, so the access objects have the contract's shape.
func TestAnswers_CarryWhatTheReaderMayDo_FR011(t *testing.T) {
	server := apitest.New(t)
	sara, priya := server.SignIn(t, sarasEmail), server.SignIn(t, priyasEmail)
	const viewOnly = "isn't part of your role (Viewer). Ask an administrator if you need it."

	for _, c := range []struct {
		path   string
		action func(body map[string]any) any // finds one availability in the answer
	}{
		{"/api/plans/P-1001", func(b map[string]any) any { return at(b, "access", "edit") }},
		{"/api/plans/P-1001", func(b map[string]any) any { return at(b, "access", "content", "addActivity") }},
		{"/api/plans", func(b map[string]any) any { return at(b, "items", 0, "access", "edit") }},
		{"/api/plans/P-1001/campaigns", func(b map[string]any) any { return at(b, "access", "addCampaign") }},
		{"/api/plans/P-1001/activities", func(b map[string]any) any { return at(b, "access", "editActivities") }},
		{"/api/campaigns/C-1001", func(b map[string]any) any { return at(b, "access", "addChild") }},
		{"/api/initiatives/I-1001", func(b map[string]any) any { return at(b, "access", "addChild") }},
		{"/api/activities/A-1001", func(b map[string]any) any { return at(b, "access", "edit") }},
		{"/api/activities/A-1001/line-items", func(b map[string]any) any { return at(b, "access", "add") }},
		{"/api/line-items/L-1001", func(b map[string]any) any { return at(b, "access", "edit") }},
	} {
		for _, p := range []struct {
			name      string
			session   *apitest.Client
			available bool
		}{{"Sara", sara, false}, {"Priya", priya, true}} {
			var body map[string]any
			response := p.session.Do(t, http.MethodGet, c.path, nil)
			response.Decode(t, &body)
			got, ok := asAvailability(c.action(body))
			if !ok {
				t.Errorf("%s: GET %s has no access object: %s", p.name, c.path, response.Body)
				continue
			}
			if got.Available != p.available || (!got.Available && !strings.HasSuffix(got.Reason, viewOnly)) || (got.Available && got.Reason != "") {
				t.Errorf("%s: GET %s access = %+v, want available %v", p.name, c.path, got, p.available)
			}
		}
	}

	var plan struct {
		Access struct{ Delete availability }
	}
	priya.Do(t, http.MethodGet, "/api/plans/P-1001", nil).Decode(t, &plan)
	want := "Deleting plans isn't part of your role (Planner). Ask an administrator if you need it."
	if plan.Access.Delete.Available || plan.Access.Delete.Reason != want {
		t.Errorf("Priya's delete plan = %+v, want unavailable with %q", plan.Access.Delete, want)
	}
	for _, path := range []string{"/api/activities/A-1001", "/api/activities/A-1001/line-items"} {
		var answer struct {
			Access struct{ SeeFinance availability }
		}
		sara.Do(t, http.MethodGet, path, nil).Decode(t, &answer)
		if !answer.Access.SeeFinance.Available {
			t.Errorf("Sara: GET %s seeFinance = %+v, want available (every default role sees financial data)", path, answer.Access.SeeFinance)
		}
	}
}

// at walks a decoded JSON answer by object keys and array indexes; nil when something is missing.
func at(v any, path ...any) any {
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, ok := v.(map[string]any)
			if !ok {
				return nil
			}
			v = object[key]
		case int:
			list, ok := v.([]any)
			if !ok || key >= len(list) {
				return nil
			}
			v = list[key]
		}
	}
	return v
}

// asAvailability reads a decoded ActionAvailability.
func asAvailability(v any) (availability, bool) {
	object, ok := v.(map[string]any)
	if !ok {
		return availability{}, false
	}
	available, okAvailable := object["available"].(bool)
	reason, okReason := object["reason"].(string)
	return availability{Available: available, Reason: reason}, okAvailable && okReason
}
