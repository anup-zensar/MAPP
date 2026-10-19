package campaigns

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"northstar/internal/people"
	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// The labels of a campaign's and an initiative's own needed-to-publish fields (data model §4.4). The dates
// count as one field, "Dates": they are saved both or neither (D-092).
const (
	ownerLabel = "Owner"
	datesLabel = "Dates"
)

// planCampaigns builds the campaign tree of plan as the API returns it: every campaign with its
// initiatives, every derived value (data model §4) and the totals row. It reads the plan's campaigns, their
// initiatives and the activities below once, and works out every roll-up in one pass with package rollup
// (D-090), so nothing derived is ever stored (FR-012, FR-015). It runs inside the caller's transaction.
func (s *Service) planCampaigns(ctx context.Context, plan PlanFacts) (PlanCampaigns, error) {
	campaigns, err := s.repo.CampaignsOfPlan(ctx, plan.ID)
	if err != nil {
		return PlanCampaigns{}, err
	}
	campaignIDs := make([]string, 0, len(campaigns))
	for _, c := range campaigns {
		campaignIDs = append(campaignIDs, c.ID)
	}
	initiatives, err := s.repo.InitiativesOfCampaigns(ctx, campaignIDs)
	if err != nil {
		return PlanCampaigns{}, err
	}
	initiativeIDs := make([]string, 0, len(initiatives))
	initiativesOf := map[string][]Initiative{}
	for _, i := range initiatives {
		initiativeIDs = append(initiativeIDs, i.ID)
		initiativesOf[i.CampaignID] = append(initiativesOf[i.CampaignID], i)
	}
	summaries, err := s.deps.Activities.Summaries(ctx, initiativeIDs)
	if err != nil {
		return PlanCampaigns{}, err
	}
	activitiesOf := map[string][]ActivitySummary{}
	for _, a := range summaries {
		activitiesOf[a.InitiativeID] = append(activitiesOf[a.InitiativeID], a)
	}
	businessUnit, err := s.deps.Settings.BusinessUnit(ctx, plan.Attributes)
	if err != nil {
		return PlanCampaigns{}, err
	}
	b := &viewBuilder{
		ctx: ctx, s: s, plan: plan, businessUnit: businessUnit,
		today: date.Today(s.now()), owners: map[string]*people.PersonRef{},
	}

	campaignBudgets := make([]*money.Money, 0, len(campaigns))
	for _, c := range campaigns {
		campaignBudgets = append(campaignBudgets, c.Budget)
	}
	allocated := rollup.Total(campaignBudgets)
	planRemaining := rollup.Remaining(plan.Budget, allocated)

	views := make([]CampaignView, 0, len(campaigns))
	plannedCosts := make([]*money.Money, 0, len(campaigns))
	activities := 0
	for _, c := range campaigns {
		view, err := b.campaign(c, initiativesOf[c.ID], activitiesOf, planRemaining)
		if err != nil {
			return PlanCampaigns{}, err
		}
		views = append(views, view)
		plannedCosts = append(plannedCosts, view.BudgetAndCost.PlannedCost)
		activities += view.Counts.Activities
	}
	sortInTree(views, func(c CampaignView) (*date.Date, string, string) { return c.StartDate, c.Name, c.ID })

	planOwner, err := b.owner(plan.OwnerID)
	if err != nil {
		return PlanCampaigns{}, err
	}
	return PlanCampaigns{
		Plan:      PlanRef{ID: plan.ID, Name: plan.Name, Year: plan.Year},
		PlanOwner: planOwner,
		Items:     views,
		Total:     len(views),
		Totals: PlanCampaignTotals{
			Campaigns:            len(views),
			Initiatives:          len(initiatives),
			Activities:           activities,
			AllocatedToCampaigns: allocated,
			PlanBudget:           plan.Budget,
			RemainingPlanBudget:  planRemaining,
			OverAllocatedBy:      rollup.OverBy(planRemaining),
			PlannedCost:          rollup.Sum(plannedCosts),
		},
	}, nil
}

// viewBuilder holds what every view of one plan's tree shares, read once.
type viewBuilder struct {
	ctx          context.Context
	s            *Service
	plan         PlanFacts
	businessUnit []string                     // the plan's business unit: which attributes are shown (R9)
	today        date.Date                    // for Timing (D-091)
	owners       map[string]*people.PersonRef // people already read, by id
}

