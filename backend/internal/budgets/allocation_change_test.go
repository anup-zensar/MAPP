package budgets

import (
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Changing an allocation sets the plan's budget in the same save and logs both; the allocation's and the
// plan's versions are both checked; an unchanged amount changes and logs nothing (FR-030, US6 scenario 2).
func TestUpdateAllocation_FR030_FR037(t *testing.T) { //nolint:gocyclo // 13: one change and five refused or unchanged ones, then the store, the log and the calls
	t.Parallel()
	f := newFixture(t)
	demo := f.demo(t)
	f.clock.Advance(time.Hour)

	view, err := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1300000.00", Currency: "USD"}, Version: ptr(1), PlanVersion: ptr(1),
	})
	_, staleAllocation := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1.00", Currency: "USD"}, Version: ptr(1), PlanVersion: ptr(2),
	})
	_, stalePlan := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1.00", Currency: "USD"}, Version: ptr(2), PlanVersion: ptr(1),
	})
	_, missing := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{})
	_, gone := f.service.UpdateAllocation(t.Context(), carlos, "BA-9999", AllocationUpdate{})
	unchanged, errUnchanged := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1300000.00", Currency: "USD"}, Version: ptr(2), PlanVersion: ptr(1),
	})

	if err != nil || view.Allocations[0].Amount.Format() != "1,300,000.00 USD" || view.Allocations[0].Version != 2 ||
		f.plans.plans[planEurope].Budget.String() != "1300000.00" {
		t.Fatalf("update = %+v, %v", view.Allocations, err)
	}
	assertError(t, staleAllocation, errs.VersionConflict("budget allocation", "Carlos Mendes", start.Add(time.Hour)))
	assertError(t, stalePlan, errs.VersionConflict("plan", "Priya Raman", start))
	assertError(t, missing, errs.Validation(
		errs.FieldError{Field: "amount", Message: "Enter an amount of 0 or more."},
		errs.FieldError{Field: "version", Message: "Reload the page and try again."},
		errs.FieldError{Field: "planVersion", Message: "Reload the page and try again."},
	))
	assertError(t, gone, errs.NotFound("This allocation no longer exists — someone may have removed it."))
	if errUnchanged != nil || unchanged.Allocations[0].Version != 2 || f.plans.plans[planEurope].Budget.String() != "1300000.00" {
		t.Errorf("unchanged = %+v, %v", unchanged.Allocations, errUnchanged)
	}
	if stored := f.allocations(t); stored[0].Amount.Amount.String() != "1300000.00" || stored[0].Version != 2 {
		t.Errorf("stored = %+v, want 1300000.00 at version 2 after the refused changes", stored[0])
	}
	entries := f.entries(t, demo.ID)
	if len(entries) != 4 || changesText(entries[0].Changes) != "Allocation to 'FY2026 Lubricants Europe': 1,200,000.00 USD → 1,300,000.00 USD" {
		t.Errorf("log = %d entries, newest %+v", len(entries), entries[0])
	}
	// The demo's two allocations, the change, and the refused change of the stale plan; the unchanged save
	// does not call.
	wantCalls := []budgetCall{
		{planID: planEurope, amount: "1200000.00", version: 1, comment: "Allocated from 'FY2026 Global marketing budget'"},
		{planID: planChina, amount: "800000.00", version: 1, comment: "Allocated from 'FY2026 Global marketing budget'"},
		{planID: planEurope, amount: "1300000.00", version: 1, comment: "Allocated from 'FY2026 Global marketing budget'"},
		{planID: planEurope, amount: "1.00", version: 1, comment: "Allocated from 'FY2026 Global marketing budget'"},
	}
	if !slices.Equal(f.plans.calls, wantCalls) {
		t.Errorf("plan calls = %+v\nwant %+v", f.plans.calls, wantCalls)
	}
}

// Changing or removing an allocation needs "Create global budgets and allocate them to plans" in the plan's
// scope (FR-030).
func TestChangeAndRemoveAllocation_NeedTheirPlansScope_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.demo(t)
	f.access.manageIn = map[string]error{planEurope: refusal(refusedPlan)}

	_, errUpdate := f.service.UpdateAllocation(t.Context(), priya, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1.00", Currency: "USD"}, Version: ptr(1), PlanVersion: ptr(2),
	})
	_, errRemove := f.service.RemoveAllocation(t.Context(), priya, "BA-1001")

	assertError(t, errUpdate, refusal(refusedPlan))
	assertError(t, errRemove, refusal(refusedPlan))
	if n := len(f.allocations(t)); n != 2 {
		t.Errorf("allocations = %d, want 2", n)
	}
}

