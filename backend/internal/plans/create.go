package plans

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Create creates a plan from the whole form in one Write (FR-010): every problem is reported at once and
// nothing is saved while there is one; otherwise the plan is Draft with a new permanent ID and a "created"
// log entry. Only name, year and model block saving; what is missing for publishing does not.
func (s *Service) Create(ctx context.Context, a actor.Actor, input PlanInput) (PlanView, error) {
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		var problems errs.Problems
		p, err := s.check(ctx, formOf(input), nil, &problems)
		if err != nil {
			return err
		}
		checkTeams(ctx, authz.PlansCreate, nil, authz.PlanFacts{TeamIDs: p.TeamIDs, CreatedByID: a.ID}, &problems) // 014 FR-013
		if err := problems.Err(); err != nil {
			return err
		}
		if p.ID, err = s.ids.Next(ctx, ids.PrefixPlan); err != nil {
			return err
		}
		p.Status = StatusDraft
		p.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, p); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: p.ID, RecordName: p.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.view(ctx, a, p)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}
