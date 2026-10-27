package app

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"mapp/internal/activity"
	"mapp/internal/allwork"
	"mapp/internal/campaign"
	"mapp/internal/lineitem"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// container is a campaign or initiative as 005's plan tree shows it.
type container struct {
	level                allwork.Level
	id, parentID, planID string
	order                int
	name, description    string
	ownerID              *string
	start, end           *date.Date
	budget               *money.Money
	progress             campaign.ApprovalSummary
	timing               *string
	completeness         campaign.RecordCompleteness
	meta                 record.Meta
	attributes           settings.AttributeValues
	phasing              rollup.PhasingView // 016: the budget by month, in the plan currency
}

// addContainer adds the node of a campaign or initiative.
func (y *yearReader) addContainer(ctx context.Context, c container) error {
	var owner *allwork.Owner
	if c.ownerID != nil {
		var err error
		if owner, err = y.owner(ctx, *c.ownerID); err != nil {
			return err
		}
	}
	node := allwork.Node{
		Level: c.level, ID: c.id, ParentID: c.parentID, PlanID: c.planID, Order: c.order, Name: c.name,
		Description: c.description, Owner: owner, StartDate: c.start, EndDate: c.end,
		Status: c.progress.State, Approved: c.progress.Approved, Total: c.progress.Total,
		Missing: len(c.completeness.Missing), CreatedAt: c.meta.CreatedAt, UpdatedAt: c.meta.UpdatedAt,
	}
	if c.timing != nil {
		node.Timing = *c.timing
	}
	if c.budget != nil {
		node.Budget, node.BudgetPlan = y.usd(c.planID, &c.budget.Amount), c.budget
	}
	y.planFacts(&node)
	monthly := make([]money.Amount, 0, len(c.phasing.Monthly))
	for _, m := range c.phasing.Monthly {
		monthly = append(monthly, m.Amount)
	}
	node.PhasingMethod = string(c.phasing.Method)
	node.PhasedPlan, node.PhasedBudgetUSD = y.phased(c.planID, monthly)
	level := settings.LevelCampaign
	if c.level == allwork.LevelInitiative {
		level = settings.LevelInitiative
	}
	node.Fields = map[string]allwork.Value{string(c.level) + ".description": {Text: c.description}}
	y.addAttributes(node.Fields, level, c.attributes)
	node.Attrs = y.attrValues(level, c.attributes)
	y.nodes = append(y.nodes, node)
	return nil
}

// addActivityDetails adds to the activity nodes what 006's plan-tree rows leave out: external name,
// description, workflow, attributes and dates of change from the stored activities and, when the view shows
// INQ, MQL or revenue, the goals as 006 shows them (typed, or calculated with the plan's model).
func (y *yearReader) addActivityDetails(ctx context.Context) error {
	if len(y.activities) == 0 {
		return nil
	}
	stored, err := y.s.activitiesRepo.ListByInitiatives(ctx, y.initiativeIDs)
	if err != nil {
		return err
	}
	y.stored = make(map[string]string, len(stored))
	for _, a := range stored {
		i, found := y.activities[a.ID]
		if !found {
			continue
		}
		if err := y.addStoredActivity(ctx, &y.nodes[i], a); err != nil {
			return err
		}
	}
	if !y.needs("inq", "mql", "revenue") {
		return nil
	}
	return y.addActivityGoals(ctx)
}

func (y *yearReader) addStoredActivity(ctx context.Context, n *allwork.Node, a activity.Activity) error {
	n.ExternalName, n.Description, n.CreatedAt, n.UpdatedAt = a.ExternalName, a.Description, a.CreatedAt, a.UpdatedAt
	y.stored[a.ID] = string(a.Status)
	n.Fields["activity.externalName"] = allwork.Value{Text: a.ExternalName}
	n.Fields["activity.description"] = allwork.Value{Text: a.Description}
	n.Fields["activity.setLaterStages"] = allwork.Value{Yes: &a.SetLaterStagesMyself}
	if a.ApprovalWorkflowID != "" {
		w, found, err := y.s.workflows.GetWorkflow(ctx, a.ApprovalWorkflowID)
		if err != nil {
			return err
		}
		if found {
			n.Fields["activity.approvalWorkflow"] = allwork.Value{Text: w.Name, Inactive: !w.Active}
		}
	}
	y.addAttributes(n.Fields, settings.LevelActivity, a.Attributes)
	n.Attrs = y.attrValues(settings.LevelActivity, a.Attributes)
	return nil
}

func (y *yearReader) addActivityGoals(ctx context.Context) error {
	goals, err := y.s.activities.Summaries(ctx, y.initiativeIDs)
	if err != nil {
		return err
	}
	for _, g := range goals {
		if i, found := y.activities[g.ID]; found {
			n := &y.nodes[i]
			n.INQ, n.MQL = g.INQ, g.MQL
			if g.Revenue != nil {
				n.Revenue = &g.Revenue.Amount
			}
		}
	}
	return nil
}

