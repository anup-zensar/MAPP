package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"northstar/internal/auth"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/apitest"
	"northstar/internal/platform/store"
	"northstar/internal/platform/web"
	"northstar/seed"
)

const (
	priyasEmail  = "priya.raman@northstar.demo"
	demoPassword = "northstar"
)

// lubricantsEurope is the team T-1001 as answers show it.
const lubricantsEurope = `{"id":"T-1001","name":"Lubricants Europe","colour":"blue","active":true}`

// priyaBody is Priya Raman as the API answers her: with her access (feature 014, FR-024) — the Planner role
// for her teams, its twelve permissions in Lubricants Europe, the administrator to ask and the
// segregation-of-duties rule.
var priyaBody = `{"id":"U-1002","name":"Priya Raman","firstName":"Priya","email":"priya.raman@northstar.demo",` +
	`"jobTitle":"Marketer","initials":"PR",` +
	`"preferences":{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams"},` +
	`"version":1,"teams":[` + lubricantsEurope + `],` +
	`"access":{"roles":[{"id":"RA-1005","role":{"id":"RL-1003","name":"Planner","builtIn":"planner"},` +
	`"scope":{"type":"myTeams","teams":[` + lubricantsEurope + `],"text":"My teams"},"scopeEmpty":false}],` +
	`"permissions":[` + inLubricantsEurope("plans.view", "plans.create", "plans.edit", "plans.publish", "campaigns.edit",
	"campaigns.delete", "activities.edit", "activities.delete", "activities.submit", "data.export", "finance.view",
	"finance.link") + `],` +
	`"administrators":[{"id":"U-1001","name":"Alex Morgan","initials":"AM","jobTitle":"Marketing administrator","active":true}],` +
	`"noSelfApproval":true}}`

// inLubricantsEurope writes permissions held for Lubricants Europe only, as MyAccess lists them.
func inLubricantsEurope(keys ...string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, `{"key":"`+key+`","all":false,"teams":[`+lubricantsEurope+`]}`)
	}
	return strings.Join(parts, ",")
}

// The bodies of the API's answers, as the web app receives them.
const (
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	incorrectBody      = `{"code":"incorrect_credentials","message":"Incorrect e-mail or password.","fields":[]}`
)

