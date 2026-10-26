package home

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// The service tests run on fake Sources (009 plan "Tests"), with the demo people and teams.

// The people of these tests.
var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	sara   = actor.Actor{ID: "U-1006", Name: "Sara Novak"}
	nina   = actor.Actor{ID: "U-1007", Name: "Nina Berg"} // in no team
)

// now is the time of the tests: 12 Mar 2026 09:00 UTC.
var now = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The teams of these tests.
var (
	lubricantsEurope   = Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true}      // Priya, Daniel, Sara
	mobilityChina      = Team{ID: "T-1002", Name: "Mobility China", Colour: "teal", Active: true}         // Mei, Sara
	globalMarketingOps = Team{ID: "T-1003", Name: "Global Marketing Ops", Colour: "violet", Active: true} // Alex
)

// fakeSources are Sources as the tests set them. Containers, activities and waiting lists are kept per
// person: the real sources filter by owner and approver.
type fakeSources struct {
	teams      map[string][]Team      // person id → their teams
	plans      []Plan                 // every plan
	containers map[string][]Container // owner id → their campaigns and initiatives
	activities map[string][]Activity  // owner id → their activities
	waiting    map[string][]Waiting   // approver id → the Waiting for me tab, oldest first
	currencies map[string]Currency    // person id → display currency; none means US dollars
	failWith   error                  // when set, every source fails with it

	logAsked  [][]string // the plan ids of every RecentChanges call; nil for every plan
	logLimit  int        // the limit of the last RecentChanges call
	logAnswer RecentChangesPage
}

// newFakeSources returns the demo teams and nothing else.
func newFakeSources() *fakeSources {
	return &fakeSources{
		teams: map[string][]Team{
			priya.ID: {lubricantsEurope}, daniel.ID: {lubricantsEurope}, sara.ID: {lubricantsEurope, mobilityChina},
			mei.ID: {mobilityChina}, alex.ID: {globalMarketingOps},
		},
		containers: map[string][]Container{},
		activities: map[string][]Activity{},
		waiting:    map[string][]Waiting{},
		currencies: map[string]Currency{},
	}
}

// sources returns the fake as Sources.
func (f *fakeSources) sources() Sources {
	return Sources{
		MyTeams: func(_ context.Context, a actor.Actor) ([]Team, error) {
			return slices.Clone(f.teams[a.ID]), f.failWith
		},
		Plans: func(context.Context, actor.Actor) ([]Plan, error) {
			return slices.Clone(f.plans), f.failWith
		},
		Containers: func(_ context.Context, _ actor.Actor, ownerID string) ([]Container, error) {
			return slices.Clone(f.containers[ownerID]), f.failWith
		},
		Activities: func(_ context.Context, _ actor.Actor, ownerID string) ([]Activity, error) {
			return slices.Clone(f.activities[ownerID]), f.failWith
		},
		WaitingForMe: func(_ context.Context, a actor.Actor) ([]Waiting, error) {
			return slices.Clone(f.waiting[a.ID]), f.failWith
		},
		DisplayCurrency: func(_ context.Context, a actor.Actor) (Currency, error) {
			if c, found := f.currencies[a.ID]; found {
				return c, f.failWith
			}
			return Currency{Code: "USD", Rate: money.One}, f.failWith
		},
		RecentChanges: func(_ context.Context, _ actor.Actor, planIDs []string, limit int) (RecentChangesPage, error) {
			f.logAsked = append(f.logAsked, slices.Clone(planIDs))
			f.logLimit = limit
			return f.logAnswer, f.failWith
		},
	}
}

// newService returns the Home service on the fake sources, with a temporary store for its transactions.
func newService(t testing.TB, f *fakeSources) *Service {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return NewService(db, f.sources())
}

// newPlan returns a plan of the year with the teams, Draft, without budget and without anything below.
func newPlan(id, name string, year int, teams ...Team) Plan {
	return Plan{ID: id, Name: name, Year: year, Status: "draft", Teams: teams, Missing: []MissingField{}, UpdatedAt: now}
}

// usd returns the US-dollar amount of text, such as "1200000.00".
func usd(t testing.TB, text string) money.Amount {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return amount
}

// usdPtr is usd as a pointer.
func usdPtr(t testing.TB, text string) *money.Amount {
	t.Helper()
	amount := usd(t, text)
	return &amount
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T {
	return &v
}

// rate returns the exchange rate of text, such as "0.92".
func rate(t testing.TB, text string) money.Rate {
	t.Helper()
	r, err := money.ParseRate(text)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", text, err)
	}
	return r
}

// planIDs returns the ids of plans, in order.
func planIDs(plans []Plan) []string {
	list := []string{}
	for _, p := range plans {
		list = append(list, p.ID)
	}
	return list
}

// assertError checks that err is exactly want.
func assertError(t testing.TB, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// scopeRefused is the answer to a scope other than myTeams or everything.
var scopeRefused = errs.Validation(errs.FieldError{Field: "scope", Message: "Choose My teams or Everything."})
