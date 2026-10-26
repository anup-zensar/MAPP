package home

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of the Home dashboard: one read-only route per card (FR-022).
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the Home routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the Home routes on r.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/home/summary", h.summary)
	r.GET("/home/plans", h.plans)
	r.GET("/home/attention", h.attention)
	r.GET("/home/waiting", h.waiting)
	r.GET("/home/my-activities", h.myActivities)
	r.GET("/home/recent-changes", h.recentChanges)
}

// recentChanges answers GET /api/v1/home/recent-changes?scope=: Recent changes (FR-015, FR-016).
func (h *Handler) recentChanges(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.RecentChanges(c.Request.Context(), a, c.Request.URL.Query().Get("scope"))
	answer(c, page, err)
}

// myActivities answers GET /api/v1/home/my-activities: My activities (FR-014).
func (h *Handler) myActivities(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.MyActivities(c.Request.Context(), a)
	answer(c, view, err)
}

// attention answers GET /api/v1/home/attention: Needs your attention (FR-011, FR-012).
func (h *Handler) attention(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.Attention(c.Request.Context(), a)
	answer(c, page, err)
}

// waiting answers GET /api/v1/home/waiting: Waiting for my approval (FR-013).
func (h *Handler) waiting(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.Waiting(c.Request.Context(), a)
	answer(c, page, err)
}

// summary answers GET /api/v1/home/summary?scope=: the four summary cards (FR-005 – FR-007).
func (h *Handler) summary(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	view, err := h.service.Summary(c.Request.Context(), a, c.Request.URL.Query().Get("scope"))
	answer(c, view, err)
}

// plans answers GET /api/v1/home/plans?scope=: Plans at a glance (FR-008 – FR-010).
func (h *Handler) plans(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	page, err := h.service.PlansAtAGlance(c.Request.Context(), a, c.Request.URL.Query().Get("scope"))
	answer(c, page, err)
}

// answer writes view with 200, or the error.
func answer(c *gin.Context, view any, err error) {
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, view)
}
