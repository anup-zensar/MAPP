package approvals

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

// workflowField is the activity form field of the Approval workflow, the key of its problems.
const workflowField = "approvalWorkflowId"

// The kinds of a submission (ApprovalSubmission.kind).
const (
	kindSubmit   = "submit"
	kindResubmit = "resubmit"
)

// Submission says what Submit for approval (or Re-submit for approval) would do now, and changes nothing
// (FR-010, FR-011; 017 FR-012, FR-013): every problem of the ready-to-submit check and of the workflow's
// approver sources, or the workflow with what would happen to each level — who approves it and how they were
// found, or why it is not needed or approved automatically — with a positive unassigned planned cost as a
// note.
func (s *Service) Submission(ctx context.Context, a actor.Actor, activityID string) (Submission, error) {
	var submission Submission
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		check, workflow, res, err := s.submitCheck(ctx, a, facts, d)
		if err != nil {
			return err
		}
		rounds, err := s.repo.RoundsOf(ctx, facts.ID)
		if err != nil {
			return err
		}
		shown, err := s.displayOf(ctx, a)
		if err != nil {
			return err
		}

		submission = Submission{
			ActivityID: facts.ID, ActivityName: facts.Name, ActivityVersion: facts.Version,
			Kind: kindSubmit, Available: submittable(facts.Status), RoundNumber: nextRoundNumber(rounds),
			Ready: len(check.Problems) == 0, Problems: check.Problems, Levels: []LevelView{}, Outcomes: []SubmissionLevel{},
			UnassignedPlannedCost: shown.optional(check.UnassignedPlannedCost), // in a's display currency (D-213)
		}
		if facts.Status == StatusDeclined {
			submission.Kind = kindResubmit
		}
		if !submission.Available {
			submission.Reason = onlyCreatedOrDeclinedMessage
		}
		if workflow != nil {
			rc, err := s.ruleContext(ctx)
			if err != nil {
				return err
			}
			submission.Workflow = workflowRef(*workflow)
			submission.Levels = levelViews(workflow.Levels, rc)
		}
		if res != nil {
			for _, l := range res.levels {
				submission.Outcomes = append(submission.Outcomes, SubmissionLevel{
					Number: l.Number, Label: levelLabel(l.Name, l.Number), Outcome: l.Resolution, Approvers: d.copiedAll(l.Approvers),
					Source: l.Source, Mode: l.Mode, RuleText: l.RuleText, Excluded: excludedViews(l.Excluded, d),
					BackupUsed: l.BackupUsed, NoApprover: l.NoApprover, Advice: noApproverAdvice(l, a.ID),
				})
			}
			submission.ApprovedAutomatically = len(res.problems) == 0 && len(res.levels) > 0 && firstDecideLevel(res.levels, 0) == 0
		}
		return nil
	})
	if err != nil {
		return Submission{}, err
	}
	return submission, nil
}

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
		d, err := s.directory(ctx)
		if err != nil {
			return err
		}
		check, workflow, res, err := s.submitCheck(ctx, a, facts, d)
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

		rounds, err := s.repo.RoundsOf(ctx, facts.ID)
		if err != nil {
			return err
		}
		now := s.now()
		used := ruleFacts(facts)
		round := Round{
			ActivityID: facts.ID, Number: nextRoundNumber(rounds), WorkflowID: workflow.ID, WorkflowName: workflow.Name,
			Levels: res.levels, Outcome: OutcomeOpen, SubmittedAt: now, SubmittedBy: a, Comment: comment,
			Facts: &used, ReminderDays: workflow.ReminderDays, EscalationDays: workflow.EscalationDays, Meta: record.NewMeta(a, now),
		}
		first := firstDecideLevel(round.Levels, 0)
		if first > 0 {
			round.CurrentLevel = first
			round.Levels[first-1].StartedAt = &now
		} else {
			round.CurrentLevel, round.Outcome, round.EndedAt, round.AutoApproved = len(round.Levels), OutcomeApproved, &now, true
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
		if err := s.addAction(ctx, round, action, first, a, nil, now, comment); err != nil {
			return err
		}
		var automatic []string // the reasons of the levels approved automatically
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
			entry, err := s.newAction(ctx, round, kind, l.Number, actor.System, nil, now, "")
			if err != nil {
				return err
			}
			entry.Rule = l.RuleText
			if err := s.repo.PutAction(ctx, entry); err != nil {
				return err
			}
		}

		notes := levelNotes(round.Levels)
		if first == 0 {
			return s.approvedAtSubmission(ctx, a, facts, round, verb, comment, notes, automatic, &approval)
		}
		updated, err := s.activities.SetStatus(ctx, a, facts.ID, StatusSubmitted)
		if err != nil {
			return err
		}
		waiting := round.Levels[first-1]
		details := fmt.Sprintf("Round %d · %s", round.Number, round.WorkflowName)
		if notes != "" {
			details += " · " + notes
		}
		err = s.logOnActivity(ctx, a, updated, statusLog{
			verb: verb, oldStatus: facts.Status, newer: StatusSubmitted, comment: comment,
			details: fmt.Sprintf("%s · waiting for level %d (%s)", details, waiting.Number, approversText(waiting)),
		})
		if err != nil {
			return err
		}
		dl, err := s.currentDelegations(ctx, d)
		if err != nil {
			return err
		}
		err = s.notifyLevel(ctx, s.notifier(a, updated, now), waiting, dl, fmt.Sprintf(text, a.Name, activityText(updated)))
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

