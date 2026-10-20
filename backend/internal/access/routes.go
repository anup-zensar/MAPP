package access

import (
	"maps"
	"slices"

	"northstar/internal/platform/authz"
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
	Kind       RuleKind
	Permission authz.Permission // for Need and InPlan; for View and Custom, the permission the service also checks, if any
}

func public() Rule                       { return Rule{Kind: Public} }
func signedIn() Rule                     { return Rule{Kind: SignedIn} }
func need(p authz.Permission) Rule       { return Rule{Kind: Need, Permission: p} }
func inPlan(p authz.Permission) Rule     { return Rule{Kind: InPlan, Permission: p} }
func view() Rule                         { return Rule{Kind: View} }
func viewAnd(p authz.Permission) Rule    { return Rule{Kind: View, Permission: p} }
func custom() Rule                       { return Rule{Kind: Custom} }
func customWith(p authz.Permission) Rule { return Rule{Kind: Custom, Permission: p} }

// routeRules is plan Table C: the rule of every operation of the contract, by its ServeMux pattern
// ("METHOD /path"), including the operations of features 012 and 015–017 that are not served yet, so that
// they are protected the moment they are (D-301). An operation without a rule is refused (fail closed,
// FR-010); app/access_routes_test.go fails for any operation of the contract without one.
var routeRules = map[string]Rule{
	// 001 shell, sign-in, profile and the developer pages
	"GET /api/health":                           public(),
	"GET /api/auth/demo-people":                 public(),
	"POST /api/auth/sign-in":                    public(),
	"POST /api/auth/sign-out":                   public(),
	"GET /api/me":                               signedIn(),
	"PATCH /api/me":                             signedIn(),
	"PATCH /api/me/preferences":                 signedIn(),
	"GET /api/me/display-currencies":            signedIn(),
	"GET /api/sample-folders":                   signedIn(),
	"POST /api/sample-folders":                  signedIn(),
	"PATCH /api/sample-folders/{folderId}":      signedIn(),
	"DELETE /api/sample-folders/{folderId}":     signedIn(),
	"POST /api/sample-folders/{folderId}/notes": signedIn(),
	"DELETE /api/sample-notes/{noteId}":         signedIn(),

	// 002 Settings: readable by everyone (forms need the lists, D-240), changed with settings.manage;
	// currencies with exchangeRates.manage.
	"GET /api/settings/overview":                                    signedIn(),
	"GET /api/models":                                               signedIn(),
	"POST /api/models":                                              need(authz.SettingsManage),
	"GET /api/models/{modelId}":                                     signedIn(),
	"PATCH /api/models/{modelId}":                                   need(authz.SettingsManage),
	"DELETE /api/models/{modelId}":                                  need(authz.SettingsManage),
	"POST /api/models/{modelId}/impact":                             need(authz.SettingsManage),
	"POST /api/models/{modelId}/status":                             need(authz.SettingsManage),
	"GET /api/activity-types":                                       signedIn(),
	"POST /api/activity-types":                                      need(authz.SettingsManage),
	"GET /api/activity-types/{activityTypeId}":                      signedIn(),
	"PATCH /api/activity-types/{activityTypeId}":                    need(authz.SettingsManage),
	"DELETE /api/activity-types/{activityTypeId}":                   need(authz.SettingsManage),
	"POST /api/activity-types/{activityTypeId}/impact":              need(authz.SettingsManage),
	"POST /api/activity-types/{activityTypeId}/status":              need(authz.SettingsManage),
	"GET /api/shell-attributes":                                     signedIn(),
	"POST /api/shell-attributes":                                    need(authz.SettingsManage),
	"GET /api/shell-attributes/{attributeId}":                       signedIn(),
	"PATCH /api/shell-attributes/{attributeId}":                     need(authz.SettingsManage),
	"DELETE /api/shell-attributes/{attributeId}":                    need(authz.SettingsManage),
	"POST /api/shell-attributes/{attributeId}/impact":               need(authz.SettingsManage),
	"POST /api/shell-attributes/{attributeId}/status":               need(authz.SettingsManage),
	"POST /api/shell-attributes/{attributeId}/move":                 need(authz.SettingsManage),
	"GET /api/shell-attributes/{attributeId}/options":               signedIn(),
	"POST /api/shell-attributes/{attributeId}/options":              need(authz.SettingsManage),
	"POST /api/shell-attributes/{attributeId}/options/import-check": need(authz.SettingsManage),
	"POST /api/shell-attributes/{attributeId}/options/import":       need(authz.SettingsManage),
	"GET /api/shell-attribute-options/{optionId}":                   signedIn(),
	"PATCH /api/shell-attribute-options/{optionId}":                 need(authz.SettingsManage),
	"DELETE /api/shell-attribute-options/{optionId}":                need(authz.SettingsManage),
	"POST /api/shell-attribute-options/{optionId}/status":           need(authz.SettingsManage),
	"POST /api/shell-attribute-options/{optionId}/move":             need(authz.SettingsManage),
	"GET /api/shell-attribute-forms/{level}":                        signedIn(),
	"POST /api/shell-attribute-forms/{level}/check":                 signedIn(),
	"GET /api/currencies":                                           signedIn(),
	"POST /api/currencies":                                          need(authz.ExchangeRatesManage),
	"GET /api/currencies/{code}":                                    signedIn(),
	"PATCH /api/currencies/{code}":                                  need(authz.ExchangeRatesManage),
	"DELETE /api/currencies/{code}":                                 need(authz.ExchangeRatesManage),
	"POST /api/currencies/{code}/impact":                            need(authz.ExchangeRatesManage),
	"POST /api/currencies/{code}/status":                            need(authz.ExchangeRatesManage),
	"GET /api/finance-budgets":                                      signedIn(),
	"PATCH /api/finance-budgets/{budgetId}":                         need(authz.SettingsManage),
	"POST /api/finance-budget-items":                                need(authz.SettingsManage),
	"GET /api/finance-budget-items/{itemId}":                        signedIn(),
	"PATCH /api/finance-budget-items/{itemId}":                      need(authz.SettingsManage),
	"DELETE /api/finance-budget-items/{itemId}":                     need(authz.SettingsManage),
	"POST /api/finance-budget-items/{itemId}/status":                need(authz.SettingsManage),

	// 003 People and teams: pickers need everyone; changes need people.manage (+ the last-administrator
	// guard on a person's status, FR-021).
	"GET /api/people":                    signedIn(),
	"POST /api/people":                   need(authz.PeopleManage),
	"GET /api/people/{personId}":         signedIn(),
	"PATCH /api/people/{personId}":       need(authz.PeopleManage),
	"POST /api/people/{personId}/impact": need(authz.PeopleManage),
	"POST /api/people/{personId}/status": need(authz.PeopleManage),
	"GET /api/teams":                     signedIn(),
	"POST /api/teams":                    need(authz.PeopleManage),
	"GET /api/teams/{teamId}":            signedIn(),
	"PATCH /api/teams/{teamId}":          need(authz.PeopleManage),
	"DELETE /api/teams/{teamId}":         need(authz.PeopleManage),
	"POST /api/teams/{teamId}/impact":    need(authz.PeopleManage),
	"POST /api/teams/{teamId}/status":    need(authz.PeopleManage),

	// 004 Plans
	"GET /api/plan-defaults":                  need(authz.PlansCreate),
	"GET /api/plans":                          signedIn(), // filtered to the plans the person can see
	"POST /api/plans":                         inPlan(authz.PlansCreate),
	"GET /api/plans/{planId}":                 view(),
	"PATCH /api/plans/{planId}":               inPlan(authz.PlansEdit),
	"DELETE /api/plans/{planId}":              inPlan(authz.PlansDelete),
	"POST /api/plans/{planId}/publish":        inPlan(authz.PlansPublish),
	"POST /api/plans/{planId}/unpublish":      inPlan(authz.PlansPublish),
	"GET /api/plans/{planId}/copy-options":    viewAnd(authz.PlansCreate),
	"POST /api/plans/{planId}/copy":           inPlan(authz.PlansCreate),
	"POST /api/plans/{planId}/change-effects": inPlan(authz.PlansEdit),
	"GET /api/plans/{planId}/campaigns":       view(),

	// 005 Campaigns and initiatives
	"POST /api/plans/{planId}/campaigns":               inPlan(authz.CampaignsEdit),
	"GET /api/campaigns/{campaignId}":                  view(),
	"PATCH /api/campaigns/{campaignId}":                inPlan(authz.CampaignsEdit),
	"DELETE /api/campaigns/{campaignId}":               inPlan(authz.CampaignsDelete),
	"GET /api/campaigns/{campaignId}/destinations":     viewAnd(authz.CampaignsEdit), // filtered to destinations with campaigns.edit
	"POST /api/campaigns/{campaignId}/move":            inPlan(authz.CampaignsEdit),
	"POST /api/campaigns/{campaignId}/copy":            inPlan(authz.CampaignsEdit),
	"POST /api/campaigns/{campaignId}/initiatives":     inPlan(authz.CampaignsEdit),
	"GET /api/initiatives/{initiativeId}":              view(),
	"PATCH /api/initiatives/{initiativeId}":            inPlan(authz.CampaignsEdit),
	"DELETE /api/initiatives/{initiativeId}":           inPlan(authz.CampaignsDelete),
	"GET /api/initiatives/{initiativeId}/destinations": viewAnd(authz.CampaignsEdit),
	"POST /api/initiatives/{initiativeId}/move":        inPlan(authz.CampaignsEdit),
	"POST /api/initiatives/{initiativeId}/copy":        inPlan(authz.CampaignsEdit),

	// 006 Activities
	"GET /api/initiatives/{initiativeId}/activity-form": inPlan(authz.ActivitiesEdit),
	"POST /api/initiatives/{initiativeId}/activities":   inPlan(authz.ActivitiesEdit),
	"GET /api/plans/{planId}/activities":                view(),
	"GET /api/activities/{activityId}":                  view(), // the plan, the owner or a named approver
	"PATCH /api/activities/{activityId}":                inPlan(authz.ActivitiesEdit),
	"DELETE /api/activities/{activityId}":               inPlan(authz.ActivitiesDelete),
	"GET /api/activities/{activityId}/form":             inPlan(authz.ActivitiesEdit),
	"GET /api/activities/{activityId}/destinations":     viewAnd(authz.ActivitiesEdit), // filtered to destinations with activities.edit
	"POST /api/activities/{activityId}/move":            inPlan(authz.ActivitiesEdit),
	"POST /api/activities/{activityId}/copy":            inPlan(authz.ActivitiesEdit),

	// 007 Line items
	"GET /api/activities/{activityId}/line-items":              view(),
	"POST /api/activities/{activityId}/line-items":             inPlan(authz.ActivitiesEdit),
	"GET /api/activities/{activityId}/line-items/form-options": inPlan(authz.ActivitiesEdit),
	"GET /api/line-items/{lineItemId}":                         view(),
	"PATCH /api/line-items/{lineItemId}":                       inPlan(authz.ActivitiesEdit),
	"DELETE /api/line-items/{lineItemId}":                      inPlan(authz.ActivitiesDelete),
	"POST /api/line-items/{lineItemId}/duplicate":              inPlan(authz.ActivitiesEdit),

	// 008 Approvals and notifications
	"GET /api/approval-workflows":                          signedIn(),
	"POST /api/approval-workflows":                         need(authz.WorkflowsManage),
	"GET /api/approval-workflows/{workflowId}":             signedIn(),
	"PATCH /api/approval-workflows/{workflowId}":           need(authz.WorkflowsManage),
	"DELETE /api/approval-workflows/{workflowId}":          need(authz.WorkflowsManage),
	"POST /api/approval-workflows/{workflowId}/deactivate": need(authz.WorkflowsManage),
	"POST /api/approval-workflows/{workflowId}/activate":   need(authz.WorkflowsManage),
	"GET /api/activities/{activityId}/approval":            view(),
	"GET /api/activities/{activityId}/approval/submission": inPlan(authz.ActivitiesSubmit),
	"POST /api/activities/{activityId}/approval/submit":    inPlan(authz.ActivitiesSubmit),
	"POST /api/activities/{activityId}/approval/approve":   custom(), // check order of plan "Approve/decline"
	"POST /api/activities/{activityId}/approval/decline":   custom(),
	"POST /api/activities/{activityId}/approval/recall":    custom(),   // owner or submitter with activities.submit, or on behalf
	"GET /api/approvals":                                   signedIn(), // filtered
	"GET /api/approvals/summary":                           signedIn(), // filtered
	"GET /api/approvals/activities/{activityId}":           view(),
	"GET /api/notifications":                               signedIn(), // own
	"POST /api/notifications/{notificationId}/read":        signedIn(), // own
	"POST /api/notifications/read-all":                     signedIn(), // own

	// 009 Home: filtered to what the person can see
	"GET /api/home/summary":        signedIn(),
	"GET /api/home/plans":          signedIn(),
	"GET /api/home/attention":      signedIn(),
	"GET /api/home/waiting":        signedIn(),
	"GET /api/home/my-activities":  signedIn(),
	"GET /api/home/recent-changes": signedIn(),

	// 010 All work: filtered; export needs data.export
	"GET /api/all-work":                    signedIn(),
	"GET /api/all-work/options":            signedIn(),
	"GET /api/all-work/export":             inPlan(authz.DataExport),
	"GET /api/all-work/records/{recordId}": signedIn(), // a record the person cannot see answers 403
	"GET /api/all-work/preferences":        signedIn(), // own
	"PUT /api/all-work/preferences":        signedIn(), // own
	"GET /api/saved-views":                 signedIn(), // own
	"POST /api/saved-views":                signedIn(), // own
	"PATCH /api/saved-views/{viewId}":      signedIn(), // own
	"DELETE /api/saved-views/{viewId}":     signedIn(), // own

	// 011 Activity log and recycle bin (D-238)
	"GET /api/activity-log":                          signedIn(), // filtered
	"GET /api/activity-log/summary":                  signedIn(), // filtered
	"GET /api/recycle-bin":                           signedIn(), // filtered, rows answer access
	"POST /api/recycle-bin/{entryId}/undo":           custom(),   // history.Access.CanRestore
	"POST /api/recycle-bin/{entryId}/restore":        custom(),
	"GET /api/recycle-bin/{entryId}/restore-preview": custom(),
	"DELETE /api/recycle-bin/{entryId}":              need(authz.RecycleBinPurge),
	"POST /api/recycle-bin/empty":                    need(authz.RecycleBinPurge),

	// 012 Bulk actions and import (deferred; declared so the table is complete when they are built)
	"GET /api/bulk-actions/fields":               signedIn(),
	"POST /api/bulk-actions/targets":             signedIn(), // filtered
	"POST /api/bulk-actions/preview":             custom(),   // each target like its single operation
	"POST /api/bulk-actions":                     custom(),
	"POST /api/bulk-actions/{bulkActionId}/undo": custom(),
	"GET /api/plans/{planId}/imports":            view(),
	"POST /api/plans/{planId}/imports":           inPlan(authz.ActivitiesEdit),
	"POST /api/plans/{planId}/imports/check":     inPlan(authz.ActivitiesEdit),
	"GET /api/plans/{planId}/import-template":    inPlan(authz.ActivitiesEdit),
	"GET /api/imports/{importId}":                view(),
	"POST /api/imports/{importId}/undo":          inPlan(authz.ActivitiesDelete),

	// 014 Access control and global budgets
	"GET /api/access/permissions":                                 need(authz.AccessManage),
	"GET /api/access/people":                                      need(authz.AccessManage),
	"GET /api/access/changes":                                     need(authz.AccessManage),
	"GET /api/roles":                                              need(authz.AccessManage),
	"POST /api/roles":                                             need(authz.AccessManage),
	"GET /api/roles/{roleId}":                                     need(authz.AccessManage),
	"PATCH /api/roles/{roleId}":                                   need(authz.AccessManage),
	"DELETE /api/roles/{roleId}":                                  need(authz.AccessManage),
	"GET /api/people/{personId}/roles":                            need(authz.AccessManage),
	"PUT /api/people/{personId}/roles":                            need(authz.AccessManage),
	"POST /api/people/{personId}/roles/impact":                    need(authz.AccessManage),
	"GET /api/access-rules":                                       need(authz.AccessManage),
	"PATCH /api/access-rules":                                     need(authz.AccessManage),
	"GET /api/global-budgets":                                     need(authz.GlobalBudgetsView),   // rows filtered (FR-033)
	"POST /api/global-budgets":                                    need(authz.GlobalBudgetsManage), // the service also needs All markets
	"GET /api/global-budgets/{globalBudgetId}":                    need(authz.GlobalBudgetsView),
	"PATCH /api/global-budgets/{globalBudgetId}":                  need(authz.GlobalBudgetsManage),
	"DELETE /api/global-budgets/{globalBudgetId}":                 need(authz.GlobalBudgetsManage),
	"GET /api/global-budgets/{globalBudgetId}/allocation-options": need(authz.GlobalBudgetsManage),
	"POST /api/global-budgets/{globalBudgetId}/allocations":       inPlan(authz.GlobalBudgetsManage),
	"PATCH /api/budget-allocations/{allocationId}":                inPlan(authz.GlobalBudgetsManage),
	"DELETE /api/budget-allocations/{allocationId}":               inPlan(authz.GlobalBudgetsManage),

	// 015 Financial transactions (consolidation D-301)
	"GET /api/financials/records/{recordId}":                   inPlan(authz.FinanceView), // View of the record + finance.view in its plan
	"GET /api/financials/transactions":                         need(authz.FinanceView),   // filtered
	"GET /api/financials/transactions/{transactionId}":         need(authz.FinanceView),
	"POST /api/financials/transactions/{transactionId}/link":   inPlan(authz.FinanceLink),
	"POST /api/financials/transactions/{transactionId}/unlink": inPlan(authz.FinanceLink),
	"GET /api/financials/overview":                             need(authz.FinanceView), // filtered
	"GET /api/financials/imports":                              need(authz.FinanceView),
	"POST /api/financials/imports":                             need(authz.FinanceImport),
	"POST /api/financials/imports/check":                       need(authz.FinanceImport),
	"GET /api/financials/imports/template":                     need(authz.FinanceImport),

	// 016 Planning and currency extensions (D-301)
	"GET /api/currencies/{code}/rates":                   signedIn(),
	"POST /api/currencies/{code}/rates":                  need(authz.ExchangeRatesManage),
	"POST /api/currencies/{code}/rates/impact":           need(authz.ExchangeRatesManage),
	"PATCH /api/exchange-rates/{rateId}":                 need(authz.ExchangeRatesManage),
	"DELETE /api/exchange-rates/{rateId}":                need(authz.ExchangeRatesManage),
	"POST /api/exchange-rates/{rateId}/impact":           need(authz.ExchangeRatesManage),
	"GET /api/exchange-rates/current":                    signedIn(),
	"POST /api/exchange-rates/import-check":              need(authz.ExchangeRatesManage),
	"POST /api/exchange-rates/import":                    need(authz.ExchangeRatesManage),
	"GET /api/exchange-rates/imports":                    signedIn(),
	"GET /api/exchange-rates/import-sample":              need(authz.ExchangeRatesManage),
	"GET /api/field-help":                                signedIn(),
	"PATCH /api/field-help/{helpKey}":                    need(authz.SettingsManage),
	"PUT /api/plans/{planId}/budget-phasing":             inPlan(authz.PlansEdit),
	"PUT /api/campaigns/{campaignId}/budget-phasing":     inPlan(authz.CampaignsEdit),
	"PUT /api/initiatives/{initiativeId}/budget-phasing": inPlan(authz.CampaignsEdit),
	"GET /api/plans/{planId}/budget-by-month":            view(),
	"GET /api/activities/{activityId}/split-options":     inPlan(authz.ActivitiesEdit),
	"POST /api/activities/{activityId}/split-preview":    inPlan(authz.ActivitiesEdit),
	"POST /api/activities/{activityId}/split":            inPlan(authz.ActivitiesEdit),
	"GET /api/all-work/breakdown":                        signedIn(), // filtered; with plan: View of that plan

	// 017 Approval rules (D-301)
	"GET /api/approval-rule-options":                       signedIn(),
	"GET /api/approval-delegations":                        customWith(authz.WorkflowsManage), // scope=all needs workflows.manage
	"POST /api/approval-delegations":                       customWith(authz.WorkflowsManage), // for someone else
	"PATCH /api/approval-delegations/{delegationId}":       customWith(authz.WorkflowsManage), // the delegating person, or workflows.manage
	"POST /api/approval-delegations/{delegationId}/cancel": customWith(authz.WorkflowsManage),
	"GET /api/approvals/monitoring":                        signedIn(), // filtered
	"POST /api/approvals/escalations/run":                  need(authz.WorkflowsManage),
}

// RuleFor returns the rule of the operation with the ServeMux pattern ("GET /api/plans/{planId}"), and false
// when the operation has none: the middleware then refuses it.
func RuleFor(pattern string) (Rule, bool) {
	rule, found := routeRules[pattern]
	return rule, found
}

// RulePatterns returns the pattern of every operation with a rule, sorted: for the test that compares them
// with the contract and the served routes (SC-001).
func RulePatterns() []string {
	return slices.Sorted(maps.Keys(routeRules))
}
