package activities

import (
	"context"

	"mapp/internal/platform/actor"
)

// Get returns the activity id with everything derived from it (FR-019 – FR-024).
func (s *Service) Get(ctx context.Context, a actor.Actor, id string) (View, error) {
	var view View
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, activity, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}
