package overspend

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
	r.GET("/overspend-configuration", h.configuration)
	r.PUT("/overspend-configuration", h.saveConfiguration)
}

func actorOf(c *gin.Context) actor.Actor {
	a, _ := actor.From(c.Request.Context())
	return a
}

func (h *Handler) configuration(c *gin.Context) {
	view, err := h.service.Configuration(c.Request.Context(), actorOf(c))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}

func (h *Handler) saveConfiguration(c *gin.Context) {
	var in ConfigurationInput
	if err := web.BindJSON(c, &in); err != nil {
		web.RespondError(c, err)
		return
	}
	view, err := h.service.SaveConfiguration(c.Request.Context(), actorOf(c), in)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}
