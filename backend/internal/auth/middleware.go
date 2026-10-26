package auth

import (
	"net/http"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// RequireSignIn wraps the handler of every route that needs a signed-in person (FR-005, FR-007). It finds
// the person from the session cookie with Authenticate and calls next with that person as the actor in the
// request's context. Without a usable session it answers 401, and next does not run.
//
// When demo sign-in is off it answers sign_in_required to every request, even with a session made earlier
// in demo mode: until Shell single sign-on arrives there is no way to sign in (FR-009).
//
// next receives a context with the actor and without a transaction: Authenticate's Write has finished, so
// next can start transactions of its own.
func (s *Service) RequireSignIn(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.demoMode {
			web.WriteError(w, r, errs.SignInRequired())
			return
		}
		a, err := s.Authenticate(r.Context(), sessionToken(r))
		if err != nil {
			web.WriteError(w, r, err)
			return
		}
		next(w, r.WithContext(actor.With(r.Context(), a)))
	}
}

// sessionToken returns the session token the browser sent in its cookie, or "" when it sent none.
func sessionToken(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "" // no cookie: nobody is signed in in this browser
	}
	return cookie.Value
}
