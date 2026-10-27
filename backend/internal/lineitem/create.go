package lineitem

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Create adds a line item to the activity activityID (FR-009, FR-010): every problem of the form at once;
// on a Submitted or Approved activity only with in.RestartApproval, which then sends the activity back to
// Created (FR-031); a new ID L-…; one "created" log entry; all in one Write.
func (s *Service) Create(ctx context.Context, a actor.Actor, activityID string, in Input) (View, error) { //nolint:gocyclo // 10: one Write whose every step returns its error.
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
		if err = confirmRestart(problems, f, in.RestartApproval); err != nil {
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
		if f, err = s.restart(ctx, a, f, fmt.Sprintf(addedReason, li.Name, li.ID, li.PlannedCost.Format(), li.Currency)); err != nil {
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
