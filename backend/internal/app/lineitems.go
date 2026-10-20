package app

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"northstar/internal/activities"
	"northstar/internal/campaigns"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/settings"
)

// The wiring of feature 007, line items (D-196): the adapters that implement the dependency interfaces of
// package lineitems over the other features, and the ones that hand line items to activities (006) and to
// the plans' level registry (004). They run inside the caller's transaction.

// lineItemDeps returns what the line-items service reads of the other features.
func lineItemDeps(settingsService *settings.Service, settingsRepo settings.Repository, activitiesService *activities.Service,
	activitiesRepo activities.Repository, campaignsRepo campaigns.Repository, plansRepo plans.Repository,
	peopleRepo people.Repository, peopleService *people.Service) lineitems.Deps {
	return lineitems.Deps{
		Activities: lineItemActivities{
			activities: activitiesRepo, campaigns: campaignsRepo, plans: plansRepo,
			settings: settingsService, service: activitiesService,
		},
		Currencies:     settingsService,
		FinanceBudgets: lineItemBudgets{repo: settingsRepo},
		Attributes:     lineItemAttributes{settings: settingsService},
		People:         lineItemPeople{repo: peopleRepo, service: peopleService},
	}
}

// lineItemActivities implements lineitems.Activities over the activities (006), campaigns (005) and plans
// (004) repositories and Settings (002); RestartApproval is 006's.
type lineItemActivities struct {
	activities activities.Repository
	campaigns  campaigns.Repository
	plans      plans.Repository
	settings   *settings.Service
	service    *activities.Service
}

// Facts returns the activity id with its initiative, campaign and plan, and false when it is not stored.
func (x lineItemActivities) Facts(ctx context.Context, id string) (lineitems.ActivityFacts, bool, error) {
	act, found, err := x.activities.Get(ctx, id)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, err
	}
	initiative, found, err := x.campaigns.GetInitiative(ctx, act.InitiativeID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "initiative", act.InitiativeID)
	}
	campaign, found, err := x.campaigns.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "campaign", initiative.CampaignID)
	}
	plan, found, err := x.plans.Get(ctx, campaign.PlanID)
	if err != nil || !found {
		return lineitems.ActivityFacts{}, false, missing(err, "plan", campaign.PlanID)
	}
	f := lineitems.ActivityFacts{
		ID: act.ID, Name: act.Name, OwnerID: act.OwnerID, Status: string(act.Status),
		StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		Path: []errs.RecordRef{
			{Type: "plan", ID: plan.ID, Name: plan.Name},
			{Type: "campaign", ID: campaign.ID, Name: campaign.Name},
			{Type: "initiative", ID: initiative.ID, Name: initiative.Name},
		},
		PlanYear:      plan.Year,
		DefaultSpread: lineitems.SpreadMethod(plan.DefaultSpread),
		Access:        plans.AccessFacts(plan),
		PlanCurrency:  plan.Currency, // 016 FR-015
	}
	if act.PlannedCost != nil {
		planned := act.PlannedCost.Amount
		f.PlannedCost = &planned
	}
	if plan.DefaultFinanceBudgetID != "" {
		budget, found, err := x.settings.FinanceBudget(ctx, plan.DefaultFinanceBudgetID)
		if err != nil {
			return lineitems.ActivityFacts{}, false, err
		}
		if found {
			f.DefaultFinanceBudget = budget.Name
		}
	}
	if f.BusinessUnit, err = x.settings.BusinessUnit(ctx, plan.Attributes); err != nil {
		return lineitems.ActivityFacts{}, false, err
	}
	return f, true, nil
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

// lineItemBudgets implements lineitems.FinanceBudgets over the Settings repository (002): each item with its
// budget's year and name. Settings' readers give the budgets of one year only, and line items need every year.
type lineItemBudgets struct {
	repo settings.Repository
}

