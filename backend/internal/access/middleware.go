package access

import (
	"log/slog"
	"net/http"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// noRuleMessage refuses an operation that has no access rule: a programming mistake that a test rules out,
// answered safely (fail closed, FR-010).
const noRuleMessage = "This action isn't available. Ask an administrator if you need it."

// Authorize wraps every route that needs a signed-in person, after auth.RequireSignIn, which put the actor in
// the context (plan "Authorization model"). It builds the person's Grants, puts them in the request's
// context for the services, and applies the operation's rule from routes.go: Need and InPlan operations are
// refused with 403 forbidden when no role of the person includes the permission; the services check scopes
// and records. An operation without a rule is refused.
//
// Every refusal, by the middleware or by a service, is written to the technical log with the operation, the
// person's ID and the code forbidden, never a name or an e-mail (FR-039, constitution VI). Refused requests
// change nothing and are not in the activity log (D-241).
func (s *Service) Authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, _ := actor.From(r.Context())
		rule, known := RuleFor(r.Pattern)
		if !known {
			slog.Error("operation without an access rule", "operation", r.Pattern, "person", a.ID)
			web.WriteError(w, r, errs.Forbidden(noRuleMessage))
			return
		}
		g, err := s.GrantsFor(r.Context(), a.ID)
		if err != nil {
			web.WriteError(w, r, err)
			return
		}
		if (rule.Kind == Need || rule.Kind == InPlan) && !g.Has(rule.Permission) {
			logForbidden(r, a.ID)
			web.WriteError(w, r, errs.Forbidden(g.Refusal(rule.Permission, nil)))
			return
		}

		answer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next(answer, r.WithContext(authz.With(r.Context(), g)))
		if answer.status == http.StatusForbidden {
			logForbidden(r, a.ID) // refused by the service: a scope or a record rule
		}
	}
}

// logForbidden writes the technical log line of a refusal: the operation, the person's ID and the code.
func logForbidden(r *http.Request, personID string) {
	slog.Warn("request refused", "operation", r.Pattern, "person", personID, "code", errs.CodeForbidden)
}

// statusWriter passes the answer on and remembers its status, so Authorize can log refusals made by
// services.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status code and sends it.
func (s *statusWriter) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
