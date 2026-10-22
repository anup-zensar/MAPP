package home

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
)

// Service calculates the Home dashboard. It only reads: every operation takes the actor, reads its
// Sources inside one read transaction, so all of a card's figures come from the same moment, and changes
// nothing.
type Service struct {
	tx  store.Transactor
	src Sources
}

// NewService returns the Home service. It reads through src, inside read transactions of tx.
func NewService(tx store.Transactor, src Sources) *Service {
	return &Service{tx: tx, src: src}
}

// converter turns US dollars into the person's display currency: usd × rate, half up to the cent. Totals
// are added in US dollars and converted once (FR-017, D-010).
type converter struct {
	code string
	rate money.Rate
}

// converter returns the converter of a's display currency; one without a usable rate shows US dollars, as
// the plan page does.
func (s *Service) converter(ctx context.Context, a actor.Actor) (converter, error) {
	c, err := s.src.DisplayCurrency(ctx, a)
	if err != nil {
		return converter{}, err
	}
	if c.Code == "" || c.Rate.Micros() <= 0 {
		return converter{code: money.BaseCurrency, rate: money.One}, nil
	}
	return converter{code: c.Code, rate: c.Rate}, nil
}

// of converts usd.
func (c converter) of(usd money.Amount) money.Money {
	return money.Money{Amount: money.FromUSD(usd, c.rate), Currency: c.code}
}

// optional converts usd, or gives nil when there is no amount.
func (c converter) optional(usd *money.Amount) *money.Money {
	if usd == nil {
		return nil
	}
	converted := c.of(*usd)
	return &converted
}

// planScope is every plan and the person's teams: what the cards that follow the My teams / Everything
// switch start from.
type planScope struct {
	all     []Plan
	myTeams map[string]bool // the ids of the person's teams
}

// planScope reads every plan and a's teams.
func (s *Service) planScope(ctx context.Context, a actor.Actor) (planScope, error) {
	teams, err := s.src.MyTeams(ctx, a)
	if err != nil {
		return planScope{}, err
	}
	all, err := s.src.Plans(ctx, a)
	if err != nil {
		return planScope{}, err
	}
	ps := planScope{all: all, myTeams: map[string]bool{}}
	for _, t := range teams {
		ps.myTeams[t.ID] = true
	}
	return ps, nil
}

// in returns the plans of scope (FR-003, FR-004): every plan for Everything; for My teams the plans with at
// least one of the person's teams — each plan once, however many of its teams are the person's.
func (ps planScope) in(scope Scope) []Plan {
	if scope == ScopeEverything {
		return slices.Clone(ps.all)
	}
	return slices.DeleteFunc(slices.Clone(ps.all), func(p Plan) bool { return !ps.isTeamPlan(p) })
}

// isTeamPlan reports whether one of p's teams is one of the person's.
func (ps planScope) isTeamPlan(p Plan) bool {
	return slices.ContainsFunc(p.Teams, func(t Team) bool { return ps.myTeams[t.ID] })
}

// The plan and activity statuses Home counts (domain model §2.1, §2.3).
const (
	planPublished     = "published"
	activityDeclined  = "declined"
	activitySubmitted = "submitted"
)

// Summary returns the four summary cards for the scope (FR-005, FR-006, FR-017): the plans in scope by
// status; the activities a owns, in every plan, with how many are Declined and Submitted; how many wait for
// a's approval and since when the oldest waits; and the planned cost of the plans in scope against their
// plan budgets, added up in US dollars and converted once.
func (s *Service) Summary(ctx context.Context, a actor.Actor, scopeText string) (SummaryView, error) {
	scope, err := ParseScope(scopeText)
	if err != nil {
		return SummaryView{}, err
	}
	var view SummaryView
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		ps, err := s.planScope(ctx, a)
		if err != nil {
			return err
		}
		owned, err := s.src.Activities(ctx, a, a.ID)
		if err != nil {
			return err
		}
		waiting, err := s.src.WaitingForMe(ctx, a)
		if err != nil {
			return err
		}
		inScope := ps.in(scope)
		view = SummaryView{
			Scope: scope, DisplayCurrency: c.code, GettingStarted: gettingStarted(ps),
			Plans: planCounts(inScope), ActivitiesOwned: activityCounts(owned), WaitingForMe: waitingCounts(waiting),
			PlannedCost: plannedCostOf(inScope, c),
		}
		return nil
	})
	if err != nil {
		return SummaryView{}, err
	}
	return view, nil
}

