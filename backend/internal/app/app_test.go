package app_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"northstar/internal/app"
	"northstar/internal/auth"
	"northstar/internal/platform/apitest"
	"northstar/internal/platform/store"
)

const (
	notFoundBody       = `{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}`
	otherSiteBody      = `{"code":"bad_request","message":"This request came from another site and was refused.","fields":[]}`
	signInRequiredBody = `{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}`
)

// priyaSignsIn is the body of a sign-in as Priya Raman with the demo password.
var priyaSignsIn = map[string]string{"email": "priya.raman@northstar.demo", "password": "northstar"}

// GET /api/health answers without a session and says in which environment the API runs and whether demo
// sign-in is on (FR-009).
func TestHealth_EnvironmentAndDemoSignIn_FR009(t *testing.T) {
	for env, demoSignIn := range map[string]bool{"local": true, "development": true, "production": false} {
		t.Run(env, func(t *testing.T) {
			server := apitest.New(t, apitest.WithEnv(env))

			response := server.Do(t, http.MethodGet, "/api/health", nil)

			if response.Status != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.Status)
			}
			var health struct {
				Status      string `json:"status"`
				Environment string `json:"environment"`
				DemoSignIn  bool   `json:"demoSignIn"`
			}
			response.Decode(t, &health)
			if health.Status != "ok" || health.Environment != env || health.DemoSignIn != demoSignIn {
				t.Errorf("health = %+v, want status ok, environment %s, demoSignIn %v", health, env, demoSignIn)
			}
		})
	}
}

// Demo sign-in exists only in the local and development environments. In production both of its routes
// answer 404, as for any path the API does not serve, and the health check says it is off (FR-009).
func TestDemoMode_RoutesOnlyInLocalAndDevelopment_FR009(t *testing.T) {
	environments := []struct {
		env        string
		demoSignIn bool
	}{
		{"local", true},
		{"development", true},
		{"production", false},
	}
	for _, tt := range environments {
		t.Run(tt.env, func(t *testing.T) {
			server := apitest.New(t, apitest.WithEnv(tt.env))

			demoPeople := server.Do(t, http.MethodGet, "/api/auth/demo-people", nil)
			signIn := server.Do(t, http.MethodPost, "/api/auth/sign-in", priyaSignsIn)
			var health struct {
				DemoSignIn bool `json:"demoSignIn"`
			}
			server.Do(t, http.MethodGet, "/api/health", nil).Decode(t, &health)

			if tt.demoSignIn {
				if demoPeople.Status != http.StatusOK || signIn.Status != http.StatusOK {
					t.Errorf("demo people %d, sign-in %d; want 200 and 200", demoPeople.Status, signIn.Status)
				}
			} else {
				for name, response := range map[string]*apitest.Response{"demo people": demoPeople, "sign-in": signIn} {
					if response.Status != http.StatusNotFound || string(response.Body) != notFoundBody {
						t.Errorf("%s = %d %s\nwant 404 %s", name, response.Status, response.Body, notFoundBody)
					}
				}
			}
			if health.DemoSignIn != tt.demoSignIn {
				t.Errorf("health demoSignIn = %v, want %v", health.DemoSignIn, tt.demoSignIn)
			}
		})
	}
}

// A session made while demo sign-in was on is refused once the API runs in production, where no way to
// sign in exists until Shell single sign-on arrives (FR-009).
func TestDemoModeOff_ExistingSessionRefused_FR009(t *testing.T) {
	dir := t.TempDir()
	local := apitest.New(t, apitest.WithDataDir(dir))
	signIn := local.Do(t, http.MethodPost, "/api/auth/sign-in", priyaSignsIn)
	cookie := signIn.Cookie(auth.CookieName)
	if signIn.Status != http.StatusOK || cookie == nil {
		t.Fatalf("sign-in = %d %s, want 200 with a session cookie", signIn.Status, signIn.Body)
	}
	production := apitest.New(t, apitest.WithEnv("production"), apitest.WithDataDir(dir))

	response := production.Do(t, http.MethodGet, "/api/me", nil, apitest.WithCookie(cookie))

	if response.Status != http.StatusUnauthorized || string(response.Body) != signInRequiredBody {
		t.Errorf("production: GET /api/me = %d %s\nwant 401 %s", response.Status, response.Body, signInRequiredBody)
	}
	// The session itself is fine: where demo sign-in is on, it still works.
	if me := local.Do(t, http.MethodGet, "/api/me", nil, apitest.WithCookie(cookie)); me.Status != http.StatusOK {
		t.Errorf("local: GET /api/me = %d %s, want 200", me.Status, me.Body)
	}
}

// Every other path or method under /api/ answers the API's JSON error format, never plain text.
func TestUnknownPathOrMethod_AnswersJSONNotFound(t *testing.T) {
	server := apitest.New(t)
	calls := []struct{ method, path string }{
		{http.MethodGet, "/api/nowhere"},
		{http.MethodGet, "/api/"},
		{http.MethodPost, "/api/health"},
		{http.MethodDelete, "/api/health"},
	}
	for _, call := range calls {
		response := server.Do(t, call.method, call.path, nil, apitest.SkipValidation())

		if response.Status != http.StatusNotFound || string(response.Body) != notFoundBody {
			t.Errorf("%s %s = %d %s\nwant 404 %s", call.method, call.path, response.Status, response.Body, notFoundBody)
		}
	}
}

// A request addressed to another host is refused: protection against DNS rebinding (research R6).
func TestOtherHost_IsRefused(t *testing.T) {
	server := apitest.New(t)

	response := server.Do(t, http.MethodGet, "/api/health", nil, apitest.WithHost("evil.example"))

	if response.Status != http.StatusBadRequest || string(response.Body) != otherSiteBody {
		t.Errorf("answer = %d %s\nwant 400 %s", response.Status, response.Body, otherSiteBody)
	}
}

// Every answer is JSON, and tells the browser not to keep a copy and not to guess the content type
// (research R10).
func TestAnswers_CarryTheAPIHeaders(t *testing.T) {
	server := apitest.New(t)
	answers := map[string]*apitest.Response{
		"health":       server.Do(t, http.MethodGet, "/api/health", nil),
		"unknown path": server.Do(t, http.MethodGet, "/api/nowhere", nil, apitest.SkipValidation()),
		"refused host": server.Do(t, http.MethodGet, "/api/health", nil, apitest.WithHost("evil.example")),
	}
	headers := map[string]string{
		"Content-Type":           "application/json",
		"Cache-Control":          "no-store",
		"X-Content-Type-Options": "nosniff",
	}
	for name, response := range answers {
		for header, want := range headers {
			if got := response.Header.Get(header); got != want {
				t.Errorf("%s: %s = %q, want %q", name, header, got, want)
			}
		}
	}
}

// Routes lists the operations the API serves, which tests compare with the contract; the catch-all for
// unknown paths is not one of them.
func TestNew_RoutesListTheOperations(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	cfg, err := app.LoadConfig(environment(nil))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	fixedClock := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }

	api, err := app.New(cfg, db, fixedClock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if !slices.Contains(api.Routes, "GET /api/health") {
		t.Errorf("Routes = %q, want it to contain GET /api/health", api.Routes)
	}
	if slices.Contains(api.Routes, "/api/") {
		t.Errorf("Routes = %q, want no catch-all", api.Routes)
	}
}
