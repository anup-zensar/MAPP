package app

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/errs"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// The adapters below implement package lineitem's dependencies over the other areas (D-196), inside the caller's transaction.

type lineItemActivities struct {
	activities stores.ActivityStore
	campaigns  stores.CampaignStore
	plans      stores.PlanStore
	settings   *settings.Service
}

func (x lineItemActivities) Facts(ctx context.Context, id string) (lineitem.ActivityFacts, bool, error) {
	act, found, err := x.activities.Get(ctx, id)
	if err != nil || !found {
		return lineitem.ActivityFacts{}, false, err
	}
	path, p, err := x.pathOf(ctx, act.InitiativeID)
	if err != nil {
		return lineitem.ActivityFacts{}, false, err
	}
	f := lineitem.ActivityFacts{
		ID: act.ID, Name: act.Name, OwnerID: act.OwnerID, Status: string(act.Status),
		StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		Path:          path,
		PlanYear:      p.Year,
		DefaultSpread: lineitem.SpreadMethod(p.DefaultSpread),
		Access:        plan.AccessFacts(p),
		PlanCurrency:  p.Currency,
	}
	if act.PlannedCost != nil {
		planned := act.PlannedCost.Amount
		f.PlannedCost = &planned
	}
	if f.DefaultFinanceBudget, err = x.financeBudgetName(ctx, p.DefaultFinanceBudgetID); err != nil {
		return lineitem.ActivityFacts{}, false, err
	}
	if f.BusinessUnit, err = x.settings.BusinessUnit(ctx, p.Attributes); err != nil {
		return lineitem.ActivityFacts{}, false, err
	}
	return f, true, nil
}

// pathOf treats a missing initiative, campaign or plan as an error: the activity below them is stored.
func (x lineItemActivities) pathOf(ctx context.Context, initiativeID string) ([]errs.RecordRef, plan.Plan, error) {
	initiative, found, err := x.campaigns.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return nil, plan.Plan{}, missing(err, initiativeLevel, initiativeID)
	}
	c, found, err := x.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return nil, plan.Plan{}, missing(err, campaignLevel, initiative.CampaignID)
	}
	p, found, err := x.plans.Get(ctx, c.PlanID)
	if err != nil || !found {
		return nil, plan.Plan{}, missing(err, planKind, c.PlanID)
	}
	return []errs.RecordRef{
		{Type: planKind, ID: p.ID, Name: p.Name},
		{Type: campaignLevel, ID: c.ID, Name: c.Name},
		{Type: initiativeLevel, ID: initiative.ID, Name: initiative.Name},
	}, p, nil
}

// financeBudgetName is "" for no budget and for one that is not stored.
func (x lineItemActivities) financeBudgetName(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	budget, found, err := x.settings.FinanceBudget(ctx, id)
	if err != nil || !found {
		return "", err
	}
	return budget.Name, nil
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

func budgetItem(budget settings.FinanceBudget, item settings.FinanceBudgetItem) lineitem.BudgetItem {
	return lineitem.BudgetItem{
		ID: item.ID, Year: budget.Year, BudgetName: budget.Name, Name: item.Name, Code: item.Code, Active: item.Active,
	}
}

type lineItemPeople struct {
	repo    stores.PeopleStore
	service *people.Service
}

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

type lineItemAttributes struct {
	settings *settings.Service
}

func (x lineItemAttributes) Check(ctx context.Context, in settings.AttributeInput) (settings.AttributeResult, error) {
	return x.settings.CheckAttributes(ctx, settings.LevelLineItem, in)
}

func (x lineItemAttributes) Missing(ctx context.Context, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error) {
	return x.settings.MissingAttributes(ctx, settings.LevelLineItem, values, businessUnit)
}

func (x lineItemAttributes) Describe(ctx context.Context, field string, value any) (string, string, error) {
	return x.settings.DescribeAttribute(ctx, field, value)
}