// addLineItems adds the line items of the year's activities in creation order, each with its USD planned
// cost and spread (007's PlannedCostUSD, SpreadMonths and USDMonths at today's rates), its SWBS request
// status and its completeness (007's Summaries).
func (y *yearReader) addLineItems(ctx context.Context) error {
	if len(y.activities) == 0 {
		return nil
	}
	activityIDs := slices.Collect(maps.Keys(y.activities))
	items, err := y.s.lineItemsRepo.ListByActivities(ctx, activityIDs) // ordered by id: creation order
	if err != nil || len(items) == 0 {
		return err
	}
	missing, err := y.lineItemsMissing(ctx, activityIDs)
	if err != nil {
		return err
	}
	rates, err := y.s.settings.Rates(ctx)
	if err != nil {
		return err
	}
	for order, li := range items {
		node, err := y.lineItemNode(ctx, li, rates)
		if err != nil {
			return err
		}
		node.Order, node.Missing = order, missing[li.ID]
		y.nodes = append(y.nodes, node)
	}
	return nil
}

// lineItemsMissing returns how many needed fields each incomplete line item of the activities still lacks.
func (y *yearReader) lineItemsMissing(ctx context.Context, activityIDs []string) (map[string]int, error) {
	summaries, err := y.s.lineItems.Summaries(ctx, activityIDs)
	if err != nil {
		return nil, err
	}
	missing := map[string]int{}
	for _, summary := range summaries {
		for _, incomplete := range summary.Incomplete {
			missing[incomplete.LineItemID] = len(incomplete.Missing)
		}
	}
	return missing, nil
}

// lineItemNode returns the node of the line item li, whose activity's node is already added.
func (y *yearReader) lineItemNode(ctx context.Context, li lineitem.LineItem, rates money.Rates) (allwork.Node, error) {
	act := y.nodes[y.activities[li.ActivityID]]
	f := lineitem.ActivityFacts{
		ID: act.ID, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
		Status: y.stored[li.ActivityID],
	}
	planID := act.PlanID
	rate, known := rates[li.Currency]
	if !known { // Settings never deletes a currency in use (domain model §6)
		return allwork.Node{}, fmt.Errorf("app: line item %s has the unknown currency %q", li.ID, li.Currency)
	}
	owner, err := y.owner(ctx, li.OwnerID)
	if err != nil {
		return allwork.Node{}, err
	}
	plannedUSD := lineitem.PlannedCostUSD(li, rate)
	planRate := y.planRate(planID)
	plannedPlan := money.Convert(li.PlannedCost, rate, planRate) // 016 FR-015
	node := allwork.Node{
		Level: allwork.LevelLineItem, ID: li.ID, ParentID: li.ActivityID, PlanID: planID, Name: li.Name,
		Description: li.Description, Owner: owner, PlannedCost: &plannedUSD,
		LocalPlannedCost: &money.Money{Amount: li.PlannedCost, Currency: li.Currency},
		Swbs:             string(lineitem.SwbsStatusOf(li, f)), CreatedAt: li.CreatedAt, UpdatedAt: li.UpdatedAt,
		MonthsUSD: []allwork.MonthAmount{}, MonthsPlan: []allwork.PlanMonth{}, PlannedCostPlan: y.inPlan(planID, &plannedPlan),
		Fields: map[string]allwork.Value{
			"lineItem.description":      {Text: li.Description},
			"lineItem.localPlannedCost": {Money: &money.Money{Amount: li.PlannedCost, Currency: li.Currency}},
			"lineItem.currency":         {Text: li.Currency},
			"lineItem.spread":           {Text: spreadWords[string(li.Spread.Method)]},
			"lineItem.requestSwbs":      {Yes: &li.RequestSwbs},
		},
	}
	if err := y.addFinanceFields(ctx, node.Fields, li); err != nil {
		return allwork.Node{}, err
	}
	y.addAttributes(node.Fields, settings.LevelLineItem, li.Attributes)
	node.Attrs = y.attrValues(settings.LevelLineItem, li.Attributes)
	y.planFacts(&node)
	months, spread := lineitem.SpreadMonths(li, f)
	if !spread {
		node.UnspreadUSD, node.UnspreadPlan = plannedUSD, plannedPlan
		return node, nil
	}
	for i, amount := range lineitem.USDMonths(months, rate) {
		node.MonthsUSD = append(node.MonthsUSD, allwork.MonthAmount{Month: months[i].Month, USD: amount})
	}
	for i, amount := range lineitem.ConvertMonths(months, rate, planRate) {
		node.MonthsPlan = append(node.MonthsPlan, allwork.PlanMonth{Month: months[i].Month, Amount: amount})
	}
	return node, nil
}

func (y *yearReader) addFinanceFields(ctx context.Context, fields map[string]allwork.Value, li lineitem.LineItem) error {
	if li.ContractedAmount != nil {
		fields["lineItem.contractedAmount"] = allwork.Value{Money: &money.Money{Amount: *li.ContractedAmount, Currency: li.Currency}}
	}
	if li.FinanceBudgetYear != nil {
		year := int64(*li.FinanceBudgetYear)
		fields["lineItem.financeBudgetYear"] = allwork.Value{Number: &year}
	}
	if li.FinanceBudgetItemID != "" {
		item, found, err := y.s.budgetItems.Item(ctx, li.FinanceBudgetItemID)
		if err != nil {
			return err
		}
		if found {
			fields["lineItem.financeBudgetItem"] = allwork.Value{Text: item.Name, Inactive: !item.Active}
		}
	}
	return nil
}
