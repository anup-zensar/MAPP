package budget

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"mapp/internal/overspend"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The clock starts at 12 Mar 2026 09:00 UTC, so a global budget's year is 2025–2031.

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

var (
	carlos = actor.Actor{ID: "U-1005", Name: "Carlos Mendes"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
)

// The fake plans: Lubricants Europe and Mobility China of 2026, Brazil 2026 in BRL, and a 2027 plan.
const (
	planEurope = "P-1001"
	planChina  = "P-1002"
	planBrazil = "P-1003"
	plan2027   = "P-1004"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

type budgetCall struct {
	planID, amount, comment string
	version                 int
	overspend               overspend.Mode
}

// fakePlans refuses an out-of-date version naming Priya, and gives a changed budget a new version, as plans does.
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

func (f *fakePlans) SetBudgetFromAllocation(_ context.Context, _ actor.Actor, in PlanBudget) error {
	f.calls = append(f.calls, budgetCall{planID: in.PlanID, amount: in.Amount.String(), comment: in.Comment,
		version: in.PlanVersion, overspend: in.Overspend})
	p, found := f.plans[in.PlanID]
	if !found {
		return errs.NotFound("This plan no longer exists — someone may have deleted it.")
	}
	if in.PlanVersion != p.Version {
		return errs.VersionConflict("plan", "Priya Raman", start)
	}
	changed := in.Overspend != "" && in.Overspend != p.Overspend.Normalised()
	if p.Budget == nil || p.Budget.Cmp(in.Amount) != 0 || changed {
		p.Budget, p.Version = &in.Amount, p.Version+1
	}
	if in.Overspend != "" {
		p.Overspend = in.Overspend
	}
	f.plans[in.PlanID] = p
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

func refusal(message string) *errs.Error {
	return &errs.Error{Status: http.StatusForbidden, Code: "forbidden", Message: message, Fields: []errs.FieldError{}}
}

const (
	refusedAll  = "Creating global budgets isn't part of your role (Planner). Ask an administrator if you need it."
	refusedPlan = "Creating global budgets is allowed for you only in plans of Lubricants Europe. Ask an administrator if you need it here."
)

// fakeAccess allows everything by default: every plan is visible and Carlos and Alex may own a global budget.
type fakeAccess struct {
	manageAll error
	manage    error
	manageIn  map[string]error
	hidden    map[string]bool
	noView    bool            // the person does not hold globalBudgets.view
	managers  map[string]bool // who holds globalBudgets.manage for All markets; nil: Carlos and Alex
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

// priyasAccess sees her plan only and manages nothing for all markets, as a local planner.
func priyasAccess() *fakeAccess {
	return &fakeAccess{
		manageAll: refusal(refusedAll), manage: refusal(refusedAll),
		manageIn: map[string]error{planEurope: refusal(refusedAll), planChina: refusal(refusedAll), planBrazil: refusal(refusedAll)},
		hidden:   map[string]bool{planChina: true, planBrazil: true},
	}
}
