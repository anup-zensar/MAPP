package approvals

import (
	"context"

	"mapp/internal/platform/authz"
)

// Who may decide, recall and approve under access control (feature 014, FR-025 – FR-028, D-231, D-232). The
// access middleware has already checked that the person sees the activity (check order step 1) and, for
// Submit, that they hold "Submit activities for approval and recall them" in its plan. A context without a
// person (an internal call, service tests) may do everything, as before feature 014.

// The messages of approvals under access control (014 plan "User-visible messages" 5 and 8; FR-027).
const (
	msgCannotApprove          = "%s can't approve activities. Give them a role that approves first."                 // 8
	msgRecallOwnerOrSubmitter = "Only the activity's owner or the person who submitted it can recall this activity." // FR-027
)

// Approvers is what approvals asks of access control (feature 014): who holds "Approve or decline when
// named as an approver" (FR-028). internal/app sets it over the access service; unset, everyone may.
type Approvers interface {
	CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error)
}

// UseApprovers tells approvals who may approve (feature 014, FR-028).
func (s *Service) UseApprovers(a Approvers) {
	s.approvers = a
}

// recallRefusal says why the person acting may not recall the open round (014 FR-027): "" when they may.
// They may with "Approve or decline on behalf of the named approvers" in the activity's plan, or as its owner
// or the round's submitter holding "Submit activities for approval and recall them" there. shown is false for
// someone who is neither, whose screen hides Recall.
func recallRefusal(ctx context.Context, facts ActivityFacts, round Round) (refusal string, shown bool) {
	g, restricted := authz.From(ctx)
	if !restricted || g.InScope(authz.ApprovalsDecideOnBehalf, facts.Access) {
		return "", false
	}
	if facts.OwnerID != g.PersonID && round.SubmittedBy.ID != g.PersonID {
		return msgRecallOwnerOrSubmitter, false
	}
	if !g.InScope(authz.ActivitiesSubmit, facts.Access) {
		return g.Refusal(authz.ActivitiesSubmit, &facts.Access), true
	}
	return "", false
}

// submitRefusal says why the person acting may not submit or re-submit the activity (014 FR-027, messages 2
// and 3): "" when they may.
func submitRefusal(ctx context.Context, facts ActivityFacts) string {
	return authz.Available(ctx, authz.ActivitiesSubmit, facts.Access).Reason
}

// limitDecisions makes Recall unavailable to a person who may not use it on the open round (014 FR-027),
// with the reason the screen shows, or no reason when the screen hides it. Approve and Decline come from
// 017's authority (data-model §6), which already follows 014's check order.
func limitDecisions(ctx context.Context, actions *ActionSet, facts ActivityFacts, open Round) {
	if refusal, shown := recallRefusal(ctx, facts, open); refusal != "" && actions.Recall.Available {
		reason := ""
		if shown {
			reason = refusal
		}
		actions.Recall = ActionAvailability{Available: false, Reason: reason}
	}
}

// limitSubmit makes Submit and Re-submit unavailable, with the reason, to a person who may not submit the
// activity (014 FR-027).
func limitSubmit(ctx context.Context, actions *ActionSet, facts ActivityFacts) {
	reason := submitRefusal(ctx, facts)
	if reason == "" {
		return
	}
	for _, action := range []*ActionAvailability{&actions.Submit, &actions.Resubmit} {
		if action.Available {
			*action = ActionAvailability{Available: false, Reason: reason}
		}
	}
}
