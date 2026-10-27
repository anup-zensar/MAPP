package budgets

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// The service tests run the real service on a temporary JSON store, with fakes for plans, people, rates
// and the access rules. The clock starts at 12 Mar 2026 09:00 UTC, so a global budget's year is 2025–2031.

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

var (
	carlos = actor.Actor{ID: "U-1005", Name: "Carlos Mendes"} // the global planner
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}   // a local planner
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}   // the administrator
)

// The fake plans: Lubricants Europe and Mobility China of 2026, a 2026 plan of Brazil in BRL (016), and a
// 2027 plan.
const (
	planEurope = "P-1001"
	planChina  = "P-1002"
	planBrazil = "P-1003"
	plan2027   = "P-1004"
)

// testClock is a clock the test moves forward.
type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// budgetCall is one call of Plans.SetBudgetFromAllocation.
type budgetCall struct {
	planID, amount, comment string
	version                 int
}

// fakePlans is Plans over plans kept in memory. SetBudgetFromAllocation behaves as the plans service: an
// out-of-date version is refused naming Priya, and a changed budget gives the plan a new version.
type fakePlans struct {
	plans    map[string]PlanInfo
	planned  map[string]*money.Amount
	calls    []budgetCall
	removals []string // "P-1002 Allocation from '…' removed"
}

func newFakePlans(t *testing.T) *fakePlans {
	return &fakePlans{
		plans: map[string]PlanInfo{
			planEurope: {ID: planEurope, Name: "FY2026 Lubricants Europe", Year: 2026, Status: "published", Version: 1,
				TeamIDs: []string{"T-1001"}, OwnerID: "U-1002", CreatedByID: "U-1001", Currency: "USD", Budget: amount(t, "1200000.00")},
			planChina: {ID: planChina, Name: "FY2026 Mobility China", Year: 2026, Status: "draft", Version: 1,
				TeamIDs: []string{"T-1002"}, OwnerID: "U-1004", CreatedByID: "U-1001", Currency: "USD", Budget: amount(t, "800000.00")},
			planBrazil: {ID: planBrazil, Name: "FY2026 Brazil", Year: 2026, Status: "draft", Version: 1,
				TeamIDs: []string{}, CreatedByID: "U-1001", Currency: "BRL"},
			plan2027: {ID: plan2027, Name: "FY2027 Global Brand", Year: 2027, Status: "draft", Version: 1,
				TeamIDs: []string{"T-1003"}, OwnerID: "U-1001", CreatedByID: "U-1001", Currency: "USD", Budget: amount(t, "500000.00")},
		},
		planned: map[string]*money.Amount{planEurope: amount(t, "750000.00"), planChina: amount(t, "300000.00")},
	}
}

func (f *fakePlans) Plan(_ context.Context, id string) (PlanInfo, bool, error) {
	p, found := f.plans[id]
	return p, found, nil
}

func (f *fakePlans) PlansOfYear(_ context.Context, year int) ([]PlanInfo, error) {
	of := []PlanInfo{}
	for _, id := range slices.Sorted(maps.Keys(f.plans)) {
		if f.plans[id].Year == year {
			of = append(of, f.plans[id])
		}
	}
	return of, nil
}

func (f *fakePlans) PlannedCosts(_ context.Context, planIDs []string) (map[string]*money.Amount, error) {
	costs := map[string]*money.Amount{}
	for _, id := range planIDs {
		costs[id] = f.planned[id]
	}
	return costs, nil
}

func (f *fakePlans) SetBudgetFromAllocation(_ context.Context, _ actor.Actor, planID string, amount money.Amount, version int, comment string) error {
	f.calls = append(f.calls, budgetCall{planID: planID, amount: amount.String(), comment: comment, version: version})
	p, found := f.plans[planID]
	if !found {
		return errs.NotFound("This plan no longer exists — someone may have deleted it.")
	}
	if version != p.Version {
		return errs.VersionConflict("plan", "Priya Raman", start)
	}
	if p.Budget == nil || p.Budget.Cmp(amount) != 0 {
		p.Budget, p.Version = &amount, p.Version+1
	}
	f.plans[planID] = p
	return nil
}

func (f *fakePlans) AllocationRemoved(_ context.Context, _ actor.Actor, planID, comment string) error {
	f.removals = append(f.removals, planID+" "+comment)
	return nil
}

// planKind is feature 004's history kind "plan" over the fake plans, so allocations can be its children.
func (f *fakePlans) planKind() history.Kind {
	return history.Kind{
		Name: kindPlan, Singular: "plan", Plural: "plans",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			p, found := f.plans[id]
			if !found {
				return history.Snapshot{}, false, nil
			}
			data, err := json.Marshal(p)
			return history.Snapshot{ID: p.ID, Name: p.Name, Data: data}, true, err
		},
		Remove: func(_ context.Context, id string) error { delete(f.plans, id); return nil },
		Put: func(_ context.Context, s history.Snapshot) error {
			var p PlanInfo
			if err := json.Unmarshal(s.Data, &p); err != nil {
				return err
			}
			f.plans[p.ID] = p
			return nil
		},
	}
}

