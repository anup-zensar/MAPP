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
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/workbook"
)

// Create creates a record under parentID with the rules of its single create, empty fields taking the
// on-screen defaults (FR-032): the owner and dates of the parent, the type's planned cost, the line item's
// form defaults. Activities and line items are kept until Flush (038 FR-008).
func (w *workbookWriter) Create(ctx context.Context, a actor.Actor, r workbook.NewRecord) (string, error) {
	parentID, v := r.ParentID, r.Values
	switch r.Level {
	case workbook.LevelCampaign:
		return w.p.createCampaign(ctx, a, parentID, v)
	case workbook.LevelInitiative:
		if err := w.permit(ctx, parentID, authz.CampaignsEdit); err != nil {
			return "", err
		}
		return w.p.createInitiative(ctx, a, parentID, v)
	case workbook.LevelActivity:
		return w.createActivity(ctx, parentID, v)
	case workbook.LevelLineItem:
	}
	return w.createLineItem(ctx, parentID, v)
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

// createActivity fills in the new form's defaults (not its overspend limits: an import is never refused for
// overspend, 037 FR-012).
func (w *workbookWriter) createActivity(ctx context.Context, initiativeID string, v workbook.Values) (string, error) {
	if err := w.permit(ctx, initiativeID, authz.ActivitiesEdit); err != nil {
		return "", err
	}
	defaults, err := w.activities.Defaults(ctx, initiativeID)
	if err != nil {
		return "", err
	}
	in := activity.Input{Name: v.Name, ExternalName: v.ExternalName, ActivityTypeID: v.ActivityTypeID,
		OwnerID: textOrNil(orText(v.OwnerID, personID(defaults.Owner))), Ongoing: v.Ongoing,
		StartDate: dateText(orDate(v.StartDate, defaults.StartDate)), EndDate: dateText(orDate(v.EndDate, defaults.EndDate)),
		PlannedCost: amountInput(v.PlannedCost, v.Currency), ApprovalWorkflowID: textOrNil(v.WorkflowID),
		Description: v.Description, Attributes: filled(v.Attributes)}
	if v.WorkflowID == "" {
		in.ApprovalWorkflowID = defaults.ApprovalWorkflowID
	}
	if cost, found := defaults.TypeCosts[v.ActivityTypeID]; found && in.PlannedCost == nil {
		in.PlannedCost = &money.Input{Amount: cost.Amount.String(), Currency: cost.Currency}
	}
	id, err := w.activities.Create(ctx, initiativeID, in)
	if err == nil {
		w.newOwners[id] = orText(v.OwnerID, personID(defaults.Owner))
	}
	return id, err
}

func (w *workbookWriter) createLineItem(ctx context.Context, activityID string, v workbook.Values) (string, error) {
	if err := w.permit(ctx, activityID, authz.ActivitiesEdit); err != nil {
		return "", err
	}
	d, err := w.lineItems.Defaults(ctx, activityID)
	if err != nil {
		return "", err
	}
	in := lineitem.Input{Name: v.Name, OwnerID: textOrNil(orText(v.OwnerID, personID(d.Owner))), Description: v.Description,
		Currency: v.Currency, PlannedCost: amountText(v.PlannedCost), Spread: lineitem.SpreadInput{Method: d.SpreadMethod},
		Attributes: filled(v.Attributes), RequestSwbs: d.RequestSwbs, FinanceBudgetYear: &d.FinanceBudgetYear,
		FinanceBudgetItemID: d.FinanceBudgetItemID}
	return w.lineItems.Create(ctx, activityID, in)
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
