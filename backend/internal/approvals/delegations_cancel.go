package approvals

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

// CancelDelegation cancels the delegation id at once (017 FR-024): the delegate loses the rights it gave;
// decisions already made stay recorded. The rules are UpdateDelegation's. It is logged as cancelled.
func (s *Service) CancelDelegation(ctx context.Context, a actor.Actor, id string, cancel DelegationCancel) (DelegationView, error) {
	var view DelegationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, manage, err := s.changeableDelegation(ctx, a, id, cancel.Version)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		now := s.now()
		changed := stored
		changed.CancelledAt, changed.CancelledBy = &now, &a
		changes, err := history.Diff(stored.logView(), changed.logView())
		if err != nil {
			return err
		}
		changed.Touch(a, now)
		if err = s.repo.PutDelegation(ctx, changed); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: delegationKind, RecordID: changed.ID, RecordName: delegationName(changed, d), Action: history.ActionStatusChanged,
			Verb: "cancelled", Changes: changes,
		})
		if err != nil {
			return err
		}
		view = delegationView(changed, d, a, manage, date.Today(now))
		return nil
	})
	if err != nil {
		return DelegationView{}, err
	}
	return view, nil
}
