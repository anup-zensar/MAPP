package access

import (
	"context"
	"net/http"
	"strings"

	"northstar/internal/platform/authz"
)

// The record types of the planning records whose path parameter the middleware resolves, as the no-access
// message names them mid-sentence (message 4).
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
	RecordType string               // mid-sentence, such as "campaign": what the no-access message names
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

// targetOf finds the record the request's path names, through the finder of the pattern's path parameter;
// false when the pattern has no parameter with a finder or the record does not exist.
func (s *Service) targetOf(r *http.Request) (Target, bool, error) {
	for _, name := range pathParameters(r.Pattern) {
		if find, known := s.targets[name]; known {
			return find(r.Context(), r.PathValue(name))
		}
	}
	return Target{}, false, nil
}

// pathParameters returns the names of the path parameters of a ServeMux pattern, in order:
// "GET /api/campaigns/{campaignId}/move" gives ["campaignId"].
func pathParameters(pattern string) []string {
	var names []string
	for rest := pattern; ; {
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			return names
		}
		end := strings.IndexByte(rest[open:], '}')
		if end < 0 {
			return names
		}
		names = append(names, strings.TrimSuffix(rest[open+1:open+end], "..."))
		rest = rest[open+end+1:]
	}
}

// checkTarget applies the rule to the record the path names (plan "Authorization model"): a record the
// person cannot see is refused with message 4, which names nothing of it (FR-008); for an InPlan rule, a
// permission the person holds but not in the record's plan is refused with message 3 (FR-009). It returns
// the refusal, or "" when the request may go on; the service checks everything else.
func checkTarget(g authz.Grants, rule Rule, t Target) string {
	if !t.visibleTo(g) {
		return authz.NoAccessMessage(t.RecordType)
	}
	if rule.Kind == InPlan && !g.InScope(rule.Permission, t.Plan) {
		return g.Refusal(rule.Permission, &t.Plan)
	}
	return ""
}
