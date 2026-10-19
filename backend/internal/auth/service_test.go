package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// The demo people of these tests, as the seed creates them.
var (
	priya  = demoPerson("U-1002", "Priya Raman", "priya.raman@northstar.demo")
	daniel = demoPerson("U-1003", "Daniel Okoro", "daniel.okoro@northstar.demo")
	mei    = demoPerson("U-1004", "Mei Lin", "mei.lin@northstar.demo")
)

const demoPassword = "northstar"

// start is the time on the clock when every test begins: 12 Mar 2026 09:00 UTC.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The e-mail is found whatever its letter case and the spaces around it (FR-003).
func TestSignIn_EmailIgnoresCaseAndSpaces_FR003(t *testing.T) {
	f := newFixture(t)

	_, person, err := f.service.SignIn(t.Context(), "  PRIYA.Raman@NorthStar.demo ", demoPassword, "")

	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if person.ID != priya.ID {
		t.Errorf("signed in as %s, want %s", person.ID, priya.ID)
	}
}

// The password must match exactly: letter case and spaces count (FR-004). A refused sign-in starts no
// session.
func TestSignIn_WrongPassword_FR004(t *testing.T) {
	f := newFixture(t)

	for _, password := range []string{"Northstar", "northstar ", " northstar", "north", "northstar!"} {
		t.Run(password, func(t *testing.T) {
			_, _, err := f.service.SignIn(t.Context(), priya.Email, password, "")

			assertError(t, err, errs.IncorrectCredentials())
		})
	}
	if sessions := f.sessions(t); len(sessions) != 0 {
		t.Errorf("sessions = %v, want none", sessions)
	}
}

// An unknown e-mail, a wrong password and an inactive person get the very same answer, so the page never
// tells which one was wrong or which e-mails exist (FR-004).
func TestSignIn_SameErrorForUnknownWrongOrInactive_FR004(t *testing.T) {
	f := newFixture(t)
	f.deactivate(t, mei)
	attempts := []struct {
		name, email, password string
	}{
		{"unknown e-mail", "nobody@northstar.demo", demoPassword},
		{"wrong password", priya.Email, "wrong"},
		{"inactive person", mei.Email, demoPassword},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			_, _, err := f.service.SignIn(t.Context(), attempt.email, attempt.password, "")

			assertError(t, err, errs.IncorrectCredentials())
		})
	}
}

// Every empty field is reported in one answer (FR-004, FR-020).
func TestSignIn_EmptyFieldsAllReported_FR004(t *testing.T) {
	f := newFixture(t)
	emptyEmail := errs.FieldError{Field: "email", Message: "Enter your e-mail"}
	emptyPassword := errs.FieldError{Field: "password", Message: "Enter the password"}
	tests := []struct {
		name            string
		email, password string
		want            *errs.Error
	}{
		{"both empty", "", "", errs.Validation(emptyEmail, emptyPassword)},
		{"e-mail of spaces only", "   ", demoPassword, errs.Validation(emptyEmail)},
		{"password empty", priya.Email, "", errs.Validation(emptyPassword)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := f.service.SignIn(t.Context(), tt.email, tt.password, "")

			assertError(t, err, tt.want)
		})
	}
}

// Signing in ends the session this browser had before, and every expired session of anyone, so old
// sessions do not pile up (FR-006).
func TestSignIn_ReplacesOldSessionAndPrunesExpired_FR006(t *testing.T) {
	f := newFixture(t)
	f.signIn(t, daniel.Email, "") // 09:00
	f.clock.Advance(time.Hour)
	oldPriya := f.signIn(t, priya.Email, "") // 10:00
	f.clock.Advance(time.Hour)
	meis := f.signIn(t, mei.Email, "") // 11:00
	// 21:01:00.001: Daniel's session has had no request for more than 12 h 1 min; the others have.
	f.clock.Advance(10*time.Hour + time.Minute + time.Millisecond)

	newPriya := f.signIn(t, priya.Email, oldPriya)

	got := sessionIDs(f.sessions(t))
	want := []string{hashToken(meis), hashToken(newPriya)}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("stored sessions = %q, want only Mei's and Priya's new one %q", got, want)
	}
}

