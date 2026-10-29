package dashboard

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/allwork"
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
	r.GET("/dashboards/executive", h.dashboard(KindExecutive))
	r.GET("/dashboards/regional", h.dashboard(KindRegional))
	r.GET("/dashboards/executive/export", h.export(KindExecutive))
	r.GET("/dashboards/regional/export", h.export(KindRegional))
	r.GET("/plans/:planId/export", h.exportPlan)
}

func (h *Handler) dashboard(kind Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		q, err := ParseQuery(c.Request.URL.Query())
		if err != nil {
			web.RespondError(c, err)
			return
		}
		a, _ := actor.From(c.Request.Context())
		view, err := h.service.view(c.Request.Context(), a, request{kind: kind, query: q})
		if err != nil {
			web.RespondError(c, err)
			return
		}
		web.Respond(c, http.StatusOK, view)
	}
}

func (h *Handler) export(kind Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		values := c.Request.URL.Query()
		q, err := ParseQuery(values)
		if err != nil {
			web.RespondError(c, err)
			return
		}
		format, err := ParseFormat(values)
		if err != nil {
			web.RespondError(c, err)
			return
		}
		a, _ := actor.From(c.Request.Context())
		file, err := h.service.DashboardExport(c.Request.Context(), a, request{kind: kind, query: q}, format)
		respondFile(c, file, err)
	}
}

func (h *Handler) exportPlan(c *gin.Context) {
	values := c.Request.URL.Query()
	q, err := ParseQuery(values)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	format, err := ParseFormat(values)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	file, err := h.service.PlanExport(c.Request.Context(), a, PlanFile{PlanID: c.Param("planId"), Currency: q.Currency, Format: format})
	respondFile(c, file, err)
}

func respondFile(c *gin.Context, file File, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	c.Header("Content-Disposition", allwork.ContentDisposition(file.Name))
	c.Data(http.StatusOK, file.ContentType, file.Body)
}
