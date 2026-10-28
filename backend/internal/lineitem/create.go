package lineitem

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Create adds a line item to the activity activityID: every problem of the form at once; a new ID L-…; one
// "created" log entry; all in one Write. The activity's status never changes, whatever it is (D-322).
func (s *Service) Create(ctx context.Context, a actor.Actor, activityID string, in Input) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		siblings, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		li, problems, err := s.check(ctx, f, siblings, in, nil)
		if err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}
		if li.ID, err = s.ids.Next(ctx, ids.PrefixLineItem); err != nil {
			return err
		}
		li.ActivityID, li.Meta = f.ID, record.NewMeta(a, s.now())
		if err = s.repo.Put(ctx, li); err != nil {
			return err
		}
		if _, err = s.history.Log(ctx, a, history.LogInput{
			Kind: lineItemKind, RecordID: li.ID, RecordName: li.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.shownView(ctx, a, f, li)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return view, nil
}