// Item returns the finance budget item id with its budget's year and name.
func (x lineItemBudgets) Item(ctx context.Context, id string) (lineitems.BudgetItem, bool, error) {
	item, found, err := x.repo.GetFinanceBudgetItem(ctx, id)
	if err != nil || !found {
		return lineitems.BudgetItem{}, false, err
	}
	budget, found, err := x.repo.GetFinanceBudget(ctx, item.BudgetID)
	if err != nil || !found {
		return lineitems.BudgetItem{}, false, missing(err, "finance budget", item.BudgetID)
	}
	return budgetItem(budget, item), true, nil
}

// Items returns every finance budget item with its budget's year and name.
func (x lineItemBudgets) Items(ctx context.Context) ([]lineitems.BudgetItem, error) {
	budgets, err := x.repo.ListFinanceBudgets(ctx)
	if err != nil {
		return nil, err
	}
	items, err := x.repo.ListFinanceBudgetItems(ctx)
	if err != nil {
		return nil, err
	}
	list := make([]lineitems.BudgetItem, 0, len(items))
	for _, item := range items {
		if i := slices.IndexFunc(budgets, func(b settings.FinanceBudget) bool { return b.ID == item.BudgetID }); i >= 0 {
			list = append(list, budgetItem(budgets[i], item))
		}
	}
	return list, nil
}

// budgetItem returns item of budget as line items need it.
func budgetItem(budget settings.FinanceBudget, item settings.FinanceBudgetItem) lineitems.BudgetItem {
	return lineitems.BudgetItem{
		ID: item.ID, Year: budget.Year, BudgetName: budget.Name, Name: item.Name, Code: item.Code, Active: item.Active,
	}
}

