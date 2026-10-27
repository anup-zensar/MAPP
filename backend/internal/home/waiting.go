package home

import (
	"context"

	"mapp/internal/platform/actor"
)

// maxWaitingRows is how many activities Waiting for my approval shows (FR-013).
const maxWaitingRows = 5

// Waiting returns Waiting for my approval (FR-013): the first 5 activities of the Waiting for me tab of the
// Approvals page — the same list, oldest first — and how many wait in all, with their planned cost in a's
// display currency.
func (s *Service) Waiting(ctx context.Context, a actor.Actor) (WaitingPage, error) {
	var page WaitingPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		waiting, err := s.src.WaitingForMe(ctx, a)
		if err != nil {
			return err
		}
		shown := waiting[:min(len(waiting), maxWaitingRows)]
		page = WaitingPage{Items: make([]WaitingItem, 0, len(shown)), Total: len(waiting)}
		for _, w := range shown {
			page.Items = append(page.Items, WaitingItem{
				ID: w.ActivityID, Name: w.ActivityName, Plan: w.Plan, Owner: w.Owner, PlannedCost: c.optional(w.PlannedCost),
				Level: w.Level, Levels: w.Levels, WaitingSince: w.WaitingSince,
			})
		}
		return nil
	})
	if err != nil {
		return WaitingPage{}, err
	}
	return page, nil
}
