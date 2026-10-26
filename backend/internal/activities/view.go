package activities

import (
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// derived are the values worked out from a stored activity when it is read (D-041).
type derived struct {
	activityType  TypeRef
	owner         *people.PersonRef
	workflow      *WorkflowRef
	displayStatus DisplayStatus
	unassigned    *money.Money
	readiness     SubmitCheck
	completeness  Completeness
	missing       []errs.MissingField // the items of "This activity" as fields still needed
}

// derive works out the derived values of activity a in its plan context, with its line items' facts.
func (s *Service) derive(ctx context.Context, a Activity, pc planContext, facts LineItemFacts) (derived, error) {
	d := derived{displayStatus: displayStatus(a, date.Today(s.now())), unassigned: unassigned(a, facts)}

	t, _, err := s.settings.ActivityType(ctx, a.ActivityTypeID)
	if err != nil {
		return derived{}, err
	}
	d.activityType = TypeRef{ID: a.ActivityTypeID, Name: t.Name, Active: t.Active, Allowed: pc.model.Allows(a.ActivityTypeID)}
	if d.owner, err = s.personRef(ctx, a.OwnerID); err != nil {
		return derived{}, err
	}
	var workflow *WorkflowFacts
	if a.ApprovalWorkflowID != "" {
		w, found, err := s.workflows.GetWorkflow(ctx, a.ApprovalWorkflowID)
		if err != nil {
			return derived{}, err
		}
		if found {
			workflow = &w
			d.workflow = &WorkflowRef{ID: w.ID, Name: w.Name, Active: w.Active}
		}
	}
	missing, err := s.settings.MissingAttributes(ctx, settings.LevelActivity, a.Attributes, pc.businessUnit)
	if err != nil {
		return derived{}, err
	}
	d.readiness = check(checkInput{
		activity: a, initiative: pc.initiative, missing: missing, typeName: t.Name, typeAllowed: d.activityType.Allowed,
		modelName: pc.model.Name, workflow: workflow, lineItems: facts,
	})
	d.completeness = completeness(d.readiness, facts)
	d.missing = missingFields(d.readiness, missing)
	return d, nil
}

// viewOf returns activity a as the API answers it (Activity), with its line items' facts.
func (s *Service) viewOf(ctx context.Context, a Activity, pc planContext) (View, error) {
	facts, err := s.lineItemFacts(ctx, []Activity{a})
	if err != nil {
		return View{}, err
	}
	d, err := s.derive(ctx, a, pc, facts[a.ID])
	if err != nil {
		return View{}, err
	}
	attributes := a.Attributes
	if attributes == nil {
		attributes = settings.AttributeValues{}
	}
	li := facts[a.ID]
	view := View{
		ID: a.ID, Name: a.Name, ExternalName: a.ExternalName,
		Plan:         errs.RecordRef{Type: "plan", ID: pc.plan.ID, Name: pc.plan.Name},
		Campaign:     errs.RecordRef{Type: "campaign", ID: pc.campaign.ID, Name: pc.campaign.Name},
		Initiative:   errs.RecordRef{Type: "initiative", ID: pc.initiative.ID, Name: pc.initiative.Name},
		ActivityType: d.activityType, Owner: d.owner, StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing,
		PlannedCost: a.PlannedCost, ApprovalWorkflow: d.workflow, Description: a.Description, Attributes: attributes,
		Goals: goalsOf(a, pc.model), Status: a.Status, DisplayStatus: d.displayStatus, UnassignedPlannedCost: d.unassigned,
		LineItems: LineItemTotals{
			Count: li.Count, PlannedCost: money.Money{Amount: li.PlannedCostPlan, Currency: pc.plan.Currency},
			AutoSpread: li.AutoSpread, CustomSpread: li.CustomSpread,
		},
		Completeness: d.completeness, Readiness: d.readiness, Meta: a.Meta,
	}
	if view.SplitFrom, view.SplitInto, err = s.splitLinks(ctx, a, pc.plan.ID); err != nil {
		return View{}, err
	}
	return displayView(money.DisplayUSD, view), nil // in USD until the service converts it for the reader
}

// lineItemFacts asks feature 007 once for the line items of every activity of list.
func (s *Service) lineItemFacts(ctx context.Context, list []Activity) (map[string]LineItemFacts, error) {
	ids := make([]string, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	facts, err := s.lineItems.Facts(ctx, ids)
	if err != nil {
		return nil, err
	}
	if facts == nil {
		facts = map[string]LineItemFacts{}
	}
	return facts, nil
}

// personRef returns the person id as other records show them, or nil for "" or a person who is not found.
func (s *Service) personRef(ctx context.Context, id string) (*people.PersonRef, error) {
	if id == "" {
		return nil, nil
	}
	person, found, err := s.people.Get(ctx, id)
	if err != nil || !found {
		return nil, err
	}
	ref := person.Ref()
	return &ref, nil
}
