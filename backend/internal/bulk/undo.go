package bulk

import (
	"context"
	"errors"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Undo brings back every record of the person's own bulk delete id in one step, or none: the first entry
// that cannot come back refuses them all (FR-020, D-130).
func (s *Service) Undo(ctx context.Context, a actor.Actor, id string) (UndoResult, error) {
	result := UndoResult{BulkActionID: id, Restored: []history.RestoreResult{}}
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		action, found, err := s.repo.GetAction(ctx, id, a.ID)
		switch {
		case err != nil:
			return err
		case !found:
			return errs.NotFound(msgActionGone)
		case action.Kind != ActionDelete:
			return errs.BadRequest(msgUndoOnlyDelete)
		case action.UndoneAt != nil:
			return errs.AlreadyResolved(action.Note, errs.Resolution{How: "undone", By: *action.UndoneBy, At: *action.UndoneAt})
		}
		ctx = history.WithNote(ctx, "Undo of bulk delete ("+id+")")
		for _, entryID := range action.BinEntryIDs {
			restored, err := s.history.Undo(ctx, a, entryID)
			if err != nil {
				return tooLate(err)
			}
			result.Restored = append(result.Restored, restored)
		}
		return s.repo.MarkUndone(ctx, id, a, s.now())
	})
	if err != nil {
		return UndoResult{}, err
	}
	return result, nil
}

func tooLate(err error) error {
	if apiErr, ok := errors.AsType[*errs.Error](err); ok && apiErr.Code == errs.CodeUndoExpired {
		expired := errs.UndoExpired()
		expired.Message = msgUndoTooLate
		return expired
	}
	return err
}
