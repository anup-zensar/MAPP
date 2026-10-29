package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/people"
	"mapp/internal/platform/web"
)

type Handler struct {
	service *Service
	people  *people.Service
}

func NewHandler(s *Service, peopleService *people.Service) *Handler {
	return &Handler{service: s, people: peopleService}
}

// RegisterRoutes leaves out demo-people and sign-in when demo sign-in is off, so they answer 404 (FR-009).
func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	if h.service.DemoMode() {
		r.GET("/auth/demo-people", h.demoPeople)
		r.POST("/auth/sign-in", h.signIn)
	}
	r.POST("/auth/sign-out", h.signOut)
}

func (h *Handler) demoPeople(c *gin.Context) {
	active, err := h.people.ActivePeople(c.Request.Context())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, DemoPeople{Password: h.service.DemoPassword(), People: active})
}

type signInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) signIn(c *gin.Context) {
	var request signInRequest
	if err := web.BindJSON(c, &request); err != nil {
		web.RespondError(c, err)
		return
	}
	token, person, err := h.service.SignIn(c.Request.Context(), SignInAttempt{
		Email: request.Email, Password: request.Password, Client: c.ClientIP(), OldToken: sessionToken(c.Request),
	})
	if err != nil {
		web.RespondError(c, err)
		return
	}
	http.SetCookie(c.Writer, sessionCookie(c.Request, token))
	user, err := h.people.CurrentUser(c.Request.Context(), person.Actor())
	if err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, user)
}

func (h *Handler) signOut(c *gin.Context) {
	// The cookie is cleared first, so this browser is signed out even if the session cannot be deleted.
	http.SetCookie(c.Writer, clearedCookie(c.Request))
	if err := h.service.SignOut(c.Request.Context(), sessionToken(c.Request)); err != nil {
		web.RespondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// sessionCookie has no expiry, so it lasts until the browser closes; Strict keeps other sites from sending it.
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

func clearedCookie(r *http.Request) *http.Cookie {
	cookie := sessionCookie(r, "")
	cookie.MaxAge = -1 // sent as Max-Age=0: delete it now
	return cookie
}
