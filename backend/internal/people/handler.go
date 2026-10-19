package people

import (
	"net/http"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/web"
)

// Handler answers the HTTP requests about the signed-in person and their preferences.
type Handler struct {
	service *Service
}

// NewHandler returns the handler of the people routes.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Routes registers the people routes on r. Both need a signed-in person, so both are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/me", signedIn(h.currentUser))
	r.Handle("PATCH /api/me/preferences", signedIn(h.updatePreferences))
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
