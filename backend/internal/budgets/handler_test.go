package budgets_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/budgets"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/money"
)

// The handler tests run the real API on the demo data (apitest), and every request and answer is checked
// against the contract. The seed has the global budget GB-1001 "FY2026 Global marketing budget" (2026,
// 2,500,000.00 USD, owner Carlos Mendes) with BA-1001 → P-1001 FY2026 Lubricants Europe (1,200,000.00) and
// BA-1002 → P-1002 FY2026 Mobility China (800,000.00); P-1003 FY2027 Global Brand has none (FR-040).

const (
	carlosEmail = "carlos.mendes@shell.com"
	priyaEmail  = "priya.raman@shell.com"
	meiEmail    = "mei.lin@shell.com"
)

const (
	budgetGoneBody     = `{"code":"not_found","message":"This global budget no longer exists — someone may have deleted it.","fields":[]}`
	allocationGoneBody = `{"code":"not_found","message":"This allocation no longer exists — someone may have removed it.","fields":[]}`
)

// The Global budgets list of 2026 has the demo global budget with its four totals; 2027 has none; the years
// are those with a global budget; New global budget is available (FR-036, FR-032).
func TestListGlobalBudgets_200_FR036(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	of2026 := carlos.Do(t, http.MethodGet, "/api/global-budgets?year=2026", nil)
	of2027 := carlos.Do(t, http.MethodGet, "/api/global-budgets?year=2027", nil)
	every := carlos.Do(t, http.MethodGet, "/api/global-budgets", nil)
	refused := carlos.Do(t, http.MethodGet, "/api/global-budgets?year=next", nil, apitest.SkipRequestValidation())

	var page budgets.Page
	decode(t, of2026, http.StatusOK, &page)
	if page.Total != 1 || len(page.Items) != 1 || !slices.Equal(page.Years, []int{2026}) || !page.Create.Available {
		t.Fatalf("2026 = %s", of2026.Body)
	}
	s := page.Items[0]
	if got := fmt.Sprintf("%s %s %d %s %d %s", s.ID, s.Name, s.Year, s.Owner.Name, s.AllocationCount, figuresText(s.Figures)); got !=
		"GB-1001 FY2026 Global marketing budget 2026 Carlos Mendes 2 "+demoFigures(t, server) {
		t.Errorf("item = %s", got)
	}
	assertJSON(t, of2027, http.StatusOK, map[string]any{"items": []any{}, "total": 0, "years": []int{2026},
		"create": map[string]any{"available": true, "reason": ""}})
	decode(t, every, http.StatusOK, &page)
	if page.Total != 1 {
		t.Errorf("every year = %s", every.Body)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"year","message":"Choose a year such as 2026."}]}`)
}

// The global budget page (US6 scenario 1): Global budget 2,500,000.00 USD, Allocated 2,000,000.00,
// Unallocated 500,000.00 and the plans' planned cost; one row per plan by name with teams, owner, status,
// planned cost, left to plan and budget used; no plan of 2026 without an allocation; the year lock and the
// delete lock (FR-032). An unknown global budget is not found.
func TestGetGlobalBudget_200_404_FR032(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	found := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-1001", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-9999", nil)

	var view budgets.GlobalBudgetView
	decode(t, found, http.StatusOK, &view)
	if got := figuresText(view.Figures); !strings.HasPrefix(got, "2,500,000.00 USD allocated 2,000,000.00 USD unallocated 500,000.00 USD planned ") {
		t.Errorf("figures = %s", got)
	}
	var rows []string
	for _, r := range view.Allocations {
		var teams []string
		for _, team := range r.Teams {
			teams = append(teams, team.Name)
		}
		rows = append(rows, fmt.Sprintf("%s %s %s v%d [%s] %s %s %s v%d %v", r.ID, r.Plan.Name, r.Plan.Status, r.Plan.Version,
			strings.Join(teams, ", "), r.Owner.Name, r.Amount.Format(), r.AmountUSD.Format(), r.Version, r.Access.Edit.Available))
	}
	want := []string{
		"BA-1001 FY2026 Lubricants Europe published v1 [Lubricants Europe] Priya Raman 1,200,000.00 USD 1,200,000.00 USD v1 true",
		"BA-1002 FY2026 Mobility China draft v1 [Mobility China] Mei Lin 800,000.00 USD 800,000.00 USD v1 true",
	}
	if !slices.Equal(rows, want) {
		t.Errorf("rows =\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	for _, r := range view.Allocations { // left to plan = allocation − planned cost (domain model §4.2)
		planned := "0.00"
		if r.PlannedCost != nil {
			planned = r.PlannedCost.Amount.String()
		}
		if r.LeftToPlan.Amount.Add(mustAmount(t, planned)) != r.Amount.Amount {
			t.Errorf("%s: left to plan %s + planned %s ≠ %s", r.ID, r.LeftToPlan.Format(), planned, r.Amount.Format())
		}
	}
	if view.OtherMarkets != nil || len(view.PlansWithoutAllocation) != 0 || view.Owner.Name != "Carlos Mendes" ||
		len(view.Locks) != 2 || view.Locks[1].Reason != "'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first." ||
		view.Access.Delete.Available || !view.Access.Edit.Available || !view.Access.Allocate.Available {
		t.Errorf("page = %s", found.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// Creating a global budget answers 201 with GB-1002 and its empty figures; a form with problems answers
// every problem at once (FR-029, messages 24 and 26).
func TestCreateGlobalBudget_201_400_FR029(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	body := map[string]any{"name": "FY2027 Global brand budget", "year": 2027, "amount": usd("900000.00"), "ownerId": "U-1005",
		"description": "For the global brand plan."}

	created := carlos.Do(t, http.MethodPost, "/api/global-budgets", body)
	refused := carlos.Do(t, http.MethodPost, "/api/global-budgets", map[string]any{
		"name": "fy2026 global marketing budget", "year": 2033, "amount": usd("-5.00"), "ownerId": "",
	})

	assertFields(t, created, http.StatusCreated, map[string]any{
		"id": "GB-1002", "name": "FY2027 Global brand budget", "year": 2027, "amount": usd("900000.00"), "version": 1,
		"description": "For the global brand plan.", "allocations": []any{}, "otherMarkets": nil,
		"figures": map[string]any{"amount": usd("900000.00"), "allocated": usd("0.00"), "unallocated": usd("900000.00"),
			"plannedCost": usd("0.00"), "overAllocated": false},
		"createdBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})
	var view budgets.GlobalBudgetView
	decode(t, created, http.StatusCreated, &view)
	if len(view.PlansWithoutAllocation) != 2 || view.PlansWithoutAllocation[0].Plan.Name != "FY2027 Global Brand" ||
		view.PlansWithoutAllocation[1].Plan.Name != "FY2027 Lubricants UK & Ireland" {
		t.Errorf("plans without allocation = %+v, want FY2027 Global Brand and FY2027 Lubricants UK & Ireland", view.PlansWithoutAllocation)
	}
	assertAnswer(t, refused, http.StatusBadRequest, validation(
		`{"field":"name","message":"A global budget called 'FY2026 Global marketing budget' already exists."},`+
			`{"field":"year","message":"Choose a year from 2025 to 2031."},`+
			`{"field":"amount","message":"Enter an amount of 0 or more."},`+
			`{"field":"ownerId","message":"Choose a person who can manage global budgets for all markets."}`))
}

// Saving changed fields answers the global budget after the change, version 2; a save from the old version
// is refused naming who changed it and when; an unknown global budget is not found (FR-029).
func TestUpdateGlobalBudget_200_409_404_FR029(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	server.Advance(90 * time.Minute)

	saved := carlos.Do(t, http.MethodPatch, "/api/global-budgets/GB-1001", map[string]any{"amount": usd("2600000.00"), "version": 1})
	stale := carlos.Do(t, http.MethodPatch, "/api/global-budgets/GB-1001", map[string]any{"name": "Old", "version": 1})
	gone := carlos.Do(t, http.MethodPatch, "/api/global-budgets/GB-9999", map[string]any{"version": 1})

	assertFields(t, saved, http.StatusOK, map[string]any{
		"amount": usd("2600000.00"), "version": 2, "updatedAt": "2026-03-12T10:30:00Z",
		"updatedBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})
	assertAnswer(t, stale, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this global budget",`+
		`"fields":[],"conflict":{"changedBy":"Carlos Mendes","changedAt":"2026-03-12T10:30:00Z"}}`)
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
	log := carlos.Do(t, http.MethodGet, "/api/activity-log?recordType=globalBudget&recordId=GB-1001", nil)
	var page history.LogPage
	decode(t, log, http.StatusOK, &page)
	if page.Total != 1 || changesText(page.Items[0].Changes) != "Amount: 2,500,000.00 USD → 2,600,000.00 USD" {
		t.Errorf("log = %s", log.Body)
	}
}

