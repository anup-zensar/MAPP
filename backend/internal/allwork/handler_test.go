package allwork_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/allwork"
	"mapp/internal/platform/apitest"
)

// The handler tests run the real API on the demo data (apitest, clock at 12 Mar 2026), and every request and
// answer is checked against the contract. 2026 has two plans: P-1001 FY2026 Lubricants Europe (Published)
// and P-1002 FY2026 Mobility China (Draft); P-1003 FY2027 Global Brand is in 2027.

const (
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
)

// The default view is the grid of the current year: both 2026 plans with everything below them, the cards,
// the count and the totals row, which equals the cards (FR-001, FR-003, FR-011, FR-014).
func TestGetAllWork_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	response := priya.Do(t, http.MethodGet, "/api/v1/all-work", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/all-work = %d %s", response.Status, response.Body)
	}
	var r allwork.Result
	response.Decode(t, &r)
	var plans []string
	levels := map[allwork.Level]int{}
	for _, row := range r.Rows {
		levels[row.Level]++
		if row.Level == allwork.LevelPlan {
			plans = append(plans, row.Name)
		}
	}
	if !slices.Equal(plans, []string{"FY2026 Lubricants Europe", "FY2026 Mobility China"}) || r.Year != 2026 ||
		r.View != "grid" || r.DisplayCurrency != "USD" || len(r.Columns) != 11 || r.Today.String() != "2026-03-12" {
		t.Errorf("plans %q, year %d, view %s, currency %s, %d columns, today %s", plans, r.Year, r.View,
			r.DisplayCurrency, len(r.Columns), r.Today)
	}
	if levels[allwork.LevelCampaign] != 3 || levels[allwork.LevelInitiative] != 6 || levels[allwork.LevelActivity] != 10 ||
		levels[allwork.LevelLineItem] != 8 || levels[allwork.LevelUnassigned] == 0 {
		t.Errorf("rows per level = %v", levels)
	}
	if r.Summary.Activities != 10 || r.Count.Shown != 10 || r.Count.Total != 10 || r.Count.Noun != "activities" ||
		r.Totals.Activities != 10 || r.Summary.PlannedCost == nil || r.Totals.PlannedCost == nil ||
		r.Summary.PlannedCost.Amount != r.Totals.PlannedCost.Amount || r.Summary.PlanBudget.Amount.String() != "2000000.00" {
		t.Errorf("summary %+v, count %+v, totals %d activities", r.Summary, r.Count, r.Totals.Activities)
	}
	// The other views answer in the contract's shape too: the calendar without line items and grid columns.
	calendar := priya.Do(t, http.MethodGet, "/api/v1/all-work?view=calendar&year=2026", nil)
	var c allwork.Result
	calendar.Decode(t, &c)
	for _, row := range c.Rows {
		if row.Level == allwork.LevelLineItem || row.Level == allwork.LevelUnassigned {
			t.Errorf("calendar row %s is a %s", row.Key, row.Level)
		}
	}
	// A query in the order the person chose its columns keeps that order.
	ordered := priya.Do(t, http.MethodGet, "/api/v1/all-work?column=plannedCost&column=id&column=createdAt", nil)
	var o allwork.Result
	ordered.Decode(t, &o)
	var keys []string
	for _, col := range o.Columns {
		keys = append(keys, col.Key)
	}
	if strings.Join(keys, ",") != "plannedCost,id,createdAt" {
		t.Errorf("columns = %q", keys)
	}
	// A year without plans answers no rows.
	empty := priya.Do(t, http.MethodGet, "/api/v1/all-work?year=2030", nil)
	var e allwork.Result
	empty.Decode(t, &e)
	if len(e.Rows) != 0 || e.Summary.PlannedCost != nil || e.Count.Total != 0 {
		t.Errorf("2030 = %+v", e)
	}
}

// A row's record is located before it opens (FR-016, D-123): a stored record answers its level, name and
// plan; a deleted one answers 404 with what became of it.
func TestLocate_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	found := priya.Do(t, http.MethodGet, "/api/v1/all-work/records/L-1002", nil)
	deleted := priya.Do(t, http.MethodDelete, "/api/v1/campaigns/C-1002", nil)
	inBin := priya.Do(t, http.MethodGet, "/api/v1/all-work/records/A-1005", nil) // Fleet roadshow, under Rimula fleet
	unknown := priya.Do(t, http.MethodGet, "/api/v1/all-work/records/I-9999", nil)

	assertAnswer(t, found, http.StatusOK,
		`{"recordId":"L-1002","level":"lineItem","name":"Film production","planId":"P-1001","activityId":"A-1001"}`)
	if deleted.Status != http.StatusOK {
		t.Fatalf("delete C-1002 = %d %s", deleted.Status, deleted.Body)
	}
	assertAnswer(t, inBin, http.StatusNotFound,
		`{"error":{"code":"not_found","message":"This activity was deleted — it is in the recycle bin","fields":[]}}`)
	assertAnswer(t, unknown, http.StatusNotFound,
		`{"error":{"code":"not_found","message":"This initiative no longer exists — someone may have deleted it.","fields":[]}}`)
}

