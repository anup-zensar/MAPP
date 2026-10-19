package people_test

import (
	"net/http"
	"testing"
	"time"

	"northstar/internal/people"
	"northstar/internal/platform/apitest"
)

const priyasEmail = "priya.raman@northstar.demo"

// The bodies of the API's refusals, as the web app receives them.
const (
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	sessionExpiredBody = `{"code":"session_expired","message":"Please sign in again","fields":[]}`
)

// GET /api/me answers who is signed in, with everything the sidebar and the account menu show (FR-012).
func TestGetMe_SignedInPerson_FR012(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/me", nil)

	assertAnswer(t, response, http.StatusOK, `{"id":"U-1002","name":"Priya Raman","firstName":"Priya",`+
		`"email":"priya.raman@northstar.demo","jobTitle":"Marketer","initials":"PR",`+
		`"preferences":{"theme":"system","sidebarCollapsed":false,"homeScope":"myTeams"}}`)
}

// Without a session cookie, the API asks the person to sign in (FR-005).
func TestGetMe_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodGet, "/api/me", nil)

	assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
}

// After more than 12 hours without a request the session has expired, and the person is asked to sign in
// again (FR-006, US4 scenario 2).
func TestGetMe_SessionExpired_FR006(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.Advance(12*time.Hour + 2*time.Minute)

	response := priya.Do(t, http.MethodGet, "/api/me", nil)

	assertAnswer(t, response, http.StatusUnauthorized, sessionExpiredBody)
}

// Sending no preference changes nothing (FR-011, FR-014).
func TestPatchPreferences_NothingSent_FR011(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{})

	assertAnswer(t, response, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"homeScope":"myTeams"}`)
}

// Only the preferences sent change, and the person keeps them (FR-011, FR-014).
func TestPatchPreferences_ThemeKeepsSidebarCollapsed_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	collapse := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"sidebarCollapsed": true})
	assertAnswer(t, collapse, http.StatusOK, `{"theme":"system","sidebarCollapsed":true,"homeScope":"myTeams"}`)

	response := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusOK, `{"theme":"dark","sidebarCollapsed":true,"homeScope":"myTeams"}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	if want := (people.PreferencesView{Theme: "dark", SidebarCollapsed: true, HomeScope: "myTeams"}); me.Preferences != want {
		t.Errorf("GET /api/me preferences = %+v, want %+v", me.Preferences, want)
	}
}

// A theme other than system, light or dark is refused next to the field (FR-014, FR-020).
func TestPatchPreferences_InvalidTheme_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"theme": "blue"},
		apitest.SkipRequestValidation()) // the contract itself allows only the three themes

	assertAnswer(t, response, http.StatusBadRequest, `{"code":"validation_failed",`+
		`"message":"Some fields need your attention.","fields":[{"field":"theme","message":"Choose system, light or dark."}]}`)
}

// Home's My teams / Everything switch is saved per person and comes back with GET /api/me; a value other
// than myTeams or everything is refused next to the field (009 FR-003).
func TestPatchPreferences_HomeScope_FR003(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"homeScope": "everything"})
	refused := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"homeScope": "all"},
		apitest.SkipRequestValidation()) // the contract itself allows only the two scopes

	assertAnswer(t, response, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"homeScope":"everything"}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	if me.Preferences.HomeScope != "everything" {
		t.Errorf("GET /api/me home scope = %q, want everything", me.Preferences.HomeScope)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed",`+
		`"message":"Some fields need your attention.","fields":[{"field":"homeScope","message":"Choose My teams or Everything."}]}`)
}

// Changing preferences needs a signed-in person (FR-005).
func TestPatchPreferences_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
