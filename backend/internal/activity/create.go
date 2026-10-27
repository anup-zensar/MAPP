package activity

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Create creates an activity under the initiative initiativeID with the status Created (FR-003, FR-004):
// only the name and the type are required; every problem is reported at once (FR-006 – FR-018). It writes
// a "created" log entry in the same Write.
func (s *Service) Create(ctx context.Context, a actor.Actor, initiativeID string, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		pc, found, err := s.contextOf(ctx, initiativeID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		var problems errs.Problems
		activity, err := s.checkFields(ctx, &problems, fieldCheck{pc: pc, in: in})
		if err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}
		if activity.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		activity.Status, activity.Meta = StatusCreated, record.NewMeta(a, s.now())
		if err = s.repo.Put(ctx, activity); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: activity.ID, RecordName: activity.Name, Action: history.ActionCreated,
		})
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
