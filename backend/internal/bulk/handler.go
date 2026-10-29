package bulk

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/bulk-actions/fields", h.fields)
	r.POST("/bulk-actions/targets", h.targets)
	r.POST("/bulk-actions/preview", h.preview)
	r.POST("/bulk-actions", h.run)
	r.POST("/bulk-actions/:bulkActionId/undo", h.undo)
}

func (h *Handler) fields(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	list, err := h.service.Fields(c.Request.Context(), a, Level(c.Query("level")))
	answer(c, http.StatusOK, list, err)
}

func (h *Handler) targets(c *gin.Context) {
	var q TargetQuery
	if err := web.BindJSON(c, &q); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.Targets(c.Request.Context(), a, q)
	answer(c, http.StatusOK, page, err)
}

func (h *Handler) preview(c *gin.Context) {
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Preview(c.Request.Context(), a, req)
	answer(c, http.StatusOK, result, err)
}

func (h *Handler) run(c *gin.Context) {
	var req Request
	if err := web.BindJSON(c, &req); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Run(c.Request.Context(), a, req)
	answer(c, http.StatusCreated, result, err)
}

func (h *Handler) undo(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Undo(c.Request.Context(), a, c.Param("bulkActionId"))
	answer(c, http.StatusOK, result, err)
}

func answer(c *gin.Context, status int, v any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, v)
}
