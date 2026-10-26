package access

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/web"
)

// noRuleMessage answers an operation without an access rule: a programming mistake, refused (fail closed).
const noRuleMessage = "This action isn't available. Ask an administrator if you need it."

// Authorize applies the operation's rule from routes.go (014 plan "Authorization model"), puts the person's
// Grants in the request's context and logs every refusal, its own or a service's, with operation and code.
func (s *Service) Authorize(c *gin.Context) {
	ctx := c.Request.Context()
	operation := web.Operation(c)
	grants, refusal, err := s.decide(c, operation)
	if err != nil {
		web.RespondError(c, err)
		return
	}
	if refusal != nil {
		logRefusal(ctx, operation, refusal.Code)
		web.RespondError(c, refusal)
		return
	}
	c.Request = c.Request.WithContext(authz.With(ctx, grants))
	c.Next()
	if c.Writer.Status() == http.StatusForbidden {
		logRefusal(ctx, operation, errs.CodeForbidden) // refused by the service: a scope or a record rule
	}
}

func (s *Service) decide(c *gin.Context, operation string) (authz.Grants, *errs.Error, error) {
	ctx := c.Request.Context()
	rule, known := RuleFor(operation)
	if !known {
		slog.ErrorContext(ctx, "operation without an access rule", "operation", operation)
		return authz.Grants{}, errs.Forbidden(noRuleMessage), nil
	}
	a, _ := actor.From(ctx)
	grants, err := s.GrantsFor(ctx, a.ID)
	if err != nil {
		return grants, nil, err
	}
	if rule.lacksPermission(grants) {
		return grants, errs.Forbidden(grants.Refusal(rule.Permission, nil)), nil
	}
	if !rule.namesRecord() {
		return grants, nil, nil
	}
	target, found, err := s.targetOf(c)
	if err != nil || !found {
		return grants, nil, err
	}
	return grants, checkTarget(grants, rule, target), nil
}

// targetOf finds the record the path names through the first parameter that has a finder.
func (s *Service) targetOf(c *gin.Context) (Target, bool, error) {
	for _, parameter := range c.Params {
		if find, known := s.targets[parameter.Key]; known {
			return find(c.Request.Context(), parameter.Value)
		}
	}
	return Target{}, false, nil
}

func (r Rule) lacksPermission(grants authz.Grants) bool {
	return r.Permission != "" && r.Kind != Custom && !grants.Has(r.Permission)
}

func (r Rule) namesRecord() bool {
	return r.Kind == View || r.Kind == InPlan || r.Kind == Custom
}

func logRefusal(ctx context.Context, operation, code string) {
	slog.WarnContext(ctx, "request refused", "operation", operation, "code", code)
}
