package access

import (
	"maps"
	"slices"

	"mapp/internal/platform/authz"
)

// RuleKind (plan Table C): Need needs the permission anywhere, InPlan also in the record's plan (the service
// checks), View and Custom leave the check to the service, Public needs no sign-in.
type RuleKind string

const (
	Public   RuleKind = "public"
	SignedIn RuleKind = "signedIn"
	Need     RuleKind = "need"
	InPlan   RuleKind = "inPlan"
	View     RuleKind = "view"
	Custom   RuleKind = "custom"
)

// Rule's Permission is needed anywhere by a View rule; a Custom rule's only documents its service's check.
type Rule struct {
	Kind       RuleKind
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

// routeRules is plan Table C, keyed as the contract writes the operation; an operation without a rule is refused.
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
	simulatedGsapRoutes(),
	purchaseRequestRoutes(),
	spaRoutes(),
)

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

func RuleFor(operation string) (Rule, bool) {
	rule, found := routeRules[operation]
	return rule, found
}

func RulePatterns() []string {
	return slices.Sorted(maps.Keys(routeRules))
}
