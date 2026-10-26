package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/money"
)

// The readers below are how later features read Settings (002 plan, "How later features read Settings").
// They take no actor: they only read, inside the caller's transaction when ctx carries one (the store's
// Read joins it), so a check and the save that follows see the same Settings. Records in the recycle bin
// are never returned; lists are sorted as the Settings pages show them and are never nil.

// Model returns the model id, and false when there is none.
func (s *Service) Model(ctx context.Context, id string) (Model, bool, error) {
	return get(ctx, s, func(ctx context.Context) (Model, bool, error) { return s.repo.GetModel(ctx, id) })
}

// Models returns every model, active and inactive, by name ignoring letter case.
func (s *Service) Models(ctx context.Context) ([]Model, error) {
	return list(ctx, s, s.repo.ListModels, func(a, b Model) int { return compareNames(a.Name, b.Name) })
}

// ActivityType returns the activity type id, and false when there is none.
func (s *Service) ActivityType(ctx context.Context, id string) (ActivityType, bool, error) {
	return get(ctx, s, func(ctx context.Context) (ActivityType, bool, error) { return s.repo.GetActivityType(ctx, id) })
}

// ActivityTypes returns every activity type, active and inactive, by name ignoring letter case.
func (s *Service) ActivityTypes(ctx context.Context) ([]ActivityType, error) {
	return list(ctx, s, s.repo.ListActivityTypes, func(a, b ActivityType) int { return compareNames(a.Name, b.Name) })
}

// Attributes returns the Shell attributes of level, active and inactive, in display order (FR-019).
func (s *Service) Attributes(ctx context.Context, level Level) ([]Attribute, error) {
	var attributes []Attribute
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.ListAttributes(ctx)
		attributes = ofLevel(all, level)
		return err
	})
	return attributes, err
}

// Options returns the options of the attribute attributeID, active and inactive, in display order.
func (s *Service) Options(ctx context.Context, attributeID string) ([]Option, error) {
	return list(ctx, s, func(ctx context.Context) ([]Option, error) { return s.repo.OptionsOf(ctx, attributeID) }, compareOptions)
}

// Option returns the option id, and false when there is none.
func (s *Service) Option(ctx context.Context, id string) (Option, bool, error) {
	return get(ctx, s, func(ctx context.Context) (Option, bool, error) { return s.repo.GetOption(ctx, id) })
}

// Currency returns the currency with the ISO code, and false when there is none. Its Rate is its current
// rate (016 FR-002), with who recorded it and when.
func (s *Service) Currency(ctx context.Context, code string) (Currency, bool, error) {
	return get(ctx, s, func(ctx context.Context) (Currency, bool, error) {
		c, found, err := s.repo.GetCurrency(ctx, code)
		if err != nil || !found {
			return c, found, err
		}
		book, err := s.rateBook(ctx)
		return withRate(c, book, s.today()), true, err
	})
}

// Currencies returns every currency, active and inactive: USD first, then by code; each with its current
// rate (016 FR-002).
func (s *Service) Currencies(ctx context.Context) ([]Currency, error) {
	return list(ctx, s, s.currenciesWithRates, compareCurrencies)
}

// currenciesWithRates reads every currency with its current rate, inside the caller's transaction.
func (s *Service) currenciesWithRates(ctx context.Context) ([]Currency, error) {
	currencies, err := s.repo.ListCurrencies(ctx)
	if err != nil {
		return nil, err
	}
	book, err := s.rateBook(ctx)
	if err != nil {
		return nil, err
	}
	today := s.today()
	for i, c := range currencies {
		currencies[i] = withRate(c, book, today)
	}
	return currencies, nil
}

// Rates returns the current rate of every currency, active or not (FR-029; 016 FR-002: the latest dated rate
// on or before today), for conversions with money.Rates at read time (D-041).
func (s *Service) Rates(ctx context.Context) (money.Rates, error) {
	currencies, err := s.Currencies(ctx)
	if err != nil {
		return nil, err
	}
	rates := money.Rates{}
	for _, c := range currencies {
		rates[c.Code] = c.Rate
	}
	return rates, nil
}

