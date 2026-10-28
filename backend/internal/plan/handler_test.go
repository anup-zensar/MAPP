package plan_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"mapp/internal/plan"
	"mapp/internal/platform/apitest"
)

// The handler tests run the real API on the demo data (apitest), every request and answer checked against the contract.
// Seed plans: P-1001 FY2026 Lubricants Europe (Published), P-1002 FY2026 Mobility China, P-1003 FY2027 Global Brand and
// P-1004 FY2027 Lubricants UK & Ireland, planned in GBP.

const (
	priyasEmail  = "priya.raman@shell.com"
	danielsEmail = "daniel.okoro@shell.com"
	meisEmail    = "mei.lin@shell.com"
)

const planGoneBody = `{"error":{"code":"not_found","message":"This plan no longer exists — someone may have deleted it.","fields":[]}}`

// newPlanBody is FY2026 Lubricants Test: Lubricants 1.0, Lubricants Europe, Priya as owner, business unit Lubricants.
func newPlanBody() map[string]any {
	return map[string]any{
		"name": "FY2026 Lubricants Test", "year": 2026, "modelId": "MOD-1001", "teamIds": []string{"T-1001"},
		"ownerId": "U-1002", "aop": false, "description": "", "budget": nil, "defaultFinanceBudgetId": nil,
		"defaultSpread": "evenly", "bcpir": false, "goalStage": "inq", "goalAmount": "250000",
		"attributes": map[string]any{"SA-1001": []string{"SO-1002"}},
	}
}

// The seed has two active models, so a new plan's model is not chosen.
func TestPlanDefaults_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/plan-defaults", nil)

	assertAnswer(t, response, http.StatusOK, `{"currentYear":2026,"years":[2026,2027,2028,2029,2030,2031,2032,2033,2034,2035,2036],`+
		`"year":2026,"teamIds":["T-1001"],"ownerId":"U-1002","modelId":null,"aop":false,"bcpir":false,`+
		`"defaultSpread":"evenly","goalStage":"inq","currency":"USD","planType":"operational","overspend":"warn"}`)
}

// The new plan is P-1005, after the seed's four; a form without name, year and model gets every problem at once.
func TestCreatePlan_201AndValidation_FR010(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	missing := newPlanBody()
	missing["name"], missing["year"], missing["modelId"] = " ", nil, nil

	created := priya.Do(t, http.MethodPost, "/api/v1/plans", newPlanBody())
	refused := priya.Do(t, http.MethodPost, "/api/v1/plans", missing)

	if created.Status != http.StatusCreated {
		t.Fatalf("create = %d %s, want 201", created.Status, created.Body)
	}
	var p plan.PlanView
	created.Decode(t, &p)
	if p.ID != "P-1005" || p.Status != plan.StatusDraft || p.Name != "FY2026 Lubricants Test" ||
		p.Owner == nil || p.Owner.Name != "Priya Raman" || len(p.Completeness.Missing) != 0 ||
		p.Goals.Plan.MQL == nil || *p.Goals.Plan.MQL != 5000 || p.Goals.Plan.Revenue == nil ||
		p.Goals.Plan.Revenue.Amount.String() != "2500000.00" || p.CreatedBy.Name != "Priya Raman" {
		t.Errorf("created plan = %s", created.Body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a plan name."},`+
		`{"field":"year","message":"Choose a year from 2026 to 2036."},{"field":"modelId","message":"Choose a model."}`))
}

func validationBody(fields string) string {
	return `{"error":{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}}`
}

func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, body)
	}
}

func assertJSON(t *testing.T, response *apitest.Response, status int, want any) {
	t.Helper()
	var got, expected any
	encoded, _ := json.Marshal(want)
	_ = json.Unmarshal(encoded, &expected)
	if err := json.Unmarshal(response.Body, &got); err != nil || response.Status != status || !reflect.DeepEqual(got, expected) {
		t.Errorf("answer = %d %s\nwant     %d %s", response.Status, response.Body, status, encoded)
	}
}

func listOf(t *testing.T, response *apitest.Response) plan.PlanList {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("list = %d %s, want 200", response.Status, response.Body)
	}
	var list plan.PlanList
	response.Decode(t, &list)
	return list
}

func listIDs(list plan.PlanList) []string {
	planIDs := []string{}
	for _, item := range list.Items {
		planIDs = append(planIDs, item.ID)
	}
	return planIDs
}

