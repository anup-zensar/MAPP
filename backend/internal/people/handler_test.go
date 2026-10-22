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

// The bodies of the API's refusals, as the web app receives them.
const (
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
	sessionExpiredBody = `{"code":"session_expired","message":"Please sign in again","fields":[]}`
)

// The answers about Priya Raman as the seed has her: her preferences are the defaults, and she is in the
// team Lubricants Europe.
const (
	defaultPreferencesBody = `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`
	lubricantsEuropeBody   = `{"id":"T-1001","name":"Lubricants Europe","colour":"blue","active":true}`
	priyasTeamsBody        = `[` + lubricantsEuropeBody + `]`
)

// priyasAccessBody is Priya's access as the seed gives it (feature 014, FR-024): the Planner role for her
// teams, its twelve permissions in Lubricants Europe, the administrator to ask, and segregation of duties on.
var priyasAccessBody = `{"roles":[{"id":"RA-1005","role":{"id":"RL-1003","name":"Planner","builtIn":"planner"},` +
	`"scope":{"type":"myTeams","teams":` + priyasTeamsBody + `,"text":"My teams"},"scopeEmpty":false}],` +
	`"permissions":[` + inLubricantsEurope("plans.view", "plans.create", "plans.edit", "plans.publish", "campaigns.edit",
	"campaigns.delete", "activities.edit", "activities.delete", "activities.submit", "data.export", "finance.view",
	"finance.link") + `],` +
	`"administrators":[{"id":"U-1001","name":"Alex Morgan","initials":"AM","jobTitle":"Marketing administrator","active":true}],` +
	`"noSelfApproval":true}`

// inLubricantsEurope writes permissions held for Lubricants Europe only, as MyAccess lists them.
func inLubricantsEurope(keys ...string) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, `{"key":"`+key+`","all":false,"teams":[`+lubricantsEuropeBody+`]}`)
	}
	return strings.Join(parts, ",")
}

// GET /api/me answers who is signed in, with everything the sidebar and the account menu show (FR-012),
// and what My profile shows: the version, the five preferences, the person's teams (003 FR-021, FR-026) and
// their access (014 FR-024).
func TestGetMe_SignedInPerson_FR012(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodGet, "/api/me", nil)

	assertAnswer(t, response, http.StatusOK, `{"id":"U-1002","name":"Priya Raman","firstName":"Priya",`+
		`"email":"priya.raman@shell.com","jobTitle":"Marketer","initials":"PR",`+
		`"preferences":`+defaultPreferencesBody+`,"version":1,"teams":`+priyasTeamsBody+`,"access":`+priyasAccessBody+`}`)
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

	assertAnswer(t, response, http.StatusOK, defaultPreferencesBody)
}

