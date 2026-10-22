package budgets

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// A new global budget gets the first ID, version 1 and a "created" entry; its page has no allocation yet,
// the year lock, and every action for the global planner (FR-029, FR-037).
func TestCreate_GlobalBudget_FR029(t *testing.T) {
	f := newFixture(t)

	view, err := f.service.Create(t.Context(), carlos, budgetInput())

	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got := fmt.Sprintf("%s %s %d %s %s %q v%d by %s", view.ID, view.Name, view.Year, view.Amount.Format(), view.Owner.Name,
		view.Description, view.Version, view.CreatedBy.Name)
	want := `GB-1001 FY2026 Global marketing budget 2026 2,500,000.00 USD Carlos Mendes "Shared out to market plans by Global Marketing Ops." v1 by Carlos Mendes`
	if got != want {
		t.Errorf("created = %s\nwant      %s", got, want)
	}
	if figures := figuresText(view.Figures); figures != "2,500,000.00 USD allocated 0.00 USD unallocated 2,500,000.00 USD planned 0.00 USD" {
		t.Errorf("figures = %s", figures)
	}
	if !slices.Equal(view.Locks, []settings.Lock{{Field: "year", Reason: "The year can't change once the global budget is saved."}}) {
		t.Errorf("locks = %+v, want the year lock only", view.Locks)
	}
	if view.Access != (BudgetAccess{Edit: available, Delete: available, Allocate: available}) {
		t.Errorf("access = %+v, want everything available", view.Access)
	}
	entries := f.entries(t, kindBudget, view.ID)
	if len(entries) != 1 || entries[0].Action != history.ActionCreated || entries[0].Actor != carlos ||
		entries[0].RecordName != "FY2026 Global marketing budget" || entries[0].RecordTypeLabel != "global budget" {
		t.Errorf("log = %+v, want one created entry by Carlos", entries)
	}
}

