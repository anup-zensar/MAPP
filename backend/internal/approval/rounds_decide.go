package approval

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Approve approves the current level of the activity's open round, as in names it.
func (s *Service) Approve(ctx context.Context, a actor.Actor, activityID string, in DecisionInput) (ActivityApproval, error) {
	return s.decideActivity(ctx, a, activityID, SubjectDecision{Action: ActionApproved, DecisionInput: in})
}

func (s *Service) Decline(ctx context.Context, a actor.Actor, activityID string, in DecisionInput) (ActivityApproval, error) {
	return s.decideActivity(ctx, a, activityID, SubjectDecision{Action: ActionDeclined, DecisionInput: in})
}

func (s *Service) decideActivity(ctx context.Context, a actor.Actor, activityID string, decision SubjectDecision) (ActivityApproval, error) {
	sec, err := s.DecideSubject(ctx, a, KindActivity, activityID, decision)
	if err != nil {
		return ActivityApproval{}, err
	}
	return activityApproval(sec), nil
}

// deciding is one decision inside its Write, with the round's entries, the delegations and a's authority.
type deciding struct {
	a       actor.Actor
	kind    ActionKind
	comment string
	now     time.Time
	sub     subject
	round   Round
	entries []Action
	dl      []Delegation
	au      authority
}

func decisionProblems(decision SubjectDecision, comment string, kind SubjectKind) error {
	var problems errs.Problems
	if decision.Action != ActionApproved && decision.Action != ActionDeclined {
		problems.Add("action", reloadMessage)
	}
	if decision.RoundID == "" {
		problems.Add("roundId", reloadMessage)
	}
	if decision.Level == nil {
		problems.Add("level", reloadMessage)
	}
	switch {
	case decision.Action == ActionDeclined && comment == "":
		problems.Add("comment", fmt.Sprintf(declineCommentMessage, kind.word()))
	case tooLong(comment, maxCommentLength):
		problems.Add("comment", tooLong2000Message)
	}
	return problems.Err()
}

func (s *Service) roundOf(ctx context.Context, sub subject, roundID string) (Round, error) {
	round, found, err := s.repo.GetRound(ctx, roundID)
	if err != nil {
		return Round{}, err
	}
	if !found || round.SubjectID != sub.ID {
		return Round{}, errs.NotFound(fmt.Sprintf(roundNotOfMessage, sub.Kind.word()))
	}
	return round, nil
}

// mayDecide refuses a person holding neither approvals permission in the plan before any word on the round.
func (s *Service) mayDecide(ctx context.Context, a actor.Actor, sub subject) error {
	decide, err := s.access.Check(ctx, a.ID, PermissionDecide, &sub.Plan)
	if err != nil || decide.Allowed {
		return err
	}
	onBehalf, err := s.access.Check(ctx, a.ID, PermissionDecideOnBehalf, &sub.Plan)
	if err != nil || onBehalf.Allowed {
		return err
	}
	return errs.Forbidden(decide.Refusal)
}

func (s *Service) roundToDecide(ctx context.Context, sub subject, in DecisionInput) (Round, error) {
	round, err := s.roundOf(ctx, sub, in.RoundID)
	if err != nil {
		return Round{}, err
	}
	if round.Outcome != OutcomeOpen || round.CurrentLevel != *in.Level {
		return Round{}, s.movedOn(ctx, round, *in.Level)
	}
	return round, nil
}

// decidingOn reads what a's decision on the current level of round needs, or a's refusal.
func (s *Service) decidingOn(ctx context.Context, a actor.Actor, sub subject, round Round) (deciding, error) {
	all, err := s.repo.ActionsOf(ctx, sub.ID)
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
	au, err := s.authority(ctx, dc, decisionPoint{sub: sub, round: round, entries: entries, onBehalf: true})
	if err != nil {
		return deciding{}, err
	}
	if !au.allowed {
		return deciding{}, au.refusal
	}
	return deciding{a: a, sub: sub, round: round, entries: entries, dl: dl, au: au}, nil
}

// recordDecision stores dg's decision, the round after it, the record's change and its log entry.
func (s *Service) recordDecision(ctx context.Context, dg deciding) (subject, error) {
	entry, err := s.decisionEntry(ctx, dg)
	if err != nil {
		return subject{}, err
	}
	dg.entries = append(dg.entries, entry)
	round, logEntry, event, err := s.decisionOutcome(ctx, dg)
	if err != nil {
		return subject{}, err
	}
	if dg.kind == ActionApproved {
		logEntry.details += overageDetails(dg.sub)
	}
	round.Touch(dg.a, dg.now)
	if err = s.repo.PutRound(ctx, round); err != nil {
		return subject{}, err
	}
	updated := dg.sub
	if event != "" {
		if updated, err = s.moved(ctx, dg.a, dg.sub, event); err != nil {
			return subject{}, err
		}
	}
	logEntry.oldStatus, logEntry.newer = dg.sub.Status, updated.Status
	return updated, s.logOn(ctx, dg.a, updated, logEntry)
}

