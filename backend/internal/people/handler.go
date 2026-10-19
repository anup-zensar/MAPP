package people

import (
	"net/http"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/web"
)

// Handler answers the HTTP requests about people, teams, the signed-in person and their preferences.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the people routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the people and team routes on r. All of them need a signed-in person, so all are
// wrapped in signedIn. The handlers of people are in people_handler.go, those of teams in teams_handler.go.
// There is no DELETE /api/people/{personId}: people are never deleted (FR-008).
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/me", signedIn(h.currentUser))
	r.Handle("PATCH /api/me", signedIn(h.updateProfile))
	r.Handle("PATCH /api/me/preferences", signedIn(h.updatePreferences))
	r.Handle("GET /api/me/display-currencies", signedIn(h.listDisplayCurrencies))

	r.Handle("GET /api/people", signedIn(h.listPeople))
	r.Handle("POST /api/people", signedIn(h.createPerson))
	r.Handle("GET /api/people/{personId}", signedIn(h.getPerson))
	r.Handle("PATCH /api/people/{personId}", signedIn(h.updatePerson))
	r.Handle("POST /api/people/{personId}/impact", signedIn(h.checkPersonImpact))
	r.Handle("POST /api/people/{personId}/status", signedIn(h.setPersonStatus))

	r.Handle("GET /api/teams", signedIn(h.listTeams))
	r.Handle("POST /api/teams", signedIn(h.createTeam))
	r.Handle("GET /api/teams/{teamId}", signedIn(h.getTeam))
	r.Handle("PATCH /api/teams/{teamId}", signedIn(h.updateTeam))
	r.Handle("DELETE /api/teams/{teamId}", signedIn(h.deleteTeam))
	r.Handle("POST /api/teams/{teamId}/impact", signedIn(h.checkTeamImpact))
	r.Handle("POST /api/teams/{teamId}/status", signedIn(h.setTeamStatus))
}

// currentUser answers GET /api/me: who is signed in, with their preferences.
func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	user, err := h.service.CurrentUser(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, user)
}

// updateProfile answers PATCH /api/me: My profile changes the signed-in person's name and job title, and the
// answer is the signed-in person after the change.
func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	var update ProfileUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	user, err := h.service.UpdateProfile(r.Context(), a, update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, user)
}

// listDisplayCurrencies answers GET /api/me/display-currencies: the currencies the signed-in person can
// choose to show amounts in.
func (h *Handler) listDisplayCurrencies(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	options, err := h.service.DisplayCurrencies(r.Context(), a)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, DisplayCurrencyPage{Items: options, Total: len(options)})
}

// updatePreferences answers PATCH /api/me/preferences: it changes the preferences that were sent and
// answers all of them.
func (h *Handler) updatePreferences(w http.ResponseWriter, r *http.Request) {
	var update PreferencesUpdate
	if err := web.ReadJSON(w, r, &update); err != nil {
		web.WriteError(w, r, err)
		return
	}
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	prefs, err := h.service.UpdatePreferences(r.Context(), a, update)
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, prefs)
}