// A global budget with allocations cannot be deleted (409 in_use, message 30); one without goes to the
// recycle bin, and the answer offers Undo (FR-035).
func TestDeleteGlobalBudget_409_200_404_FR035(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	created := carlos.Do(t, http.MethodPost, "/api/global-budgets", map[string]any{
		"name": "Spare", "year": 2026, "amount": usd("1.00"), "ownerId": "U-1005",
	})
	if created.Status != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Status, created.Body)
	}

	inUse := carlos.Do(t, http.MethodDelete, "/api/global-budgets/GB-1001", nil)
	deleted := carlos.Do(t, http.MethodDelete, "/api/global-budgets/GB-1002", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-1002", nil)

	assertAnswer(t, inUse, http.StatusConflict, `{"code":"in_use","message":"'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first.","fields":[]}`)
	var result history.DeleteResult
	decode(t, deleted, http.StatusOK, &result)
	if result.RecycleBinEntry.RecordType != "globalBudget" || result.RecycleBinEntry.RecordName != "Spare" {
		t.Errorf("delete = %s", deleted.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
	assertAnswer(t, carlos.Do(t, http.MethodDelete, "/api/global-budgets/GB-1002", nil), http.StatusNotFound, budgetGoneBody)
}

// The Allocate dialog offers the plans of 2026 without an allocation: none while both are allocated, Mobility
// China once its allocation is removed (FR-030).
func TestListAllocationOptions_200_404_FR030(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	none := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-1001/allocation-options", nil)
	carlos.Do(t, http.MethodDelete, "/api/budget-allocations/BA-1002", nil)
	one := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-1001/allocation-options", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-9999/allocation-options", nil)

	assertJSON(t, none, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
	var page budgets.AllocationOptionPage
	decode(t, one, http.StatusOK, &page)
	if page.Total != 1 || page.Items[0].Plan.Name != "FY2026 Mobility China" || page.Items[0].Budget.Format() != "800,000.00 USD" ||
		page.Items[0].Owner.Name != "Mei Lin" || !page.Items[0].CanAllocate.Available {
		t.Errorf("options = %s", one.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// Allocating answers 201 with the global budget after the allocation and sets the plan's budget; the form's
// problems are answered at once; an out-of-date plan version is refused naming who changed the plan
// (FR-030, messages 28 and 29).
func TestCreateBudgetAllocation_201_400_409_FR030(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	carlos.Do(t, http.MethodDelete, "/api/budget-allocations/BA-1002", nil)

	stale := carlos.Do(t, http.MethodPost, "/api/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1002", "amount": usd("900000.00"), "planVersion": 9,
	})
	created := carlos.Do(t, http.MethodPost, "/api/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1002", "amount": usd("900000.00"), "planVersion": 1,
	})
	refused := carlos.Do(t, http.MethodPost, "/api/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1003", "amount": usd("1.00"), "planVersion": 1,
	})
	taken := carlos.Do(t, http.MethodPost, "/api/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1001", "amount": map[string]any{"amount": "1.00", "currency": "EUR"}, "planVersion": 1,
	})
	gone := carlos.Do(t, http.MethodPost, "/api/global-budgets/GB-9999/allocations", map[string]any{
		"planId": "P-1001", "amount": usd("1.00"), "planVersion": 1,
	})

	if stale.Status != http.StatusConflict || !strings.Contains(string(stale.Body), `"code":"version_conflict"`) {
		t.Errorf("stale = %d %s, want 409 version_conflict", stale.Status, stale.Body)
	}
	var view budgets.GlobalBudgetView
	decode(t, created, http.StatusCreated, &view)
	if len(view.Allocations) != 2 || view.Allocations[1].ID != "BA-1003" || view.Allocations[1].Amount.Format() != "900,000.00 USD" ||
		view.Allocations[1].Plan.Version != 2 || view.Figures.Allocated.Format() != "2,100,000.00 USD" {
		t.Errorf("created = %s", created.Body)
	}
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/plans/P-1002", nil), http.StatusOK, map[string]any{
		"budget": usd("900000.00"), "version": 2,
		"allocation": map[string]any{"allocationId": "BA-1003", "globalBudget": map[string]any{"id": "GB-1001", "name": "FY2026 Global marketing budget"},
			"amount": usd("900000.00"), "owner": map[string]any{"id": "U-1005", "name": "Carlos Mendes", "initials": "CM",
				"jobTitle": "Business champion", "active": true}, "canOpen": true},
	})
	assertAnswer(t, refused, http.StatusBadRequest, validation(`{"field":"planId","message":"Choose a plan of 2026."}`))
	assertAnswer(t, taken, http.StatusBadRequest, validation(
		`{"field":"planId","message":"'FY2026 Lubricants Europe' already has an allocation from 'FY2026 Global marketing budget'."},`+
			`{"field":"amount","message":"Enter the amount in USD, the plan's currency."}`))
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// US6 scenario 2: Carlos changes the Lubricants allocation to 1,300,000.00; the plan's budget becomes
// 1,300,000.00 at once, and the log shows the change on the global budget and on the plan, with the
// comment. A change from an out-of-date allocation version is refused; an unknown allocation is not found
// (FR-030, FR-037).
func TestUpdateBudgetAllocation_200_409_404_FR030_FR037(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	server.Advance(time.Hour)

	saved := carlos.Do(t, http.MethodPatch, "/api/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1300000.00"), "version": 1, "planVersion": 1,
	})
	stale := carlos.Do(t, http.MethodPatch, "/api/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1400000.00"), "version": 1, "planVersion": 2,
	})
	stalePlan := carlos.Do(t, http.MethodPatch, "/api/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1400000.00"), "version": 2, "planVersion": 1,
	})
	gone := carlos.Do(t, http.MethodPatch, "/api/budget-allocations/BA-9999", map[string]any{
		"amount": usd("1.00"), "version": 1, "planVersion": 1,
	})

	var view budgets.GlobalBudgetView
	decode(t, saved, http.StatusOK, &view)
	if view.Allocations[0].Amount.Format() != "1,300,000.00 USD" || view.Allocations[0].Version != 2 || view.Figures.Allocated.Format() != "2,100,000.00 USD" {
		t.Errorf("saved = %s", saved.Body)
	}
	conflict := `"fields":[],"conflict":{"changedBy":"Carlos Mendes","changedAt":"2026-03-12T10:00:00Z"}}`
	assertAnswer(t, stale, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this budget allocation",`+conflict)
	assertAnswer(t, stalePlan, http.StatusConflict, `{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this plan",`+conflict)
	assertAnswer(t, gone, http.StatusNotFound, allocationGoneBody)
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/plans/P-1001", nil), http.StatusOK, map[string]any{
		"budget": usd("1300000.00"), "version": 2, "updatedBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})

	var planLog, budgetLog history.LogPage
	decode(t, carlos.Do(t, http.MethodGet, "/api/activity-log?recordType=plan&recordId=P-1001", nil), http.StatusOK, &planLog)
	decode(t, carlos.Do(t, http.MethodGet, "/api/activity-log?recordType=globalBudget&recordId=GB-1001", nil), http.StatusOK, &budgetLog)
	if planLog.Total < 1 || changesText(planLog.Items[0].Changes) != "Plan budget: 1,200,000.00 USD → 1,300,000.00 USD" || // newest first, after the seeded log (011)
		planLog.Items[0].Comment != "Allocated from 'FY2026 Global marketing budget'" || planLog.Items[0].Actor.Name != "Carlos Mendes" {
		t.Errorf("plan log = %+v", planLog.Items)
	}
	if budgetLog.Total != 1 || changesText(budgetLog.Items[0].Changes) != "Allocation to 'FY2026 Lubricants Europe': 1,200,000.00 USD → 1,300,000.00 USD" {
		t.Errorf("global budget log = %+v", budgetLog.Items)
	}
}