// FinanceBudget returns the finance budget id, and false when there is none.
func (s *Service) FinanceBudget(ctx context.Context, id string) (FinanceBudget, bool, error) {
	return get(ctx, s, func(ctx context.Context) (FinanceBudget, bool, error) { return s.repo.GetFinanceBudget(ctx, id) })
}

// FinanceBudgetsOfYear returns the finance budgets of year that have at least one item, by name: a budget
// without items is not offered for new choices (data model §7).
func (s *Service) FinanceBudgetsOfYear(ctx context.Context, year int) ([]FinanceBudget, error) {
	var budgets []FinanceBudget
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.ListFinanceBudgets(ctx)
		if err != nil {
			return err
		}
		items, err := s.repo.ListFinanceBudgetItems(ctx)
		if err != nil {
			return err
		}
		hasItems := map[string]bool{}
		for _, i := range items {
			hasItems[i.BudgetID] = true
		}
		budgets = filter(all, func(b FinanceBudget) bool { return b.Year == year && hasItems[b.ID] })
		slices.SortStableFunc(budgets, func(a, b FinanceBudget) int { return compareNames(a.Name, b.Name) })
		return nil
	})
	return budgets, err
}

// FinanceBudgetItem returns the finance budget item id, and false when there is none.
func (s *Service) FinanceBudgetItem(ctx context.Context, id string) (FinanceBudgetItem, bool, error) {
	return get(ctx, s, func(ctx context.Context) (FinanceBudgetItem, bool, error) {
		return s.repo.GetFinanceBudgetItem(ctx, id)
	})
}

// FinanceBudgetItems returns the items of the finance budget budgetID, active and inactive, by name.
func (s *Service) FinanceBudgetItems(ctx context.Context, budgetID string) ([]FinanceBudgetItem, error) {
	return list(ctx, s, func(ctx context.Context) ([]FinanceBudgetItem, error) { return s.repo.ItemsOf(ctx, budgetID) },
		func(a, b FinanceBudgetItem) int { return compareNames(a.Name, b.Name) })
}

// CanChoose is the rule of inactive Settings records (FR-004, domain model §6): a record may choose id when
// it is active, or keep it when it already has it (previousID), but never newly choose an inactive one.
func CanChoose(active bool, id, previousID string) bool {
	return active || id == previousID
}

// Visible reports whether the attribute a is shown on a record whose plan's business unit is businessUnit
// (Business unit option ids): always when it has no condition, otherwise when the business unit includes
// any of its condition's options (R9, FR-023, FR-026). A hidden attribute keeps its value and is never
// missing.
func Visible(a Attribute, businessUnit []string) bool {
	if len(a.Condition) == 0 {
		return true
	}
	for _, optionID := range a.Condition {
		if slices.Contains(businessUnit, optionID) {
			return true
		}
	}
	return false
}

// get reads one record with read inside a Read of s's store.
func get[T any](ctx context.Context, s *Service, read func(context.Context) (T, bool, error)) (T, bool, error) {
	var record T
	var found bool
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		record, found, err = read(ctx)
		return err
	})
	return record, found, err
}

// list reads records with read inside a Read of s's store and sorts them with compare.
func list[T any](ctx context.Context, s *Service, read func(context.Context) ([]T, error), compare func(a, b T) int) ([]T, error) {
	var records []T
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		records, err = read(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(records, compare)
	return records, nil
}

// compareCurrencies orders currencies as the Currencies page lists them: USD first, then by code.
func compareCurrencies(a, b Currency) int {
	switch {
	case a.Code == b.Code:
		return 0
	case a.Code == money.BaseCurrency:
		return -1
	case b.Code == money.BaseCurrency:
		return 1
	}
	return compareIDs(a.Code, b.Code)
}
