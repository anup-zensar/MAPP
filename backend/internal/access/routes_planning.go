package access

import "mapp/internal/platform/authz"

func planRoutes() map[string]Rule {
	return map[string]Rule{
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
	}
}

func campaignRoutes() map[string]Rule {
	return map[string]Rule{
		"POST /api/v1/plans/{planId}/campaigns":               inPlan(authz.CampaignsEdit),
		"GET /api/v1/campaigns/{campaignId}":                  view(),
		"PATCH /api/v1/campaigns/{campaignId}":                inPlan(authz.CampaignsEdit),
		"DELETE /api/v1/campaigns/{campaignId}":               inPlan(authz.CampaignsDelete),
		"GET /api/v1/campaigns/{campaignId}/destinations":     viewAnd(authz.CampaignsEdit),
		"POST /api/v1/campaigns/{campaignId}/move":            inPlan(authz.CampaignsEdit),
		"POST /api/v1/campaigns/{campaignId}/copy":            viewAnd(authz.CampaignsEdit),
		"POST /api/v1/campaigns/{campaignId}/initiatives":     inPlan(authz.CampaignsEdit),
		"GET /api/v1/initiatives/{initiativeId}":              view(),
		"PATCH /api/v1/initiatives/{initiativeId}":            inPlan(authz.CampaignsEdit),
		"DELETE /api/v1/initiatives/{initiativeId}":           inPlan(authz.CampaignsDelete),
		"GET /api/v1/initiatives/{initiativeId}/destinations": inPlan(authz.CampaignsEdit),
		"POST /api/v1/initiatives/{initiativeId}/move":        inPlan(authz.CampaignsEdit),
		"POST /api/v1/initiatives/{initiativeId}/copy":        inPlan(authz.CampaignsEdit),
	}
}

func activityRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/initiatives/{initiativeId}/activity-form": inPlan(authz.ActivitiesEdit),
		"POST /api/v1/initiatives/{initiativeId}/activities":   inPlan(authz.ActivitiesEdit),
		"GET /api/v1/plans/{planId}/activities":                view(),
		"GET /api/v1/activities/{activityId}":                  view(), // the plan, the owner or a named approver
		"PATCH /api/v1/activities/{activityId}":                inPlan(authz.ActivitiesEdit),
		"DELETE /api/v1/activities/{activityId}":               inPlan(authz.ActivitiesDelete),
		"GET /api/v1/activities/{activityId}/form":             inPlan(authz.ActivitiesEdit),
		"GET /api/v1/activities/{activityId}/destinations":     inPlan(authz.ActivitiesEdit),
		"POST /api/v1/activities/{activityId}/move":            inPlan(authz.ActivitiesEdit),
		"POST /api/v1/activities/{activityId}/copy":            inPlan(authz.ActivitiesEdit),
	}
}

func lineItemRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/activities/{activityId}/line-items":              view(),
		"POST /api/v1/activities/{activityId}/line-items":             inPlan(authz.ActivitiesEdit),
		"GET /api/v1/activities/{activityId}/line-items/form-options": inPlan(authz.ActivitiesEdit),
		"GET /api/v1/line-items/{lineItemId}":                         view(),
		"PATCH /api/v1/line-items/{lineItemId}":                       inPlan(authz.ActivitiesEdit),
		"DELETE /api/v1/line-items/{lineItemId}":                      inPlan(authz.ActivitiesDelete),
		"POST /api/v1/line-items/{lineItemId}/duplicate":              inPlan(authz.ActivitiesEdit),
	}
}

func bulkRoutes() map[string]Rule {
	return map[string]Rule{
		"GET /api/v1/bulk-actions/fields":               signedIn(),
		"POST /api/v1/bulk-actions/targets":             signedIn(),
		"POST /api/v1/bulk-actions/preview":             custom(), // each target like its single operation
		"POST /api/v1/bulk-actions":                     custom(),
		"POST /api/v1/bulk-actions/{bulkActionId}/undo": custom(),
		"GET /api/v1/plans/{planId}/imports":            view(),
		"POST /api/v1/plans/{planId}/imports":           inPlan(authz.ActivitiesEdit),
		"POST /api/v1/plans/{planId}/imports/check":     inPlan(authz.ActivitiesEdit),
		"GET /api/v1/plans/{planId}/import-template":    inPlan(authz.ActivitiesEdit),
		"GET /api/v1/imports/{importId}":                view(),
		"POST /api/v1/imports/{importId}/undo":          inPlan(authz.ActivitiesDelete),
	}
}
