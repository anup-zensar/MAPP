package approval

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

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
