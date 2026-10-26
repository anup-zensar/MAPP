package auth

import (
	"net/http"

	"mapp/internal/people"
	"mapp/internal/platform/web"
)

// Handler answers the HTTP requests of demo sign-in.
type Handler struct {
	service *Service
	people  *people.Service
}

// NewHandler returns the handler of the sign-in routes. It uses the people service for what the sign-in
// page lists and for the person a sign-in answers.
func NewHandler(s *Service, peopleService *people.Service) *Handler {
	return &Handler{service: s, people: peopleService}
}

// Routes registers the sign-in routes on r. GET /api/v1/auth/demo-people and POST /api/v1/auth/sign-in exist only
// when demo sign-in is on (FR-009); elsewhere they answer 404 like any unknown path. POST
// /api/v1/auth/sign-out always exists. None of them needs a session, so the sign-in check that every
// package's Routes receives is not used here.
func (h *Handler) Routes(r *web.Router, _ web.Wrap) {
	if h.service.DemoMode() {
		r.Handle("GET /api/v1/auth/demo-people", h.demoPeople)
		r.Handle("POST /api/v1/auth/sign-in", h.signIn)
	}
	r.Handle("POST /api/v1/auth/sign-out", h.signOut)
}

// demoPeople answers GET /api/v1/auth/demo-people: the active demo people, sorted by name, and the shared
// demo password, which the sign-in page shows (FR-001, FR-002). Without demo data the list is empty
// (FR-028).
func (h *Handler) demoPeople(w http.ResponseWriter, r *http.Request) {
	active, err := h.people.ActivePeople(r.Context())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, DemoPeople{Password: h.service.DemoPassword(), People: active})
}

// signInRequest is the body of POST /api/v1/auth/sign-in.
type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// signIn answers POST /api/v1/auth/sign-in: it starts a session, sets its cookie and answers the signed-in
// person (FR-003, FR-004). The session this browser had before ends.
func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var request signInRequest
	if err := web.ReadJSON(w, r, &request); err != nil {
		web.WriteError(w, r, err)
		return
	}
	token, person, err := h.service.SignIn(r.Context(), request.Email, request.Password, sessionToken(r))
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	http.SetCookie(w, sessionCookie(r, token)) // the session exists now, so the browser gets its token
	user, err := h.people.CurrentUser(r.Context(), person.Actor())
	if err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, user)
}

// signOut answers POST /api/v1/auth/sign-out with 204: it ends the session and tells the browser to forget
// the cookie. It succeeds also when nobody is signed in (FR-006).
func (h *Handler) signOut(w http.ResponseWriter, r *http.Request) {
	// The cookie is cleared first, so this browser is signed out even if the session cannot be deleted.
	http.SetCookie(w, clearedCookie(r))
	if err := h.service.SignOut(r.Context(), sessionToken(r)); err != nil {
		web.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// sessionCookie is the cookie that carries a new session's token to the browser (research R6): HttpOnly,
// so page scripts cannot read it; SameSite=Strict, so other sites cannot make the browser send it; for the
// whole site; without an expiry date, so it lasts until the browser closes; Secure when the page came over
// HTTPS.
func sessionCookie(r *http.Request, token string) *http.Cookie {
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   r.TLS != nil,
	}
}

// clearedCookie tells the browser to delete the session cookie.
func clearedCookie(r *http.Request) *http.Cookie {
	cookie := sessionCookie(r, "")
	cookie.MaxAge = -1 // sent as Max-Age=0: delete it now
	return cookie
}
