package app

import (
	"context"
	"maps"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

const recordGoneMessage = "This record no longer exists."

// planningEdits edits one planning record through its area's single edit (012 research R5): the whole form
// built from the stored record, the fields a bulk action or an import sets, and the stored version.
type planningEdits struct {
	campaignsRepo  stores.CampaignStore
	activitiesRepo stores.ActivityStore
	lineItemsRepo  stores.LineItemStore
	campaigns      *campaign.Service
	activities     *activity.Service
	lineItems      *lineitem.Service
	targets        planningTargets
}

// permit checks what the access middleware checks for the single operation on id (014 Table A): a record the
// person cannot see is not found, one they see without p in its plan's scope is forbidden (012 FR-024).
func (e planningEdits) permit(ctx context.Context, id string, p authz.Permission) error {
	target, found, err := e.targets.record(ctx, id)
	if err != nil || !found {
		return readProblem(err, found)
	}
	if target.Activity != nil {
		err = authz.RequireViewActivity(ctx, *target.Activity, target.RecordType)
	} else {
		err = authz.RequireView(ctx, target.Plan, target.RecordType)
	}
	if err != nil {
		return err
	}
	return authz.RequireIn(ctx, p, target.Plan)
}

// editCampaign applies change to the campaign's form; change reports whether it changed anything, and
// nothing is sent when it did not.
func (e planningEdits) editCampaign(ctx context.Context, a actor.Actor, id string, change func(*campaign.CampaignUpdate) bool) (bool, error) {
	if err := e.permit(ctx, id, authz.CampaignsEdit); err != nil {
		return false, err
	}
	stored, found, err := e.campaignsRepo.GetCampaign(ctx, id)
	if err != nil || !found {
		return false, readProblem(err, found)
	}
	form := campaign.CampaignUpdate{Name: stored.Name, Description: stored.Description, OwnerID: stored.OwnerID,
		StartDate: stored.StartDate, EndDate: stored.EndDate, Budget: moneyInput(stored.Budget),
		Attributes: attributesOf(stored.Attributes), Version: &stored.Version}
	if !change(&form) {
		return false, nil
	}
	_, err = e.campaigns.UpdateCampaign(ctx, a, id, form)
	return err == nil, err
}

func (e planningEdits) editInitiative(ctx context.Context, a actor.Actor, id string, change func(*campaign.InitiativeUpdate) bool) (bool, error) {
	if err := e.permit(ctx, id, authz.CampaignsEdit); err != nil {
		return false, err
	}
	stored, found, err := e.campaignsRepo.GetInitiative(ctx, id)
	if err != nil || !found {
		return false, readProblem(err, found)
	}
	form := campaign.InitiativeUpdate{Name: stored.Name, Description: stored.Description, OwnerID: stored.OwnerID,
		StartDate: stored.StartDate, EndDate: stored.EndDate, Budget: moneyInput(stored.Budget),
		Attributes: attributesOf(stored.Attributes), Version: &stored.Version}
	if !change(&form) {
		return false, nil
	}
	_, err = e.campaigns.UpdateInitiative(ctx, a, id, form)
	return err == nil, err
}

func (e planningEdits) editActivity(ctx context.Context, a actor.Actor, id string, change func(*activity.Update) bool) (bool, error) {
	if err := e.permit(ctx, id, authz.ActivitiesEdit); err != nil {
		return false, err
	}
	stored, found, err := e.activitiesRepo.Get(ctx, id)
	if err != nil || !found {
		return false, readProblem(err, found)
	}
	form := activity.Update{Name: stored.Name, ExternalName: stored.ExternalName, ActivityTypeID: stored.ActivityTypeID,
		OwnerID: textOrNil(stored.OwnerID), StartDate: dateText(stored.StartDate), EndDate: dateText(stored.EndDate),
		Ongoing: stored.Ongoing, PlannedCost: moneyInput(stored.PlannedCost),
		ApprovalWorkflowID: textOrNil(stored.ApprovalWorkflowID), Description: stored.Description,
		Attributes: attributesOf(stored.Attributes), Version: &stored.Version,
		Goals: activity.GoalsInput{INQ: stored.INQ, SetLaterStagesMyself: stored.SetLaterStagesMyself, MQL: stored.MQL,
			Revenue: moneyInput(stored.Revenue)}}
	if !change(&form) {
		return false, nil
	}
	_, err = e.activities.Update(ctx, a, id, form)
	return err == nil, err
}

func (e planningEdits) editLineItem(ctx context.Context, a actor.Actor, id string, change func(*lineitem.Update) bool) (bool, error) {
	if err := e.permit(ctx, id, authz.ActivitiesEdit); err != nil {
		return false, err
	}
	stored, found, err := e.lineItemsRepo.Get(ctx, id)
	if err != nil || !found {
		return false, readProblem(err, found)
	}
	planned := stored.PlannedCost.String()
	form := lineitem.Update{Name: stored.Name, OwnerID: textOrNil(stored.OwnerID), Description: stored.Description,
		Currency: stored.Currency, PlannedCost: &planned, Spread: spreadInput(stored.Spread),
		Attributes: attributesOf(stored.Attributes), ContractedAmount: amountText(stored.ContractedAmount),
		RequestSwbs: stored.RequestSwbs, FinanceBudgetYear: stored.FinanceBudgetYear,
		FinanceBudgetItemID: textOrNil(stored.FinanceBudgetItemID), Version: &stored.Version}
	if !change(&form) {
		return false, nil
	}
	_, err = e.lineItems.Update(ctx, a, id, form)
	return err == nil, err
}

// readProblem is the error of a read that found no record, or failed.
func readProblem(err error, found bool) error {
	if err == nil && !found {
		return errs.NotFound(recordGoneMessage)
	}
	return err
}

func moneyInput(m *money.Money) *money.Input {
	if m == nil {
		return nil
	}
	return &money.Input{Amount: m.Amount.String(), Currency: m.Currency}
}

func textOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func dateText(d *date.Date) *string {
	if d == nil {
		return nil
	}
	text := d.String()
	return &text
}

func amountText(a *money.Amount) *string {
	if a == nil {
		return nil
	}
	text := a.String()
	return &text
}

func spreadInput(s lineitem.Spread) lineitem.SpreadInput {
	months := make([]lineitem.SpreadMonthInput, 0, len(s.Months))
	for _, m := range s.Months {
		months = append(months, lineitem.SpreadMonthInput{Month: m.Month, Amount: m.Amount.String()})
	}
	return lineitem.SpreadInput{Method: s.Method, Months: months}
}

// attributesOf copies values into a map the edit may change.
func attributesOf(values settings.AttributeValues) settings.AttributeValues {
	copied := settings.AttributeValues{}
	maps.Copy(copied, values)
	return copied
}