// The reasons to show the Getting started card (HomeSummary.gettingStarted, FR-020).
const (
	notInTeam   = "notInTeam"
	noTeamPlans = "noTeamPlans"
)

// gettingStarted says why a new person sees the Getting started card (FR-020, D-170): they belong to no
// team, or none of their teams has a plan. It is nil for everyone else, whatever the scope.
func gettingStarted(ps planScope) *string {
	var reason string
	switch {
	case len(ps.myTeams) == 0:
		reason = notInTeam
	case !slices.ContainsFunc(ps.all, ps.isTeamPlan):
		reason = noTeamPlans
	default:
		return nil
	}
	return &reason
}

// planCounts counts plans by status (FR-005 card 1).
func planCounts(plans []Plan) PlanCounts {
	counts := PlanCounts{Count: len(plans)}
	for _, p := range plans {
		if p.Status == planPublished {
			counts.Published++
		} else {
			counts.Draft++
		}
	}
	return counts
}

// activityCounts counts the person's activities, and the Declined and Submitted ones by stored status
// (FR-005 card 2).
func activityCounts(owned []Activity) ActivityCounts {
	counts := ActivityCounts{Count: len(owned)}
	for _, act := range owned {
		switch act.Status {
		case activityDeclined:
			counts.Declined++
		case activitySubmitted:
			counts.Submitted++
		}
	}
	return counts
}

// waitingCounts counts what waits for the person and finds the earliest "waiting since" (FR-005 card 3,
// D-175).
func waitingCounts(waiting []Waiting) WaitingCounts {
	counts := WaitingCounts{Count: len(waiting)}
	for _, w := range waiting {
		if counts.OldestWaitingSince == nil || w.WaitingSince.Before(*counts.OldestWaitingSince) {
			since := w.WaitingSince
			counts.OldestWaitingSince = &since
		}
	}
	return counts
}

// plannedCostOf adds up the planned cost and the plan budgets of plans in US dollars and converts the totals
// once (FR-005 card 4, FR-006, FR-017): budget used is the total planned cost over the total budget of the
// plans that have one; plans without a budget are counted.
func plannedCostOf(plans []Plan, c converter) PlannedCostTotal {
	total := PlannedCostTotal{PlanCount: len(plans)}
	var planned money.Amount // a plan with nothing planned adds 0.00
	var budget *money.Amount
	for _, p := range plans {
		if p.PlannedCost != nil {
			planned = planned.Add(*p.PlannedCost)
		}
		if p.Budget == nil {
			total.PlansWithoutBudget++
			continue
		}
		sum := *p.Budget
		if budget != nil {
			sum = budget.Add(sum)
		}
		budget = &sum
	}
	total.PlannedCost, total.PlanBudget = c.of(planned), c.optional(budget)
	total.BudgetUsedPercent, total.OverBudget = budgetUsed(planned, budget)
	return total
}

// budgetUsed is planned × 100 ÷ budget rounded half up to a whole percent, and whether planned is above the
// budget, compared exactly so that 100.4 % shows "100 %" in red (domain model §4.2, D-176) — the rule of
// the plan page. Without a budget there is neither; a budget of 0.00 has no percent.
func budgetUsed(planned money.Amount, budget *money.Amount) (*int, bool) {
	if budget == nil {
		return nil, false
	}
	plannedUSD, budgetUSD := money.USD(planned), money.USD(*budget)
	return rollup.BudgetUsedPercent(&plannedUSD, &budgetUSD), planned.Cmp(*budget) > 0
}

// maxPlanRows is how many plans Plans at a glance shows (FR-010).
const maxPlanRows = 10

