package app

import (
	"context"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/allwork"
	"mapp/internal/campaign"
	"mapp/internal/financials"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

// The wiring of feature 010, All work (D-196): allWorkSource implements allwork.Source over plans (004),
// campaigns and initiatives (005), activities (006), line items (007), people (003), Settings (002) and the
// recycle bin (001). Every value comes from the owning feature's own calculation: a plan's completeness from
// 004's Plans list; Approval progress, Timing and completeness from 005's plan tree; display status,
// completeness and goals from 006; USD amounts, spreads, SWBS status and completeness from 007's functions
// (010 plan, consolidation: "007's month amounts come through an internal/app adapter over 007's spread
// functions"). It reads inside the caller's transaction and never writes.

// allWorkSource is allwork.Source over the planning features.
type allWorkSource struct {
	plans          *plan.Service
	plansRepo      stores.PlanStore
	campaigns      *campaign.Service
	campaignsRepo  stores.CampaignStore
	activities     *activity.Service
	activitiesRepo stores.ActivityStore
	lineItems      *lineitem.Service
	lineItemsRepo  stores.LineItemStore
	settings       *settings.Service
	budgetItems    lineitem.FinanceBudgets // 002's finance budget items, as line items read them
	workflows      activity.Workflows      // 008's approval workflows, as activities read them
	people         *people.Service
	peopleRepo     stores.PeopleStore
	history        *history.Service
	financials     *financials.Service // 015's GSAP figures of line items (financials.go)
	financeAccess  financials.Access   // who may see them (015 FR-028)
}

// The compiler checks here that allWorkSource implements allwork.Source.
var _ allwork.Source = allWorkSource{}

// Year returns every plan of year with its campaigns, initiatives, activities and line items, each with its
// position in the plan tree (allwork data model §2). Records in the recycle bin are not stored, so they are
// never among them. The values that only some columns show — a plan's completeness, the plan goals, the
// activities' goals — are read only when columns names one of those columns, because the owning features
// work them out record by record.
func (s allWorkSource) Year(ctx context.Context, year int, today date.Date, columns []string) (allwork.YearData, error) {
	yearPlans, err := plansOfYear(ctx, s.plansRepo, year) // rows, totals, calendar and export: what the person may see
	if err != nil {
		return allwork.YearData{Nodes: []allwork.Node{}}, err
	}
	data, err := s.nodesOf(ctx, yearPlans, year, columns)
	if err != nil || len(data.Nodes) == 0 {
		return data, err
	}
	if err := s.addFinance(ctx, &data); err != nil { // feature 015 (financials.go)
		return allwork.YearData{}, err
	}
	return data, nil
}

// nodesOf returns the nodes of the plans yearPlans of year and of everything below them, without the GSAP
// figures.
func (s allWorkSource) nodesOf(ctx context.Context, yearPlans []plan.Plan, year int, columns []string) (allwork.YearData, error) {
	data := allwork.YearData{Nodes: []allwork.Node{}}
	if len(yearPlans) == 0 {
		return data, nil
	}
	var err error
	y := yearReader{s: s, owners: map[string]*allwork.Owner{}, columns: columns, plans: map[string]plan.Plan{}}
	if err := y.loadAttributes(ctx); err != nil {
		return allwork.YearData{}, err
	}
	if y.rates, err = s.settings.Rates(ctx); err != nil { // 016: each plan currency's amounts converted to USD once
		return allwork.YearData{}, err
	}
	missing := map[string]int{}
	if y.needs("completeness") {
		if missing, err = s.planMissing(ctx, year); err != nil {
			return allwork.YearData{}, err
		}
	}
	if err := y.addPlans(ctx, yearPlans, missing); err != nil {
		return allwork.YearData{}, err
	}
	data.Nodes = y.nodes
	return data, nil
}

// addPlans adds the plans with their own missing fields and everything below them.
func (y *yearReader) addPlans(ctx context.Context, yearPlans []plan.Plan, missing map[string]int) error {
	if err := y.s.readTrees(ctx, yearPlans); err != nil {
		return err
	}
	for _, p := range yearPlans {
		if err := y.addPlan(ctx, p, missing[p.ID]); err != nil {
			return err
		}
	}
	if err := y.addActivityDetails(ctx); err != nil {
		return err
	}
	return y.addLineItems(ctx)
}

// planMissing returns the number of each plan's own needed-to-publish fields still empty, from 004's Plans
// list: one call for every plan of the year.
func (s allWorkSource) planMissing(ctx context.Context, year int) (map[string]int, error) {
	list, err := s.plans.List(ctx, actor.System, plan.ListFilter{Scope: plan.ScopeAll, Year: &year})
	if err != nil {
		return nil, err
	}
	missing := make(map[string]int, len(list.Items))
	for _, p := range list.Items {
		missing[p.ID] = len(p.Completeness.Missing)
	}
	return missing, nil
}

// addPlan adds the plan p with its own missing fields, its campaigns and initiatives as 005's plan tree
// orders them, and its activities as 006's plan-tree rows order them.
func (y *yearReader) addPlan(ctx context.Context, p plan.Plan, missing int) error {
	owner, err := y.owner(ctx, p.OwnerID)
	if err != nil {
		return err
	}
	units, err := y.s.settings.BusinessUnit(ctx, p.Attributes)
	if err != nil {
		return err
	}
	fields, err := y.planFields(ctx, p)
	if err != nil {
		return err
	}
	y.plans[p.ID] = p
	planNode := allwork.Node{
		Level: allwork.LevelPlan, ID: p.ID, PlanID: p.ID, Name: p.Name, Description: p.Description, Owner: owner,
		Status: string(p.Status), Missing: missing, Budget: y.usd(p.ID, p.Budget), BudgetPlan: y.inPlan(p.ID, p.Budget),
		CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Strategic: p.AOP, TeamIDs: append([]string{}, p.TeamIDs...),
		BusinessUnitIDs: units, Fields: fields,
	}
	y.planFacts(&planNode)
	planNode.Attrs = y.attrValues(settings.LevelPlan, p.Attributes)
	planNode.PhasingMethod = string(p.BudgetPhasing.Normalised().Method)
	planNode.PhasedPlan, planNode.PhasedBudgetUSD = y.phased(p.ID, p.BudgetPhasing.MonthAmounts(p.Budget, p.Year))
	y.nodes = append(y.nodes, planNode)
	if err = y.addCampaigns(ctx, p.ID); err != nil {
		return err
	}
	return y.addActivities(ctx, p.ID)
}

func (y *yearReader) addCampaigns(ctx context.Context, planID string) error {
	tree, err := y.s.campaigns.ListPlanCampaigns(ctx, actor.System, planID)
	if err != nil {
		return err
	}
	for ci, c := range tree.Items {
		err = y.addContainer(ctx, container{
			level: allwork.LevelCampaign, id: c.ID, parentID: planID, planID: planID, order: ci, name: c.Name,
			description: c.Description, ownerID: c.OwnerID, start: c.StartDate, end: c.EndDate, budget: c.Budget,
			progress: c.ApprovalProgress, timing: c.Timing, completeness: c.Completeness, meta: c.Meta,
			attributes: c.Attributes, phasing: c.BudgetPhasing,
		})
		if err != nil {
			return err
		}
		for ii, i := range c.Initiatives {
			err = y.addContainer(ctx, container{
				level: allwork.LevelInitiative, id: i.ID, parentID: c.ID, planID: planID, order: ii, name: i.Name,
				description: i.Description, ownerID: i.OwnerID, start: i.StartDate, end: i.EndDate, budget: i.Budget,
				progress: i.ApprovalProgress, timing: i.Timing, completeness: i.Completeness, meta: i.Meta,
				attributes: i.Attributes, phasing: i.BudgetPhasing,
			})
			if err != nil {
				return err
			}
			y.initiativeIDs = append(y.initiativeIDs, i.ID)
		}
	}
	return nil
}

func (y *yearReader) addActivities(ctx context.Context, planID string) error {
	summaries, err := y.s.activities.ListForPlan(ctx, actor.System, planID)
	if err != nil {
		return err
	}
	if y.activities == nil {
		y.activities = map[string]int{}
	}
	for order, a := range summaries {
		var planned *money.Amount // in the plan currency
		if a.PlannedCost != nil {
			planned = &a.PlannedCost.Amount
		}
		months, _ := lineitem.ActivityMonths(lineitem.ActivityFacts{StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing})
		y.activities[a.ID] = len(y.nodes)
		y.nodes = append(y.nodes, allwork.Node{
			Level: allwork.LevelActivity, ID: a.ID, ParentID: a.InitiativeID, PlanID: planID, Order: order, Name: a.Name,
			Owner: ownerOf(a.Owner), StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing, Months: months,
			Status: string(a.DisplayStatus), Missing: a.Completeness.MissingFields, PlannedCost: y.usd(planID, planned),
			PlannedCostPlan: y.inPlan(planID, planned), ActivityTypeID: a.ActivityType.ID,
			Fields: map[string]allwork.Value{
				"activity.type":    {Text: a.ActivityType.Name, Inactive: !a.ActivityType.Active},
				"activity.ongoing": {Yes: &a.Ongoing},
			},
		})
		y.planFacts(&y.nodes[len(y.nodes)-1])
	}
	return nil
}

// plansOfYear returns the plans of year the person acting may see (014 FR-007, SC-004).
func plansOfYear(ctx context.Context, repo stores.PlanStore, year int) ([]plan.Plan, error) {
	all, err := repo.ListInScope(ctx, authz.ScopeOf(ctx))
	return slices.DeleteFunc(all, func(p plan.Plan) bool { return p.Year != year }), err
}
