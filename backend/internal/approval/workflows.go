package approval

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const (
	maxWorkflowNameLength = 120
	maxLevelNameLength    = 60
)

func (s *Service) UpdateWorkflow(ctx context.Context, a actor.Actor, id string, update WorkflowUpdate) (WorkflowView, error) {
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		changed, rc, err := s.checkedUpdate(ctx, stored, update)
		if err != nil {
			return err
		}
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		shown := stored
		if len(changes) > 0 {
			shown, err = s.saveWorkflow(ctx, a, changed, history.LogInput{
				Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
			})
			if err != nil {
				return err
			}
		}
		view, err = s.viewOf(ctx, shown, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

func (s *Service) checkedUpdate(ctx context.Context, stored Workflow, update WorkflowUpdate) (Workflow, ruleContext, error) {
	var problems errs.Problems
	if update.Version == nil || *update.Version < 1 {
		problems.Add(versionField, reloadMessage)
	} else if err := stored.CheckVersion(*update.Version, workflowSingular); err != nil {
		return Workflow{}, ruleContext{}, err
	}
	changed, amountProblems := withUpdate(stored, update)
	rc, err := s.ruleContext(ctx)
	if err != nil {
		return Workflow{}, ruleContext{}, err
	}
	if err = s.checkObjectTypeChange(ctx, &problems, stored, changed); err != nil {
		return Workflow{}, ruleContext{}, err
	}
	err = s.checkWorkflow(ctx, &problems, workflowCheck{rc: rc, w: changed, amountProblems: amountProblems, stored: stored})
	if err != nil {
		return Workflow{}, ruleContext{}, err
	}
	if err = problems.Err(); err != nil {
		return Workflow{}, ruleContext{}, err
	}
	return changed, rc, nil
}

func withUpdate(stored Workflow, update WorkflowUpdate) (Workflow, map[string]string) {
	changed, amountProblems := stored, map[string]string{}
	if update.Name != nil {
		changed.Name = strings.TrimSpace(*update.Name)
	}
	if update.Description != nil {
		changed.Description = strings.TrimSpace(*update.Description)
	}
	if update.ObjectType != nil {
		changed.ObjectType = *update.ObjectType
	}
	if update.Levels != nil {
		changed.Levels, amountProblems = levelsOf(*update.Levels)
	}
	if update.ReminderDays.Sent {
		changed.ReminderDays = update.ReminderDays.Days
	}
	if update.EscalationDays.Sent {
		changed.EscalationDays = update.EscalationDays.Days
	}
	return changed, amountProblems
}

func (s *Service) SetWorkflowActive(ctx context.Context, a actor.Actor, id string, active bool, version *int) (WorkflowView, error) { //nolint:revive // argument-limit 5: the version the body sends, and whether the route deactivates or activates
	if version == nil || *version < 1 {
		return WorkflowView{}, errs.Validation(errs.FieldError{Field: versionField, Message: reloadMessage})
	}
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		if err = stored.CheckVersion(*version, workflowSingular); err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		shown, err := s.withActive(ctx, a, stored, active)
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, shown, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

func (s *Service) withActive(ctx context.Context, a actor.Actor, stored Workflow, active bool) (Workflow, error) {
	if stored.Active == active {
		return stored, nil
	}
	changed := stored
	changed.Active = active
	changes, err := history.Diff(stored, changed)
	if err != nil {
		return Workflow{}, err
	}
	verb := "reactivated"
	if !active {
		verb = "deactivated"
	}
	return s.saveWorkflow(ctx, a, changed, history.LogInput{
		Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionStatusChanged,
		Verb: verb, Changes: changes,
	})
}

func (s *Service) saveWorkflow(ctx context.Context, a actor.Actor, changed Workflow, entry history.LogInput) (Workflow, error) {
	changed.Touch(a, s.now())
	if err := s.repo.PutWorkflow(ctx, changed); err != nil {
		return Workflow{}, err
	}
	if _, err := s.history.Log(ctx, a, entry); err != nil {
		return Workflow{}, err
	}
	return changed, nil
}

const (
	maxReminderDays   = 60
	maxEscalationDays = 90
)

type workflowCheck struct {
	rc             ruleContext
	w              Workflow
	amountProblems map[string]string
	stored         Workflow
}

// checkObjectTypeChange refuses a change of what a used workflow approves.
func (s *Service) checkObjectTypeChange(ctx context.Context, problems *errs.Problems, stored, changed Workflow) error {
	if changed.Kind() == stored.Kind() {
		return nil
	}
	u, err := s.usage(ctx)
	if err == nil && len(u.usedBy[stored.ID]) > 0 {
		problems.Add(objectTypeField, fmt.Sprintf(objectTypeFixedMessage, stored.Kind().plural()))
	}
	return err
}

func (s *Service) checkWorkflow(ctx context.Context, problems *errs.Problems, wc workflowCheck) error {
	if err := s.checkWorkflowName(ctx, problems, wc.w.Name, wc.stored.ID); err != nil {
		return err
	}
	if !wc.w.Kind().IsValid() {
		problems.Add(objectTypeField, chooseObjectTypeMessage)
	}
	if tooLong(wc.w.Description, maxCommentLength) {
		problems.Add("description", tooLong2000Message)
	}
	checkDays(problems, wc.w)
	if len(wc.w.Levels) == 0 {
		problems.Add("levels", addLevelMessage)
	}
	for i, l := range wc.w.Levels {
		c := levelCheck{problems: problems, rc: wc.rc, n: i + 1, l: l, kind: wc.w.Kind()}
		if i < len(wc.stored.Levels) {
			c.before = wc.stored.Levels[i]
		}
		checkLevel(c, wc.amountProblems, wc.w.EscalationDays != nil)
	}
	return nil
}

func (s *Service) checkWorkflowName(ctx context.Context, problems *errs.Problems, name, storedID string) error {
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case tooLong(name, maxWorkflowNameLength):
		problems.Add("name", nameTooLongMessage)
	default:
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		if other, taken := workflowNamed(workflows, name, storedID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenMessage, other.Name))
		}
	}
	return nil
}

func checkDays(problems *errs.Problems, w Workflow) {
	remindOK := w.ReminderDays == nil || withinDays(*w.ReminderDays, maxReminderDays)
	if !remindOK {
		problems.Add("reminderDays", reminderDaysMessage)
	}
	switch {
	case w.EscalationDays == nil:
	case !withinDays(*w.EscalationDays, maxEscalationDays):
		problems.Add("escalationDays", escalationDaysMessage)
	case remindOK && w.ReminderDays != nil && *w.EscalationDays <= *w.ReminderDays:
		problems.Add("escalationDays", escalationOrderMessage)
	}
}

func withinDays(days, most int) bool {
	return days >= 1 && days <= most
}
