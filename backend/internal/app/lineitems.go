package app

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// The wiring of feature 007, line items (D-196): the adapters that implement the dependency interfaces of
// package lineitems over the other features, and the ones that hand line items to activities (006) and to
// the plans' level registry (004). They run inside the caller's transaction.

// lineItemActivities implements lineitem.Activities over the activities (006), campaigns (005) and plans
// (004) repositories and Settings (002); RestartApproval is 006's.
type lineItemActivities struct {
	activities stores.ActivityStore
	campaigns  stores.CampaignStore
	plans      stores.PlanStore
	settings   *settings.Service
	service    *activity.Service
}

// Facts returns the activity id with its initiative, campaign and plan, and false when it is not stored.
func (x lineItemActivities) Facts(ctx context.Context, id string) (lineitem.ActivityFacts, bool, error) {
	act, found, err := x.activities.Get(ctx, id)
	if err != nil || !found {
		return lineitem.ActivityFacts{}, false, err
	}
	initiative, found, err := x.campaigns.GetInitiative(ctx, act.InitiativeID)
	if err != nil || !found {
		return lineitem.ActivityFacts{}, false, missing(err, "initiative", act.InitiativeID)
	}
	campaign, found, err := x.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return lineitem.ActivityFacts{}, false, missing(err, "campaign", initiative.CampaignID)
	}
	p, found, err := x.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return lineitem.ActivityFacts{}, false, missing(err, "plan", campaign.PlanID)
	}
	f := lineitem.ActivityFacts{
		ID: act.ID, Name: act.Name, OwnerID: act.OwnerID, Status: string(act.Status),
		StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		Path: []errs.RecordRef{
			{Type: "plan", ID: p.ID, Name: p.Name},
			{Type: "campaign", ID: campaign.ID, Name: campaign.Name},
			{Type: "initiative", ID: initiative.ID, Name: initiative.Name},
		},
		PlanYear:      p.Year,
		DefaultSpread: lineitem.SpreadMethod(p.DefaultSpread),
		Access:        plan.AccessFacts(p),
		PlanCurrency:  p.Currency, // 016 FR-015
	}
	if act.PlannedCost != nil {
		planned := act.PlannedCost.Amount
		f.PlannedCost = &planned
	}
	if p.DefaultFinanceBudgetID != "" {
		budget, found, err := x.settings.FinanceBudget(ctx, p.DefaultFinanceBudgetID)
		if err != nil {
			return lineitem.ActivityFacts{}, false, err
		}
		if found {
			f.DefaultFinanceBudget = budget.Name
		}
	}
	if f.BusinessUnit, err = x.settings.BusinessUnit(ctx, p.Attributes); err != nil {
		return lineitem.ActivityFacts{}, false, err
	}
	return f, true, nil
}

// FactsOf returns the facts of the stored activities among ids: their initiatives, campaigns and plans are
// read once for all of them, and kept for the transaction, before each activity's facts are put together.
func (x lineItemActivities) FactsOf(ctx context.Context, ids []string) (map[string]lineitem.ActivityFacts, error) {
	list, err := x.activities.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if err := x.readParents(ctx, list); err != nil {
		return nil, err
	}
	facts := make(map[string]lineitem.ActivityFacts, len(list))
	for _, act := range list {
		f, found, err := x.Facts(ctx, act.ID)
		if err != nil {
			return nil, err
		}
		if found {
			facts[act.ID] = f
		}
	}
	return facts, nil
}

// readParents reads the initiatives, campaigns and plans of the activities list, one level at a time.
func (x lineItemActivities) readParents(ctx context.Context, list []activity.Activity) error {
	initiativeIDs := make([]string, 0, len(list))
	for _, act := range list {
		initiativeIDs = append(initiativeIDs, act.InitiativeID)
	}
	initiatives, err := x.campaigns.InitiativesByIDs(ctx, initiativeIDs)
	if err != nil {
		return err
	}
	campaignIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		campaignIDs = append(campaignIDs, i.CampaignID)
	}
	campaignList, err := x.campaigns.CampaignsByIDs(ctx, campaignIDs)
	if err != nil {
		return err
	}
	planIDs := make([]string, 0, len(campaignList))
	for _, c := range campaignList {
		planIDs = append(planIDs, c.PlanID)
	}
	_, err = x.plans.ListByIDs(ctx, planIDs)
	return err
}

