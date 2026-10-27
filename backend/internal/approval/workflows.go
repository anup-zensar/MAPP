package approval

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The longest workflow name and level name, in characters (data model §1).
const (
	maxWorkflowNameLength = 120
	maxLevelNameLength    = 60
)

// UpdateWorkflow changes the fields of the approval workflow id that update sends (D-203), levels as a
// whole new list, after checking every rule on the result at once (FR-003). update.Version must be the
// version the person started from (D-030). An approver already in a level may stay there after becoming
// inactive, but may not be added (008-P8). A save that changes nothing makes no new version and no log
// entry; otherwise one "changed" entry lists every changed field (FR-026). Open rounds keep the levels they
// were submitted with (FR-006).
func (s *Service) UpdateWorkflow(ctx context.Context, a actor.Actor, id string, update WorkflowUpdate) (WorkflowView, error) {
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", reloadMessage)
		} else if err := stored.CheckVersion(*update.Version, workflowSingular); err != nil {
			return err
		}
		changed := stored
		if update.Name != nil {
			changed.Name = strings.TrimSpace(*update.Name)
		}
		if update.Description != nil {
			changed.Description = strings.TrimSpace(*update.Description)
		}
		amountProblems := map[string]string{}
		if update.Levels != nil {
			changed.Levels, amountProblems = levelsOf(*update.Levels)
		}
		if update.ReminderDays.Sent {
			changed.ReminderDays = update.ReminderDays.Days
		}
		if update.EscalationDays.Sent {
			changed.EscalationDays = update.EscalationDays.Days
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		if err := s.checkWorkflow(ctx, &problems, rc, changed, amountProblems, stored); err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}

		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.viewOf(ctx, stored, rc)
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutWorkflow(ctx, changed); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
		})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, changed, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

// SetWorkflowActive sets the approval workflow id inactive (active false) or active again (FR-007), from the
// version the person saw. An inactive workflow is not offered on activities; those that use it keep it, and
// open rounds finish with the levels they were submitted with. Setting the value it already has changes
// nothing; otherwise the change is logged as "deactivated" or "reactivated" (FR-026).
func (s *Service) SetWorkflowActive(ctx context.Context, a actor.Actor, id string, active bool, version *int) (WorkflowView, error) {
	if version == nil || *version < 1 {
		return WorkflowView{}, errs.Validation(errs.FieldError{Field: "version", Message: reloadMessage})
	}
	var view WorkflowView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.storedWorkflow(ctx, id)
		if err != nil {
			return err
		}
		if err := stored.CheckVersion(*version, workflowSingular); err != nil {
			return err
		}
		rc, err := s.ruleContext(ctx)
		if err != nil {
			return err
		}
		if stored.Active == active {
			view, err = s.viewOf(ctx, stored, rc)
			return err
		}
		changed := stored
		changed.Active = active
		changes, err := history.Diff(stored, changed)
		if err != nil {
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.PutWorkflow(ctx, changed); err != nil {
			return err
		}
		verb := "reactivated"
		if !active {
			verb = "deactivated"
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind: workflowKind, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionStatusChanged,
			Verb: verb, Changes: changes,
		})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, changed, rc)
		return err
	})
	if err != nil {
		return WorkflowView{}, err
	}
	return view, nil
}

// The day counts of reminders and escalation (017 FR-007).
const (
	maxReminderDays   = 60
	maxEscalationDays = 90
)

// checkWorkflow adds every problem of the workflow w to problems (FR-002, FR-003; 017 FR-008, FR-025, data
// model §1–§2); amountProblems are the amounts levelsOf could not read; stored is the workflow before an
// update (its id "" on a create), whose inactive approvers and backup approvers may stay in their level.
func (s *Service) checkWorkflow(ctx context.Context, problems *errs.Problems, rc ruleContext, w Workflow,
	amountProblems map[string]string, stored Workflow) error {
	switch {
	case w.Name == "":
		problems.Add("name", enterNameMessage)
	case tooLong(w.Name, maxWorkflowNameLength):
		problems.Add("name", nameTooLongMessage)
	default:
		workflows, err := s.repo.ListWorkflows(ctx)
		if err != nil {
			return err
		}
		if other, taken := workflowNamed(workflows, w.Name, stored.ID); taken {
			problems.Add("name", fmt.Sprintf(nameTakenMessage, other.Name))
		}
	}
	if tooLong(w.Description, maxCommentLength) {
		problems.Add("description", tooLong2000Message)
	}
	remindOK := w.ReminderDays == nil || (*w.ReminderDays >= 1 && *w.ReminderDays <= maxReminderDays)
	if !remindOK {
		problems.Add("reminderDays", reminderDaysMessage)
	}
	switch {
	case w.EscalationDays == nil:
	case *w.EscalationDays < 1 || *w.EscalationDays > maxEscalationDays:
		problems.Add("escalationDays", escalationDaysMessage)
	case remindOK && w.ReminderDays != nil && *w.EscalationDays <= *w.ReminderDays:
		problems.Add("escalationDays", escalationOrderMessage)
	}
	if len(w.Levels) == 0 {
		problems.Add("levels", addLevelMessage)
	}
	for i, l := range w.Levels {
		var before Level // the level at this position before the update: its people may stay when inactive
		if i < len(stored.Levels) {
			before = stored.Levels[i]
		}
		checkLevel(problems, rc, i+1, l, before, w.EscalationDays != nil, amountProblems)
	}
	return nil
}

