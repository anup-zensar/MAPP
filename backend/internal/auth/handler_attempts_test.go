package auth_test

import (
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
)

const tooManyBody = `{"error":{"code":"too_many_attempts","message":"Too many sign-in attempts. Try again in 15 minutes.","fields":[]}}`

func TestSignIn_SixthAttemptIsRefused_038_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	for range 5 {
		assertAnswer(t, server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, "wrong")),
			http.StatusUnauthorized, incorrectBody)
	}

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(" PRIYA.Raman@shell.com", demoPassword))

	assertAnswer(t, response, http.StatusTooManyRequests, tooManyBody)
	if got := response.Header.Get("Retry-After"); got != "900" {
		t.Errorf("Retry-After = %q, want 900", got)
	}
	if cookie := response.Cookie("mapp_session"); cookie != nil {
		t.Errorf("a session cookie was set: %+v", cookie)
	}
}

func TestSignIn_SignsInAgainAfterTheLock_038_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	for range 5 {
		server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, "wrong"))
	}
	server.Advance(14 * time.Minute)
	locked := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, demoPassword))
	if got := locked.Header.Get("Retry-After"); locked.Status != http.StatusTooManyRequests || got != "60" {
		t.Errorf("after 14 minutes: %d with Retry-After %q, want 429 with 60", locked.Status, got)
	}

	server.Advance(time.Minute)

	assertAnswer(t, server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, demoPassword)),
		http.StatusOK, priyaBody)
}

func TestSignIn_NULInTheEmailIsAFieldProblem_038_FR017_M3(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn("priya\x00@shell.com", demoPassword))

	assertAnswer(t, response, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"email","message":"Remove the control characters from this text."}]}}`)
}
