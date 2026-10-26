package access

import (
	"log/slog"
	"net/http"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// noRuleMessage refuses an operation that has no access rule: a programming mistake that a test rules out,
// answered safely (fail closed, FR-010).
const noRuleMessage = "This action isn't available. Ask an administrator if you need it."

// Authorize wraps every route that needs a signed-in person, after auth.RequireSignIn, which put the actor in
// the context (plan "Authorization model"). It builds the person's Grants, puts them in the request's
// context for the services, and applies the operation's rule from routes.go:
//
//   - Need and InPlan operations, and View operations with a permission, are refused with 403 forbidden
//     when no role of the person includes the permission (messages 1 and 2);
//   - when the path names a planning record (a plan, campaign, initiative, activity or line item, through
//     the finders of UseTargets), View, InPlan and Custom operations answer 404 when the person cannot
//     see the record, exactly as for a missing one, and InPlan operations are refused when the permission
//     is not theirs in the record's plan (message 3, FR-009);
//   - the services check what the path does not say: destinations of moves and copies, the teams of a
//     plan, list filters and the rules of Custom operations.
//
// An operation without a rule is refused.
//
// Every refusal, by the middleware or by a service, is written to the technical log with the operation, the
// person's ID and the code (forbidden, or not_found for a record they cannot see), never a name or an
// e-mail (FR-039, constitution VI). Refused requests change nothing and are not in the activity log (D-241).
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
		if rule.Permission != "" && rule.Kind != Custom && !g.Has(rule.Permission) {
			logRefusal(r, a.ID, errs.CodeForbidden)
			web.WriteError(w, r, errs.Forbidden(g.Refusal(rule.Permission, nil)))
			return
		}
		if rule.Kind == View || rule.Kind == InPlan || rule.Kind == Custom {
			target, found, err := s.targetOf(r)
			if err != nil {
				web.WriteError(w, r, err)
				return
			}
			if refusal := checkTarget(g, rule, target); found && refusal != nil {
				logRefusal(r, a.ID, refusal.Code)
				web.WriteError(w, r, refusal)
				return
			}
		}

		answer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next(answer, r.WithContext(authz.With(r.Context(), g)))
		if answer.status == http.StatusForbidden {
			logRefusal(r, a.ID, errs.CodeForbidden) // refused by the service: a scope or a record rule
		}
	}
}

// logRefusal writes the technical log line of a refusal: the operation, the person's ID and the code.
func logRefusal(r *http.Request, personID, code string) {
	slog.Warn("request refused", "operation", r.Pattern, "person", personID, "code", code)
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
