package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

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

// RegisterRoutes registers the sign-in routes on r. GET /api/v1/auth/demo-people and POST
// /api/v1/auth/sign-in exist only when demo sign-in is on (FR-009); elsewhere they answer 404 like any
// unknown path. POST /api/v1/auth/sign-out always exists.
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	if h.service.DemoMode() {
		r.GET("/auth/demo-people", h.demoPeople)
		r.POST("/auth/sign-in", h.signIn)
	}
	r.POST("/auth/sign-out", h.signOut)
}

// demoPeople answers GET /api/v1/auth/demo-people: the active demo people, sorted by name, and the shared
// demo password, which the sign-in page shows (FR-001, FR-002). Without demo data the list is empty
// (FR-028).
func (h *Handler) demoPeople(c *gin.Context) {
	active, err := h.people.ActivePeople(c.Request.Context())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, DemoPeople{Password: h.service.DemoPassword(), People: active})
}

// signInRequest is the body of POST /api/v1/auth/sign-in.
type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// signIn answers POST /api/v1/auth/sign-in: it starts a session, sets its cookie and answers the signed-in
// person (FR-003, FR-004). The session this browser had before ends.
func (h *Handler) signIn(c *gin.Context) {
	var request signInRequest
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	token, person, err := h.service.SignIn(c.Request.Context(), request.Email, request.Password, sessionToken(c.Request))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	http.SetCookie(c.Writer, sessionCookie(c.Request, token)) // the session exists now, so the browser gets its token
	user, err := h.people.CurrentUser(c.Request.Context(), person.Actor())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, user)
}

// signOut answers POST /api/v1/auth/sign-out with 204: it ends the session and tells the browser to forget
// the cookie. It succeeds also when nobody is signed in (FR-006).
func (h *Handler) signOut(c *gin.Context) {
	// The cookie is cleared first, so this browser is signed out even if the session cannot be deleted.
	http.SetCookie(c.Writer, clearedCookie(c.Request))
	if err := h.service.SignOut(c.Request.Context(), sessionToken(c.Request)); err != nil {
		web.RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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