// PlansAtAGlance returns Plans at a glance for the scope (FR-008 – FR-010): the plans in scope, newest year
// first, then by name ignoring letter case, then by ID, at most 10, and how many are in scope. Each row has
// the plan's teams, the person's own first, its campaigns and activities, how many activities are approved
// (Approved, In progress or Complete, domain model §2.2), and its budget, planned cost and budget used as
// the plan page shows them, in the person's display currency.
func (s *Service) PlansAtAGlance(ctx context.Context, a actor.Actor, scopeText string) (PlanPage, error) {
	scope, err := ParseScope(scopeText)
	if err != nil {
		return PlanPage{}, err
	}
	var page PlanPage
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		ps, err := s.planScope(ctx, a)
		if err != nil {
			return err
		}
		inScope := ps.in(scope)
		slices.SortFunc(inScope, func(x, y Plan) int {
			return cmp.Or(cmp.Compare(y.Year, x.Year), compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
		})
		shown := inScope[:min(len(inScope), maxPlanRows)]
		page = PlanPage{Items: make([]PlanRow, 0, len(shown)), Total: len(inScope)}
		for _, p := range shown {
			page.Items = append(page.Items, ps.row(p, c))
		}
		return nil
	})
	if err != nil {
		return PlanPage{}, err
	}
	return page, nil
}

// row returns the plan p as a row of Plans at a glance, its amounts converted with c.
func (ps planScope) row(p Plan, c converter) PlanRow {
	var planned money.Amount // nothing planned yet counts as 0.00 for the budget used, as on the plan page
	if p.PlannedCost != nil {
		planned = *p.PlannedCost
	}
	percent, over := budgetUsed(planned, p.Budget)
	return PlanRow{
		ID: p.ID, Name: p.Name, Year: p.Year, StrategicAOP: p.StrategicAOP, Status: p.Status, Teams: ps.teamViews(p),
		CampaignCount: p.CampaignCount, ActivityCount: p.ActivityCount, ApprovedActivityCount: p.ApprovedActivityCount,
		Budget: c.optional(p.Budget), PlannedCost: c.optional(p.PlannedCost), BudgetUsedPercent: percent, OverBudget: over,
	}
}

// teamViews returns p's teams, the person's own first, then by name ignoring letter case (FR-008).
func (ps planScope) teamViews(p Plan) []TeamView {
	teams := make([]TeamView, 0, len(p.Teams))
	for _, t := range p.Teams {
		teams = append(teams, TeamView{ID: t.ID, Name: t.Name, Colour: t.Colour, Active: t.Active, Mine: ps.myTeams[t.ID]})
	}
	slices.SortFunc(teams, func(x, y TeamView) int {
		if x.Mine != y.Mine {
			if x.Mine {
				return -1
			}
			return 1
		}
		return cmp.Or(compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
	})
	return teams
}

// compareNames orders names as people read them: ignoring letter case.
func compareNames(x, y string) int {
	return strings.Compare(strings.ToLower(x), strings.ToLower(y))
}

// maxWaitingRows is how many activities Waiting for my approval shows (FR-013).
const maxWaitingRows = 5

// Waiting returns Waiting for my approval (FR-013): the first 5 activities of the Waiting for me tab of the
// Approvals page — the same list, oldest first — and how many wait in all, with their planned cost in a's
// display currency.
func (s *Service) Waiting(ctx context.Context, a actor.Actor) (WaitingPage, error) {
	var page WaitingPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		waiting, err := s.src.WaitingForMe(ctx, a)
		if err != nil {
			return err
		}
		shown := waiting[:min(len(waiting), maxWaitingRows)]
		page = WaitingPage{Items: make([]WaitingItem, 0, len(shown)), Total: len(waiting)}
		for _, w := range shown {
			page.Items = append(page.Items, WaitingItem{
				ID: w.ActivityID, Name: w.ActivityName, Plan: w.Plan, Owner: w.Owner, PlannedCost: c.optional(w.PlannedCost),
				Level: w.Level, Levels: w.Levels, WaitingSince: w.WaitingSince,
			})
		}
		return nil
	})
	if err != nil {
		return WaitingPage{}, err
	}
	return page, nil
}

