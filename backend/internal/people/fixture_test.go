package people

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

func newTestService(t *testing.T) (*Service, *fake.DB) {
	t.Helper()
	f := newFixture(t)
	return f.service, f.db
}

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture registers the people kinds as internal/app does, but sets no hooks: tests that need plans or usage
// counts set fakes with UsePlans and AddPersonUsage.
type fixture struct {
	service *Service
	history *history.Service
	db      *fake.DB
	store   fakeStore
	ids     *ids.Generator
	clock   *testClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	generator := ids.NewGenerator(fake.NewCounters(db))
	hist := history.NewService(db, historytest.NewStore(db), generator, clock.Now)
	repo := newFakeStore(db)
	service := NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, repo)
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return &fixture{service: service, history: hist, db: db, store: repo, ids: generator, clock: clock}
}

// addPerson stores a person as the seed does: an id from the counter U, made by MAPP itself, no log entry.
func (f *fixture) addPerson(t *testing.T, name, jobTitle string, active bool) Person {
	t.Helper()
	var p Person
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		id, err := f.ids.Next(ctx, ids.PrefixPerson)
		if err != nil {
			return err
		}
		p = newPerson(id, name, active)
		p.JobTitle = jobTitle
		return f.store.Put(ctx, p)
	})
	if err != nil {
		t.Fatalf("addPerson(%q): %v", name, err)
	}
	return p
}

func (f *fixture) addTeam(t *testing.T, name, colour string, active bool, members ...Person) Team { //nolint:revive // argument-limit 5: name, colour and status say which team; most callers add members
	t.Helper()
	var team Team
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		id, err := f.ids.Next(ctx, ids.PrefixTeam)
		if err != nil {
			return err
		}
		team = Team{ID: id, Name: name, Colour: colour, Active: active, Meta: record.NewMeta(actor.System, start)}
		if err = f.store.PutTeam(ctx, team); err != nil {
			return err
		}
		for _, p := range members {
			membershipID, err := f.ids.Next(ctx, ids.PrefixTeamMembership)
			if err != nil {
				return err
			}
			m := Membership{ID: membershipID, TeamID: team.ID, PersonID: p.ID, Meta: record.NewMeta(actor.System, start)}
			if err = f.store.PutMembership(ctx, m); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("addTeam(%q): %v", name, err)
	}
	return team
}

func (f *fixture) read(t *testing.T, fn func(ctx context.Context, repo fakeStore) error) {
	t.Helper()
	err := f.db.Read(t.Context(), func(ctx context.Context) error { return fn(ctx, f.store) })
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

func (f *fixture) storedPerson(t *testing.T, id string) Person {
	t.Helper()
	var p Person
	var found bool
	f.read(t, func(ctx context.Context, repo fakeStore) (err error) {
		p, found, err = repo.Get(ctx, id)
		return err
	})
	if !found {
		t.Fatalf("person %s is not stored", id)
	}
	return p
}

func (f *fixture) storedTeam(t *testing.T, id string) (Team, bool) {
	t.Helper()
	var team Team
	var found bool
	f.read(t, func(ctx context.Context, repo fakeStore) (err error) {
		team, found, err = repo.GetTeam(ctx, id)
		return err
	})
	return team, found
}

// memberships returns every stored membership as "<team id>/<person id>", ordered by membership id.
func (f *fixture) memberships(t *testing.T) []string {
	t.Helper()
	var list []string
	f.read(t, func(ctx context.Context, repo fakeStore) error {
		stored, err := repo.ListMemberships(ctx)
		for _, m := range stored {
			list = append(list, m.TeamID+"/"+m.PersonID)
		}
		return err
	})
	return list
}

func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	entries, err := historytest.NewStore(f.db).Entries(t.Context())
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	return entries
}

// sentences writes each log entry as "<person> <verb> <record type> '<name>' (<id>)", then its changes and
// details.
func sentences(entries []history.Entry) []string {
	list := make([]string, 0, len(entries))
	for _, e := range entries {
		text := fmt.Sprintf("%s %s %s '%s' (%s)", e.Actor.Name, e.Verb, e.RecordTypeLabel, e.RecordName, e.RecordID)
		for _, c := range e.Changes {
			text += fmt.Sprintf(": %s %s → %s", c.Label, c.OldText, c.NewText)
		}
		if e.Details != "" {
			text += " — " + e.Details
		}
		list = append(list, text)
	}
	return list
}

func plansHook(plans ...PlanUse) PlanSource {
	return func(context.Context) ([]PlanUse, error) { return plans, nil }
}

func version(v int) *int {
	return &v
}

func text(s string) *string {
	return &s
}

func idList(list ...string) *[]string {
	if list == nil {
		list = []string{}
	}
	return &list
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }

func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

// newPerson returns a person with the e-mail "first.last@shell.com" and the job title Marketer.
func newPerson(id, name string, active bool) Person {
	email := strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@shell.com"
	created := time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
	return Person{ID: id, Name: name, Email: email, JobTitle: "Marketer", Active: active, Meta: record.NewMeta(actor.System, created)}
}

func addPeople(t *testing.T, db *fake.DB, list ...Person) {
	t.Helper()
	repo := newFakeStore(db)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for _, p := range list {
			if err := repo.Put(ctx, p); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func storedPreferences(t *testing.T, db *fake.DB, personID string) (Preferences, bool) {
	t.Helper()
	prefs, found, err := newFakeStore(db).GetPreferences(t.Context(), personID)
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	return prefs, found
}

// assertError checks that err is exactly want: the same status, code, message and field problems.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}

func assertLog(t *testing.T, f *fixture, want []string) {
	t.Helper()
	if got := sentences(f.entries(t)); !slices.Equal(got, want) {
		t.Errorf("log =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
