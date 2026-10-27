package lineitems

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// FormOptions returns what a new line item of the activity activityID starts with and the choices the
// line-item form offers (FR-008, FR-011, FR-015): the defaults, each with its source; the active currencies,
// USD first; the plan's year and every year with active finance budget items; their active items, the plan's
// default finance budget first. Its balance stays in USD, the currency the form calculates in (D-213).
// Every signed-in person may add line items (D-004), so the actor is not used yet.
func (s *Service) FormOptions(ctx context.Context, _ actor.Actor, activityID string) (FormOptions, error) {
	var options FormOptions
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		options = FormOptions{Activity: activityView(f), Balance: balanceOf(f, views)}
		if options.Defaults, err = s.defaults(ctx, f, items, options.Balance); err != nil {
			return err
		}
		if options.Currencies, err = s.activeCurrencies(ctx); err != nil {
			return err
		}
		budgetItems, err := s.deps.FinanceBudgets.Items(ctx)
		if err != nil {
			return err
		}
		options.FinanceBudgetYears, options.FinanceBudgetItems = budgetChoices(f, budgetItems)
		options.Defaults.FinanceBudgetItemID, options.Defaults.FinanceBudgetItemPreselected = preselectedItem(f, budgetItems)
		return nil
	})
	if err != nil {
		return FormOptions{}, err
	}
	return options, nil
}

// defaults are the values of a new line item (FR-008): the name "<activity name> – line item" (" 2", " 3", …
// while taken), the activity's owner when active, USD, the unassigned planned cost when it is above 0.00, the
// plan's default spread and year, and Request SWBS code from GSAP on. The finance budget item is filled in by
// the caller.
func (s *Service) defaults(ctx context.Context, f ActivityFacts, items []LineItem, balance UnassignedBalance) (Defaults, error) {
	d := Defaults{
		Name:              uniqueName(f.Name, defaultNameSuffix, items, maxNameLength),
		Currency:          f.planCurrency(), // 016: amounts are typed in the plan currency first
		PlannedCostSource: SourceNoActivityPlannedCost,
		SpreadMethod:      defaultSpread(f),
		FinanceBudgetYear: f.PlanYear,
		RequestSwbs:       true,
	}
	if f.OwnerID != "" {
		owner, found, err := s.deps.People.Person(ctx, f.OwnerID)
		if err != nil {
			return Defaults{}, err
		}
		if found && owner.Active { // a new choice must be active, so an inactive owner is no default
			d.Owner = &owner
		}
	}
	switch balance.State {
	case BalanceLeft:
		d.PlannedCost, d.PlannedCostSource = balance.Balance, SourceUnassignedBalance
	case BalanceAllAssigned, BalanceExceeded:
		d.PlannedCostSource = SourceNothingLeftToAssign
	}
	return d, nil
}

// defaultNameSuffix is the suffix of the n-th default name: " – line item", " – line item 2" and so on.
func defaultNameSuffix(n int) string {
	if n == 1 {
		return " – line item"
	}
	return fmt.Sprintf(" – line item %d", n)
}

// activeCurrencies returns the currencies a new choice may use: the active ones, USD first, then by code.
func (s *Service) activeCurrencies(ctx context.Context) ([]CurrencyView, error) {
	all, err := s.deps.Currencies.Currencies(ctx)
	if err != nil {
		return nil, err
	}
	views := []CurrencyView{}
	for _, c := range all {
		if c.Active {
			views = append(views, currencyView(c))
		}
	}
	slices.SortFunc(views, func(x, y CurrencyView) int {
		return cmp.Or(cmp.Compare(rankUSD(x.Code), rankUSD(y.Code)), strings.Compare(x.Code, y.Code))
	})
	return views, nil
}

// rankUSD puts USD before every other currency.
func rankUSD(code string) int {
	if code == money.BaseCurrency {
		return 0
	}
	return 1
}

// budgetChoices returns the finance budget years the form offers — the plan's year and every year with
// active items, ascending — and the active items of those years: by year, then the plan's default finance
// budget first in the plan's year, then by budget name and item name (FR-015).
func budgetChoices(f ActivityFacts, items []BudgetItem) ([]int, []FinanceBudgetItemRef) {
	years := yearsWithActiveItems(items)
	if !slices.Contains(years, f.PlanYear) {
		years = append(years, f.PlanYear)
		slices.Sort(years)
	}
	choices := []FinanceBudgetItemRef{}
	for _, item := range items {
		if item.Active {
			choices = append(choices, itemRef(item))
		}
	}
	isDefault := func(item FinanceBudgetItemRef) int {
		if item.Year == f.PlanYear && f.DefaultFinanceBudget != "" && item.BudgetName == f.DefaultFinanceBudget {
			return 0
		}
		return 1
	}
	slices.SortFunc(choices, func(x, y FinanceBudgetItemRef) int {
		return cmp.Or(cmp.Compare(x.Year, y.Year), cmp.Compare(isDefault(x), isDefault(y)),
			strings.Compare(x.BudgetName, y.BudgetName), strings.Compare(x.Name, y.Name))
	})
	return years, choices
}

// preselectedItem returns the only active item of the plan's default finance budget in the plan's year, when
// it has exactly one (R12).
func preselectedItem(f ActivityFacts, items []BudgetItem) (*string, bool) {
	var only []string
	for _, item := range items {
		if item.Active && item.Year == f.PlanYear && f.DefaultFinanceBudget != "" && item.BudgetName == f.DefaultFinanceBudget {
			only = append(only, item.ID)
		}
	}
	if len(only) != 1 {
		return nil, false
	}
	return &only[0], true
}
