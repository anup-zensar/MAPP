package allwork_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/allwork"
	"mapp/internal/platform/apitest"
)

// The remembered columns and sort are read and saved per person (FR-019, FR-021): the defaults first, then
// what Priya saved; Daniel still has the defaults; a wrong choice is refused.
func TestPreferences_GetPut_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans
	daniel := server.SignIn(t, danielsEmail)
	defaults := `{"columns":["id","status","owner","startDate","endDate","planBudget","plannedCost","lineItems","unassigned",` +
		`"committed","invoiced"],"sort":null}`

	before := priya.Do(t, http.MethodGet, "/api/v1/all-work/preferences", nil)
	saved := priya.Do(t, http.MethodPut, "/api/v1/all-work/preferences",
		map[string]any{"columns": []string{"plannedCost", "id"}, "sort": map[string]string{"column": "plannedCost", "direction": "desc"}})
	after := priya.Do(t, http.MethodGet, "/api/v1/all-work/preferences", nil)
	daniels := daniel.Do(t, http.MethodGet, "/api/v1/all-work/preferences", nil)
	refused := priya.Do(t, http.MethodPut, "/api/v1/all-work/preferences", map[string]any{"columns": []string{"id", "id"}, "sort": nil})

	assertAnswer(t, before, http.StatusOK, defaults)
	want := `{"columns":["plannedCost","id"],"sort":{"column":"plannedCost","direction":"desc"}}`
	assertAnswer(t, saved, http.StatusOK, want)
	assertAnswer(t, after, http.StatusOK, want)
	assertAnswer(t, daniels, http.StatusOK, defaults)
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"columns","message":"Each column can be chosen once."}]}}`)
	// The grid in the remembered order, sorted, as the web app asks for it.
	sorted := priya.Do(t, http.MethodGet, "/api/v1/all-work?column=plannedCost&column=id&sort=plannedCost:desc", nil)
	var r allwork.Result
	sorted.Decode(t, &r)
	var plans []string
	for _, row := range r.Rows {
		if row.Level == allwork.LevelPlan {
			plans = append(plans, row.Name)
		}
	}
	if !slices.Equal(plans, []string{"FY2026 Lubricants Europe", "FY2026 Mobility China"}) || len(r.Columns) != 2 {
		t.Errorf("plans by planned cost, highest first = %q", plans)
	}
}

// newViewBody is the body of a new saved view: "Lubricants by month", Budget & cost of FY2026 Lubricants
// Europe by month.
func newViewBody(name string, isDefault bool) map[string]any {
	return map[string]any{"name": name, "isDefault": isDefault, "state": map[string]any{
		"view": "budget", "year": 2026, "search": "", "columns": []string{"id", "status"}, "sort": nil,
		"filters": map[string]any{
			"plans": []map[string]string{{"id": "P-1001", "label": "FY2026 Lubricants Europe"}}, "planStatuses": []string{},
			"containerStatuses": []string{}, "activityStatuses": []string{}, "owners": []any{}, "teams": []any{},
			"businessUnits": []any{}, "activityTypes": []any{}, "strategic": "include", "showLevels": "everything",
		},
		"expansion": map[string]any{"mode": "default", "toggled": []string{}}, "showUnassigned": true,
		"period": "month", "calendarLayout": "timeline",
	}}
}

// Saved views are created (201), listed by name, changed, made the default and deleted by their owner; for
// anyone else they are not found (FR-029 – FR-031, US6).
func TestSavedViews_CRUD_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)

	created := priya.Do(t, http.MethodPost, "/api/v1/saved-views", newViewBody("Helix by month", false))
	refused := priya.Do(t, http.MethodPost, "/api/v1/saved-views", newViewBody("helix BY month", false))
	var view allwork.SavedView
	created.Decode(t, &view)
	updated := priya.Do(t, http.MethodPatch, "/api/v1/saved-views/"+view.ID, map[string]any{"name": "Helix monthly", "isDefault": true})
	listed := priya.Do(t, http.MethodGet, "/api/v1/saved-views", nil)
	danielsList := daniel.Do(t, http.MethodGet, "/api/v1/saved-views", nil)
	danielsPatch := daniel.Do(t, http.MethodPatch, "/api/v1/saved-views/"+view.ID, map[string]any{"name": "Mine"})
	danielsDelete := daniel.Do(t, http.MethodDelete, "/api/v1/saved-views/"+view.ID, nil)
	deleted := priya.Do(t, http.MethodDelete, "/api/v1/saved-views/"+view.ID, nil)
	again := priya.Do(t, http.MethodDelete, "/api/v1/saved-views/"+view.ID, nil)

	if created.Status != http.StatusCreated || view.ID != "SV-1003" || view.Name != "Helix by month" ||
		view.State.View != "budget" || len(view.State.Filters.Plans) != 1 {
		t.Errorf("created = %d %s", created.Status, created.Body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"name","message":"You already have a view called 'Helix by month'."}]}}`)
	var after allwork.SavedView
	updated.Decode(t, &after)
	if updated.Status != http.StatusOK || after.Name != "Helix monthly" || !after.IsDefault {
		t.Errorf("updated = %d %s", updated.Status, updated.Body)
	}
	var page allwork.SavedViewPage
	listed.Decode(t, &page)
	var names []string
	for _, v := range page.Items {
		names = append(names, v.Name)
	}
	if !slices.Equal(names, []string{"Declined activities", "Helix monthly", "Lubricants by month"}) || page.Total != 3 {
		t.Errorf("Priya's views (two of the seed) = %q", names)
	}
	assertAnswer(t, danielsList, http.StatusOK, `{"items":[],"total":0}`)
	gone := `{"error":{"code":"not_found","message":"This view no longer exists — it may have been deleted in another tab.","fields":[]}}`
	assertAnswer(t, danielsPatch, http.StatusNotFound, gone)
	assertAnswer(t, danielsDelete, http.StatusNotFound, gone)
	if deleted.Status != http.StatusOK {
		t.Errorf("deleted = %d %s", deleted.Status, deleted.Body)
	}
	assertAnswer(t, again, http.StatusNotFound, gone)
}

// Saved views and the remembered columns are personal settings, not business records (D-121): saving,
// changing and deleting them writes no activity-log entry and nothing to the recycle bin.
func TestViews_NotLogged_D121(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	logBefore := priya.Do(t, http.MethodGet, "/api/v1/activity-log", nil)
	binBefore := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil)

	created := priya.Do(t, http.MethodPost, "/api/v1/saved-views", newViewBody("Not logged", true))
	var view allwork.SavedView
	created.Decode(t, &view)
	priya.Do(t, http.MethodPatch, "/api/v1/saved-views/"+view.ID, map[string]any{"name": "Still not logged"})
	priya.Do(t, http.MethodDelete, "/api/v1/saved-views/"+view.ID, nil)
	priya.Do(t, http.MethodPut, "/api/v1/all-work/preferences", map[string]any{"columns": []string{"id"}, "sort": nil})

	if logAfter := priya.Do(t, http.MethodGet, "/api/v1/activity-log", nil); string(logAfter.Body) != string(logBefore.Body) {
		t.Errorf("activity log changed:\nbefore %s\nafter  %s", logBefore.Body, logAfter.Body)
	}
	if binAfter := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil); string(binAfter.Body) != string(binBefore.Body) {
		t.Errorf("recycle bin changed:\nbefore %s\nafter  %s", binBefore.Body, binAfter.Body)
	}
}
