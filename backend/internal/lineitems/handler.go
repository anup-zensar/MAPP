package lineitems

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests about line items (contract tag "Line items").
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the line-item routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the line-item routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/activities/:activityId/line-items", h.section)
	r.GET("/activities/:activityId/line-items/form-options", h.formOptions)
	r.POST("/activities/:activityId/line-items", h.create)
	r.GET("/line-items/:lineItemId", h.get)
	r.PATCH("/line-items/:lineItemId", h.update)
	r.DELETE("/line-items/:lineItemId", h.delete)
	r.POST("/line-items/:lineItemId/duplicate", h.duplicate)
}

// get answers GET /api/v1/line-items/{lineItemId} with the line item, its activity and the balance.
func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	details, err := h.service.Get(c.Request.Context(), a, c.Param("lineItemId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, details)
}

// update answers PATCH /api/v1/line-items/{lineItemId} with the line item after the save.
func (h *Handler) update(c *gin.Context) {
	var update Update
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.Update(c.Request.Context(), a, c.Param("lineItemId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// delete answers DELETE /api/v1/line-items/{lineItemId}?restartApproval=true with the recycle-bin entry, so the
// web app can offer Undo. The query value is true or false (missing: false); anything else is refused.
func (h *Handler) delete(c *gin.Context) {
	var restart bool
	switch c.Request.URL.Query().Get("restartApproval") {
	case "", "false":
	case "true":
		restart = true
	default:
		web.RespondError(c, errs.BadRequest(unreadableMessage))
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	entry, err := h.service.Delete(c.Request.Context(), a, c.Param("lineItemId"), restart)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// unreadableMessage is the answer to a query value the API cannot read (001's bad_request text).
const unreadableMessage = "MAPP couldn't read this request. Reload the page and try again."

// duplicate answers POST /api/v1/line-items/{lineItemId}/duplicate with 201 and the copy.
func (h *Handler) duplicate(c *gin.Context) {
	var input DuplicateInput
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.Duplicate(c.Request.Context(), a, c.Param("lineItemId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

// section answers GET /api/v1/activities/{activityId}/line-items with the activity's Line items section.
func (h *Handler) section(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	section, err := h.service.Section(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, section)
}

// formOptions answers GET /api/v1/activities/{activityId}/line-items/form-options with the defaults and choices
// of a new line item.
func (h *Handler) formOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	options, err := h.service.FormOptions(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, options)
}

// create answers POST /api/v1/activities/{activityId}/line-items with 201 and the new line item.
func (h *Handler) create(c *gin.Context) {
	var input Input
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.Create(c.Request.Context(), a, c.Param("activityId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}
