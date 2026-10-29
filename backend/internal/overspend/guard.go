package overspend

import (
	"context"

	"mapp/internal/platform/money"
)

type guardKey struct{}

// Guard is a transactor whose outermost Write refuses, rolling everything back, a save that takes a budget set
// to Refuse over it (FR-009, research R3). Every Write inside it joins it and is checked once, with it.
type Guard struct {
	tx      Transactor
	budgets *reader
	checks  bool
}

func NewGuard(tx Transactor, repo Budgets) *Guard {
	return &Guard{tx: tx, budgets: &reader{repo: repo}, checks: true}
}

func (g *Guard) UseRates(r Rates) {
	g.budgets.rates = r
}

// Unchecked is the same transaction without the check: a guarded Write inside one of its Writes is not checked.
func (g *Guard) Unchecked() *Guard {
	return &Guard{tx: g.tx, budgets: g.budgets}
}

func (g *Guard) Read(ctx context.Context, fn func(ctx context.Context) error) error {
	return g.tx.Read(ctx, fn)
}

func (g *Guard) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if ctx.Value(guardKey{}) != nil {
		return g.tx.Write(ctx, fn)
	}
	return g.tx.Write(ctx, func(ctx context.Context) error {
		ctx = context.WithValue(ctx, guardKey{}, true)
		if !g.checks {
			return fn(ctx)
		}
		return g.checked(ctx, fn)
	})
}

func (g *Guard) checked(ctx context.Context, fn func(ctx context.Context) error) error {
	before, err := g.budgets.all(ctx)
	if err != nil {
		return err
	}
	if err = fn(ctx); err != nil {
		return err
	}
	after, err := g.budgets.all(ctx)
	if err != nil {
		return err
	}
	if overages := Overages(before, after); len(overages) > 0 {
		return Refusal(overages)
	}
	return nil
}

// reader reads every budget with its planned cost, global budgets in USD at the current rates (FR-008).
type reader struct {
	repo  Budgets
	rates Rates
}

func (r *reader) all(ctx context.Context) ([]Budget, error) {
	budgets, err := r.repo.PlanningBudgets(ctx)
	if err != nil {
		return nil, err
	}
	globals, err := r.repo.GlobalBudgets(ctx)
	if err != nil || len(globals) == 0 {
		return budgets, err
	}
	rates, err := r.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	for _, g := range globals {
		b, err := inUSD(g, rates)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, b)
	}
	return budgets, nil
}

func (r *reader) currentRates(ctx context.Context) (money.Rates, error) {
	if r.rates == nil {
		return money.Rates{}, nil
	}
	return r.rates.Rates(ctx)
}

func inUSD(g GlobalBudget, rates money.Rates) (Budget, error) {
	var planned money.Amount
	planIDs := make([]string, 0, len(g.Plans))
	for _, p := range g.Plans {
		usd, err := rates.ToUSD(p.Planned)
		if err != nil {
			return Budget{}, err
		}
		planned = planned.Add(usd)
		planIDs = append(planIDs, p.PlanID)
	}
	return Budget{Kind: KindGlobalBudget, ID: g.ID, Name: g.Name, Mode: g.Mode.Normalised(), Amount: money.USD(g.Amount),
		Planned: money.USD(planned), PlanIDs: planIDs}, nil
}
