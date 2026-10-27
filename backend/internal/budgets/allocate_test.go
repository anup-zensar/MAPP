package budgets

import (
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Allocations that add up to more than the global budget save, and the figures show a negative
// unallocated amount, over-allocated (US6 scenario 5, never blocks as D-015).
func TestAllocate_OverAllocationNeverBlocks_FR032(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	demo := f.demo(t)

	view, err := f.service.UpdateAllocation(t.Context(), carlos, "BA-1001", AllocationUpdate{
		Amount: &money.Input{Amount: "1800000.00", Currency: "USD"}, Version: ptr(1), PlanVersion: ptr(1),
	})

	if err != nil || figuresText(view.Figures) != "2,500,000.00 USD allocated 2,600,000.00 USD unallocated -100,000.00 USD planned 1,050,000.00 USD" ||
		!view.Figures.OverAllocated || demo.Figures.OverAllocated {
		t.Errorf("figures = %s over=%v, %v", figuresText(view.Figures), view.Figures.OverAllocated, err)
	}
}

// Allocating sets the plan's budget in the same save from the plan version the person saw, and logs the
// allocation on the global budget, Not allocated → 1,200,000.00 USD, and with a comment on the plan (FR-030,
// FR-037, data model §3).
func TestAllocate_SetsThePlanBudgetAndLogs_FR030_FR037(t *testing.T) { //nolint:gocyclo // 12: one allocation, then its row, the plan budget call and the log entry
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())

	view, err := f.service.Allocate(t.Context(), carlos, b.ID, AllocationInput{
		PlanID: " " + planEurope + " ", Amount: &money.Input{Amount: "1300000.00", Currency: "USD"}, PlanVersion: ptr(1),
	})

	if err != nil || len(view.Allocations) != 1 || view.Allocations[0].ID != "BA-1001" || view.Allocations[0].Amount.Format() != "1,300,000.00 USD" {
		t.Fatalf("allocate = %+v, %v", view.Allocations, err)
	}
	wantCall := []budgetCall{{planID: planEurope, amount: "1300000.00", version: 1, comment: "Allocated from 'FY2026 Global marketing budget'"}}
	if !slices.Equal(f.plans.calls, wantCall) || f.plans.plans[planEurope].Budget.String() != "1300000.00" {
		t.Errorf("plan budget calls = %+v, want %+v", f.plans.calls, wantCall)
	}
	entries := f.entries(t, b.ID)
	if len(entries) != 2 || entries[0].Action != history.ActionChanged ||
		changesText(entries[0].Changes) != "Allocation to 'FY2026 Lubricants Europe': Not allocated → 1,300,000.00 USD" ||
		entries[0].Changes[0].Field != "allocations.P-1001" {
		t.Errorf("log = %+v", entries)
	}
	if got := view.Allocations[0].Plan; got.Version != 2 {
		t.Errorf("row plan = %+v, want the plan's new version 2", got)
	}
}

