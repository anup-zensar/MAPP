package access

import "mapp/internal/platform/authz"

// purchaseRequestRoutes are the rules of features 032 and 033 (their contracts/routes.md).
func purchaseRequestRoutes() map[string]Rule {
	const pr = "/api/v1/purchase-requests/{purchaseRequestId}"
	return map[string]Rule{
		"GET /api/v1/purchase-requests":                   need(authz.PurchaseRequestsList),
		"POST /api/v1/purchase-requests":                  inPlan(authz.PurchaseRequestsEdit),
		"GET /api/v1/purchase-requests/new-defaults":      inPlan(authz.PurchaseRequestsEdit),
		"GET /api/v1/purchase-requests/line-item-options": inPlan(authz.PurchaseRequestsEdit),
		"GET " + pr:                                view(),
		"PUT " + pr:                                inPlan(authz.PurchaseRequestsEdit),
		"DELETE " + pr:                             inPlan(authz.PurchaseRequestsDelete),
		"GET " + pr + "/close-preview":             inPlan(authz.PurchaseRequestsEdit),
		"POST " + pr + "/close":                    inPlan(authz.PurchaseRequestsEdit),
		"POST " + pr + "/resend":                   inPlan(authz.PurchaseRequestsEdit),
		"GET " + pr + "/routings/{routingId}/form": view(),
		"GET " + pr + "/approval":                  view(),
		"GET " + pr + "/approval/submission":       inPlan(authz.PurchaseRequestsSubmit),
		"POST " + pr + "/approval/submit":          inPlan(authz.PurchaseRequestsSubmit),
		"POST " + pr + "/approval/approve":         customWith(authz.ApprovalsDecide),
		"POST " + pr + "/approval/decline":         customWith(authz.ApprovalsDecide),
		"POST " + pr + "/approval/recall":          customWith(authz.PurchaseRequestsSubmit),

		"GET /api/v1/approvals/purchase-requests/{purchaseRequestId}":                      view(),
		"GET /api/v1/line-items/{lineItemId}/purchase-requests":                            view(),
		"GET /api/v1/activities/{activityId}/purchase-requests":                            view(),
		"GET /api/v1/purchase-request-configuration":                                       signedIn(),
		"PUT /api/v1/purchase-request-configuration":                                       need(authz.SettingsManage),
		"POST /api/v1/simulated-gsap/purchase-requests/{purchaseRequestId}/purchase-order": need(authz.FinanceImport),
	}
}
