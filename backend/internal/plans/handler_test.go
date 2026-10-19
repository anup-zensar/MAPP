package plans_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/plans"
	"northstar/internal/platform/apitest"
)

// The handler tests run the real API on the demo data (apitest), and every request and answer is checked
// against the contract. The seed has three plans: P-1001 FY2026 Lubricants Europe (Published), P-1002
// FY2026 Mobility China (Draft, no business unit) and P-1003 FY2027 Global Brand (Draft).

const (
	priyasEmail  = "priya.raman@northstar.demo"
	danielsEmail = "daniel.okoro@northstar.demo"
	meisEmail    = "mei.lin@northstar.demo"
)

const (
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	planGoneBody       = `{"code":"not_found","message":"This plan no longer exists — someone may have deleted it.","fields":[]}`
)

// newPlanBody is the whole form of a new plan, as the web app sends it: FY2026 Lubricants Test with
// Lubricants 1.0, Lubricants Europe, Priya as owner, business unit Lubricants and an INQ goal of 250,000.
func newPlanBody() map[string]any {
	return map[string]any{
		"name": "FY2026 Lubricants Test", "year": 2026, "modelId": "MOD-1001", "teamIds": []string{"T-1001"},
		"ownerId": "U-1002", "aop": false, "description": "", "budget": nil, "defaultFinanceBudgetId": nil,
		"defaultSpread": "evenly", "bcpir": false, "goalStage": "inq", "goalAmount": "250000",
		"attributes": map[string]any{"SA-1001": []string{"SO-1002"}},
	}
}

// What a new plan starts with for Priya: 2026, the years 2025–2031, her team Lubricants Europe and herself
// as owner; the seed has two active models, so none is chosen (FR-006).
func TestPlanDefaults_FR006(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/plan-defaults", nil)

	assertAnswer(t, response, http.StatusOK, `{"currentYear":2026,"years":[2025,2026,2027,2028,2029,2030,2031],`+
		`"year":2026,"teamIds":["T-1001"],"ownerId":"U-1002","modelId":null,"aop":false,"bcpir":false,`+
		`"defaultSpread":"evenly","goalStage":"inq"}`)
}

// Creating a plan answers 201 with the new Draft plan, P-1004 after the seed, its figures and goals; a
// form without name, year and model answers every problem at once (FR-010, US1 scenarios 2 and 3).
func TestCreatePlan_201AndValidation_FR010(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	missing := newPlanBody()
	missing["name"], missing["year"], missing["modelId"] = " ", nil, nil

	created := priya.Do(t, http.MethodPost, "/api/plans", newPlanBody())
	refused := priya.Do(t, http.MethodPost, "/api/plans", missing)

	if created.Status != http.StatusCreated {
		t.Fatalf("create = %d %s, want 201", created.Status, created.Body)
	}
	var plan plans.PlanView
	created.Decode(t, &plan)
	if plan.ID != "P-1004" || plan.Status != plans.StatusDraft || plan.Name != "FY2026 Lubricants Test" ||
		plan.Owner == nil || plan.Owner.Name != "Priya Raman" || len(plan.Completeness.Missing) != 0 ||
		plan.Goals.Plan.MQL == nil || *plan.Goals.Plan.MQL != 5000 || plan.Goals.Plan.Revenue == nil ||
		plan.Goals.Plan.Revenue.Amount.String() != "2500000.00" || plan.CreatedBy.Name != "Priya Raman" {
		t.Errorf("created plan = %s", created.Body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a plan name."},`+
		`{"field":"year","message":"Choose a year from 2025 to 2031."},{"field":"modelId","message":"Choose a model."}`))
}