// The allocation form: a plan is required and must be a plan of the global budget's year (message 28);
// one plan has one allocation across all global budgets (message 29); the amount is 0 or more in the plan's
// currency (D-303); the plan version is required. A person who may not manage global budgets in the plan's
// scope is refused (403). Nothing is saved (FR-030).
func TestAllocate_Rules_FR030(t *testing.T) { //nolint:tparallel // the checks after the subtests change the fixture they share
	t.Parallel()
	f := newFixture(t)
	demo := f.demo(t)
	other := f.create(t, GlobalBudgetInput{Name: "Other", Year: ptr(2026), Amount: &money.Input{Amount: "1.00", Currency: "USD"}, OwnerID: carlos.ID})
	cases := []struct {
		name   string
		in     AllocationInput
		fields []errs.FieldError
	}{
		{"nothing chosen", AllocationInput{}, []errs.FieldError{
			{Field: "planId", Message: "Choose a plan."},
			{Field: "amount", Message: "Enter an amount of 0 or more."},
			{Field: "planVersion", Message: "Reload the page and try again."},
		}},
		{"plan of another year", AllocationInput{PlanID: plan2027, Amount: &money.Input{Amount: "1.00", Currency: "USD"}, PlanVersion: ptr(1)},
			[]errs.FieldError{{Field: "planId", Message: "Choose a plan of 2026."}}},
		{"unknown plan", AllocationInput{PlanID: "P-9999", Amount: &money.Input{Amount: "1.00", Currency: "USD"}, PlanVersion: ptr(1)},
			[]errs.FieldError{{Field: "planId", Message: "Choose a plan of 2026."}}},
		{"plan allocated by another global budget", AllocationInput{PlanID: planEurope, Amount: &money.Input{Amount: "-1.00", Currency: "USD"}, PlanVersion: ptr(2)},
			[]errs.FieldError{
				{Field: "planId", Message: "'FY2026 Lubricants Europe' already has an allocation from 'FY2026 Global marketing budget'."},
				{Field: "amount", Message: "Enter an amount of 0 or more."},
			}},
		{"amount not in the plan's currency", AllocationInput{PlanID: planBrazil, Amount: &money.Input{Amount: "1.00", Currency: "USD"}, PlanVersion: ptr(1)},
			[]errs.FieldError{{Field: "amount", Message: "Enter the amount in BRL, the plan's currency."}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := f.service.Allocate(t.Context(), carlos, other.ID, c.in)

			assertError(t, err, errs.Validation(c.fields...))
		})
	}
	f.access.manageIn = map[string]error{planBrazil: refusal(refusedPlan)}
	_, refused := f.service.Allocate(t.Context(), priya, demo.ID, AllocationInput{
		PlanID: planBrazil, Amount: &money.Input{Amount: "1.00", Currency: "BRL"}, PlanVersion: ptr(1),
	})
	assertError(t, refused, refusal(refusedPlan))
	_, gone := f.service.Allocate(t.Context(), carlos, "GB-9999", AllocationInput{})
	assertError(t, gone, errs.NotFound("This global budget no longer exists — someone may have deleted it."))
	if n := len(f.allocations(t)); n != 2 {
		t.Errorf("allocations = %d, want the demo's 2 only", n)
	}
}

// An allocation from a plan version that is out of date is refused naming who changed the plan, and
// nothing is saved: the allocation would otherwise overwrite a newer plan budget unseen (FR-030).
func TestAllocate_PlanVersionConflict_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())

	_, err := f.service.Allocate(t.Context(), carlos, b.ID, AllocationInput{
		PlanID: planEurope, Amount: &money.Input{Amount: "1.00", Currency: "USD"}, PlanVersion: ptr(7),
	})

	assertError(t, err, errs.VersionConflict("plan", "Priya Raman", start))
	if len(f.allocations(t)) != 0 || len(f.entries(t, b.ID)) != 1 {
		t.Errorf("allocations %+v, log %+v; want nothing saved", f.allocations(t), f.entries(t, b.ID))
	}
}

// A plan in another currency is allocated in that currency (D-303): its row shows the allocation, planned
// cost and left to plan in BRL and the allocation in USD, which the figures count at the current rate.
func TestAllocate_InThePlansCurrency_FR030(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.plans.planned[planBrazil] = amount(t, "100.00") // USD

	view := f.allocate(t, b.ID, planBrazil, "5000.00")

	want := []string{"BA-1001 FY2026 Brazil v2 draft [] - 5,000.00 BRL (1,000.00 USD) planned 500.00 BRL left 4,500.00 BRL used 10.00 v1 edit=true remove=true"}
	if got := rowsText(view.Allocations); !slices.Equal(got, want) {
		t.Errorf("rows = %v\nwant %v", got, want)
	}
	if got := figuresText(view.Figures); got != "2,500,000.00 USD allocated 1,000.00 USD unallocated 2,499,000.00 USD planned 100.00 USD" {
		t.Errorf("figures = %s", got)
	}
}
