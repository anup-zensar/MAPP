package approval

import (
	"context"
	"fmt"

	"mapp/internal/platform/authz"
)

const (
	msgCannotApprove          = "%s can't approve activities. Give them a role that approves first."
	msgRecallOwnerOrSubmitter = "Only the %s's owner or the person who submitted it can recall this %s."
)

type Approvers interface {
	CanApprove(ctx context.Context, personIDs []string) (map[string]bool, error)
}

func (s *Service) UseApprovers(a Approvers) {
	s.approvers = a
}

// recallRefusal is why the person may not recall, "" when they may; shown false hides Recall.
func recallRefusal(ctx context.Context, sub subject, round Round) (refusal string, shown bool) {
	g, restricted := authz.From(ctx)
	if !restricted || g.InScope(authz.ApprovalsDecideOnBehalf, sub.Access) {
		return "", false
	}
	if sub.OwnerID != g.PersonID && round.SubmittedBy.ID != g.PersonID {
		return fmt.Sprintf(msgRecallOwnerOrSubmitter, sub.Kind.word(), sub.Kind.word()), false
	}
	if sub.Kind == KindActivity && !g.InScope(authz.ActivitiesSubmit, sub.Access) {
		return g.Refusal(authz.ActivitiesSubmit, &sub.Access), true
	}
	return "", false
}

func limitDecisions(ctx context.Context, actions *ActionSet, sub subject, open Round) {
	if refusal, shown := recallRefusal(ctx, sub, open); refusal != "" && actions.Recall.Available {
		reason := ""
		if shown {
			reason = refusal
		}
		actions.Recall = ActionAvailability{Available: false, Reason: reason}
	}
}

// limitSubmit applies 014's submit permission to activities; feature 032's routes guard purchase requests.
func limitSubmit(ctx context.Context, actions *ActionSet, sub subject) {
	if sub.Kind != KindActivity {
		return
	}
	reason := authz.Available(ctx, authz.ActivitiesSubmit, sub.Access).Reason
	if reason == "" {
		return
	}
	for _, action := range []*ActionAvailability{&actions.Submit, &actions.Resubmit} {
		if action.Available {
			*action = ActionAvailability{Available: false, Reason: reason}
		}
	}
}
