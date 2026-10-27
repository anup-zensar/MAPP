package campaigns

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// DeleteCampaign moves the campaign id, with its initiatives and everything below them, to the recycle bin
// (FR-028), and returns the recycle-bin entry so that the person can undo the delete (FR-029).
func (s *Service) DeleteCampaign(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetCampaign(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(campaignGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, campaignKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// DeleteInitiative moves the initiative id, with its activities and line items, to the recycle bin
// (FR-028), and returns the recycle-bin entry so that the person can undo the delete (FR-029).
func (s *Service) DeleteInitiative(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, initiativeKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}
