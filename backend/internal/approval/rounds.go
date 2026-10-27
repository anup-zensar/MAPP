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

// Submit submits a Created activity, or re-submits a Declined one (FR-009 – FR-012, FR-020; 017 FR-010 –
// FR-014). in.Version must be the activity version the person saw (D-030). The ready-to-submit check runs
// again, with the problems of the workflow's approver sources: with any problem nothing changes and every
// problem is reported at once (FR-010, 017 FR-012). Otherwise, in one Write, the rules are evaluated once on
// the activity's values now and a new round (the next number) keeps what they decided for each level, the
// approvers found, the values used and the workflow's timings (FR-006, 017 FR-010). The round waits at the
// first level that needs a decision: the activity becomes Submitted, the round's history and the activity
// log record it, and that level's approvers and their current delegates are notified (FR-035, 017 FR-032).
// When no level needs a decision the activity is Approved at once: the log gets the submitter's
// "submitted" and MAPP's "approved automatically", and the owner is told when someone else submitted
// (017 FR-014, 017-P4). Levels not needed or approved automatically get a history entry by MAPP. The
// plan's status is not checked (FR-012, D-206).
func (s *Service) Submit(ctx context.Context, a actor.Actor, activityID string, in SubmitInput) (ActivityApproval, error) {
	comment := strings.TrimSpace(in.Comment)
	if err := submitProblems(in, comment); err != nil {
		return ActivityApproval{}, err
	}

	var approval ActivityApproval
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		sb, err := s.submittingOf(ctx, a, activityID, *in.Version)
		if err != nil {
			return err
		}
		sb.comment = comment
		if err = s.startRound(ctx, &sb); err != nil {
			return err
		}
		var updated ActivityFacts
		if sb.round.AutoApproved {
			updated, err = s.approvedAtSubmission(ctx, sb)
		} else {
			updated, err = s.waitAtFirstLevel(ctx, sb)
		}
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

// submitting is one submission inside Submit's Write: what it read, then the round it started.
type submitting struct {
	a         actor.Actor
	comment   string
	now       time.Time
	facts     ActivityFacts
	d         directory
	workflow  Workflow
	levels    []RoundLevel // the workflow's levels as the rules and the approver sources resolved them
	round     Round
	automatic []string // the reasons of the levels approved automatically
}

// submittingOf reads what the submission needs, or every problem of the ready-to-submit check at once.
func (s *Service) submittingOf(ctx context.Context, a actor.Actor, activityID string, version int) (submitting, error) {
	facts, err := s.activity(ctx, activityID)
	if err != nil {
		return submitting{}, err
	}
	if err = facts.CheckVersion(version, "activity"); err != nil {
		return submitting{}, err
	}
	if !submittable(facts.Status) {
		return submitting{}, errs.BadRequest(onlyCreatedOrDeclinedMessage)
	}
	d, err := s.directory(ctx)
	if err != nil {
		return submitting{}, err
	}
	check, workflow, res, err := s.submitCheck(ctx, a, facts, d)
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
	return submitting{a: a, facts: facts, d: d, workflow: *workflow, levels: res.levels}, nil
}

// startRound stores sb's round with the submission's history entry, then MAPP's for each level needing no decision.
func (s *Service) startRound(ctx context.Context, sb *submitting) error {
	rounds, err := s.repo.RoundsOf(ctx, sb.facts.ID)
	if err != nil {
		return err
	}
	sb.now = s.now()
	round, first := openingRound(*sb, nextRoundNumber(rounds))
	if round.ID, err = s.ids.Next(ctx, ids.PrefixApprovalRound); err != nil {
		return err
	}
	if err = s.repo.PutRound(ctx, round); err != nil {
		return err
	}
	kind, _, _ := submitWords(sb.facts.Status)
	if err = s.addAction(ctx, round, Action{Action: kind, Level: first, Actor: sb.a, At: sb.now, Comment: sb.comment}); err != nil {
		return err
	}
	sb.round = round
	sb.automatic, err = s.levelEntries(ctx, round)
	return err
}

// openingRound returns round n and the level it waits at; without one it is approved at once and the level is 0.
func openingRound(sb submitting, n int) (Round, int) {
	used, now := ruleFacts(sb.facts), sb.now
	round := Round{
		ActivityID: sb.facts.ID, Number: n, WorkflowID: sb.workflow.ID, WorkflowName: sb.workflow.Name,
		Levels: sb.levels, Outcome: OutcomeOpen, SubmittedAt: now, SubmittedBy: sb.a, Comment: sb.comment,
		Facts: &used, ReminderDays: sb.workflow.ReminderDays, EscalationDays: sb.workflow.EscalationDays,
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

// levelEntries returns the reasons of the levels approved automatically.
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

// submitWords are the history entry, log verb and notification text: a re-submission when it was declined.
func submitWords(status string) (ActionKind, string, string) {
	if status == StatusDeclined {
		return ActionResubmitted, "re-submitted", resubmittedText
	}
	return ActionSubmitted, "submitted", submittedText
}

func (s *Service) waitAtFirstLevel(ctx context.Context, sb submitting) (ActivityFacts, error) {
	updated, err := s.activities.SetStatus(ctx, sb.a, sb.facts.ID, StatusSubmitted)
	if err != nil {
		return ActivityFacts{}, err
	}
	_, verb, text := submitWords(sb.facts.Status)
	waiting := sb.round.Levels[sb.round.CurrentLevel-1]
	details := fmt.Sprintf("Round %d · %s", sb.round.Number, sb.round.WorkflowName)
	if notes := levelNotes(sb.round.Levels); notes != "" {
		details += " · " + notes
	}
	err = s.logOnActivity(ctx, sb.a, updated, statusLog{
		verb: verb, oldStatus: sb.facts.Status, newer: StatusSubmitted, comment: sb.comment,
		details: fmt.Sprintf("%s · waiting for level %d (%s)", details, waiting.Number, approversText(waiting)),
	})
	if err != nil {
		return ActivityFacts{}, err
	}
	dl, err := s.currentDelegations(ctx, sb.d)
	if err != nil {
		return ActivityFacts{}, err
	}
	notify := s.notifier(sb.a, updated, sb.now)
	if err = notify.sendToLevel(ctx, waiting, dl, fmt.Sprintf(text, sb.a.Name, activityText(updated))); err != nil {
		return ActivityFacts{}, err
	}
	return updated, nil
}

// approvedAtSubmission ends a submission whose round has no level that needs a decision (017 FR-014,
// 017-P4), inside Submit's Write: the activity becomes Approved, the log gets the submitter's "submitted"
// (to Submitted) and MAPP's "approved automatically" (Submitted to Approved), and the owner is told
// when someone else submitted.
func (s *Service) approvedAtSubmission(ctx context.Context, sb submitting) (ActivityFacts, error) {
	a, facts, round := sb.a, sb.facts, sb.round
	updated, err := s.activities.SetStatus(ctx, a, facts.ID, StatusApproved)
	if err != nil {
		return ActivityFacts{}, err
	}
	_, verb, _ := submitWords(facts.Status)
	err = s.logOnActivity(ctx, a, updated, statusLog{
		verb: verb, oldStatus: facts.Status, newer: StatusSubmitted, comment: sb.comment,
		details: fmt.Sprintf("Round %d · %s", round.Number, round.WorkflowName),
	})
	if err != nil {
		return ActivityFacts{}, err
	}
	notes := levelNotes(round.Levels)
	if len(sb.automatic) == 0 { // every level was not needed
		notes = noLevelAppliesNote
	}
	err = s.logOnActivity(ctx, actor.System, updated, statusLog{
		verb: "approved automatically", oldStatus: StatusSubmitted, newer: StatusApproved,
		details: fmt.Sprintf("Round %d · %s", round.Number, notes),
	})
	if err != nil {
		return ActivityFacts{}, err
	}
	if facts.OwnerID == a.ID {
		return updated, nil
	}
	reason := strings.Join(sb.automatic, "; ")
	if reason == "" {
		reason = noLevelAppliesNote
	}
	err = s.notifier(actor.System, updated, round.SubmittedAt).send(ctx, []string{facts.OwnerID}, NotifyAutoApproved,
		autoApprovedTitle, fmt.Sprintf(autoApprovedText, activityText(updated), reason))
	if err != nil {
		return ActivityFacts{}, err
	}
	return updated, nil
}

// sendToLevel tells the approvers of a level the round has reached that it waits for them, with text, and
// each one's current delegates the same with "— you approve as delegate of <name>" (data-model §8). n never
// tells anyone twice, nor the person who acted.
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
	entry.ID, entry.RoundID, entry.RoundNumber, entry.ActivityID = id, round.ID, round.Number, round.ActivityID
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
