package approval

import "github.com/gin-gonic/gin"

// Handler answers the HTTP requests of approval workflows and approvals.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the approvals routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the approvals routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/approval-workflows", h.listWorkflows)
	r.POST("/approval-workflows", h.createWorkflow)
	r.GET("/approval-workflows/:workflowId", h.getWorkflow)
	r.PATCH("/approval-workflows/:workflowId", h.updateWorkflow)
	r.DELETE("/approval-workflows/:workflowId", h.deleteWorkflow)
	r.POST("/approval-workflows/:workflowId/deactivate", h.setWorkflowActive(false))
	r.POST("/approval-workflows/:workflowId/activate", h.setWorkflowActive(true))
	r.GET("/activities/:activityId/approval", h.approval)
	r.GET("/activities/:activityId/approval/submission", h.submission)
	r.POST("/activities/:activityId/approval/submit", h.submit)
	r.POST("/activities/:activityId/approval/approve", h.decision(h.service.Approve))
	r.POST("/activities/:activityId/approval/decline", h.decision(h.service.Decline))
	r.POST("/activities/:activityId/approval/recall", h.recall)
	r.GET("/approvals", h.inbox)
	r.GET("/approvals/summary", h.summary)
	r.GET("/approvals/activities/:activityId", h.preview)
	r.GET("/approvals/monitoring", h.monitoring)
	r.POST("/approvals/escalations/run", h.runEscalations)
	r.GET("/approval-rule-options", h.ruleOptions)
	r.GET("/approval-delegations", h.listDelegations)
	r.POST("/approval-delegations", h.createDelegation)
	r.PATCH("/approval-delegations/:delegationId", h.updateDelegation)
	r.POST("/approval-delegations/:delegationId/cancel", h.cancelDelegation)
	r.GET("/notifications", h.listNotifications)
	r.POST("/notifications/:notificationId/read", h.markRead)
	r.POST("/notifications/read-all", h.markAllRead)
}
