package app

import (
	"context"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/financials"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/money"
)

// Nodes returns every live plan, campaign, initiative, activity and line item, in the plan tree's order. It
// reads each level once for all the plans and works out nothing the figures of financials do not need.
func (p financialPlanning) Nodes(ctx context.Context) ([]financials.Node, error) {
	all, err := p.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	return p.nodesOf(ctx, all)
}

func (p financialPlanning) PlanNodes(ctx context.Context, planIDs []string) ([]financials.Node, error) {
	all, err := p.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	asked := slices.DeleteFunc(all, func(p plan.Plan) bool { return !slices.Contains(planIDs, p.ID) })
	return p.nodesOf(ctx, asked)
}

func (p financialPlanning) PlanIDsOfYear(ctx context.Context, year int) ([]string, error) {
	ofYear, err := p.plans.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	planIDs := make([]string, 0, len(ofYear))
	for _, p := range ofYear {
		planIDs = append(planIDs, p.ID)
	}
	return planIDs, nil
}

func (p financialPlanning) nodesOf(ctx context.Context, all []plan.Plan) ([]financials.Node, error) {
	planIDs := make([]string, 0, len(all))
	for _, p := range all {
		planIDs = append(planIDs, p.ID)
	}
	outlines, err := p.campaigns.PlanOutlines(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	o := outline{plans: all, planOf: map[string]plan.Plan{}}
	for _, p := range all {
		o.nodes = append(o.nodes, financials.Node{
			Level: financials.LevelPlan, ID: p.ID, PlanID: p.ID, Name: p.Name, PlanYear: p.Year,
		})
		for ci, c := range outlines[p.ID] {
			o.nodes = append(o.nodes, financials.Node{
				Level: financials.LevelCampaign, ID: c.Campaign.ID, ParentID: p.ID, PlanID: p.ID,
				Name: c.Campaign.Name, PlanYear: p.Year, Order: ci,
			})
			for ii, i := range c.Initiatives {
				o.nodes = append(o.nodes, financials.Node{
					Level: financials.LevelInitiative, ID: i.ID, ParentID: c.Campaign.ID, PlanID: p.ID, Name: i.Name,
					PlanYear: p.Year, Order: ii,
				})
				o.planOf[i.ID] = p
				o.initiativeIDs = append(o.initiativeIDs, i.ID)
			}
		}
	}
	return p.addActivities(ctx, o)
}

// outline is the plans with their campaign and initiative nodes, and the plan of each initiative.
type outline struct {
	nodes         []financials.Node
	plans         []plan.Plan
	planOf        map[string]plan.Plan // initiative id → its plan
	initiativeIDs []string
}

// addActivities adds the activities of the initiatives, each plan's in the order of its list of activities.
func (p financialPlanning) addActivities(ctx context.Context, o outline) ([]financials.Node, error) {
	nodes := o.nodes
	list, err := p.activities.ListByInitiatives(ctx, o.initiativeIDs)
	if err != nil {
		return nil, err
	}
	rates, err := p.settings.Rates(ctx)
	if err != nil {
		return nil, err
	}
	ofPlan := map[string][]activity.Activity{}
	for _, a := range list {
		planID := o.planOf[a.InitiativeID].ID
		ofPlan[planID] = append(ofPlan[planID], a)
	}
	activityPlans := map[string]plan.Plan{}
	var activityIDs []string
	for _, p := range o.plans {
		inPlan := ofPlan[p.ID]
		activity.SortInPlanOrder(inPlan)
		for order, a := range inPlan {
			node, err := activityNode(a, p, order, rates)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, node)
			activityPlans[a.ID] = p
			activityIDs = append(activityIDs, a.ID)
		}
	}
	return p.addLineItems(ctx, nodes, activityPlans, activityIDs)
}

// activityNode is the activity a with its planned cost in USD, a plan currency at the current rate.
func activityNode(a activity.Activity, p plan.Plan, order int, rates money.Rates) (financials.Node, error) {
	node := financials.Node{
		Level: financials.LevelActivity, ID: a.ID, ParentID: a.InitiativeID, PlanID: p.ID, Name: a.Name,
		PlanYear: p.Year, Order: order,
	}
	if a.PlannedCost == nil {
		return node, nil
	}
	planned := a.PlannedCost.Amount
	if c := a.PlannedCost.Currency; c != "" && c != money.BaseCurrency {
		var err error
		if planned, err = rates.ToUSD(*a.PlannedCost); err != nil {
			return financials.Node{}, err
		}
	}
	node.PlannedCost = &planned
	return node, nil
}

// addLineItems adds the line items of the activities with their USD planned cost at the current rate.
func (p financialPlanning) addLineItems(ctx context.Context, nodes []financials.Node, planOf map[string]plan.Plan,
	activityIDs []string) ([]financials.Node, error) {
	if len(activityIDs) == 0 {
		return nodes, nil
	}
	items, err := p.lineItems.ListByActivities(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	rates, err := p.settings.Rates(ctx)
	if err != nil {
		return nil, err
	}
	for order, li := range items {
		p := planOf[li.ActivityID]
		node := financials.Node{
			Level: financials.LevelLineItem, ID: li.ID, ParentID: li.ActivityID, PlanID: p.ID, Name: li.Name,
			PlanYear: p.Year, Order: order,
		}
		if rate, known := rates[li.Currency]; known { // Settings never deletes a currency in use
			planned := lineitem.PlannedCostUSD(li, rate)
			node.PlannedCost = &planned
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}
