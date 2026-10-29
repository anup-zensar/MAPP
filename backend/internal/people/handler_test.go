package people_test

import (
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/apitest"
)

const priyasEmail = "priya.raman@shell.com"

const (
	signInRequiredBody = `{"error":{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}}`
	sessionExpiredBody = `{"error":{"code":"session_expired","message":"Please sign in again","fields":[]}}`
)

const (
	defaultPreferencesBody = `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`
	lubricantsEuropeBody   = `{"id":"T-1001","name":"Lubricants Europe","colour":"blue","active":true}`
	priyasTeamsBody        = `[` + lubricantsEuropeBody + `]`
)

var priyasAccessBody = `{"roles":[{"id":"RA-1005","role":{"id":"RL-1003","name":"Planner","builtIn":"planner"},` +
	`"scope":{"type":"myTeams","teams":` + priyasTeamsBody + `,"text":"My teams"},"scopeEmpty":false}],` +
	`"permissions":[` + inLubricantsEurope("plans.view", "plans.create", "plans.edit", "plans.publish", "campaigns.edit",
	"campaigns.delete", "activities.edit", "activities.delete", "activities.submit", "data.export", "finance.view",
	"purchaseRequests.edit", "purchaseRequests.delete", "purchaseRequests.submit") + `],` +
	`"administrators":[{"id":"U-1001","name":"Alex Morgan","initials":"AM","jobTitle":"Marketing administrator","active":true}],` +
	`"noSelfApproval":true}`

func inLubricantsEurope(keys ...string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, `{"key":"`+key+`","all":false,"teams":[`+lubricantsEuropeBody+`]}`)
	}
	return strings.Join(parts, ",")
}

// GET /api/v1/me answers who is signed in, with everything the sidebar and the account menu show (FR-012),
// and what My profile shows: the version, the five preferences, the person's teams (003 FR-021, FR-026) and
// their access (014 FR-024).
func TestGetMe_SignedInPerson_FR012(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/v1/me", nil)

	assertAnswer(t, response, http.StatusOK, `{"id":"U-1002","name":"Priya Raman","firstName":"Priya",`+
		`"email":"priya.raman@shell.com","jobTitle":"Marketer","initials":"PR",`+
		`"preferences":`+defaultPreferencesBody+`,"version":1,"teams":`+priyasTeamsBody+`,"access":`+priyasAccessBody+`}`)
}

// After more than 12 hours without a request the session has expired, and the person is asked to sign in
// again (FR-006, US4 scenario 2).
func TestGetMe_SessionExpired_FR006(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.Advance(12*time.Hour + 2*time.Minute)

	response := priya.Do(t, http.MethodGet, "/api/v1/me", nil)

	assertAnswer(t, response, http.StatusUnauthorized, sessionExpiredBody)
}

func TestPatchPreferences_ThemeKeepsSidebarCollapsed_FR014(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	collapse := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"sidebarCollapsed": true})
	assertAnswer(t, collapse, http.StatusOK, `{"theme":"system","sidebarCollapsed":true,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)

	response := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusOK, `{"theme":"dark","sidebarCollapsed":true,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	want := people.PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}
	if !reflect.DeepEqual(me.Preferences, want) {
		t.Errorf("GET /api/v1/me preferences = %+v, want %+v", me.Preferences, want)
	}
}

// Home's My teams / Everything switch is saved per person and comes back with GET /api/v1/me; a value other
// than myTeams or everything is refused next to the field (009 FR-003).
func TestPatchPreferences_HomeScope_FR003(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"homeScope": "everything"})
	refused := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"homeScope": "all"},
		apitest.SkipRequestValidation())

	assertAnswer(t, response, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"everything","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	if me.Preferences.HomeScope != "everything" {
		t.Errorf("GET /api/v1/me home scope = %q, want everything", me.Preferences.HomeScope)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"error":{"code":"validation_failed",`+
		`"message":"Some fields need your attention.","fields":[{"field":"homeScope","message":"Choose My teams or Everything."}]}}`)
}

// Pinned plans are saved per person like the other preferences and come back with GET /api/v1/me, in the
// order pinned; a repeated id is pinned once; an id that is no plan id is refused next to the field
// (018 FR-004 – FR-006, D-317).
func TestPatchPreferences_PinnedPlans_FR004_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	pinned := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"pinnedPlanIds": []string{"P-1002", "P-1001", "P-1002"}})
	refused := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"pinnedPlanIds": []string{"plan-7"}},
		apitest.SkipRequestValidation())

	assertAnswer(t, pinned, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":["P-1002","P-1001"]}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"pinnedPlanIds","message":"Pin plans by their id, like P-1001."}`))
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	if want := []string{"P-1002", "P-1001"}; !slices.Equal(me.Preferences.PinnedPlanIDs, want) {
		t.Errorf("GET /api/v1/me pinned plans = %q, want %q (the refusal changed nothing)", me.Preferences.PinnedPlanIDs, want)
	}
}

