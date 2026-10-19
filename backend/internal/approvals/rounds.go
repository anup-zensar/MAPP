package approvals

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
)

// workflowField is the activity form field of the Approval workflow, the key of its problems.
const workflowField = "approvalWorkflowId"

// The kinds of a submission (ApprovalSubmission.kind).
const (
	kindSubmit   = "submit"
	kindResubmit = "resubmit"
)

// Submission says what Submit for approval (or Re-submit for approval) would do now, and changes nothing
// (FR-010, FR-011): every problem of the ready-to-submit check, or the workflow and its levels to confirm,
// with a positive unassigned planned cost as a note.
func (s *Service) Submission(ctx context.Context, a actor.Actor, activityID string) (Submission, error) {
	var submission Submission
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		check, workflow, err := s.submitCheck(ctx, a, facts)
		if err != nil {
			return err
		}
		rounds, err := s.repo.RoundsOf(ctx, facts.ID)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}

		submission = Submission{
			ActivityID: facts.ID, ActivityName: facts.Name, ActivityVersion: facts.Version,
			Kind: kindSubmit, Available: submittable(facts.Status), RoundNumber: nextRoundNumber(rounds),
			Ready: len(check.Problems) == 0, Problems: check.Problems, Levels: []LevelView{},
			UnassignedPlannedCost: check.UnassignedPlannedCost,
		}
		if facts.Status == StatusDeclined {
			submission.Kind = kindResubmit
		}
		if !submission.Available {
			submission.Reason = onlyCreatedOrDeclinedMessage
		}
		if workflow != nil {
			submission.Workflow = workflowRef(*workflow)
			submission.Levels = levelViews(workflow.Levels, d)
		}
		return nil
	})
	if err != nil {
		return Submission{}, err
	}
	return submission, nil
}

