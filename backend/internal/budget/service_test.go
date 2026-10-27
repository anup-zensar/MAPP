package budget

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The list shows the global budgets of the chosen year by name with their four totals, every year that has
// one, newest first, and whether the person may create one (FR-036).
func TestList_YearsTotalsAndCreate_FR036(t *testing.T) { //nolint:gocyclo // 13: three lists, then the rows, totals, years and create of each in turn
	t.Parallel()
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
func TestGet_FiguresAndRows_FR032(t *testing.T) { //nolint:gocyclo // 11: one page, then its figures, rows, other markets, plans without allocation and locks in turn
	t.Parallel()
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
	t.Parallel()
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

// Deleting a plan takes its allocation to the recycle bin with it, so the global budget no longer counts
// it; restoring the plan brings the allocation back (FR-035).
func TestPlanDelete_TakesTheAllocation_FR035(t *testing.T) { //nolint:gocyclo // 10: the page and the bin entry while the plan is in the bin, then the page after the restore
	t.Parallel()
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
	t.Parallel()
	f := newFixture(t)
	b := f.create(t, budgetInput())
	f.allocate(t, b.ID, planChina, "800000.00")
	entry, err := f.history.MoveToBin(t.Context(), carlos, kindPlan, planChina)
	if err != nil {
		t.Fatalf("MoveToBin: %v", err)
	}
	if _, err = f.service.Delete(t.Context(), carlos, b.ID); err != nil {
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
	t.Parallel()
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
