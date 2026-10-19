package activities

import (
	"context"
	"fmt"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// NewForm returns everything the new-activity form under the initiative initiativeID needs, with its
// defaults (FR-001, FR-002, FR-007 – FR-009, FR-012, FR-014, FR-015, D-104): the plan context, the active
// activity types the plan's model allows, the active approval workflows, the activity-level Shell
// attributes and the names already used in the initiative. A model without types gives an empty type list
// (edge case).
func (s *Service) NewForm(ctx context.Context, a actor.Actor, initiativeID string) (FormView, error) {
	var form FormView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		pc, found, err := s.contextOf(ctx, initiativeID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		form, err = s.form(ctx, a, pc, nil)
		if err != nil {
			return err
		}
		form.Defaults, err = s.defaults(ctx, pc, form.ApprovalWorkflows)
		return err
	})
	if err != nil {
		return FormView{}, err
	}
	return form, nil
}

// EditForm returns everything the edit form of the activity id needs (FR-013, FR-026): as NewForm, plus
// its current type, workflow and attributes even when inactive or no longer allowed; the names exclude its
// own; no defaults.
func (s *Service) EditForm(ctx context.Context, a actor.Actor, id string) (FormView, error) {
	var form FormView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		activity, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		form, err = s.form(ctx, a, pc, &activity)
		return err
	})
	if err != nil {
		return FormView{}, err
	}
	return form, nil
}

// form builds the form of a new activity (current nil) or of current.
func (s *Service) form(ctx context.Context, a actor.Actor, pc planContext, current *Activity) (FormView, error) {
	formContext, err := s.formContext(ctx, pc)
	if err != nil {
		return FormView{}, err
	}
	form := FormView{
		Context: formContext, ActivityTypes: []TypeOption{}, ApprovalWorkflows: []WorkflowOption{},
		Attributes: []settings.ShellAttributeView{}, ExistingNames: []string{},
	}

	types, err := s.settings.ActivityTypes(ctx)
	if err != nil {
		return FormView{}, err
	}
	for _, t := range types {
		allowed := pc.model.Allows(t.ID)
		if (t.Active && allowed) || (current != nil && current.ActivityTypeID == t.ID) {
			form.ActivityTypes = append(form.ActivityTypes,
				TypeOption{ID: t.ID, Name: t.Name, DefaultPlannedCost: t.DefaultPlannedCost, Active: t.Active, Allowed: allowed})
		}
	}

	workflows, err := s.workflows.ListWorkflows(ctx)
	if err != nil {
		return FormView{}, err
	}
	for _, w := range workflows {
		if !w.Active && (current == nil || current.ApprovalWorkflowID != w.ID) {
			continue
		}
		option, err := s.workflowOption(ctx, w)
		if err != nil {
			return FormView{}, err
		}
		form.ApprovalWorkflows = append(form.ApprovalWorkflows, option)
	}

	level := settings.LevelActivity
	page, err := s.settings.ListAttributes(ctx, a, &level)
	if err != nil {
		return FormView{}, err
	}
	for _, attr := range page.Items {
		if attr.Active || (current != nil && !isEmptyValue(current.Attributes[attr.ID])) {
			form.Attributes = append(form.Attributes, attr)
		}
	}

	siblings, err := s.repo.ListByInitiative(ctx, pc.initiative.ID)
	if err != nil {
		return FormView{}, err
	}
	for _, sibling := range siblings {
		if current == nil || sibling.ID != current.ID {
			form.ExistingNames = append(form.ExistingNames, sibling.Name)
		}
	}
	return form, nil
}

// formContext is the plan context as the form shows it.
func (s *Service) formContext(ctx context.Context, pc planContext) (FormContext, error) {
	owner, err := s.personRef(ctx, deref(pc.initiative.OwnerID))
	if err != nil {
		return FormContext{}, err
	}
	dealSize := pc.model.AverageDealSize
	if dealSize.Currency == "" {
		dealSize = money.USD(dealSize.Amount) // a plan whose model is gone still answers in the contract's shape
	}
	units := []BusinessUnit{}
	for _, id := range pc.businessUnit {
		option, found, err := s.settings.Option(ctx, id)
		if err != nil {
			return FormContext{}, err
		}
		if found {
			units = append(units, BusinessUnit{ID: option.ID, Label: option.Label})
		}
	}
	return FormContext{
		Plan:     PlanFacts{ID: pc.plan.ID, Name: pc.plan.Name, Year: pc.plan.Year, Status: string(pc.plan.Status)},
		Campaign: errs.RecordRef{Type: "campaign", ID: pc.campaign.ID, Name: pc.campaign.Name},
		Initiative: InitiativeFacts{ID: pc.initiative.ID, Name: pc.initiative.Name, Owner: owner,
			StartDate: pc.initiative.StartDate, EndDate: pc.initiative.EndDate},
		Model: ModelFacts{ID: pc.model.ID, Name: pc.model.Name, InqToMqlRate: pc.model.InqToMqlRate,
			MqlToDealRate: pc.model.MqlToDealRate, AverageDealSize: dealSize, Active: pc.model.Active},
		BusinessUnits: units,
		LatestEndDate: latestEndDate(pc.plan.Year),
	}, nil
}

// workflowOption is a workflow as the form offers it, with its levels and their approvers; a level
// without a name is called "Level <n>".
func (s *Service) workflowOption(ctx context.Context, w WorkflowFacts) (WorkflowOption, error) {
	option := WorkflowOption{ID: w.ID, Name: w.Name, Active: w.Active, Levels: []WorkflowLevel{}}
	for i, l := range w.Levels {
		level := WorkflowLevel{Name: l.Name, Approvers: []people.PersonRef{}}
		if level.Name == "" {
			level.Name = fmt.Sprintf("Level %d", i+1)
		}
		for _, id := range l.ApproverIDs {
			ref, err := s.personRef(ctx, id)
			if err != nil {
				return WorkflowOption{}, err
			}
			if ref != nil {
				level.Approvers = append(level.Approvers, *ref)
			}
		}
		option.Levels = append(option.Levels, level)
	}
	return option, nil
}

// defaults are what a new activity starts with (FR-008, FR-009, FR-012): the initiative's owner when
// active, the initiative's dates, and the approval workflow when exactly one is active.
func (s *Service) defaults(ctx context.Context, pc planContext, active []WorkflowOption) (*Defaults, error) {
	defaults := &Defaults{StartDate: pc.initiative.StartDate, EndDate: pc.initiative.EndDate}
	owner, err := s.personRef(ctx, deref(pc.initiative.OwnerID))
	if err != nil {
		return nil, err
	}
	if owner != nil && owner.Active {
		defaults.Owner = owner
	}
	if len(active) == 1 {
		defaults.ApprovalWorkflowID = &active[0].ID
	}
	return defaults, nil
}
