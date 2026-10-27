package plans

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/store"
)

// planGoneMessage answers a plan that does not exist (any more).
const planGoneMessage = "This plan no longer exists — someone may have deleted it."

// fallbackColour is the colour of a team that no longer resolves: a neutral one of the palette.
const fallbackColour = "slate"

// Service holds the rules of plans (feature 004): every check, calculation and change of a plan.
type Service struct {
	tx        store.Transactor
	repo      Store
	ids       *ids.Generator
	history   *history.Service
	settings  Settings
	directory Directory
	now       clock.Clock
	levels    []Level // the levels below the plan, top-down (RegisterLevel)

	allocations Allocations // 014's global-budget allocations (UseAllocations); nil: no plan is allocated
}

// NewService returns the plans service. It keeps plans in repo and reads Settings and people and teams
// through settings and directory. It knows no level below the plan until RegisterLevel adds one.
func NewService(tools history.Tools, repo Store, settings Settings, directory Directory) *Service {
	return &Service{
		tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, settings: settings, directory: directory, now: tools.Now,
	}
}

// Defaults returns what a new plan starts with for a (FR-006): the current year and the years a plan may
// have, a's active teams by name, a as owner when they have one (R1), the only active model, No for AOP
// and BCPIR, Evenly and INQ.
func (s *Service) Defaults(ctx context.Context, a actor.Actor) (Defaults, error) {
	first, last := s.years()
	defaults := Defaults{
		CurrentYear: s.now().Year(), Year: s.now().Year(), Years: []int{}, TeamIDs: []string{},
		DefaultSpread: SpreadEvenly, GoalStage: StageINQ, Currency: money.BaseCurrency, PlanType: TypeOperational,
	}
	for year := first; year <= last; year++ {
		defaults.Years = append(defaults.Years, year)
	}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		teamIDs, err := s.directory.TeamIDsOf(ctx, a.ID)
		if err != nil {
			return err
		}
		teams, err := s.directory.TeamRefs(ctx, teamIDs)
		if err != nil {
			return err
		}
		active := []people.TeamRef{}
		for _, id := range teamIDs {
			if team, found := teams[id]; found && team.Active {
				active = append(active, team)
			}
		}
		for _, team := range sortedTeams(active) {
			defaults.TeamIDs = append(defaults.TeamIDs, team.ID)
		}
		if len(defaults.TeamIDs) > 0 {
			defaults.OwnerID = &a.ID
		}
		models, err := s.settings.Models(ctx)
		if err != nil {
			return err
		}
		var activeModels []string
		for _, m := range models {
			if m.Active {
				activeModels = append(activeModels, m.ID)
			}
		}
		if len(activeModels) == 1 {
			defaults.ModelID = &activeModels[0]
		}
		return nil
	})
	if err != nil {
		return Defaults{}, err
	}
	return defaults, nil
}

// The messages of the list's filters (plan "User-visible messages").
const (
	msgChooseScope    = "Choose all or myTeams."
	msgChooseStatus   = "Choose draft or published."
	msgChooseListYear = "Choose a year such as 2026."
)

// filterProblems checks the team scope, the status, the year range and the plan types of a list filter;
// empty means every plan.
func filterProblems(filter ListFilter) errs.Problems {
	var problems errs.Problems
	if !slices.Contains([]string{"", ScopeAll, ScopeMyTeams}, filter.Scope) {
		problems.Add("scope", msgChooseScope)
	}
	if !slices.Contains([]string{"", string(StatusDraft), string(StatusPublished)}, filter.Status) {
		problems.Add("status", msgChooseStatus)
	}
	if filter.YearFrom != nil && filter.YearTo != nil && *filter.YearTo < *filter.YearFrom { // 016 FR-033
		problems.Add("yearTo", msgYearToBeforeFrom)
	}
	for _, t := range filter.Types {
		if !slices.Contains(planTypes, PlanType(t)) {
			problems.Add("type", msgChoosePlanType)
			break
		}
	}
	return problems
}

// msgYearToBeforeFrom refuses a year range that ends before it starts (016 FR-033).
const msgYearToBeforeFrom = "Choose a last year that is not before the first year."

// inRange reports whether p matches the year range and the plan types of filter (016 FR-033): year is the
// alias of a one-year range.
func (filter ListFilter) inRange(p Plan) bool {
	from, to := filter.YearFrom, filter.YearTo
	if filter.Year != nil {
		from, to = filter.Year, filter.Year
	}
	if (from != nil && p.Year < *from) || (to != nil && p.Year > *to) {
		return false
	}
	return len(filter.Types) == 0 || slices.Contains(filter.Types, string(p.PlanType))
}

