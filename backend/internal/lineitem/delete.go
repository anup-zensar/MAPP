package lineitem

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Delete moves the line item id to the recycle bin and returns the bin entry, so the web app can offer Undo.
// The activity's status never changes (D-322).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		entry, err = s.history.MoveToBin(ctx, a, lineItemKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}
