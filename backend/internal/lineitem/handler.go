package lineitem

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the line-item routes; the sign-in middleware has put the actor in every request.
type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/activities/:activityId/line-items", h.section)
	r.GET("/activities/:activityId/line-items/form-options", h.formOptions)
	r.POST("/activities/:activityId/line-items", h.create)
	r.GET("/line-items/:lineItemId", h.get)
	r.PATCH("/line-items/:lineItemId", h.update)
	r.DELETE("/line-items/:lineItemId", h.delete)
	r.POST("/line-items/:lineItemId/duplicate", h.duplicate)
}

func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	details, err := h.service.Get(c.Request.Context(), a, c.Param("lineItemId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, details)
}

func (h *Handler) update(c *gin.Context) {
	var update Update
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Update(c.Request.Context(), a, c.Param("lineItemId"), update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

// delete ignores the deprecated query parameter restartApproval.
func (h *Handler) delete(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	entry, err := h.service.Delete(c.Request.Context(), a, c.Param("lineItemId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, history.DeleteResult{RecycleBinEntry: entry.View()})
}

// duplicate reads the body only for its deprecated restartApproval, which it ignores.
func (h *Handler) duplicate(c *gin.Context) {
	var ignored DuplicateInput
	if err := web.BindJSON(c, &ignored); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Duplicate(c.Request.Context(), a, c.Param("lineItemId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}

func (h *Handler) section(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	section, err := h.service.Section(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, section)
}

func (h *Handler) formOptions(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	options, err := h.service.FormOptions(c.Request.Context(), a, c.Param("activityId"))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, options)
}

func (h *Handler) create(c *gin.Context) {
	var input Input
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	view, err := h.service.Create(c.Request.Context(), a, c.Param("activityId"), input)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, view)
}