// Every problem of the form is reported at once and nothing is saved (FR-029, messages 24, 26 and 27).
func TestCreate_EveryProblemAtOnce_FR029(t *testing.T) {
	f := newFixture(t)
	f.create(t, budgetInput())
	tooLong := strings.Repeat("x", 121)
	cases := []struct {
		name   string
		change func(in *GlobalBudgetInput)
		fields []errs.FieldError
	}{
		{"everything wrong", func(in *GlobalBudgetInput) {
			in.Name, in.Year, in.Amount, in.OwnerID, in.Description = " ", ptr(2032), nil, "", strings.Repeat("d", 2001)
		}, []errs.FieldError{
			{Field: "name", Message: "Enter a name for the global budget."},
			{Field: "year", Message: "Choose a year from 2025 to 2031."},
			{Field: "amount", Message: "Enter an amount of 0 or more."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
			{Field: "description", Message: "Use 2,000 characters or fewer."},
		}},
		{"name taken ignoring case, negative amount, owner who cannot manage all markets", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = "fy2026 GLOBAL marketing budget", &money.Input{Amount: "-1.00", Currency: "USD"}, priya.ID
		}, []errs.FieldError{
			{Field: "name", Message: "A global budget called 'FY2026 Global marketing budget' already exists."},
			{Field: "amount", Message: "Enter an amount of 0 or more."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
		{"name too long, three decimals, unknown owner", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = tooLong, &money.Input{Amount: "1.005", Currency: "USD"}, "U-9999"
		}, []errs.FieldError{
			{Field: "name", Message: "Use 120 characters or fewer."},
			{Field: "amount", Message: "Use at most 2 decimals."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
		{"amount in another currency, inactive owner", func(in *GlobalBudgetInput) {
			in.Name, in.Amount, in.OwnerID = "Other", &money.Input{Amount: "1.00", Currency: "EUR"}, "U-1006"
		}, []errs.FieldError{
			{Field: "amount", Message: "Enter the amount in USD."},
			{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := budgetInput()
			c.change(&in)

			_, err := f.service.Create(t.Context(), carlos, in)

			assertError(t, err, errs.Validation(c.fields...))
		})
	}
	if _, found := f.stored(t, "GB-1002"); found {
		t.Errorf("a refused form saved GB-1002")
	}
}

// Creating, changing and deleting a global budget needs "Create global budgets and allocate them to plans"
// for All markets (FR-029): a local planner is refused with the access rules' message and nothing changes.
func TestCreateUpdateDelete_NeedManageForAllMarkets_FR029(t *testing.T) {
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.as(priyasAccess())

	_, errCreate := f.service.Create(t.Context(), priya, budgetInput())
	_, errUpdate := f.service.Update(t.Context(), priya, b.ID, GlobalBudgetUpdate{Name: ptr("Mine"), Version: ptr(1)})
	_, errDelete := f.service.Delete(t.Context(), priya, b.ID)

	for _, err := range []error{errCreate, errUpdate, errDelete} {
		assertError(t, err, refusal(refusedAll))
	}
	if stored, found := f.stored(t, b.ID); !found || stored.Name != b.Name || stored.Version != 1 {
		t.Errorf("stored = %+v, %v; want it unchanged", stored, found)
	}
}

// Saving changed fields gives a new version and one "changed" entry with every field in words; the year
// cannot be sent; an out-of-date version is refused naming who changed it; a missing version is a field
// problem; a save that changes nothing changes nothing (FR-029, FR-037, data model §3).
func TestUpdate_ChangedFieldsAndVersion_FR029_FR037(t *testing.T) {
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.clock.Advance(time.Hour)

	saved, err := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{
		Name: ptr(" FY2026 Global budget "), Amount: &money.Input{Amount: "3000000.00", Currency: "USD"}, Version: ptr(1),
	})
	_, stale := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Description: ptr("Old"), Version: ptr(1)})
	_, missing := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Description: ptr("New")})
	same, errSame := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{Name: ptr("FY2026 Global budget"), Version: ptr(2)})

	if err != nil || saved.Name != "FY2026 Global budget" || saved.Amount.Format() != "3,000,000.00 USD" || saved.Version != 2 ||
		!saved.UpdatedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	assertError(t, stale, errs.VersionConflict("global budget", "Carlos Mendes", start.Add(time.Hour)))
	assertError(t, missing, errs.Validation(errs.FieldError{Field: "version", Message: "Reload the page and try again."}))
	if errSame != nil || same.Version != 2 {
		t.Errorf("unchanged save = version %d, %v; want version 2", same.Version, errSame)
	}
	entries := f.entries(t, kindBudget, b.ID)
	if len(entries) != 2 || entries[0].Action != history.ActionChanged {
		t.Fatalf("log = %+v, want created and one changed entry", entries)
	}
	if got := changesText(entries[0].Changes); got != "Amount: 2,500,000.00 USD → 3,000,000.00 USD; Name: FY2026 Global marketing budget → FY2026 Global budget" {
		t.Errorf("changes = %s", got)
	}
}

// A new owner must manage global budgets for all markets (message 27) and is logged by name; an owner the
// global budget already has stays when other fields change, even after losing that role (FR-029, FR-037).
func TestUpdate_Owner_FR029_FR037(t *testing.T) {
	f := newFixture(t)
	b := f.create(t, budgetInput())

	_, refused := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(priya.ID), Version: ptr(1)})
	handed, err := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(alex.ID), Version: ptr(1)})
	f.access.managers = map[string]bool{} // Alex loses the role
	kept, errKept := f.service.Update(t.Context(), carlos, b.ID, GlobalBudgetUpdate{OwnerID: ptr(alex.ID), Description: ptr("New"), Version: ptr(2)})

	assertError(t, refused, errs.Validation(errs.FieldError{Field: "ownerId", Message: "Choose a person who can manage global budgets for all markets."}))
	if err != nil || handed.Owner.Name != "Alex Morgan" {
		t.Fatalf("handed = %+v, %v", handed.Owner, err)
	}
	if errKept != nil || kept.Description != "New" || kept.Owner.Name != "Alex Morgan" {
		t.Errorf("kept = %+v, %v", kept, errKept)
	}
	entries := f.entries(t, kindBudget, b.ID)
	if got := changesText(entries[1].Changes); got != "Owner: Carlos Mendes → Alex Morgan" {
		t.Errorf("owner change = %s", got)
	}
}