// The token is 32 random bytes, and only its SHA-256 hash is stored, so a copied data file cannot be used
// to sign in (FR-006, research R6).
func TestSignIn_StoresOnlyTokenHash_FR006(t *testing.T) {
	f := newFixture(t)

	token := f.signIn(t, priya.Email, "")

	if raw, err := base64.RawURLEncoding.DecodeString(token); err != nil || len(raw) != 32 {
		t.Errorf("token %q = %d bytes (%v), want 32 bytes written as base64url", token, len(raw), err)
	}
	hash := sha256.Sum256([]byte(token))
	want := Session{ID: hex.EncodeToString(hash[:]), PersonID: priya.ID, CreatedAt: start, LastUsedAt: start}
	if got := f.sessions(t); len(got) != 1 || got[0] != want {
		t.Errorf("stored sessions = %+v, want only %+v", got, want)
	}
	file, err := os.ReadFile(filepath.Join(f.dir, "sessions.json"))
	if err != nil {
		t.Fatalf("reading sessions.json: %v", err)
	}
	if bytes.Contains(file, []byte(token)) {
		t.Error("sessions.json contains the token itself")
	}
}

// Every request carries the signed-in person as the actor (FR-007).
func TestAuthenticate_ReturnsThePersonAsActor_FR007(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")

	got, err := f.service.Authenticate(t.Context(), token)

	if err != nil || got != (actor.Actor{ID: "U-1002", Name: "Priya Raman"}) {
		t.Errorf("Authenticate = %+v, %v; want Priya Raman (U-1002)", got, err)
	}
}

// A browser without a session cookie is not signed in (FR-005).
func TestAuthenticate_WithoutTokenMustSignIn_FR005(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Authenticate(t.Context(), "")

	assertError(t, err, errs.SignInRequired())
}

// A session in use stays alive: each request moves its 12-hour limit forward (FR-006).
func TestAuthenticate_SlidingExpiry_FR006(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")

	for range 3 { // 36 hours in all, far beyond 12 hours after signing in
		f.clock.Advance(12 * time.Hour)

		if _, err := f.service.Authenticate(t.Context(), token); err != nil {
			t.Fatalf("Authenticate %v after signing in: %v", f.clock.now.Sub(start), err)
		}
	}
}

// lastUsedAt is rewritten only when it is at least a minute old, so most requests save nothing; at
// exactly one minute it is rewritten (FR-006).
func TestAuthenticate_TouchesAtMostOncePerMinute_FR006(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")

	f.clock.Advance(time.Minute - time.Millisecond)
	f.authenticate(t, token)
	if got := f.sessions(t)[0].LastUsedAt; !got.Equal(start) {
		t.Errorf("59.999 s after signing in: lastUsedAt = %v, want it unchanged at %v", got, start)
	}

	f.clock.Advance(time.Millisecond)
	f.authenticate(t, token)
	if got, want := f.sessions(t)[0].LastUsedAt, start.Add(time.Minute); !got.Equal(want) {
		t.Errorf("1 min after signing in: lastUsedAt = %v, want %v", got, want)
	}
}

// No session ends before 12 full hours without a request, although lastUsedAt can lag behind the last
// request by almost a minute (FR-006, spec Assumptions).
func TestAuthenticate_NeverExpiresBefore12HoursIdle_FR006(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")
	f.clock.Advance(time.Minute - time.Millisecond)
	f.authenticate(t, token) // the last request: too early to rewrite lastUsedAt

	f.clock.Advance(12 * time.Hour) // 12 full hours after the last request

	if _, err := f.service.Authenticate(t.Context(), token); err != nil {
		t.Errorf("12 hours after the last request: %v, want the session to work", err)
	}
}

// A session is accepted exactly 12 h 1 min after lastUsedAt and refused with session_expired after that.
// The refused session is deleted, so it never works again (FR-006).
func TestAuthenticate_ExpiredIsDeleted_FR006(t *testing.T) {
	f := newFixture(t)
	onTime := f.signIn(t, priya.Email, "")
	late := f.signIn(t, priya.Email, "") // the same person in a second browser
	f.clock.Advance(12*time.Hour + time.Minute)

	if _, err := f.service.Authenticate(t.Context(), onTime); err != nil {
		t.Errorf("exactly 12 h 1 min after the last request: %v, want the session to work", err)
	}
	f.clock.Advance(time.Millisecond)
	_, err := f.service.Authenticate(t.Context(), late)
	assertError(t, err, errs.SessionExpired())

	if slices.Contains(sessionIDs(f.sessions(t)), hashToken(late)) {
		t.Error("the expired session is still stored")
	}
	_, err = f.service.Authenticate(t.Context(), late)
	assertError(t, err, errs.SignInRequired())
}

