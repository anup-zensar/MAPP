package approvals

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

// viewOf returns w as the API shows it, reading its usage inside the caller's transaction.
func (s *Service) viewOf(ctx context.Context, w Workflow, rc ruleContext) (WorkflowView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return WorkflowView{}, err
	}
	return workflowView(w, u, rc), nil
}

// workflowView returns w as the API shows it: its levels with their rules and approvers now, the levels
// text, its timings, its usage and its warnings — a named-people level without an active approver, a
// backup approver who is inactive or cannot approve, a role that no longer exists (FR-005; 017 FR-009).
func workflowView(w Workflow, u usages, rc ruleContext) WorkflowView {
	levels := levelViews(w.Levels, rc)
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
		ID: w.ID, Name: w.Name, Description: w.Description, Active: w.Active, Levels: levels, LevelsText: levelsText(levels),
		UsedByCount: len(u.usedBy[w.ID]), WaitingCount: len(u.waiting[w.ID]), Warnings: warnings,
		ReminderDays: w.ReminderDays, EscalationDays: w.EscalationDays, Meta: w.Meta,
	}
}

// backupWarning warns that the backup approver b of the level n is inactive or cannot approve; "" when
// there is nothing to say.
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

// ruleContext is what the views and checks of workflows read once per operation: the people, whether each
// can approve (014), the roles, and the labels of the values conditions name.
type ruleContext struct {
	people        directory
	canApprove    map[string]bool
	roles         map[string]RoleInfo
	names         choiceNames
	activityTypes []Choice
	businessUnits []Choice
}

// ruleContext reads the rule context inside the caller's transaction.
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

// levelViews returns a workflow's levels as the API shows them, with their approvers' names now and their
// rules in words; never nil.
func levelViews(levels []Level, rc ruleContext) []LevelView {
	d := rc.people
	views := make([]LevelView, 0, len(levels))
	for i, l := range levels {
		n := i + 1
		view := LevelView{
			Number: n, Name: l.Name, Label: levelLabel(l.Name, n), Approvers: make([]PersonView, 0, len(l.ApproverIDs)),
			ApproverType: l.ApproverType, Mode: l.Mode,
			AppliesWhen: conditionView(l.AppliesWhen, rc), AutoApproveWhen: conditionView(l.AutoApproveWhen, rc),
		}
		for _, id := range l.ApproverIDs {
			view.Approvers = append(view.Approvers, d.person(id))
		}
		if l.BackupApproverID != "" {
			backup := d.person(l.BackupApproverID)
			view.BackupApprover = &backup
		}
		var source string
		switch l.ApproverType {
		case SourceManager:
			source = "Line manager"
		case SourceRole:
			role, known := rc.roles[l.RoleID]
			if !known {
				role.RoleRef = RoleRef{ID: l.RoleID, Name: l.RoleID}
			}
			ref := role.RoleRef
			view.Role = &ref
			source = "Role: " + role.Name
		default:
			names := make([]string, 0, len(view.Approvers))
			for _, p := range view.Approvers {
				if p.Active {
					names = append(names, p.Name)
				} else {
					names = append(names, p.Name+" (inactive)")
				}
			}
			source = strings.Join(names, ", ")
		}
		view.Summary = levelSummary(l, n, source, rc.names)
		views = append(views, view)
	}
	return views
}

// levelsText writes the levels in one line, their summaries joined with " → " (data-model §2, FR-005):
// "Level 1 · Line manager · automatic when planned cost is below $10,000.00 → Level 2 · Alex Morgan".
func levelsText(levels []LevelView) string {
	parts := make([]string, 0, len(levels))
	for _, l := range levels {
		parts = append(parts, l.Summary)
	}
	return strings.Join(parts, " → ")
}

// conditionView returns a condition as the API answers it, with its values' labels and its text; nil for
// none.
func conditionView(c *Condition, rc ruleContext) *ConditionView {
	if c == nil {
		return nil
	}
	view := &ConditionView{BusinessUnits: []OptionRef{}, ActivityTypes: []SettingRef{}, Text: conditionText(*c, rc.names)}
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
