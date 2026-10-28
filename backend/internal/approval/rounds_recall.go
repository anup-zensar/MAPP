package approval

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Recall takes back the submission of a Submitted activity, as in.RoundID names its open round, with an
// optional reason.
func (s *Service) Recall(ctx context.Context, a actor.Actor, activityID string, in RecallInput) (ActivityApproval, error) {
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
		updated, err := s.recall(ctx, a, activitySubject(facts), recalling{roundID: in.RoundID, reason: reason})
		if err != nil {
			return err
		}
		sec, err := s.section(ctx, a, updated)
		approval = activityApproval(sec)
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

type recalling struct {
	roundID, reason string
}

// recall ends the open round as recalled at the level it waited at, hands the record the event, logs it and
// tells the owner and the waiting approvers.
func (s *Service) recall(ctx context.Context, a actor.Actor, sub subject, in recalling) (subject, error) {
	round, err := s.roundToRecall(ctx, sub, in.roundID)
	if err != nil {
		return subject{}, err
	}
	sub = sub.as(round.kind())
	now := s.now()
	waiting := round.Levels[round.CurrentLevel-1]
	round.Outcome, round.EndedAt = OutcomeRecalled, &now
	round.Touch(a, now)
	if err = s.repo.PutRound(ctx, round); err != nil {
		return subject{}, err
	}
	if err = s.addAction(ctx, round, Action{Action: ActionRecalled, Level: waiting.Number, Actor: a, At: now, Comment: in.reason}); err != nil {
		return subject{}, err
	}
	updated, err := s.moved(ctx, a, sub, OutcomeRecalled)
	if err != nil {
		return subject{}, err
	}
	err = s.logOn(ctx, a, updated, statusLog{
		verb: "recalled", oldStatus: sub.Status, newer: updated.Status, comment: in.reason,
		details: fmt.Sprintf("Round %d · was waiting for level %d (%s)", round.Number, waiting.Number, names(waiting.Approvers)),
	})
	if err != nil {
		return subject{}, err
	}
	recipients := append([]string{sub.OwnerID}, idsOf(waiting.Approvers)...)
	err = s.notifier(a, updated, now).send(ctx, recipients, NotifyRecalled, recalledTitle,
		fmt.Sprintf(recalledText, a.Name, subjectText(updated)))
	return updated, err
}

// roundToRecall returns the open round roundID if the person acting may recall it, or why they may not.
func (s *Service) roundToRecall(ctx context.Context, sub subject, roundID string) (Round, error) {
	if sub.Kind == KindActivity && sub.Status == StatusApproved {
		return Round{}, errs.BadRequest(approvedNoRecallMessage)
	}
	round, err := s.roundOf(ctx, sub, roundID)
	if err != nil {
		return Round{}, err
	}
	if round.Outcome != OutcomeOpen {
		return Round{}, s.movedOn(ctx, round, 0)
	}
	if refusal, _ := recallRefusal(ctx, sub, round); refusal != "" {
		return Round{}, errs.Forbidden(refusal)
	}
	return round, nil
}
