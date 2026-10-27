package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/app"
	"mapp/internal/auth"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/fake"
	"mapp/internal/stores/storestest"
)

const (
	notFoundBody       = `{"error":{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}}`
	otherSiteBody      = `{"error":{"code":"bad_request","message":"This request came from another site and was refused.","fields":[]}}`
	signInRequiredBody = `{"error":{"code":"sign_in_required","message":"Please sign in to continue.","fields":[]}}`
)

var priyaSignsIn = map[string]string{"email": "priya.raman@shell.com", "password": "mapp"}

// In production the demo sign-in routes answer 404 like any path the API does not serve.
func TestDemoMode_RoutesOnlyInLocalAndDevelopment_FR009(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			server := apitest.New(t, apitest.WithEnv(tt.env))

			demoPeople := server.Do(t, http.MethodGet, "/api/v1/auth/demo-people", nil)
			signIn := server.Do(t, http.MethodPost, "/api/v1/auth/sign-in", priyaSignsIn)
			var environment struct {
				Environment string `json:"environment"`
				DemoSignIn  bool   `json:"demoSignIn"`
			}
			server.Do(t, http.MethodGet, "/api/v1/environment", nil).Decode(t, &environment)

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
			if environment.Environment != tt.env || environment.DemoSignIn != tt.demoSignIn {
				t.Errorf("environment = %+v, want %s with demoSignIn %v", environment, tt.env, tt.demoSignIn)
			}
		})
	}
}

// No way to sign in exists in production until Shell single sign-on arrives.
func TestDemoModeOff_ExistingSessionRefused_FR009(t *testing.T) {
	t.Parallel()
	db := apitest.DemoDB(t, false)
	local := apitest.New(t, apitest.WithDB(db))
	signIn := local.Do(t, http.MethodPost, "/api/v1/auth/sign-in", priyaSignsIn)
	cookie := signIn.Cookie(auth.CookieName)
	if signIn.Status != http.StatusOK || cookie == nil {
		t.Fatalf("sign-in = %d %s, want 200 with a session cookie", signIn.Status, signIn.Body)
	}
	production := apitest.New(t, apitest.WithEnv("production"), apitest.WithDB(db))

	response := production.Do(t, http.MethodGet, "/api/v1/me", nil, apitest.WithCookie(cookie))

	if response.Status != http.StatusUnauthorized || string(response.Body) != signInRequiredBody {
		t.Errorf("production: GET /api/v1/me = %d %s\nwant 401 %s", response.Status, response.Body, signInRequiredBody)
	}
	// The session itself is fine: where demo sign-in is on, it still works.
	if me := local.Do(t, http.MethodGet, "/api/v1/me", nil, apitest.WithCookie(cookie)); me.Status != http.StatusOK {
		t.Errorf("local: GET /api/v1/me = %d %s, want 200", me.Status, me.Body)
	}
}

// The unversioned paths of before /api/v1 answer the JSON error too.
func TestUnknownPathOrMethod_AnswersJSONNotFound_022_FR001(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	calls := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/nowhere"},
		{http.MethodGet, "/api/v1/"},
		{http.MethodPost, "/api/v1/environment"},
		{http.MethodGet, "/api/health"},
		{http.MethodGet, "/api/plans"},
		{http.MethodGet, "/api/"},
	}
	for _, call := range calls {
		response := server.Do(t, call.method, call.path, nil, apitest.SkipValidation())

		if response.Status != http.StatusNotFound || string(response.Body) != notFoundBody {
			t.Errorf("%s %s = %d %s\nwant 404 %s", call.method, call.path, response.Status, response.Body, notFoundBody)
		}
	}
}

// Protection against DNS rebinding (research R6).
func TestOtherHost_IsRefused(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	response := server.Do(t, http.MethodGet, "/api/v1/environment", nil, apitest.WithHost("evil.example"))

	if response.Status != http.StatusBadRequest || string(response.Body) != otherSiteBody {
		t.Errorf("answer = %d %s\nwant 400 %s", response.Status, response.Body, otherSiteBody)
	}
}

// The browser keeps no copy and does not guess the content type (research R10).
func TestAnswers_CarryTheAPIHeaders(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	answers := map[string]*apitest.Response{
		"environment":  server.Do(t, http.MethodGet, "/api/v1/environment", nil),
		"unknown path": server.Do(t, http.MethodGet, "/api/v1/nowhere", nil, apitest.SkipValidation()),
		"refused host": server.Do(t, http.MethodGet, "/api/v1/environment", nil, apitest.WithHost("evil.example")),
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

func TestSampleRecords_OnlyInLocalAndDevelopment_D036(t *testing.T) {
	t.Parallel()
	for env, served := range map[string]bool{"local": true, "development": true, "production": false} {
		t.Run(env, func(t *testing.T) {
			var sampleRoutes []string
			for _, route := range newApp(t, env).Routes {
				if strings.Contains(route, " /api/v1/sample-") {
					sampleRoutes = append(sampleRoutes, route)
				}
			}

			want := 0
			if served {
				want = 6 // the six /api/v1/sample-* operations of the contract
			}
			if len(sampleRoutes) != want {
				t.Errorf("sample routes = %q, want %d", sampleRoutes, want)
			}
		})
	}

	production := apitest.New(t, apitest.WithEnv("production"))
	response := production.Do(t, http.MethodGet, "/api/v1/sample-folders", nil)
	if response.Status != http.StatusNotFound || string(response.Body) != notFoundBody {
		t.Errorf("production: GET /api/v1/sample-folders = %d %s\nwant 404 %s", response.Status, response.Body, notFoundBody)
	}
}

// A deleted Settings record is in the recycle bin in every environment, and its delete can be undone.
func TestSettingsKindsRegistered_FR006(t *testing.T) {
	t.Parallel()
	for _, env := range []string{"local", "development", "production"} {
		if !slices.Contains(newApp(t, env).Routes, "DELETE /api/v1/currencies/{code}") {
			t.Errorf("%s: the Settings routes are not served", env)
		}
	}

	server := apitest.New(t)
	alex := server.SignIn(t, "alex.morgan@shell.com")
	alex.Do(t, http.MethodPost, "/api/v1/currencies", map[string]string{"code": "JPY", "name": "Japanese yen", "symbol": "¥", "rate": "150"})
	var deleted struct {
		RecycleBinEntry struct{ ID, RecordType, RecordTypeLabel string }
	}
	alex.Do(t, http.MethodDelete, "/api/v1/currencies/JPY", nil).Decode(t, &deleted)
	undone := alex.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)

	if deleted.RecycleBinEntry.RecordType != "currency" || deleted.RecycleBinEntry.RecordTypeLabel != "currency" {
		t.Errorf("recycle-bin entry = %+v, want a currency", deleted.RecycleBinEntry)
	}
	if undone.Status != http.StatusOK {
		t.Errorf("undo = %d %s, want 200", undone.Status, undone.Body)
	}
}

func newApp(t *testing.T, env string) *app.App {
	t.Helper()
	cfg, err := app.LoadConfig(environment(map[string]string{"APP_ENV": env}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	fixedClock := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }

	built, err := app.New(cfg, storestest.New(fake.NewDB()), fixedClock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built
}
