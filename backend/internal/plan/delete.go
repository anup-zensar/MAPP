package plan

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Delete moves the plan id, with everything in it (the kinds of 005–007 registered below "plan"), to the
// recycle bin as one entry, in one Write; history logs one "deleted" entry naming what it held (FR-029).
// It returns the recycle-bin entry, so the person can undo the delete (FR-030).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if _, err := s.stored(ctx, id); err != nil {
			return err
		}
		var err error
		entry, err = s.history.MoveToBin(ctx, a, kindName, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}
