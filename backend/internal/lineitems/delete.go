package lineitems

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Delete moves the line item id to the recycle bin and returns the bin entry, so the web app can offer Undo
// (FR-030). On a Submitted or Approved activity it needs restartApproval, and sends the activity back to
// Created (FR-031).
func (s *Service) Delete(ctx context.Context, a actor.Actor, id string, restartApproval bool) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		li, found, err := s.repo.Get(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return s.gone(ctx, a, id)
		}
		f, err := s.facts(ctx, li.ActivityID)
		if err != nil {
			return err
		}
		if err = confirmRestart(nil, f, restartApproval); err != nil {
			return err
		}
		if entry, err = s.history.MoveToBin(ctx, a, lineItemKind, id); err != nil {
			return err
		}
		_, err = s.restart(ctx, a, f, fmt.Sprintf(deletedReason, li.Name, li.ID, li.PlannedCost.Format(), li.Currency))
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}
