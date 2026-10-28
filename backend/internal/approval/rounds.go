package approval

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// Submit submits a Created activity or re-submits a Declined or Submitted one, from the version the person saw.
func (s *Service) Submit(ctx context.Context, a actor.Actor, activityID string, in SubmitInput) (ActivityApproval, error) {
	comment := strings.TrimSpace(in.Comment)
	if err := submitProblems(in, comment); err != nil {
		return ActivityApproval{}, err
	}
	var approval ActivityApproval
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		if err = facts.CheckVersion(*in.Version, "activity"); err != nil {
			return err
		}
		updated, err := s.submit(ctx, a, activitySubject(facts), comment)
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

func submitProblems(in SubmitInput, comment string) error {
	var problems errs.Problems
	if in.Version == nil || *in.Version < 1 {
		problems.Add(versionField, reloadMessage)
	}
	if tooLong(comment, maxCommentLength) {
		problems.Add("comment", tooLong2000Message)
	}
	return problems.Err()
}

// submitting is one submission inside its Write: what it read, then the round it started.
type submitting struct {
	a         actor.Actor
	comment   string
	now       time.Time
	before    subject
	sub       subject
	kind      string
	rounds    []Round
	open      *Round
	d         directory
	workflow  Workflow
	levels    []RoundLevel
	round     Round
	automatic []string
}

// submit starts a round for sub, in the caller's Write; a re-submission first ends the open round.
func (s *Service) submit(ctx context.Context, a actor.Actor, sub subject, comment string) (subject, error) {
	sb, err := s.submittingOf(ctx, a, sub)
	if err != nil {
		return subject{}, err
	}
	sb.comment, sb.now = comment, s.now()
	if err = s.supersede(ctx, &sb); err != nil {
		return subject{}, err
	}
	if err = s.startRound(ctx, &sb); err != nil {
		return subject{}, err
	}
	if sb.round.AutoApproved {
		return s.approvedAtSubmission(ctx, sb)
	}
	return s.waitAtFirstLevel(ctx, sb)
}

// submittingOf reads what the submission needs, or every problem of the ready-to-submit check at once.
func (s *Service) submittingOf(ctx context.Context, a actor.Actor, sub subject) (submitting, error) {
	rounds, err := s.repo.RoundsOf(ctx, sub.ID)
	if err != nil {
		return submitting{}, err
	}
	open := openRoundPointer(rounds)
	kind := submissionKind(sub, open)
	if kind == "" {
		return submitting{}, errs.BadRequest(notSubmittableMessage(sub.Kind))
	}
	d, err := s.directory(ctx)
	if err != nil {
		return submitting{}, err
	}
	check, workflow, res, err := s.submitCheck(ctx, a, sub, d)
	if err != nil {
		return submitting{}, err
	}
	if len(check.Problems) > 0 {
		fields := make([]errs.FieldError, 0, len(check.Problems))
		for _, p := range check.Problems {
			fields = append(fields, errs.FieldError{Field: p.Field, Message: p.Message})
		}
		return submitting{}, errs.Validation(fields...)
	}
	return submitting{a: a, before: sub, sub: sub, kind: kind, rounds: rounds, open: open, d: d, workflow: *workflow,
		levels: res.levels}, nil
}

// startRound stores sb's round with the submission's history entry, then MAPP's for each level needing no decision.
func (s *Service) startRound(ctx context.Context, sb *submitting) error {
	round, first := openingRound(*sb, nextRoundNumber(sb.rounds))
	var err error
	if round.ID, err = s.ids.Next(ctx, ids.PrefixApprovalRound); err != nil {
		return err
	}
	if err = s.repo.PutRound(ctx, round); err != nil {
		return err
	}
	words := submitWordsOf(sb.sub.Kind, sb.kind)
	if err = s.addAction(ctx, round, Action{Action: words.action, Level: first, Actor: sb.a, At: sb.now, Comment: sb.comment}); err != nil {
		return err
	}
	sb.round = round
	sb.automatic, err = s.levelEntries(ctx, round)
	return err
}

// openingRound returns round n and the level it waits at; without one it is approved at once and the level is 0.
func openingRound(sb submitting, n int) (Round, int) {
	used, now := ruleFacts(sb.sub), sb.now
	round := Round{
		SubjectKind: sb.sub.Kind, SubjectID: sb.sub.ID, Number: n, WorkflowID: sb.workflow.ID,
		WorkflowName: sb.workflow.Name, Levels: sb.levels, Outcome: OutcomeOpen, SubmittedAt: now, SubmittedBy: sb.a,
		Comment: sb.comment, Facts: &used, ReminderDays: sb.workflow.ReminderDays, EscalationDays: sb.workflow.EscalationDays,
		Meta: record.NewMeta(sb.a, now),
	}
	first := firstDecideLevel(round.Levels, 0)
	if first > 0 {
		round.CurrentLevel = first
		round.Levels[first-1].StartedAt = &now
	} else {
		round.CurrentLevel, round.Outcome, round.EndedAt, round.AutoApproved = len(round.Levels), OutcomeApproved, &now, true
	}
	return round, first
}

func (s *Service) levelEntries(ctx context.Context, round Round) ([]string, error) {
	var automatic []string
	for _, l := range round.Levels {
		kind := ActionAutoApproved
		switch l.Resolution {
		case ResolutionDecide:
			continue
		case ResolutionSkipped:
			kind = ActionSkipped
		default:
			automatic = append(automatic, l.RuleText)
		}
		entry := Action{Action: kind, Level: l.Number, Actor: actor.System, At: round.SubmittedAt, Rule: l.RuleText}
		if err := s.addAction(ctx, round, entry); err != nil {
			return nil, err
		}
	}
	return automatic, nil
}

func (s *Service) waitAtFirstLevel(ctx context.Context, sb submitting) (subject, error) {
	updated, err := s.moved(ctx, sb.a, sb.sub, OutcomeOpen)
	if err != nil {
		return subject{}, err
	}
	words := submitWordsOf(sb.sub.Kind, sb.kind)
	waiting := sb.round.Levels[sb.round.CurrentLevel-1]
	details := roundDetails(sb)
	if notes := levelNotes(sb.round.Levels); notes != "" {
		details += " · " + notes
	}
	err = s.logOn(ctx, sb.a, updated, statusLog{
		verb: words.verb, oldStatus: sb.before.Status, newer: updated.Status, comment: sb.comment,
		details: fmt.Sprintf("%s · waiting for level %d (%s)", details, waiting.Number, approversText(waiting)),
	})
	if err != nil {
		return subject{}, err
	}
	dl, err := s.currentDelegations(ctx, sb.d)
	if err != nil {
		return subject{}, err
	}
	notify := s.notifier(sb.a, updated, sb.now)
	if err = notify.sendToLevel(ctx, waiting, dl, fmt.Sprintf(words.text, sb.a.Name, subjectText(updated))); err != nil {
		return subject{}, err
	}
	return updated, nil
}

// roundDetails is the start of the submission's log details: "Round 3 · <workflow> · replaces round 2".
func roundDetails(sb submitting) string {
	details := fmt.Sprintf("Round %d · %s", sb.round.Number, sb.round.WorkflowName)
	if sb.open != nil {
		details += fmt.Sprintf(" · replaces round %d", sb.open.Number)
	}
	return details
}

// approvedAtSubmission ends a submission whose round has no level that needs a decision: the
// log gets the submitter's submission and MAPP's "approved automatically"; the owner is told when someone
// else submitted.
func (s *Service) approvedAtSubmission(ctx context.Context, sb submitting) (subject, error) {
	opened, approved, err := s.openedAndApproved(ctx, sb)
	if err != nil {
		return subject{}, err
	}
	round := sb.round
	err = s.logOn(ctx, sb.a, approved, statusLog{
		verb: submitWordsOf(sb.sub.Kind, sb.kind).verb, oldStatus: sb.before.Status, newer: opened.Status,
		comment: sb.comment, details: roundDetails(sb),
	})
	if err != nil {
		return subject{}, err
	}
	notes := levelNotes(round.Levels)
	if len(sb.automatic) == 0 {
		notes = noLevelAppliesNote
	}
	err = s.logOn(ctx, actor.System, approved, statusLog{
		verb: "approved automatically", oldStatus: opened.Status, newer: approved.Status,
		details: fmt.Sprintf("Round %d · %s", round.Number, notes),
	})
	if err != nil || sb.sub.OwnerID == sb.a.ID {
		return approved, err
	}
	reason := strings.Join(sb.automatic, "; ")
	if reason == "" {
		reason = noLevelAppliesNote
	}
	err = s.notifier(actor.System, approved, round.SubmittedAt).send(ctx, []string{sb.sub.OwnerID}, NotifyAutoApproved,
		fmt.Sprintf(autoApprovedTitle, sb.sub.Kind.Label()), fmt.Sprintf(autoApprovedText, subjectText(approved), reason))
	return approved, err
}

// openedAndApproved hands the record the round's start and its approval: an activity goes to Approved in one
// change, a purchase request hears both events.
func (s *Service) openedAndApproved(ctx context.Context, sb submitting) (subject, subject, error) {
	if sb.sub.Kind == KindActivity {
		approved, err := s.moved(ctx, sb.a, sb.sub, OutcomeApproved)
		opened := approved
		opened.Status = StatusSubmitted
		return opened, approved, err
	}
	opened, err := s.moved(ctx, sb.a, sb.sub, OutcomeOpen)
	if err != nil {
		return subject{}, subject{}, err
	}
	approved, err := s.moved(ctx, sb.a, opened, OutcomeApproved)
	return opened, approved, err
}

// sendToLevel tells the approvers of a level the round has reached that it waits for them, and each one's
// current delegates the same with "— you approve as delegate of <name>".
func (n *notifier) sendToLevel(ctx context.Context, level RoundLevel, dl []Delegation, text string) error {
	if err := n.send(ctx, idsOf(level.Approvers), NotifyApprovalNeeded, approvalNeededTitle, text); err != nil {
		return err
	}
	for _, approver := range level.Approvers {
		err := n.send(ctx, delegatesOf(dl, approver.ID), NotifyApprovalNeeded, approvalNeededTitle,
			text+fmt.Sprintf(delegateSuffix, approver.Name))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) addAction(ctx context.Context, round Round, entry Action) error {
	complete, err := s.newAction(ctx, round, entry)
	if err != nil {
		return err
	}
	return s.repo.PutAction(ctx, complete)
}

// newAction completes entry for round and entry.Level (0 for none), with empty lists where it has none.
func (s *Service) newAction(ctx context.Context, round Round, entry Action) (Action, error) {
	id, err := s.ids.Next(ctx, ids.PrefixApprovalAction)
	if err != nil {
		return Action{}, err
	}
	entry.ID, entry.RoundID, entry.RoundNumber = id, round.ID, round.Number
	entry.SubjectKind, entry.SubjectID = round.kind(), round.SubjectID
	entry.Approvers, entry.OnBehalfOf = []actor.Actor{}, slices.Clone(entry.OnBehalfOf)
	if entry.OnBehalfOf == nil {
		entry.OnBehalfOf = []actor.Actor{}
	}
	if entry.Changes == nil {
		entry.Changes = []ChangeText{}
	}
	if entry.Level > 0 {
		l := round.Levels[entry.Level-1]
		entry.LevelName, entry.Approvers = l.Name, slices.Clone(l.Approvers)
	}
	return entry, nil
}
