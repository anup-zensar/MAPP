package access

import "mapp/internal/platform/authz"

func shellRoutes() map[string]Rule {
	return map[string]Rule{
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
	}
}

func peopleRoutes() map[string]Rule {
	return map[string]Rule{
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
	}
}

func accessRoutes() map[string]Rule {
	return map[string]Rule{
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
	}
}
