package plans

import (
	"context"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// carlos is the global planner who owns the global budget of these tests.
var carlos = actor.Actor{ID: "U-1005", Name: "Carlos Mendes"}

// fakeAllocations is 014's Allocations hook with the allocations the test sets, by plan id.
type fakeAllocations map[string]Allocation

func (f fakeAllocations) AllocationOf(_ context.Context, planID string) (Allocation, bool, error) {
	a, found := f[planID]
	return a, found, nil
}

// allocatedFixture is the fixture with one plan, "FY2026 Lubricants Test" (budget 1,200,000.00), that the
// global budget "FY2026 Global marketing budget" of Carlos Mendes (U-1005) allocates to.
func allocatedFixture(t *testing.T) (*fixture, PlanView) {
	t.Helper()
	f := newFixture(t)
	in := input("FY2026 Lubricants Test")
	in.Budget = &money.Input{Amount: "1200000.00", Currency: "USD"}
	created := f.create(t, priya, in)
	f.service.UseAllocations(fakeAllocations{created.ID: {
		ID: "BA-1001", GlobalBudgetID: "GB-1001", GlobalBudgetName: "FY2026 Global marketing budget", OwnerID: "U-1005",
		Amount: money.USD(usd(t, "1200000.00")), CanOpen: true,
	}})
	return f, created
}

// The plan page says where an allocated plan's budget comes from and whom to ask; a plan without an
// allocation has none (FR-031, FR-033).
func TestGet_AllocationOfThePlan_FR031_FR033(t *testing.T) {
	f, created := allocatedFixture(t)
	other := f.create(t, priya, input("FY2026 Other"))

	view, err := f.service.Get(t.Context(), priya, created.ID)
	plain, errPlain := f.service.Get(t.Context(), priya, other.ID)

	want := &PlanAllocation{
		AllocationID: "BA-1001", GlobalBudget: GlobalBudgetRef{ID: "GB-1001", Name: "FY2026 Global marketing budget"},
		Amount: money.USD(usd(t, "1200000.00")), CanOpen: true,
		Owner: &people.PersonRef{ID: "U-1005", Name: "Carlos Mendes", Active: true},
	}
	if err != nil || !reflect.DeepEqual(view.Allocation, want) {
		t.Errorf("allocation = %+v, %v\nwant %+v", view.Allocation, err, want)
	}
	if errPlain != nil || plain.Allocation != nil {
		t.Errorf("plan without allocation: allocation = %+v, %v; want nil", plain.Allocation, errPlain)
	}
}

// The budget of an allocated plan is read-only: a save that changes it is refused with 409 not_allowed and
// the sentence of the plan form, and changes nothing; a save that keeps it changes the rest (FR-031, US6
// scenario 3).
func TestUpdate_AllocatedBudgetIsReadOnly_FR031(t *testing.T) {
	f, created := allocatedFixture(t)
	changedBudget := updateOf(f.stored(t, created.ID))
	changedBudget.Budget = &money.Input{Amount: "1300000.00", Currency: "USD"}
	noBudget := updateOf(f.stored(t, created.ID))
	noBudget.Budget = nil
	sameBudget := updateOf(f.stored(t, created.ID))
	sameBudget.Description = "Spring campaign included."

	_, errChanged := f.service.Update(t.Context(), priya, created.ID, changedBudget)
	_, errCleared := f.service.Update(t.Context(), priya, created.ID, noBudget)
	saved, errSame := f.service.Update(t.Context(), priya, created.ID, sameBudget)

	refusal := errs.NotAllowed("Allocated from 'FY2026 Global marketing budget' — ask Carlos Mendes to change it.")
	assertError(t, errChanged, refusal)
	assertError(t, errCleared, refusal)
	if errSame != nil || saved.Description != "Spring campaign included." || saved.Budget == nil ||
		saved.Budget.Amount.String() != "1200000.00" || saved.Version != 2 {
		t.Errorf("same budget: %+v, %v; want the description saved, budget 1,200,000.00, version 2", saved, errSame)
	}
}

// An allocation sets the plan budget through the plans service (FR-030, D-235): the plan gets a new version
// and one "changed" entry, Plan budget 1,200,000.00 USD → 1,300,000.00 USD, with the comment naming the global
// budget (data model §3). A plan version that is out of date is refused naming who changed the plan.
func TestSetBudgetFromAllocation_FR030(t *testing.T) {
	f, created := allocatedFixture(t)
	f.clock.Advance(time.Hour)

	err := f.service.SetBudgetFromAllocation(t.Context(), carlos, created.ID, usd(t, "1300000.00"), 1,
		"Allocated from 'FY2026 Global marketing budget'")
	stale := f.service.SetBudgetFromAllocation(t.Context(), carlos, created.ID, usd(t, "1400000.00"), 1, "")

	stored := f.stored(t, created.ID)
	if err != nil || stored.Budget == nil || stored.Budget.String() != "1300000.00" || stored.Version != 2 || stored.UpdatedBy != carlos {
		t.Fatalf("stored = budget %v, version %d, by %+v, %v; want 1300000.00, version 2, by Carlos", stored.Budget,
			stored.Version, stored.UpdatedBy, err)
	}
	assertError(t, stale, errs.VersionConflict("plan", "Carlos Mendes", start.Add(time.Hour)))
	entries := f.entries(t)
	if len(entries) != 2 || entries[0].Action != history.ActionChanged || entries[0].Actor != carlos ||
		entries[0].Comment != "Allocated from 'FY2026 Global marketing budget'" || len(entries[0].Changes) != 1 {
		t.Fatalf("log = %+v, want the created entry and one changed entry by Carlos with the comment", entries)
	}
	c := entries[0].Changes[0]
	if c.Field != "budget" || c.Label != "Plan budget" || c.OldText != "1,200,000.00 USD" || c.NewText != "1,300,000.00 USD" {
		t.Errorf("change = %+v, want Plan budget 1,200,000.00 USD → 1,300,000.00 USD", c)
	}
}

// An allocation of the amount the plan already has (the plan is newly allocated) logs only the comment and
// keeps the plan's version (data model §3).
func TestSetBudgetFromAllocation_SameAmountLogsTheCommentOnly_FR030(t *testing.T) {
	f, created := allocatedFixture(t)

	err := f.service.SetBudgetFromAllocation(t.Context(), carlos, created.ID, usd(t, "1200000.00"), 1,
		"Allocated from 'FY2026 Global marketing budget'")

	entries := f.entries(t)
	if err != nil || f.stored(t, created.ID).Version != 1 || len(entries) != 2 || len(entries[0].Changes) != 0 ||
		entries[0].Comment != "Allocated from 'FY2026 Global marketing budget'" {
		t.Errorf("version %d, log %+v, %v; want version 1 and one changed entry with only the comment",
			f.stored(t, created.ID).Version, entries, err)
	}
}

// Removing an allocation logs on the plan that it was removed and leaves the plan, its budget and its
// version as they are (FR-034, data model §3).
func TestAllocationRemoved_FR034(t *testing.T) {
	f, created := allocatedFixture(t)

	err := f.service.AllocationRemoved(t.Context(), carlos, created.ID, "Allocation from 'FY2026 Global marketing budget' removed")

	stored := f.stored(t, created.ID)
	entries := f.entries(t)
	if err != nil || stored.Version != 1 || stored.Budget.String() != "1200000.00" || len(entries) != 2 ||
		entries[0].Action != history.ActionChanged || len(entries[0].Changes) != 0 ||
		entries[0].Comment != "Allocation from 'FY2026 Global marketing budget' removed" {
		t.Errorf("stored %+v, log %+v, %v", stored, entries, err)
	}
}

// A copy of an allocated plan copies its budget as a typed amount and has no allocation (FR-035).
func TestCopy_AllocatedPlanCopyHasNoAllocation_FR035(t *testing.T) {
	f, created := allocatedFixture(t)

	result, err := f.service.Copy(t.Context(), priya, created.ID, CopyRequest{Name: "FY2026 Lubricants Copy", Year: ptr(2026)})

	if err != nil || result.Plan.Allocation != nil || result.Plan.Budget == nil || result.Plan.Budget.Amount.String() != "1200000.00" {
		t.Errorf("copy = allocation %+v, budget %+v, %v; want no allocation and 1,200,000.00", result.Plan.Allocation,
			result.Plan.Budget, err)
	}
}

// PlannedCosts gives each plan's planned cost in USD from the levels below, nil for a plan without one, with
// one call per level (FR-032, SC-009).
func TestPlannedCosts_FR032(t *testing.T) {
	f := newFixture(t)
	first := f.create(t, priya, input("FY2026 First"))
	second := f.create(t, priya, input("FY2026 Second"))
	planned := usd(t, "750000.00")
	level := f.addLevel(t, "activity", map[string]Facts{first.ID: {PlannedCost: &planned}})

	costs, err := f.service.PlannedCosts(t.Context(), []string{first.ID, second.ID})

	if err != nil || costs[first.ID] == nil || costs[first.ID].String() != "750000.00" || costs[second.ID] != nil || level.factCalls != 1 {
		t.Errorf("costs = %v, %v, %d calls; want 750000.00 and nil, one call", costs, err, level.factCalls)
	}
}
