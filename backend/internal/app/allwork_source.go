package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"northstar/internal/activities"
	"northstar/internal/allwork"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/rollup"
	"northstar/internal/settings"
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
	plans          *plans.Service
	plansRepo      plans.Repository
	campaigns      *campaigns.Service
	campaignsRepo  campaigns.Repository
	activities     *activities.Service
	activitiesRepo activities.Repository
	lineItems      *lineitems.Service
	lineItemsRepo  lineitems.Repository
	settings       *settings.Service
	budgetItems    lineitems.FinanceBudgets // 002's finance budget items, as line items read them
	workflows      activities.Workflows     // 008's approval workflows, as activities read them
	people         *people.Service
	peopleRepo     people.Repository
	history        *history.Service
}

// The compiler checks here that allWorkSource implements allwork.Source.
var _ allwork.Source = allWorkSource{}

// Year returns every plan of year with its campaigns, initiatives, activities and line items, each with its
// position in the plan tree (allwork data model §2). Records in the recycle bin are not stored, so they are
// never among them. The values that only some columns show — a plan's completeness, the plan goals, the
// activities' goals — are read only when columns names one of those columns, because the owning features
// work them out record by record.
func (s allWorkSource) Year(ctx context.Context, year int, today date.Date, columns []string) (allwork.YearData, error) {
	data := allwork.YearData{Nodes: []allwork.Node{}}
	yearPlans, err := s.plansRepo.PlansOfYear(ctx, year)
	if err != nil || len(yearPlans) == 0 {
		return data, err
	}
	y := yearReader{s: s, owners: map[string]*allwork.Owner{}, columns: columns, plans: map[string]plans.Plan{}}
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
	for _, p := range yearPlans {
		if err := y.addPlan(ctx, p, missing[p.ID]); err != nil {
			return allwork.YearData{}, err
		}
	}
	if err := y.addActivityDetails(ctx); err != nil {
		return allwork.YearData{}, err
	}
	if err := y.addLineItems(ctx); err != nil {
		return allwork.YearData{}, err
	}
	data.Nodes = y.nodes
	return data, nil
}

// planMissing returns the number of each plan's own needed-to-publish fields still empty, from 004's Plans
// list: one call for every plan of the year.
func (s allWorkSource) planMissing(ctx context.Context, year int) (map[string]int, error) {
	list, err := s.plans.List(ctx, actor.System, plans.ListFilter{Scope: plans.ScopeAll, Year: &year})
	if err != nil {
		return nil, err
	}
	missing := make(map[string]int, len(list.Items))
	for _, p := range list.Items {
		missing[p.ID] = len(p.Completeness.Missing)
	}
	return missing, nil
}

// yearReader collects the nodes of one year.
type yearReader struct {
	s             allWorkSource
	columns       []string // the columns the view shows or sorts by
	nodes         []allwork.Node
	owners        map[string]*allwork.Owner // read once each
	attributes    map[settings.Level][]settings.Attribute
	options       map[string]settings.Option // the options of the choice attributes, by id
	initiativeIDs []string
	activities    map[string]int    // activity id → index of its node
	stored        map[string]string // activity id → its stored status (for the SWBS request status)
	rates         money.Rates       // the current rates (016)
	plans         map[string]plans.Plan
}

// planRate returns the current rate of the plan currency of the plan planID (016 FR-014); 1 for USD.
func (y *yearReader) planRate(planID string) money.Rate {
	if rate, err := y.rates.Of(y.plans[planID].Currency); err == nil {
		return rate
	}
	return money.One
}

// usd converts an amount of the plan planID's currency to USD (016 FR-014): once, half up; nil stays nil.
func (y *yearReader) usd(planID string, amount *money.Amount) *money.Amount {
	if amount == nil {
		return nil
	}
	usd := money.Convert(*amount, y.planRate(planID), money.One)
	return &usd
}

// inPlan returns an amount of the plan planID's currency as money; nil stays nil.
func (y *yearReader) inPlan(planID string, amount *money.Amount) *money.Money {
	if amount == nil {
		return nil
	}
	return &money.Money{Amount: *amount, Currency: y.plans[planID].Currency}
}

