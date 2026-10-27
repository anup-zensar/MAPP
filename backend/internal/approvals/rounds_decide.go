package approvals

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Approve approves the current level of the activity's open round, as in.RoundID and in.Level name it
// (FR-014, FR-015; 017 FR-015, FR-017 – FR-020). Only the people of data-model §6 may: a waiting approver, a
// current delegate of one, the backup approver of an escalated level, or someone allowed to decide on
// behalf; never the owner or the submitter while 014's access rule is on (403 otherwise). The decision
// records its capacity. A level is done by its first approval when any one approves, by every approver
// when all must approve, and by the backup approver or someone deciding on behalf; a partial approval
// changes no status and tells nobody. A done level moves the round to the next level that needs a decision,
// whose approvers and their delegates are notified, and tells the owner; after the last the activity is
// Approved and the owner is told (FR-035). A level or round that has moved on is refused, saying what
// happened (FR-018).
func (s *Service) Approve(ctx context.Context, a actor.Actor, activityID string, in DecisionInput) (ActivityApproval, error) {
	return s.decide(ctx, a, activityID, in, ActionApproved)
}

// Decline declines the activity at the current level of its open round, with a comment that is not blank
// (FR-016): the round ends, the activity becomes Declined, the comment heads its Approval section, and the
// owner is told with the start of the comment (FR-035). Anyone who may decide may decline, in any capacity
// (017 FR-015). The other rules are Approve's.
func (s *Service) Decline(ctx context.Context, a actor.Actor, activityID string, in DecisionInput) (ActivityApproval, error) {
	return s.decide(ctx, a, activityID, in, ActionDeclined)
}

