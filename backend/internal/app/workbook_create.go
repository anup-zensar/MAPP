package app

import (
	"context"
	"encoding/json"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/workbook"
)

// workbookForms keeps each parent's form for the rest of a check's or import's Write: the defaults taken from it
// depend on the parent alone, which no later sheet changes, and every create would otherwise read it again.
const workbookForms = "workbookForms:"

// Create creates a record under parentID with its single create, empty fields taking the on-screen defaults
// (FR-032): the owner and dates of the parent, the type's planned cost, the line item's form defaults.
func (p workbookPlanning) Create(ctx context.Context, a actor.Actor, r workbook.NewRecord) (string, error) {
	parentID, v := r.ParentID, r.Values
	switch r.Level {
	case workbook.LevelCampaign:
		return p.createCampaign(ctx, a, parentID, v)
	case workbook.LevelInitiative:
		return p.createInitiative(ctx, a, parentID, v)
	case workbook.LevelActivity:
		return p.createActivity(ctx, a, parentID, v)
	case workbook.LevelLineItem:
	}
	return p.createLineItem(ctx, a, parentID, v)
}

func (p workbookPlanning) createCampaign(ctx context.Context, a actor.Actor, planID string, v workbook.Values) (string, error) {
	stored, found, err := p.plans.Get(ctx, planID)
	if err != nil || !found {
		return "", readProblem(err, found)
	}
	if err = authz.RequireIn(ctx, authz.CampaignsEdit, plan.AccessFacts(stored)); err != nil {
		return "", err
	}
	owner := stored.OwnerID
	start, end := date.New(stored.Year, 1, 1), date.New(stored.Year, 12, 31)
	in := campaign.CampaignInput{Name: v.Name, Description: v.Description, OwnerID: textOrNil(orText(v.OwnerID, owner)),
		StartDate: orDate(v.StartDate, &start), EndDate: orDate(v.EndDate, &end), Budget: amountInput(v.Budget, v.Currency),
		Attributes: filled(v.Attributes)}
	view, err := p.campaigns.CreateCampaign(ctx, a, planID, in)
	return view.ID, err
}

func (p workbookPlanning) createInitiative(ctx context.Context, a actor.Actor, campaignID string, v workbook.Values) (string, error) {
	if err := p.permit(ctx, campaignID, authz.CampaignsEdit); err != nil {
		return "", err
	}
	parent, found, err := p.campaignsRepo.GetCampaign(ctx, campaignID)
	if err != nil || !found {
		return "", readProblem(err, found)
	}
	owner := ""
	if parent.OwnerID != nil {
		owner = *parent.OwnerID
	}
	in := campaign.InitiativeInput{Name: v.Name, Description: v.Description, OwnerID: textOrNil(orText(v.OwnerID, owner)),
		StartDate: orDate(v.StartDate, parent.StartDate), EndDate: orDate(v.EndDate, parent.EndDate),
		Budget: amountInput(v.Budget, v.Currency), Attributes: filled(v.Attributes)}
	view, err := p.campaigns.CreateInitiative(ctx, a, campaignID, in)
	return view.ID, err
}

func (p workbookPlanning) createActivity(ctx context.Context, a actor.Actor, initiativeID string, v workbook.Values) (string, error) {
	if err := p.permit(ctx, initiativeID, authz.ActivitiesEdit); err != nil {
		return "", err
	}
	form, err := memo.Get(ctx, workbookForms+"activity:"+initiativeID, func(ctx context.Context) (activity.FormView, error) {
		return p.activities.NewForm(ctx, a, initiativeID)
	})
	if err != nil {
		return "", err
	}
	defaults := activity.Defaults{}
	if form.Defaults != nil {
		defaults = *form.Defaults
	}
	in := activity.Input{Name: v.Name, ExternalName: v.ExternalName, ActivityTypeID: v.ActivityTypeID,
		OwnerID: textOrNil(orText(v.OwnerID, personID(defaults.Owner))), Ongoing: v.Ongoing,
		StartDate: dateText(orDate(v.StartDate, defaults.StartDate)), EndDate: dateText(orDate(v.EndDate, defaults.EndDate)),
		PlannedCost: amountInput(v.PlannedCost, v.Currency), ApprovalWorkflowID: textOrNil(v.WorkflowID),
		Description: v.Description, Attributes: filled(v.Attributes)}
	if v.WorkflowID == "" {
		in.ApprovalWorkflowID = defaults.ApprovalWorkflowID
	}
	if in.PlannedCost == nil {
		in.PlannedCost = typeDefault(form.ActivityTypes, v.ActivityTypeID)
	}
	view, err := p.activities.Create(ctx, a, initiativeID, in)
	return view.ID, err
}

func (p workbookPlanning) createLineItem(ctx context.Context, a actor.Actor, activityID string, v workbook.Values) (string, error) {
	if err := p.permit(ctx, activityID, authz.ActivitiesEdit); err != nil {
		return "", err
	}
	options, err := memo.Get(ctx, workbookForms+"lineItem:"+activityID, func(ctx context.Context) (lineitem.FormOptions, error) {
		return p.lineItems.FormOptions(ctx, a, activityID)
	})
	if err != nil {
		return "", err
	}
	d := options.Defaults
	in := lineitem.Input{Name: v.Name, OwnerID: textOrNil(orText(v.OwnerID, personID(d.Owner))), Description: v.Description,
		Currency: v.Currency, PlannedCost: amountText(v.PlannedCost), Spread: lineitem.SpreadInput{Method: d.SpreadMethod},
		Attributes: filled(v.Attributes), RequestSwbs: d.RequestSwbs, FinanceBudgetYear: &d.FinanceBudgetYear,
		FinanceBudgetItemID: d.FinanceBudgetItemID}
	view, err := p.lineItems.Create(ctx, a, activityID, in)
	return view.ID, err
}

func (p workbookPlanning) Delete(ctx context.Context, a actor.Actor, level workbook.Level, id string) error {
	permission := authz.CampaignsDelete
	if level == workbook.LevelActivity || level == workbook.LevelLineItem {
		permission = authz.ActivitiesDelete
	}
	if err := p.permit(ctx, id, permission); err != nil {
		return err
	}
	var err error
	switch level {
	case workbook.LevelCampaign:
		_, err = p.campaigns.DeleteCampaign(ctx, a, id)
	case workbook.LevelInitiative:
		_, err = p.campaigns.DeleteInitiative(ctx, a, id)
	case workbook.LevelActivity:
		_, err = p.activities.Delete(ctx, a, id)
	case workbook.LevelLineItem:
		_, err = p.lineItems.Delete(ctx, a, id)
	}
	return err
}

func orText(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func orDate(value, fallback *date.Date) *date.Date {
	if value != nil {
		return value
	}
	return fallback
}

func personID(p *people.PersonRef) string {
	if p == nil {
		return ""
	}
	return p.ID
}

func amountInput(a *money.Amount, currency string) *money.Input {
	if a == nil {
		return nil
	}
	return &money.Input{Amount: a.String(), Currency: currency}
}

func typeDefault(types []activity.TypeOption, typeID string) *money.Input {
	for _, t := range types {
		if t.ID == typeID && t.DefaultPlannedCost != nil {
			return &money.Input{Amount: t.DefaultPlannedCost.Amount.String(), Currency: t.DefaultPlannedCost.Currency}
		}
	}
	return nil
}

// filled is the values a new record starts with: the attributes the file gives a value.
func filled(values map[string]json.RawMessage) settings.AttributeValues {
	out := settings.AttributeValues{}
	for id, raw := range values {
		if raw != nil {
			out[id] = raw
		}
	}
	return out
}