// phased returns the 12 phased monthly budgets of the plan planID's year, in the plan currency and in USD
// (016 FR-024, FR-038); nil, nil when not phased.
func (y *yearReader) phased(planID string, monthly []money.Amount) ([]money.Amount, []allwork.MonthAmount) {
	if len(monthly) != 12 {
		return nil, nil
	}
	keys := rollup.MonthKeys(y.plans[planID].Year)
	usd := make([]allwork.MonthAmount, 0, 12)
	for i, amount := range money.ConvertParts(monthly, y.planRate(planID), money.One) {
		usd = append(usd, allwork.MonthAmount{Month: keys[i], USD: amount})
	}
	return monthly, usd
}

// planFacts sets the plan fields every node of the plan planID carries (016 data-model §5.1).
func (y *yearReader) planFacts(node *allwork.Node) {
	p := y.plans[node.PlanID]
	node.PlanYear, node.PlanType, node.PlanCurrency = p.Year, string(p.PlanType), p.Currency
}

// needs reports whether the view shows or sorts by one of the columns keys.
func (y *yearReader) needs(keys ...string) bool {
	return slices.ContainsFunc(keys, func(key string) bool { return slices.Contains(y.columns, key) })
}

// addPlan adds the plan p with its own missing fields, its campaigns and initiatives as 005's plan tree
// orders them, and its activities as 006's plan-tree rows order them.
func (y *yearReader) addPlan(ctx context.Context, p plans.Plan, missing int) error {
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
	tree, err := y.s.campaigns.ListPlanCampaigns(ctx, actor.System, p.ID)
	if err != nil {
		return err
	}
	for ci, c := range tree.Items {
		err := y.addContainer(ctx, container{
			level: allwork.LevelCampaign, id: c.ID, parentID: p.ID, planID: p.ID, order: ci, name: c.Name,
			description: c.Description, ownerID: c.OwnerID, start: c.StartDate, end: c.EndDate, budget: c.Budget,
			progress: c.ApprovalProgress, timing: c.Timing, completeness: c.Completeness, meta: c.Meta,
			attributes: c.Attributes, phasing: c.BudgetPhasing,
		})
		if err != nil {
			return err
		}
		for ii, i := range c.Initiatives {
			err := y.addContainer(ctx, container{
				level: allwork.LevelInitiative, id: i.ID, parentID: c.ID, planID: p.ID, order: ii, name: i.Name,
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
	summaries, err := y.s.activities.ListForPlan(ctx, actor.System, p.ID)
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
		months, _ := lineitems.ActivityMonths(lineitems.ActivityFacts{StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing})
		y.activities[a.ID] = len(y.nodes)
		y.nodes = append(y.nodes, allwork.Node{
			Level: allwork.LevelActivity, ID: a.ID, ParentID: a.InitiativeID, PlanID: p.ID, Order: order, Name: a.Name,
			Owner: ownerOf(a.Owner), StartDate: a.StartDate, EndDate: a.EndDate, Ongoing: a.Ongoing, Months: months,
			Status: string(a.DisplayStatus), Missing: a.Completeness.MissingFields, PlannedCost: y.usd(p.ID, planned),
			PlannedCostPlan: y.inPlan(p.ID, planned), ActivityTypeID: a.ActivityType.ID,
			Fields: map[string]allwork.Value{
				"activity.type":    {Text: a.ActivityType.Name, Inactive: !a.ActivityType.Active},
				"activity.ongoing": {Yes: &a.Ongoing},
			},
		})
		y.planFacts(&y.nodes[len(y.nodes)-1])
	}
	return nil
}

// The words of stored values, as the plan and line-item forms show them (004, 007).
var (
	spreadWords = map[string]string{
		"evenly": "Evenly", "firstMonth": "All in first month", "lastMonth": "All in last month", "custom": "Custom",
	}
	goalStageWords = map[plans.GoalStage]string{plans.StageINQ: "INQ", plans.StageMQL: "MQL", plans.StageRevenue: "Revenue"}
)

// planFields returns the plan-level columns of p (allwork data model §6.2): its stored fields with the names
// of its model, teams (by name, as 004 shows them) and default finance budget, and — only when the view shows
// them — the plan goals 004 works out from the model.
func (y *yearReader) planFields(ctx context.Context, p plans.Plan) (map[string]allwork.Value, error) {
	year := int64(p.Year)
	fields := map[string]allwork.Value{
		"plan.year":          {Number: &year},
		"plan.strategic":     {Yes: &p.AOP},
		"plan.description":   {Text: p.Description},
		"plan.defaultSpread": {Text: spreadWords[string(p.DefaultSpread)]},
		"plan.bcpir":         {Yes: &p.BCPIR},
		"plan.goalStage":     {Text: goalStageWords[p.GoalStage]},
	}
	if m, found, err := y.s.settings.Model(ctx, p.ModelID); err != nil {
		return nil, err
	} else if found {
		fields["plan.model"] = allwork.Value{Text: m.Name, Inactive: !m.Active}
	}
	refs, err := y.s.people.TeamRefs(ctx, p.TeamIDs)
	if err != nil {
		return nil, err
	}
	teams := []allwork.Choice{}
	for _, id := range p.TeamIDs {
		if t, found := refs[id]; found {
			teams = append(teams, allwork.Choice{Label: t.Name, Inactive: !t.Active})
		}
	}
	slices.SortFunc(teams, func(x, y allwork.Choice) int {
		return strings.Compare(strings.ToLower(x.Label), strings.ToLower(y.Label))
	})
	fields["plan.teams"] = allwork.Value{Choices: teams}
	if p.DefaultFinanceBudgetID != "" {
		if b, found, err := y.s.settings.FinanceBudget(ctx, p.DefaultFinanceBudgetID); err != nil {
			return nil, err
		} else if found {
			fields["plan.defaultFinanceBudget"] = allwork.Value{Text: b.Name}
		}
	}
	if y.needs("plan.goalMql", "plan.goalRevenue") {
		view, err := y.s.plans.Get(ctx, actor.System, p.ID) // actor.System sees amounts in USD
		if err != nil {
			return nil, err
		}
		fields["plan.goalMql"] = allwork.Value{Number: view.Goals.Plan.MQL}
		if view.Goals.Plan.Revenue != nil {
			fields["plan.goalRevenue"] = allwork.Value{USD: &view.Goals.Plan.Revenue.Amount}
		}
	}
	if p.GoalAmount != "" {
		if p.GoalStage == plans.StageRevenue {
			if amount, err := money.ParseAmount(p.GoalAmount); err == nil {
				fields["plan.goalAmount"] = allwork.Value{USD: &amount}
			}
		} else if n, err := strconv.ParseInt(p.GoalAmount, 10, 64); err == nil {
			fields["plan.goalAmount"] = allwork.Value{Number: &n}
		}
	}
	y.addAttributes(fields, settings.LevelPlan, p.Attributes)
	return fields, nil
}

// loadAttributes reads every Shell attribute of every level and the options of the choice attributes once.
func (y *yearReader) loadAttributes(ctx context.Context) error {
	y.attributes, y.options = map[settings.Level][]settings.Attribute{}, map[string]settings.Option{}
	for _, level := range settings.Levels {
		list, err := y.s.settings.Attributes(ctx, level)
		if err != nil {
			return err
		}
		y.attributes[level] = list
		for _, a := range list {
			if !a.IsChoice() {
				continue
			}
			options, err := y.s.settings.Options(ctx, a.ID)
			if err != nil {
				return err
			}
			for _, o := range options {
				y.options[o.ID] = o
			}
		}
	}
	return nil
}

// addAttributes adds to fields the column "attr.<id>" of every Shell attribute of level that values has (002
// data model §9): a choice by its options' labels, a % split with the shares, Yes or No (an empty yes/no reads
// as No), a text or a whole number.
func (y *yearReader) addAttributes(fields map[string]allwork.Value, level settings.Level, values settings.AttributeValues) {
	for _, a := range y.attributes[level] {
		key := "attr." + a.ID
		raw := values[a.ID]
		if emptyValue(raw) {
			if a.Type == settings.TypeYesNo {
				no := false
				fields[key] = allwork.Value{Yes: &no}
			}
			continue
		}
		var v allwork.Value
		switch a.Type {
		case settings.TypeSingleChoice:
			var id string
			if json.Unmarshal(raw, &id) == nil {
				v.Choices = []allwork.Choice{y.choice(id)}
			}
		case settings.TypeMultipleChoice:
			var ids []string
			if json.Unmarshal(raw, &ids) == nil {
				for _, id := range ids {
					v.Choices = append(v.Choices, y.choice(id))
				}
			}
		case settings.TypeMultipleChoiceSplit:
			var shares []settings.SplitShare
			if json.Unmarshal(raw, &shares) == nil {
				for _, s := range shares {
					c := y.choice(s.OptionID)
					c.Share = &s.Share
					v.Choices = append(v.Choices, c)
				}
			}
		case settings.TypeYesNo:
			var yes bool
			if json.Unmarshal(raw, &yes) == nil {
				v.Yes = &yes
			}
		case settings.TypeNumber:
			var n int64
			if json.Unmarshal(raw, &n) == nil {
				v.Number = &n
			}
		default:
			_ = json.Unmarshal(raw, &v.Text)
		}
		fields[key] = v
	}
}

// attrValues returns the values of the choice and yes/no Shell attributes of level that values has, by
// attribute id, for the attribute filters and the Breakdown of 016 (FR-039 – FR-041): the options in their
// order (with the shares of a % split), or the yes/no answer — an empty yes/no reads as No.
func (y *yearReader) attrValues(level settings.Level, values settings.AttributeValues) map[string]allwork.AttrValue {
	attrs := map[string]allwork.AttrValue{}
	for _, a := range y.attributes[level] {
		raw := values[a.ID]
		var v allwork.AttrValue
		switch a.Type {
		case settings.TypeSingleChoice:
			var id string
			if !emptyValue(raw) && json.Unmarshal(raw, &id) == nil && id != "" {
				v.OptionIDs = []string{id}
			}
		case settings.TypeMultipleChoice:
			if !emptyValue(raw) {
				_ = json.Unmarshal(raw, &v.OptionIDs)
			}
		case settings.TypeMultipleChoiceSplit:
			var shares []settings.SplitShare
			if !emptyValue(raw) && json.Unmarshal(raw, &shares) == nil {
				v.Shares = map[string]money.Percent{}
				for _, share := range shares {
					v.OptionIDs = append(v.OptionIDs, share.OptionID)
					v.Shares[share.OptionID] = share.Share
				}
			}
		case settings.TypeYesNo:
			yes := false
			if !emptyValue(raw) {
				_ = json.Unmarshal(raw, &yes)
			}
			v.Yes = &yes
		default:
			continue
		}
		if len(v.OptionIDs) > 0 || v.Yes != nil {
			attrs[a.ID] = v
		}
	}
	return attrs
}

// choice returns the option id as a choice; an option that no longer resolves shows its id.
func (y *yearReader) choice(id string) allwork.Choice {
	o, found := y.options[id]
	if !found {
		return allwork.Choice{Label: id}
	}
	return allwork.Choice{Label: o.Label, Inactive: !o.Active}
}

// emptyValue reports whether a stored attribute value is no value: missing, null, "", [] or {} (002 data
// model §9).
func emptyValue(raw json.RawMessage) bool {
	switch string(bytes.TrimSpace(raw)) {
	case "", "null", `""`, "[]", "{}":
		return true
	}
	return false
}

// container is a campaign or initiative as 005's plan tree shows it.
type container struct {
	level                allwork.Level
	id, parentID, planID string
	order                int
	name, description    string
	ownerID              *string
	start, end           *date.Date
	budget               *money.Money
	progress             campaigns.ApprovalSummary
	timing               *string
	completeness         campaigns.RecordCompleteness
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
		n := &y.nodes[i]
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
	}
	if !y.needs("inq", "mql", "revenue") {
		return nil
	}
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
	activityIDs := make([]string, 0, len(y.activities))
	for id := range y.activities {
		activityIDs = append(activityIDs, id)
	}
	items, err := y.s.lineItemsRepo.ListByActivities(ctx, activityIDs) // ordered by id: creation order
	if err != nil || len(items) == 0 {
		return err
	}
	summaries, err := y.s.lineItems.Summaries(ctx, activityIDs)
	if err != nil {
		return err
	}
	missing := map[string]int{}
	for _, summary := range summaries {
		for _, incomplete := range summary.Incomplete {
			missing[incomplete.LineItemID] = len(incomplete.Missing)
		}
	}
	rates, err := y.s.settings.Rates(ctx)
	if err != nil {
		return err
	}
	for order, li := range items {
		activity := y.nodes[y.activities[li.ActivityID]]
		facts := lineitems.ActivityFacts{
			ID: activity.ID, StartDate: activity.StartDate, EndDate: activity.EndDate, Ongoing: activity.Ongoing,
			Status: y.stored[li.ActivityID],
		}
		node, err := y.lineItemNode(ctx, li, facts, rates, activity.PlanID)
		if err != nil {
			return err
		}
		node.Order, node.Missing = order, missing[li.ID]
		y.nodes = append(y.nodes, node)
	}
	return nil
}

// lineItemNode returns the node of the line item li of the activity f.
func (y *yearReader) lineItemNode(ctx context.Context, li lineitems.LineItem, f lineitems.ActivityFacts, rates money.Rates,
	planID string) (allwork.Node, error) {
	rate, known := rates[li.Currency]
	if !known { // Settings never deletes a currency in use (domain model §6)
		return allwork.Node{}, fmt.Errorf("app: line item %s has the unknown currency %q", li.ID, li.Currency)
	}
	owner, err := y.owner(ctx, li.OwnerID)
	if err != nil {
		return allwork.Node{}, err
	}
	plannedUSD := lineitems.PlannedCostUSD(li, rate)
	planRate := y.planRate(planID)
	plannedPlan := money.Convert(li.PlannedCost, rate, planRate) // 016 FR-015
	node := allwork.Node{
		Level: allwork.LevelLineItem, ID: li.ID, ParentID: li.ActivityID, PlanID: planID, Name: li.Name,
		Description: li.Description, Owner: owner, PlannedCost: &plannedUSD,
		LocalPlannedCost: &money.Money{Amount: li.PlannedCost, Currency: li.Currency},
		Swbs:             string(lineitems.SwbsStatusOf(li, f)), CreatedAt: li.CreatedAt, UpdatedAt: li.UpdatedAt,
		MonthsUSD: []allwork.MonthAmount{}, MonthsPlan: []allwork.PlanMonth{}, PlannedCostPlan: y.inPlan(planID, &plannedPlan),
		Fields: map[string]allwork.Value{
			"lineItem.description":      {Text: li.Description},
			"lineItem.localPlannedCost": {Money: &money.Money{Amount: li.PlannedCost, Currency: li.Currency}},
			"lineItem.currency":         {Text: li.Currency},
			"lineItem.spread":           {Text: spreadWords[string(li.Spread.Method)]},
			"lineItem.requestSwbs":      {Yes: &li.RequestSwbs},
		},
	}
	if li.ContractedAmount != nil {
		node.Fields["lineItem.contractedAmount"] = allwork.Value{Money: &money.Money{Amount: *li.ContractedAmount, Currency: li.Currency}}
	}
	if li.FinanceBudgetYear != nil {
		year := int64(*li.FinanceBudgetYear)
		node.Fields["lineItem.financeBudgetYear"] = allwork.Value{Number: &year}
	}
	if li.FinanceBudgetItemID != "" {
		item, found, err := y.s.budgetItems.Item(ctx, li.FinanceBudgetItemID)
		if err != nil {
			return allwork.Node{}, err
		}
		if found {
			node.Fields["lineItem.financeBudgetItem"] = allwork.Value{Text: item.Name, Inactive: !item.Active}
		}
	}
	y.addAttributes(node.Fields, settings.LevelLineItem, li.Attributes)
	node.Attrs = y.attrValues(settings.LevelLineItem, li.Attributes)
	y.planFacts(&node)
	months, spread := lineitems.SpreadMonths(li, f)
	if !spread {
		node.UnspreadUSD, node.UnspreadPlan = plannedUSD, plannedPlan
		return node, nil
	}
	for i, amount := range lineitems.USDMonths(months, rate) {
		node.MonthsUSD = append(node.MonthsUSD, allwork.MonthAmount{Month: months[i].Month, USD: amount})
	}
	for i, amount := range lineitems.ConvertMonths(months, rate, planRate) {
		node.MonthsPlan = append(node.MonthsPlan, allwork.PlanMonth{Month: months[i].Month, Amount: amount})
	}
	return node, nil
}

// owner returns the person id as an owner, nil for "". A person who is not found is shown by their ID.
func (y *yearReader) owner(ctx context.Context, id string) (*allwork.Owner, error) {
	if id == "" {
		return nil, nil
	}
	if owner, found := y.owners[id]; found {
		return owner, nil
	}
	owner := &allwork.Owner{ID: id, Name: id}
	p, found, err := y.s.peopleRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if found {
		owner = &allwork.Owner{ID: p.ID, Name: p.Name, Inactive: !p.Active}
	}
	y.owners[id] = owner
	return owner, nil
}

// ownerOf returns a person as an owner, nil for none.
func ownerOf(ref *people.PersonRef) *allwork.Owner {
	if ref == nil {
		return nil
	}
	return &allwork.Owner{ID: ref.ID, Name: ref.Name, Inactive: !ref.Active}
}

// Years returns the years that have plans (004).
func (s allWorkSource) Years(ctx context.Context) ([]int, error) {
	all, err := s.plansRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	years := []int{}
	for _, p := range all {
		if !slices.Contains(years, p.Year) {
			years = append(years, p.Year)
		}
	}
	return years, nil
}

// Choices returns every team with the person's own marked (003), the options of the plan-level Business
// unit attribute and every activity type (002), and every person (003), active and inactive.
func (s allWorkSource) Choices(ctx context.Context, personID string) (allwork.Choices, error) {
	c := allwork.Choices{Teams: []allwork.Option{}, BusinessUnits: []allwork.Option{}, ActivityTypes: []allwork.Option{},
		People: []allwork.Option{}}
	mine, err := s.people.TeamIDsOf(ctx, personID)
	if err != nil {
		return allwork.Choices{}, err
	}
	teams, err := s.peopleRepo.ListTeams(ctx)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, t := range teams {
		c.Teams = append(c.Teams, allwork.Option{ID: t.ID, Label: t.Name, Inactive: !t.Active, Mine: slices.Contains(mine, t.ID)})
	}
	attributes, err := s.settings.Attributes(ctx, settings.LevelPlan)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, a := range attributes {
		if a.Role != settings.RoleBusinessUnit {
			continue
		}
		options, err := s.settings.Options(ctx, a.ID)
		if err != nil {
			return allwork.Choices{}, err
		}
		for _, o := range options {
			c.BusinessUnits = append(c.BusinessUnits, allwork.Option{ID: o.ID, Label: o.Label, Inactive: !o.Active})
		}
	}
	types, err := s.settings.ActivityTypes(ctx)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, t := range types {
		c.ActivityTypes = append(c.ActivityTypes, allwork.Option{ID: t.ID, Label: t.Name, Inactive: !t.Active})
	}
	everyone, err := s.peopleRepo.List(ctx)
	if err != nil {
		return allwork.Choices{}, err
	}
	for _, p := range everyone {
		c.People = append(c.People, allwork.Option{ID: p.ID, Label: p.Name, Inactive: !p.Active})
	}
	return c, nil
}

// Attributes returns every Shell attribute of Settings (002), active and inactive, level by level top-down
// and in display order within a level.
func (s allWorkSource) Attributes(ctx context.Context) ([]allwork.Attribute, error) {
	list := []allwork.Attribute{}
	for _, level := range settings.Levels {
		attributes, err := s.settings.Attributes(ctx, level)
		if err != nil {
			return nil, err
		}
		for _, a := range attributes {
			attribute := allwork.Attribute{
				ID: a.ID, Label: a.Label, Level: allwork.Level(level), Number: a.Type == settings.TypeNumber, Active: a.Active,
				Type: string(a.Type), Condition: slices.Clone(a.Condition), Options: []allwork.Option{},
			}
			if a.IsChoice() { // 016: the options of the attribute filters and the Breakdown, in Settings' order
				options, err := s.settings.Options(ctx, a.ID)
				if err != nil {
					return nil, err
				}
				for _, o := range options {
					attribute.Options = append(attribute.Options, allwork.Option{ID: o.ID, Label: o.Label, Inactive: !o.Active})
				}
			}
			list = append(list, attribute)
		}
	}
	return list, nil
}

// RatesUsed states the current rate of every currency other than USD, with its date and source (016 FR-009):
// "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import".
func (s allWorkSource) RatesUsed(ctx context.Context) ([]string, error) {
	infos, err := s.settings.CurrentRates(ctx)
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, info := range infos {
		if info.Currency == money.BaseCurrency {
			continue
		}
		line := fmt.Sprintf("1 USD = %s %s", info.Rate.String(), info.Currency)
		if info.RateDate != nil {
			line += " · rate of " + date.Format(*info.RateDate)
		}
		if info.Source != nil {
			line += " · " + rateSourceWords[*info.Source]
		}
		lines = append(lines, line)
	}
	return lines, nil
}

// rateSourceWords are the sources of a rate as people read them (016 FR-001).
var rateSourceWords = map[settings.RateSource]string{settings.RateManual: "Entered by hand", settings.RateGSAPImport: "GSAP import"}

// Currency returns the person's display currency and its current rate (003's DisplayRateOf); an unusable
// rate shows US dollars.
func (s allWorkSource) Currency(ctx context.Context, personID string) (allwork.Currency, error) {
	d, err := s.people.DisplayRateOf(ctx, personID)
	if err != nil {
		return allwork.Currency{}, err
	}
	rate, err := money.ParseRate(d.Rate)
	if err != nil || d.Code == "" {
		return allwork.Currency{Code: money.BaseCurrency, Rate: money.One}, nil
	}
	return allwork.Currency{Code: d.Code, Rate: rate}, nil
}

// Locate returns the stored record recordID with its plan (and, for a line item, its activity), and false
// when it, or a record above it, is not stored.
func (s allWorkSource) Locate(ctx context.Context, recordID string) (allwork.Location, bool, error) {
	prefix, _, _ := strings.Cut(recordID, "-")
	switch prefix {
	case ids.PrefixPlan:
		p, found, err := s.plansRepo.Get(ctx, recordID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		return allwork.Location{Level: allwork.LevelPlan, Name: p.Name, PlanID: p.ID, Year: p.Year}, true, nil
	case ids.PrefixCampaign:
		c, found, err := s.campaignsRepo.GetCampaign(ctx, recordID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		return s.withPlan(ctx, allwork.Location{Level: allwork.LevelCampaign, Name: c.Name}, c.PlanID)
	case ids.PrefixInitiative:
		i, found, err := s.campaignsRepo.GetInitiative(ctx, recordID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		return s.underCampaign(ctx, allwork.Location{Level: allwork.LevelInitiative, Name: i.Name}, i.CampaignID)
	case ids.PrefixActivity:
		a, found, err := s.activitiesRepo.Get(ctx, recordID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		return s.underInitiative(ctx, allwork.Location{Level: allwork.LevelActivity, Name: a.Name}, a.InitiativeID)
	case ids.PrefixLineItem:
		li, found, err := s.lineItemsRepo.Get(ctx, recordID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		a, found, err := s.activitiesRepo.Get(ctx, li.ActivityID)
		if err != nil || !found {
			return allwork.Location{}, false, err
		}
		location := allwork.Location{Level: allwork.LevelLineItem, Name: li.Name, ActivityID: a.ID}
		return s.underInitiative(ctx, location, a.InitiativeID)
	}
	return allwork.Location{}, false, nil
}

// underInitiative completes location with the plan of the initiative initiativeID.
func (s allWorkSource) underInitiative(ctx context.Context, location allwork.Location, initiativeID string) (allwork.Location, bool, error) {
	i, found, err := s.campaignsRepo.GetInitiative(ctx, initiativeID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.underCampaign(ctx, location, i.CampaignID)
}

// underCampaign completes location with the plan of the campaign campaignID.
func (s allWorkSource) underCampaign(ctx context.Context, location allwork.Location, campaignID string) (allwork.Location, bool, error) {
	c, found, err := s.campaignsRepo.GetCampaign(ctx, campaignID)
	if err != nil || !found {
		return allwork.Location{}, false, err
	}
	return s.withPlan(ctx, location, c.PlanID)
}

// withPlan completes location with the plan planID, when it is stored.
func (s allWorkSource) withPlan(ctx context.Context, location allwork.Location, planID string) (allwork.Location, bool, error) {
	if _, found, err := s.plansRepo.Get(ctx, planID); err != nil || !found {
		return allwork.Location{}, false, err
	}
	location.PlanID = planID
	return location, true, nil
}

// InRecycleBin reports whether a recycle-bin entry not yet undone or restored holds the record recordID:
// deleted itself or with a record above it.
func (s allWorkSource) InRecycleBin(ctx context.Context, recordID string) (bool, error) {
	entries, err := s.history.ListBin(ctx, actor.System) // entries undone or restored are left out
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		for _, snapshot := range entry.Snapshots {
			if snapshot.ID == recordID {
				return true, nil
			}
		}
	}
	return false, nil
}
