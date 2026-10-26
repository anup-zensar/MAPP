package home

import (
	"net/http"

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

// Routes registers the Home routes on r. All of them need a signed-in person, so all are wrapped in
// signedIn.
func (h *Handler) Routes(r *web.Router, signedIn web.Wrap) {
	r.Handle("GET /api/home/summary", signedIn(h.summary))
	r.Handle("GET /api/home/plans", signedIn(h.plans))
	r.Handle("GET /api/home/attention", signedIn(h.attention))
	r.Handle("GET /api/home/waiting", signedIn(h.waiting))
	r.Handle("GET /api/home/my-activities", signedIn(h.myActivities))
	r.Handle("GET /api/home/recent-changes", signedIn(h.recentChanges))
}

// recentChanges answers GET /api/home/recent-changes?scope=: Recent changes (FR-015, FR-016).
func (h *Handler) recentChanges(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.RecentChanges(r.Context(), a, r.URL.Query().Get("scope"))
	answer(w, r, page, err)
}

// myActivities answers GET /api/home/my-activities: My activities (FR-014).
func (h *Handler) myActivities(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.MyActivities(r.Context(), a)
	answer(w, r, view, err)
}

// attention answers GET /api/home/attention: Needs your attention (FR-011, FR-012).
func (h *Handler) attention(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.Attention(r.Context(), a)
	answer(w, r, page, err)
}

// waiting answers GET /api/home/waiting: Waiting for my approval (FR-013).
func (h *Handler) waiting(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.Waiting(r.Context(), a)
	answer(w, r, page, err)
}

// summary answers GET /api/home/summary?scope=: the four summary cards (FR-005 – FR-007).
func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	view, err := h.service.Summary(r.Context(), a, r.URL.Query().Get("scope"))
	answer(w, r, view, err)
}

// plans answers GET /api/home/plans?scope=: Plans at a glance (FR-008 – FR-010).
func (h *Handler) plans(w http.ResponseWriter, r *http.Request) {
	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
	page, err := h.service.PlansAtAGlance(r.Context(), a, r.URL.Query().Get("scope"))
	answer(w, r, page, err)
}

// answer writes view with 200, or the error.
func answer(w http.ResponseWriter, r *http.Request, view any, err error) {
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, view)
}
