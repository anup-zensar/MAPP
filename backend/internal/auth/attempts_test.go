package auth

import (
	"bytes"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/errs"
)

const (
	client      = "192.0.2.10"
	otherClient = "192.0.2.20"
)

func newTestAttempts() (*attempts, *testClock) {
	clock := &testClock{now: start}
	return newAttempts(clock.Now), clock
}

func failTimes(a *attempts, n int, email string) {
	for range n {
		a.failed(email, client)
	}
}

func TestAttempts_038_FR014_FiveFailuresOfOneEmailLockIt(t *testing.T) {
	t.Parallel()
	a, clock := newTestAttempts()
	failTimes(a, 4, priya.Email)
	if wait := a.wait(priya.Email, client); wait != 0 {
		t.Fatalf("after 4 failures wait = %v, want none", wait)
	}
	clock.Advance(time.Minute)

	locked := a.failed(priya.Email, otherClient)

	if !slices.Equal(locked, []string{keyEmail}) {
		t.Errorf("the 5th failure locked %v, want [%s]", locked, keyEmail)
	}
	if wait := a.wait(priya.Email, "192.0.2.99"); wait != lockFor {
		t.Errorf("wait = %v, want %v from the 5th failure", wait, lockFor)
	}
	if wait := a.wait(daniel.Email, client); wait != 0 {
		t.Errorf("another e-mail waits %v, want none", wait)
	}
}

func TestAttempts_038_FR014_EmailIgnoresCaseAndSpaces(t *testing.T) {
	t.Parallel()
	a, _ := newTestAttempts()
	for i, email := range []string{"PRIYA.raman@shell.com", " priya.raman@shell.com", "Priya.Raman@Shell.Com ", priya.Email, "priya.RAMAN@shell.com"} {
		a.failed(email, fmt.Sprintf("192.0.2.%d", i))
	}

	if wait := a.wait("  priya.raman@SHELL.COM", otherClient); wait != lockFor {
		t.Errorf("wait = %v, want %v", wait, lockFor)
	}
}

func TestAttempts_038_FR014_UnknownEmailsCountAlike(t *testing.T) {
	t.Parallel()
	a, _ := newTestAttempts()
	failTimes(a, 5, "nobody@shell.com")

	if wait := a.wait("nobody@shell.com", otherClient); wait != lockFor {
		t.Errorf("wait = %v, want %v", wait, lockFor)
	}
}

func TestAttempts_038_FR014_TwentyFailuresOfOneClientLockIt(t *testing.T) {
	t.Parallel()
	a, _ := newTestAttempts()
	for i := range 19 {
		a.failed(fmt.Sprintf("person%d@shell.com", i), client)
	}
	if wait := a.wait("someone@shell.com", client); wait != 0 {
		t.Fatalf("after 19 failures wait = %v, want none", wait)
	}

	locked := a.failed("person19@shell.com", client)

	if !slices.Equal(locked, []string{keyClient}) {
		t.Errorf("the 20th failure locked %v, want [%s]", locked, keyClient)
	}
	if wait := a.wait(priya.Email, client); wait != lockFor {
		t.Errorf("the client waits %v, want %v", wait, lockFor)
	}
	if wait := a.wait(priya.Email, otherClient); wait != 0 {
		t.Errorf("another client waits %v, want none", wait)
	}
}

func TestAttempts_038_FR014_OnlyFailuresWithinTheWindowCount(t *testing.T) {
	t.Parallel()
	a, clock := newTestAttempts()
	failTimes(a, 4, priya.Email)
	clock.Advance(window)

	a.failed(priya.Email, client)

	if wait := a.wait(priya.Email, client); wait != 0 {
		t.Errorf("wait = %v, want none: the first 4 failures are 15 minutes old", wait)
	}
}