// Plans come newest year first, then by name; the search finds a name or an ID in any letter case.
func TestListPlans_SeedOrderSummaryAndFilters_FR001_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all")

	all := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans", nil))
	draft2026 := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans?status=draft&year=2026", nil))
	byName := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans?q=mobility", nil))
	byID := listOf(t, priya.Do(t, http.MethodGet, "/api/v1/plans?q=p-1002", nil))
	wrong := priya.Do(t, http.MethodGet, "/api/v1/plans?status=open&year=next", nil, apitest.SkipRequestValidation())

	// P-1004's 600,000.00 GBP budget is 759,493.67 USD and its 65,000.00 GBP planned 82,278.48 USD: each plan converted once.
	if ids := listIDs(all); !reflect.DeepEqual(ids, []string{"P-1003", "P-1004", "P-1001", "P-1002"}) || all.Total != 4 ||
		!reflect.DeepEqual(all.Years, []int{2027, 2026}) {
		t.Errorf("all plans: %q, total %d, years %v", ids, all.Total, all.Years)
	}
	summary := all.Summary
	if summary.Plans != 4 || summary.Draft != 3 || summary.Published != 1 || summary.PlannedCost == nil ||
		summary.PlannedCost.Format() != "732,278.48 USD" || summary.Budget == nil || summary.Budget.Format() != "3,259,493.67 USD" {
		t.Errorf("summary = %+v; want 4 plans, 3 Draft, 1 Published, 732,278.48 USD planned, 3,259,493.67 USD", summary)
	}
	for name, list := range map[string]plan.PlanList{"draft 2026": draft2026, "mobility": byName, "p-1002": byID} {
		if ids := listIDs(list); !reflect.DeepEqual(ids, []string{"P-1002"}) || list.Total != 4 {
			t.Errorf("%s: %q, total %d; want P-1002 of 4", name, ids, list.Total)
		}
	}
	assertAnswer(t, wrong, http.StatusBadRequest, validationBody(`{"field":"status","message":"Choose draft or published."},`+
		`{"field":"year","message":"Choose a year such as 2026."}`))
}

