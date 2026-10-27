package approval

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
func (s *Service) Recall(ctx context.Context, a actor.Actor, activityID string, in RecallInput) (ActivityApproval, error) { //nolint:gocyclo // 10: one Write whose every step returns its error.
	reason := strings.TrimSpace(in.Reason)
	if err := recallProblems(in, reason); err != nil {
		return ActivityApproval{}, err
	}

	var approval ActivityApproval
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		round, err := s.roundToRecall(ctx, facts, in.RoundID)
		if err != nil {
			return err
		}

		now := s.now()
		waiting := round.Levels[round.CurrentLevel-1]
		round.Outcome, round.EndedAt = OutcomeRecalled, &now
		round.Touch(a, now)
		if err = s.repo.PutRound(ctx, round); err != nil {
			return err
		}
		if err = s.addAction(ctx, round, ActionRecalled, waiting.Number, a, nil, now, reason); err != nil {
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

func recallProblems(in RecallInput, reason string) error {
	var problems errs.Problems
	if in.RoundID == "" {
		problems.Add("roundId", reloadMessage)
	}
	if tooLong(reason, maxCommentLength) {
		problems.Add("reason", tooLong2000Message)
	}
	return problems.Err()
}

// roundToRecall returns the open round roundID if the person acting may recall it, or why they may not.
func (s *Service) roundToRecall(ctx context.Context, facts ActivityFacts, roundID string) (Round, error) {
	if facts.Status == StatusApproved {
		return Round{}, errs.BadRequest(approvedNoRecallMessage)
	}
	round, err := s.roundOf(ctx, facts, roundID)
	if err != nil {
		return Round{}, err
	}
	if round.Outcome != OutcomeOpen {
		return Round{}, s.movedOn(ctx, round, 0) // no level is decided by a recall: what ended the round explains it
	}
	if refusal, _ := recallRefusal(ctx, facts, round); refusal != "" { // 014 FR-027
		return Round{}, errs.Forbidden(refusal)
	}
	return round, nil
}
