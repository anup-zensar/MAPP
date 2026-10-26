package app_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// Seeded roles: Alex Administrator, Carlos Global planner, Daniel Planner and Approver, Priya and Mei
// Planners, Sara Viewer; Lubricants Europe: Priya, Daniel, Sara; Mobility China: Mei, Carlos, Sara.
const (
	alexsEmail   = "alex.morgan@shell.com"
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meisEmail    = "mei.lin@shell.com"
	carlosEmail  = "carlos.mendes@shell.com"
	sarasEmail   = "sara.novak@shell.com"
)

type errorAnswer struct {
	Error struct {
		Code, Message string
		Fields        []struct{ Field, Message string }
	}
}

func refused(t *testing.T, what string, r *apitest.Response, message string) {
	t.Helper()
	var answer errorAnswer
	r.Decode(t, &answer)
	if e := answer.Error; r.Status != http.StatusForbidden || e.Code != "forbidden" || e.Message != message {
		t.Errorf("%s = %d %s\nwant 403 forbidden %q", what, r.Status, r.Body, message)
	}
}

func hidden(t *testing.T, what string, r *apitest.Response, record string) {
	t.Helper()
	want := `{"error":{"code":"not_found","message":"This ` + record + ` no longer exists — someone may have deleted it.","fields":[]}}`
	if r.Status != http.StatusNotFound || string(r.Body) != want {
		t.Errorf("%s = %d %s\nwant 404 %s", what, r.Status, r.Body, want)
	}
}

func itemIDs(t *testing.T, r *apitest.Response) []string {
	t.Helper()
	var page struct{ Items []struct{ ID string } }
	r.Decode(t, &page)
	list := []string{}
	for _, item := range page.Items {
		list = append(list, item.ID)
	}
	return list
}

func planForm(name string, teamIDs ...string) map[string]any {
	return map[string]any{
		"name": name, "year": 2026, "modelId": "MOD-1001", "teamIds": teamIDs, "ownerId": nil, "aop": false,
		"description": "", "budget": nil, "defaultFinanceBudgetId": nil, "defaultSpread": "evenly", "bcpir": false,
		"goalStage": "inq", "goalAmount": "", "attributes": map[string]any{},
	}
}