// Every plan operation needs a signed-in person (FR-005 of feature 001).
func TestPlansOperations_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t)
	calls := []struct {
		method, path string
		body         any
	}{
		{http.MethodGet, "/api/plan-defaults", nil},
		{http.MethodGet, "/api/plans", nil},
		{http.MethodPost, "/api/plans", newPlanBody()},
		{http.MethodGet, "/api/plans/P-1001", nil},
		{http.MethodPatch, "/api/plans/P-1001", europeForm(1)},
		{http.MethodDelete, "/api/plans/P-1001", nil},
		{http.MethodPost, "/api/plans/P-1001/change-effects", map[string]any{"modelId": "MOD-1002", "attributes": map[string]any{}}},
		{http.MethodPost, "/api/plans/P-1002/publish", nil},
		{http.MethodPost, "/api/plans/P-1001/unpublish", map[string]any{"reason": "Budget under review"}},
		{http.MethodGet, "/api/plans/P-1001/copy-options", nil},
		{http.MethodPost, "/api/plans/P-1001/copy", map[string]any{"name": "Copy", "year": 2026, "includeContents": false}},
	}
	for _, call := range calls {
		response := server.Do(t, call.method, call.path, call.body)

		assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
	}
}

// validationBody is the answer of a validation_failed error with the given field problems.
func validationBody(fields string) string {
	return `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}`
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

// listOf decodes a Plans list answer.
func listOf(t *testing.T, response *apitest.Response) plans.PlanList {
	t.Helper()
	if response.Status != http.StatusOK {
		t.Fatalf("list = %d %s, want 200", response.Status, response.Body)
	}
	var list plans.PlanList
	response.Decode(t, &list)
	return list
}

// listIDs returns the IDs of a list's plans, in order.
func listIDs(list plans.PlanList) []string {
	planIDs := []string{}
	for _, item := range list.Items {
		planIDs = append(planIDs, item.ID)
	}
	return planIDs
}

// The Plans page shows the three seed plans, newest year first, then by name, with 3 plans, 2 Draft and 1
// Published; Draft and 2026 leave FY2026 Mobility China, "1 of 3"; the search finds a name or an ID in any
// letter case; wrong filters are refused (FR-001, FR-003, US2 scenarios 1–3).
func TestListPlans_SeedOrderSummaryAndFilters_FR001_FR003(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	all := listOf(t, priya.Do(t, http.MethodGet, "/api/plans", nil))
	draft2026 := listOf(t, priya.Do(t, http.MethodGet, "/api/plans?status=draft&year=2026", nil))
	byName := listOf(t, priya.Do(t, http.MethodGet, "/api/plans?q=mobility", nil))
	byID := listOf(t, priya.Do(t, http.MethodGet, "/api/plans?q=p-1002", nil))
	wrong := priya.Do(t, http.MethodGet, "/api/plans?status=open&year=next", nil, apitest.SkipRequestValidation())

	if ids := listIDs(all); !reflect.DeepEqual(ids, []string{"P-1003", "P-1001", "P-1002"}) || all.Total != 3 ||
		!reflect.DeepEqual(all.Years, []int{2027, 2026}) {
		t.Errorf("all plans: %q, total %d, years %v", ids, all.Total, all.Years)
	}
	summary := all.Summary
	if summary.Plans != 3 || summary.Draft != 2 || summary.Published != 1 || summary.PlannedCost == nil ||
		summary.PlannedCost.Format() != "650,000.00 USD" || summary.Budget == nil || summary.Budget.Format() != "2,500,000.00 USD" {
		t.Errorf("summary = %+v; want 3 plans, 2 Draft, 1 Published, 650,000.00 USD planned, 2,500,000.00 USD", summary)
	}
	for name, list := range map[string]plans.PlanList{"draft 2026": draft2026, "mobility": byName, "p-1002": byID} {
		if ids := listIDs(list); !reflect.DeepEqual(ids, []string{"P-1002"}) || list.Total != 3 {
			t.Errorf("%s: %q, total %d; want P-1002 of 3", name, ids, list.Total)
		}
	}
	assertAnswer(t, wrong, http.StatusBadRequest, validationBody(`{"field":"status","message":"Choose draft or published."},`+
		`{"field":"year","message":"Choose a year such as 2026."}`))
}

// My teams shows Mei Lin only her team's plan (FR-004, US2 scenario 4).
func TestListPlans_MyTeams_FR004(t *testing.T) {
	server := apitest.New(t)
	mei := server.SignIn(t, meisEmail)

	mine := listOf(t, mei.Do(t, http.MethodGet, "/api/plans?scope=myTeams", nil))

	if ids := listIDs(mine); !reflect.DeepEqual(ids, []string{"P-1002"}) || mine.Total != 1 {
		t.Errorf("Mei's teams: %q, total %d; want P-1002 of 1", ids, mine.Total)
	}
}

// The plan page of FY2026 Lubricants Europe shows every stored field, its model with rates, team, owner,
// business unit, budget and goals 250,000 → 5,000 → 250 → 2,500,000.00, all filled in; FY2026 Mobility
// China needs its business unit, and below it its campaign Retail growth, initiative Fleet cards,
// activities Grand opening Shanghai and Lunar new year promo and line item L-1007 (005's, 006's and 007's
// demo data) are incomplete; an unknown plan is not found (FR-011 – FR-017, US3). The figures that
// come from the levels below are proved with fakes in the service tests.
func TestGetPlan_ViewAnd404_FR011(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	europe := priya.Do(t, http.MethodGet, "/api/plans/P-1001", nil)
	china := priya.Do(t, http.MethodGet, "/api/plans/P-1002", nil)
	gone := priya.Do(t, http.MethodGet, "/api/plans/P-9999", nil)

	system := map[string]any{"id": "system", "name": "NorthStar"}
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
	var plan plans.PlanView
	europe.Decode(t, &plan)
	goal := plan.Goals.Plan
	if goal.INQ == nil || *goal.INQ != 250_000 || goal.MQL == nil || *goal.MQL != 5_000 || goal.Deals == nil || *goal.Deals != 250 ||
		goal.Revenue == nil || goal.Revenue.Format() != "2,500,000.00 USD" || plan.Goals.Stage != plans.StageINQ {
		t.Errorf("goals = %+v", plan.Goals)
	}
	if len(plan.Completeness.Missing) != 0 || plan.Figures.Currency != "USD" || plan.Figures.Budget.Format() != "1,200,000.00 USD" {
		t.Errorf("completeness %+v, figures %+v; want all filled in, in USD", plan.Completeness, plan.Figures)
	}
	assertFields(t, china, http.StatusOK, map[string]any{"status": "draft", "businessUnit": []any{}})
	var chinaPlan plans.PlanView
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

// assertFields checks the status of an answer and the given fields of its JSON body.
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

// europeForm is the edit form of the seed plan FY2026 Lubricants Europe, unchanged, from version.
func europeForm(version int) map[string]any {
	return map[string]any{
		"version": version, "name": "FY2026 Lubricants Europe", "modelId": "MOD-1001", "teamIds": []string{"T-1001"},
		"ownerId": "U-1002", "aop": false, "description": "Lubricants marketing in Europe for 2026.",
		"budget": map[string]any{"amount": "1200000.00", "currency": "USD"}, "defaultFinanceBudgetId": "FB-1001",
		"defaultSpread": "evenly", "bcpir": false, "goalStage": "inq", "goalAmount": "250000",
		"attributes": map[string]any{"SA-1001": []string{"SO-1002"}},
	}
}

// Saving the edit form answers the plan after the change: still Published, version 2, by Daniel. A save
// from the old version is refused naming Daniel and the time; a missing version and an empty name are
// reported at once; an unknown plan is not found (FR-018, FR-022, FR-023, US4).
func TestUpdatePlan_200_409_404_FR018_FR023(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	server.Advance(90 * time.Minute)
	edited := europeForm(1)
	edited["budget"] = map[string]any{"amount": "1300000.00", "currency": "USD"}
	invalid := europeForm(0)
	invalid["name"] = ""

	saved := daniel.Do(t, http.MethodPatch, "/api/plans/P-1001", edited)
	outOfDate := priya.Do(t, http.MethodPatch, "/api/plans/P-1001", europeForm(1))
	refused := priya.Do(t, http.MethodPatch, "/api/plans/P-1001", invalid)
	gone := priya.Do(t, http.MethodPatch, "/api/plans/P-9999", europeForm(1))

	assertFields(t, saved, http.StatusOK, map[string]any{
		"status": "published", "version": 2, "year": 2026, "budget": map[string]any{"amount": "1300000.00", "currency": "USD"},
		"updatedAt": "2026-03-12T10:30:00Z", "updatedBy": map[string]any{"id": "U-1003", "name": "Daniel Okoro"},
	})
	assertAnswer(t, outOfDate, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Daniel Okoro changed this plan",`+
		`"fields":[],"conflict":{"changedBy":"Daniel Okoro","changedAt":"2026-03-12T10:30:00Z"}}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a plan name."},`+
		`{"field":"version","message":"Reload the page and try again."}`))
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// The consequences of a new model and business unit are listed before saving, without saving anything:
// removing Lubricants from FY2027 Global Brand hides the Lubricants attributes of its activities (FR-019,
// FR-020, US4 scenarios 2 and 3).
func TestChangeEffects_FR019_FR020(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	effects := priya.Do(t, http.MethodPost, "/api/plans/P-1003/change-effects",
		map[string]any{"modelId": "MOD-1002", "attributes": map[string]any{"SA-1001": []string{"SO-1001"}}})
	unknown := priya.Do(t, http.MethodPost, "/api/plans/P-1003/change-effects",
		map[string]any{"modelId": "MOD-9999", "attributes": map[string]any{}})
	gone := priya.Do(t, http.MethodPost, "/api/plans/P-9999/change-effects", map[string]any{"modelId": nil, "attributes": map[string]any{}})

	assertAnswer(t, effects, http.StatusOK, `{"modelChanged":true,"disallowedActivities":[],"hiddenGroups":[`+
		`{"unit":"Lubricants","levels":["activity"],"attributes":["Brands","Global key accounts","Sector","Sub-sector","Product launch"]}],`+
		`"shownGroups":[]}`)
	assertAnswer(t, unknown, http.StatusBadRequest, validationBody(`{"field":"modelId","message":"Choose a model from the list."}`))
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
	var plan plans.PlanView
	priya.Do(t, http.MethodGet, "/api/plans/P-1003", nil).Decode(t, &plan)
	if plan.Version != 1 || plan.Model.ID != "MOD-1001" || len(plan.BusinessUnit) != 2 {
		t.Errorf("FY2027 Global Brand changed: version %d, model %s, business unit %+v", plan.Version, plan.Model.ID, plan.BusinessUnit)
	}
}

// formOf is the edit form of a plan as the plan page shows it, unchanged, as the web app sends it.
func formOf(p plans.PlanView) map[string]any {
	teamIDs := []string{}
	for _, team := range p.Teams {
		teamIDs = append(teamIDs, team.ID)
	}
	form := map[string]any{
		"version": p.Version, "name": p.Name, "modelId": p.Model.ID, "teamIds": teamIDs, "ownerId": nil,
		"aop": p.AOP, "description": p.Description, "budget": p.Budget, "defaultFinanceBudgetId": nil,
		"defaultSpread": p.DefaultSpread, "bcpir": p.BCPIR, "goalStage": p.GoalStage, "goalAmount": p.GoalAmount,
		"attributes": p.Attributes,
	}
	if p.Owner != nil {
		form["ownerId"] = p.Owner.ID
	}
	if p.DefaultFinanceBudget != nil {
		form["defaultFinanceBudgetId"] = p.DefaultFinanceBudget.ID
	}
	return form
}

// FY2026 Mobility China has no business unit, and its campaign Retail growth has no cluster and region and
// its initiative Fleet cards no dates (005's demo data). Publish lists the plan's field first, then each
// record below that blocks publishing, and the plan stays Draft; once the business unit is filled in and the
// incomplete campaign is gone, Publish succeeds (FR-024, US5 scenarios 1 and 2).
func TestPublish_NotReadyThenPublished_FR024(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	refused := priya.Do(t, http.MethodPost, "/api/plans/P-1002/publish", nil)
	var china plans.PlanView
	priya.Do(t, http.MethodGet, "/api/plans/P-1002", nil).Decode(t, &china)
	form := formOf(china)
	form["attributes"] = map[string]any{"SA-1001": []string{"SO-1002"}}
	saved := priya.Do(t, http.MethodPatch, "/api/plans/P-1002", form)
	stillRefused := priya.Do(t, http.MethodPost, "/api/plans/P-1002/publish", nil)
	deleted := priya.Do(t, http.MethodDelete, "/api/campaigns/C-1003", nil)
	published := priya.Do(t, http.MethodPost, "/api/plans/P-1002/publish", nil)
	gone := priya.Do(t, http.MethodPost, "/api/plans/P-9999/publish", nil)

	below := `{"record":{"type":"campaign","id":"C-1003","name":"Retail growth"},` +
		`"fields":[{"field":"attributes.SA-1003","label":"Cluster"},{"field":"attributes.SA-1004","label":"Region"}]},` +
		`{"record":{"type":"initiative","id":"I-1006","name":"Fleet cards"},"fields":[{"field":"startDate","label":"Dates"}]}`
	assertAnswer(t, refused, http.StatusConflict, `{"code":"not_ready","message":"Publish needs 4 fields","fields":[],`+
		`"missing":[{"record":{"type":"plan","id":"P-1002","name":"FY2026 Mobility China"},`+
		`"fields":[{"field":"attributes.SA-1001","label":"Business unit"}]},`+below+`]}`)
	if china.Status != plans.StatusDraft || china.Version != 1 {
		t.Errorf("after the refusal: %s, version %d; want Draft, version 1", china.Status, china.Version)
	}
	assertFields(t, saved, http.StatusOK, map[string]any{"status": "draft", "version": 2})
	assertAnswer(t, stillRefused, http.StatusConflict, `{"code":"not_ready","message":"Publish needs 3 fields","fields":[],`+
		`"missing":[`+below+`]}`)
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
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	empty := priya.Do(t, http.MethodPost, "/api/plans/P-1001/unpublish", map[string]any{"reason": "  "})
	long := priya.Do(t, http.MethodPost, "/api/plans/P-1001/unpublish", map[string]any{"reason": strings.Repeat("x", 501)})
	unpublished := priya.Do(t, http.MethodPost, "/api/plans/P-1001/unpublish", map[string]any{"reason": "Budget under review"})
	log := priya.Do(t, http.MethodGet, "/api/activity-log?recordType=plan&recordId=P-1001", nil)
	gone := priya.Do(t, http.MethodPost, "/api/plans/P-9999/unpublish", map[string]any{"reason": "Budget under review"})

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

// Deleting a plan answers its recycle-bin entry, so the web app can offer Undo; Undo brings it back with
// the same ID and status (FR-029, FR-030, US6 scenarios 3 and 4).
func TestDeletePlan_AndUndo_FR029_FR030(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	deleted := priya.Do(t, http.MethodDelete, "/api/plans/P-1001", nil)
	inBin := priya.Do(t, http.MethodGet, "/api/plans/P-1001", nil)
	var result struct {
		RecycleBinEntry struct {
			ID              string `json:"id"`
			RecordType      string `json:"recordType"`
			RecordTypeLabel string `json:"recordTypeLabel"`
			RecordID        string `json:"recordId"`
			RecordName      string `json:"recordName"`
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(t, &result)
	undone := priya.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil)
	back := priya.Do(t, http.MethodGet, "/api/plans/P-1001", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/plans/P-9999", nil)

	if deleted.Status != http.StatusOK || result.RecycleBinEntry.RecordType != "plan" ||
		result.RecycleBinEntry.RecordTypeLabel != "plan" || result.RecycleBinEntry.RecordID != "P-1001" ||
		result.RecycleBinEntry.RecordName != "FY2026 Lubricants Europe" {
		t.Errorf("delete = %d %s", deleted.Status, deleted.Body)
	}
	assertAnswer(t, inBin, http.StatusNotFound, planGoneBody)
	assertFields(t, undone, http.StatusOK, map[string]any{
		"recordType": "plan", "recordId": "P-1001", "recordName": "FY2026 Lubricants Europe", "details": "",
	})
	assertFields(t, back, http.StatusOK, map[string]any{"id": "P-1001", "status": "published", "version": 1})
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// The copy dialog of FY2026 Lubricants Europe offers "(copy)", 2026 or 2027, and the counts of what comes
// with it: 2 campaigns, 4 initiatives, 7 activities (4 approved) and 6 line items (FR-026, FR-028).
func TestCopyOptions_FR026(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	options := priya.Do(t, http.MethodGet, "/api/plans/P-1001/copy-options", nil)
	gone := priya.Do(t, http.MethodGet, "/api/plans/P-9999/copy-options", nil)

	assertAnswer(t, options, http.StatusOK, `{"name":"FY2026 Lubricants Europe (copy)","sameYear":2026,`+
		`"nextYear":{"year":2027,"available":true,"reason":""},`+
		`"contents":{"campaigns":2,"initiatives":4,"activities":7,"approvedActivities":4,"lineItems":6}}`)
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}

// Copying FY2026 Lubricants Europe into 2027 with everything in it answers 201 with a Draft 2027 plan
// without a default finance budget and a note saying so; everything below comes along with new IDs, every
// date one year later and every activity Created; one log entry counts it all (FR-027, FR-028, R13). A
// wrong name and year answer 400 with both problems.
func TestCopy_201_FR027_FR028(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	copied := priya.Do(t, http.MethodPost, "/api/plans/P-1001/copy",
		map[string]any{"name": "FY2027 Lubricants Europe", "year": 2027, "includeContents": true})
	refused := priya.Do(t, http.MethodPost, "/api/plans/P-1001/copy",
		map[string]any{"name": "fy2026 mobility china", "year": 2025, "includeContents": false})
	gone := priya.Do(t, http.MethodPost, "/api/plans/P-9999/copy",
		map[string]any{"name": "Copy", "year": 2026, "includeContents": false})

	var result plans.CopyResult
	copied.Decode(t, &result)
	p := result.Plan
	if copied.Status != http.StatusCreated || p.ID != "P-1004" || p.Year != 2027 || p.Status != plans.StatusDraft ||
		p.DefaultFinanceBudget != nil || p.Contents != (plans.Contents{Campaigns: 2, Initiatives: 4, Activities: 7, LineItems: 6}) ||
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
	priya.Do(t, http.MethodGet, "/api/plans/P-1004/campaigns", nil).Decode(t, &campaigns)
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
	priya.Do(t, http.MethodGet, "/api/plans/P-1004/activities", nil).Decode(t, &activities)
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
	priya.Do(t, http.MethodGet, "/api/activity-log?recordType=plan&recordId=P-1004", nil).Decode(t, &log)
	if len(log.Items) != 1 || log.Items[0].Action != "copied" || log.Items[0].Details !=
		"from 'FY2026 Lubricants Europe' (P-1001), with 2 campaigns, 4 initiatives, 7 activities and 6 line items" {
		t.Errorf("log of the copy = %+v, want one copied entry with the counts", log.Items)
	}
}
