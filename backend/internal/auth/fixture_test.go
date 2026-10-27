package auth

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/people"
	"mapp/internal/people/peopletest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/record"
)

var (
	priya  = demoPerson("U-1002", "Priya Raman", "priya.raman@shell.com")
	daniel = demoPerson("U-1003", "Daniel Okoro", "daniel.okoro@shell.com")
	mei    = demoPerson("U-1004", "Mei Lin", "mei.lin@shell.com")
)

const demoPassword = "mapp"

var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// fixture is one test's service on a database of its own holding Priya, Daniel and Mei.
type fixture struct {
	service *Service
	db      *fake.DB
	clock   *testClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	clock := &testClock{now: start}
	f := &fixture{
		service: NewService(db, newFakeStore(db), peopletest.NewStore(db),
			Options{DemoMode: true, DemoPassword: demoPassword, Now: clock.Now}),
		db:    db,
		clock: clock,
	}
	f.putPeople(t, priya, daniel, mei)
	return f
}

func (f *fixture) signIn(t *testing.T, email, oldToken string) string {
	t.Helper()
	token, _, err := f.service.SignIn(t.Context(), email, demoPassword, oldToken)
	if err != nil {
		t.Fatalf("SignIn(%s): %v", email, err)
	}
	return token
}

func (f *fixture) authenticate(t *testing.T, token string) {
	t.Helper()
	if _, err := f.service.Authenticate(t.Context(), token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
}

func (f *fixture) deactivate(t *testing.T, p people.Person) {
	t.Helper()
	p.Active = false
	f.putPeople(t, p)
}

func (f *fixture) putPeople(t *testing.T, list ...people.Person) {
	t.Helper()
	repo := peopletest.NewStore(f.db)
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
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

func (f *fixture) putSession(t *testing.T, s Session) {
	t.Helper()
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		return newFakeStore(f.db).Put(ctx, s)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

func (f *fixture) sessions(t *testing.T) []Session {
	t.Helper()
	var list []Session
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		list, err = newFakeStore(f.db).List(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return list
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time { return c.now }

func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }

func demoPerson(id, name, email string) people.Person {
	return people.Person{ID: id, Name: name, Email: email, JobTitle: "Marketer", Active: true, Meta: record.NewMeta(actor.System, start)}
}

func sessionIDs(sessions []Session) []string {
	ids := make([]string, 0, len(sessions))
	for _, s := range sessions {
		ids = append(ids, s.ID)
	}
	return ids
}

// assertError checks the status, code, message and field problems at once.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}
