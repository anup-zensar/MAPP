package app_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pb33f/libopenapi"

	"northstar/api"
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
	routes := newApp(t, "local").Routes

	if !slices.Contains(routes, "GET /api/health") {
		t.Errorf("Routes = %q, want it to contain GET /api/health", routes)
	}
	if slices.Contains(routes, "/api/") {
		t.Errorf("Routes = %q, want no catch-all", routes)
	}
}

// The API serves exactly the operations of its contract, backend/api/openapi.yaml: every operation of the
// contract is served, and every route the API serves is in the contract (research R11). The API runs in
// demo mode, so that demo sign-in and the developer sample records are served too.
//
// Operations of the Plan module that are designed but not built yet are listed in notYetServed and skipped.
func TestRoutesMatchContract(t *testing.T) {
	served := newApp(t, "local").Routes
	var inContract []string
	for _, op := range contractOperations(t) {
		inContract = append(inContract, op.route)
	}

	for _, route := range inContract {
		if !slices.Contains(served, route) && !slices.Contains(notYetServed, route) {
			t.Errorf("the contract's operation %s is not served (and not listed in notYetServed)", route)
		}
	}
	for _, route := range served {
		if !slices.Contains(inContract, route) {
			t.Errorf("the API serves %s, which is not in the contract", route)
		}
	}
}

// Every operation except health and demo sign-in needs a signed-in person, and refuses a request without a
// session with sign_in_required before it reads anything else of the request (FR-005). Operations listed in
// notYetServed are skipped until they are built.
func TestEveryProtectedOperationRequiresSignIn_FR005(t *testing.T) {
	server := apitest.New(t)
	var public []string

	for _, op := range contractOperations(t) {
		if !op.needsSession {
			public = append(public, op.route)
			continue
		}
		if slices.Contains(notYetServed, op.route) {
			continue // not built yet: it would answer 404
		}
		method, path, _ := strings.Cut(op.route, " ")
		path = pathParameter.ReplaceAllString(path, "X-1001") // any value: the sign-in check comes first

		response := server.Do(t, method, path, nil, apitest.SkipRequestValidation()) // no cookie and no body

		if response.Status != http.StatusUnauthorized || string(response.Body) != signInRequiredBody {
			t.Errorf("%s without a session = %d %s\nwant 401 %s", op.route, response.Status, response.Body, signInRequiredBody)
		}
	}
	wantPublic := []string{"GET /api/health", "GET /api/auth/demo-people", "POST /api/auth/sign-in", "POST /api/auth/sign-out"}
	if !slices.Equal(public, wantPublic) {
		t.Errorf("operations without a session = %q, want only %q", public, wantPublic)
	}
}

// The developer sample records exist only where demo sign-in is on (D-036). Elsewhere their routes are
// not served, and they answer 404 like any unknown path.
func TestSampleRecords_OnlyInLocalAndDevelopment_D036(t *testing.T) {
	for env, served := range map[string]bool{"local": true, "development": true, "production": false} {
		t.Run(env, func(t *testing.T) {
			var sampleRoutes []string
			for _, route := range newApp(t, env).Routes {
				if strings.Contains(route, " /api/sample-") {
					sampleRoutes = append(sampleRoutes, route)
				}
			}

			want := 0
			if served {
				want = 6 // the six /api/sample-* operations of the contract
			}
			if len(sampleRoutes) != want {
				t.Errorf("sample routes = %q, want %d", sampleRoutes, want)
			}
		})
	}

	production := apitest.New(t, apitest.WithEnv("production"))
	response := production.Do(t, http.MethodGet, "/api/sample-folders", nil)
	if response.Status != http.StatusNotFound || string(response.Body) != notFoundBody {
		t.Errorf("production: GET /api/sample-folders = %d %s\nwant 404 %s", response.Status, response.Body, notFoundBody)
	}
}

// Settings exist in every environment, not only where demo sign-in is on, and their kinds are registered
// with history, so that a deleted Settings record is in the recycle bin and its delete can be undone
// (FR-006, FR-004).
func TestSettingsKindsRegistered_FR006(t *testing.T) {
	for _, env := range []string{"local", "development", "production"} {
		if !slices.Contains(newApp(t, env).Routes, "DELETE /api/currencies/{code}") {
			t.Errorf("%s: the Settings routes are not served", env)
		}
	}

	server := apitest.New(t)
	alex := server.SignIn(t, "alex.morgan@northstar.demo")
	alex.Do(t, http.MethodPost, "/api/currencies", map[string]string{"code": "JPY", "name": "Japanese yen", "symbol": "¥", "rate": "150"})
	var deleted struct {
		RecycleBinEntry struct{ ID, RecordType, RecordTypeLabel string }
	}
	alex.Do(t, http.MethodDelete, "/api/currencies/JPY", nil).Decode(t, &deleted)
	undone := alex.Do(t, http.MethodPost, "/api/recycle-bin/"+deleted.RecycleBinEntry.ID+"/undo", nil)

	if deleted.RecycleBinEntry.RecordType != "currency" || deleted.RecycleBinEntry.RecordTypeLabel != "currency" {
		t.Errorf("recycle-bin entry = %+v, want a currency", deleted.RecycleBinEntry)
	}
	if undone.Status != http.StatusOK {
		t.Errorf("undo = %d %s, want 200", undone.Status, undone.Body)
	}
}

