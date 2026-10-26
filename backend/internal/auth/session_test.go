package auth

import (
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

func TestAuthenticate_ReturnsThePersonAsActor_FR007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")

	got, err := f.service.Authenticate(t.Context(), token)

	if err != nil || got != (actor.Actor{ID: "U-1002", Name: "Priya Raman"}) {
		t.Errorf("Authenticate = %+v, %v; want Priya Raman (U-1002)", got, err)
	}
}

func TestAuthenticate_WithoutTokenMustSignIn_FR005(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, err := f.service.Authenticate(t.Context(), "")

	assertError(t, err, errs.SignInRequired())
}

// Each request moves the 12-hour limit forward: 36 hours in all.
func TestAuthenticate_SlidingExpiry_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")

	for range 3 {
		f.clock.Advance(12 * time.Hour)

		if _, err := f.service.Authenticate(t.Context(), token); err != nil {
			t.Fatalf("Authenticate %v after signing in: %v", f.clock.now.Sub(start), err)
		}
	}
}

// Most requests save nothing; lastUsedAt is rewritten once it is a minute old.
func TestAuthenticate_TouchesAtMostOncePerMinute_FR006(t *testing.T) {
	t.Parallel()
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

// lastUsedAt can lag behind the last request by almost a minute (spec Assumptions).
func TestAuthenticate_NeverExpiresBefore12HoursIdle_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	token := f.signIn(t, priya.Email, "")
	f.clock.Advance(time.Minute - time.Millisecond)
	f.authenticate(t, token)

	f.clock.Advance(12 * time.Hour)

	if _, err := f.service.Authenticate(t.Context(), token); err != nil {
		t.Errorf("12 hours after the last request: %v, want the session to work", err)
	}
}

// Accepted exactly 12 h 1 min after lastUsedAt; refused after that, and deleted, so it never works again.
func TestAuthenticate_ExpiredIsDeleted_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	onTime := f.signIn(t, priya.Email, "")
	late := f.signIn(t, priya.Email, "")
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

// Authenticate only updates a session that is still stored.
func TestAuthenticate_DoesNotBringBackSignedOutSession_FR006(t *testing.T) {
	t.Parallel()
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

// Feature 003's Deactivate relies on it.
func TestAuthenticate_InactivePersonSessionEnds_FR007(t *testing.T) {
	t.Parallel()
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

func TestSessions_IndependentPerBrowser_EdgeCase(t *testing.T) {
	t.Parallel()
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
