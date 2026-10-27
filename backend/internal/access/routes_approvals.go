package access

import "mapp/internal/platform/authz"

func approvalRoutes() map[string]Rule {
	return map[string]Rule{
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
	}
}

func approvalRuleRoutes() map[string]Rule {
	return map[string]Rule{
		// 017 Approval rules (D-301)
		"GET /api/v1/approval-rule-options":                       signedIn(),
		"GET /api/v1/approval-delegations":                        customWith(authz.WorkflowsManage), // scope=all needs workflows.manage
		"POST /api/v1/approval-delegations":                       customWith(authz.WorkflowsManage), // for someone else
		"PATCH /api/v1/approval-delegations/{delegationId}":       customWith(authz.WorkflowsManage), // the delegating person, or workflows.manage
		"POST /api/v1/approval-delegations/{delegationId}/cancel": customWith(authz.WorkflowsManage),
		"GET /api/v1/approvals/monitoring":                        signedIn(), // filtered
		"POST /api/v1/approvals/escalations/run":                  need(authz.WorkflowsManage),
	}
}
