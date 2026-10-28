package approval

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

const recordOwnerLabel = "Record owner"

func (s *Service) viewOf(ctx context.Context, w Workflow, rc ruleContext) (WorkflowView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return WorkflowView{}, err
	}
	return workflowView(w, u, rc), nil
}

// workflowView returns w with its levels and approvers now, its usage and its warnings.
func workflowView(w Workflow, u usages, rc ruleContext) WorkflowView {
	levels := levelViews(w.Levels, rc, w.Kind())
	warnings := []string{}
	for i, l := range w.Levels {
		n := i + 1
		view := levels[i]
		if l.ApproverType == SourcePeople && !slices.ContainsFunc(view.Approvers, func(p PersonView) bool { return p.Active }) {
			warnings = append(warnings, fmt.Sprintf(noActiveApproverWarning, n))
		}
		if warning := backupWarning(n, view.BackupApprover, rc); warning != "" {
			warnings = append(warnings, warning)
		}
		if _, known := rc.roles[l.RoleID]; l.ApproverType == SourceRole && !known {
			warnings = append(warnings, fmt.Sprintf(roleGoneWarning, n, l.RoleID))
		}
	}
	return WorkflowView{
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, ObjectType: w.Kind(),
		ObjectTypeFixed: len(u.usedBy[w.ID]) > 0, Levels: levels, LevelsText: levelsText(levels),
		UsedByCount: len(u.usedBy[w.ID]), WaitingCount: len(u.waiting[w.ID]), Warnings: warnings,
		ReminderDays: w.ReminderDays, EscalationDays: w.EscalationDays, Meta: w.Meta,
	}
}

func backupWarning(n int, b *PersonView, rc ruleContext) string {
	switch {
	case b == nil:
		return ""
	case !b.Active:
		return fmt.Sprintf(backupInactiveWarning, n, b.Name)
	case !rc.canApprove[b.ID]:
		return fmt.Sprintf(backupCannotApproveWarning, n, b.Name)
	}
	return ""
}

type ruleContext struct {
	people        directory
	canApprove    map[string]bool
	roles         map[string]RoleInfo
	names         choiceNames
	activityTypes []Choice
	businessUnits []Choice
}

func (s *Service) ruleContext(ctx context.Context) (ruleContext, error) {
	d, err := s.directory(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	everyone := make([]string, 0, len(d.byID))
	for id := range d.byID {
		everyone = append(everyone, id)
	}
	slices.Sort(everyone)
	can, err := s.access.CanApprove(ctx, everyone)
	if err != nil {
		return ruleContext{}, err
	}
	roles, err := s.roles(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	types, err := s.choices.ActivityTypes(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	units, err := s.choices.BusinessUnits(ctx)
	if err != nil {
		return ruleContext{}, err
	}
	return ruleContext{people: d, canApprove: can, roles: roles, names: namesOf(units, types), activityTypes: types, businessUnits: units}, nil
}

// levelViews returns a workflow's levels as the API shows them, worded for what the workflow approves.
func levelViews(levels []Level, rc ruleContext, kind SubjectKind) []LevelView {
	views := make([]LevelView, 0, len(levels))
	for i, l := range levels {
		views = append(views, levelView(l, i+1, rc, kind))
	}
	return views
}

func levelView(l Level, n int, rc ruleContext, kind SubjectKind) LevelView {
	names := rc.names.forKind(kind)
	view := LevelView{
		Number: n, Name: l.Name, Label: levelLabel(l.Name, n), Approvers: make([]PersonView, 0, len(l.ApproverIDs)),
		ApproverType: l.ApproverType, Mode: l.Mode,
		AppliesWhen: conditionView(l.AppliesWhen, rc, names), AutoApproveWhen: conditionView(l.AutoApproveWhen, rc, names),
	}
	for _, id := range l.ApproverIDs {
		view.Approvers = append(view.Approvers, rc.people.person(id))
	}
	if l.BackupApproverID != "" {
		backup := rc.people.person(l.BackupApproverID)
		view.BackupApprover = &backup
	}
	if field, known := kind.recordField(l.RecordField); known && l.ApproverType == SourceRecordField {
		view.RecordField = &field
	}
	view.Summary = levelSummary(l, n, levelSource(l, &view, rc), names)
	return view
}

// levelSource is how a level's summary names its approvers: "Line manager", "Role: Approver", "Record owner",
// the record field's label, or the named people.
func levelSource(l Level, view *LevelView, rc ruleContext) string {
	switch l.ApproverType {
	case SourceManager:
		return "Line manager"
	case SourceRole:
		role, known := rc.roles[l.RoleID]
		if !known {
			role.RoleRef = RoleRef{ID: l.RoleID, Name: l.RoleID}
		}
		ref := role.RoleRef
		view.Role = &ref
		return "Role: " + role.Name
	case SourceOwner:
		return recordOwnerLabel
	case SourceRecordField:
		if view.RecordField != nil {
			return view.RecordField.Label
		}
		return l.RecordField
	}
	names := make([]string, 0, len(view.Approvers))
	for _, p := range view.Approvers {
		if p.Active {
			names = append(names, p.Name)
		} else {
			names = append(names, p.Name+" (inactive)")
		}
	}
	return strings.Join(names, ", ")
}

func levelsText(levels []LevelView) string {
	parts := make([]string, 0, len(levels))
	for _, l := range levels {
		parts = append(parts, l.Summary)
	}
	return strings.Join(parts, " → ")
}

func conditionView(c *Condition, rc ruleContext, names choiceNames) *ConditionView {
	if c == nil {
		return nil
	}
	view := &ConditionView{BusinessUnits: []OptionRef{}, ActivityTypes: []SettingRef{}, Text: conditionText(*c, names)}
	if c.PlannedCostAtLeast != nil {
		usd := money.USD(*c.PlannedCostAtLeast)
		view.PlannedCostAtLeast = &usd
	}
	if c.PlannedCostBelow != nil {
		usd := money.USD(*c.PlannedCostBelow)
		view.PlannedCostBelow = &usd
	}
	for _, id := range c.BusinessUnitIDs {
		choice, known := rc.names.businessUnits[id]
		if !known {
			choice = Choice{ID: id, Label: id}
		}
		view.BusinessUnits = append(view.BusinessUnits, OptionRef{ID: id, Label: choice.Label, Active: choice.Active})
	}
	for _, id := range c.ActivityTypeIDs {
		choice, known := rc.names.activityTypes[id]
		if !known {
			choice = Choice{ID: id, Label: id}
		}
		view.ActivityTypes = append(view.ActivityTypes, SettingRef{ID: id, Name: choice.Label, Active: choice.Active})
	}
	return view
}
