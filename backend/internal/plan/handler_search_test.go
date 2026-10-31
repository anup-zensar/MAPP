package plan_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/plan"
	"mapp/internal/platform/apitest"
)

func TestSearchPlan_FindsLineItemsOnTheDemoData_Contract_039_US1(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	found := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/search?q=VENUE", nil)
	empty := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/search?q=%20", nil, apitest.SkipRequestValidation())
	gone := priya.Do(t, http.MethodGet, "/api/v1/plans/P-9999/search?q=venue", nil)

	var answer plan.PlanSearch
	found.Decode(t, &answer)
	if found.Status != http.StatusOK || !slices.ContainsFunc(answer.Items, func(m plan.SearchMatch) bool {
		return m.Level == plan.LevelLineItem && m.Name == "Venue hire" && strings.HasPrefix(m.ParentID, "A-")
	}) {
		t.Errorf("search venue = %d %s, want the line item Venue hire under its activity", found.Status, found.Body)
	}
	if empty.Status != http.StatusBadRequest || !strings.Contains(string(empty.Body), `"field":"q"`) {
		t.Errorf("empty search = %d %s, want 400 on q", empty.Status, empty.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

func TestPlansList_FoundInTheContents_Contract_039_US2(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/plans?q=venue%20hire", nil)

	var list plan.PlanList
	response.Decode(t, &list)
	i := slices.IndexFunc(list.Items, func(p plan.PlanSummary) bool { return p.ID == "P-1001" })
	if response.Status != http.StatusOK || i < 0 || list.Items[i].FoundIn == nil ||
		list.Items[i].FoundIn.Match.Name != "Venue hire" {
		t.Errorf("plans for venue hire = %d %s, want P-1001 found in Venue hire", response.Status, response.Body)
	}
}