// lineItemPeople implements lineitems.People over the people repository and service (003).
type lineItemPeople struct {
	repo    people.Repository
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

// lineItemAttributes implements lineitems.Attributes with Settings' attribute rules (002) at level
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

// The line-item problem codes activities shows in its Ready to submit panel (007 data model §3.6).
const (
	lineItemMissingField         = "missingField"
	lineItemSplitIncomplete      = "splitIncomplete"
	lineItemInactiveBudgetItem   = "inactiveFinanceBudgetItem"
	lineItemCustomSpreadMismatch = "customSpreadMismatch"
)

// lineItemsForActivities is activities.LineItems over the line-items service.
type lineItemsForActivities struct {
	svc *lineitems.Service
}

// Facts returns each activity's line items in short, with every missing field of each incomplete line item
// as a problem.
func (x lineItemsForActivities) Facts(ctx context.Context, activityIDs []string) (map[string]activities.LineItemFacts, error) {
	summaries, err := x.svc.Summaries(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	facts := make(map[string]activities.LineItemFacts, len(summaries))
	for id, s := range summaries {
		f := activities.LineItemFacts{Count: s.Count, PlannedCostPlan: s.PlannedCostPlan, Currency: s.Currency,
			AutoSpread: s.AutoSpread, CustomSpread: s.CustomSpread}
		for _, item := range s.Incomplete {
			for _, m := range item.Missing {
				f.Problems = append(f.Problems, lineItemProblem(item, m))
			}
		}
		facts[id] = f
	}
	return facts, nil
}

// lineItemProblem is one missing field of a line item as a problem of its activity.
func lineItemProblem(item lineitems.Incomplete, m settings.MissingField) activities.LineItemProblem {
	p := activities.LineItemProblem{
		LineItemID: item.LineItemID, LineItemName: item.LineItemName, Field: m.Field,
		Code: lineItemMissingField, Message: m.Label + " — needed to submit",
	}
	switch {
	case m.Field == "spread":
		p.Code, p.Message = lineItemCustomSpreadMismatch, m.Label
	case m.Field == "financeBudgetItemId" && m.Label != "Finance budget item":
		p.Code, p.Message = lineItemInactiveBudgetItem, m.Label
	case m.Reason != "":
		p.Code, p.Message = lineItemSplitIncomplete, m.Reason
	}
	return p
}

// CopyAll copies every line item of an activity into another activity: the copies of an activity, and of
// the campaigns and initiatives above it (006 FR-030, 005 FR-027).
func (x lineItemsForActivities) CopyAll(ctx context.Context, a actor.Actor, fromActivityID, toActivityID string, yearShift int) (int, error) {
	return x.svc.CopyLineItems(ctx, a, fromActivityID, toActivityID, yearShift)
}

// DatesChanged needs nothing stored: a line item spread Evenly, First or Last month follows its activity's
// dates when it is read, and a custom spread is kept as typed (007 FR-022, D-141).
func (lineItemsForActivities) DatesChanged(context.Context, actor.Actor, activities.Activity, activities.Activity) error {
	return nil
}

// lineItemLevel is the line-item level of feature 004's registry: how many line items each plan has and
// those still incomplete, below the plan's activities, and their copies in a plan copy (the line items of
// every activity the activity level copied, each copied once).
func lineItemLevel(svc *lineitems.Service, activitiesRepo activities.Repository, campaignsRepo campaigns.Repository) plans.Level {
	return plans.Level{
		Name: "lineItem",
		AddFacts: func(ctx context.Context, planIDs []string, facts map[string]*plans.Facts) error {
			for _, planID := range planIDs {
				activityIDs, err := activitiesOfPlan(ctx, planID, activitiesRepo, campaignsRepo)
				if err != nil {
					return err
				}
				summaries, err := svc.Summaries(ctx, activityIDs)
				if err != nil {
					return err
				}
				var attention []plans.AttentionItem
				for _, id := range activityIDs { // in the activities' order
					s, found := summaries[id]
					if !found {
						continue
					}
					facts[planID].LineItems += s.Count
					for _, item := range s.Incomplete {
						attention = append(attention, plans.AttentionItem{
							Record:  errs.RecordRef{Type: "lineItem", ID: item.LineItemID, Name: item.LineItemName},
							Missing: item.Missing,
						})
					}
				}
				facts[planID].NeedsAttention = append(facts[planID].NeedsAttention, attention...)
			}
			return nil
		},
		Copy: func(ctx context.Context, job *plans.CopyJob) error {
			newIDs, err := svc.CopyForActivities(ctx, job.Actor, job.NewIDs, job.Years)
			if err != nil {
				return err
			}
			maps.Copy(job.NewIDs, newIDs)
			job.Counts["lineItem"] += len(newIDs)
			return nil
		},
	}
}

// activitiesOfPlan returns the ids of the activities of the plan, campaign by campaign, initiative by
// initiative.
func activitiesOfPlan(ctx context.Context, planID string, activitiesRepo activities.Repository, campaignsRepo campaigns.Repository) ([]string, error) {
	campaignList, err := campaignsRepo.CampaignsOfPlan(ctx, planID)
	if err != nil {
		return nil, err
	}
	campaignIDs := make([]string, 0, len(campaignList))
	for _, c := range campaignList {
		campaignIDs = append(campaignIDs, c.ID)
	}
	initiatives, err := campaignsRepo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return nil, err
	}
	initiativeIDs := make([]string, 0, len(initiatives))
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	list, err := activitiesRepo.ListByInitiatives(ctx, initiativeIDs)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, a := range list {
		if !slices.Contains(ids, a.ID) {
			ids = append(ids, a.ID)
		}
	}
	return ids, nil
}

// Items returns the line items of an activity in short, with their planned costs in the plan currency (016
// FR-026, the Split dialog).
func (x lineItemsForActivities) Items(ctx context.Context, activityID string) ([]activities.LineItemBrief, error) {
	briefs, err := x.svc.Briefs(ctx, activityID)
	if err != nil {
		return nil, err
	}
	items := make([]activities.LineItemBrief, 0, len(briefs))
	for _, b := range briefs {
		items = append(items, activities.LineItemBrief{ID: b.ID, Name: b.Name, PlannedCost: b.PlannedCost, PlannedCostPlan: b.PlannedCostPlan})
	}
	return items, nil
}

// MoveToActivity moves line items to another activity: the split of an activity (016 FR-028).
func (x lineItemsForActivities) MoveToActivity(ctx context.Context, a actor.Actor, lineItemIDs []string, toActivityID string) error {
	return x.svc.MoveToActivity(ctx, a, lineItemIDs, toActivityID)
}
