package budget

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// fixture is the service under test with its store, fakes and history.
type fixture struct {
	service *Service
	db      *store.DB
	clock   *testClock
	plans   *fakePlans
	access  *fakeAccess
	history *history.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONStore(), generator, clock.Now)
	f := &fixture{db: db, clock: clock, plans: newFakePlans(t), access: &fakeAccess{}, history: hist}
	tools := history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}
	f.service = NewService(tools, NewJSONStore(), Deps{
		Plans: f.plans, Directory: fakeDirectory{}, Rates: fakeRates{t: t}, Access: f.access,
	})
	for _, kind := range append([]history.Kind{f.plans.planKind()}, f.service.Kinds()...) {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return f
}

// as makes the fixture's service answer with the access rules a, for the rest of the test.
func (f *fixture) as(a *fakeAccess) {
	*f.access = *a
}

// budgetInput is the whole form of a new global budget: FY2026 Global marketing budget, 2,500,000.00 USD,
// owned by Carlos.
func budgetInput() GlobalBudgetInput {
	return GlobalBudgetInput{
		Name: "FY2026 Global marketing budget", Year: ptr(2026), Amount: &money.Input{Amount: "2500000.00", Currency: "USD"},
		OwnerID: carlos.ID, Description: "Shared out to market plans by Global Marketing Ops.",
	}
}

// create creates a global budget and stops the test when that fails.
func (f *fixture) create(t *testing.T, in GlobalBudgetInput) GlobalBudgetView {
	t.Helper()
	view, err := f.service.Create(t.Context(), carlos, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return view
}

// allocate allocates amount (USD) of the global budget id to planID from the plan's current version and
// stops the test when that fails.
func (f *fixture) allocate(t *testing.T, id, planID, amount string) GlobalBudgetView {
	t.Helper()
	view, err := f.service.Allocate(t.Context(), carlos, id, AllocationInput{
		PlanID: planID, Amount: &money.Input{Amount: amount, Currency: f.plans.plans[planID].Currency},
		PlanVersion: ptr(f.plans.plans[planID].Version),
	})
	if err != nil {
		t.Fatalf("Allocate(%s): %v", planID, err)
	}
	return view
}

// demo is the demo global budget with its two allocations: 1,200,000.00 to Lubricants Europe and
// 800,000.00 to Mobility China (FR-040).
func (f *fixture) demo(t *testing.T) GlobalBudgetView {
	t.Helper()
	b := f.create(t, budgetInput())
	f.allocate(t, b.ID, planEurope, "1200000.00")
	return f.allocate(t, b.ID, planChina, "800000.00")
}

// stored reads the global budget id from the store.
func (f *fixture) stored(t *testing.T, id string) (GlobalBudget, bool) {
	t.Helper()
	var b GlobalBudget
	var found bool
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		b, found, err = NewJSONStore().GetBudget(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	return b, found
}

// allocations reads every stored allocation.
func (f *fixture) allocations(t *testing.T) []Allocation {
	t.Helper()
	var all []Allocation
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		all, err = NewJSONStore().ListAllocations(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("ListAllocations: %v", err)
	}
	return all
}

// entries returns the activity log of the global budget id, newest first.
func (f *fixture) entries(t *testing.T, id string) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), carlos, history.LogFilter{RecordType: kindBudget, RecordID: id, Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// assertError checks that err is the API error want: same status, code, message, fields and conflict.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) {
		t.Fatalf("error = %v, want %s", err, want)
	}
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if got.Status != want.Status || string(gotJSON) != string(wantJSON) {
		t.Errorf("error = %d %s\nwant    %d %s", got.Status, gotJSON, want.Status, wantJSON)
	}
}

// amount parses a money amount and stops the test when it is not one.
func amount(t *testing.T, text string) *money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return &a
}

func ptr[T any](v T) *T { return &v }