// Only the preferences sent change, and the person keeps them (FR-011, FR-014).
func TestPatchPreferences_ThemeKeepsSidebarCollapsed_FR014(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	collapse := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"sidebarCollapsed": true})
	assertAnswer(t, collapse, http.StatusOK, `{"theme":"system","sidebarCollapsed":true,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)

	response := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusOK, `{"theme":"dark","sidebarCollapsed":true,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	want := people.PreferencesView{Theme: "dark", SidebarCollapsed: true, DisplayCurrency: "USD", DateFormat: "DD MMM YYYY", HomeScope: "myTeams",
		PinnedPlanIDs: []string{}}
	if !reflect.DeepEqual(me.Preferences, want) {
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

	assertAnswer(t, response, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"everything","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	if me.Preferences.HomeScope != "everything" {
		t.Errorf("GET /api/me home scope = %q, want everything", me.Preferences.HomeScope)
	}
	assertAnswer(t, refused, http.StatusBadRequest, `{"code":"validation_failed",`+
		`"message":"Some fields need your attention.","fields":[{"field":"homeScope","message":"Choose My teams or Everything."}]}`)
}

// Pinned plans are saved per person like the other preferences and come back with GET /api/me, in the
// order pinned; a repeated id is pinned once; an id that is no plan id is refused next to the field
// (018 FR-004 – FR-006, D-317).
func TestPatchPreferences_PinnedPlans_FR004_FR005(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	pinned := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"pinnedPlanIds": []string{"P-1002", "P-1001", "P-1002"}})
	refused := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"pinnedPlanIds": []string{"plan-7"}},
		apitest.SkipRequestValidation()) // the contract itself allows only plan ids

	assertAnswer(t, pinned, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"USD","dateFormat":"DD MMM YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":["P-1002","P-1001"]}`)
	assertAnswer(t, refused, http.StatusBadRequest, validationBody(`{"field":"pinnedPlanIds","message":"Pin plans by their id, like P-1001."}`))
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	if want := []string{"P-1002", "P-1001"}; !slices.Equal(me.Preferences.PinnedPlanIDs, want) {
		t.Errorf("GET /api/me pinned plans = %q, want %q (the refusal changed nothing)", me.Preferences.PinnedPlanIDs, want)
	}
}

// Changing preferences needs a signed-in person (FR-005).
func TestPatchPreferences_WithoutSession_FR005(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"theme": "dark"})

	assertAnswer(t, response, http.StatusUnauthorized, signInRequiredBody)
}

// My profile changes the signed-in person's name and job title: the answer is the signed-in person after
// the change, with initials and version, and every screen naming them shows the new name (003 FR-021,
// FR-022, US4 scenario 2).
func TestPatchMe_UpdatesProfile_FR021_FR022(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)

	response := priya.Do(t, http.MethodPatch, "/api/me", map[string]any{"name": "Priya Sharma", "jobTitle": "Senior marketer", "version": 1})

	assertAnswer(t, response, http.StatusOK, `{"id":"U-1002","name":"Priya Sharma","firstName":"Priya",`+
		`"email":"priya.raman@shell.com","jobTitle":"Senior marketer","initials":"PS",`+
		`"preferences":`+defaultPreferencesBody+`,"version":2,"teams":`+priyasTeamsBody+`,"access":`+priyasAccessBody+`}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	var details people.PersonDetails
	alex.Do(t, http.MethodGet, "/api/people/U-1002", nil).Decode(t, &details)
	if me.Name != "Priya Sharma" || details.Name != "Priya Sharma" || details.UpdatedBy.Name != "Priya Raman" {
		t.Errorf("after the change: GET /api/me %q, GET /api/people/U-1002 %q changed by %q; want the new name",
			me.Name, details.Name, details.UpdatedBy.Name)
	}
}

// A profile change from an out-of-date copy answers 409, naming who changed the person and when; nothing
// is saved (003 FR-022, 001 FR-029).
func TestPatchMe_VersionConflict_FR022(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	alex := server.SignIn(t, alexsEmail)
	server.Advance(time.Hour)
	changed := alex.Do(t, http.MethodPatch, "/api/people/U-1002", map[string]any{"name": "Priya Raman",
		"email": priyasEmail, "jobTitle": "Team leader", "teamIds": []string{"T-1001"}, "version": 1})
	if changed.Status != http.StatusOK {
		t.Fatalf("Alex's change = %d %s", changed.Status, changed.Body)
	}

	response := priya.Do(t, http.MethodPatch, "/api/me", map[string]any{"name": "Priya Sharma", "jobTitle": "Marketer", "version": 1})

	assertAnswer(t, response, http.StatusConflict, `{"code":"version_conflict",`+
		`"message":"Not saved — Alex Morgan changed this person","fields":[],`+
		`"conflict":{"changedBy":"Alex Morgan","changedAt":"2026-03-12T10:00:00Z"}}`)
	var me people.CurrentUser
	priya.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)
	if me.Name != "Priya Raman" || me.JobTitle != "Team leader" || me.Version != 2 {
		t.Errorf("GET /api/me = %+v, want only Alex's change", me)
	}
}

// An empty name and a job title over 120 characters are reported together (003 FR-022, SC-007).
func TestPatchMe_AllProblemsAtOnce_FR022(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	response := priya.Do(t, http.MethodPatch, "/api/me", map[string]any{"name": " ", "jobTitle": strings.Repeat("j", 121), "version": 1})

	assertAnswer(t, response, http.StatusBadRequest, validationBody(`{"field":"name","message":"Enter a name."},`+
		`{"field":"jobTitle","message":"Use 120 characters or fewer."}`))
}