// notYetServed lists the operations of the contract that are designed but not served yet: the operations of
// the Plan module, features 002-012 and 014-017, merged into backend/api/openapi.yaml before they are built
// (decisions D-192, D-298). The routes are written as the tests format them, "METHOD /path" with the contract's path.
//
// Each feature's backend removes its own operations from this list in the change that serves them.
// TestNotYetServedIsAccurate fails when a listed operation is served or is not in the contract, so the list
// cannot go stale. The list must be empty when the Plan module is done; then remove it and its uses.
var notYetServed = []string{
	// 009 Home dashboard: Recent changes reads 011's activity-log query, and 011 is set aside (D-209)
	"GET /api/home/recent-changes",
	// 011 Activity log and recycle bin
	"GET /api/activity-log/summary",
	"POST /api/recycle-bin/empty",
	"DELETE /api/recycle-bin/{entryId}",
	"GET /api/recycle-bin/{entryId}/restore-preview",
	// 012 Bulk actions and import
	"GET /api/bulk-actions/fields",
	"POST /api/bulk-actions/targets",
	"POST /api/bulk-actions/preview",
	"POST /api/bulk-actions",
	"POST /api/bulk-actions/{bulkActionId}/undo",
	"GET /api/plans/{planId}/imports",
	"POST /api/plans/{planId}/imports",
	"POST /api/plans/{planId}/imports/check",
	"GET /api/plans/{planId}/import-template",
	"GET /api/imports/{importId}",
	"POST /api/imports/{importId}/undo",
	// 014 Access control and global budgets
	"GET /api/access/permissions",
	"GET /api/access/people",
	"GET /api/access/changes",
	"GET /api/roles",
	"POST /api/roles",
	"GET /api/roles/{roleId}",
	"PATCH /api/roles/{roleId}",
	"DELETE /api/roles/{roleId}",
	"GET /api/people/{personId}/roles",
	"PUT /api/people/{personId}/roles",
	"POST /api/people/{personId}/roles/impact",
	"GET /api/access-rules",
	"PATCH /api/access-rules",
	"GET /api/global-budgets",
	"POST /api/global-budgets",
	"GET /api/global-budgets/{globalBudgetId}",
	"PATCH /api/global-budgets/{globalBudgetId}",
	"DELETE /api/global-budgets/{globalBudgetId}",
	"GET /api/global-budgets/{globalBudgetId}/allocation-options",
	"POST /api/global-budgets/{globalBudgetId}/allocations",
	"PATCH /api/budget-allocations/{allocationId}",
	"DELETE /api/budget-allocations/{allocationId}",
	// 015 Financial transactions
	"GET /api/financials/records/{recordId}",
	"GET /api/financials/transactions",
	"GET /api/financials/transactions/{transactionId}",
	"POST /api/financials/transactions/{transactionId}/link",
	"POST /api/financials/transactions/{transactionId}/unlink",
	"GET /api/financials/overview",
	"GET /api/financials/imports",
	"POST /api/financials/imports",
	"POST /api/financials/imports/check",
	"GET /api/financials/imports/template",
	// 016 Planning and currency extensions
	"PUT /api/plans/{planId}/budget-phasing",
	"PUT /api/campaigns/{campaignId}/budget-phasing",
	"PUT /api/initiatives/{initiativeId}/budget-phasing",
	"GET /api/plans/{planId}/budget-by-month",
	"GET /api/activities/{activityId}/split-options",
	"POST /api/activities/{activityId}/split-preview",
	"POST /api/activities/{activityId}/split",
	"GET /api/all-work/breakdown",
	// 017 Approval rules
	"GET /api/approval-rule-options",
	"GET /api/approval-delegations",
	"POST /api/approval-delegations",
	"PATCH /api/approval-delegations/{delegationId}",
	"POST /api/approval-delegations/{delegationId}/cancel",
	"GET /api/approvals/monitoring",
	"POST /api/approvals/escalations/run",
}

// The list of operations not yet served stays accurate: every entry is an operation of the contract, none
// is served already, and none is listed twice (decision D-192).
func TestNotYetServedIsAccurate(t *testing.T) {
	served := newApp(t, "local").Routes
	var inContract []string
	for _, op := range contractOperations(t) {
		inContract = append(inContract, op.route)
	}

	seen := map[string]bool{}
	for _, route := range notYetServed {
		if seen[route] {
			t.Errorf("notYetServed lists %s twice", route)
		}
		seen[route] = true
		if !slices.Contains(inContract, route) {
			t.Errorf("notYetServed lists %s, which is not an operation of the contract", route)
		}
		if slices.Contains(served, route) {
			t.Errorf("notYetServed lists %s, which the API now serves: remove it from the list", route)
		}
	}
}

// newApp builds the API in the environment env on an empty data folder.
func newApp(t *testing.T, env string) *app.App {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	cfg, err := app.LoadConfig(environment(map[string]string{"APP_ENV": env}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	fixedClock := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }

	built, err := app.New(cfg, db, fixedClock)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built
}

// pathParameter finds the parameters of a contract path, such as {folderId}.
var pathParameter = regexp.MustCompile(`\{[^/]+\}`)

// operation is one operation of the contract.
type operation struct {
	route        string // "METHOD /path", the form in which web.Router records its patterns
	needsSession bool
}

// contractOperations returns the operations of the contract, backend/api/openapi.yaml, in its order.
func contractOperations(t *testing.T) []operation {
	t.Helper()
	document, err := libopenapi.NewDocument(api.OpenAPI)
	if err != nil {
		t.Fatalf("reading the contract: %v", err)
	}
	contract, err := document.BuildV3Model()
	if err != nil {
		t.Fatalf("reading the contract: %v", err)
	}

	var operations []operation
	for path, item := range contract.Model.Paths.PathItems.FromOldest() {
		for method, op := range item.GetOperations().FromOldest() {
			operations = append(operations, operation{
				route: strings.ToUpper(method) + " " + path,
				// An operation without a security list of its own needs the contract's default, the session
				// cookie. The public ones say "security: []", which libopenapi reads as an empty list.
				needsSession: op.Security == nil || len(op.Security) > 0,
			})
		}
	}
	return operations
}
