package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

func TestSignIn_EmailIgnoresCaseAndSpaces_FR003(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	_, person, err := f.service.SignIn(t.Context(), SignInAttempt{Email: "  PRIYA.Raman@SHELL.com ", Password: demoPassword})

	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if person.ID != priya.ID {
		t.Errorf("signed in as %s, want %s", person.ID, priya.ID)
	}
}

// Letter case and spaces count, and a refused sign-in starts no session.
func TestSignIn_WrongPassword_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)

	for _, password := range []string{"Mapp", "mapp ", " mapp", "map", "mapp!"} {
		t.Run(password, func(t *testing.T) {
			_, _, err := f.service.SignIn(t.Context(), SignInAttempt{Email: priya.Email, Password: password})

			assertError(t, err, errs.IncorrectCredentials())
		})
	}
	if sessions := f.sessions(t); len(sessions) != 0 {
		t.Errorf("sessions = %v, want none", sessions)
	}
}

// The page never tells which one was wrong or which e-mails exist.
func TestSignIn_SameErrorForUnknownWrongOrInactive_FR004(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.deactivate(t, mei)
	attempts := []struct {
		name, email, password string
	}{
		{"unknown e-mail", "nobody@shell.com", demoPassword},
		{"wrong password", priya.Email, "wrong"},
		{"inactive person", mei.Email, demoPassword},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			_, _, err := f.service.SignIn(t.Context(), SignInAttempt{Email: attempt.email, Password: attempt.password})

			assertError(t, err, errs.IncorrectCredentials())
		})
	}
}

func TestSignIn_EmptyFieldsAllReported_FR004(t *testing.T) {
	t.Parallel()
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
			_, _, err := f.service.SignIn(t.Context(), SignInAttempt{Email: tt.email, Password: tt.password})

			assertError(t, err, tt.want)
		})
	}
}

// Old sessions do not pile up.
func TestSignIn_ReplacesOldSessionAndPrunesExpired_FR006(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.signIn(t, daniel.Email, "")
	f.clock.Advance(time.Hour)
	oldPriya := f.signIn(t, priya.Email, "")
	f.clock.Advance(time.Hour)
	meis := f.signIn(t, mei.Email, "")
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

// A copy of the stored data cannot be used to sign in (research R6).
func TestSignIn_StoresOnlyTokenHash_FR006(t *testing.T) {
	t.Parallel()
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
	stored, err := json.Marshal(f.sessions(t))
	if err != nil {
		t.Fatalf("encoding the stored sessions: %v", err)
	}
	if bytes.Contains(stored, []byte(token)) {
		t.Error("the stored sessions contain the token itself")
	}
}
