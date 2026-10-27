package budgets_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/budgets"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

const (
	carlosEmail = "carlos.mendes@shell.com"
	priyaEmail  = "priya.raman@shell.com"
	meiEmail    = "mei.lin@shell.com"
)

const (
	budgetGoneBody     = `{"error":{"code":"not_found","message":"This global budget no longer exists — someone may have deleted it.","fields":[]}}`
	allocationGoneBody = `{"error":{"code":"not_found","message":"This allocation no longer exists — someone may have removed it.","fields":[]}}`
)

// demoFigures is the figures text of the demo global budget, with its plans' planned cost as the plan pages
// show it.
func demoFigures(t *testing.T, server *apitest.Server) string {
	t.Helper()
	carlos := server.SignIn(t, carlosEmail)
	var view budgets.GlobalBudgetView
	decode(t, carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1001", nil), http.StatusOK, &view)
	return figuresText(view.Figures)
}

// planForm is the edit form of the plan id as the edit page sends it unchanged.
func planForm(t *testing.T, c *apitest.Client, id string) map[string]any {
	t.Helper()
	var plan map[string]any
	if err := json.Unmarshal(c.Do(t, http.MethodGet, "/api/v1/plans/"+id, nil).Body, &plan); err != nil {
		t.Fatalf("plan %s: %v", id, err)
	}
	form := map[string]any{}
	for _, field := range []string{"version", "name", "aop", "description", "budget", "defaultSpread", "bcpir",
		"goalStage", "goalAmount", "attributes"} {
		form[field] = plan[field]
	}
	form["modelId"] = plan["model"].(map[string]any)["id"]
	form["ownerId"] = nil
	if owner, ok := plan["owner"].(map[string]any); ok {
		form["ownerId"] = owner["id"]
	}
	form["defaultFinanceBudgetId"] = nil
	if fb, ok := plan["defaultFinanceBudget"].(map[string]any); ok {
		form["defaultFinanceBudgetId"] = fb["id"]
	}
	var teamIDs []any
	for _, team := range plan["teams"].([]any) {
		teamIDs = append(teamIDs, team.(map[string]any)["id"])
	}
	form["teamIds"] = teamIDs
	return form
}

// usd is a money value in US dollars as the API sends and receives it.
func usd(amount string) map[string]any {
	return map[string]any{"amount": amount, "currency": "USD"}
}

// mustAmount parses a money amount and stops the test when it is not one.
func mustAmount(t *testing.T, text string) money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return a
}

// validation is the answer of a form with the field problems fields (a JSON list without brackets).
func validation(fields string) string {
	return `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}}`
}

// figuresText writes figures as "<amount> allocated <a> unallocated <u> planned <p>".
func figuresText(f budgets.Figures) string {
	return fmt.Sprintf("%s allocated %s unallocated %s planned %s", f.Amount.Format(), f.Allocated.Format(), f.Unallocated.Format(), f.PlannedCost.Format())
}

// changesText writes log changes as "<label>: <old> → <new>", joined by "; ".
func changesText(changes []history.Change) string {
	var parts []string
	for _, c := range changes {
		parts = append(parts, fmt.Sprintf("%s: %s → %s", c.Label, c.OldText, c.NewText))
	}
	return strings.Join(parts, "; ")
}

// decode checks the status of an answer and reads its JSON body into v.
func decode(t *testing.T, response *apitest.Response, status int, v any) {
	t.Helper()
	if response.Status != status {
		t.Fatalf("answer = %d %s, want %d", response.Status, response.Body, status)
	}
	response.Decode(t, v)
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, body)
	}
}

// assertJSON checks the status of an answer and that its body equals want as JSON.
func assertJSON(t *testing.T, response *apitest.Response, status int, want any) {
	t.Helper()
	var got, expected any
	encoded, _ := json.Marshal(want)
	_ = json.Unmarshal(encoded, &expected)
	if err := json.Unmarshal(response.Body, &got); err != nil || response.Status != status || !reflect.DeepEqual(got, expected) {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, encoded)
	}
}

// assertFields checks the status of an answer and the fields of its JSON object named in want.
func assertFields(t *testing.T, response *apitest.Response, status int, want map[string]any) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(response.Body, &body); err != nil || response.Status != status {
		t.Fatalf("answer = %d %s, want %d", response.Status, response.Body, status)
	}
	for field, value := range want {
		var expected any
		encoded, _ := json.Marshal(value)
		_ = json.Unmarshal(encoded, &expected)
		if !reflect.DeepEqual(body[field], expected) {
			got, _ := json.Marshal(body[field])
			t.Errorf("%s = %s, want %s", field, got, encoded)
		}
	}
}
