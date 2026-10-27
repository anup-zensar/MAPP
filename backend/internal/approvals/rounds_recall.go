package approvals

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Recall takes back the submission of a Submitted activity, as in.RoundID names its open round, with an
// optional reason (FR-021): the round ends as recalled at the level it waited at, the activity is Created
// again and leaves the approvers' lists, the round's history and the activity log record it with the reason
// (FR-026), and the owner and the approvers of the waiting level are told (FR-035). The owner and the
// round's submitter recall with "Submit activities for approval and recall them" in the plan, others only
// on behalf of the approvers (014 FR-027). An Approved activity cannot be recalled — a change of domain model §2.5 sends it back
// for approval — and the refusal says so. A round that ended meanwhile is refused, saying what ended it
// (FR-018).
func (s *Service) Recall(ctx context.Context, a actor.Actor, activityID string, in RecallInput) (ActivityApproval, error) {
	reason := strings.TrimSpace(in.Reason)
	var problems errs.Problems
	if in.RoundID == "" {
		problems.Add("roundId", reloadMessage)
	}
	if tooLong(reason, maxCommentLength) {
		problems.Add("reason", tooLong2000Message)
	}
	if err := problems.Err(); err != nil {
		return ActivityApproval{}, err
	}

	var approval ActivityApproval
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		if facts.Status == StatusApproved {
			return errs.BadRequest(approvedNoRecallMessage)
		}
		round, found, err := s.repo.GetRound(ctx, in.RoundID)
		if err != nil {
			return err
		}
		if !found || round.ActivityID != facts.ID {
			return errs.NotFound(roundNotOfActivity)
		}
		if round.Outcome != OutcomeOpen {
			return s.movedOn(ctx, round, 0) // no level is decided by a recall: what ended the round explains it
		}
		if refusal, _ := recallRefusal(ctx, facts, round); refusal != "" { // 014 FR-027
			return errs.Forbidden(refusal)
		}

		now := s.now()
		waiting := round.Levels[round.CurrentLevel-1]
		round.Outcome, round.EndedAt = OutcomeRecalled, &now
		round.Touch(a, now)
		if err := s.repo.PutRound(ctx, round); err != nil {
			return err
		}
		if err := s.addAction(ctx, round, ActionRecalled, waiting.Number, a, nil, now, reason); err != nil {
			return err
		}
		updated, err := s.activities.SetStatus(ctx, a, facts.ID, StatusCreated)
		if err != nil {
			return err
		}
		err = s.logOnActivity(ctx, a, updated, statusLog{
			verb: "recalled", oldStatus: facts.Status, newer: StatusCreated, comment: reason,
			details: fmt.Sprintf("Round %d · was waiting for level %d (%s)", round.Number, waiting.Number, names(waiting.Approvers)),
		})
		if err != nil {
			return err
		}
		recipients := append([]string{facts.OwnerID}, idsOf(waiting.Approvers)...)
		err = s.notifier(a, updated, now).send(ctx, recipients, NotifyRecalled, recalledTitle,
			fmt.Sprintf(recalledText, a.Name, activityText(updated)))
		if err != nil {
			return err
		}
		approval, err = s.approval(ctx, a, updated)
		return err
	})
	if err != nil {
		return ActivityApproval{}, err
	}
	return approval, nil
}