// The display currency and the date format are preferences like the theme: the answer has all five, and
// anything but an active currency or one of the four date formats is refused next to its field, both at once
// (003 FR-023, FR-024).
func TestPatchPreferences_DisplayCurrencyAndDateFormat_FR023_FR024(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)

	chosen := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"displayCurrency": "EUR", "dateFormat": "DD/MM/YYYY"})
	invalid := priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"displayCurrency": "XYZ", "dateFormat": "DD.MM.YYYY"})

	assertAnswer(t, chosen, http.StatusOK, `{"theme":"system","sidebarCollapsed":false,"displayCurrency":"EUR","dateFormat":"DD/MM/YYYY","homeScope":"myTeams","historyIncludeInside":false,"pinnedPlanIds":[]}`)
	assertAnswer(t, invalid, http.StatusBadRequest, validationBody(`{"field":"displayCurrency","message":"Choose one of the active currencies."},`+
		`{"field":"dateFormat","message":"Choose one of the date formats."}`))
}

// The display currencies offered are the active currencies of Settings, by code; a currency set inactive
// after a person chose it is still offered to that person, marked inactive, and to nobody else (003
// FR-023, D-063).
func TestGetDisplayCurrencies_FR023(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	alex := server.SignIn(t, alexsEmail)

	before := daniel.Do(t, http.MethodGet, "/api/me/display-currencies", nil)
	priya.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"displayCurrency": "GBP"})
	var gbp struct {
		Version int `json:"version"`
	}
	alex.Do(t, http.MethodGet, "/api/currencies/GBP", nil).Decode(t, &gbp)
	inactive := alex.Do(t, http.MethodPost, "/api/currencies/GBP/status", map[string]any{"active": false, "version": gbp.Version})
	if inactive.Status != http.StatusOK {
		t.Fatalf("setting GBP inactive = %d %s", inactive.Status, inactive.Body)
	}
	priyas := priya.Do(t, http.MethodGet, "/api/me/display-currencies", nil)
	daniels := daniel.Do(t, http.MethodGet, "/api/me/display-currencies", nil)

	cny, eur := `{"code":"CNY","name":"Chinese yuan","active":true}`, `{"code":"EUR","name":"Euro","active":true}`
	inr, usd := `{"code":"INR","name":"Indian rupee","active":true}`, `{"code":"USD","name":"US dollar","active":true}`
	assertAnswer(t, before, http.StatusOK, `{"items":[`+cny+`,`+eur+`,{"code":"GBP","name":"Pound sterling","active":true},`+
		inr+`,`+usd+`],"total":5}`)
	assertAnswer(t, priyas, http.StatusOK, `{"items":[`+cny+`,`+eur+`,{"code":"GBP","name":"Pound sterling","active":false},`+
		inr+`,`+usd+`],"total":5}`)
	assertAnswer(t, daniels, http.StatusOK, `{"items":[`+cny+`,`+eur+`,`+inr+`,`+usd+`],"total":4}`)
}

// Preferences belong to the person, not to the browser: signed in from another browser, the person sees
// the display currency and date format they chose (003 FR-025).
func TestPreferences_SameInAnotherBrowser_FR025(t *testing.T) {
	server := apitest.New(t)
	firstBrowser := server.SignIn(t, priyasEmail)
	secondBrowser := server.SignIn(t, priyasEmail)

	firstBrowser.Do(t, http.MethodPatch, "/api/me/preferences", map[string]any{"displayCurrency": "EUR", "dateFormat": "YYYY-MM-DD"})
	var me people.CurrentUser
	secondBrowser.Do(t, http.MethodGet, "/api/me", nil).Decode(t, &me)

	if me.Preferences.DisplayCurrency != "EUR" || me.Preferences.DateFormat != "YYYY-MM-DD" {
		t.Errorf("preferences in the second browser = %+v, want EUR and YYYY-MM-DD", me.Preferences)
	}
}

// assertAnswer checks the status and the exact body of an answer.
func assertAnswer(t *testing.T, response *apitest.Response, status int, body string) {
	t.Helper()
	if response.Status != status || string(response.Body) != body {
		t.Errorf("answer = %d %s\nwant %d %s", response.Status, response.Body, status, body)
	}
}
