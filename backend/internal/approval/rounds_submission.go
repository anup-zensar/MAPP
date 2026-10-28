package approval

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
)

const (
	workflowField             = "approvalWorkflowId"
	modificationWorkflowField = "modificationWorkflowId"
	versionField              = "version"
)

// Submission says what Submit (or Re-submit) for approval would do now, and changes nothing.
func (s *Service) Submission(ctx context.Context, a actor.Actor, activityID string) (Submission, error) {
	var submission Submission
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		facts, err := s.activity(ctx, activityID)
		if err != nil {
			return err
		}
		submission, err = s.submission(ctx, a, activitySubject(facts))
		return err
	})
	if err != nil {
		return Submission{}, err
	}
	return submission, nil
}

func (s *Service) submission(ctx context.Context, a actor.Actor, sub subject) (Submission, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return Submission{}, err
	}
	check, workflow, res, err := s.submitCheck(ctx, a, sub, d)
	if err != nil {
		return Submission{}, err
	}
	rounds, err := s.repo.RoundsOf(ctx, sub.ID)
	if err != nil {
		return Submission{}, err
	}
	shown, err := s.displayOf(ctx, a)
	if err != nil {
		return Submission{}, err
	}
	open := openRoundPointer(rounds)
	submission := Submission{
		ActivityID: sub.ID, ActivityName: sub.Name, ActivityVersion: sub.Version, Kind: kindSubmit,
		RoundNumber: nextRoundNumber(rounds), Ready: len(check.Problems) == 0, Problems: check.Problems,
		Levels: []LevelView{}, Outcomes: []SubmissionLevel{}, Replaces: replaced(sub, open),
		UnassignedPlannedCost: shown.optional(check.UnassignedPlannedCost),
	}
	if kind := submissionKind(sub, open); kind != "" {
		submission.Kind, submission.Available = kind, true
	} else {
		submission.Reason = notSubmittableMessage(sub.Kind)
	}
	if submission.Workflow, submission.Levels, err = s.submissionWorkflow(ctx, workflow); err != nil {
		return Submission{}, err
	}
	if res != nil {
		submission.Outcomes, submission.ApprovedAutomatically = submissionOutcomes(*res, d, a)
	}
	return submission, nil
}

// nextRoundNumber is the number of a record's next round: 1, then the highest number + 1; rounds are by number.
func nextRoundNumber(rounds []Round) int {
	if len(rounds) == 0 {
		return 1
	}
	return rounds[len(rounds)-1].Number + 1
}

func (s *Service) submissionWorkflow(ctx context.Context, workflow *Workflow) (*WorkflowRef, []LevelView, error) {
	if workflow == nil {
		return nil, []LevelView{}, nil
	}
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	return workflowRef(*workflow), levelViews(workflow.Levels, rc, workflow.Kind()), nil
}

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

// submitCheck runs the record's ready-to-submit check and adds its workflow's problem — none, inactive, or
// of another record type — unless the check lists that field; with a usable workflow it
// resolves the approvers as a would submit now, adding the problems of levels whose source names nobody.
func (s *Service) submitCheck(ctx context.Context, a actor.Actor, sub subject, d directory) (ReadyCheck, *Workflow, *resolution, error) {
	check, err := s.readyCheck(ctx, a, sub)
	if err != nil {
		return ReadyCheck{}, nil, nil, err
	}
	workflow, err := s.workflowOf(ctx, sub)
	if err != nil {
		return ReadyCheck{}, nil, nil, err
	}
	field, group := workflowFieldOf(sub.Kind), recordGroup(sub.Kind)
	problems := slices.Clone(check.Problems)
	var res *resolution
	added := []SubmitProblem{}
	listed := slices.ContainsFunc(problems, func(p SubmitProblem) bool { return p.Field == field })
	message := workflowProblem(sub.Kind, workflow)
	switch {
	case listed:
	case message != "":
		added = append(added, SubmitProblem{Field: field, Group: group, Message: message})
	default:
		resolved, err := s.resolvedLevels(ctx, resolving{sub: sub, submitter: a, d: d}, workflow)
		if err != nil {
			return ReadyCheck{}, nil, nil, err
		}
		res, added = &resolved, resolved.problems
	}
	check.Problems = withWorkflowProblems(problems, added, group)
	return check, workflow, res, nil
}

func (s *Service) resolvedLevels(ctx context.Context, r resolving, workflow *Workflow) (resolution, error) {
	names, err := s.choiceNames(ctx)
	if err != nil {
		return resolution{}, err
	}
	plans := evaluate(workflow.Levels, ruleFacts(r.sub), names.forKind(r.sub.Kind))
	return s.resolve(ctx, plans, r)
}

// workflowProblem is the problem of the record's workflow, "" when it can be used.
func workflowProblem(kind SubjectKind, workflow *Workflow) string {
	switch {
	case workflow == nil && kind == KindModifiedPurchaseRequest:
		return modificationWorkflowNeededMessage
	case workflow == nil:
		return workflowNeededMessage
	case !workflow.Active:
		return fmt.Sprintf(workflowInactiveMessage, workflow.Name)
	case workflow.Kind() != kind:
		return fmt.Sprintf(workflowOtherKindMessage, workflow.Name, workflow.Kind().plural(), kind.plural())
	}
	return ""
}

func workflowFieldOf(kind SubjectKind) string {
	if kind == KindModifiedPurchaseRequest {
		return modificationWorkflowField
	}
	return workflowField
}

func recordGroup(kind SubjectKind) string {
	if kind.isPurchaseRequest() {
		return GroupPurchaseRequest
	}
	return GroupActivity
}

func (s *Service) workflowOf(ctx context.Context, sub subject) (*Workflow, error) {
	id := sub.workflowID()
	if id == "" {
		return nil, nil
	}
	w, found, err := s.repo.GetWorkflow(ctx, id)
	if err != nil || !found {
		return nil, err
	}
	return &w, nil
}

// withWorkflowProblems puts the workflow's problems last among the record's own, before those of its lines.
func withWorkflowProblems(problems, added []SubmitProblem, group string) []SubmitProblem {
	if len(added) > 0 {
		at := len(problems)
		for at > 0 && problems[at-1].Group != group {
			at--
		}
		problems = slices.Insert(problems, at, added...)
	}
	if problems == nil {
		problems = []SubmitProblem{}
	}
	return problems
}

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

func workflowRef(w Workflow) *WorkflowRef {
	return &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
}
