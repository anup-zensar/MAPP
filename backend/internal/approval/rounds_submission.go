package approval

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
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
func (s *Service) Submission(ctx context.Context, a actor.Actor, activityID string) (Submission, error) { //nolint:gocyclo // 11: one Read whose every step returns its error.
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
		if submission.Workflow, submission.Levels, err = s.submissionWorkflow(ctx, workflow); err != nil {
			return err
		}
		if res != nil {
			submission.Outcomes, submission.ApprovedAutomatically = submissionOutcomes(*res, d, a)
		}
		return nil
	})
	if err != nil {
		return Submission{}, err
	}
	return submission, nil
}

// submissionWorkflow returns the workflow and its levels as the confirmation shows them: none without one.
func (s *Service) submissionWorkflow(ctx context.Context, workflow *Workflow) (*WorkflowRef, []LevelView, error) {
	if workflow == nil {
		return nil, []LevelView{}, nil
	}
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	return workflowRef(*workflow), levelViews(workflow.Levels, rc), nil
}

// submissionOutcomes says what would happen to each level, and whether the activity would be approved at once.
func submissionOutcomes(res resolution, d directory, a actor.Actor) ([]SubmissionLevel, bool) {
	outcomes := []SubmissionLevel{}
	for _, l := range res.levels {
		outcomes = append(outcomes, SubmissionLevel{
			Number: l.Number, Label: levelLabel(l.Name, l.Number), Outcome: l.Resolution, Approvers: d.copiedAll(l.Approvers),
			Source: l.Source, Mode: l.Mode, RuleText: l.RuleText, Excluded: excludedViews(l.Excluded, d),
			BackupUsed: l.BackupUsed, NoApprover: l.NoApprover, Advice: noApproverAdvice(l, a.ID),
		})
	}
	return outcomes, len(res.problems) == 0 && len(res.levels) > 0 && firstDecideLevel(res.levels, 0) == 0
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
	workflow, err := s.workflowOf(ctx, facts)
	if err != nil {
		return ReadyCheck{}, nil, nil, err
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
	check.Problems = withWorkflowProblems(problems, added)
	return check, workflow, res, nil
}

// workflowOf returns the approval workflow of the activity, nil when it has none or it is no longer stored.
func (s *Service) workflowOf(ctx context.Context, facts ActivityFacts) (*Workflow, error) {
	var workflow *Workflow
	if facts.WorkflowID != "" {
		w, found, err := s.repo.GetWorkflow(ctx, facts.WorkflowID)
		if err != nil {
			return nil, err
		}
		if found {
			workflow = &w
		}
	}
	return workflow, nil
}

func withWorkflowProblems(problems, added []SubmitProblem) []SubmitProblem {
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
	return problems
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

// workflowRef is how an activity refers to w.
func workflowRef(w Workflow) *WorkflowRef {
	return &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
}
