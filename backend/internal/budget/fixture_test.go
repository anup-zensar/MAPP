package budget

import (
	"encoding/json"
	"errors"
	"testing"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
)

type fixture struct {
	service *Service
	db      *fake.DB
	clock   *testClock
	plans   *fakePlans
	access  *fakeAccess
	history *history.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, clock.Now)
	f := &fixture{db: db, clock: clock, plans: newFakePlans(t), access: &fakeAccess{}, history: hist}
	tools := history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}
	f.service = NewService(tools, newFakeStore(db), Deps{
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

func budgetInput() GlobalBudgetInput {
	return GlobalBudgetInput{
		Name: "FY2026 Global marketing budget", Year: ptr(2026), Amount: &money.Input{Amount: "2500000.00", Currency: "USD"},
		OwnerID: carlos.ID, Description: "Shared out to market plans by Global Marketing Ops.",
	}
}

func (f *fixture) create(t *testing.T, in GlobalBudgetInput) GlobalBudgetView {
	t.Helper()
	view, err := f.service.Create(t.Context(), carlos, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return view
}

// allocate gives amount of the global budget id to planID, from the plan's current version.
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

// demo is the demo global budget: 1,200,000.00 to Lubricants Europe and 800,000.00 to Mobility China.
func (f *fixture) demo(t *testing.T) GlobalBudgetView {
	t.Helper()
	b := f.create(t, budgetInput())
	f.allocate(t, b.ID, planEurope, "1200000.00")
	return f.allocate(t, b.ID, planChina, "800000.00")
}

func (f *fixture) stored(t *testing.T, id string) (GlobalBudget, bool) {
	t.Helper()
	b, found, err := newFakeStore(f.db).GetBudget(t.Context(), id)
	if err != nil {
		t.Fatalf("GetBudget: %v", err)
	}
	return b, found
}

func (f *fixture) allocations(t *testing.T) []Allocation {
	t.Helper()
	all, err := newFakeStore(f.db).ListAllocations(t.Context())
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

// assertError compares the status and the whole API error: code, message, fields and conflict.
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

func amount(t *testing.T, text string) *money.Amount {
	t.Helper()
	a, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return &a
}

func ptr[T any](v T) *T { return &v }