// decide does the work of Approve and Decline, in one Write.
func (s *Service) decide(ctx context.Context, a actor.Actor, activityID string, in DecisionInput, decision ActionKind) (ActivityApproval, error) {
	comment := strings.TrimSpace(in.Comment)
	var problems errs.Problems
	if in.RoundID == "" {
		problems.Add("roundId", reloadMessage)
	}
	if in.Level == nil {
		problems.Add("level", reloadMessage)
	}
	switch {
	case decision == ActionDeclined && comment == "":
		problems.Add("comment", declineCommentMessage)
	case tooLong(comment, maxCommentLength):
		problems.Add("comment", tooLong2000Message)
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
		round, found, err := s.repo.GetRound(ctx, in.RoundID)
		if err != nil {
			return err
		}
		if !found || round.ActivityID != facts.ID {
			return errs.NotFound(roundNotOfActivity)
		}
		if round.Outcome != OutcomeOpen || round.CurrentLevel != *in.Level {
			return s.movedOn(ctx, round, *in.Level)
		}
		all, err := s.repo.ActionsOf(ctx, facts.ID)
		if err != nil {
			return err
		}
		entries := entriesOf(all, round.ID)
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		dl, err := s.currentDelegations(ctx, d)
		if err != nil {
			return err
		}
		dc, err := s.deciderOf(ctx, a, dl)
		if err != nil {
			return err
		}
		au, err := s.authority(ctx, dc, facts, round, entries, true)
		if err != nil {
			return err
		}
		if !au.allowed {
			return au.refusal
		}

		now := s.now()
		level := round.Levels[round.CurrentLevel-1]
		entry, err := s.newAction(ctx, round, decision, level.Number, a, au.onBehalfOf, now, comment)
		if err != nil {
			return err
		}
		entry.Capacity = au.capacity
		if au.capacity == CapacityDelegate {
			entry.AsDelegateOf = &au.asDelegateOf[0]
		}
		if err := s.repo.PutAction(ctx, entry); err != nil {
			return err
		}
		suffix := capacitySuffix(au)
		decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, level.Number, len(round.Levels))

		updated := facts
		logEntry := statusLog{comment: comment, details: decided + suffix}
		notify := s.notifier(a, facts, now)
		owner := []string{facts.OwnerID}
		entries = append(entries, entry)
		next := firstDecideLevel(round.Levels, level.Number)
		switch {
		case decision == ActionDeclined:
			round.Outcome, round.EndedAt = OutcomeDeclined, &now
			logEntry.verb, logEntry.oldStatus, logEntry.newer = "declined", facts.Status, StatusDeclined
			err = notify.send(ctx, owner, NotifyDeclined, activityDeclinedTitle,
				fmt.Sprintf(declinedText, a.Name, activityText(facts), quoted(comment)))
		case !levelDone(level, entries):
			// An "all must approve" level waits for more approvals: nobody is told (017 FR-032).
			waiting := waitingOn(level, entries)
			logEntry.verb = fmt.Sprintf("approved level %d of", level.Number)
			logEntry.details = fmt.Sprintf("%s · %d of %d approved · still waiting for %s%s", decided,
				len(level.Approvers)-len(waiting), len(level.Approvers), names(waiting), suffix)
		case next == 0:
			round.Outcome, round.EndedAt = OutcomeApproved, &now
			logEntry.verb, logEntry.oldStatus, logEntry.newer = "approved", facts.Status, StatusApproved
			err = notify.send(ctx, owner, NotifyApproved, activityApprovedTitle, fmt.Sprintf(approvedText, a.Name, activityText(facts)))
		default:
			round.CurrentLevel = next
			round.Levels[next-1].StartedAt = &now
			waiting := round.Levels[next-1]
			logEntry.verb = fmt.Sprintf("approved level %d of", level.Number)
			logEntry.details = fmt.Sprintf("%s · now waiting for level %d (%s)%s", decided, next, approversText(waiting), suffix)
			// The next approvers first: an owner who approves the next level is asked to, not told.
			err = s.notifyLevel(ctx, notify, waiting, dl, fmt.Sprintf(nextLevelWaitsText, a.Name, level.Number, activityText(facts), next))
			if err == nil {
				err = notify.send(ctx, owner, NotifyLevelApproved, fmt.Sprintf(levelApprovedTitle, level.Number),
					fmt.Sprintf(levelApprovedText, a.Name, level.Number, len(round.Levels), activityText(facts)))
			}
		}
		if err != nil {
			return err
		}
		round.Touch(a, now)
		if err := s.repo.PutRound(ctx, round); err != nil {
			return err
		}
		if logEntry.newer != "" {
			if updated, err = s.activities.SetStatus(ctx, a, facts.ID, logEntry.newer); err != nil {
				return err
			}
		}
		if err := s.logOnActivity(ctx, a, updated, logEntry); err != nil {
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

// movedOn explains why a decision on level of round comes too late (FR-018, D-155): the level was decided,
// or the round ended, after the person opened the activity. Only the first decision counts, so nothing
// changes. A level the round has not reached yet is a stale request.
func (s *Service) movedOn(ctx context.Context, round Round, level int) error {
	entries, err := s.repo.ActionsOf(ctx, round.ActivityID)
	if err != nil {
		return err
	}
	mine := entriesOf(entries, round.ID)
	last, found := decisionOn(mine, level)
	if !found && round.Outcome != OutcomeOpen && len(mine) > 0 {
		last, found = mine[len(mine)-1], true // what ended the round
	}
	if !found {
		return errs.Validation(errs.FieldError{Field: "level", Message: reloadMessage})
	}
	var message string
	switch last.Action {
	case ActionApproved:
		message = fmt.Sprintf(alreadyApprovedMessage, last.Level, last.Actor.Name)
	case ActionDeclined:
		message = fmt.Sprintf(alreadyDeclinedMessage, last.Level, last.Actor.Name)
	case ActionRecalled:
		message = fmt.Sprintf(recalledByMessage, last.Actor.Name)
	case ActionRestarted:
		message = fmt.Sprintf(restartedByMessage, last.Actor.Name)
	default:
		message = cancelledMessage
	}
	return decidedConflict(message, last.Actor.Name, last.At)
}