// groupOrder is the order of the groups of My activities: the displayed statuses (FR-014, domain model §2.3).
var groupOrder = []string{"declined", "created", "submitted", "approved", "inProgress", "complete"}

// MyActivities returns My activities (FR-014): the activities a owns, in every plan, grouped by the status
// their page shows — an ongoing activity is never Complete (domain model §2.3) — in the order Declined,
// Created, Submitted, Approved, In progress, Complete, empty groups left out. Within a group they are
// ordered by start date (none last), then by name ignoring letter case, then by ID; every row is returned
// (D-173), with its planned cost in a's display currency.
func (s *Service) MyActivities(ctx context.Context, a actor.Actor) (MyActivitiesView, error) {
	var view MyActivitiesView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		owned, err := s.src.Activities(ctx, a, a.ID)
		if err != nil {
			return err
		}
		slices.SortFunc(owned, func(x, y Activity) int {
			return cmp.Or(compareStart(x.StartDate, y.StartDate), compareNames(x.Name, y.Name), strings.Compare(x.ID, y.ID))
		})
		byStatus := map[string][]MyActivityRow{}
		for _, act := range owned {
			missing := act.Missing
			if missing == nil {
				missing = []MissingField{}
			}
			byStatus[act.DisplayStatus] = append(byStatus[act.DisplayStatus], MyActivityRow{
				ID: act.ID, Name: act.Name, Plan: act.Plan, Campaign: act.Campaign, Initiative: act.Initiative,
				Status: act.DisplayStatus, StartDate: act.StartDate, EndDate: act.EndDate, Ongoing: act.Ongoing,
				PlannedCost: c.optional(act.PlannedCost), Missing: missing, ItemsBelow: act.LineItemsNeedingAttention,
			})
		}
		view = MyActivitiesView{Groups: []MyActivityGroup{}, Total: len(owned)}
		for _, status := range groupOrder {
			if rows := byStatus[status]; len(rows) > 0 {
				view.Groups = append(view.Groups, MyActivityGroup{Status: status, Count: len(rows), Items: rows})
			}
		}
		return nil
	})
	if err != nil {
		return MyActivitiesView{}, err
	}
	return view, nil
}

// compareStart orders start dates, earliest first and no date last.
func compareStart(x, y *date.Date) int {
	switch {
	case x == nil && y == nil:
		return 0
	case x == nil:
		return 1
	case y == nil:
		return -1
	}
	return x.Compare(*y)
}

// maxRecentChanges is how many entries Recent changes shows (FR-015).
const maxRecentChanges = 10

// RecentChanges returns Recent changes for the scope (FR-015, FR-016): the 10 newest activity-log entries
// about plans, campaigns, initiatives, activities and line items, for My teams only those of the plans in
// scope and whatever is or was inside them, for Everything those of every plan, and how many match. A
// deleted record keeps its entries, with where the record is now (in the recycle bin). For My teams without
// team plans the page is empty and the log is not read.
func (s *Service) RecentChanges(ctx context.Context, a actor.Actor, scopeText string) (RecentChangesPage, error) {
	scope, err := ParseScope(scopeText)
	if err != nil {
		return RecentChangesPage{}, err
	}
	page := RecentChangesPage{Items: []any{}}
	err = s.tx.Read(ctx, func(ctx context.Context) error {
		var planIDs []string // nil: every plan
		if scope == ScopeMyTeams {
			ps, err := s.planScope(ctx, a)
			if err != nil {
				return err
			}
			for _, p := range ps.in(scope) {
				planIDs = append(planIDs, p.ID)
			}
			if len(planIDs) == 0 {
				return nil
			}
		}
		found, err := s.src.RecentChanges(ctx, a, planIDs, maxRecentChanges)
		if err != nil {
			return err
		}
		page.Total = found.Total
		if found.Items != nil {
			page.Items = found.Items
		}
		return nil
	})
	if err != nil {
		return RecentChangesPage{}, err
	}
	return page, nil
}
