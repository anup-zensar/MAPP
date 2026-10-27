package people_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/auth"
	"mapp/internal/platform/apitest"
)

// A new person is offered on the sign-in page and can sign in at once with the demo password (FR-003).
func TestNewPersonCanSignIn_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	createPerson(t, server.SignIn(t, alexsEmail), map[string]any{"name": "Jane Doe", "email": janesEmail})

	offered := slices.Contains(demoEmails(t, server), janesEmail)
	jane := server.SignIn(t, janesEmail)
	me := jane.Do(t, http.MethodGet, "/api/v1/me", nil)

	if !offered {
		t.Errorf("the sign-in page does not offer %s", janesEmail)
	}
	if me.Status != http.StatusOK || !strings.Contains(string(me.Body), `"name":"Jane Doe"`) {
		t.Errorf("GET /api/v1/me as the new person = %d %s", me.Status, me.Body)
	}
}

// A deactivated person is no longer offered on the sign-in page, cannot sign in, and their open session ends
// at their next request (FR-005).
func TestDeactivatedPersonCannotSignInAndSessionEnds_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail})
	janesSession := server.SignIn(t, janesEmail)

	setStatus(t, alex, jane.ID, false, 1)
	next := janesSession.Do(t, http.MethodGet, "/api/v1/people", nil)
	signIn := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", map[string]string{"email": janesEmail, "password": demoPassword(t, server)})

	assertAnswer(t, next, http.StatusUnauthorized, signInRequiredBody)
	assertAnswer(t, signIn, http.StatusUnauthorized, incorrectSignInBody)
	if slices.Contains(demoEmails(t, server), janesEmail) {
		t.Errorf("the sign-in page still offers %s", janesEmail)
	}
}

// A reactivated person can sign in again, with their teams as before (FR-007, SC-004).
func TestReactivatedPersonCanSignIn_FR007_SC004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	jane := createPerson(t, alex, map[string]any{"name": "Jane Doe", "email": janesEmail, "teamIds": []string{"T-1001"}})
	setStatus(t, alex, jane.ID, false, 1)

	back := setStatus(t, alex, jane.ID, true, 2)
	me := server.SignIn(t, janesEmail).Do(t, http.MethodGet, "/api/v1/me", nil)

	if !back.Active || len(back.Teams) != 1 || back.Teams[0].ID != "T-1001" {
		t.Errorf("reactivated person = %+v, want active in Lubricants Europe", back)
	}
	if me.Status != http.StatusOK {
		t.Errorf("GET /api/v1/me after reactivation = %d %s", me.Status, me.Body)
	}
}

// demoSignIn returns what the sign-in page shows: the people it offers and the demo password.
func demoSignIn(t *testing.T, server *apitest.Server) auth.DemoPeople {
	t.Helper()
	var demo auth.DemoPeople
	server.Do(t, http.MethodGet, "/api/v1/auth/demo-people", nil).Decode(t, &demo)
	return demo
}

// demoEmails returns the e-mails of the people the sign-in page offers.
func demoEmails(t *testing.T, server *apitest.Server) []string {
	t.Helper()
	var emails []string
	for _, p := range demoSignIn(t, server).People {
		emails = append(emails, p.Email)
	}
	return emails
}

// demoPassword returns the demo password the sign-in page shows.
func demoPassword(t *testing.T, server *apitest.Server) string {
	t.Helper()
	return demoSignIn(t, server).Password
}
