package access

import "mapp/internal/platform/authz"

// dashboardRoutes are the rules of feature 036 (D-343): the views count only what the person may see.
func dashboardRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/dashboards/executive":        need(authz.PlansView),
		"GET /api/v1/dashboards/regional":         need(authz.PlansView),
		"GET /api/v1/dashboards/executive/export": need(authz.DataExport),
		"GET /api/v1/dashboards/regional/export":  need(authz.DataExport),
		"GET /api/v1/plans/{planId}/export":       inPlan(authz.DataExport),
	}
}
