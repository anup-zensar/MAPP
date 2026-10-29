package plan

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const (
	msgChooseScope    = "Choose all or myTeams."
	msgChooseStatus   = "Choose draft or published."
	msgChooseListYear = "Choose a year such as 2026."
	msgChooseBCPIR    = "Choose include, only or exclude."
	bcpirOnly         = "only"
	bcpirExclude      = "exclude"
)

func filterProblems(filter ListFilter) errs.Problems {
	var problems errs.Problems
	if !slices.Contains([]string{"", ScopeAll, ScopeMyTeams}, filter.Scope) {
		problems.Add("scope", msgChooseScope)
	}
	if !slices.Contains([]string{"", string(StatusDraft), string(StatusPublished)}, filter.Status) {
		problems.Add("status", msgChooseStatus)
	}
	if filter.YearFrom != nil && filter.YearTo != nil && *filter.YearTo < *filter.YearFrom {
		problems.Add("yearTo", msgYearToBeforeFrom)
	}
	if !slices.Contains([]string{"", "include", bcpirOnly, bcpirExclude}, filter.BCPIR) {
		problems.Add("bcpir", msgChooseBCPIR)
	}
	for _, t := range filter.Types {
		if !slices.Contains(planTypes, PlanType(t)) {
			problems.Add("type", msgChoosePlanType)
			break
		}
	}
	return problems
}

const msgYearToBeforeFrom = "Choose a last year that is not before the first year."

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

// List shows only the plans a may see; the summary covers the team scope, the years and the plan types
// whatever the status, the BCPIR filter and the search, and adds each plan's totals in USD.
func (s *Service) List(ctx context.Context, a actor.Actor, filter ListFilter) (PlanList, error) {
	if err := filterProblems(filter).Err(); err != nil {
		return PlanList{}, err
	}
	var list PlanList
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		scoped, err := s.inScope(ctx, a, filter.Scope)
		if err != nil {
			return err
		}
		slices.SortFunc(scoped, func(x, y Plan) int {
			return cmp.Or(cmp.Compare(y.Year, x.Year), strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)),
				strings.Compare(x.ID, y.ID))
		})
		r, err := s.listReadsOf(ctx, a, scoped)
		if err != nil {
			return err
		}
		ranged := filter.plansInRange(scoped)
		list = PlanList{Total: len(scoped), Years: yearsOf(scoped), Summary: r.summaryOf(ranged)}
		list.Items, err = s.itemsOf(ctx, ranged, filter, r)
		return err
	})
	if err != nil {
		return PlanList{}, err
	}
	return list, nil
}

type listReads struct {
	facts   map[string]*Facts
	display converter
	rates   money.Rates
	lookups lookups
}

// listReadsOf reads what the list needs once for all the plans, not plan by plan.
func (s *Service) listReadsOf(ctx context.Context, a actor.Actor, plans []Plan) (listReads, error) {
	planIDs := make([]string, 0, len(plans))
	for _, p := range plans {
		planIDs = append(planIDs, p.ID)
	}
	var r listReads
	var err error
	if r.facts, err = s.facts(ctx, planIDs); err != nil {
		return listReads{}, err
	}
	if r.display, err = s.converter(ctx, a); err != nil {
		return listReads{}, err
	}
	if r.rates, err = s.settings.Rates(ctx); err != nil {
		return listReads{}, err
	}
	if r.lookups, err = s.lookupsFor(ctx, plans); err != nil {
		return listReads{}, err
	}
	return r, nil
}

func (filter ListFilter) plansInRange(plans []Plan) []Plan {
	return slices.DeleteFunc(slices.Clone(plans), func(p Plan) bool { return !filter.inRange(p) })
}

// yearsOf keeps the plans' order, so sorted plans give the newest year first.
func yearsOf(plans []Plan) []int {
	years := []int{}
	for _, p := range plans {
		if !slices.Contains(years, p.Year) {
			years = append(years, p.Year)
		}
	}
	return years
}

func (r listReads) summaryOf(plans []Plan) ListSummary {
	var summary ListSummary
	var budgets, plannedCosts []money.Amount
	for _, p := range plans {
		summary.Plans++
		if p.Status == StatusPublished {
			summary.Published++
		} else {
			summary.Draft++
		}
		if p.Budget != nil {
			budgets = append(budgets, usdOf(*p.Budget, p.Currency, r.rates))
		}
		if planned := r.facts[p.ID].PlannedCost; planned != nil {
			plannedCosts = append(plannedCosts, usdOf(*planned, p.Currency, r.rates))
		}
	}
	summary.Budget = r.display.optional(sumOf(budgets))
	summary.PlannedCost = r.display.optional(sumOf(plannedCosts))
	return summary
}

func (s *Service) itemsOf(ctx context.Context, plans []Plan, filter ListFilter, r listReads) ([]PlanSummary, error) {
	items := []PlanSummary{}
	query := strings.ToLower(strings.TrimSpace(filter.Q))
	for _, p := range plans {
		if !filter.matches(p, query) {
			continue
		}
		item, err := s.itemOf(ctx, p, r)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (filter ListFilter) matches(p Plan, query string) bool {
	if filter.Status != "" && string(p.Status) != filter.Status {
		return false
	}
	if (filter.BCPIR == bcpirOnly && !p.BCPIR) || (filter.BCPIR == bcpirExclude && p.BCPIR) {
		return false
	}
	return strings.Contains(strings.ToLower(p.Name), query) || strings.Contains(strings.ToLower(p.ID), query)
}

func (s *Service) itemOf(ctx context.Context, p Plan, r listReads) (PlanSummary, error) {
	f := *r.facts[p.ID]
	sh, err := s.shownOf(ctx, p, f, r.lookups, r.display.reading(p.Currency, r.rates))
	if err != nil {
		return PlanSummary{}, err
	}
	own := identity(p.Currency)
	return PlanSummary{
		ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status, AOP: p.AOP, BCPIR: p.BCPIR, Owner: sh.owner, Teams: sh.teams,
		BusinessUnit: sh.businessUnit, Figures: sh.figures, Completeness: sh.completeness,
		Access:   authz.PlanAccessOf(ctx, AccessFacts(p)),
		PlanType: p.PlanType, Currency: p.Currency, PlanBudget: own.optional(p.Budget), PlanPlannedCost: own.optional(f.PlannedCost),
	}, nil
}

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

func sumOf(amounts []money.Amount) *money.Amount {
	if len(amounts) == 0 {
		return nil
	}
	total := money.Sum(amounts...)
	return &total
}
