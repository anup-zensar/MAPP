package access

import (
	"maps"
	"slices"

	"mapp/internal/platform/authz"
)

// RuleKind says how the access middleware treats an operation (plan Table C).
type RuleKind string

// The rule kinds of plan Table C.
const (
	// Public operations need no sign-in; they never pass the middleware.
	Public RuleKind = "public"
	// SignedIn operations are open to every signed-in person; their answers are filtered to what the person
	// can see where the service says so (FR-007).
	SignedIn RuleKind = "signedIn"
	// Need operations need the permission in any scope; the middleware refuses them otherwise.
	Need RuleKind = "need"
	// InPlan operations need the permission for the record's plan: the middleware refuses a person who does
	// not hold it anywhere, and the service checks the plan's scope (authz.RequireIn).
	InPlan RuleKind = "inPlan"
	// View operations read one record: the service checks that the person can see it (authz.RequireView).
	View RuleKind = "view"
	// Custom operations have rules of their own, checked by the service (approve, recall, restore, …).
	Custom RuleKind = "custom"
)

// Rule is the access rule of one API operation.
type Rule struct {
	Kind RuleKind
	// Permission is what Need and InPlan operations need. A View operation with a permission (viewAnd) needs
	// it in any scope, and the service checks where it applies: the destination of a copy, the plans a
	// destination list offers. A Custom operation's permission only documents what its service checks.
	Permission authz.Permission
}

func public() Rule                       { return Rule{Kind: Public} }
func signedIn() Rule                     { return Rule{Kind: SignedIn} }
func need(p authz.Permission) Rule       { return Rule{Kind: Need, Permission: p} }
func inPlan(p authz.Permission) Rule     { return Rule{Kind: InPlan, Permission: p} }
func view() Rule                         { return Rule{Kind: View} }
func viewAnd(p authz.Permission) Rule    { return Rule{Kind: View, Permission: p} }
func custom() Rule                       { return Rule{Kind: Custom} }
func customWith(p authz.Permission) Rule { return Rule{Kind: Custom, Permission: p} }

// routeRules is plan Table C: the rule of every operation of the contract, keyed as the contract writes it
// ("METHOD /path"; web.Operation builds it from the matched route), including the operations of features 012 and 015–017 that are not served yet, so that
// they are protected the moment they are (D-301). An operation without a rule is refused (fail closed,
// FR-010); app/access_routes_test.go fails for any operation of the contract without one. Each area of the API
// keeps its part in a routes_<area>.go file.
var routeRules = joinRules( //nolint:gochecknoglobals // the route table: built once when the program starts, then only read
	shellRoutes(),
	settingsRoutes(),
	peopleRoutes(),
	planRoutes(),
	campaignRoutes(),
	activityRoutes(),
	lineItemRoutes(),
	approvalRoutes(),
	homeRoutes(),
	allWorkRoutes(),
	historyRoutes(),
	bulkRoutes(),
	accessRoutes(),
	financeRoutes(),
	extensionRoutes(),
	approvalRuleRoutes(),
)

// joinRules gathers the tables of the areas into one. An operation in two of them, which one map literal would
// not compile, stops the program when it starts.
func joinRules(areas ...map[string]Rule) map[string]Rule {
	table := map[string]Rule{}
	for _, area := range areas {
		for operation, rule := range area {
			if _, twice := table[operation]; twice {
				panic("access: two rules for " + operation)
			}
			table[operation] = rule
		}
	}
	return table
}

// RuleFor returns the rule of the operation ("GET /api/v1/plans/{planId}"), and false when the operation has
// none: the middleware then refuses it.
func RuleFor(operation string) (Rule, bool) {
	rule, found := routeRules[operation]
	return rule, found
}

// RulePatterns returns the pattern of every operation with a rule, sorted: for the test that compares them
// with the contract and the served routes (SC-001).
func RulePatterns() []string {
	return slices.Sorted(maps.Keys(routeRules))
}
