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
	"mapp/internal/platform/fake"
	"mapp/internal/platform/money"
)

var (
	alex   = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = actor.Actor{ID: "U-1004", Name: "Mei Lin"}
	sara   = actor.Actor{ID: "U-1006", Name: "Sara Novak"}
	nina   = actor.Actor{ID: "U-1007", Name: "Nina Berg"} // in no team
)

var now = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

var (
	lubricantsEurope   = Team{ID: "T-1001", Name: "Lubricants Europe", Colour: "blue", Active: true}
	mobilityChina      = Team{ID: "T-1002", Name: "Mobility China", Colour: "teal", Active: true}
	globalMarketingOps = Team{ID: "T-1003", Name: "Global Marketing Ops", Colour: "violet", Active: true}
)

// fakeSources keep containers, activities and waiting lists per person: the real sources filter by owner and approver.
type fakeSources struct {
	teams      map[string][]Team
	plans      []Plan
	containers map[string][]Container
	activities map[string][]Activity
	waiting    map[string][]Waiting // oldest first
	currencies map[string]Currency  // none means US dollars
	failWith   error                // when set, every source fails with it

	logAsked  [][]string // nil plan ids ask for every plan
	logLimit  int
	logAnswer RecentChangesPage
}

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

func newService(t testing.TB, f *fakeSources) *Service {
	t.Helper()
	return NewService(fake.NewDB(), f.sources())
}

func newPlan(id, name string, year int, teams ...Team) Plan {
	return Plan{ID: id, Name: name, Year: year, Status: "draft", Teams: teams, Missing: []MissingField{}, UpdatedAt: now}
}

func usd(t testing.TB, text string) money.Amount {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return amount
}

func usdPtr(t testing.TB, text string) *money.Amount {
	t.Helper()
	amount := usd(t, text)
	return &amount
}

func ptr[T any](v T) *T {
	return &v
}

func rate(t testing.TB, text string) money.Rate {
	t.Helper()
	r, err := money.ParseRate(text)
	if err != nil {
		t.Fatalf("ParseRate(%q): %v", text, err)
	}
	return r
}

func planIDs(plans []Plan) []string {
	list := []string{}
	for _, p := range plans {
		list = append(list, p.ID)
	}
	return list
}

func assertError(t testing.TB, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

var scopeRefused = errs.Validation(errs.FieldError{Field: "scope", Message: "Choose My teams or Everything."})