func (s *Service) decisionEntry(ctx context.Context, dg deciding) (Action, error) {
	level := dg.round.Levels[dg.round.CurrentLevel-1]
	entry, err := s.newAction(ctx, dg.round, Action{
		Action: dg.kind, Level: level.Number, Actor: dg.a, OnBehalfOf: dg.au.onBehalfOf, At: dg.now, Comment: dg.comment,
	})
	if err != nil {
		return Action{}, err
	}
	entry.Capacity, entry.SelfApproval = dg.au.capacity, selfApproval(dg)
	if dg.au.capacity == CapacityDelegate {
		entry.AsDelegateOf = &dg.au.asDelegateOf[0]
	}
	if err = s.repo.PutAction(ctx, entry); err != nil {
		return Action{}, err
	}
	return entry, nil
}

// selfApproval reports a decision on a purchase request by its owner or the round's submitter.
func selfApproval(dg deciding) bool {
	self := dg.a.ID == dg.sub.OwnerID || dg.a.ID == dg.round.SubmittedBy.ID
	return dg.sub.Kind.isPurchaseRequest() && self
}

// decisionOutcome moves the round on or ends it and tells who needs to know; it returns the round, the log
// entry and the event the record hears ("" when the level waits for more approvals).
func (s *Service) decisionOutcome(ctx context.Context, dg deciding) (Round, statusLog, Outcome, error) {
	round, sub, a, now := dg.round, dg.sub, dg.a, dg.now
	level := round.Levels[round.CurrentLevel-1]
	suffix := capacitySuffix(dg.au)
	if selfApproval(dg) {
		suffix += selfApprovalSuffix
	}
	decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, level.Number, len(round.Levels))
	logEntry := statusLog{comment: dg.comment, details: decided + suffix}
	notify := s.notifier(a, sub, now)
	owner := []string{sub.OwnerID}
	next := firstDecideLevel(round.Levels, level.Number)
	var err error
	var event Outcome
	switch {
	case dg.kind == ActionDeclined:
		round.Outcome, round.EndedAt, event, logEntry.verb = OutcomeDeclined, &now, OutcomeDeclined, "declined"
		err = notify.send(ctx, owner, NotifyDeclined, fmt.Sprintf(declinedTitle, sub.Kind.Label()),
			fmt.Sprintf(declinedText, a.Name, subjectText(sub), quoted(dg.comment)))
	case !levelDone(level, dg.entries):
		waiting := waitingOn(level, dg.entries)
		logEntry.verb = fmt.Sprintf("approved level %d of", level.Number)
		logEntry.details = fmt.Sprintf("%s · %d of %d approved · still waiting for %s%s", decided,
			len(level.Approvers)-len(waiting), len(level.Approvers), names(waiting), suffix)
	case next == 0:
		round.Outcome, round.EndedAt, event, logEntry.verb = OutcomeApproved, &now, OutcomeApproved, "approved"
		err = notify.send(ctx, owner, NotifyApproved, fmt.Sprintf(approvedTitle, sub.Kind.Label()),
			fmt.Sprintf(approvedText, a.Name, subjectText(sub)))
	default:
		event, logEntry.verb = continued(sub.Kind), fmt.Sprintf("approved level %d of", level.Number)
		logEntry.details = fmt.Sprintf("%s · now waiting for level %d (%s)%s", decided, next,
			approversText(round.Levels[next-1]), suffix)
		err = s.moveToLevel(ctx, dg, &round, next)
	}
	return round, logEntry, event, err
}

// moveToLevel makes round wait at level next and tells its approvers, then the owner (who is asked, not told,
// when they approve the next level).
func (s *Service) moveToLevel(ctx context.Context, dg deciding, round *Round, next int) error {
	level := round.Levels[round.CurrentLevel-1]
	round.CurrentLevel = next
	round.Levels[next-1].StartedAt = &dg.now
	waiting := round.Levels[next-1]
	notify := s.notifier(dg.a, dg.sub, dg.now)
	text := fmt.Sprintf(nextLevelWaitsText, dg.a.Name, level.Number, subjectText(dg.sub), next)
	if err := notify.sendToLevel(ctx, waiting, dg.dl, text); err != nil {
		return err
	}
	return notify.send(ctx, []string{dg.sub.OwnerID}, NotifyLevelApproved, fmt.Sprintf(levelApprovedTitle, level.Number),
		fmt.Sprintf(levelApprovedText, dg.a.Name, level.Number, len(round.Levels), subjectText(dg.sub)))
}

// movedOn explains why a decision on level of round comes too late; a level the round has
// not reached yet is a stale request.
func (s *Service) movedOn(ctx context.Context, round Round, level int) error {
	entries, err := s.repo.ActionsOf(ctx, round.SubjectID)
	if err != nil {
		return err
	}
	mine := entriesOf(entries, round.ID)
	last, found := decisionOn(mine, level)
	if !found && round.Outcome != OutcomeOpen && len(mine) > 0 {
		last, found = mine[len(mine)-1], true
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
	case ActionSuperseded:
		return fmt.Sprintf(resubmittedByMessage, last.Actor.Name)
	default:
		return cancelledMessage
	}
}
