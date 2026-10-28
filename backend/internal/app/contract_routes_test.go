package app_test

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"

	"mapp/api"
	"mapp/internal/platform/apitest"
)

// Demo mode, so demo sign-in and the sample records are served too; notYetServed lists what is not built.
func TestRoutesMatchContract(t *testing.T) {
	t.Parallel()
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

func TestEveryProtectedOperationRequiresSignIn_FR005(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	var public []string

	for _, op := range contractOperations(t) {
		if !op.needsSession {
			public = append(public, op.route)
			continue
		}
		if slices.Contains(notYetServed, op.route) {
			continue
		}
		method, path, _ := strings.Cut(op.route, " ")
		path = pathParameter.ReplaceAllString(path, "X-1001") // any value: the sign-in check comes first

		response := server.Do(t, method, path, nil, apitest.SkipRequestValidation())

		if response.Status != http.StatusUnauthorized || string(response.Body) != signInRequiredBody {
			t.Errorf("%s without a session = %d %s\nwant 401 %s", op.route, response.Status, response.Body, signInRequiredBody)
		}
	}
	wantPublic := []string{"GET /healthz", "GET /readyz", "GET /api/v1/environment", "GET /api/v1/auth/demo-people",
		"POST /api/v1/auth/sign-in", "POST /api/v1/auth/sign-out"}
	if !slices.Equal(public, wantPublic) {
		t.Errorf("operations without a session = %q, want only %q", public, wantPublic)
	}
}

// notYetServed: the contract's operations designed but not built yet (D-192, D-298). Each feature removes
// its own in the change that serves them; the list must be empty when the Plan module is done.
var notYetServed = []string{}

func TestNotYetServedIsAccurate(t *testing.T) {
	t.Parallel()
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

var pathParameter = regexp.MustCompile(`\{[^/]+\}`)

type operation struct {
	route        string
	needsSession bool
}

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
