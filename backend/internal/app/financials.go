package app

import (
	"context"
	"slices"

	"northstar/internal/activities"
	"northstar/internal/campaigns"
	"northstar/internal/financials"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The wiring of feature 015, financial transactions (D-196, D-251): the ports of internal/financials over
// plans (004), campaigns and initiatives (005), activities (006), line items (007), the recycle bin (001),
// Settings' rates (002), people's display currencies and teams (003), and access (014).

// financialPlanning is financials.Planning over the planning features. It reads the plan tree as the
// owning features order it — 005's campaigns and initiatives, 006's activities with their planned costs —
// and 007's line items with their USD planned cost at the current rates. It reads inside the caller's
// transaction and never writes.
type financialPlanning struct {
	plans      plans.Repository
	campaigns  *campaigns.Service
	activities *activities.Service
	lineItems  lineitems.Repository
	settings   *settings.Service
	people     *people.Service
	history    *history.Service
}

// The compiler checks here that the adapters implement the ports of financials.
var (
	_ financials.Planning = financialPlanning{}
	_ financials.Displays = financialDisplays{}
	_ financials.Access   = financialAccess{}
	_ financials.Rates    = (*settings.Service)(nil)
)

// Nodes returns every live plan, campaign, initiative, activity and line item, in the plan tree's order.
func (p financialPlanning) Nodes(ctx context.Context) ([]financials.Node, error) {
	all, err := p.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var nodes []financials.Node
	planOf := map[string]plans.Plan{} // activity id → its plan
	var activityIDs []string
	for _, plan := range all {
		nodes = append(nodes, financials.Node{
			Level: financials.LevelPlan, ID: plan.ID, PlanID: plan.ID, Name: plan.Name, PlanYear: plan.Year,
		})
		tree, err := p.campaigns.ListPlanCampaigns(ctx, actor.System, plan.ID)
		if err != nil {
			return nil, err
		}
		for ci, c := range tree.Items {
			nodes = append(nodes, financials.Node{
				Level: financials.LevelCampaign, ID: c.ID, ParentID: plan.ID, PlanID: plan.ID, Name: c.Name,
				PlanYear: plan.Year, Order: ci,
			})
			for ii, i := range c.Initiatives {
				nodes = append(nodes, financials.Node{
					Level: financials.LevelInitiative, ID: i.ID, ParentID: c.ID, PlanID: plan.ID, Name: i.Name,
					PlanYear: plan.Year, Order: ii,
				})
			}
		}
		summaries, err := p.activities.ListForPlan(ctx, actor.System, plan.ID) // actor.System: amounts in USD
		if err != nil {
			return nil, err
		}
		for order, a := range summaries {
			node := financials.Node{
				Level: financials.LevelActivity, ID: a.ID, ParentID: a.InitiativeID, PlanID: plan.ID, Name: a.Name,
				PlanYear: plan.Year, Order: order,
			}
			if a.PlannedCost != nil {
				planned := a.PlannedCost.Amount
				node.PlannedCost = &planned
			}
			nodes = append(nodes, node)
			planOf[a.ID] = plan
			activityIDs = append(activityIDs, a.ID)
		}
	}
	return p.addLineItems(ctx, nodes, planOf, activityIDs)
}

// addLineItems adds the line items of the activities, in creation order, with their USD planned cost at
// the current rate (007's PlannedCostUSD).
func (p financialPlanning) addLineItems(ctx context.Context, nodes []financials.Node, planOf map[string]plans.Plan,
	activityIDs []string) ([]financials.Node, error) {
	if len(activityIDs) == 0 {
		return nodes, nil
	}
	items, err := p.lineItems.ListByActivities(ctx, activityIDs) // ordered by id: creation order
	if err != nil {
		return nil, err
	}
	rates, err := p.settings.Rates(ctx)
	if err != nil {
		return nil, err
	}
	for order, li := range items {
		plan := planOf[li.ActivityID]
		node := financials.Node{
			Level: financials.LevelLineItem, ID: li.ID, ParentID: li.ActivityID, PlanID: plan.ID, Name: li.Name,
			PlanYear: plan.Year, Order: order,
		}
		if rate, known := rates[li.Currency]; known { // Settings never deletes a currency in use
			planned := lineitems.PlannedCostUSD(li, rate)
			node.PlannedCost = &planned
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// LineItemsInBin returns the ids of the line items in the recycle bin: deleted themselves or with a record
// above them, in entries not yet undone or restored (001's SnapshotsInBin).
func (p financialPlanning) LineItemsInBin(ctx context.Context) ([]string, error) {
	snapshots, err := p.history.SnapshotsInBin(ctx, "lineItem") // 007's kind
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(snapshots))
	for _, s := range snapshots {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// MyTeamPlanIDs returns the plans with at least one of the person's teams (003, as Home's My teams).
func (p financialPlanning) MyTeamPlanIDs(ctx context.Context, personID string) ([]string, error) {
	teams, err := p.people.TeamIDsOf(ctx, personID)
	if err != nil {
		return nil, err
	}
	all, err := p.plans.List(ctx)
	if err != nil {
		return nil, err
	}
	var mine []string
	for _, plan := range all {
		if slices.ContainsFunc(plan.TeamIDs, func(id string) bool { return slices.Contains(teams, id) }) {
			mine = append(mine, plan.ID)
		}
	}
	return mine, nil
}

// financialDisplays is financials.Displays over people's display currencies (003) and Settings' rates.
type financialDisplays struct {
	people *people.Service
}

// Display returns the person's display currency with its current rate; an unusable rate shows US dollars.
func (d financialDisplays) Display(ctx context.Context, personID string) (money.Display, error) {
	rate, err := d.people.DisplayRateOf(ctx, personID)
	if err != nil {
		return money.Display{}, err
	}
	return money.DisplayIn(rate.Code, rate.Rate), nil
}

// financialAccess is financials.Access (FR-028, D-252, D-301). Until feature 014's authz is merged, every
// signed-in person may do everything (D-004); 014's lane replaces these bodies with the checks of its
// permissions finance.view, finance.import and finance.link (014 plan, Table C additions), keeping this
// type, so nothing else changes.
type financialAccess struct{}

// SeeFinance allows every signed-in person until 014.
func (financialAccess) SeeFinance(context.Context, actor.Actor) error { return nil }

// SeeFinanceIn allows every signed-in person until 014.
func (financialAccess) SeeFinanceIn(context.Context, actor.Actor, string) error { return nil }

// Import allows every signed-in person until 014.
func (financialAccess) Import(context.Context, actor.Actor) error { return nil }

// Link allows every signed-in person until 014.
func (financialAccess) Link(context.Context, actor.Actor, string) error { return nil }

// VisiblePlanIDs gives every plan until 014.
func (financialAccess) VisiblePlanIDs(context.Context, actor.Actor) ([]string, error) {
	return nil, nil
}