// US6 scenarios 3 and 6: Priya's plan budget is set by the global budget, so a save that changes it is
// refused with 409 not_allowed and the plan form's sentence; once Carlos removes the Mobility allocation,
// Mei can change her plan's budget, which kept 800,000.00 (FR-031, FR-034).
func TestRemoveBudgetAllocation_200_404_AndThePlanBudgetLock_FR031_FR034(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	priya := server.SignIn(t, priyaEmail)
	mei := server.SignIn(t, meiEmail)
	europe := planForm(t, priya, "P-1001")
	europe["budget"] = usd("1300000.00")

	locked := priya.Do(t, http.MethodPatch, "/api/plans/P-1001", europe)
	removed := carlos.Do(t, http.MethodDelete, "/api/budget-allocations/BA-1002", nil)
	gone := carlos.Do(t, http.MethodDelete, "/api/budget-allocations/BA-1002", nil)
	china := planForm(t, mei, "P-1002")
	kept := china["budget"]
	china["budget"] = usd("850000.00")
	saved := mei.Do(t, http.MethodPatch, "/api/plans/P-1002", china)

	assertAnswer(t, locked, http.StatusConflict, `{"code":"not_allowed","message":"Allocated from 'FY2026 Global marketing budget' — ask Carlos Mendes to change it.","fields":[]}`)
	var view budgets.GlobalBudgetView
	decode(t, removed, http.StatusOK, &view)
	if len(view.Allocations) != 1 || view.Figures.Allocated.Format() != "1,200,000.00 USD" || len(view.PlansWithoutAllocation) != 1 {
		t.Errorf("removed = %s", removed.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, allocationGoneBody)
	if !reflect.DeepEqual(kept, map[string]any{"amount": "800000.00", "currency": "USD"}) {
		t.Errorf("kept budget = %v, want 800,000.00 USD", kept)
	}
	assertFields(t, saved, http.StatusOK, map[string]any{"budget": usd("850000.00"), "allocation": nil})
	var planLog history.LogPage
	decode(t, carlos.Do(t, http.MethodGet, "/api/activity-log?recordType=plan&recordId=P-1002", nil), http.StatusOK, &planLog)
	if planLog.Total < 2 || planLog.Items[1].Comment != "Allocation from 'FY2026 Global marketing budget' removed" || len(planLog.Items[1].Changes) != 0 {
		t.Errorf("plan log = %+v", planLog.Items)
	}
}

// Deleting a plan takes its allocation to the recycle bin; once the global budget is deleted too, restoring
// the plan brings it back without the allocation, with note 31 (FR-035).
func TestPlanDeleteAndRestore_TheAllocation_FR035(t *testing.T) {
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	carlos.Do(t, http.MethodDelete, "/api/budget-allocations/BA-1001", nil) // GB-1001 keeps one allocation only

	deleted := carlos.Do(t, http.MethodDelete, "/api/plans/P-1002", nil)
	var result history.DeleteResult
	decode(t, deleted, http.StatusOK, &result)
	emptied := carlos.Do(t, http.MethodDelete, "/api/global-budgets/GB-1001", nil)
	restored := carlos.Do(t, http.MethodPost, "/api/recycle-bin/"+result.RecycleBinEntry.ID+"/restore", nil)

	if emptied.Status != http.StatusOK {
		t.Fatalf("delete of the global budget whose plan is in the bin = %d %s", emptied.Status, emptied.Body)
	}
	var restore history.RestoreResult
	decode(t, restored, http.StatusOK, &restore)
	if !strings.HasSuffix(restore.Details, " The allocation was not restored because its global budget was deleted.") {
		t.Errorf("restore = %s", restored.Body)
	}
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/plans/P-1002", nil), http.StatusOK, map[string]any{
		"budget": usd("800000.00"), "allocation": nil,
	})
}

// demoFigures is the figures text of the demo global budget, with its plans' planned cost as the plan pages
// show it.
func demoFigures(t *testing.T, server *apitest.Server) string {
	t.Helper()
	carlos := server.SignIn(t, carlosEmail)
	var view budgets.GlobalBudgetView
	decode(t, carlos.Do(t, http.MethodGet, "/api/global-budgets/GB-1001", nil), http.StatusOK, &view)
	return figuresText(view.Figures)
}

// planForm is the edit form of the plan id as the edit page sends it unchanged.
func planForm(t *testing.T, c *apitest.Client, id string) map[string]any {
	t.Helper()
	var plan map[string]any
	if err := json.Unmarshal(c.Do(t, http.MethodGet, "/api/plans/"+id, nil).Body, &plan); err != nil {
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
	return `{"code":"validation_failed","message":"Some fields need your attention.","fields":[` + fields + `]}`
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
