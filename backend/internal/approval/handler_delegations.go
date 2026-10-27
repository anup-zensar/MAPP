package approval

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

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
