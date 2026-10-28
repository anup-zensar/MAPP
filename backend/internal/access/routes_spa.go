package access

// spaRoutes are feature 013's: the service reads the plan in scope and needs finance.view for financial figures.
func spaRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/plans/{planId}/spa": view(),
	}
}