// campaign builds the view of campaign c with its initiatives; planRemaining is the plan's remaining budget.
func (b *viewBuilder) campaign(c Campaign, initiatives []Initiative, activitiesOf map[string][]ActivitySummary,
	planRemaining *money.Money) (CampaignView, error) {
	budgets := make([]*money.Money, 0, len(initiatives))
	for _, i := range initiatives {
		budgets = append(budgets, i.Budget)
	}
	remaining := rollup.Remaining(c.Budget, rollup.Total(budgets)) // what the campaign has left for initiatives
	ref := CampaignRef{ID: c.ID, Name: c.Name, StartDate: c.StartDate, EndDate: c.EndDate}

	views := make([]InitiativeView, 0, len(initiatives))
	plannedCosts := make([]*money.Money, 0, len(initiatives))
	goals := make([]rollup.Goals, 0, len(initiatives))
	counts := CampaignCounts{Initiatives: len(initiatives)}
	itemsBelow := 0
	for _, i := range initiatives {
		view, err := b.initiative(i, ref, activitiesOf[i.ID], remaining)
		if err != nil {
			return CampaignView{}, err
		}
		views = append(views, view)
		plannedCosts = append(plannedCosts, view.BudgetAndCost.PlannedCost)
		goals = append(goals, rollup.Goals{INQ: view.Goals.INQ, MQL: view.Goals.MQL, Revenue: view.Goals.Revenue})
		counts.Activities += view.Counts.Activities
		counts.ApprovedActivities += view.Counts.ApprovedActivities
		counts.LineItems += view.Counts.LineItems
		itemsBelow += view.Completeness.ItemsBelowNeedingAttention
		if len(view.Completeness.Missing) > 0 {
			itemsBelow++ // the initiative itself needs attention
		}
	}
	sortInTree(views, func(i InitiativeView) (*date.Date, string, string) { return i.StartDate, i.Name, i.ID })

	owner, err := b.owner(c.OwnerID)
	if err != nil {
		return CampaignView{}, err
	}
	missing, err := b.missing(settings.LevelCampaign, c.OwnerID, c.StartDate, c.Attributes)
	if err != nil {
		return CampaignView{}, err
	}
	sum := rollup.AddGoals(goals)
	return CampaignView{
		ID:                    c.ID,
		PlanID:                c.PlanID,
		Name:                  c.Name,
		Description:           c.Description,
		OwnerID:               c.OwnerID,
		Owner:                 owner,
		StartDate:             c.StartDate,
		EndDate:               c.EndDate,
		Budget:                c.Budget,
		Attributes:            orEmpty(c.Attributes),
		Plan:                  PlanRef{ID: b.plan.ID, Name: b.plan.Name, Year: b.plan.Year},
		OutsidePlanYear:       outsideYear(c.StartDate, c.EndDate, b.plan.Year),
		ParentRemainingBudget: planRemaining,
		BudgetAndCost:         budgetAndCost(rollup.Container(c.Budget, budgets, rollup.Sum(plannedCosts))),
		ApprovalProgress:      approvalSummary(counts.ApprovedActivities, counts.Activities),
		Timing:                timing(c.StartDate, c.EndDate, b.today),
		Completeness:          RecordCompleteness{Missing: missing, ItemsBelowNeedingAttention: itemsBelow},
		Goals:                 GoalSums{INQ: sum.INQ, MQL: sum.MQL, Revenue: sum.Revenue},
		Counts:                counts,
		Initiatives:           views,
		Meta:                  c.Meta,
	}, nil
}

// initiative builds the view of initiative i of campaign with its activities; campaignRemaining is the
// campaign's remaining budget.
func (b *viewBuilder) initiative(i Initiative, campaign CampaignRef, activities []ActivitySummary,
	campaignRemaining *money.Money) (InitiativeView, error) {
	plannedCosts := make([]*money.Money, 0, len(activities))
	goals := make([]rollup.Goals, 0, len(activities))
	counts := InitiativeCounts{Activities: len(activities)}
	itemsBelow := 0
	for _, a := range activities {
		plannedCosts = append(plannedCosts, a.PlannedCost)
		goals = append(goals, rollup.Goals{INQ: a.INQ, MQL: a.MQL, Revenue: a.Revenue})
		if a.Approved {
			counts.ApprovedActivities++
		}
		counts.LineItems += a.LineItems
		if a.Incomplete {
			itemsBelow++
		}
		itemsBelow += len(a.IncompleteLineItems)
	}

	owner, err := b.owner(i.OwnerID)
	if err != nil {
		return InitiativeView{}, err
	}
	missing, err := b.missing(settings.LevelInitiative, i.OwnerID, i.StartDate, i.Attributes)
	if err != nil {
		return InitiativeView{}, err
	}
	sum := rollup.AddGoals(goals)
	return InitiativeView{
		ID:                    i.ID,
		PlanID:                b.plan.ID,
		CampaignID:            i.CampaignID,
		Name:                  i.Name,
		Description:           i.Description,
		OwnerID:               i.OwnerID,
		Owner:                 owner,
		StartDate:             i.StartDate,
		EndDate:               i.EndDate,
		Budget:                i.Budget,
		Attributes:            orEmpty(i.Attributes),
		Plan:                  PlanRef{ID: b.plan.ID, Name: b.plan.Name, Year: b.plan.Year},
		Campaign:              campaign,
		ParentRemainingBudget: campaignRemaining,
		BudgetAndCost:         budgetAndCost(rollup.Container(i.Budget, nil, rollup.Sum(plannedCosts))),
		ApprovalProgress:      approvalSummary(counts.ApprovedActivities, counts.Activities),
		Timing:                timing(i.StartDate, i.EndDate, b.today),
		Completeness:          RecordCompleteness{Missing: missing, ItemsBelowNeedingAttention: itemsBelow},
		Goals:                 GoalSums{INQ: sum.INQ, MQL: sum.MQL, Revenue: sum.Revenue},
		Counts:                counts,
		Meta:                  i.Meta,
	}, nil
}