// An allocation whose plan the person cannot see answers like a missing one, and nothing changes (022 FR-005).
func TestChangeAndRemoveAllocation_HiddenPlanIsNotFound_022_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.demo(t)
	f.access.hidden = map[string]bool{planEurope: true}

	_, errUpdate := f.service.UpdateAllocation(t.Context(), priya, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1.00", Currency: "USD"}, Version: ptr(1), PlanVersion: ptr(2),
	})
	_, errRemove := f.service.RemoveAllocation(t.Context(), priya, "BA-1001")

	gone := errs.NotFound("This allocation no longer exists — someone may have removed it.")
	assertError(t, errUpdate, gone)
	assertError(t, errRemove, gone)
	if n := len(f.allocations(t)); n != 2 {
		t.Errorf("allocations = %d, want 2", n)
	}
}

// Removing an allocation deletes it outright: the plan keeps its budget, which is editable again, the plan's
// log says the allocation was removed, and the global budget's log keeps the amount (FR-034, US6 scenario 6).
func TestRemoveAllocation_KeepsThePlanBudget_FR034(t *testing.T) { //nolint:gocyclo // 11: one removal, then the page, the plan, its allocation and the log
	t.Parallel()
	f := newFixture(t)
	demo := f.demo(t)

	view, err := f.service.RemoveAllocation(t.Context(), carlos, "BA-1002")
	_, gone := f.service.RemoveAllocation(t.Context(), carlos, "BA-1002")

	if err != nil || len(view.Allocations) != 1 || view.Figures.Allocated.Format() != "1,200,000.00 USD" ||
		len(view.PlansWithoutAllocation) != 2 || view.PlansWithoutAllocation[1].Plan.ID != planChina ||
		view.PlansWithoutAllocation[1].Budget.Format() != "800,000.00 USD" {
		t.Fatalf("remove = %+v, %v", view, err)
	}
	assertError(t, gone, errs.NotFound("This allocation no longer exists — someone may have removed it."))
	if !slices.Equal(f.plans.removals, []string{"P-1002 Allocation from 'FY2026 Global marketing budget' removed"}) ||
		f.plans.plans[planChina].Budget.String() != "800000.00" {
		t.Errorf("plan = %v, budget %s", f.plans.removals, f.plans.plans[planChina].Budget)
	}
	if _, allocated, _ := f.service.AllocationOf(t.Context(), planChina); allocated {
		t.Errorf("the plan still has an allocation")
	}
	entries := f.entries(t, demo.ID)
	if changesText(entries[0].Changes) != "Allocation to 'FY2026 Mobility China': 800,000.00 USD → Not allocated" {
		t.Errorf("log = %+v", entries[0])
	}
}

// The Allocate dialog offers the plans of the global budget's year without an allocation that the person
// may allocate to, by name (FR-030).
func TestAllocationOptions_FR030(t *testing.T) { //nolint:gocyclo // 12: the options of every plan, then of the plans in scope
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.allocate(t, b.ID, planChina, "800000.00")

	options, err := f.service.AllocationOptions(t.Context(), carlos, b.ID)
	f.access.manageIn = map[string]error{planBrazil: refusal(refusedPlan)}
	scoped, errScoped := f.service.AllocationOptions(t.Context(), carlos, b.ID)
	_, gone := f.service.AllocationOptions(t.Context(), carlos, "GB-9999")

	if err != nil || options.Total != 2 || options.Items[0].Plan.Name != "FY2026 Brazil" || options.Items[1].Plan.Name != "FY2026 Lubricants Europe" ||
		options.Items[1].Budget.Format() != "1,200,000.00 USD" || options.Items[1].Owner.Name != "Priya Raman" ||
		len(options.Items[1].Teams) != 1 || options.Items[1].Teams[0].Name != "Lubricants Europe" {
		t.Errorf("options = %+v, %v", options, err)
	}
	if errScoped != nil || scoped.Total != 1 || scoped.Items[0].Plan.ID != planEurope {
		t.Errorf("scoped options = %+v, %v; want Lubricants Europe only", scoped, errScoped)
	}
	assertError(t, gone, errs.NotFound("This global budget no longer exists — someone may have deleted it."))
}

// A plan's allocation, for the plan page and the budget lock: its global budget, and whether the reader may
// open it (FR-031, FR-033).
func TestAllocationOf_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.demo(t)

	found, allocated, err := f.service.AllocationOf(t.Context(), planEurope)
	f.access.noView = true
	hidden, _, _ := f.service.AllocationOf(t.Context(), planEurope)
	_, none, errNone := f.service.AllocationOf(t.Context(), planBrazil)

	if err != nil || !allocated || found.Allocation.ID != "BA-1001" || found.GlobalBudget.Name != "FY2026 Global marketing budget" || !found.CanOpen {
		t.Errorf("allocation = %+v, %v, %v", found, allocated, err)
	}
	if hidden.CanOpen {
		t.Errorf("CanOpen = true without globalBudgets.view")
	}
	if none || errNone != nil {
		t.Errorf("a plan without allocation = %v, %v", none, errNone)
	}
}
