package app_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"mapp/internal/access"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/authz"
)

// lubricantsRecords names a record of the Lubricants Europe plan for each path parameter of a planning
// record (seed: FY2026 Lubricants Europe and its first campaign, initiative, activity and line item).
var lubricantsRecords = map[string]string{
	"{planId}": "P-1001", "{campaignId}": "C-1001", "{initiativeId}": "I-1001", "{activityId}": "A-1001",
	"{lineItemId}": "L-1001", "{recordId}": "A-1001",
}

// matrixPath fills the path parameters of route: planning records with Lubricants records, any other with
// an ID that does not exist (the access middleware refuses before anything is read).
func matrixPath(route string) (method, path string, planning bool) {
	method, path, _ = strings.Cut(route, " ")
	for parameter, id := range lubricantsRecords {
		if strings.Contains(path, parameter) {
			path, planning = strings.ReplaceAll(path, parameter, id), true
		}
	}
	return method, pathParameter.ReplaceAllString(path, "X-1001"), planning
}

// heldPermissions returns the permissions the signed-in person holds in any scope, from GET /api/v1/me.
func heldPermissions(t *testing.T, client *apitest.Client) []authz.Permission {
	t.Helper()
	var me struct {
		Access struct {
			Permissions []struct{ Key authz.Permission }
		}
	}
	client.Do(t, http.MethodGet, "/api/v1/me", nil).Decode(t, &me)
	var held []authz.Permission
	for _, p := range me.Access.Permissions {
		held = append(held, p.Key)
	}
	return held
}

// Every operation of plan Table C refuses the people its rule does not allow, and every refusal is an
// answer of the contract (SC-001, FR-009, FR-010, 022 FR-005):
//
//   - a Need or InPlan operation refuses a person who holds its permission nowhere — Sara (Viewer) for every
//     change of planning records, Priya (Planner) for the administration operations — with 403 and message 2;
//   - a View or InPlan operation on a Lubricants record answers Mei, who plans for Mobility China only, with
//     404 not_found, as for a record that does not exist.
//
// Operations that are not served yet (notYetServed: 012 and the operations of later features) are skipped;
// their rules are checked by TestRouteRules_EveryOperation_SC001 and apply as soon as they are served.
func TestAccessMatrix_EveryOperationRefusesWhomItsRuleExcludes_SC001(t *testing.T) {
	server := apitest.New(t)
	served := newApp(t, "local").Routes
	people := []struct {
		name  string
		email string
	}{{"Sara", sarasEmail}, {"Priya", priyasEmail}, {"Mei", meisEmail}}
	checked := map[string]int{}

	for _, person := range people {
		client := server.SignIn(t, person.email)
		held := heldPermissions(t, client)
		roles := map[string]string{"Sara": "your role (Viewer)", "Priya": "your role (Planner)", "Mei": "your role (Planner)"}[person.name]

		for _, op := range contractOperations(t) {
			if !op.needsSession || !slices.Contains(served, op.route) {
				continue
			}
			rule, found := access.RuleFor(op.route)
			if !found {
				t.Errorf("%s has no access rule", op.route)
				continue
			}
			method, path, planning := matrixPath(op.route)
			var want string
			switch {
			case (rule.Kind == access.Need || rule.Kind == access.InPlan) && !slices.Contains(held, rule.Permission):
				info, _ := authz.Lookup(string(rule.Permission))
				want = info.Noun + " isn't part of " + roles + ". Ask an administrator if you need it."
			case person.name == "Mei" && planning && (rule.Kind == access.View || rule.Kind == access.InPlan):
				want = "" // not found, with the record type of the path
			default:
				continue // allowed by the rule; the service's own checks are tested by each feature
			}

			response := client.Do(t, method, path, nil, apitest.SkipRequestValidation())
			var answer errorAnswer
			response.Decode(t, &answer)
			e := answer.Error
			switch {
			case want == "" && (response.Status != http.StatusNotFound || e.Code != "not_found" ||
				!strings.HasSuffix(e.Message, " no longer exists — someone may have deleted it.")):
				t.Errorf("%s: %s %s = %d %s, want 404 not_found", person.name, method, path, response.Status, response.Body)
			case want != "" && (response.Status != http.StatusForbidden || e.Code != "forbidden" || e.Message != want):
				t.Errorf("%s: %s %s = %d %s\nwant 403 forbidden %q", person.name, method, path, response.Status, response.Body, want)
			}
			checked[person.name]++
		}
	}

	// The matrix is not empty by mistake: Sara is refused every change of planning records and every
	// administration operation; Priya every administration operation; Mei every operation on a Lubricants record.
	for name, least := range map[string]int{"Sara": 90, "Priya": 60, "Mei": 40} {
		if checked[name] < least {
			t.Errorf("%s: %d operations checked, want at least %d", name, checked[name], least)
		}
	}
}