func TestPatchPreferences_WithoutSession_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
}

func TestPatchMe_UpdatesProfile_FR021_FR022(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	response := priya.Do(t, http.MethodPatch, "/api/v1/me", map[string]any{"name": "Priya Sharma", "jobTitle": "Senior marketer", "version": 1})

	assertAnswer(t, response, http.StatusOK, `{"id":"U-1002","name":"Priya Sharma","firstName":"Priya",`+
		`"email":"priya.raman@shell.com","jobTitle":"Senior marketer","initials":"PS",`+
		`"preferences":`+defaultPreferencesBody+`,"version":2,"teams":`+priyasTeamsBody+`,"access":`+priyasAccessBody+`}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	var details people.PersonDetails
	alex.Do(t, http.MethodGet, "/api/v1/people/U-1002", nil).Decode(t, &details)
	if me.Name != "Priya Sharma" || details.Name != "Priya Sharma" || details.UpdatedBy.Name != "Priya Raman" {
		t.Errorf("after the change: GET /api/v1/me %q, GET /api/v1/people/U-1002 %q changed by %q; want the new name",
			me.Name, details.Name, details.UpdatedBy.Name)
	}
}

func TestPatchPreferences_DisplayCurrencyAndDateFormat_FR023_FR024(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	chosen := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "EUR", "dateFormat": "DD/MM/YYYY"})
	invalid := priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "XYZ", "dateFormat": "DD.MM.YYYY"})

	assertAnswer(t, chosen, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"EUR","dateFormat":"DD/MM/YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"displayCurrency","message":"Choose one of the active currencies."},`+
		`{"field":"dateFormat","message":"Choose one of the date formats."}`))
}

func TestGetDisplayCurrencies_FR023(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	alex := server.SignIn(t, alexsEmail)

	before := daniel.Do(t, http.MethodGet, "/api/v1/me/display-currencies", nil)
	priya.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "GBP"})
	var gbp struct {
		Version int `json:"version"`
	}
	alex.Do(t, http.MethodGet, "/api/v1/currencies/GBP", nil).Decode(t, &gbp)
	inactive := alex.Do(t, http.MethodPost, "/api/v1/currencies/GBP/status", map[string]any{"active": false, "version": gbp.Version})
	if inactive.Status != http.StatusOK {
		t.Fatalf("setting GBP inactive = %d %s", inactive.Status, inactive.Body)
	}
	priyas := priya.Do(t, http.MethodGet, "/api/v1/me/display-currencies", nil)
	daniels := daniel.Do(t, http.MethodGet, "/api/v1/me/display-currencies", nil)

	cny, eur := `{"code":"CNY","name":"Chinese yuan","active":true}`, `{"code":"EUR","name":"Euro","active":true}`
	inr, usd := `{"code":"INR","name":"Indian rupee","active":true}`, `{"code":"USD","name":"US dollar","active":true}`
	assertAnswer(t, before, http.StatusOK, `{"items":[`+cny+`,`+eur+`,{"code":"GBP","name":"Pound sterling","active":true},`+
		inr+`,`+usd+`],"total":5}`)
	assertAnswer(t, priyas, http.StatusOK, `{"items":[`+cny+`,`+eur+`,{"code":"GBP","name":"Pound sterling","active":false},`+
		inr+`,`+usd+`],"total":5}`)
	assertAnswer(t, daniels, http.StatusOK, `{"items":[`+cny+`,`+eur+`,`+inr+`,`+usd+`],"total":4}`)
}

func TestPreferences_SameInAnotherBrowser_FR025(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	firstBrowser := server.SignIn(t, priyasEmail)
	secondBrowser := server.SignIn(t, priyasEmail)

	firstBrowser.Do(t, http.MethodPatch, "/api/v1/me/preferences", map[string]any{"displayCurrency": "EUR", "dateFormat": "YYYY-MM-DD"})
	var me people.CurrentUser
	secondBrowser.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)

	if me.Preferences.DisplayCurrency != "EUR" || me.Preferences.DateFormat != "YYYY-MM-DD" {
		t.Errorf("preferences in the second browser = %+v, want EUR and YYYY-MM-DD", me.Preferences)
	}
}

func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
