package budget

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"

	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

const fallbackColour = "slate"

type figurer struct {
	s     *Service
	rates money.Rates
}

func (s *Service) newFigurer(ctx context.Context) (*figurer, error) {
	rates, err := s.rates.Rates(ctx)
	if err != nil {
		return nil, err
	}
	return &figurer{s: s, rates: rates}, nil
}

type allocated struct {
	allocation Allocation
	plan       PlanInfo
	amountUSD  money.Amount
	plannedUSD *money.Amount
}

func (f *figurer) allocated(ctx context.Context, allocations []Allocation) ([]allocated, error) {
	planIDs := make([]string, 0, len(allocations))
	for _, a := range allocations {
		planIDs = append(planIDs, a.PlanID)
	}
	planned, err := f.s.plans.PlannedCosts(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	list := make([]allocated, 0, len(allocations))
	for _, a := range allocations {
		plan, found, err := f.s.plans.Plan(ctx, a.PlanID)
		if err != nil {
			return nil, err
		}
		if !found {
			plan = PlanInfo{ID: a.PlanID, Name: a.PlanID, Currency: a.Amount.Currency, TeamIDs: []string{}}
		}
		usd, err := f.rates.ToUSD(a.Amount)
		if err != nil {
			return nil, err
		}
		list = append(list, allocated{allocation: a, plan: plan, amountUSD: usd, plannedUSD: planned[a.PlanID]})
	}
	return list, nil
}

func figuresOf(amount money.Amount, list []allocated) Figures {
	var total, planned money.Amount
	for _, x := range list {
		total = total.Add(x.amountUSD)
		if x.plannedUSD != nil {
			planned = planned.Add(*x.plannedUSD)
		}
	}
	unallocated := amount.Sub(total)
	return Figures{
		Amount: money.USD(amount), Allocated: money.USD(total), Unallocated: money.USD(unallocated),
		PlannedCost: money.USD(planned), OverAllocated: unallocated.Sign() < 0,
	}
}

func (f *figurer) row(ctx context.Context, x allocated, l lookups) (AllocationRow, error) {
	currency := x.allocation.Amount.Currency
	var plannedCost *money.Money
	var planned money.Amount
	if x.plannedUSD != nil {
		local, err := f.rates.FromUSD(*x.plannedUSD, currency)
		if err != nil {
			return AllocationRow{}, err
		}
		plannedCost, planned = &local, local.Amount
	}
	manage, err := availabilityOf(f.s.access.ManageIn(ctx, x.plan))
	if err != nil {
		return AllocationRow{}, err
	}
	return AllocationRow{
		ID: x.allocation.ID, Plan: planRef(x.plan), Teams: l.teamsOf(x.plan), Owner: l.person(x.plan.OwnerID),
		Amount: x.allocation.Amount, AmountUSD: money.USD(x.amountUSD), PlannedCost: plannedCost,
		LeftToPlan:        money.Money{Amount: x.allocation.Amount.Amount.Sub(planned), Currency: currency},
		BudgetUsedPercent: percentUsed(planned, x.allocation.Amount.Amount), Version: x.allocation.Version,
		Overspend: x.plan.Overspend.Normalised(),
		Access:    RowAccess{Edit: manage, Remove: manage},
	}, nil
}

func percentUsed(planned, allocation money.Amount) *string {
	if allocation.IsZero() {
		return nil
	}
	numerator := new(big.Int).Mul(big.NewInt(planned.Cents()), big.NewInt(10000))
	denominator := big.NewInt(allocation.Cents())
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	hundredths := quotient.Int64()
	text := fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100)
	return &text
}

func planRef(p PlanInfo) PlanRef {
	return PlanRef{ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status, Version: p.Version}
}

func (s *Service) unallocatedPlans(ctx context.Context, year int) ([]PlanInfo, error) {
	ofYear, err := s.plans.PlansOfYear(ctx, year)
	if err != nil {
		return nil, err
	}
	all, err := s.repo.ListAllocations(ctx)
	if err != nil {
		return nil, err
	}
	plans := []PlanInfo{}
	for _, p := range ofYear {
		taken := slices.ContainsFunc(all, func(a Allocation) bool { return a.PlanID == p.ID })
		if !taken && s.access.CanViewPlan(ctx, p) {
			plans = append(plans, p)
		}
	}
	slices.SortFunc(plans, byPlanName)
	return plans, nil
}

