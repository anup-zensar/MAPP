package purchaserequest

import (
	"context"
	"slices"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

var _ approval.PurchaseRequests = (*Service)(nil)

func (s *Service) PurchaseRequest(ctx context.Context, id string) (approval.PurchaseRequestFacts, bool, error) {
	f, found, err := s.factsByID(ctx, id)
	return engineFacts(f), found, err
}

// ListInScope returns the purchase requests among ids that the engine's plans scope or the person's
// purchaseRequests.list scope sees.
func (s *Service) ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]approval.PurchaseRequestFacts, error) {
	list, err := s.factsInScope(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	facts := make([]approval.PurchaseRequestFacts, 0, len(list))
	for _, f := range list {
		facts = append(facts, engineFacts(f))
	}
	return facts, nil
}

func (s *Service) ReadyCheck(ctx context.Context, a actor.Actor, id string, kind approval.SubjectKind) (approval.ReadyCheck, error) {
	problems, err := s.readyCheck(ctx, a, id, RoundKind(kind))
	if err != nil {
		return approval.ReadyCheck{}, err
	}
	check := approval.ReadyCheck{Problems: make([]approval.SubmitProblem, 0, len(problems))}
	for _, p := range problems {
		group := p.Group
		if group == groupLine {
			group = approval.GroupPurchaseRequestLine
		}
		check.Problems = append(check.Problems, approval.SubmitProblem{Field: p.Field, Group: group, Message: p.Message})
	}
	return check, nil
}

// RoundChanged sets the status for a round event (data-model §3.5); the engine logs the event itself.
//
//nolint:revive // argument-limit 5: the port's signature, declared by approval (031 contracts/go-api.md)
func (s *Service) RoundChanged(ctx context.Context, a actor.Actor, id string, kind approval.SubjectKind, outcome approval.Outcome) (approval.PurchaseRequestFacts, error) {
	f, err := s.roundChanged(ctx, a, id, roundEvent{kind: RoundKind(kind), event: RoundEvent(outcome)})
	return engineFacts(f), err
}

func overagesOf(pr PurchaseRequest, facts map[string]LineItemFacts) []approval.Overage {
	overages := []approval.Overage{}
	for _, l := range pr.Lines {
		f := facts[l.LineItemID]
		seen := slices.ContainsFunc(overages, func(o approval.Overage) bool { return o.Activity.ID == f.ActivityID })
		if f.ActivityOverBy != nil && !seen {
			overages = append(overages, approval.Overage{
				Activity: errs.RecordRef{Type: groupActivity, ID: f.ActivityID, Name: f.ActivityName}, OverBy: *f.ActivityOverBy})
		}
	}
	return overages
}

func engineFacts(f Facts) approval.PurchaseRequestFacts {
	var amount *money.Money
	if f.Amount != nil {
		usd := money.USD(*f.Amount)
		amount = &usd
	}
	return approval.PurchaseRequestFacts{
		ID: f.ID, Name: f.Name, Status: string(f.Status), OwnerID: f.OwnerID, MoAApproverID: f.MoAApproverID,
		Amount: amount, WorkflowID: f.WorkflowID, ModificationWorkflowID: f.ModificationWorkflowID, Path: f.Path,
		Meta: f.Meta, Access: f.Plan, Overages: f.Overages,
		Plan: approval.PlanFacts{ID: f.Plan.ID, TeamIDs: f.Plan.TeamIDs, OwnerID: f.Plan.OwnerID, CreatedByID: f.Plan.CreatedByID},
	}
}
