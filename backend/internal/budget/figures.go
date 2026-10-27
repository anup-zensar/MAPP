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

// fallbackColour is the colour of a team that no longer resolves: a neutral one of the palette.
const fallbackColour = "slate"

// figurer works out the figures of global budgets inside one transaction, with the current exchange rates
// read once: allocations kept in a plan's own currency count in USD at the current rate, on read (D-303).
type figurer struct {
	s     *Service
	rates money.Rates
}

// newFigurer reads the current rates for the figures of one answer.
func (s *Service) newFigurer(ctx context.Context) (*figurer, error) {
	rates, err := s.rates.Rates(ctx)
	if err != nil {
		return nil, err
	}
	return &figurer{s: s, rates: rates}, nil
}

// allocated is one allocation with what its figures need: its plan, its amount in USD and the plan's planned
// cost in USD (nil when nothing in the plan has one).
type allocated struct {
	allocation Allocation
	plan       PlanInfo
	amountUSD  money.Amount
	plannedUSD *money.Amount
}

// allocated reads the plans and planned costs of allocations, the planned costs in one call (SC-009).
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
		if !found { // cannot happen while deleting a plan takes its allocation along; shown by its id
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

// figuresOf works out the four totals of a global budget of amount (USD) with its allocations (FR-032):
// allocated is every allocation, unallocated what is left — negative, and over-allocated, when the
// allocations add up to more, which never blocks (D-015) — and planned cost that of every allocated plan.
//
//	2,500,000.00 with 1,200,000.00 and 800,000.00 → allocated 2,000,000.00, unallocated 500,000.00
//	2,500,000.00 with 1,800,000.00 and 800,000.00 → unallocated −100,000.00, over-allocated
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

// row returns one allocation as the global budget page shows it: amount, planned cost and left to plan in
// the plan's currency, and budget used = planned cost ÷ allocation (domain model §4.2).
func (f *figurer) row(ctx context.Context, x allocated, l lookups) (AllocationRow, error) {
	currency := x.allocation.Amount.Currency
	var plannedCost *money.Money
	var planned money.Amount // no planned cost counts as 0 for left to plan and budget used
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
		Access: RowAccess{Edit: manage, Remove: manage},
	}, nil
}

// percentUsed is planned ÷ allocation × 100 with two decimals, rounded half up ("62.50"); nil when the
// allocation is 0.00. Both amounts are 0 or more.
//
//	750,000.00 of 1,200,000.00 → "62.50"     1.00 of 3.00 → "33.33"     2.00 of 3.00 → "66.67"
func percentUsed(planned, allocation money.Amount) *string {
	if allocation.IsZero() {
		return nil
	}
	numerator := new(big.Int).Mul(big.NewInt(planned.Cents()), big.NewInt(10000)) // hundredths of a percent
	denominator := big.NewInt(allocation.Cents())
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	hundredths := quotient.Int64()
	text := fmt.Sprintf("%d.%02d", hundredths/100, hundredths%100)
	return &text
}

// planRef names a plan in a row or an option.
func planRef(p PlanInfo) PlanRef {
	return PlanRef{ID: p.ID, Name: p.Name, Year: p.Year, Status: p.Status, Version: p.Version}
}

// unallocatedPlans returns the plans of year that the person can see and that no global budget allocates
// to, by name: "Plans of <year> without an allocation" (FR-032).
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

// byPlanName orders plans by name ignoring letter case, then by id.
func byPlanName(x, y PlanInfo) int {
	return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
}

// option returns a plan without an allocation as the page and the Allocate dialog offer it.
func (s *Service) option(ctx context.Context, p PlanInfo, l lookups) (AllocationOption, error) {
	canAllocate, err := availabilityOf(s.access.ManageIn(ctx, p))
	if err != nil {
		return AllocationOption{}, err
	}
	option := AllocationOption{Plan: planRef(p), Teams: l.teamsOf(p), Owner: l.person(p.OwnerID), CanAllocate: canAllocate}
	if p.Budget != nil {
		option.Budget = &money.Money{Amount: *p.Budget, Currency: p.Currency}
	}
	return option, nil
}

// view returns the global budget b as its page shows it, inside the caller's transaction (FR-032, FR-033):
// the four totals over every allocation; a row per allocation to a plan the person can see, by plan name;
// the allocations to other plans added up as Other markets; the visible plans of the year without an
// allocation; the locks; and what the person may do.
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
	if len(hidden) > 0 { // FR-033: other markets' allocations are added up, never shown one by one
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
	if len(allocations) > 0 {
		inUse := inUseMessage(b.Name, len(allocations))
		view.Locks = append(view.Locks, settings.Lock{Field: "delete", Reason: inUse})
		if view.Access.Delete.Available {
			view.Access.Delete = Availability{Reason: inUse}
		}
	}
	return view, nil
}