// Every filter parameter of the contract is accepted, together, and the answer follows them (FR-004):
// Priya's declined activities in the published plans of Lubricants Europe.
func TestGetAllWork_EveryFilterParameter_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	query := "/api/v1/all-work?year=2026&view=grid&plan=P-1001&planStatus=published&containerStatus=inProgress" +
		"&activityStatus=declined&owner=U-1002&owner=U-1003&team=T-1001&businessUnit=SO-1002&strategic=exclude" +
		"&showLevels=completeBranches&search=fleet&column=id&column=status&sort=plannedCost:desc&period=month"

	response := priya.Do(t, http.MethodGet, query, nil)

	if response.Status != http.StatusOK {
		t.Fatalf("GET = %d %s", response.Status, response.Body)
	}
	var r allwork.Result
	response.Decode(t, &r)
	var activities []string
	for _, row := range r.Rows {
		if row.Level == allwork.LevelActivity && row.Role == allwork.RoleMatch {
			activities = append(activities, row.Name)
		}
	}
	if !slices.Equal(activities, []string{"Fleet roadshow"}) || r.Count.Shown != 1 || len(r.LeftOut) != 0 {
		t.Errorf("matching activities %q, count %+v, left out %+v", activities, r.Count, r.LeftOut)
	}
}

// The filter bar's choices (FR-004): the years, the plans and owners of the year, the teams with the
// person's own first, the business units, the activity types and the column catalogue.
func TestGetOptions_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	response := priya.Do(t, http.MethodGet, "/api/v1/all-work/options", nil)
	wrongYear := priya.Do(t, http.MethodGet, "/api/v1/all-work/options?year=3000", nil, apitest.SkipRequestValidation())

	if response.Status != http.StatusOK {
		t.Fatalf("GET options = %d %s", response.Status, response.Body)
	}
	var o allwork.Options
	response.Decode(t, &o)
	var plans, owners []string
	for _, p := range o.Plans {
		plans = append(plans, p.Label)
	}
	for _, p := range o.Owners {
		owners = append(owners, p.Label)
	}
	if o.Year != 2026 || o.CurrentYear != 2026 || !slices.Equal(o.Years, []int{2027, 2026}) ||
		!slices.Equal(plans, []string{"FY2026 Lubricants Europe", "FY2026 Mobility China"}) ||
		!slices.Contains(owners, "Daniel Okoro") || len(o.Teams) == 0 || !o.Teams[0].Mine ||
		o.Teams[0].Label != "Lubricants Europe" || len(o.BusinessUnits) == 0 || len(o.ActivityTypes) == 0 ||
		len(o.ColumnGroups) == 0 || len(o.DefaultColumns) != 11 {
		t.Errorf("options = %s", response.Body)
	}
	assertAnswer(t, wrongYear, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"year","message":"Choose a year from 2000 to 2100."}]}}`)
}

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

// Export answers the file of the view, .xlsx or CSV, named in Content-Disposition after the view, year and
// date; a wrong format is refused (FR-032 – FR-034).
func TestExport_XlsxAndCsvContentTypeAndDisposition_FR034(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: every market's plans

	xlsx := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=xlsx&activityStatus=declined", nil)
	csv := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=csv&view=budget&unassigned=false", nil)
	refused := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=pdf&unassigned=maybe", nil, apitest.SkipRequestValidation())

	if xlsx.Status != http.StatusOK || xlsx.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
		xlsx.Header.Get("Content-Disposition") != `attachment; filename="MAPP All work - Grid - 2026 - 12 Mar 2026.xlsx"; `+
			`filename*=UTF-8''MAPP%20All%20work%20%E2%80%93%20Grid%20%E2%80%93%202026%20%E2%80%93%2012%20Mar%202026.xlsx` ||
		!strings.HasPrefix(string(xlsx.Body), "PK") {
		t.Errorf("xlsx = %d %v", xlsx.Status, xlsx.Header)
	}
	body := string(csv.Body)
	if csv.Status != http.StatusOK || csv.Header.Get("Content-Type") != "text/csv" ||
		!strings.Contains(csv.Header.Get("Content-Disposition"), `filename="MAPP All work - Budget & cost - 2026 - 12 Mar 2026.csv"`) ||
		!strings.HasPrefix(body, "\xEF\xBB\xBFView,Budget & cost\nYear,2026\nFilters,None\n") ||
		!strings.Contains(body, "Exported by,Priya Raman\n") || strings.Contains(body, "Unassigned planned cost (Sys_Gen_Balance)") ||
		!strings.Contains(body, "Total · 10 activities") {
		t.Errorf("csv = %d %v\n%s", csv.Status, csv.Header, body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"format","message":"Choose one of the listed values."},`+
		`{"field":"unassigned","message":"Choose one of the listed values."}]}}`)
}

// assertAnswer checks an answer's status and exact body.
func assertAnswer(t *testing.T, r *apitest.Response, status int, body string) {
	t.Helper()
	if r.Status != status || string(r.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", r.Status, r.Body, status, body)
	}
}
