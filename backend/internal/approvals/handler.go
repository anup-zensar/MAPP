package approvals

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

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

// listNotifications answers GET /api/v1/notifications: the signed-in person's 50 most recent notifications
// and the unread count.
func (h *Handler) listNotifications(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.ListNotifications(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// markRead answers POST /api/v1/notifications/{notificationId}/read with the notification, now read.
func (h *Handler) markRead(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	notification, err := h.service.MarkRead(c.Request.Context(), a, c.Param("notificationId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, notification)
}

// markAllRead answers POST /api/v1/notifications/read-all with how many notifications it marked read.
func (h *Handler) markAllRead(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	result, err := h.service.MarkAllRead(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, result)
}

// decision answers POST /api/v1/activities/{activityId}/approval/approve and …/decline, with decide being
// the service's Approve or Decline, with the Approval section after the decision.
func (h *Handler) decision(decide func(context.Context, actor.Actor, string, DecisionInput) (ActivityApproval, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input DecisionInput
		if err := web.BindJSON(c, &input); err != nil {
			web.RespondError(c, err)
			return
		}
		a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
		approval, err := decide(c.Request.Context(), a, c.Param("activityId"), input)
		if err != nil {
			web.RespondError(c, err)
			return
		}
		web.Respond(c, http.StatusOK, approval)
	}
}

// approval answers GET /api/v1/activities/{activityId}/approval: the activity's Approval section.
func (h *Handler) approval(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	approval, err := h.service.Approval(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, approval)
}

// submission answers GET /api/v1/activities/{activityId}/approval/submission: what Submit would do now.
func (h *Handler) submission(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	submission, err := h.service.Submission(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, submission)
}

// submit answers POST /api/v1/activities/{activityId}/approval/submit with the Approval section after the
// submission.
func (h *Handler) submit(c *gin.Context) {
	var input SubmitInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	approval, err := h.service.Submit(c.Request.Context(), a, c.Param("activityId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, approval)
}

// recall answers POST /api/v1/activities/{activityId}/approval/recall with the Approval section after the
// recall.
func (h *Handler) recall(c *gin.Context) {
	var input RecallInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	approval, err := h.service.Recall(c.Request.Context(), a, c.Param("activityId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, approval)
}

// inbox answers GET /api/v1/approvals?tab=waiting|open|decided (default waiting): every row of the tab.
func (h *Handler) inbox(c *gin.Context) {
	tab := c.Request.URL.Query().Get("tab")
	if tab == "" {
		tab = TabWaiting
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	rows, err := h.service.Rows(c.Request.Context(), a, tab)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, RowPage{Items: rows, Total: len(rows)})
}

// summary answers GET /api/v1/approvals/summary?monthStart&monthEnd: the counts of the summary cards.
func (h *Handler) summary(c *gin.Context) {
	query := c.Request.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	counts, err := h.service.Summary(c.Request.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, counts)
}

// instant reads the RFC 3339 time of the query parameter name: nil when it is absent; a value that is not
// a time adds a problem on name.
func instant(query url.Values, name string, problems *errs.Problems) *time.Time {
	text := query.Get(name)
	if text == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, text)
	if err != nil {
		problems.Add(name, reloadMessage)
		return nil
	}
	return &t
}

// preview answers GET /api/v1/approvals/activities/{activityId}: the preview panel of the Approvals page.
func (h *Handler) preview(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	preview, err := h.service.Preview(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, preview)
}

// listWorkflows answers GET /api/v1/approval-workflows: every approval workflow, sorted by name.
func (h *Handler) listWorkflows(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	workflows, err := h.service.ListWorkflows(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, WorkflowPage{Items: workflows, Total: len(workflows)})
}

// createWorkflow answers POST /api/v1/approval-workflows with 201 and the new workflow.
func (h *Handler) createWorkflow(c *gin.Context) {
	var input WorkflowInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	workflow, err := h.service.CreateWorkflow(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, workflow)
}

// getWorkflow answers GET /api/v1/approval-workflows/{workflowId}: the workflow and the activities using it.
func (h *Handler) getWorkflow(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	details, err := h.service.GetWorkflow(c.Request.Context(), a, c.Param("workflowId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, details)
}

// updateWorkflow answers PATCH /api/v1/approval-workflows/{workflowId} with the workflow after the change.
func (h *Handler) updateWorkflow(c *gin.Context) {
	var update WorkflowUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	workflow, err := h.service.UpdateWorkflow(c.Request.Context(), a, c.Param("workflowId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, workflow)
}

// deleteWorkflow answers DELETE /api/v1/approval-workflows/{workflowId} with the recycle-bin entry of the
// workflow, so the web app can offer Undo.
func (h *Handler) deleteWorkflow(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	entry, err := h.service.DeleteWorkflow(c.Request.Context(), a, c.Param("workflowId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// setWorkflowActive answers POST /api/v1/approval-workflows/{workflowId}/deactivate (active false) and
// …/activate (active true) with the workflow after the change.
func (h *Handler) setWorkflowActive(active bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var change StatusChange
		if err := web.BindJSON(c, &change); err != nil {
			web.RespondError(c, err)
			return
		}
		a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
		workflow, err := h.service.SetWorkflowActive(c.Request.Context(), a, c.Param("workflowId"), active, change.Version)
		if err != nil {
			web.RespondError(c, err)
			return
		}
		web.Respond(c, http.StatusOK, workflow)
	}
}

// monitoring answers GET /api/v1/approvals/monitoring?monthStart&monthEnd: the Monitoring tab (017 FR-030).
func (h *Handler) monitoring(c *gin.Context) {
	query := c.Request.URL.Query()
	var problems errs.Problems
	monthStart := instant(query, "monthStart", &problems)
	monthEnd := instant(query, "monthEnd", &problems)
	if err := problems.Err(); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	m, err := h.service.Monitoring(c.Request.Context(), a, monthStart, monthEnd)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, m)
}

// runEscalations answers POST /api/v1/approvals/escalations/run: the reminders and escalations sent now (017
// FR-028).
func (h *Handler) runEscalations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	run, err := h.service.RunEscalations(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, run)
}

// ruleOptions answers GET /api/v1/approval-rule-options: what a level's rule may name (017 FR-001).
func (h *Handler) ruleOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	options, err := h.service.RuleOptions(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, options)
}

// listDelegations answers GET /api/v1/approval-delegations?scope=mine|all (017 FR-024).
func (h *Handler) listDelegations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.ListDelegations(c.Request.Context(), a, c.Request.URL.Query().Get("scope"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

// createDelegation answers POST /api/v1/approval-delegations with 201 and the new delegation (017 FR-022).
func (h *Handler) createDelegation(c *gin.Context) {
	var input DelegationInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.CreateDelegation(c.Request.Context(), a, input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

// updateDelegation answers PATCH /api/v1/approval-delegations/{delegationId} with the delegation after the
// change (017 FR-024).
func (h *Handler) updateDelegation(c *gin.Context) {
	var update DelegationUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.UpdateDelegation(c.Request.Context(), a, c.Param("delegationId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// cancelDelegation answers POST /api/v1/approval-delegations/{delegationId}/cancel with the cancelled
// delegation (017 FR-024).
func (h *Handler) cancelDelegation(c *gin.Context) {
	var cancel DelegationCancel
	if err := web.BindJSON(c, &cancel); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.CancelDelegation(c.Request.Context(), a, c.Param("delegationId"), cancel)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}
