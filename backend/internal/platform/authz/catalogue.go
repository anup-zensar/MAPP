// Package authz is MAPP's access policy (feature 014): the fixed list of permissions, what one signed-in
// person holds (Grants) and the checks every service calls before it shows or changes a record.
//
// Permissions are fixed here, in code: they are the actions MAPP offers. Roles, which group permissions,
// are data that administrators change in Settings (package access, decision D-223). The access middleware
// builds the Grants of the signed-in person once per request and puts them in the request's context; the
// services then ask:
//
//	if err := authz.RequireIn(ctx, authz.PlansEdit, plans.AccessFacts(p)); err != nil {
//		return err // 403 forbidden, with a message that says what to do
//	}
//
// A context without Grants (the seed, internal jobs, service unit tests) is not restricted (D-228): every
// HTTP request passes the middleware, which a test proves.
//
// The package imports only actor and errs, so every business package can use it.
package authz

import "slices"

// Permission is one thing a person may do, such as "plans.edit" (spec Table A).
type Permission string

// The 25 permissions of spec Table A, in its order.
const (
	PlansView               Permission = "plans.view"
	PlansCreate             Permission = "plans.create"
	PlansEdit               Permission = "plans.edit"
	PlansPublish            Permission = "plans.publish"
	PlansDelete             Permission = "plans.delete"
	CampaignsEdit           Permission = "campaigns.edit"
	CampaignsDelete         Permission = "campaigns.delete"
	ActivitiesEdit          Permission = "activities.edit"
	ActivitiesDelete        Permission = "activities.delete"
	ActivitiesSubmit        Permission = "activities.submit"
	DataExport              Permission = "data.export"
	FinanceView             Permission = "finance.view"
	FinanceLink             Permission = "finance.link"
	FinanceImport           Permission = "finance.import"
	ApprovalsDecide         Permission = "approvals.decide"
	ApprovalsDecideOnBehalf Permission = "approvals.decideOnBehalf"
	GlobalBudgetsView       Permission = "globalBudgets.view"
	GlobalBudgetsManage     Permission = "globalBudgets.manage"
	SettingsManage          Permission = "settings.manage"
	ExchangeRatesManage     Permission = "exchangeRates.manage"
	WorkflowsManage         Permission = "workflows.manage"
	PeopleManage            Permission = "people.manage"
	AccessManage            Permission = "access.manage"
	ActivityLogViewAll      Permission = "activityLog.viewAll"
	RecycleBinPurge         Permission = "recycleBin.purge"
)

// The groups of Table A, in its order: the role page shows the permissions under these headings.
const (
	GroupPlanning       = "Plans and planning"
	GroupFinance        = "Financial data"
	GroupApprovals      = "Approvals"
	GroupGlobalBudgets  = "Global budgets"
	GroupAdministration = "Administration"
)

// Groups lists the groups of Table A in order.
var Groups = []string{GroupPlanning, GroupFinance, GroupApprovals, GroupGlobalBudgets, GroupAdministration}

// PermissionInfo describes one permission: its group and label on the role page, whether it applies only
// to plans in the role's scope, and the noun that starts its refusal message ("<Noun> isn't part of your
// role …").
type PermissionInfo struct {
	Key    Permission
	Group  string
	Label  string
	Noun   string
	Scoped bool
}

// catalogue is spec Table A, word for word.
var catalogue = []PermissionInfo{
	{PlansView, GroupPlanning, "View plans and everything in them", "Viewing this plan", true},
	{PlansCreate, GroupPlanning, "Create and copy plans", "Creating plans", true},
	{PlansEdit, GroupPlanning, "Edit plan details", "Editing plans", true},
	{PlansPublish, GroupPlanning, "Publish and unpublish plans", "Publishing plans", true},
	{PlansDelete, GroupPlanning, "Delete plans", "Deleting plans", true},
	{CampaignsEdit, GroupPlanning, "Add, edit, move and copy campaigns and initiatives", "Changing campaigns and initiatives", true},
	{CampaignsDelete, GroupPlanning, "Delete campaigns and initiatives", "Deleting campaigns and initiatives", true},
	{ActivitiesEdit, GroupPlanning, "Add, edit, move and copy activities and line items", "Changing activities and line items", true},
	{ActivitiesDelete, GroupPlanning, "Delete activities and line items", "Deleting activities and line items", true},
	{ActivitiesSubmit, GroupPlanning, "Submit activities for approval and recall them", "Submitting activities", true},
	{DataExport, GroupPlanning, "Export from All work", "Exporting", true},
	{FinanceView, GroupFinance, "See financial data", "Seeing financial data", true},
	{FinanceLink, GroupFinance, "Link financial transactions to line items", "Linking financial transactions", true},
	{FinanceImport, GroupFinance, "Import GSAP extracts", "Importing GSAP extracts", false},
	{ApprovalsDecide, GroupApprovals, "Approve or decline when named as an approver", "Approving activities", false},
	{ApprovalsDecideOnBehalf, GroupApprovals, "Approve or decline on behalf of the named approvers", "Deciding on behalf of other approvers", true},
	{GlobalBudgetsView, GroupGlobalBudgets, "See global budgets and their allocations", "Seeing global budgets", true},
	{GlobalBudgetsManage, GroupGlobalBudgets, "Create global budgets and allocate them to plans", "Managing global budgets", true},
	{SettingsManage, GroupAdministration, "Change Settings lists (models, activity types, Shell attributes, finance budget items, field help)", "Changing Settings", false},
	{ExchangeRatesManage, GroupAdministration, "Add currencies, change exchange rates and import rate files", "Changing currencies and exchange rates", false},
	{WorkflowsManage, GroupAdministration, "Create and change approval workflows", "Changing approval workflows", false},
	{PeopleManage, GroupAdministration, "Add and change people and teams", "Changing people and teams", false},
	{AccessManage, GroupAdministration, "Give roles and change roles and access rules", "Changing roles and access", false},
	{ActivityLogViewAll, GroupAdministration, "See the whole activity log, including Settings, people and access changes", "Seeing the whole activity log", false},
	{RecycleBinPurge, GroupAdministration, "Delete permanently and empty the recycle bin", "Deleting permanently", false},
}

// Catalogue returns every permission in Table A order. GET /api/v1/access/permissions serves it.
func Catalogue() []PermissionInfo {
	return slices.Clone(catalogue)
}

// Lookup returns the permission with key, and false when there is no such permission.
func Lookup(key string) (PermissionInfo, bool) {
	i := slices.IndexFunc(catalogue, func(p PermissionInfo) bool { return string(p.Key) == key })
	if i < 0 {
		return PermissionInfo{}, false
	}
	return catalogue[i], true
}

// All returns the key of every permission, in Table A order: what the Administrator role always holds.
func All() []Permission {
	keys := make([]Permission, 0, len(catalogue))
	for _, p := range catalogue {
		keys = append(keys, p.Key)
	}
	return keys
}

// InOrder returns the known permissions among keys, without repeats, in Table A order. Never nil.
func InOrder(keys []Permission) []Permission {
	ordered := []Permission{}
	for _, p := range catalogue {
		if slices.Contains(keys, p.Key) {
			ordered = append(ordered, p.Key)
		}
	}
	return ordered
}

// info returns what the catalogue says about p; an unknown permission (a programming mistake) is shown by
// its key and treated as scoped.
func info(p Permission) PermissionInfo {
	if i, found := Lookup(string(p)); found {
		return i
	}
	return PermissionInfo{Key: p, Label: string(p), Noun: string(p), Scoped: true}
}
