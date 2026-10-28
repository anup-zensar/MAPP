package app

import (
	"context"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/internal/workbook"
)

const planGoneMessage = "This plan no longer exists — someone may have deleted it."

// workbookPlanning is workbook.Planning on the planning areas (012 D-339, D-340).
type workbookPlanning struct {
	planningEdits
	plans     stores.PlanStore
	people    stores.PeopleStore
	settings  *settings.Service
	workflows activityWorkflows
}

var _ workbook.Planning = workbookPlanning{}

func (p workbookPlanning) PlanFacts(ctx context.Context, planID string) (authz.PlanFacts, bool, error) {
	stored, found, err := p.plans.Get(ctx, planID)
	return plan.AccessFacts(stored), found, err
}

// Plan reads the plan the person sees with every record under it, parents first, and its choices.
func (p workbookPlanning) Plan(ctx context.Context, _ actor.Actor, planID string) (workbook.Plan, error) {
	stored, found, err := p.plans.Get(ctx, planID)
	if err != nil {
		return workbook.Plan{}, err
	}
	if !found || !authz.CanView(ctx, plan.AccessFacts(stored)) {
		return workbook.Plan{}, errs.NotFound(planGoneMessage)
	}
	stored = stored.Normalised()
	out := workbook.Plan{ID: stored.ID, Name: stored.Name, Currency: stored.Currency}
	if out.Records, err = p.records(ctx, planID); err != nil {
		return workbook.Plan{}, err
	}
	out.Choices, err = p.choices(ctx, stored)
	return out, err
}

func (p workbookPlanning) records(ctx context.Context, planID string) ([]workbook.Stored, error) {
	campaigns, err := p.campaignsRepo.CampaignsOfPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	records, campaignIDs := []workbook.Stored{}, []string{}
	for _, c := range campaigns {
		campaignIDs = append(campaignIDs, c.ID)
		records = append(records, workbook.Stored{Level: workbook.LevelCampaign, ID: c.ID, ParentID: planID, Values: containerValues(
			campaign.Initiative{Name: c.Name, Description: c.Description, OwnerID: c.OwnerID, Attributes: c.Attributes, StartDate: c.StartDate, EndDate: c.EndDate, Budget: c.Budget})})
	}
	initiatives, err := p.campaignsRepo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return nil, err
	}
	initiativeIDs := []string{}
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
		records = append(records, workbook.Stored{Level: workbook.LevelInitiative, ID: i.ID, ParentID: i.CampaignID,
			Values: containerValues(i)})
	}
	activities, err := p.activitiesRepo.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return nil, err
	}
	activityIDs := []string{}
	for _, a := range activities {
		activityIDs = append(activityIDs, a.ID)
		records = append(records, workbook.Stored{Level: workbook.LevelActivity, ID: a.ID, ParentID: a.InitiativeID,
			Status: string(a.Status), Values: activityValues(a)})
	}
	lineItems, err := p.lineItemsRepo.ListByActivities(ctx, activityIDs)
	for _, li := range lineItems {
		records = append(records, workbook.Stored{Level: workbook.LevelLineItem, ID: li.ID, ParentID: li.ActivityID,
			Values: lineItemValues(li)})
	}
	return records, err
}

// containerValues reads a campaign or an initiative, both kept as an Initiative's fields.
func containerValues(c campaign.Initiative) workbook.Values {
	v := workbook.Values{Name: c.Name, Description: c.Description, StartDate: c.StartDate, EndDate: c.EndDate,
		Attributes: attributesOf(c.Attributes)}
	if c.OwnerID != nil {
		v.OwnerID = *c.OwnerID
	}
	if c.Budget != nil {
		v.Budget, v.Currency = &c.Budget.Amount, c.Budget.Currency
	}
	return v
}

func activityValues(a activity.Activity) workbook.Values {
	v := workbook.Values{Name: a.Name, Description: a.Description, ExternalName: a.ExternalName, OwnerID: a.OwnerID,
		StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing, ActivityTypeID: a.ActivityTypeID,
		WorkflowID: a.ApprovalWorkflowID, Attributes: attributesOf(a.Attributes)}
	if a.PlannedCost != nil {
		v.PlannedCost, v.Currency = &a.PlannedCost.Amount, a.PlannedCost.Currency
	}
	return v
}

func lineItemValues(li lineitem.LineItem) workbook.Values {
	planned := li.PlannedCost
	return workbook.Values{Name: li.Name, Description: li.Description, OwnerID: li.OwnerID, Currency: li.Currency,
		PlannedCost: &planned, Attributes: attributesOf(li.Attributes)}
}

func (p workbookPlanning) Exists(ctx context.Context, level workbook.Level, id string) (bool, error) {
	var found bool
	var err error
	switch level {
	case workbook.LevelCampaign:
		_, found, err = p.campaignsRepo.GetCampaign(ctx, id)
	case workbook.LevelInitiative:
		_, found, err = p.campaignsRepo.GetInitiative(ctx, id)
	case workbook.LevelActivity:
		_, found, err = p.activitiesRepo.Get(ctx, id)
	case workbook.LevelLineItem:
		_, found, err = p.lineItemsRepo.Get(ctx, id)
	}
	return found, err
}