// fakeDirectory is Directory over the demo people and teams.
type fakeDirectory struct{}

var directoryPeople = map[string]people.PersonRef{
	"U-1001": {ID: "U-1001", Name: "Alex Morgan", Initials: "AM", Active: true},
	"U-1002": {ID: "U-1002", Name: "Priya Raman", Initials: "PR", Active: true},
	"U-1004": {ID: "U-1004", Name: "Mei Lin", Initials: "ML", Active: true},
	"U-1005": {ID: "U-1005", Name: "Carlos Mendes", Initials: "CM", Active: true},
	"U-1006": {ID: "U-1006", Name: "Sara Novak", Initials: "SN", Active: false},
}

var directoryTeams = map[string]people.TeamRef{
	"T-1001": {ID: "T-1001", Name: "Lubricants Europe", Colour: "teal", Active: true},
	"T-1002": {ID: "T-1002", Name: "Mobility China", Colour: "amber", Active: true},
}

func (fakeDirectory) PersonRefs(_ context.Context, ids []string) (map[string]people.PersonRef, error) {
	refs := map[string]people.PersonRef{}
	for _, id := range ids {
		if p, found := directoryPeople[id]; found {
			refs[id] = p
		}
	}
	return refs, nil
}

func (fakeDirectory) TeamRefs(_ context.Context, ids []string) (map[string]people.TeamRef, error) {
	refs := map[string]people.TeamRef{}
	for _, id := range ids {
		if t, found := directoryTeams[id]; found {
			refs[id] = t
		}
	}
	return refs, nil
}

// fakeRates is Rates with BRL at 5.00 per USD.
type fakeRates struct{ t *testing.T }

func (f fakeRates) Rates(context.Context) (money.Rates, error) {
	brl, err := money.ParseRate("5")
	if err != nil {
		f.t.Fatalf("ParseRate: %v", err)
	}
	return money.Rates{"USD": money.One, "BRL": brl}, nil
}

// refusal is how the access rules refuse (014: 403 forbidden with their message).
func refusal(message string) *errs.Error {
	return &errs.Error{Status: http.StatusForbidden, Code: "forbidden", Message: message, Fields: []errs.FieldError{}}
}

const (
	refusedAll  = "Creating global budgets isn't part of your role (Planner). Ask an administrator if you need it."
	refusedPlan = "Creating global budgets is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."
)

// fakeAccess is Access with the answers a test sets: by default everything is allowed, every plan is
// visible and Carlos and Alex may own a global budget.
type fakeAccess struct {
	manageAll error
	manage    error
	manageIn  map[string]error // by plan id
	hidden    map[string]bool  // plans the person cannot see
	noView    bool             // the person does not hold globalBudgets.view
	managers  map[string]bool  // who holds globalBudgets.manage for All markets; nil: Carlos and Alex
}

func (f *fakeAccess) ManageAll(context.Context) error { return f.manageAll }
func (f *fakeAccess) Manage(context.Context) error    { return f.manage }
func (f *fakeAccess) ManageIn(_ context.Context, plan PlanInfo) error {
	return f.manageIn[plan.ID]
}
func (f *fakeAccess) SeeGlobalBudgets(context.Context) bool { return !f.noView }
func (f *fakeAccess) CanViewPlan(_ context.Context, plan PlanInfo) bool {
	return !f.hidden[plan.ID]
}
func (f *fakeAccess) CanManageAll(_ context.Context, personID string) (bool, error) {
	if f.managers == nil {
		return personID == carlos.ID || personID == alex.ID, nil
	}
	return f.managers[personID], nil
}

// priyasAccess is what a local planner of Lubricants Europe may do: see her plan only, manage nothing for
// all markets.
func priyasAccess() *fakeAccess {
	return &fakeAccess{
		manageAll: refusal(refusedAll), manage: refusal(refusedAll),
		manageIn: map[string]error{planEurope: refusal(refusedAll), planChina: refusal(refusedAll), planBrazil: refusal(refusedAll)},
		hidden:   map[string]bool{planChina: true, planBrazil: true},
	}
}

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

// as returns the fixture's service answering with the access rules a, for the rest of the test.
func (f *fixture) as(a *fakeAccess) *fixture {
	*f.access = *a
	return f
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

// entries returns the activity log of the record id, newest first.
func (f *fixture) entries(t *testing.T, kind, id string) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), carlos, history.LogFilter{RecordType: kind, RecordID: id, Limit: 200})
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