// List returns the plans of the Plans page (FR-001 – FR-004, D-080): only the plans a may see (014 FR-007).
// The team scope comes next: All plans, or My teams — plans with one of a's teams, which only narrows the
// list (Q-006). Total counts the plans in the scope; the summary covers the scope, the year range and the
// plan types (016 FR-033), whatever the status and the search; the items match every filter — the search is
// any part of the name or the ID, ignoring letter case — newest year first, then by name. Each level below
// is asked once for all the plans (SC-009). Each plan's amounts are in its plan currency with the display
// currency beside; the summary converts each plan's totals to USD before adding them (016 FR-014).
func (s *Service) List(ctx context.Context, a actor.Actor, filter ListFilter) (PlanList, error) {
	if err := filterProblems(filter).Err(); err != nil {
		return PlanList{}, err
	}
	list := PlanList{Items: []PlanSummary{}, Years: []int{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		scoped, err := s.inScope(ctx, a, filter.Scope)
		if err != nil {
			return err
		}
		slices.SortFunc(scoped, func(x, y Plan) int {
			return cmp.Or(cmp.Compare(y.Year, x.Year), strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)),
				strings.Compare(x.ID, y.ID))
		})
		planIDs := make([]string, 0, len(scoped))
		for _, p := range scoped {
			planIDs = append(planIDs, p.ID)
			if !slices.Contains(list.Years, p.Year) {
				list.Years = append(list.Years, p.Year) // newest first, as the plans are sorted
			}
		}
		list.Total = len(scoped)
		facts, err := s.facts(ctx, planIDs)
		if err != nil {
			return err
		}
		c, err := s.converter(ctx, a)
		if err != nil {
			return err
		}
		rates, err := s.settings.Rates(ctx)
		if err != nil {
			return err
		}
		l, err := s.lookupsFor(ctx, scoped)
		if err != nil {
			return err
		}

		var budgets, plannedCosts []money.Amount // in USD, each plan converted once
		query := strings.ToLower(strings.TrimSpace(filter.Q))
		for _, p := range scoped {
			if !filter.inRange(p) {
				continue
			}
			f := *facts[p.ID]
			list.Summary.Plans++
			if p.Status == StatusPublished {
				list.Summary.Published++
			} else {
				list.Summary.Draft++
			}
			if p.Budget != nil {
				budgets = append(budgets, usdOf(*p.Budget, p.Currency, rates))
			}
			if f.PlannedCost != nil {
				plannedCosts = append(plannedCosts, usdOf(*f.PlannedCost, p.Currency, rates))
			}
			if (filter.Status != "" && string(p.Status) != filter.Status) ||
				!(strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.ID), query)) {
				continue
			}
			sh, err := s.shownOf(ctx, p, f, l, c.reading(p.Currency, rates))
			if err != nil {
				return err
			}
			own := identity(p.Currency)
			list.Items = append(list.Items, PlanSummary{
				ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status, AOP: p.AOP, Owner: sh.owner, Teams: sh.teams,
				BusinessUnit: sh.businessUnit, Figures: sh.figures, Completeness: sh.completeness,
				Access:   authz.PlanAccessOf(ctx, AccessFacts(p)), // 014 FR-011
				PlanType: p.PlanType, Currency: p.Currency, PlanBudget: own.optional(p.Budget), PlanPlannedCost: own.optional(f.PlannedCost),
			})
		}
		list.Summary.Budget = c.optional(sumOf(budgets))
		list.Summary.PlannedCost = c.optional(sumOf(plannedCosts))
		return nil
	})
	if err != nil {
		return PlanList{}, err
	}
	return list, nil
}

// inScope returns the plans of the team scope: every plan, or with myTeams those with one of a's teams.
func (s *Service) inScope(ctx context.Context, a actor.Actor, scope string) ([]Plan, error) {
	all, err := s.repo.ListInScope(ctx, authz.ScopeOf(ctx))
	if err != nil || scope != ScopeMyTeams {
		return all, err
	}
	mine, err := s.directory.TeamIDsOf(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	scoped := []Plan{}
	for _, p := range all {
		if slices.ContainsFunc(p.TeamIDs, func(id string) bool { return slices.Contains(mine, id) }) {
			scoped = append(scoped, p)
		}
	}
	return scoped, nil
}

// sumOf adds amounts up; nil when there are none, so a total of nothing shows "—", not 0.00.
func sumOf(amounts []money.Amount) *money.Amount {
	if len(amounts) == 0 {
		return nil
	}
	total := money.Sum(amounts...)
	return &total
}
