package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/logging"
	"mapp/internal/platform/web"
)

// RequireSignIn answers 401 without a usable session, and sign_in_required to everyone while demo sign-in
// is off, even with an older session; otherwise it puts the actor and actor_id in the request's context.
func (s *Service) RequireSignIn(c *gin.Context) {
	if !s.demoMode {
		web.RespondError(c, errs.SignInRequired())
		return
	}
	a, err := s.Authenticate(c.Request.Context(), sessionToken(c.Request))
	if err != nil {
		web.RespondError(c, err)
		return
	}
	ctx := logging.WithAttrs(actor.With(c.Request.Context(), a), slog.String("actor_id", a.ID))
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
