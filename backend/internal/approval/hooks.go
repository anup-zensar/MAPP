package approval

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// ApprovalRestarted ends the approval of an activity that a re-approval change (domain model §2.5) sent back
// to Created from previousStatus, inside the caller's Write; it writes no log entry.
func (s *Service) ApprovalRestarted(ctx context.Context, a actor.Actor, activityID, previousStatus string, changes []history.Change) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		rounds, err := s.repo.RoundsOf(ctx, activityID)
		if err != nil {
			return err
		}
		now := s.now()

		var round Round
		var level int
		var approvers []string
		switch previousStatus {
		case StatusSubmitted:
			open, found := openRound(rounds)
			if !found {
				return nil
			}
			waiting := open.Levels[open.CurrentLevel-1]
			open.Outcome, open.EndedAt = OutcomeRestarted, &now
			open.Touch(a, now)
			if err := s.repo.PutRound(ctx, open); err != nil {
				return err
			}
			round, level, approvers = open, waiting.Number, idsOf(waiting.Approvers)
		case StatusApproved:
			approved, found := lastApproved(rounds)
			if !found {
				return nil
			}
			entries, err := s.repo.ActionsOf(ctx, activityID)
			if err != nil {
				return err
			}
			for _, e := range entriesOf(entries, approved.ID) {
				if e.Action == ActionApproved {
					approvers = append(approvers, e.Actor.ID)
				}
			}
			round = approved
		default:
			return nil
		}

		entry, err := s.newAction(ctx, round, Action{Action: ActionRestarted, Level: level, Actor: a, At: now})
		if err != nil {
			return err
		}
		for _, c := range changes {
			entry.Changes = append(entry.Changes, ChangeText{Label: c.Label, OldText: c.OldText, NewText: c.NewText})
		}
		if err := s.repo.PutAction(ctx, entry); err != nil {
			return err
		}
		recipients := append([]string{facts.OwnerID}, approvers...)
		sub := activitySubject(facts)
		return s.notifier(a, sub, now).send(ctx, recipients, NotifyRestarted, restartedTitle,
			fmt.Sprintf(restartedText, a.Name, subjectText(sub)))
	})
}

// ActivityRestored ends, as cancelled, the round that was open when the activity was deleted.
func (s *Service) ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) ([]string, error) {
	var notes []string
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		open, found, err := s.openRoundOf(ctx, activityID)
		if err != nil || !found {
			return err
		}
		deletedAt := rc.DeletedAt
		open.Outcome, open.EndedAt = OutcomeCancelled, &deletedAt
		open.Touch(rc.Actor, rc.Now)
		if err := s.repo.PutRound(ctx, open); err != nil {
			return err
		}
		if err = s.addAction(ctx, open, Action{Action: ActionCancelled, Level: open.CurrentLevel, Actor: rc.DeletedBy, At: rc.DeletedAt}); err != nil {
			return err
		}
		notes = []string{fmt.Sprintf(cancelledNote, open.Number)}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *Service) openRoundOf(ctx context.Context, activityID string) (Round, bool, error) {
	rounds, err := s.repo.OpenRounds(ctx)
	if err != nil {
		return Round{}, false, err
	}
	open, found := openRound(where(rounds, func(r Round) bool { return r.SubjectID == activityID }))
	return open, found, nil
}

func lastApproved(rounds []Round) (Round, bool) {
	for i := len(rounds) - 1; i >= 0; i-- {
		if rounds[i].Outcome == OutcomeApproved {
			return rounds[i], true
		}
	}
	return Round{}, false
}