// Authenticate never brings back a session that was signed out: it only updates a session that is still
// stored (FR-006).
func TestAuthenticate_DoesNotBringBackSignedOutSession_FR006(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")
	f.clock.Advance(2 * time.Minute) // old enough for lastUsedAt to be rewritten
	if err := f.service.SignOut(t.Context(), token); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	_, err := f.service.Authenticate(t.Context(), token)

	assertError(t, err, errs.SignInRequired())
	if sessions := f.sessions(t); len(sessions) != 0 {
		t.Errorf("sessions = %+v, want none", sessions)
	}
}

// When a person is deactivated, or no longer exists, their open sessions end at their next request
// (FR-007; feature 003 relies on it).
func TestAuthenticate_InactivePersonSessionEnds_FR007(t *testing.T) {
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")
	f.deactivate(t, priya)

	_, err := f.service.Authenticate(t.Context(), token)

	assertError(t, err, errs.SignInRequired())
	if sessions := f.sessions(t); len(sessions) != 0 {
		t.Errorf("after deactivating: sessions = %+v, want none", sessions)
	}

	gone := "the token of a person who no longer exists"
	f.putSession(t, Session{ID: hashToken(gone), PersonID: "U-9999", CreatedAt: start, LastUsedAt: start})

	_, err = f.service.Authenticate(t.Context(), gone)

	assertError(t, err, errs.SignInRequired())
	if sessions := f.sessions(t); len(sessions) != 0 {
		t.Errorf("after the person was gone: sessions = %+v, want none", sessions)
	}
}

// The same person can be signed in in two browsers; each session works and ends on its own (spec edge
// case).
func TestSessions_IndependentPerBrowser_EdgeCase(t *testing.T) {
	f := newFixture(t)
	first := f.signIn(t, priya.Email, "")
	second := f.signIn(t, priya.Email, "")
	if first == second {
		t.Fatal("both sign-ins got the same token")
	}

	if err := f.service.SignOut(t.Context(), first); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	_, err := f.service.Authenticate(t.Context(), first)
	assertError(t, err, errs.SignInRequired())
	if got, err := f.service.Authenticate(t.Context(), second); err != nil || got != priya.Actor() {
		t.Errorf("the other browser: Authenticate = %+v, %v; want Priya", got, err)
	}
}

// fixture is one test's auth service on a temporary data folder holding Priya, Daniel and Mei, with a clock
// that moves only when the test says so.
type fixture struct {
	service *Service
	db      *store.DB
	dir     string
	clock   *testClock
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	f := &fixture{
		service: NewService(db, NewJSONRepository(), people.NewJSONRepository(), true, demoPassword, clock.Now),
		db:      db,
		dir:     dir,
		clock:   clock,
	}
	f.putPeople(t, priya, daniel, mei)
	return f
}

// signIn signs in with the demo password and returns the new session's token. It stops the test when the
// sign-in fails.
func (f *fixture) signIn(t *testing.T, email, oldToken string) string {
	t.Helper()
	token, _, err := f.service.SignIn(t.Context(), email, demoPassword, oldToken)
	if err != nil {
		t.Fatalf("SignIn(%s): %v", email, err)
	}
	return token
}

// authenticate authenticates token and stops the test when that fails.
func (f *fixture) authenticate(t *testing.T, token string) {
	t.Helper()
	if _, err := f.service.Authenticate(t.Context(), token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
}

// deactivate saves p as inactive, as feature 003's Deactivate will.
func (f *fixture) deactivate(t *testing.T, p people.Person) {
	t.Helper()
	p.Active = false
	f.putPeople(t, p)
}

func (f *fixture) putPeople(t *testing.T, list ...people.Person) {
	t.Helper()
	repo := people.NewJSONRepository()
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
		return NewJSONRepository().Put(ctx, s)
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
}

// sessions returns every stored session.
func (f *fixture) sessions(t *testing.T) []Session {
	t.Helper()
	var list []Session
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		list, err = NewJSONRepository().List(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return list
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
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

// assertError checks that err is exactly want: the same status, code, message and field problems.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v, want %#v", err, want)
	}
}
