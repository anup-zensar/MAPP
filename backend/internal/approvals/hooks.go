package approvals

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// The two events of features 006 and 007 that change an activity's approval. internal/app passes them on
// from activities.ApprovalHooks (approvals_adapter.go): approvals never imports activities. Both run inside
// the caller's Write, which ctx carries, so the activity's change and its approval's end are saved
// together, or neither is.

// ApprovalRestarted ends the approval of an activity that a re-approval change (domain model §2.5) sent
// back from Submitted or Approved (previousStatus) to Created (FR-022). Features 006 and 007 call it after
// they saved the change with the person's confirmation, set the status and logged both, so it writes no
// log entry. changes are the changed values of that save, with their labels and texts.
//
// A Submitted activity's open round ends as restarted at the level it waited at; an Approved activity's
// last approved round keeps its outcome and gets a restart entry without a level (data model §2). Either
// entry holds the changes (old → new). The owner is told, and the approvers of the waiting level or, for an
// Approved activity, the people who approved that round (FR-035, D-161) — never a about their own change
// (FR-036). An activity without such a round is left alone.
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
		var approvers []string // the people told besides the owner
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
			round = approved // it stays approved: no level waits, so the entry has none
		default:
			return nil
		}

		entry, err := s.newAction(ctx, round, ActionRestarted, level, a, nil, now, "")
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
		return s.notifier(a, facts, now).send(ctx, recipients, NotifyRestarted, restartedTitle,
			fmt.Sprintf(restartedText, a.Name, activityText(facts)))
	})
}

// ActivityRestored ends the round that was open when the activity was deleted, now that a restore from the
// recycle bin brings the activity back Created (FR-023, R14). Feature 006 calls it from the activity kind's
// PrepareRestore, never for an Undo, which puts the activity back Submitted with its open round (D-031).
// The round ends as cancelled at the level it waited at, dated at the delete and attributed to the person
// who deleted the activity (008-P3); nobody is told. It returns the note for the "restored" log entry, or
// none when no round was open.
func (s *Service) ActivityRestored(ctx context.Context, rc history.RestoreContext, activityID string) ([]string, error) {
	var notes []string
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		rounds, err := s.repo.OpenRounds(ctx)
		if err != nil {
			return err
		}
		open, found := openRound(where(rounds, func(r Round) bool { return r.ActivityID == activityID }))
		if !found {
			return nil
		}
		deletedAt := rc.DeletedAt
		open.Outcome, open.EndedAt = OutcomeCancelled, &deletedAt
		open.Touch(rc.Actor, rc.Now)
		if err := s.repo.PutRound(ctx, open); err != nil {
			return err
		}
		if err := s.addAction(ctx, open, ActionCancelled, open.CurrentLevel, rc.DeletedBy, nil, rc.DeletedAt, ""); err != nil {
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

// lastApproved returns the approved round with the highest number among rounds, which are ordered by
// number, and false when none was approved.
func lastApproved(rounds []Round) (Round, bool) {
	for i := len(rounds) - 1; i >= 0; i-- {
		if rounds[i].Outcome == OutcomeApproved {
			return rounds[i], true
		}
	}
	return Round{}, false
}