// checkLevel adds every problem of the level l at position n to problems (data-model §2), in the order of
// its fields. before is the level at that position before an update.
func checkLevel(problems *errs.Problems, rc ruleContext, n int, l, before Level, escalates bool, amountProblems map[string]string) {
	d := rc.people
	field := func(name string) string { return fmt.Sprintf("levels.%d.%s", n, name) }
	if tooLong(l.Name, maxLevelNameLength) {
		problems.Add(field("name"), fmt.Sprintf(levelNameTooLongMessage, n))
	}
	switch l.ApproverType {
	case SourcePeople, SourceManager, SourceRole:
	default:
		problems.Add(field("approverType"), fmt.Sprintf(chooseSourceMessage, n))
	}

	if l.ApproverType == SourcePeople {
		approvers := field("approverIds")
		if len(l.ApproverIDs) == 0 {
			problems.Add(approvers, fmt.Sprintf(chooseApproverMessage, n))
		}
		seen, unknown := map[string]bool{}, false
		for _, id := range l.ApproverIDs {
			p, found := d.byID[id]
			added := !slices.Contains(before.ApproverIDs, id)
			switch {
			case !found:
				unknown = true
			case seen[id]:
				problems.Add(approvers, fmt.Sprintf(listedTwiceMessage, n, p.Name))
			case !p.Active && added:
				problems.Add(approvers, fmt.Sprintf(inactiveApproverMessage, n, p.Name))
			case !rc.canApprove[id] && added:
				problems.Add(approvers, fmt.Sprintf(approverCannotApproveMessage, p.Name))
			}
			seen[id] = true
		}
		if unknown {
			problems.Add(approvers, fmt.Sprintf(unknownApproverMessage, n))
		}
	}

	if l.ApproverType == SourceRole {
		role, known := rc.roles[l.RoleID]
		switch {
		case l.RoleID == "":
			problems.Add(field("roleId"), fmt.Sprintf(chooseRoleMessage, n))
		case !known:
			problems.Add(field("roleId"), fmt.Sprintf(roleFromListMessage, n))
		case !role.Approves:
			problems.Add(field("roleId"), fmt.Sprintf(roleCannotApproveMessage, n, role.Name))
		}
	}
	if l.Mode != ModeAny && l.Mode != ModeAll {
		problems.Add(field("mode"), fmt.Sprintf(chooseModeMessage, n))
	}

	for _, c := range []struct {
		name, emptyMessage string
		condition          *Condition
	}{{"appliesWhen", emptyAppliesWhenMessage, l.AppliesWhen}, {"autoApproveWhen", emptyAutoApproveMessage, l.AutoApproveWhen}} {
		if message, bad := amountProblems[field(c.name)]; bad {
			problems.Add(field(c.name), message)
			continue
		}
		if c.condition == nil {
			continue
		}
		if message := conditionProblem(rc, n, *c.condition, c.emptyMessage); message != "" {
			problems.Add(field(c.name), message)
		}
	}

	backup := field("backupApproverId")
	if l.BackupApproverID == "" {
		if escalates {
			problems.Add(backup, fmt.Sprintf(backupRequiredMessage, n))
		}
		return
	}
	p, found := d.byID[l.BackupApproverID]
	changed := l.BackupApproverID != before.BackupApproverID
	switch {
	case !found:
		problems.Add(backup, fmt.Sprintf(backupFromListMessage, n))
	case !p.Active && changed:
		problems.Add(backup, fmt.Sprintf(backupInactiveMessage, n, p.Name))
	case !rc.canApprove[p.ID] && changed:
		problems.Add(backup, fmt.Sprintf(backupCannotApproveMessage, n, p.Name))
	case l.ApproverType == SourcePeople && slices.Contains(l.ApproverIDs, p.ID):
		problems.Add(backup, fmt.Sprintf(backupIsApproverMessage, n, p.Name))
	}
}
