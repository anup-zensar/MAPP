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
// FR-010); app/access_routes_test.go fails for any operation of the contract without one.
var routeRules = map[string]Rule{
	// 001 shell, sign-in, profile and the developer pages
	"GET /healthz":                                 public(),
	"GET /readyz":                                  public(),
	"GET /api/v1/environment":                      public(),
	"GET /api/v1/auth/demo-people":                 public(),
	"POST /api/v1/auth/sign-in":                    public(),
	"POST /api/v1/auth/sign-out":                   public(),
	"GET /api/v1/me":                               signedIn(),
	"PATCH /api/v1/me":                             signedIn(),
	"PATCH /api/v1/me/preferences":                 signedIn(),
	"GET /api/v1/me/display-currencies":            signedIn(),
	"GET /api/v1/sample-folders":                   signedIn(),
	"POST /api/v1/sample-folders":                  signedIn(),
	"PATCH /api/v1/sample-folders/{folderId}":      signedIn(),
	"DELETE /api/v1/sample-folders/{folderId}":     signedIn(),
	"POST /api/v1/sample-folders/{folderId}/notes": signedIn(),
	"DELETE /api/v1/sample-notes/{noteId}":         signedIn(),

	// 002 Settings: readable by everyone (forms need the lists, D-240), changed with settings.manage;
	// currencies with exchangeRates.manage.
	"GET /api/v1/settings/overview":                                    signedIn(),
	"GET /api/v1/models":                                               signedIn(),
	"POST /api/v1/models":                                              need(authz.SettingsManage),
	"GET /api/v1/models/{modelId}":                                     signedIn(),
	"PATCH /api/v1/models/{modelId}":                                   need(authz.SettingsManage),
	"DELETE /api/v1/models/{modelId}":                                  need(authz.SettingsManage),
	"POST /api/v1/models/{modelId}/impact":                             need(authz.SettingsManage),
	"POST /api/v1/models/{modelId}/status":                             need(authz.SettingsManage),
	"GET /api/v1/activity-types":                                       signedIn(),
	"POST /api/v1/activity-types":                                      need(authz.SettingsManage),
	"GET /api/v1/activity-types/{activityTypeId}":                      signedIn(),
	"PATCH /api/v1/activity-types/{activityTypeId}":                    need(authz.SettingsManage),
	"DELETE /api/v1/activity-types/{activityTypeId}":                   need(authz.SettingsManage),
	"POST /api/v1/activity-types/{activityTypeId}/impact":              need(authz.SettingsManage),
	"POST /api/v1/activity-types/{activityTypeId}/status":              need(authz.SettingsManage),
	"GET /api/v1/shell-attributes":                                     signedIn(),
	"POST /api/v1/shell-attributes":                                    need(authz.SettingsManage),
	"GET /api/v1/shell-attributes/{attributeId}":                       signedIn(),
	"PATCH /api/v1/shell-attributes/{attributeId}":                     need(authz.SettingsManage),
	"DELETE /api/v1/shell-attributes/{attributeId}":                    need(authz.SettingsManage),
	"POST /api/v1/shell-attributes/{attributeId}/impact":               need(authz.SettingsManage),
	"POST /api/v1/shell-attributes/{attributeId}/status":               need(authz.SettingsManage),
	"POST /api/v1/shell-attributes/{attributeId}/move":                 need(authz.SettingsManage),
	"GET /api/v1/shell-attributes/{attributeId}/options":               signedIn(),
	"POST /api/v1/shell-attributes/{attributeId}/options":              need(authz.SettingsManage),
	"POST /api/v1/shell-attributes/{attributeId}/options/import-check": need(authz.SettingsManage),
	"POST /api/v1/shell-attributes/{attributeId}/options/import":       need(authz.SettingsManage),
	"GET /api/v1/shell-attribute-options/{optionId}":                   signedIn(),
	"PATCH /api/v1/shell-attribute-options/{optionId}":                 need(authz.SettingsManage),
	"DELETE /api/v1/shell-attribute-options/{optionId}":                need(authz.SettingsManage),
	"POST /api/v1/shell-attribute-options/{optionId}/status":           need(authz.SettingsManage),
	"POST /api/v1/shell-attribute-options/{optionId}/move":             need(authz.SettingsManage),
	"GET /api/v1/shell-attribute-forms/{level}":                        signedIn(),
	"POST /api/v1/shell-attribute-forms/{level}/check":                 signedIn(),
	"GET /api/v1/currencies":                                           signedIn(),
	"POST /api/v1/currencies":                                          need(authz.ExchangeRatesManage),
	"GET /api/v1/currencies/{code}":                                    signedIn(),
	"PATCH /api/v1/currencies/{code}":                                  need(authz.ExchangeRatesManage),
	"DELETE /api/v1/currencies/{code}":                                 need(authz.ExchangeRatesManage),
	"POST /api/v1/currencies/{code}/impact":                            need(authz.ExchangeRatesManage),
	"POST /api/v1/currencies/{code}/status":                            need(authz.ExchangeRatesManage),
	"GET /api/v1/finance-budgets":                                      signedIn(),
	"PATCH /api/v1/finance-budgets/{budgetId}":                         need(authz.SettingsManage),
	"POST /api/v1/finance-budget-items":                                need(authz.SettingsManage),
	"GET /api/v1/finance-budget-items/{itemId}":                        signedIn(),
	"PATCH /api/v1/finance-budget-items/{itemId}":                      need(authz.SettingsManage),
	"DELETE /api/v1/finance-budget-items/{itemId}":                     need(authz.SettingsManage),
	"POST /api/v1/finance-budget-items/{itemId}/status":                need(authz.SettingsManage),

	// 003 People and teams: pickers need everyone; changes need people.manage (+ the last-administrator
	// guard on a person's status, FR-021).
	"GET /api/v1/people":                    signedIn(),
	"POST /api/v1/people":                   need(authz.PeopleManage),
	"GET /api/v1/people/{personId}":         signedIn(),
	"PATCH /api/v1/people/{personId}":       need(authz.PeopleManage),
	"POST /api/v1/people/{personId}/impact": need(authz.PeopleManage),
	"POST /api/v1/people/{personId}/status": need(authz.PeopleManage),
	"GET /api/v1/teams":                     signedIn(),
	"POST /api/v1/teams":                    need(authz.PeopleManage),
	"GET /api/v1/teams/{teamId}":            signedIn(),
	"PATCH /api/v1/teams/{teamId}":          need(authz.PeopleManage),
	"DELETE /api/v1/teams/{teamId}":         need(authz.PeopleManage),
	"POST /api/v1/teams/{teamId}/impact":    need(authz.PeopleManage),
	"POST /api/v1/teams/{teamId}/status":    need(authz.PeopleManage),

	// 004 Plans
	"GET /api/v1/plan-defaults":                  need(authz.PlansCreate),
	"GET /api/v1/plans":                          signedIn(), // filtered to the plans the person can see
	"POST /api/v1/plans":                         inPlan(authz.PlansCreate),
	"GET /api/v1/plans/{planId}":                 view(),
	"PATCH /api/v1/plans/{planId}":               inPlan(authz.PlansEdit),
	"DELETE /api/v1/plans/{planId}":              inPlan(authz.PlansDelete),
	"POST /api/v1/plans/{planId}/publish":        inPlan(authz.PlansPublish),
	"POST /api/v1/plans/{planId}/unpublish":      inPlan(authz.PlansPublish),
	"GET /api/v1/plans/{planId}/copy-options":    viewAnd(authz.PlansCreate),
	"POST /api/v1/plans/{planId}/copy":           viewAnd(authz.PlansCreate), // the service: plans.create for the copy's teams
	"POST /api/v1/plans/{planId}/change-effects": inPlan(authz.PlansEdit),
	"GET /api/v1/plans/{planId}/campaigns":       view(),

	// 005 Campaigns and initiatives
	"POST /api/v1/plans/{planId}/campaigns":               inPlan(authz.CampaignsEdit),
	"GET /api/v1/campaigns/{campaignId}":                  view(),
	"PATCH /api/v1/campaigns/{campaignId}":                inPlan(authz.CampaignsEdit),
	"DELETE /api/v1/campaigns/{campaignId}":               inPlan(authz.CampaignsDelete),
	"GET /api/v1/campaigns/{campaignId}/destinations":     viewAnd(authz.CampaignsEdit), // filtered to destinations with campaigns.edit
	"POST /api/v1/campaigns/{campaignId}/move":            inPlan(authz.CampaignsEdit),
	"POST /api/v1/campaigns/{campaignId}/copy":            viewAnd(authz.CampaignsEdit), // the service: at the destination
	"POST /api/v1/campaigns/{campaignId}/initiatives":     inPlan(authz.CampaignsEdit),
	"GET /api/v1/initiatives/{initiativeId}":              view(),
	"PATCH /api/v1/initiatives/{initiativeId}":            inPlan(authz.CampaignsEdit),
	"DELETE /api/v1/initiatives/{initiativeId}":           inPlan(authz.CampaignsDelete),
	"GET /api/v1/initiatives/{initiativeId}/destinations": inPlan(authz.CampaignsEdit), // campaigns of the same plan
	"POST /api/v1/initiatives/{initiativeId}/move":        inPlan(authz.CampaignsEdit),
	"POST /api/v1/initiatives/{initiativeId}/copy":        inPlan(authz.CampaignsEdit), // into a campaign of the same plan

	// 006 Activities
	"GET /api/v1/initiatives/{initiativeId}/activity-form": inPlan(authz.ActivitiesEdit),
	"POST /api/v1/initiatives/{initiativeId}/activities":   inPlan(authz.ActivitiesEdit),
	"GET /api/v1/plans/{planId}/activities":                view(),
	"GET /api/v1/activities/{activityId}":                  view(), // the plan, the owner or a named approver
	"PATCH /api/v1/activities/{activityId}":                inPlan(authz.ActivitiesEdit),
	"DELETE /api/v1/activities/{activityId}":               inPlan(authz.ActivitiesDelete),
	"GET /api/v1/activities/{activityId}/form":             inPlan(authz.ActivitiesEdit),
	"GET /api/v1/activities/{activityId}/destinations":     inPlan(authz.ActivitiesEdit), // initiatives of the same plan
	"POST /api/v1/activities/{activityId}/move":            inPlan(authz.ActivitiesEdit),
	"POST /api/v1/activities/{activityId}/copy":            inPlan(authz.ActivitiesEdit), // into an initiative of the same plan

	// 007 Line items
	"GET /api/v1/activities/{activityId}/line-items":              view(),
	"POST /api/v1/activities/{activityId}/line-items":             inPlan(authz.ActivitiesEdit),
	"GET /api/v1/activities/{activityId}/line-items/form-options": inPlan(authz.ActivitiesEdit),
	"GET /api/v1/line-items/{lineItemId}":                         view(),
	"PATCH /api/v1/line-items/{lineItemId}":                       inPlan(authz.ActivitiesEdit),
	"DELETE /api/v1/line-items/{lineItemId}":                      inPlan(authz.ActivitiesDelete),
	"POST /api/v1/line-items/{lineItemId}/duplicate":              inPlan(authz.ActivitiesEdit),

	// 008 Approvals and notifications
	"GET /api/v1/approval-workflows":                          signedIn(),
	"POST /api/v1/approval-workflows":                         need(authz.WorkflowsManage),
	"GET /api/v1/approval-workflows/{workflowId}":             signedIn(),
	"PATCH /api/v1/approval-workflows/{workflowId}":           need(authz.WorkflowsManage),
	"DELETE /api/v1/approval-workflows/{workflowId}":          need(authz.WorkflowsManage),
	"POST /api/v1/approval-workflows/{workflowId}/deactivate": need(authz.WorkflowsManage),
	"POST /api/v1/approval-workflows/{workflowId}/activate":   need(authz.WorkflowsManage),
	"GET /api/v1/activities/{activityId}/approval":            view(),
	"GET /api/v1/activities/{activityId}/approval/submission": inPlan(authz.ActivitiesSubmit),
	"POST /api/v1/activities/{activityId}/approval/submit":    inPlan(authz.ActivitiesSubmit),
	"POST /api/v1/activities/{activityId}/approval/approve":   custom(), // check order of plan "Approve/decline"
	"POST /api/v1/activities/{activityId}/approval/decline":   custom(),
	"POST /api/v1/activities/{activityId}/approval/recall":    custom(),   // owner or submitter with activities.submit, or on behalf
	"GET /api/v1/approvals":                                   signedIn(), // filtered
	"GET /api/v1/approvals/summary":                           signedIn(), // filtered
	"GET /api/v1/approvals/activities/{activityId}":           view(),
	"GET /api/v1/notifications":                               signedIn(), // own
	"POST /api/v1/notifications/{notificationId}/read":        signedIn(), // own
	"POST /api/v1/notifications/read-all":                     signedIn(), // own

	// 009 Home: filtered to what the person can see
	"GET /api/v1/home/summary":        signedIn(),
	"GET /api/v1/home/plans":          signedIn(),
	"GET /api/v1/home/attention":      signedIn(),
	"GET /api/v1/home/waiting":        signedIn(),
	"GET /api/v1/home/my-activities":  signedIn(),
	"GET /api/v1/home/recent-changes": signedIn(),

	// 010 All work: filtered; export needs data.export
	"GET /api/v1/all-work":                    signedIn(),
	"GET /api/v1/all-work/options":            signedIn(),
	"GET /api/v1/all-work/export":             inPlan(authz.DataExport),
	"GET /api/v1/all-work/records/{recordId}": view(),     // a record the person cannot see answers 403
	"GET /api/v1/all-work/preferences":        signedIn(), // own
	"PUT /api/v1/all-work/preferences":        signedIn(), // own
	"GET /api/v1/saved-views":                 signedIn(), // own
	"POST /api/v1/saved-views":                signedIn(), // own
	"PATCH /api/v1/saved-views/{viewId}":      signedIn(), // own
	"DELETE /api/v1/saved-views/{viewId}":     signedIn(), // own

	// 011 Activity log and recycle bin (D-238)
	"GET /api/v1/activity-log":                          signedIn(), // filtered
	"GET /api/v1/activity-log/summary":                  signedIn(), // filtered
	"GET /api/v1/recycle-bin":                           signedIn(), // filtered, rows answer access
	"POST /api/v1/recycle-bin/{entryId}/undo":           custom(),   // history.Access.CanRestore
	"POST /api/v1/recycle-bin/{entryId}/restore":        custom(),
	"GET /api/v1/recycle-bin/{entryId}/restore-preview": custom(),
	"DELETE /api/v1/recycle-bin/{entryId}":              need(authz.RecycleBinPurge),
	"POST /api/v1/recycle-bin/empty":                    need(authz.RecycleBinPurge),

	// 012 Bulk actions and import (deferred; declared so the table is complete when they are built)
	"GET /api/v1/bulk-actions/fields":               signedIn(),
	"POST /api/v1/bulk-actions/targets":             signedIn(), // filtered
	"POST /api/v1/bulk-actions/preview":             custom(),   // each target like its single operation
	"POST /api/v1/bulk-actions":                     custom(),
	"POST /api/v1/bulk-actions/{bulkActionId}/undo": custom(),
	"GET /api/v1/plans/{planId}/imports":            view(),
	"POST /api/v1/plans/{planId}/imports":           inPlan(authz.ActivitiesEdit),
	"POST /api/v1/plans/{planId}/imports/check":     inPlan(authz.ActivitiesEdit),
	"GET /api/v1/plans/{planId}/import-template":    inPlan(authz.ActivitiesEdit),
	"GET /api/v1/imports/{importId}":                view(),
	"POST /api/v1/imports/{importId}/undo":          inPlan(authz.ActivitiesDelete),

	// 014 Access control and global budgets
	"GET /api/v1/access/permissions":                                 need(authz.AccessManage),
	"GET /api/v1/access/people":                                      need(authz.AccessManage),
	"GET /api/v1/access/changes":                                     need(authz.AccessManage),
	"GET /api/v1/roles":                                              need(authz.AccessManage),
	"POST /api/v1/roles":                                             need(authz.AccessManage),
	"GET /api/v1/roles/{roleId}":                                     need(authz.AccessManage),
	"PATCH /api/v1/roles/{roleId}":                                   need(authz.AccessManage),
	"DELETE /api/v1/roles/{roleId}":                                  need(authz.AccessManage),
	"GET /api/v1/people/{personId}/roles":                            need(authz.AccessManage),
	"PUT /api/v1/people/{personId}/roles":                            need(authz.AccessManage),
	"POST /api/v1/people/{personId}/roles/impact":                    need(authz.AccessManage),
	"GET /api/v1/access-rules":                                       need(authz.AccessManage),
	"PATCH /api/v1/access-rules":                                     need(authz.AccessManage),
	"GET /api/v1/global-budgets":                                     need(authz.GlobalBudgetsView),   // rows filtered (FR-033)
	"POST /api/v1/global-budgets":                                    need(authz.GlobalBudgetsManage), // the service also needs All markets
	"GET /api/v1/global-budgets/{globalBudgetId}":                    need(authz.GlobalBudgetsView),
	"PATCH /api/v1/global-budgets/{globalBudgetId}":                  need(authz.GlobalBudgetsManage),
	"DELETE /api/v1/global-budgets/{globalBudgetId}":                 need(authz.GlobalBudgetsManage),
	"GET /api/v1/global-budgets/{globalBudgetId}/allocation-options": need(authz.GlobalBudgetsManage),
	"POST /api/v1/global-budgets/{globalBudgetId}/allocations":       inPlan(authz.GlobalBudgetsManage),
	"PATCH /api/v1/budget-allocations/{allocationId}":                inPlan(authz.GlobalBudgetsManage),
	"DELETE /api/v1/budget-allocations/{allocationId}":               inPlan(authz.GlobalBudgetsManage),

	// 015 Financial transactions (consolidation D-301)
	"GET /api/v1/financials/records/{recordId}":                   inPlan(authz.FinanceView), // View of the record + finance.view in its plan
	"GET /api/v1/financials/transactions":                         need(authz.FinanceView),   // filtered
	"GET /api/v1/financials/transactions/{transactionId}":         need(authz.FinanceView),
	"POST /api/v1/financials/transactions/{transactionId}/link":   inPlan(authz.FinanceLink),
	"POST /api/v1/financials/transactions/{transactionId}/unlink": inPlan(authz.FinanceLink),
	"GET /api/v1/financials/overview":                             need(authz.FinanceView), // filtered
	"GET /api/v1/financials/imports":                              need(authz.FinanceView),
	"POST /api/v1/financials/imports":                             need(authz.FinanceImport),
	"POST /api/v1/financials/imports/check":                       need(authz.FinanceImport),
	"GET /api/v1/financials/imports/template":                     need(authz.FinanceImport),

	// 016 Planning and currency extensions (D-301)
	"GET /api/v1/currencies/{code}/rates":                   signedIn(),
	"POST /api/v1/currencies/{code}/rates":                  need(authz.ExchangeRatesManage),
	"POST /api/v1/currencies/{code}/rates/impact":           need(authz.ExchangeRatesManage),
	"PATCH /api/v1/exchange-rates/{rateId}":                 need(authz.ExchangeRatesManage),
	"DELETE /api/v1/exchange-rates/{rateId}":                need(authz.ExchangeRatesManage),
	"POST /api/v1/exchange-rates/{rateId}/impact":           need(authz.ExchangeRatesManage),
	"GET /api/v1/exchange-rates/current":                    signedIn(),
	"POST /api/v1/exchange-rates/import-check":              need(authz.ExchangeRatesManage),
	"POST /api/v1/exchange-rates/import":                    need(authz.ExchangeRatesManage),
	"GET /api/v1/exchange-rates/imports":                    signedIn(),
	"GET /api/v1/exchange-rates/import-sample":              need(authz.ExchangeRatesManage),
	"GET /api/v1/field-help":                                signedIn(),
	"PATCH /api/v1/field-help/{helpKey}":                    need(authz.SettingsManage),
	"PUT /api/v1/plans/{planId}/budget-phasing":             inPlan(authz.PlansEdit),
	"PUT /api/v1/campaigns/{campaignId}/budget-phasing":     inPlan(authz.CampaignsEdit),
	"PUT /api/v1/initiatives/{initiativeId}/budget-phasing": inPlan(authz.CampaignsEdit),
	"GET /api/v1/plans/{planId}/budget-by-month":            view(),
	"GET /api/v1/activities/{activityId}/split-options":     inPlan(authz.ActivitiesEdit),
	"POST /api/v1/activities/{activityId}/split-preview":    inPlan(authz.ActivitiesEdit),
	"POST /api/v1/activities/{activityId}/split":            inPlan(authz.ActivitiesEdit),
	"GET /api/v1/all-work/breakdown":                        signedIn(), // filtered; with plan: View of that plan

	// 017 Approval rules (D-301)
	"GET /api/v1/approval-rule-options":                       signedIn(),
	"GET /api/v1/approval-delegations":                        customWith(authz.WorkflowsManage), // scope=all needs workflows.manage
	"POST /api/v1/approval-delegations":                       customWith(authz.WorkflowsManage), // for someone else
	"PATCH /api/v1/approval-delegations/{delegationId}":       customWith(authz.WorkflowsManage), // the delegating person, or workflows.manage
	"POST /api/v1/approval-delegations/{delegationId}/cancel": customWith(authz.WorkflowsManage),
	"GET /api/v1/approvals/monitoring":                        signedIn(), // filtered
	"POST /api/v1/approvals/escalations/run":                  need(authz.WorkflowsManage),
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