func byPlanName(x, y PlanInfo) int {
	return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
}

func (s *Service) option(ctx context.Context, p PlanInfo, l lookups) (AllocationOption, error) {
	canAllocate, err := availabilityOf(s.access.ManageIn(ctx, p))
	if err != nil {
		return AllocationOption{}, err
	}
	option := AllocationOption{Plan: planRef(p), Teams: l.teamsOf(p), Owner: l.person(p.OwnerID), CanAllocate: canAllocate,
		Overspend: p.Overspend.Normalised()}
	if p.Budget != nil {
		option.Budget = &money.Money{Amount: *p.Budget, Currency: p.Currency}
	}
	return option, nil
}

func (s *Service) view(ctx context.Context, b GlobalBudget) (GlobalBudgetView, error) {
	f, err := s.newFigurer(ctx)
	if err != nil {
		return GlobalBudgetView{}, err
	}
	allocations, err := s.repo.AllocationsOf(ctx, b.ID)
	if err != nil {
		return GlobalBudgetView{}, err
	}
	list, err := f.allocated(ctx, allocations)
	if err != nil {
		return GlobalBudgetView{}, err
	}
	unallocated, err := s.unallocatedPlans(ctx, b.Year)
	if err != nil {
		return GlobalBudgetView{}, err
	}
	plans := slices.Clone(unallocated)
	for _, x := range list {
		plans = append(plans, x.plan)
	}
	l, err := s.lookupsFor(ctx, plans, b.OwnerID)
	if err != nil {
		return GlobalBudgetView{}, err
	}

	view := GlobalBudgetView{
		ID: b.ID, Name: b.Name, Year: b.Year, Amount: money.USD(b.Amount), Owner: l.person(b.OwnerID),
		Description: b.Description, Figures: figuresOf(b.Amount, list), Allocations: []AllocationRow{},
		PlansWithoutAllocation: []AllocationOption{}, Locks: []settings.Lock{{Field: "year", Reason: msgYearLocked}},
		Meta: b.Meta,
	}
	var hidden []allocated
	for _, x := range list {
		if !s.access.CanViewPlan(ctx, x.plan) {
			hidden = append(hidden, x)
			continue
		}
		row, err := f.row(ctx, x, l)
		if err != nil {
			return GlobalBudgetView{}, err
		}
		view.Allocations = append(view.Allocations, row)
	}
	slices.SortFunc(view.Allocations, func(x, y AllocationRow) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Plan.Name), strings.ToLower(y.Plan.Name)), strings.Compare(x.Plan.ID, y.Plan.ID))
	})
	if len(hidden) > 0 {
		other := figuresOf(money.Amount{}, hidden)
		view.OtherMarkets = &OtherMarkets{PlanCount: len(hidden), Amount: other.Allocated, PlannedCost: other.PlannedCost}
	}
	for _, p := range unallocated {
		option, err := s.option(ctx, p, l)
		if err != nil {
			return GlobalBudgetView{}, err
		}
		view.PlansWithoutAllocation = append(view.PlansWithoutAllocation, option)
	}

	manageAll, err := availabilityOf(s.access.ManageAll(ctx))
	if err != nil {
		return GlobalBudgetView{}, err
	}
	allocate, err := availabilityOf(s.access.Manage(ctx))
	if err != nil {
		return GlobalBudgetView{}, err
	}
	view.Access = BudgetAccess{Edit: manageAll, Delete: manageAll, Allocate: allocate}
	view.Overspend = controlOf(b, manageAll)
	if len(allocations) > 0 {
		inUse := inUseMessage(b.Name, len(allocations))
		view.Locks = append(view.Locks, settings.Lock{Field: "delete", Reason: inUse})
		if view.Access.Delete.Available {
			view.Access.Delete = Availability{Reason: inUse}
		}
	}
	return view, nil
}