// approvedAtSubmission ends a submission whose round has no level that needs a decision (017 FR-014,
// 017-P4), inside Submit's Write: the activity becomes Approved, the log gets the submitter's "submitted"
// (to Submitted) and MAPP's "approved automatically" (Submitted to Approved), and the owner is told
// when someone else submitted. automatic are the reasons of the levels approved automatically.
func (s *Service) approvedAtSubmission(ctx context.Context, a actor.Actor, facts ActivityFacts, round Round, verb, comment, notes string,
	automatic []string, approval *ActivityApproval) error {
	updated, err := s.activities.SetStatus(ctx, a, facts.ID, StatusApproved)
	if err != nil {
		return err
	}
	err = s.logOnActivity(ctx, a, updated, statusLog{
		verb: verb, oldStatus: facts.Status, newer: StatusSubmitted, comment: comment,
		details: fmt.Sprintf("Round %d · %s", round.Number, round.WorkflowName),
	})
	if err != nil {
		return err
	}
	if len(automatic) == 0 { // every level was not needed
		notes = noLevelAppliesNote
	}
	err = s.logOnActivity(ctx, actor.System, updated, statusLog{
		verb: "approved automatically", oldStatus: StatusSubmitted, newer: StatusApproved,
		details: fmt.Sprintf("Round %d · %s", round.Number, notes),
	})
	if err != nil {
		return err
	}
	if facts.OwnerID != a.ID {
		reason := strings.Join(automatic, "; ")
		if reason == "" {
			reason = noLevelAppliesNote
		}
		err = s.notifier(actor.System, updated, round.SubmittedAt).send(ctx, []string{facts.OwnerID}, NotifyAutoApproved,
			autoApprovedTitle, fmt.Sprintf(autoApprovedText, activityText(updated), reason))
		if err != nil {
			return err
		}
	}
	*approval, err = s.approval(ctx, a, updated)
	return err
}

