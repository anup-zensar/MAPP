package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"mapp/internal/auth"
	"mapp/internal/auth/authtest"
	"mapp/internal/people"
	"mapp/internal/people/peopletest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/web"
)

const (
	priyasEmail  = "priya.raman@shell.com"
	demoPassword = "mapp"
)

const lubricantsEurope = `{"id":"T-1001","name":"Lubricants Europe","colour":"blue","active":true}`

// priyaBody carries her access: Planner in her team, the administrator to ask and no self-approval.
var priyaBody = `{"id":"U-1002","name":"Priya Raman","firstName":"Priya","email":"priya.raman@shell.com",` +
	`"jobTitle":"Marketer","initials":"PR",` +
	`"preferences":{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]},` +
	`"version":1,"teams":[` + lubricantsEurope + `],` +
	`"access":{"roles":[{"id":"RA-1005","role":{"id":"RL-1003","name":"Planner","builtIn":"planner"},` +
	`"scope":{"type":"myTeams","teams":[` + lubricantsEurope + `],"text":"My teams"},"scopeEmpty":false}],` +
	`"permissions":[` + inLubricantsEurope("plans.view", "plans.create", "plans.edit", "plans.publish", "campaigns.edit",
	"campaigns.delete", "activities.edit", "activities.delete", "activities.submit", "data.export", "finance.view",
	"finance.link", "purchaseRequests.edit", "purchaseRequests.delete", "purchaseRequests.submit") + `],` + // 032 FR-049
	`"administrators":[{"id":"U-1001","name":"Alex Morgan","initials":"AM","jobTitle":"Marketing administrator","active":true}],` +
	`"noSelfApproval":true}}`

func inLubricantsEurope(keys ...string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, `{"key":"`+key+`","all":false,"teams":[`+lubricantsEurope+`]}`)
	}
	return strings.Join(parts, ",")
}

const (
	signInRequiredBody = `{"error":{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}}`
	incorrectBody      = `{"error":{"code":"incorrect_credentials","message":"Incorrect e-mail or password.","fields":[]}}`
)

func TestGetDemoPeople_ActivePeopleAndPassword_FR001_FR002(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodGet, "/api/v1/auth/demo-people", nil)

	if response.Status != http.StatusOK {
		t.Fatalf("status = %d %s, want 200", response.Status, response.Body)
	}
	var got auth.DemoPeople
	response.Decode(t, &got)
	want := auth.DemoPeople{Password: demoPassword, People: []people.DemoPerson{
		{ID: "U-1001", Name: "Alex Morgan", Email: "alex.morgan@shell.com", JobTitle: "Marketing administrator", Initials: "AM"},
		{ID: "U-1005", Name: "Carlos Mendes", Email: "carlos.mendes@shell.com", JobTitle: "Business champion", Initials: "CM"},
		{ID: "U-1003", Name: "Daniel Okoro", Email: "daniel.okoro@shell.com", JobTitle: "Marketing team leader", Initials: "DO"},
		{ID: "U-1007", Name: "Lea Santos", Email: "lea.santos@shell.com", JobTitle: "Requisitioner", Initials: "LS"},
		{ID: "U-1004", Name: "Mei Lin", Email: "mei.lin@shell.com", JobTitle: "Marketer", Initials: "ML"},
		{ID: "U-1002", Name: "Priya Raman", Email: "priya.raman@shell.com", JobTitle: "Marketer", Initials: "PR"},
		{ID: "U-1006", Name: "Sara Novak", Email: "sara.novak@shell.com", JobTitle: "Local finance advisor", Initials: "SN"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("demo people =\n%+v\nwant\n%+v", got, want)
	}
}

// The sign-in page can then explain how to load the demo data.
func TestDemoPeople_EmptyStore_FR028(t *testing.T) {
	t.Parallel()
	server := apitest.New(t, apitest.WithoutSeed())

	response := server.Do(t, http.MethodGet, "/api/v1/auth/demo-people", nil)

	assertAnswer(t, response, http.StatusOK, `{"password":"mapp","people":[]}`)
}

// The cookie lasts until the browser closes (research R6).
func TestSignIn_SetsTheSessionCookie_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(" Priya.Raman@Shell.COM ", demoPassword))

	assertAnswer(t, response, http.StatusOK, priyaBody)
	cookie := response.Cookie(auth.CookieName)
	if cookie == nil {
		t.Fatal("no session cookie was set")
	}
	if cookie.Value == "" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/" ||
		cookie.Secure || cookie.MaxAge != 0 || !cookie.Expires.IsZero() {
		t.Errorf("cookie = %+v, want a value, HttpOnly, SameSite=Strict, Path=/, not Secure over HTTP, no expiry", cookie)
	}
	me := server.Do(t, http.MethodGet, "/api/v1/me", nil, apitest.WithCookie(cookie))
	assertAnswer(t, me, http.StatusOK, priyaBody)
}

