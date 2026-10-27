package plans

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Get returns the plan id with its calculated figures, goals and completeness (FR-011 – FR-017).
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (PlanView, error) {
	var view PlanView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
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

// stored reads the plan id inside the caller's transaction; a plan that does not exist is not_found.
func (s *Service) stored(ctx context.Context, id string) (Plan, error) {
	p, found, err := s.repo.Get(ctx, id)
	if err != nil {
		return Plan{}, err
	}
	if !found {
		return Plan{}, errs.NotFound(planGoneMessage)
	}
	return p, nil
}

// Lookup returns the stored plan id inside the caller's transaction, and false when there is none: for the
// levels below the plan (005–007), which check their parent with it.
func (s *Service) Lookup(ctx context.Context, a actor.Actor, id string) (Plan, bool, error) {
	var p Plan
	var found bool
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		p, found, err = s.repo.Get(ctx, id)
		return err
	})
	return p, found, err
}