// FY2026 Mobility China lacks its business unit and lists the incomplete records below it; the figures from the levels
// below are proved with fakes in the service tests.
func TestGetPlan_ViewAnd404_FR011(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all")

	europe := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001", nil)
	china := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/plans/P-9999", nil)

	system := map[string]any{"id": "system", "name": "SP&A One"}
	usdOf := func(amount string) map[string]any { return map[string]any{"amount": amount, "currency": "USD"} }
	assertFields(t, europe, http.StatusOK, map[string]any{
		"id": "P-1001", "name": "FY2026 Lubricants Europe", "year": 2026, "status": "published",
		"model": map[string]any{"id": "MOD-1001", "name": "Lubricants 1.0", "active": true, "inqToMqlRate": "2.00",
			"mqlToDealRate": "5.00", "averageDealSize": usdOf("10000.00")},
		"teams":        []any{map[string]any{"id": "T-1001", "name": "Lubricants Europe", "colour": "blue", "active": true}},
		"owner":        map[string]any{"id": "U-1002", "name": "Priya Raman", "initials": "PR", "jobTitle": "Marketer", "active": true},
		"ownerProblem": "", "aop": false, "description": "Lubricants marketing in Europe for 2026.",
		"budget":               usdOf("1200000.00"),
		"defaultFinanceBudget": map[string]any{"id": "FB-1001", "year": 2026, "name": "2026 Lubricants Budget"},
		"defaultSpread":        "evenly", "bcpir": false, "goalStage": "inq", "goalAmount": "250000",
		"attributes":   map[string]any{"SA-1001": []any{"SO-1002"}},
		"businessUnit": []any{map[string]any{"id": "SO-1002", "label": "Lubricants", "active": true}},
		"version":      1, "createdAt": "2026-03-12T09:00:00Z", "createdBy": system, "updatedAt": "2026-03-12T09:00:00Z", "updatedBy": system,
	})
	var p plan.PlanView
	europe.Decode(t, &p)
	goal := p.Goals.Plan
	if goal.INQ == nil || *goal.INQ != 250_000 || goal.MQL == nil || *goal.MQL != 5_000 || goal.Deals == nil || *goal.Deals != 250 ||
		goal.Revenue == nil || goal.Revenue.Format() != "2,500,000.00 USD" || p.Goals.Stage != plan.StageINQ {
		t.Errorf("goals = %+v", p.Goals)
	}
	if len(p.Completeness.Missing) != 0 || p.Figures.Currency != "USD" || p.Figures.Budget.Format() != "1,200,000.00 USD" {
		t.Errorf("completeness %+v, figures %+v; want all filled in, in USD", p.Completeness, p.Figures)
	}
	assertFields(t, china, http.StatusOK, map[string]any{"status": "draft", "businessUnit": []any{}})
	var chinaPlan plan.PlanView
	china.Decode(t, &chinaPlan)
	if missing := chinaPlan.Completeness.Missing; len(missing) != 1 || missing[0].Field != "attributes.SA-1001" || missing[0].Label != "Business unit" {
		t.Errorf("completeness missing = %+v, want the Business unit", missing)
	}
	var below []string
	for _, item := range chinaPlan.Completeness.ItemsBelow {
		below = append(below, item.Record.Type+" "+item.Record.ID)
	}
	if want := []string{"campaign C-1003", "initiative I-1006", "activity A-1008", "activity A-1010", "lineItem L-1007"}; !slices.Equal(below, want) {
		t.Errorf("completeness items below = %q, want %q", below, want)
	}
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

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

func TestCopyOptions_FR026(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	options := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001/copy-options", nil)
	gone := priya.Do(t, http.MethodGet, "/api/v1/plans/P-9999/copy-options", nil)

	assertAnswer(t, options, http.StatusOK, `{"name":"FY2026 Lubricants Europe (copy)","sameYear":2026,`+
		`"nextYear":{"year":2027,"available":true,"reason":""},`+
		`"contents":{"campaigns":2,"initiatives":4,"activities":7,"approvedActivities":4,"lineItems":6}}`)
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// The copy takes every record below with new IDs, every date a year later and every activity Created, in one log entry;
// a name taken in other letters and a wrong year answer 400 with both problems.
func TestCopy_201_FR027_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	copied := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/copy",
		map[string]any{"name": "FY2027 Lubricants Europe", "year": 2027, "includeContents": true})
	refused := priya.Do(t, http.MethodPost, "/api/v1/plans/P-1001/copy",
		map[string]any{"name": "fy2026 mobility china", "year": 2025, "includeContents": false})
	gone := priya.Do(t, http.MethodPost, "/api/v1/plans/P-9999/copy",
		map[string]any{"name": "Copy", "year": 2026, "includeContents": false})

	var result plan.CopyResult
	copied.Decode(t, &result)
	p := result.Plan
	if copied.Status != http.StatusCreated || p.ID != "P-1005" || p.Year != 2027 || p.Status != plan.StatusDraft ||
		p.DefaultFinanceBudget != nil || p.Contents != (plan.Contents{Campaigns: 2, Initiatives: 4, Activities: 7, LineItems: 6}) ||
		p.CreatedBy.Name != "Priya Raman" {
		t.Errorf("copy = %d %s", copied.Status, copied.Body)
	}
	if want := []string{"Default finance budget cleared — it belonged to 2026."}; !slices.Equal(result.Notes, want) {
		t.Errorf("notes = %q, want %q", result.Notes, want)
	}
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(
		`{"field":"name","message":"There is already a plan called 'FY2026 Mobility China'. Choose another name."},`+
			`{"field":"year","message":"Copy into 2026 or 2027."}`))
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)

	var campaigns struct {
		Items []struct {
			ID, Name, StartDate, EndDate string
		} `json:"items"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1005/campaigns", nil).Decode(t, &campaigns)
	var helix []string
	for _, c := range campaigns.Items {
		if c.Name == "Helix launch" {
			helix = append(helix, c.ID, c.StartDate, c.EndDate)
		}
	}
	if len(helix) != 3 || helix[0] == "C-1001" || helix[1] != "2027-01-01" || helix[2] != "2027-12-31" {
		t.Errorf("Helix launch in the copy = %v, want a new ID, 2027-01-01 – 2027-12-31", helix)
	}
	var activities struct {
		Items []struct {
			Name, Status, StartDate string
		} `json:"items"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/plans/P-1005/activities", nil).Decode(t, &activities)
	for _, a := range activities.Items {
		if a.Status != "created" || (a.Name == "Partner co-marketing fund" && a.StartDate != "2027-03-01") {
			t.Errorf("copied activity %q: %s from %s, want Created, a year later", a.Name, a.Status, a.StartDate)
		}
	}
	var log struct {
		Items []struct {
			Action, Details string
		} `json:"items"`
	}
	priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=plan&recordId=P-1005", nil).Decode(t, &log)
	if len(log.Items) != 1 || log.Items[0].Action != "copied" || log.Items[0].Details !=
		"from 'FY2026 Lubricants Europe' (P-1001), with 2 campaigns, 4 initiatives, 7 activities and 6 line items" {
		t.Errorf("log of the copy = %+v, want one copied entry with the counts", log.Items)
	}
}
