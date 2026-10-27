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
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
)

// newTestService returns a people service on a new, empty data folder, and the store behind it.
func newTestService(t *testing.T) (*Service, *store.DB) {
	t.Helper()
	f := newFixture(t)
	return f.service, f.db
}

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture is the people service and the history service it logs to, on a new, empty data folder, with the
// people kinds registered as internal/app registers them, and a clock that moves only when the test says
// so. No hooks are set: tests that need plans or usage counts set fakes with UsePlans and AddPersonUsage.
type fixture struct {
	service *Service
	history *history.Service
	db      *store.DB
	ids     *ids.Generator
	clock   *testClock
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
	service := NewService(history.Tools{Tx: db, IDs: generator, Log: hist, Now: clock.Now}, NewJSONStore())
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return &fixture{service: service, history: hist, db: db, ids: generator, clock: clock}
}

// addPerson stores a person the way the seed does — an id from the counter U, made by MAPP itself,
// no log entry — with the e-mail "first.last@shell.com", and returns them.
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
		return NewJSONStore().Put(ctx, p)
	})
	if err != nil {
		t.Fatalf("addPerson(%q): %v", name, err)
	}
	return p
}

// addTeam stores a team with its members the way the seed does, and returns it.
func (f *fixture) addTeam(t *testing.T, name, colour string, active bool, members ...Person) Team { //nolint:revive // argument-limit 5: name, colour and status say which team; most callers add members
	t.Helper()
	var team Team
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		repo := NewJSONStore()
		id, err := f.ids.Next(ctx, ids.PrefixTeam)
		if err != nil {
			return err
		}
		team = Team{ID: id, Name: name, Colour: colour, Active: active, Meta: record.NewMeta(actor.System, start)}
		if err := repo.PutTeam(ctx, team); err != nil {
			return err
		}
		for _, p := range members {
			membershipID, err := f.ids.Next(ctx, ids.PrefixTeamMembership)
			if err != nil {
				return err
			}
			m := Membership{ID: membershipID, TeamID: team.ID, PersonID: p.ID, Meta: record.NewMeta(actor.System, start)}
			if err := repo.PutMembership(ctx, m); err != nil {
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

// read runs fn in a Read of the fixture's store with its repository, and stops the test when it fails.
func (f *fixture) read(t *testing.T, fn func(ctx context.Context, repo JSONStore) error) {
	t.Helper()
	err := f.db.Read(t.Context(), func(ctx context.Context) error { return fn(ctx, NewJSONStore()) })
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// storedPerson returns the stored person id and stops the test when there is none.
func (f *fixture) storedPerson(t *testing.T, id string) Person {
	t.Helper()
	var p Person
	var found bool
	f.read(t, func(ctx context.Context, repo JSONStore) (err error) {
		p, found, err = repo.Get(ctx, id)
		return err
	})
	if !found {
		t.Fatalf("person %s is not stored", id)
	}
	return p
}

// storedTeam returns the stored team id, and false when there is none.
func (f *fixture) storedTeam(t *testing.T, id string) (Team, bool) {
	t.Helper()
	var team Team
	var found bool
	f.read(t, func(ctx context.Context, repo JSONStore) (err error) {
		team, found, err = repo.GetTeam(ctx, id)
		return err
	})
	return team, found
}

// memberships returns every stored membership as "<team id>/<person id>", ordered by membership id.
func (f *fixture) memberships(t *testing.T) []string {
	t.Helper()
	var list []string
	f.read(t, func(ctx context.Context, repo JSONStore) error {
		stored, err := repo.ListMemberships(ctx)
		for _, m := range stored {
			list = append(list, m.TeamID+"/"+m.PersonID)
		}
		return err
	})
	return list
}

// entries returns the whole activity log, oldest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	var entries []history.Entry
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		entries, err = history.NewJSONStore().Entries(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return entries
}

// sentences writes each log entry as a sentence: "<person> <verb> <record type> '<name>' (<id>)", then
// its changes and details, the way feature 011 will show it.
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

// plansHook returns a PlanSource that lists plans.
func plansHook(plans ...PlanUse) PlanSource {
	return func(context.Context) ([]PlanUse, error) { return plans, nil }
}

// version returns a pointer to v, for the Version of updates.
func version(v int) *int {
	return &v
}

// text returns a pointer to s, for a text field an update sends (D-210).
func text(s string) *string {
	return &s
}

// idList returns a pointer to the list of ids, for the teams or members an update sends (D-210).
func idList(list ...string) *[]string {
	if list == nil {
		list = []string{}
	}
	return &list
}

// testClock is a clock that shows a fixed time until Advance moves it.
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

// addPeople saves people through the repository, in one Write.
func addPeople(t *testing.T, db *store.DB, list ...Person) {
	t.Helper()
	repo := NewJSONStore()
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

// storedPreferences returns the preferences record of a person, and false when there is none.
func storedPreferences(t *testing.T, db *store.DB, personID string) (Preferences, bool) {
	t.Helper()
	var prefs Preferences
	var found bool
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		prefs, found, err = NewJSONStore().GetPreferences(ctx, personID)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
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

// assertLog checks the whole activity log, written as sentences.
func assertLog(t *testing.T, f *fixture, want []string) {
	t.Helper()
	if got := sentences(f.entries(t)); !slices.Equal(got, want) {
		t.Errorf("log =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
