package budget_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/budget"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

// The Allocate dialog offers the plans of 2026 without an allocation: none while both are allocated, Mobility
// China once its allocation is removed (FR-030).
func TestListAllocationOptions_200_404_FR030(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	none := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1001/allocation-options", nil)
	carlos.Do(t, http.MethodDelete, "/api/v1/budget-allocations/BA-1002", nil)
	one := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1001/allocation-options", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-9999/allocation-options", nil)

	assertJSON(t, none, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
	var page budget.AllocationOptionPage
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
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	carlos.Do(t, http.MethodDelete, "/api/v1/budget-allocations/BA-1002", nil)

	stale := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1002", "amount": usd("900000.00"), "planVersion": 9,
	})
	created := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1002", "amount": usd("900000.00"), "planVersion": 1,
	})
	refused := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1003", "amount": usd("1.00"), "planVersion": 1,
	})
	taken := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets/GB-1001/allocations", map[string]any{
		"planId": "P-1001", "amount": map[string]any{"amount": "1.00", "currency": "EUR"}, "planVersion": 1,
	})
	gone := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets/GB-9999/allocations", map[string]any{
		"planId": "P-1001", "amount": usd("1.00"), "planVersion": 1,
	})

	if stale.Status != http.StatusConflict || !strings.Contains(string(stale.Body), `"code":"version_conflict"`) {
		t.Errorf("stale = %d %s, want 409 version_conflict", stale.Status, stale.Body)
	}
	var view budget.GlobalBudgetView
	decode(t, created, http.StatusCreated, &view)
	if len(view.Allocations) != 2 || view.Allocations[1].ID != "BA-1003" || view.Allocations[1].Amount.Format() != "900,000.00 USD" ||
		view.Allocations[1].Plan.Version != 2 || view.Figures.Allocated.Format() != "2,100,000.00 USD" {
		t.Errorf("created = %s", created.Body)
	}
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil), http.StatusOK, map[string]any{
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
func TestUpdateBudgetAllocation_200_409_404_FR030_FR037(t *testing.T) { //nolint:gocyclo // 10: one change and three refusals, then the plan and both logs
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	server.Advance(time.Hour)

	saved := carlos.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1300000.00"), "version": 1, "planVersion": 1,
	})
	stale := carlos.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1400000.00"), "version": 1, "planVersion": 2,
	})
	stalePlan := carlos.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-1001", map[string]any{
		"amount": usd("1400000.00"), "version": 2, "planVersion": 1,
	})
	gone := carlos.Do(t, http.MethodPatch, "/api/v1/budget-allocations/BA-9999", map[string]any{
		"amount": usd("1.00"), "version": 1, "planVersion": 1,
	})

	var view budget.GlobalBudgetView
	decode(t, saved, http.StatusOK, &view)
	if view.Allocations[0].Amount.Format() != "1,300,000.00 USD" || view.Allocations[0].Version != 2 || view.Figures.Allocated.Format() != "2,100,000.00 USD" {
		t.Errorf("saved = %s", saved.Body)
	}
	conflict := `"fields":[],"conflict":{"changedBy":"Carlos Mendes","changedAt":"2026-03-12T10:00:00Z"}}}`
	assertAnswer(t, stale, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this budget allocation",`+conflict)
	assertAnswer(t, stalePlan, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this plan",`+conflict)
	assertAnswer(t, gone, http.StatusNotFound, allocationGoneBody)
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/v1/plans/P-1001", nil), http.StatusOK, map[string]any{
		"budget": usd("1300000.00"), "version": 2, "updatedBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})

	var planLog, budgetLog history.LogPage
	decode(t, carlos.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=plan&recordId=P-1001", nil), http.StatusOK, &planLog)
	decode(t, carlos.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=globalBudget&recordId=GB-1001", nil), http.StatusOK, &budgetLog)
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
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	priya := server.SignIn(t, priyaEmail)
	mei := server.SignIn(t, meiEmail)
	europe := planForm(t, priya, "P-1001")
	europe["budget"] = usd("1300000.00")

	locked := priya.Do(t, http.MethodPatch, "/api/v1/plans/P-1001", europe)
	removed := carlos.Do(t, http.MethodDelete, "/api/v1/budget-allocations/BA-1002", nil)
	gone := carlos.Do(t, http.MethodDelete, "/api/v1/budget-allocations/BA-1002", nil)
	china := planForm(t, mei, "P-1002")
	kept := china["budget"]
	china["budget"] = usd("850000.00")
	saved := mei.Do(t, http.MethodPatch, "/api/v1/plans/P-1002", china)

	assertAnswer(t, locked, http.StatusConflict, `{"error":{"code":"not_allowed","message":"Allocated from 'FY2026 Global marketing budget' — ask Carlos Mendes to change it.","fields":[]}}`)
	var view budget.GlobalBudgetView
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
	decode(t, carlos.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=plan&recordId=P-1002", nil), http.StatusOK, &planLog)
	if planLog.Total < 2 || planLog.Items[1].Comment != "Allocation from 'FY2026 Global marketing budget' removed" || len(planLog.Items[1].Changes) != 0 {
		t.Errorf("plan log = %+v", planLog.Items)
	}
}

// Deleting a plan takes its allocation to the recycle bin; once the global budget is deleted too, restoring
// the plan brings it back without the allocation, with note 31 (FR-035).
func TestPlanDeleteAndRestore_TheAllocation_FR035(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	carlos.Do(t, http.MethodDelete, "/api/v1/budget-allocations/BA-1001", nil) // GB-1001 keeps one allocation only

	deleted := carlos.Do(t, http.MethodDelete, "/api/v1/plans/P-1002", nil)
	var result history.DeleteResult
	decode(t, deleted, http.StatusOK, &result)
	emptied := carlos.Do(t, http.MethodDelete, "/api/v1/global-budgets/GB-1001", nil)
	restored := carlos.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+result.RecycleBinEntry.ID+"/restore", nil)

	if emptied.Status != http.StatusOK {
		t.Fatalf("delete of the global budget whose plan is in the bin = %d %s", emptied.Status, emptied.Body)
	}
	var restore history.RestoreResult
	decode(t, restored, http.StatusOK, &restore)
	if !strings.HasSuffix(restore.Details, " The allocation was not restored because its global budget was deleted.") {
		t.Errorf("restore = %s", restored.Body)
	}
	assertFields(t, carlos.Do(t, http.MethodGet, "/api/v1/plans/P-1002", nil), http.StatusOK, map[string]any{
		"budget": usd("800000.00"), "allocation": nil,
	})
}
