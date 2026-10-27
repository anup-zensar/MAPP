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

// submittable reports whether an activity with status can be submitted: Created, or Declined to re-submit
// (FR-009).
func submittable(status string) bool {
	return status == StatusCreated || status == StatusDeclined
}

// roundOf returns the round roundID of the activity: not_found when there is none or it is another
// activity's.
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
