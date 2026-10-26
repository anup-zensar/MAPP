package access

import (
	"context"

	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

// The record types of the planning records whose path parameter the middleware resolves, as the not-found
// message names them mid-sentence.
const (
	RecordPlan       = "plan"
	RecordCampaign   = "campaign"
	RecordInitiative = "initiative"
	RecordActivity   = "activity"
	RecordLineItem   = "line item"
)

// Target is the record an operation's path names (its planId, campaignId, activityId, …) as an access
// decision needs it: its plan, and for an activity and the records under it, who else sees it — its owner
// and the people named to approve it (data model §2.2).
type Target struct {
	RecordType string               // mid-sentence, such as "campaign": what the not-found message names
	Plan       authz.PlanFacts      // the record's plan
	Activity   *authz.ActivityFacts // set for an activity, its line items and its approval
}

// visibleTo reports whether the person sees the target (FR-006): its plan, or, for an activity and what is
// under it, also as its owner or a named approver.
func (t Target) visibleTo(g authz.Grants) bool {
	if t.Activity != nil {
		return g.CanViewActivity(*t.Activity)
	}
	return g.CanViewPlan(t.Plan)
}

// TargetFinder finds the record with id, and false when there is none: the service then answers 404 as
// before. It runs inside a read transaction of its own or the caller's.
type TargetFinder func(ctx context.Context, id string) (Target, bool, error)

// UseTargets tells the middleware how to find the records that path parameters name, by parameter name
// ("planId", "activityId", …). internal/app sets the planning records of 004–007 (access_targets.go); a
// feature whose operations name another planning record by path, such as an allocation of a global budget,
// adds its finder the same way. A parameter without a finder is left to the service.
func (s *Service) UseTargets(finders map[string]TargetFinder) {
	if s.targets == nil {
		s.targets = map[string]TargetFinder{}
	}
	for name, find := range finders {
		s.targets[name] = find
	}
}

// checkTarget applies the rule to the record the path names (plan "Authorization model"): a record the
// person cannot see answers 404 like a missing one (constitution VII); for an InPlan rule, a permission the
// person holds but not in the record's plan is refused with 403 and message 3 (FR-009). It returns nil when
// the request may go on; the service checks everything else.
func checkTarget(g authz.Grants, rule Rule, t Target) *errs.Error {
	if !t.visibleTo(g) {
		return errs.NotFound(authz.NotFoundMessage(t.RecordType))
	}
	if rule.Kind == InPlan && !g.InScope(rule.Permission, t.Plan) {
		return errs.Forbidden(g.Refusal(rule.Permission, &t.Plan))
	}
	return nil
}
