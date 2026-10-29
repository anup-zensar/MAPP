package overspend

import (
	"context"
	"errors"
	"testing"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// fakeTx commits a Write only when its function succeeds, as the database does.
type fakeTx struct{ committed int }

func (f *fakeTx) Read(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

func (f *fakeTx) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	f.committed++
	return nil
}

// fakeBudgets answers one global budget whose plan's planned cost is whatever the save left in planned.
type fakeBudgets struct {
	planned money.Money
	reads   int
}

func (f *fakeBudgets) PlanningBudgets(context.Context) ([]Budget, error) {
	f.reads++
	return []Budget{}, nil
}

func (f *fakeBudgets) GlobalBudgets(context.Context) ([]GlobalBudget, error) {
	amount, _ := money.ParseAmount("100.00")
	return []GlobalBudget{{ID: "GB-1", Name: "Global", Mode: Refuse, Amount: amount,
		Plans: []PlanCost{{PlanID: "P-1", Planned: f.planned}}}}, nil
}

type fixedRates struct{}

func (fixedRates) Rates(context.Context) (money.Rates, error) {
	return money.Rates{"EUR": money.RateFromMicros(920000)}, nil
}

func eurAmount(t *testing.T, text string) money.Money {
	t.Helper()
	m := usd(t, text)
	m.Currency = "EUR"
	return m
}

func TestGuard_RefusesAndRollsBack_037_FR009_FR010(t *testing.T) {
	t.Parallel()
	tx, budgets := &fakeTx{}, &fakeBudgets{planned: eurAmount(t, "90.00")}
	guard := NewGuard(tx, budgets)
	guard.UseRates(fixedRates{})

	within := guard.Write(t.Context(), func(context.Context) error {
		budgets.planned = eurAmount(t, "92.00") // 100.00 USD exactly
		return nil
	})
	over := guard.Write(t.Context(), func(context.Context) error {
		budgets.planned = eurAmount(t, "92.01")
		return nil
	})

	var refusal *errs.Error
	if within != nil || !errors.As(over, &refusal) || refusal.Code != CodeOverBudget || tx.committed != 1 {
		t.Errorf("within = %v, over = %v, commits %d; want the second save refused and not committed", within, over, tx.committed)
	}
}

func TestGuard_ChecksTheOutermostWriteOnce_037_FR009(t *testing.T) {
	t.Parallel()
	tx, budgets := &fakeTx{}, &fakeBudgets{planned: eurAmount(t, "10.00")}
	guard := NewGuard(tx, budgets)
	guard.UseRates(fixedRates{})

	err := guard.Write(t.Context(), func(ctx context.Context) error {
		return guard.Write(ctx, func(context.Context) error { return nil })
	})
	unchecked := guard.Unchecked().Write(t.Context(), func(ctx context.Context) error {
		return guard.Write(ctx, func(context.Context) error { return nil })
	})

	if err != nil || unchecked != nil || budgets.reads != 2 {
		t.Errorf("errors %v %v, budget reads %d; want one check of two reads, none inside an unchecked write", err, unchecked,
			budgets.reads)
	}
}
