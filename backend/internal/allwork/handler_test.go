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

// The default view is the grid of the current year, and its totals row equals the cards.
func TestGetAllWork_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all")

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
	calendar := priya.Do(t, http.MethodGet, "/api/v1/all-work?view=calendar&year=2026", nil)
	var c allwork.Result
	calendar.Decode(t, &c)
	for _, row := range c.Rows {
		if row.Level == allwork.LevelLineItem || row.Level == allwork.LevelUnassigned {
			t.Errorf("calendar row %s is a %s", row.Key, row.Level)
		}
	}
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
	empty := priya.Do(t, http.MethodGet, "/api/v1/all-work?year=2030", nil)
	var e allwork.Result
	empty.Decode(t, &e)
	if len(e.Rows) != 0 || e.Summary.PlannedCost != nil || e.Count.Total != 0 {
		t.Errorf("2030 = %+v", e)
	}
}

// A deleted record answers 404 with what became of it.
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

// Every filter parameter at once: Priya's declined activities in the published plans of Lubricants Europe.
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

func TestGetOptions_Contract(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all")

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

func TestExport_XlsxAndCsvContentTypeAndDisposition_FR034(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all")

	xlsx := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=xlsx&activityStatus=declined", nil)
	csv := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=csv&view=budget&unassigned=false", nil)
	refused := priya.Do(t, http.MethodGet, "/api/v1/all-work/export?format=pdf&unassigned=maybe", nil, apitest.SkipRequestValidation())

	if xlsx.Status != http.StatusOK || xlsx.Header.Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
		xlsx.Header.Get("Content-Disposition") != `attachment; filename="SP&A One All work - Grid - 2026 - 12 Mar 2026.xlsx"; `+
			`filename*=UTF-8''SP%26A%20One%20All%20work%20%E2%80%93%20Grid%20%E2%80%93%202026%20%E2%80%93%2012%20Mar%202026.xlsx` ||
		!strings.HasPrefix(string(xlsx.Body), "PK") {
		t.Errorf("xlsx = %d %v", xlsx.Status, xlsx.Header)
	}
	body := string(csv.Body)
	if csv.Status != http.StatusOK || csv.Header.Get("Content-Type") != "text/csv" ||
		!strings.Contains(csv.Header.Get("Content-Disposition"), `filename="SP&A One All work - Budget & cost - 2026 - 12 Mar 2026.csv"`) ||
		!strings.HasPrefix(body, "\xEF\xBB\xBFView,Budget & cost\nYear,2026\nFilters,None\n") ||
		!strings.Contains(body, "Exported by,Priya Raman\n") || strings.Contains(body, "Unassigned planned cost (Sys_Gen_Balance)") ||
		!strings.Contains(body, "Total · 10 activities") {
		t.Errorf("csv = %d %v\n%s", csv.Status, csv.Header, body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"format","message":"Choose one of the listed values."},`+
		`{"field":"unassigned","message":"Choose one of the listed values."}]}}`)
}

func assertAnswer(t *testing.T, r *apitest.Response, status int, body string) {
	t.Helper()
	if r.Status != status || string(r.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", r.Status, r.Body, status, body)
	}
}