// RestartApproval sends a Submitted or Approved activity back to Created with the reasons (006's rule,
// which also tells approvals to end the open round).
func (x lineItemActivities) RestartApproval(ctx context.Context, a actor.Actor, id string, reasons []string) error {
	return x.service.RestartApproval(ctx, a, id, reasons)
}

// missing returns err, or, when there is none, an error saying that a record the data needs is not stored.
func missing(err error, kind, id string) error {
	if err != nil {
		return err
	}
	return fmt.Errorf("app: the %s %s is not stored", kind, id)
}

// lineItemBudgets implements lineitem.FinanceBudgets over the Settings repository (002): each item with its
// budget's year and name. Settings' readers give the budgets of one year only, and line items need every year.
type lineItemBudgets struct {
	repo stores.SettingsStore
}

// Item returns the finance budget item id with its budget's year and name.
func (x lineItemBudgets) Item(ctx context.Context, id string) (lineitem.BudgetItem, bool, error) {
	item, found, err := x.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil || !found {
		return lineitem.BudgetItem{}, false, err
	}
	budget, found, err := x.repo.GetFinanceBudget(ctx, item.BudgetID)
	if err != nil || !found {
		return lineitem.BudgetItem{}, false, missing(err, "finance budget", item.BudgetID)
	}
	return budgetItem(budget, item), true, nil
}

// Items returns every finance budget item with its budget's year and name.
func (x lineItemBudgets) Items(ctx context.Context) ([]lineitem.BudgetItem, error) {
	budgets, err := x.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return nil, err
	}
	items, err := x.repo.ListFinanceBudgetItems(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]lineitem.BudgetItem, 0, len(items))
	for _, item := range items {
		if i := slices.IndexFunc(budgets, func(b settings.FinanceBudget) bool { return b.ID == item.BudgetID }); i >= 0 {
			list = append(list, budgetItem(budgets[i], item))
		}
	}
	return list, nil
}

// budgetItem returns item of budget as line items need it.
func budgetItem(budget settings.FinanceBudget, item settings.FinanceBudgetItem) lineitem.BudgetItem {
	return lineitem.BudgetItem{
		ID: item.ID, Year: budget.Year, BudgetName: budget.Name, Name: item.Name, Code: item.Code, Active: item.Active,
	}
}

// lineItemPeople implements lineitem.People over the people repository and service (003).
type lineItemPeople struct {
	repo    stores.PeopleStore
	service *people.Service
}

// DisplayRateOf returns the display currency of the person personID with its rate (003 FR-023).
func (x lineItemPeople) DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error) {
	return x.service.DisplayRateOf(ctx, personID)
}

// Person returns the person id as records show them, active or not.
func (x lineItemPeople) Person(ctx context.Context, id string) (people.PersonRef, bool, error) {
	person, found, err := x.repo.Get(ctx, id)
	if err != nil || !found {
		return people.PersonRef{}, false, err
	}
	return person.Ref(), true, nil
}

// lineItemAttributes implements lineitem.Attributes with Settings' attribute rules (002) at level
// lineItem.
type lineItemAttributes struct {
	settings *settings.Service
}

// Check checks the line-item attribute values of a save (002's CheckAttributes).
func (x lineItemAttributes) Check(ctx context.Context, in settings.AttributeInput) (settings.AttributeResult, error) {
	return x.settings.CheckAttributes(ctx, settings.LevelLineItem, in)
}

// Missing lists the needed line-item attributes still empty (002's MissingAttributes).
func (x lineItemAttributes) Missing(ctx context.Context, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	return x.settings.MissingAttributes(ctx, settings.LevelLineItem, values, businessUnit)
}

// Describe gives the label and the text of an attribute value for the activity log (002's
// DescribeAttribute).
func (x lineItemAttributes) Describe(ctx context.Context, field string, value any) (string, string, error) {
	return x.settings.DescribeAttribute(ctx, field, value)
}