// notifyLevel tells the approvers of a level the round has reached that it waits for them, with text, and
// each one's current delegates the same with "— you approve as delegate of <name>" (data-model §8). n never
// tells anyone twice, nor the person who acted.
func (s *Service) notifyLevel(ctx context.Context, n *notifier, level RoundLevel, dl []Delegation, text string) error {
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
// field. With a usable workflow it evaluates the rules and resolves the approvers as a would submit now (017
// FR-010 – FR-012), adding a problem for each level whose source names nobody, after the workflow's. It
// returns the workflow too, nil when there is none, and the resolution, nil without a usable workflow.
// Problems are never nil.
func (s *Service) submitCheck(ctx context.Context, a actor.Actor, facts ActivityFacts, d directory) (ReadyCheck, *Workflow, *resolution, error) {
	check, err := s.activities.ReadyCheck(ctx, a, facts.ID)
	if err != nil {
		return ReadyCheck{}, nil, nil, err
	}
	var workflow *Workflow
	if facts.WorkflowID != "" {
		w, found, err := s.repo.GetWorkflow(ctx, facts.WorkflowID)
		if err != nil {
			return ReadyCheck{}, nil, nil, err
		}
		if found {
			workflow = &w
		}
	}

	problems := slices.Clone(check.Problems)
	listed := slices.ContainsFunc(problems, func(p SubmitProblem) bool { return p.Field == workflowField })
	var added []SubmitProblem
	var res *resolution
	switch {
	case listed:
	case workflow == nil:
		added = append(added, SubmitProblem{Field: workflowField, Group: GroupActivity, Message: workflowNeededMessage})
	case !workflow.Active:
		added = append(added, SubmitProblem{Field: workflowField, Group: GroupActivity, Message: fmt.Sprintf(workflowInactiveMessage, workflow.Name)})
	default:
		names, err := s.choiceNames(ctx)
		if err != nil {
			return ReadyCheck{}, nil, nil, err
		}
		resolved, err := s.resolve(ctx, evaluate(workflow.Levels, ruleFacts(facts), names), facts, a, d)
		if err != nil {
			return ReadyCheck{}, nil, nil, err
		}
		res = &resolved
		added = append(added, resolved.problems...)
	}
	if len(added) > 0 {
		// The activity's own problems come first, then those of its line items: the workflow's go last of
		// the activity's.
		at := len(problems)
		for at > 0 && problems[at-1].Group != GroupActivity {
			at--
		}
		problems = slices.Insert(problems, at, added...)
	}
	if problems == nil {
		problems = []SubmitProblem{}
	}
	check.Problems = problems
	return check, workflow, res, nil
}

// choiceNames reads the labels of the business units and activity types conditions name.
func (s *Service) choiceNames(ctx context.Context) (choiceNames, error) {
	types, err := s.choices.ActivityTypes(ctx)
	if err != nil {
		return choiceNames{}, err
	}
	units, err := s.choices.BusinessUnits(ctx)
	if err != nil {
		return choiceNames{}, err
	}
	return namesOf(units, types), nil
}

// nextRoundNumber is the number of an activity's next round: 1, then the highest number + 1 (FR-020).
// rounds are ordered by number.
func nextRoundNumber(rounds []Round) int {
	if len(rounds) == 0 {
		return 1
	}
	return rounds[len(rounds)-1].Number + 1
}

// addAction writes one history entry of round about level (0 for none), inside the caller's Write.
// onBehalf is nil for actions other than decisions.
func (s *Service) addAction(ctx context.Context, round Round, kind ActionKind, level int, by actor.Actor,
	onBehalf []actor.Actor, at time.Time, comment string) error {
	entry, err := s.newAction(ctx, round, kind, level, by, onBehalf, at, comment)
	if err != nil {
		return err
	}
	return s.repo.PutAction(ctx, entry)
}

// newAction returns a new history entry of round about level (0 for none), with its id and without
// changes, for the caller to store.
func (s *Service) newAction(ctx context.Context, round Round, kind ActionKind, level int, by actor.Actor,
	onBehalf []actor.Actor, at time.Time, comment string) (Action, error) {
	id, err := s.ids.Next(ctx, ids.PrefixApprovalAction)
	if err != nil {
		return Action{}, err
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
		entry.OnBehalfOf = slices.Clone(onBehalf)
	}
	return entry, nil
}

// workflowRef is how an activity refers to w.
func workflowRef(w Workflow) *WorkflowRef {
	return &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
}