// A global budget with allocations cannot be deleted (409 in_use, message 30, one plan and several plans);
// one without goes to the recycle bin and comes back with Undo (FR-035).
func TestDelete_InUseOrToTheBin_FR035(t *testing.T) {
	f := newFixture(t)
	demo := f.demo(t)
	single := f.create(t, GlobalBudgetInput{Name: "Single", Year: ptr(2026), Amount: &money.Input{Amount: "0.00", Currency: "USD"}, OwnerID: carlos.ID})
	f.allocate(t, single.ID, planBrazil, "10.00")
	empty := f.create(t, GlobalBudgetInput{Name: "Empty", Year: ptr(2027), Amount: &money.Input{Amount: "0.00", Currency: "USD"}, OwnerID: carlos.ID})

	_, errMany := f.service.Delete(t.Context(), carlos, demo.ID)
	_, errOne := f.service.Delete(t.Context(), carlos, single.ID)
	entry, err := f.service.Delete(t.Context(), carlos, empty.ID)
	_, errGone := f.service.Delete(t.Context(), carlos, empty.ID)

	assertError(t, errMany, errs.InUse("'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first."))
	assertError(t, errOne, errs.InUse("'Single' has an allocation to 1 plan. Remove it first."))
	assertError(t, errGone, errs.NotFound("This global budget no longer exists — someone may have deleted it."))
	if err != nil || entry.RecordType != kindBudget || entry.RecordName != "Empty" {
		t.Fatalf("delete = %+v, %v", entry, err)
	}
	if _, err := f.history.Undo(t.Context(), carlos, entry.ID); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if stored, found := f.stored(t, empty.ID); !found || stored.Name != "Empty" {
		t.Errorf("after Undo = %+v, %v; want it back", stored, found)
	}
}

