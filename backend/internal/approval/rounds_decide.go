package approval

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// deciding is one decision inside decide's Write, with the round's entries, the delegations and a's authority.
type deciding struct {
	a       actor.Actor
	kind    ActionKind
	comment string
	now     time.Time
	facts   ActivityFacts
	round   Round
	entries []Action
	dl      []Delegation
	au      authority
}

// decide does the work of Approve and Decline, in one Write.
func (s *Service) decide(ctx context.Context, a actor.Actor, activityID string, in DecisionInput, decision ActionKind) (ActivityApproval, error) { //nolint:revive // argument-limit 5: Approve's and Decline's parameters, and which of the two it is
	comment := strings.TrimSpace(in.Comment)
	if err := decisionProblems(in, decision, comment); err != nil {
		return ActivityApproval{}, err
	}

	var approval ActivityApproval
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		round, err := s.roundToDecide(ctx, facts, in)
		if err != nil {
			return err
		}
		dg, err := s.decidingOn(ctx, a, facts, round)
		if err != nil {
			return err
		}
		dg.kind, dg.comment, dg.now = decision, comment, s.now()
		updated, err := s.recordDecision(ctx, dg)
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

func decisionProblems(in DecisionInput, decision ActionKind, comment string) error {
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
	return problems.Err()
}

// roundOf returns the round roundID of the activity: not_found when there is none or it is another's.
func (s *Service) roundOf(ctx context.Context, facts ActivityFacts, roundID string) (Round, error) {
	round, found, err := s.repo.GetRound(ctx, roundID)
	if err != nil {
		return Round{}, err
	}
	if !found || round.ActivityID != facts.ID {
		return Round{}, errs.NotFound(roundNotOfActivity)
	}
	return round, nil
}

// roundToDecide returns the round in.RoundID if it is open at in.Level, or why a decision comes too late.
func (s *Service) roundToDecide(ctx context.Context, facts ActivityFacts, in DecisionInput) (Round, error) {
	round, err := s.roundOf(ctx, facts, in.RoundID)
	if err != nil {
		return Round{}, err
	}
	if round.Outcome != OutcomeOpen || round.CurrentLevel != *in.Level {
		return Round{}, s.movedOn(ctx, round, *in.Level)
	}
	return round, nil
}

// decidingOn reads what a's decision on the current level of round needs, or a's refusal.
func (s *Service) decidingOn(ctx context.Context, a actor.Actor, facts ActivityFacts, round Round) (deciding, error) {
	all, err := s.repo.ActionsOf(ctx, facts.ID)
	if err != nil {
		return deciding{}, err
	}
	entries := entriesOf(all, round.ID)
	d, err := s.directory(ctx)
	if err != nil {
		return deciding{}, err
	}
	dl, err := s.currentDelegations(ctx, d)
	if err != nil {
		return deciding{}, err
	}
	dc, err := s.deciderOf(ctx, a, dl)
	if err != nil {
		return deciding{}, err
	}
	au, err := s.authority(ctx, dc, facts, round, entries, true)
	if err != nil {
		return deciding{}, err
	}
	if !au.allowed {
		return deciding{}, au.refusal
	}
	return deciding{a: a, facts: facts, round: round, entries: entries, dl: dl, au: au}, nil
}

// recordDecision stores dg's decision, the round after it and the activity's log, and returns the activity.
func (s *Service) recordDecision(ctx context.Context, dg deciding) (ActivityFacts, error) {
	entry, err := s.decisionEntry(ctx, dg)
	if err != nil {
		return ActivityFacts{}, err
	}
	dg.entries = append(dg.entries, entry)
	round, logEntry, err := s.decisionOutcome(ctx, dg)
	if err != nil {
		return ActivityFacts{}, err
	}
	round.Touch(dg.a, dg.now)
	if err = s.repo.PutRound(ctx, round); err != nil {
		return ActivityFacts{}, err
	}
	updated := dg.facts
	if logEntry.newer != "" {
		if updated, err = s.activities.SetStatus(ctx, dg.a, dg.facts.ID, logEntry.newer); err != nil {
			return ActivityFacts{}, err
		}
	}
	if err = s.logOnActivity(ctx, dg.a, updated, logEntry); err != nil {
		return ActivityFacts{}, err
	}
	return updated, nil
}

func (s *Service) decisionEntry(ctx context.Context, dg deciding) (Action, error) {
	level := dg.round.Levels[dg.round.CurrentLevel-1]
	entry, err := s.newAction(ctx, dg.round, Action{
		Action: dg.kind, Level: level.Number, Actor: dg.a, OnBehalfOf: dg.au.onBehalfOf, At: dg.now, Comment: dg.comment,
	})
	if err != nil {
		return Action{}, err
	}
	entry.Capacity = dg.au.capacity
	if dg.au.capacity == CapacityDelegate {
		entry.AsDelegateOf = &dg.au.asDelegateOf[0]
	}
	if err = s.repo.PutAction(ctx, entry); err != nil {
		return Action{}, err
	}
	return entry, nil
}

// decisionOutcome moves the round on or ends it and tells who needs to know, returning the round and log entry.
func (s *Service) decisionOutcome(ctx context.Context, dg deciding) (Round, statusLog, error) {
	round, facts, a, now := dg.round, dg.facts, dg.a, dg.now
	level := round.Levels[round.CurrentLevel-1]
	suffix := capacitySuffix(dg.au)
	decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, level.Number, len(round.Levels))

	logEntry := statusLog{comment: dg.comment, details: decided + suffix}
	notify := s.notifier(a, facts, now)
	owner := []string{facts.OwnerID}
	next := firstDecideLevel(round.Levels, level.Number)
	var err error
	switch {
	case dg.kind == ActionDeclined:
		round.Outcome, round.EndedAt = OutcomeDeclined, &now
		logEntry.verb, logEntry.oldStatus, logEntry.newer = "declined", facts.Status, StatusDeclined
		err = notify.send(ctx, owner, NotifyDeclined, activityDeclinedTitle,
			fmt.Sprintf(declinedText, a.Name, activityText(facts), quoted(dg.comment)))
	case !levelDone(level, dg.entries):
		// An "all must approve" level waits for more approvals: nobody is told (017 FR-032).
		waiting := waitingOn(level, dg.entries)
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
		err = notify.sendToLevel(ctx, waiting, dg.dl, fmt.Sprintf(nextLevelWaitsText, a.Name, level.Number, activityText(facts), next))
		if err == nil {
			err = notify.send(ctx, owner, NotifyLevelApproved, fmt.Sprintf(levelApprovedTitle, level.Number),
				fmt.Sprintf(levelApprovedText, a.Name, level.Number, len(round.Levels), activityText(facts)))
		}
	}
	return round, logEntry, err
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
	return decidedConflict(movedOnMessage(last), last.Actor.Name, last.At)
}

func movedOnMessage(last Action) string {
	switch last.Action { //nolint:exhaustive // any other entry gets the deletion's message, as before
	case ActionApproved:
		return fmt.Sprintf(alreadyApprovedMessage, last.Level, last.Actor.Name)
	case ActionDeclined:
		return fmt.Sprintf(alreadyDeclinedMessage, last.Level, last.Actor.Name)
	case ActionRecalled:
		return fmt.Sprintf(recalledByMessage, last.Actor.Name)
	case ActionRestarted:
		return fmt.Sprintf(restartedByMessage, last.Actor.Name)
	default:
		return cancelledMessage
	}
}
