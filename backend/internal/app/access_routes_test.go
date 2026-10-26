package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/access"
	"mapp/internal/platform/apitest"
)

// Every operation of the contract has its access rule in access/routes.go — also the operations of 012 and
// 015–017 that are not served yet, so they are protected the moment they are (D-301) — every served route
// has one, and the table names no operation that is not in the contract (SC-001, FR-009, FR-010).
func TestEveryOperationHasAnAccessRule_SC001(t *testing.T) {
	rules := access.RulePatterns()
	var inContract []string
	for _, op := range contractOperations(t) {
		inContract = append(inContract, op.route)
		if !slices.Contains(rules, op.route) {
			t.Errorf("the contract's operation %s has no access rule in access/routes.go", op.route)
		}
	}
	for _, route := range newApp(t, "local").Routes {
		if _, known := access.RuleFor(route); !known {
			t.Errorf("the API serves %s without an access rule", route)
		}
	}
	for _, pattern := range rules {
		if !slices.Contains(inContract, pattern) {
			t.Errorf("access/routes.go has a rule for %s, which is not an operation of the contract", pattern)
		}
	}
}

// The public operations are exactly the six the contract serves without a session, and only they (FR-010).
func TestPublicRules_OnlyTheSixWithoutSession_FR010(t *testing.T) {
	var public []string
	for _, op := range contractOperations(t) {
		rule, _ := access.RuleFor(op.route)
		if (rule.Kind == access.Public) != !op.needsSession {
			t.Errorf("%s: rule %s, needs a session %v", op.route, rule.Kind, op.needsSession)
		}
		if rule.Kind == access.Public {
			public = append(public, op.route)
		}
		if (rule.Kind == access.Need || rule.Kind == access.InPlan) && rule.Permission == "" {
			t.Errorf("%s: a %s rule without a permission", op.route, rule.Kind)
		}
	}
	want := []string{"GET /healthz", "GET /readyz", "GET /api/v1/environment", "GET /api/v1/auth/demo-people",
		"POST /api/v1/auth/sign-in", "POST /api/v1/auth/sign-out"}
	if !slices.Equal(public, want) {
		t.Errorf("public operations = %q, want %q", public, want)
	}
}

// Every served operation that needs a permission refuses a signed-in person whose roles do not include it
// with 403 forbidden and the role message, before reading anything else of the request — the middleware is
// on every route (FR-009, FR-010, D-228). Sara Novak is a Viewer: she holds plans.view, data.export and
// finance.view only.
func TestEveryNeedRule_RefusesWithoutThePermission_FR009_FR010(t *testing.T) {
	server := apitest.New(t)
	sara := server.SignIn(t, "sara.novak@shell.com")
	viewer := []string{"plans.view", "data.export", "finance.view"}
	checked := 0
	for _, op := range contractOperations(t) {
		rule, _ := access.RuleFor(op.route)
		if rule.Kind != access.Need && rule.Kind != access.InPlan || slices.Contains(viewer, string(rule.Permission)) ||
			slices.Contains(notYetServed, op.route) {
			continue
		}
		method, path, _ := strings.Cut(op.route, " ")
		path = pathParameter.ReplaceAllString(path, "X-1001") // any value: the role check comes first

		response := sara.Do(t, method, path, nil, apitest.SkipRequestValidation())

		if response.Status != http.StatusForbidden || !strings.Contains(string(response.Body), `"code":"forbidden"`) ||
			!strings.Contains(string(response.Body), "isn't part of your role (Viewer). Ask an administrator if you need it.") {
			t.Errorf("%s as a Viewer = %d %s, want 403 forbidden with the role message", op.route, response.Status, response.Body)
		}
		checked++
	}
	if checked < 80 {
		t.Errorf("checked %d operations, want the 80 or more that need a permission a Viewer lacks", checked)
	}
}