// A global budget restored from the bin after another took its name is renamed; its Undo is refused
// (001's name rule, FR-035).
func TestRestore_NameRule_FR035(t *testing.T) {
	f := newFixture(t)
	b := f.create(t, budgetInput())
	entry, err := f.service.Delete(t.Context(), carlos, b.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	f.create(t, budgetInput())

	_, errUndo := f.history.Undo(t.Context(), carlos, entry.ID)
	result, err := f.history.Restore(t.Context(), carlos, entry.ID)

	assertError(t, errUndo, errs.NameTaken("Can't undo — there is now another global budget called 'FY2026 Global marketing budget'. Restore it from the recycle bin instead."))
	if err != nil || result.RecordName != "FY2026 Global marketing budget (restored)" ||
		result.Details != "Renamed to 'FY2026 Global marketing budget (restored)' because the name was taken." {
		t.Errorf("restore = %+v, %v", result, err)
	}
}

// The list shows the global budgets of the chosen year by name with their four totals, every year that has
// one, newest first, and whether the person may create one (FR-036).
func TestList_YearsTotalsAndCreate_FR036(t *testing.T) {
	f := newFixture(t)
	f.demo(t)
	f.create(t, GlobalBudgetInput{Name: "Asia reserve", Year: ptr(2026), Amount: &money.Input{Amount: "100.00", Currency: "USD"}, OwnerID: carlos.ID})
	f.create(t, GlobalBudgetInput{Name: "FY2027 Global", Year: ptr(2027), Amount: &money.Input{Amount: "1.00", Currency: "USD"}, OwnerID: carlos.ID})

	of2026, err := f.service.List(t.Context(), carlos, ptr(2026))
	every, errEvery := f.service.List(t.Context(), carlos, nil)
	f.as(priyasAccess())
	forPriya, errPriya := f.service.List(t.Context(), priya, ptr(2025))

	if err != nil || errEvery != nil || errPriya != nil {
		t.Fatalf("List: %v, %v, %v", err, errEvery, errPriya)
	}
	var got []string
	for _, s := range of2026.Items {
		got = append(got, fmt.Sprintf("%s %s %d %s %d: %s", s.ID, s.Name, s.Year, s.Owner.Name, s.AllocationCount, figuresText(s.Figures)))
	}
	want := []string{
		"GB-1002 Asia reserve 2026 Carlos Mendes 0: 100.00 USD allocated 0.00 USD unallocated 100.00 USD planned 0.00 USD",
		"GB-1001 FY2026 Global marketing budget 2026 Carlos Mendes 2: 2,500,000.00 USD allocated 2,000,000.00 USD unallocated 500,000.00 USD planned 1,050,000.00 USD",
	}
	if !slices.Equal(got, want) || of2026.Total != 2 || !slices.Equal(of2026.Years, []int{2027, 2026}) || of2026.Create != available {
		t.Errorf("2026 = %s, total %d, years %v, create %+v", strings.Join(got, " | "), of2026.Total, of2026.Years, of2026.Create)
	}
	if every.Total != 3 {
		t.Errorf("every year = %d items, want 3", every.Total)
	}
	if forPriya.Total != 0 || len(forPriya.Items) != 0 || forPriya.Create != (Availability{Reason: refusedAll}) {
		t.Errorf("Priya's 2025 = %+v, want no item and create refused with the reason", forPriya)
	}
}

// The global budget page (US6 scenario 1): Global budget 2,500,000.00, Allocated 2,000,000.00, Unallocated
// 500,000.00 and the planned cost of the allocated plans; one row per plan by name with teams, owner,
// status, allocation, planned cost, left to plan and budget used; the plans of the year without an
// allocation (FR-032).
func TestGet_FiguresAndRows_FR032(t *testing.T) {
	f := newFixture(t)
	demo := f.demo(t)

	view, err := f.service.Get(t.Context(), carlos, demo.ID)

	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := figuresText(view.Figures); got != "2,500,000.00 USD allocated 2,000,000.00 USD unallocated 500,000.00 USD planned 1,050,000.00 USD" {
		t.Errorf("figures = %s", got)
	}
	want := []string{
		"BA-1001 FY2026 Lubricants Europe v1 published [Lubricants Europe] Priya Raman 1,200,000.00 USD (1,200,000.00 USD) planned 750,000.00 USD left 450,000.00 USD used 62.50 v1 edit=true remove=true",
		"BA-1002 FY2026 Mobility China v1 draft [Mobility China] Mei Lin 800,000.00 USD (800,000.00 USD) planned 300,000.00 USD left 500,000.00 USD used 37.50 v1 edit=true remove=true",
	}
	if got := rowsText(view.Allocations); !slices.Equal(got, want) {
		t.Errorf("rows =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if view.OtherMarkets != nil {
		t.Errorf("other markets = %+v, want none: Carlos sees every plan", view.OtherMarkets)
	}
	if len(view.PlansWithoutAllocation) != 1 || view.PlansWithoutAllocation[0].Plan.ID != planBrazil ||
		view.PlansWithoutAllocation[0].Budget != nil || !view.PlansWithoutAllocation[0].CanAllocate.Available {
		t.Errorf("plans without allocation = %+v, want FY2026 Brazil without budget", view.PlansWithoutAllocation)
	}
	wantLocks := []settings.Lock{
		{Field: "year", Reason: "The year can't change once the global budget is saved."},
		{Field: "delete", Reason: "'FY2026 Global marketing budget' has allocations to 2 plans. Remove them first."},
	}
	if !slices.Equal(view.Locks, wantLocks) || view.Access.Delete != (Availability{Reason: wantLocks[1].Reason}) {
		t.Errorf("locks = %+v, delete = %+v", view.Locks, view.Access.Delete)
	}
	_, errGone := f.service.Get(t.Context(), carlos, "GB-9999")
	assertError(t, errGone, errs.NotFound("This global budget no longer exists — someone may have deleted it."))
}

// A local planner sees only her plan's row; the other allocations are added up in one Other markets row,
// and the plans of the year without an allocation are those she can see. The totals still count every
// allocation (FR-033, US6).
func TestGet_OtherMarkets_FR033(t *testing.T) {
	f := newFixture(t)
	demo := f.demo(t)
	f.as(priyasAccess())

	view, err := f.service.Get(t.Context(), priya, demo.ID)

	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := []string{
		"BA-1001 FY2026 Lubricants Europe v1 published [Lubricants Europe] Priya Raman 1,200,000.00 USD (1,200,000.00 USD) planned 750,000.00 USD left 450,000.00 USD used 62.50 v1 edit=false remove=false",
	}
	if got := rowsText(view.Allocations); !slices.Equal(got, want) {
		t.Errorf("rows = %v", got)
	}
	if view.OtherMarkets == nil || *view.OtherMarkets != (OtherMarkets{PlanCount: 1, Amount: money.USD(*amount(t, "800000.00")), PlannedCost: money.USD(*amount(t, "300000.00"))}) {
		t.Errorf("other markets = %+v, want 1 plan, 800,000.00 and 300,000.00", view.OtherMarkets)
	}
	if got := figuresText(view.Figures); got != "2,500,000.00 USD allocated 2,000,000.00 USD unallocated 500,000.00 USD planned 1,050,000.00 USD" {
		t.Errorf("figures = %s", got)
	}
	if len(view.PlansWithoutAllocation) != 0 {
		t.Errorf("plans without allocation = %+v, want none she can see", view.PlansWithoutAllocation)
	}
	refused := Availability{Reason: refusedAll}
	if view.Access != (BudgetAccess{Edit: refused, Delete: refused, Allocate: refused}) || view.Allocations[0].Access.Edit != refused {
		t.Errorf("access = %+v, row %+v; want everything refused with the reason", view.Access, view.Allocations[0].Access)
	}
}

// Allocations that add up to more than the global budget save, and the figures show a negative
// unallocated amount, over-allocated (US6 scenario 5, never blocks as D-015).
func TestAllocate_OverAllocationNeverBlocks_FR032(t *testing.T) {
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
func TestAllocate_SetsThePlanBudgetAndLogs_FR030_FR037(t *testing.T) {
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
	entries := f.entries(t, kindBudget, b.ID)
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
func TestAllocate_Rules_FR030(t *testing.T) {
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
	f := newFixture(t)
	b := f.create(t, budgetInput())

	_, err := f.service.Allocate(t.Context(), carlos, b.ID, AllocationInput{
		PlanID: planEurope, Amount: &money.Input{Amount: "1.00", Currency: "USD"}, PlanVersion: ptr(7),
	})

	assertError(t, err, errs.VersionConflict("plan", "Priya Raman", start))
	if len(f.allocations(t)) != 0 || len(f.entries(t, kindBudget, b.ID)) != 1 {
		t.Errorf("allocations %+v, log %+v; want nothing saved", f.allocations(t), f.entries(t, kindBudget, b.ID))
	}
}

// A plan in another currency is allocated in that currency (D-303): its row shows the allocation, planned
// cost and left to plan in BRL and the allocation in USD, which the figures count at the current rate.
func TestAllocate_InThePlansCurrency_FR030(t *testing.T) {
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

// Changing an allocation sets the plan's budget in the same save and logs both; the allocation's and the
// plan's versions are both checked; an unchanged amount changes and logs nothing (FR-030, US6 scenario 2).
func TestUpdateAllocation_FR030_FR037(t *testing.T) {
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
	entries := f.entries(t, kindBudget, demo.ID)
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

// Removing an allocation deletes it outright: the plan keeps its budget, which is editable again, the plan's
// log says the allocation was removed, and the global budget's log keeps the amount (FR-034, US6 scenario 6).
func TestRemoveAllocation_KeepsThePlanBudget_FR034(t *testing.T) {
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
	entries := f.entries(t, kindBudget, demo.ID)
	if changesText(entries[0].Changes) != "Allocation to 'FY2026 Mobility China': 800,000.00 USD → Not allocated" {
		t.Errorf("log = %+v", entries[0])
	}
}

// The Allocate dialog offers the plans of the global budget's year without an allocation that the person
// may allocate to, by name (FR-030).
func TestAllocationOptions_FR030(t *testing.T) {
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

// Deleting a plan takes its allocation to the recycle bin with it, so the global budget no longer counts
// it; restoring the plan brings the allocation back (FR-035).
func TestPlanDelete_TakesTheAllocation_FR035(t *testing.T) {
	f := newFixture(t)
	demo := f.demo(t)

	entry, err := f.history.MoveToBin(t.Context(), carlos, kindPlan, planChina)
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	during, _ := f.service.Get(t.Context(), carlos, demo.ID)
	result, errRestore := f.history.Restore(t.Context(), carlos, entry.ID)
	after, _ := f.service.Get(t.Context(), carlos, demo.ID)

	if len(during.Allocations) != 1 || during.Figures.Allocated.Format() != "1,200,000.00 USD" || len(entry.Contents) != 1 ||
		entry.Contents[0].RecordType != kindAllocation {
		t.Errorf("while the plan is in the bin: rows %d, allocated %s, contents %+v", len(during.Allocations),
			during.Figures.Allocated.Format(), entry.Contents)
	}
	if errRestore != nil || result.Details != "" || len(after.Allocations) != 2 || after.Figures.Allocated.Format() != "2,000,000.00 USD" {
		t.Errorf("after the restore: %+v, %v; rows %d, allocated %s", result, errRestore, len(after.Allocations), after.Figures.Allocated.Format())
	}
}

// A plan restored after its global budget was deleted comes back without the allocation, with note 31, and
// keeps its budget (FR-035).
func TestPlanRestore_GlobalBudgetGone_FR035(t *testing.T) {
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.allocate(t, b.ID, planChina, "800000.00")
	entry, err := f.history.MoveToBin(t.Context(), carlos, kindPlan, planChina)
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	if _, err := f.service.Delete(t.Context(), carlos, b.ID); err != nil {
		t.Fatalf("Delete of the global budget without allocations: %v", err)
	}

	result, err := f.history.Restore(t.Context(), carlos, entry.ID)

	if err != nil || result.Details != "The allocation was not restored because its global budget was deleted." {
		t.Errorf("restore = %+v, %v", result, err)
	}
	if len(f.allocations(t)) != 0 || f.plans.plans[planChina].Budget.String() != "800000.00" {
		t.Errorf("allocations %+v, plan budget %s; want none and 800,000.00", f.allocations(t), f.plans.plans[planChina].Budget)
	}
}

// percentUsed rounds half up to two decimals and is null for an allocation of 0.00 (domain model §4.2).
func TestPercentUsed_FR032(t *testing.T) {
	cases := []struct{ planned, allocation, want string }{
		{"750000.00", "1200000.00", "62.50"}, {"1.00", "3.00", "33.33"}, {"2.00", "3.00", "66.67"},
		{"0.00", "5.00", "0.00"}, {"15.00", "10.00", "150.00"}, {"1.00", "0.00", "null"},
	}
	for _, c := range cases {
		got := "null"
		if p := percentUsed(*amount(t, c.planned), *amount(t, c.allocation)); p != nil {
			got = *p
		}
		if got != c.want {
			t.Errorf("percentUsed(%s, %s) = %s, want %s", c.planned, c.allocation, got, c.want)
		}
	}
}

// figuresText writes figures as "<amount> allocated <a> unallocated <u> planned <p>".
func figuresText(f Figures) string {
	return fmt.Sprintf("%s allocated %s unallocated %s planned %s", f.Amount.Format(), f.Allocated.Format(), f.Unallocated.Format(), f.PlannedCost.Format())
}

// rowsText writes the allocation rows one per line.
func rowsText(rows []AllocationRow) []string {
	var lines []string
	for _, r := range rows {
		var teams []string
		for _, team := range r.Teams {
			teams = append(teams, team.Name)
		}
		owner, planned, used := "-", "-", "null"
		if r.Owner != nil {
			owner = r.Owner.Name
		}
		if r.PlannedCost != nil {
			planned = r.PlannedCost.Format()
		}
		if r.BudgetUsedPercent != nil {
			used = *r.BudgetUsedPercent
		}
		lines = append(lines, fmt.Sprintf("%s %s v%d %s [%s] %s %s (%s) planned %s left %s used %s v%d edit=%v remove=%v",
			r.ID, r.Plan.Name, r.Plan.Version, r.Plan.Status, strings.Join(teams, ", "), owner, r.Amount.Format(), r.AmountUSD.Format(),
			planned, r.LeftToPlan.Format(), used, r.Version, r.Access.Edit.Available, r.Access.Remove.Available))
	}
	return lines
}

// changesText writes log changes as "<label>: <old> → <new>", joined by "; ".
func changesText(changes []history.Change) string {
	var parts []string
	for _, c := range changes {
		parts = append(parts, fmt.Sprintf("%s: %s → %s", c.Label, c.OldText, c.NewText))
	}
	return strings.Join(parts, "; ")
}
