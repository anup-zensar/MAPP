package swbs

import (
	"net/http"
	"strconv"

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
	r.GET("/simulated-gsap/swbs-runs", h.list)
	r.POST("/simulated-gsap/swbs-runs", h.run)
}

func (h *Handler) list(c *gin.Context) {
	a, _ := actor.From(c.Request.Context())
	page, err := h.service.Runs(c.Request.Context(), a, limitOf(c.Query("limit")))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, page)
}

func (h *Handler) run(c *gin.Context) {
	var in RunInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context())
	run, err := h.service.Run(c.Request.Context(), a, in)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusCreated, run)
}

// limitOf reads the limit query value: 0 when it is left out, -1 (refused by the service) when it is not a
// whole number of at least 1.
func limitOf(raw string) int {
	if raw == "" {
		return 0
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 {
		return -1
	}
	return limit
}
