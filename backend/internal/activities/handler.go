package activities

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of activities.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the activity routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the activity routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/initiatives/:initiativeId/activity-form", h.newForm)
	r.POST("/initiatives/:initiativeId/activities", h.create)
	r.GET("/plans/:planId/activities", h.listForPlan)
	r.GET("/activities/:activityId", h.get)
	r.PATCH("/activities/:activityId", h.update)
	r.DELETE("/activities/:activityId", h.delete)
	r.GET("/activities/:activityId/form", h.editForm)
	r.GET("/activities/:activityId/destinations", h.destinations)
	r.POST("/activities/:activityId/move", h.move)
	r.POST("/activities/:activityId/copy", h.copy)
	r.GET("/activities/:activityId/split-options", h.splitOptions)  // 016 FR-026
	r.POST("/activities/:activityId/split-preview", h.splitPreview) // 016 FR-027
	r.POST("/activities/:activityId/split", h.split)                // 016 FR-028
}

// splitOptions answers GET /api/v1/activities/{activityId}/split-options: what the Split dialog offers.
func (h *Handler) splitOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	options, err := h.service.SplitOptions(c.Request.Context(), a, c.Param("activityId"))
	answer(c, http.StatusOK, options, err)
}

// splitPreview answers POST /api/v1/activities/{activityId}/split-preview: both activities as the split would
// leave them and its problems, in the body (200).
func (h *Handler) splitPreview(c *gin.Context) {
	var in SplitRequest
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	preview, err := h.service.PreviewSplit(c.Request.Context(), a, c.Param("activityId"), in)
	answer(c, http.StatusOK, preview, err)
}

// split answers POST /api/v1/activities/{activityId}/split with 201, the original and the new activity.
func (h *Handler) split(c *gin.Context) {
	var in SplitRequest
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Split(c.Request.Context(), a, c.Param("activityId"), in)
	answer(c, http.StatusCreated, result, err)
}

// newForm answers GET /api/v1/initiatives/{initiativeId}/activity-form: the new-activity form with defaults.
func (h *Handler) newForm(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	form, err := h.service.NewForm(c.Request.Context(), a, c.Param("initiativeId"))
	answer(c, http.StatusOK, form, err)
}

// create answers POST /api/v1/initiatives/{initiativeId}/activities with 201 and the new activity.
func (h *Handler) create(c *gin.Context) {
	var input Input
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Create(c.Request.Context(), a, c.Param("initiativeId"), input)
	answer(c, http.StatusCreated, view, err)
}

// listForPlan answers GET /api/v1/plans/{planId}/activities: the plan's activities as plan-tree rows.
func (h *Handler) listForPlan(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	summaries, err := h.service.ListForPlan(c.Request.Context(), a, c.Param("planId"))
	page := SummaryPage{Items: summaries, Total: len(summaries)}
	if err == nil {
		page.Access, err = h.service.PlanContentAccess(c.Request.Context(), c.Param("planId"))
	}
	answer(c, http.StatusOK, page, err)
}

// get answers GET /api/v1/activities/{activityId}.
func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Get(c.Request.Context(), a, c.Param("activityId"))
	answer(c, http.StatusOK, view, err)
}

// update answers PATCH /api/v1/activities/{activityId} with the activity after the save.
func (h *Handler) update(c *gin.Context) {
	var update Update
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Update(c.Request.Context(), a, c.Param("activityId"), update)
	answer(c, http.StatusOK, view, err)
}

// delete answers DELETE /api/v1/activities/{activityId} with the recycle-bin entry, so the web app can offer
// Undo.
func (h *Handler) delete(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.Delete(c.Request.Context(), a, c.Param("activityId"))
	answer(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()}, err)
}

// editForm answers GET /api/v1/activities/{activityId}/form: the edit form, without defaults.
func (h *Handler) editForm(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	form, err := h.service.EditForm(c.Request.Context(), a, c.Param("activityId"))
	answer(c, http.StatusOK, form, err)
}

// destinations answers GET /api/v1/activities/{activityId}/destinations: every initiative of the plan, with
// why the activity can't be moved or copied there.
func (h *Handler) destinations(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	destinations, err := h.service.Destinations(c.Request.Context(), a, c.Param("activityId"))
	answer(c, http.StatusOK, DestinationPage{Items: destinations, Total: len(destinations)}, err)
}

// move answers POST /api/v1/activities/{activityId}/move with the activity in its new initiative.
func (h *Handler) move(c *gin.Context) {
	var in MoveInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Move(c.Request.Context(), a, c.Param("activityId"), in)
	answer(c, http.StatusOK, view, err)
}

// copy answers POST /api/v1/activities/{activityId}/copy with 201 and the copy.
func (h *Handler) copy(c *gin.Context) {
	var in CopyInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Copy(c.Request.Context(), a, c.Param("activityId"), in)
	answer(c, http.StatusCreated, view, err)
}

// answer writes v with status, or the error when there is one.
func answer(c *gin.Context, status int, v any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, v)
}
