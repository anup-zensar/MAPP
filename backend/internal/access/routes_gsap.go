package access

import "mapp/internal/platform/authz"

func simulatedGsapRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/simulated-gsap/swbs-runs":  need(authz.FinanceImport),
		"POST /api/v1/simulated-gsap/swbs-runs": need(authz.FinanceImport),
	}
}
