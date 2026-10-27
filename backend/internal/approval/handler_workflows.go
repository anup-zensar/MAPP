package approval

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

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