// The sign-in page lists the active demo people, sorted by name, with the shared demo password
// (FR-001, FR-002).
func TestGetDemoPeople_ActivePeopleAndPassword_FR001_FR002(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodGet, "/api/auth/demo-people", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var got auth.DemoPeople
	response.Decode(t, &got)
	want := auth.DemoPeople{Password: demoPassword, People: []people.DemoPerson{
		{ID: "U-1001", Name: "Alex Morgan", Email: "alex.morgan@northstar.demo", JobTitle: "Marketing administrator", Initials: "AM"},
		{ID: "U-1005", Name: "Carlos Mendes", Email: "carlos.mendes@northstar.demo", JobTitle: "Business champion", Initials: "CM"},
		{ID: "U-1003", Name: "Daniel Okoro", Email: "daniel.okoro@northstar.demo", JobTitle: "Marketing team leader", Initials: "DO"},
		{ID: "U-1004", Name: "Mei Lin", Email: "mei.lin@northstar.demo", JobTitle: "Marketer", Initials: "ML"},
		{ID: "U-1002", Name: "Priya Raman", Email: "priya.raman@northstar.demo", JobTitle: "Marketer", Initials: "PR"},
		{ID: "U-1006", Name: "Sara Novak", Email: "sara.novak@northstar.demo", JobTitle: "Local finance advisor", Initials: "SN"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("demo people =\n%+v\nwant\n%+v", got, want)
	}
}

// Without demo data the list is empty, never null, and the password is still there, so the sign-in page
// can explain how to load the demo data (FR-028).
func TestDemoPeople_EmptyStore_FR028(t *testing.T) {
	server := apitest.New(t, apitest.WithoutSeed())

	response := server.Do(t, http.MethodGet, "/api/auth/demo-people", nil)

	assertAnswer(t, response, http.StatusOK, `{"password":"northstar","people":[]}`)
}

// Signing in answers the signed-in person and sets the session cookie: HttpOnly, SameSite=Strict, for the
// whole site, and without an expiry date, so it lasts until the browser closes (FR-003, FR-006, research
// R6). The e-mail's letter case and surrounding spaces do not matter.
func TestSignIn_SetsTheSessionCookie_FR003(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/auth/sign-in", signIn(" Priya.Raman@NorthStar.demo ", demoPassword))

	assertAnswer(t, response, http.StatusOK, priyaBody)
	cookie := response.Cookie(auth.CookieName)
	if cookie == nil {
		t.Fatal("no session cookie was set")
	}
	if cookie.Value == "" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" ||
		cookie.Secure || cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
		t.Errorf("cookie = %+v, want a value, HttpOnly, SameSite=Strict, Path=/, not Secure over HTTP, no expiry", cookie)
	}
	me := server.Do(t, http.MethodGet, "/api/me", nil, apitest.WithCookie(cookie))
	assertAnswer(t, me, http.StatusOK, priyaBody)
}

// Every empty field is reported in one answer (FR-004).
func TestSignIn_EmptyFields_FR004(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/auth/sign-in", signIn("", ""))

	assertAnswer(t, response, http.StatusBadRequest, `{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"email","message":"Enter your e-mail"},{"field":"password","message":"Enter the password"}]}`)
}

// An unknown e-mail and a wrong password get the same answer, without saying which was wrong, and no
// session (FR-004).
func TestSignIn_IncorrectCredentials_FR004(t *testing.T) {
	server := apitest.New(t)
	attempts := []struct {
		name string
		body map[string]string
	}{
		{"wrong password", signIn(priyasEmail, "wrong")},
		{"unknown e-mail", signIn("nobody@northstar.demo", demoPassword)},
	}

	for _, attempt := range attempts {
		t.Run(attempt.name, func(t *testing.T) {
			response := server.Do(t, http.MethodPost, "/api/auth/sign-in", attempt.body)

			assertAnswer(t, response, http.StatusUnauthorized, incorrectBody)
			if cookie := response.Cookie(auth.CookieName); cookie != nil {
				t.Errorf("a session cookie was set: %+v", cookie)
			}
		})
	}
}

// Signing in again in the same browser ends the browser's earlier session (FR-006).
func TestSignIn_EndsThisBrowsersPreviousSession_FR006(t *testing.T) {
	server := apitest.New(t)
	first := sessionCookie(t, server.Do(t, http.MethodPost, "/api/auth/sign-in", signIn(priyasEmail, demoPassword)))

	again := server.Do(t, http.MethodPost, "/api/auth/sign-in", signIn("daniel.okoro@northstar.demo", demoPassword),
		apitest.WithCookie(first))

	second := sessionCookie(t, again)
	assertAnswer(t, server.Do(t, http.MethodGet, "/api/me", nil, apitest.WithCookie(first)),
		http.StatusUnauthorized, signInRequiredBody)
	if me := server.Do(t, http.MethodGet, "/api/me", nil, apitest.WithCookie(second)); me.Status != http.StatusOK {
		t.Errorf("the new session: GET /api/me = %d %s, want 200", me.Status, me.Body)
	}
}

// Signing out ends the session and tells the browser to forget the cookie (FR-006, US4 scenario 1).
func TestSignOut_EndsTheSession_FR006(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/auth/sign-out", nil)

	if response.Status != http.StatusNoContent || len(response.Body) != 0 {
		t.Errorf("answer = %d %s, want 204 without a body", response.Status, response.Body)
	}
	if cleared := response.Cookie(auth.CookieName); cleared == nil || cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("cookie = %+v, want the session cookie cleared (empty, Max-Age=0)", cleared)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/me", nil), http.StatusUnauthorized, signInRequiredBody)
}

// Signing out always succeeds, also when nobody is signed in.
func TestSignOut_WithoutSession_FR006(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/auth/sign-out", nil)

	if response.Status != http.StatusNoContent {
		t.Errorf("answer = %d %s, want 204", response.Status, response.Body)
	}
}

// A handler behind RequireSignIn gets the signed-in person as the actor and can save changes of its own:
// the sign-in check's transaction has finished before the handler runs (FR-007, research R6).
func TestRequireSignIn_HandlerCanWrite_FR007(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err := seed.Load(t.Context(), db, now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	service := auth.NewService(db, auth.NewJSONRepository(), people.NewJSONRepository(), true, demoPassword, now)
	token, _, err := service.SignIn(t.Context(), priyasEmail, demoPassword, "")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	var who actor.Actor
	saveANote := service.RequireSignIn(func(w http.ResponseWriter, r *http.Request) {
		who, _ = actor.From(r.Context())
		err := db.Write(r.Context(), func(ctx context.Context) error {
			return store.Put(ctx, "notes", "N-1", map[string]string{"writtenBy": who.Name})
		})
		if err != nil {
			web.WriteError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/notes", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	answer := httptest.NewRecorder()

	saveANote(answer, request)

	if answer.Code != http.StatusNoContent {
		t.Fatalf("status = %d %s, want 204", answer.Code, answer.Body)
	}
	if who != (actor.Actor{ID: "U-1002", Name: "Priya Raman"}) {
		t.Errorf("actor = %+v, want Priya Raman (U-1002)", who)
	}
	reopened, err := store.Open(dir) // the note was saved to disk, not lost in a finished transaction
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	err = reopened.Read(t.Context(), func(ctx context.Context) error {
		note, found, err := store.Get[map[string]string](ctx, "notes", "N-1")
		if !found || note["writtenBy"] != "Priya Raman" {
			t.Errorf("saved note = %v (found %v), want one written by Priya Raman", note, found)
		}
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// signIn is the body of POST /api/auth/sign-in.
func signIn(email, password string) map[string]string {
	return map[string]string{"email": email, "password": password}
}

// sessionCookie returns the session cookie a sign-in answer set. It stops the test when the sign-in failed.
func sessionCookie(t *testing.T, response *apitest.Response) *http.Cookie {
	t.Helper()
	cookie := response.Cookie(auth.CookieName)
	if response.Status != http.StatusOK || cookie == nil {
		t.Fatalf("sign-in answer = %d %s, want 200 with a session cookie", response.Status, response.Body)
	}
	return cookie
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