// Submit submits a Created activity, or re-submits a Declined one (FR-009 – FR-012, FR-020). in.Version
// must be the activity version the person saw (D-030). The ready-to-submit check runs again: with any
// problem nothing changes and every problem is reported at once (FR-010). Otherwise, in one Write: a new
// round (the next number) starts at level 1 with a copy of the workflow's levels as they are now (FR-006,
// FR-011), the activity becomes Submitted, the round's history and the activity log record it (FR-026),
// and the level-1 approvers are notified (FR-035). The plan's status is not checked (FR-012, D-206).
func (s *Service) Submit(ctx context.Context, a actor.Actor, activityID string, in SubmitInput) (ActivityApproval, error) {
	comment := strings.TrimSpace(in.Comment)
	var problems errs.Problems
	if in.Version == nil || *in.Version < 1 {
		problems.Add("version", reloadMessage)
	}
	if tooLong(comment, maxCommentLength) {
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
		if err := facts.CheckVersion(*in.Version, "activity"); err != nil {
			return err
		}
		if !submittable(facts.Status) {
			return errs.BadRequest(onlyCreatedOrDeclinedMessage)
		}
		check, workflow, err := s.submitCheck(ctx, a, facts)
		if err != nil {
			return err
		}
		if len(check.Problems) > 0 {
			fields := make([]errs.FieldError, 0, len(check.Problems))
			for _, p := range check.Problems {
				fields = append(fields, errs.FieldError{Field: p.Field, Message: p.Message})
			}
			return errs.Validation(fields...)
		}

		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		rounds, err := s.repo.RoundsOf(ctx, facts.ID)
		if err != nil {
			return err
		}
		now := s.now()
		round := Round{
			ActivityID: facts.ID, Number: nextRoundNumber(rounds), WorkflowID: workflow.ID, WorkflowName: workflow.Name,
			Levels: roundLevels(workflow.Levels, d), CurrentLevel: 1, Outcome: OutcomeOpen,
			SubmittedAt: now, SubmittedBy: a, Comment: comment, Meta: record.NewMeta(a, now),
		}
		if round.ID, err = s.ids.Next(ctx, ids.PrefixApprovalRound); err != nil {
			return err
		}
		if err := s.repo.PutRound(ctx, round); err != nil {
			return err
		}

		action, verb, text := ActionSubmitted, "submitted", submittedText
		if facts.Status == StatusDeclined {
			action, verb, text = ActionResubmitted, "re-submitted", resubmittedText
		}
		if err := s.addAction(ctx, round, action, 1, a, nil, now, comment); err != nil {
			return err
		}
		updated, err := s.activities.SetStatus(ctx, a, facts.ID, StatusSubmitted)
		if err != nil {
			return err
		}
		first := round.Levels[0]
		err = s.logOnActivity(ctx, a, updated, statusLog{
			verb: verb, oldStatus: facts.Status, newer: StatusSubmitted, comment: comment,
			details: fmt.Sprintf("Round %d · %s · waiting for level 1 (%s)", round.Number, round.WorkflowName, names(first.Approvers)),
		})
		if err != nil {
			return err
		}
		err = s.notifier(a, updated, now).send(ctx, idsOf(first.Approvers), NotifyApprovalNeeded, approvalNeededTitle,
			fmt.Sprintf(text, a.Name, activityText(updated)))
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

// Approve approves the current level of the activity's open round, as in.RoundID and in.Level name it
// (FR-014, FR-015). Anyone may approve (D-020): a person who is not an approver of the level decides on
// behalf of its approvers (FR-017). A level other than the last moves the round to the next level, whose
// approvers are notified, and tells the owner; the last level makes the activity Approved and tells the
// owner (FR-035). A level or round that has moved on is refused, saying what happened (FR-018).
func (s *Service) Approve(ctx context.Context, a actor.Actor, activityID string, in DecisionInput) (ActivityApproval, error) {
	return s.decide(ctx, a, activityID, in, ActionApproved)
}

// Decline declines the activity at the current level of its open round, with a comment that is not blank
// (FR-016): the round ends, the activity becomes Declined, the comment heads its Approval section, and the
// owner is told with the start of the comment (FR-035). The other rules are Approve's.
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

		now := s.now()
		level := round.Levels[round.CurrentLevel-1]
		behalf := onBehalfOf(level.Approvers, a)
		if err := s.addAction(ctx, round, decision, level.Number, a, behalf, now, comment); err != nil {
			return err
		}
		suffix := ""
		if len(behalf) > 0 {
			suffix = "; on behalf of " + names(behalf)
		}
		decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, level.Number, len(round.Levels))

		updated := facts
		entry := statusLog{comment: comment, details: decided + suffix}
		notify := s.notifier(a, facts, now)
		owner := []string{facts.OwnerID}
		switch {
		case decision == ActionDeclined:
			round.Outcome, round.EndedAt = OutcomeDeclined, &now
			entry.verb, entry.oldStatus, entry.newer = "declined", facts.Status, StatusDeclined
			err = notify.send(ctx, owner, NotifyDeclined, activityDeclinedTitle,
				fmt.Sprintf(declinedText, a.Name, activityText(facts), quoted(comment)))
		case level.Number == len(round.Levels):
			round.Outcome, round.EndedAt = OutcomeApproved, &now
			entry.verb, entry.oldStatus, entry.newer = "approved", facts.Status, StatusApproved
			err = notify.send(ctx, owner, NotifyApproved, activityApprovedTitle, fmt.Sprintf(approvedText, a.Name, activityText(facts)))
		default:
			round.CurrentLevel++
			next := round.Levels[round.CurrentLevel-1]
			entry.verb = fmt.Sprintf("approved level %d of", level.Number)
			entry.details = fmt.Sprintf("%s · now waiting for level %d (%s)%s", decided, next.Number, names(next.Approvers), suffix)
			// The next approvers first: an owner who approves the next level is asked to, not told.
			err = notify.send(ctx, idsOf(next.Approvers), NotifyApprovalNeeded, approvalNeededTitle,
				fmt.Sprintf(nextLevelWaitsText, a.Name, level.Number, activityText(facts), next.Number))
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
		if entry.newer != "" {
			if updated, err = s.activities.SetStatus(ctx, a, facts.ID, entry.newer); err != nil {
				return err
			}
		}
		if err := s.logOnActivity(ctx, a, updated, entry); err != nil {
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

// submittable reports whether an activity with status can be submitted: Created, or Declined to re-submit
// (FR-009).
func submittable(status string) bool {
	return status == StatusCreated || status == StatusDeclined
}

// submitCheck runs the ready-to-submit check of 006 and 007 for facts and adds the problem of its approval
// workflow — not chosen or no longer stored, or inactive (FR-010 e) — unless the check already lists that
// field. It returns the workflow too, nil when there is none. Problems are never nil.
func (s *Service) submitCheck(ctx context.Context, a actor.Actor, facts ActivityFacts) (ReadyCheck, *Workflow, error) {
	check, err := s.activities.ReadyCheck(ctx, a, facts.ID)
	if err != nil {
		return ReadyCheck{}, nil, err
	}
	var workflow *Workflow
	if facts.WorkflowID != "" {
		w, found, err := s.repo.GetWorkflow(ctx, facts.WorkflowID)
		if err != nil {
			return ReadyCheck{}, nil, err
		}
		if found {
			workflow = &w
		}
	}

	problems := slices.Clone(check.Problems)
	listed := slices.ContainsFunc(problems, func(p SubmitProblem) bool { return p.Field == workflowField })
	var problem *SubmitProblem
	switch {
	case listed:
	case workflow == nil:
		problem = &SubmitProblem{Field: workflowField, Group: GroupActivity, Message: workflowNeededMessage}
	case !workflow.Active:
		problem = &SubmitProblem{Field: workflowField, Group: GroupActivity, Message: fmt.Sprintf(workflowInactiveMessage, workflow.Name)}
	}
	if problem != nil {
		// The activity's own problems come first, then those of its line items: the workflow's goes last of
		// the activity's.
		at := len(problems)
		for at > 0 && problems[at-1].Group != GroupActivity {
			at--
		}
		problems = slices.Insert(problems, at, *problem)
	}
	if problems == nil {
		problems = []SubmitProblem{}
	}
	check.Problems = problems
	return check, workflow, nil
}

// nextRoundNumber is the number of an activity's next round: 1, then the highest number + 1 (FR-020).
// rounds are ordered by number.
func nextRoundNumber(rounds []Round) int {
	if len(rounds) == 0 {
		return 1
	}
	return rounds[len(rounds)-1].Number + 1
}

// roundLevels copies a workflow's levels into a round, with the approvers' names now (FR-006, FR-011).
func roundLevels(levels []Level, d directory) []RoundLevel {
	copied := make([]RoundLevel, 0, len(levels))
	for i, l := range levels {
		copied = append(copied, RoundLevel{Number: i + 1, Name: l.Name, Approvers: d.actors(l.ApproverIDs)})
	}
	return copied
}

// addAction writes one history entry of round about level (0 for none), inside the caller's Write.
// onBehalf is nil for actions other than decisions.
func (s *Service) addAction(ctx context.Context, round Round, kind ActionKind, level int, by actor.Actor,
	onBehalf []actor.Actor, at time.Time, comment string) error {
	id, err := s.ids.Next(ctx, ids.PrefixApprovalAction)
	if err != nil {
		return err
	}
	entry := Action{
		ID: id, RoundID: round.ID, RoundNumber: round.Number, ActivityID: round.ActivityID, Action: kind,
		Level: level, Approvers: []actor.Actor{}, Actor: by, OnBehalfOf: []actor.Actor{}, At: at, Comment: comment,
		Changes: []ChangeText{},
	}
	if level > 0 {
		l := round.Levels[level-1]
		entry.LevelName, entry.Approvers = l.Name, slices.Clone(l.Approvers)
	}
	if onBehalf != nil {
		entry.OnBehalfOf = onBehalf
	}
	return s.repo.PutAction(ctx, entry)
}

// workflowRef is how an activity refers to w.
func workflowRef(w Workflow) *WorkflowRef {
	return &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
}
