package activity

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Delete moves the activity id, with its line items, to the recycle bin (FR-031) and returns the
// recycle-bin entry, so that the person can undo it. Feature 008 is told first.
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		activity, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(activityGoneMessage)
		}
		if err = s.hooks.ActivityDeleting(ctx, a, activity); err != nil {
			return err
		}
		entry, err = s.history.MoveToBin(ctx, a, kindName, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}
