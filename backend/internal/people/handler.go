package people

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests about people, teams, the signed-in person and their preferences.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the people routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// RegisterRoutes registers the people and team routes on r. The handlers of people are in people_handler.go,
// those of teams in teams_handler.go. There is no DELETE /api/v1/people/{personId}: people are never deleted
// (FR-008).
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/me", h.currentUser)
	r.PATCH("/me", h.updateProfile)
	r.PATCH("/me/preferences", h.updatePreferences)
	r.GET("/me/display-currencies", h.listDisplayCurrencies)

	r.GET("/people", h.listPeople)
	r.POST("/people", h.createPerson)
	r.GET("/people/:personId", h.getPerson)
	r.PATCH("/people/:personId", h.updatePerson)
	r.POST("/people/:personId/impact", h.checkPersonImpact)
	r.POST("/people/:personId/status", h.setPersonStatus)

	r.GET("/teams", h.listTeams)
	r.POST("/teams", h.createTeam)
	r.GET("/teams/:teamId", h.getTeam)
	r.PATCH("/teams/:teamId", h.updateTeam)
	r.DELETE("/teams/:teamId", h.deleteTeam)
	r.POST("/teams/:teamId/impact", h.checkTeamImpact)
	r.POST("/teams/:teamId/status", h.setTeamStatus)
}

// currentUser answers GET /api/v1/me: who is signed in, with their preferences.
func (h *Handler) currentUser(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	user, err := h.service.CurrentUser(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, user)
}

// updateProfile answers PATCH /api/v1/me: My profile changes the signed-in person's name and job title, and the
// answer is the signed-in person after the change.
func (h *Handler) updateProfile(c *gin.Context) {
	var update ProfileUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	user, err := h.service.UpdateProfile(c.Request.Context(), a, update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, user)
}

// listDisplayCurrencies answers GET /api/v1/me/display-currencies: the currencies the signed-in person can
// choose to show amounts in.
func (h *Handler) listDisplayCurrencies(c *gin.Context) {
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	options, err := h.service.DisplayCurrencies(c.Request.Context(), a)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, DisplayCurrencyPage{Items: options, Total: len(options)})
}

// updatePreferences answers PATCH /api/v1/me/preferences: it changes the preferences that were sent and
// answers all of them.
func (h *Handler) updatePreferences(c *gin.Context) {
	var update PreferencesUpdate
	if err := web.BindJSON(c, &update); err != nil {
		web.RespondError(c, err)
		return
	}
	a, _ := actor.From(c.Request.Context()) // put there by the sign-in middleware
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), a, update)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, prefs)
}
