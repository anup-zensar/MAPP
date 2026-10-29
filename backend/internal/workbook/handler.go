package workbook

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/allwork"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type Handler struct {
	service *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/plans/:planId/imports", h.list)
	r.POST("/plans/:planId/imports/check", h.check)
	r.POST("/plans/:planId/imports", h.importFile)
	r.GET("/plans/:planId/import-template", h.template)
	r.GET("/plans/:planId/workbook", h.export)
	r.GET("/imports/:importId", h.get)
	r.POST("/imports/:importId/undo", h.undo)
}

func (h *Handler) list(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.List(c.Request.Context(), a, c.Param("planId"))
	answer(c, http.StatusOK, page, err)
}

func (h *Handler) check(c *gin.Context) {
	var up Upload
	if err := web.BindJSON(c, &up); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	result, err := h.service.Check(c.Request.Context(), a, c.Param("planId"), up)
	answer(c, http.StatusOK, result, err)
}

func (h *Handler) importFile(c *gin.Context) {
	var up Upload
	if err := web.BindJSON(c, &up); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	detail, err := h.service.Import(c.Request.Context(), a, c.Param("planId"), up)
	answer(c, http.StatusCreated, detail, err)
}

func (h *Handler) template(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	file, err := h.service.Template(c.Request.Context(), a, c.Param("planId"))
	sendFile(c, file, err)
}

func (h *Handler) export(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	file, err := h.service.Export(c.Request.Context(), a, c.Param("planId"))
	sendFile(c, file, err)
}

func (h *Handler) get(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	detail, err := h.service.Get(c.Request.Context(), a, c.Param("importId"))
	answer(c, http.StatusOK, detail, err)
}

func (h *Handler) undo(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	detail, err := h.service.Undo(c.Request.Context(), a, c.Param("importId"))
	answer(c, http.StatusOK, detail, err)
}

func sendFile(c *gin.Context, file File, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", allwork.ContentDisposition(file.Name))
	c.Data(http.StatusOK, xlsxType, file.Data)
}

func answer(c *gin.Context, status int, v any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, status, v)
}
