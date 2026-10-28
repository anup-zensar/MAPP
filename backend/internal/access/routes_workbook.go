package access

import "mapp/internal/platform/authz"

// workbookRoutes are feature 012's operations added after bulkRoutes.
func workbookRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/plans/{planId}/export": inPlan(authz.DataExport),
	}
}