func TestSignIn_EmptyFields_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn("", ""))

	assertAnswer(t, response, http.StatusBadRequest, `{"error":{"code":"validation_failed","message":"Some fields need your attention.",`+
		`"fields":[{"field":"email","message":"Enter your e-mail"},{"field":"password","message":"Enter the password"}]}}`)
}

// The answer sets no cookie; the service test proves an unknown e-mail gets the same answer.
func TestSignIn_IncorrectCredentials_FR004(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, "wrong"))

	assertAnswer(t, response, http.StatusUnauthorized, incorrectBody)
	if cookie := response.Cookie(auth.CookieName); cookie != nil {
		t.Errorf("a session cookie was set: %+v", cookie)
	}
}

func TestSignIn_EndsThisBrowsersPreviousSession_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	first := sessionCookie(t, server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn(priyasEmail, demoPassword)))

	again := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", signIn("daniel.okoro@shell.com", demoPassword),
		apitest.WithCookie(first))

	second := sessionCookie(t, again)
	assertAnswer(t, server.Do(t, http.MethodGet, "/api/v1/me", nil, apitest.WithCookie(first)),
		http.StatusUnauthorized, signInRequiredBody)
	if me := server.Do(t, http.MethodGet, "/api/v1/me", nil, apitest.WithCookie(second)); me.Status != http.StatusOK {
		t.Errorf("the new session: GET /api/v1/me = %d %s, want 200", me.Status, me.Body)
	}
}

// The browser is told to forget the cookie.
func TestSignOut_EndsTheSession_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPost, "/api/v1/auth/sign-out", nil)

	if response.Status != http.StatusNoContent || len(response.Body) != 0 {
		t.Errorf("answer = %d %s, want 204 without a body", response.Status, response.Body)
	}
	if cleared := response.Cookie(auth.CookieName); cleared == nil || cleared.Value != "" || cleared.MaxAge >= 0 {
		t.Errorf("cookie = %+v, want the session cookie cleared (empty, Max-Age=0)", cleared)
	}
	assertAnswer(t, priya.Do(t, http.MethodGet, "/api/v1/me", nil), http.StatusUnauthorized, signInRequiredBody)
}

func TestSignOut_WithoutSession_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPost, "/api/v1/auth/sign-out", nil)

	if response.Status != http.StatusNoContent {
		t.Errorf("answer = %d %s, want 204", response.Status, response.Body)
	}
}

// The sign-in check's transaction has finished before the handler runs (research R6).
func TestRequireSignIn_HandlerCanWrite_FR007(t *testing.T) {
	t.Parallel()
	db := apitest.DemoDB(t, false)
	notes := fake.TableOf(db, "notes", func(n note) string { return n.ID })
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	service := auth.NewService(db, authtest.NewStore(db), peopletest.NewStore(db),
		auth.Options{DemoMode: true, DemoPassword: demoPassword, Now: now})
	token, _, err := service.SignIn(t.Context(), priyasEmail, demoPassword, "")
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	var who actor.Actor
	saveANote := func(c *gin.Context) {
		who, _ = actor.From(c.Request.Context())
		saveErr := db.Write(c.Request.Context(), func(ctx context.Context) error {
			return notes.Put(ctx, note{ID: "N-1", WrittenBy: who.Name})
		})
		if saveErr != nil {
			web.RespondError(c, saveErr)
			return
		}
		c.Status(http.StatusNoContent)
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.POST("/api/v1/notes", service.RequireSignIn, saveANote)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://127.0.0.1/api/v1/notes", nil)
	request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: token})
	answer := httptest.NewRecorder()

	engine.ServeHTTP(answer, request)

	if answer.Code != http.StatusNoContent {
		t.Fatalf("status = %d %s, want 204", answer.Code, answer.Body)
	}
	if who != (actor.Actor{ID: "U-1002", Name: "Priya Raman"}) {
		t.Errorf("actor = %+v, want Priya Raman (U-1002)", who)
	}
	saved, found, err := notes.Get("N-1")
	if err != nil || !found || saved.WrittenBy != "Priya Raman" {
		t.Errorf("saved note = %+v (found %v, %v), want one written by Priya Raman", saved, found, err)
	}
}

type note struct {
	ID        string `json:"id"`
	WrittenBy string `json:"writtenBy"`
}

func signIn(email, password string) map[string]string {
	return map[string]string{"email": email, "password": password}
}

func sessionCookie(t *testing.T, response *apitest.Response) *http.Cookie {
	t.Helper()
	cookie := response.Cookie(auth.CookieName)
	if response.Status != http.StatusOK || cookie == nil {
		t.Fatalf("sign-in answer = %d %s, want 200 with a session cookie", response.Status, response.Body)
	}
	return cookie
}

func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
