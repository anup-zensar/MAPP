package budget_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/budget"
	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

// The handler tests run the real API on the demo data (apitest), and every request and answer is checked
// against the contract. The seed has the global budget GB-1001 "FY2026 Global marketing budget" (2026,
// 2,500,000.00 USD, owner Carlos Mendes) with BA-1001 → P-1001 FY2026 Lubricants Europe (1,200,000.00) and
// BA-1002 → P-1002 FY2026 Mobility China (800,000.00); P-1003 FY2027 Global Brand has none (FR-040).

// The Global budgets list of 2026 has the demo global budget with its four totals; 2027 has none; the years
// are those with a global budget; New global budget is available (FR-036, FR-032).
func TestListGlobalBudgets_200_FR036(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	of2026 := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets?year=2026", nil)
	of2027 := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets?year=2027", nil)
	every := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets", nil)
	refused := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets?year=next", nil, apitest.SkipRequestValidation())

	var page budget.Page
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
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"year","message":"Choose a year such as 2026."}]}}`)
}

// The global budget page (US6 scenario 1): Global budget 2,500,000.00 USD, Allocated 2,000,000.00,
// Unallocated 500,000.00 and the plans' planned cost; one row per plan by name with teams, owner, status,
// planned cost, left to plan and budget used; no plan of 2026 without an allocation; the year lock and the
// delete lock (FR-032). An unknown global budget is not found.
func TestGetGlobalBudget_200_404_FR032(t *testing.T) { //nolint:gocyclo // 11: the figures and rows, then the page's other markets, owner, locks and access in one condition
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)

	found := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1001", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-9999", nil)

	var view budget.GlobalBudgetView
	decode(t, found, http.StatusOK, &view)
	if got := figuresText(view.Figures); !strings.HasPrefix(got, "2,500,000.00 USD allocated 2,000,000.00 USD unallocated 500,000.00 USD planned ") {
		t.Errorf("figures = %s", got)
	}
	want := []string{
		"BA-1001 FY2026 Lubricants Europe published v1 [Lubricants Europe] Priya Raman 1,200,000.00 USD 1,200,000.00 USD v1 true",
		"BA-1002 FY2026 Mobility China draft v1 [Mobility China] Mei Lin 800,000.00 USD 800,000.00 USD v1 true",
	}
	if rows := allocationLines(view.Allocations); !slices.Equal(rows, want) {
		t.Errorf("rows =\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	assertLeftToPlan(t, view.Allocations)
	if view.OtherMarkets != nil || len(view.PlansWithoutAllocation) != 0 || view.Owner.Name != "Carlos Mendes" ||
		len(view.Locks) != 2 || view.Locks[1].Reason != "'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first." ||
		view.Access.Delete.Available || !view.Access.Edit.Available || !view.Access.Allocate.Available {
		t.Errorf("page = %s", found.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
}

// allocationLines writes the allocation rows of a global budget page one per line.
func allocationLines(rows []budget.AllocationRow) []string {
	var lines []string
	for _, r := range rows {
		var teams []string
		for _, team := range r.Teams {
			teams = append(teams, team.Name)
		}
		lines = append(lines, fmt.Sprintf("%s %s %s v%d [%s] %s %s %s v%d %v", r.ID, r.Plan.Name, r.Plan.Status, r.Plan.Version,
			strings.Join(teams, ", "), r.Owner.Name, r.Amount.Format(), r.AmountUSD.Format(), r.Version, r.Access.Edit.Available))
	}
	return lines
}

// assertLeftToPlan checks that each row's left to plan is its allocation less its planned cost (domain model §4.2).
func assertLeftToPlan(t *testing.T, rows []budget.AllocationRow) {
	t.Helper()
	for _, r := range rows {
		planned := "0.00"
		if r.PlannedCost != nil {
			planned = r.PlannedCost.Amount.String()
		}
		if r.LeftToPlan.Amount.Add(mustAmount(t, planned)) != r.Amount.Amount {
			t.Errorf("%s: left to plan %s + planned %s ≠ %s", r.ID, r.LeftToPlan.Format(), planned, r.Amount.Format())
		}
	}
}

// Creating a global budget answers 201 with GB-1002 and its empty figures; a form with problems answers
// every problem at once (FR-029, messages 24 and 26).
func TestCreateGlobalBudget_201_400_FR029(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	body := map[string]any{"name": "FY2027 Global brand budget", "year": 2027, "amount": usd("900000.00"), "ownerId": "U-1005",
		"description": "For the global brand plan."}

	created := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets", body)
	refused := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets", map[string]any{
		"name": "fy2026 global marketing budget", "year": 2033, "amount": usd("-5.00"), "ownerId": "",
	})

	assertFields(t, created, http.StatusCreated, map[string]any{
		"id": "GB-1002", "name": "FY2027 Global brand budget", "year": 2027, "amount": usd("900000.00"), "version": 1,
		"description": "For the global brand plan.", "allocations": []any{}, "otherMarkets": nil,
		"figures": map[string]any{"amount": usd("900000.00"), "allocated": usd("0.00"), "unallocated": usd("900000.00"),
			"plannedCost": usd("0.00"), "overAllocated": false},
		"createdBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})
	var view budget.GlobalBudgetView
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
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	server.Advance(90 * time.Minute)

	saved := carlos.Do(t, http.MethodPatch, "/api/v1/global-budgets/GB-1001", map[string]any{"amount": usd("2600000.00"), "version": 1})
	stale := carlos.Do(t, http.MethodPatch, "/api/v1/global-budgets/GB-1001", map[string]any{"name": "Old", "version": 1})
	gone := carlos.Do(t, http.MethodPatch, "/api/v1/global-budgets/GB-9999", map[string]any{"version": 1})

	assertFields(t, saved, http.StatusOK, map[string]any{
		"amount": usd("2600000.00"), "version": 2, "updatedAt": "2026-03-12T10:30:00Z",
		"updatedBy": map[string]any{"id": "U-1005", "name": "Carlos Mendes"},
	})
	assertAnswer(t, stale, http.StatusConflict, `{"error":{"code":"version_conflict","message":"Not saved — Carlos Mendes changed this global budget",`+
		`"fields":[],"conflict":{"changedBy":"Carlos Mendes","changedAt":"2026-03-12T10:30:00Z"}}}`)
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
	log := carlos.Do(t, http.MethodGet, "/api/v1/activity-log?recordType=globalBudget&recordId=GB-1001", nil)
	var page history.LogPage
	decode(t, log, http.StatusOK, &page)
	if page.Total != 1 || changesText(page.Items[0].Changes) != "Amount: 2,500,000.00 USD → 2,600,000.00 USD" {
		t.Errorf("log = %s", log.Body)
	}
}

// A global budget with allocations cannot be deleted (409 in_use, message 30); one without goes to the
// recycle bin, and the answer offers Undo (FR-035).
func TestDeleteGlobalBudget_409_200_404_FR035(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	carlos := server.SignIn(t, carlosEmail)
	created := carlos.Do(t, http.MethodPost, "/api/v1/global-budgets", map[string]any{
		"name": "Spare", "year": 2026, "amount": usd("1.00"), "ownerId": "U-1005",
	})
	if created.Status != http.StatusCreated {
		t.Fatalf("create = %d %s", created.Status, created.Body)
	}

	inUse := carlos.Do(t, http.MethodDelete, "/api/v1/global-budgets/GB-1001", nil)
	deleted := carlos.Do(t, http.MethodDelete, "/api/v1/global-budgets/GB-1002", nil)
	gone := carlos.Do(t, http.MethodGet, "/api/v1/global-budgets/GB-1002", nil)

	assertAnswer(t, inUse, http.StatusConflict, `{"error":{"code":"in_use","message":"'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first.","fields":[]}}`)
	var result history.DeleteResult
	decode(t, deleted, http.StatusOK, &result)
	if result.RecycleBinEntry.RecordType != "globalBudget" || result.RecycleBinEntry.RecordName != "Spare" {
		t.Errorf("delete = %s", deleted.Body)
	}
	assertAnswer(t, gone, http.StatusNotFound, budgetGoneBody)
	assertAnswer(t, carlos.Do(t, http.MethodDelete, "/api/v1/global-budgets/GB-1002", nil), http.StatusNotFound, budgetGoneBody)
}