// owner returns the person id as records show them, or nil without an owner. An inactive owner is kept
// and flagged (the web app adds "(inactive)").
func (b *viewBuilder) owner(id *string) (*people.PersonRef, error) {
	if id == nil {
		return nil, nil
	}
	if ref, known := b.owners[*id]; known {
		return ref, nil
	}
	person, found, err := b.s.deps.People.Get(b.ctx, *id)
	if err != nil {
		return nil, err
	}
	var ref *people.PersonRef
	if found {
		r := person.Ref()
		ref = &r
	}
	b.owners[*id] = ref
	return ref, nil
}

// missing lists a record's own empty needed-to-publish fields in form order (FR-014, data model §4.4): the
// owner, the dates as one field, then the Shell attributes Settings says are missing.
func (b *viewBuilder) missing(level settings.Level, ownerID *string, start *date.Date,
	values settings.AttributeValues) ([]settings.MissingField, error) {
	missing := []settings.MissingField{}
	if ownerID == nil {
		missing = append(missing, settings.MissingField{Field: "ownerId", Label: ownerLabel})
	}
	if start == nil {
		missing = append(missing, settings.MissingField{Field: "startDate", Label: datesLabel})
	}
	attributes, err := b.s.deps.Settings.MissingAttributes(b.ctx, level, orEmpty(values), b.businessUnit)
	if err != nil {
		return nil, err
	}
	return append(missing, attributes...), nil
}

// budgetAndCost returns the roll-up of package rollup as the API returns it.
func budgetAndCost(c rollup.BudgetAndCost) BudgetAndCost {
	return BudgetAndCost{
		AllocatedToChildren:  c.AllocatedToChildren,
		RemainingForChildren: c.RemainingForChildren,
		OverAllocatedBy:      c.OverAllocatedBy,
		PlannedCost:          c.PlannedCost,
		BudgetUsedPercent:    c.BudgetUsedPercent,
		OverBudget:           c.OverBudget,
		LeftToPlan:           c.LeftToPlan,
	}
}

// approvalSummary is the approval progress of approved out of total activities (FR-013).
func approvalSummary(approved, total int) ApprovalSummary {
	p := rollup.ApprovalProgress(approved, total)
	return ApprovalSummary{State: p.State, Approved: p.Approved, Total: p.Total}
}

// timing is a record's timing against today; nil without dates (FR-013).
func timing(start, end *date.Date, today date.Date) *string {
	t := rollup.Timing(start, end, today)
	if t == "" {
		return nil
	}
	return &t
}

// outsideYear reports R4: the dates start before 1 January or end after 31 December of year (FR-008).
func outsideYear(start, end *date.Date, year int) bool {
	return (start != nil && start.Before(date.YearStart(year))) || (end != nil && end.After(date.YearEnd(year)))
}

// sortInTree sorts records in the order of the plan tree (FR-020); key gives a record's start date, name and
// id.
func sortInTree[T any](records []T, key func(T) (*date.Date, string, string)) {
	slices.SortFunc(records, func(x, y T) int {
		xStart, xName, xID := key(x)
		yStart, yName, yID := key(y)
		return compareInTree(xStart, xName, xID, yStart, yName, yID)
	})
}

// compareInTree orders the rows of the plan tree (FR-020): by start date, then name ignoring letter case,
// records without dates last, then by id so that the order never depends on how records were read.
func compareInTree(aStart *date.Date, aName, aID string, bStart *date.Date, bName, bID string) int {
	switch {
	case aStart == nil && bStart != nil:
		return 1
	case aStart != nil && bStart == nil:
		return -1
	case aStart != nil && bStart != nil:
		if c := aStart.Compare(*bStart); c != 0 {
			return c
		}
	}
	if c := strings.Compare(strings.ToLower(aName), strings.ToLower(bName)); c != 0 {
		return c
	}
	return cmp.Compare(aID, bID)
}

// orEmpty returns values, or an empty set for nil: attribute values are never null in the API.
func orEmpty(values settings.AttributeValues) settings.AttributeValues {
	if values == nil {
		return settings.AttributeValues{}
	}
	return values
}