func TestAttempts_038_FR014_TheLockEndsAfterFifteenMinutesAndIsNotExtended(t *testing.T) {
	t.Parallel()
	a, clock := newTestAttempts()
	failTimes(a, 5, priya.Email)
	clock.Advance(10*time.Minute + 30*time.Second)

	if wait := a.wait(priya.Email, client); wait != 4*time.Minute+30*time.Second {
		t.Errorf("wait = %v, want 4m30s: refused attempts do not extend the lock", wait)
	}
	clock.Advance(4*time.Minute + 30*time.Second)
	if wait := a.wait(priya.Email, client); wait != 0 {
		t.Errorf("after 15 minutes wait = %v, want none", wait)
	}
	a.failed(priya.Email, client)
	if wait := a.wait(priya.Email, client); wait != 0 {
		t.Errorf("one failure after the lock waits %v, want none: the count starts again", wait)
	}
}

func TestAttempts_038_FR014_SuccessClearsTheEmailAndTheClient(t *testing.T) {
	t.Parallel()
	a, _ := newTestAttempts()
	failTimes(a, 4, priya.Email)
	for i := range 15 {
		a.failed(fmt.Sprintf("person%d@shell.com", i), client)
	}

	a.succeeded(" Priya.Raman@shell.com", client)

	failTimes(a, 4, priya.Email)
	if wait := a.wait(priya.Email, client); wait != 0 {
		t.Errorf("wait = %v, want none: the success cleared both counts", wait)
	}
}

func TestAttempts_038_FR014_ForgetsKeysWithNothingToRemember(t *testing.T) {
	t.Parallel()
	a, clock := newTestAttempts()
	failTimes(a, 5, priya.Email)
	clock.Advance(lockFor)

	a.failed(daniel.Email, otherClient)

	if len(a.keys) != 2 {
		t.Errorf("remembered keys = %d, want 2 (Daniel's e-mail and client)", len(a.keys))
	}
}

func TestSignIn_038_FR014_LockedEvenWithTheRightPassword(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	wrong := SignInAttempt{Email: priya.Email, Password: "wrong", Client: client}
	for range emailLimit {
		_, _, err := f.service.SignIn(t.Context(), wrong)
		assertError(t, err, errs.IncorrectCredentials())
	}
	f.clock.Advance(time.Minute)

	_, _, err := f.service.SignIn(t.Context(), SignInAttempt{Email: priya.Email, Password: demoPassword, Client: otherClient})

	assertError(t, err, errs.TooManyAttempts("Too many sign-in attempts. Try again in 14 minutes.", 14*time.Minute))
}

func TestSignIn_038_FR014_SuccessClearsTheFailures(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for range emailLimit - 1 {
		_, _, _ = f.service.SignIn(t.Context(), SignInAttempt{Email: priya.Email, Password: "wrong", Client: client})
	}
	f.signIn(t, priya.Email, "")

	_, _, err := f.service.SignIn(t.Context(), SignInAttempt{Email: priya.Email, Password: "wrong", Client: client})

	assertError(t, err, errs.IncorrectCredentials())
	if wait := f.service.attempts.wait(priya.Email, client); wait != 0 {
		t.Errorf("wait = %v, want none after a success", wait)
	}
}

func TestSignIn_038_FR014_OneWarningPerLockWithoutTheEmail(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&out, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	f := newFixture(t)

	for range emailLimit + 2 {
		_, _, _ = f.service.SignIn(t.Context(), SignInAttempt{Email: priya.Email, Password: "wrong", Client: client})
	}

	if lines := strings.Count(out.String(), "\n"); lines != 1 || !strings.Contains(out.String(), `"key":"e-mail"`) {
		t.Errorf("log =\n%s\nwant one warning naming the key kind", out.String())
	}
	if strings.Contains(out.String(), priya.Email) || strings.Contains(out.String(), client) {
		t.Errorf("log %s holds the e-mail or the client", out.String())
	}
}

func TestRetryMessage_038_FR014_RoundsTheWaitUp(t *testing.T) {
	t.Parallel()
	tests := map[time.Duration]string{
		lockFor:          "Too many sign-in attempts. Try again in 15 minutes.",
		90 * time.Second: "Too many sign-in attempts. Try again in 2 minutes.",
		time.Second:      "Too many sign-in attempts. Try again in 1 minute.",
	}
	for wait, want := range tests {
		if got := retryMessage(wait); got != want {
			t.Errorf("retryMessage(%v) = %q, want %q", wait, got, want)
		}
	}
}
