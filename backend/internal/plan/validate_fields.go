package plan

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

func text(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// checkName leaves free the names of plans in the recycle bin, which are not stored.
func (s *Service) checkName(ctx context.Context, name, exceptID string, problems *errs.Problems) error {
	switch {
	case name == "":
		problems.Add("name", msgEnterName)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", msgNameTooLong)
	default:
		all, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		if other, taken := planNamed(all, name, exceptID); taken {
			problems.Add("name", fmt.Sprintf(msgNameTaken, other.Name))
		}
	}
	return nil
}

func planNamed(all []Plan, name, exceptID string) (Plan, bool) {
	for _, p := range all {
		if p.ID != exceptID && strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Plan{}, false
}

// years returns the years a new plan may have: the current year to + 10 (D-326).
func (s *Service) years() (first, last int) {
	current := s.now().Year()
	return current - firstYearBeforeCurrent, current + lastYearAfterCurrent
}

func (s *Service) checkYear(year *int, problems *errs.Problems) int {
	first, last := s.years()
	if year == nil || *year < first || *year > last {
		problems.Add("year", fmt.Sprintf(msgChooseYear, first, last))
		return 0
	}
	return *year
}

func (s *Service) checkModel(ctx context.Context, id string, stored *Plan, problems *errs.Problems) (string, error) {
	if id == "" {
		problems.Add("modelId", msgChooseModel)
		return "", nil
	}
	m, found, err := s.settings.Model(ctx, id)
	switch {
	case err != nil:
		return "", err
	case !found:
		problems.Add("modelId", msgUnknownModel)
	case !m.Active && (stored == nil || stored.ModelID != id):
		problems.Add("modelId", fmt.Sprintf(msgInactiveModel, m.Name))
	}
	return id, nil
}

// checkBudget returns the plan budget to store: nil for none or a wrong one.
func checkBudget(in *money.Input, currency string, problems *errs.Problems) *money.Amount {
	if in == nil {
		return nil
	}
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case in.Currency != currency:
		problems.Add("budget", fmt.Sprintf(msgAmountInPlanCurrency, currency))
	case errors.Is(err, money.ErrTooManyDecimals):
		problems.Add("budget", msgTwoDecimals)
	case err != nil:
		problems.Add("budget", msgEnterAmount)
	case amount.Sign() < 0:
		problems.Add("budget", msgBudgetNegative)
	default:
		return &amount
	}
	return nil
}

// checkFinanceBudget checks no status: finance budgets have none (D-193).
func (s *Service) checkFinanceBudget(ctx context.Context, id string, year int, problems *errs.Problems) error {
	if id == "" {
		return nil
	}
	b, found, err := s.settings.FinanceBudget(ctx, id)
	switch {
	case err != nil:
		return err
	case !found:
		problems.Add("defaultFinanceBudgetId", msgUnknownBudget)
	case year != 0 && b.Year != year: // a wrong year is reported on the year
		problems.Add("defaultFinanceBudgetId", fmt.Sprintf(msgBudgetOfYear, year))
	}
	return nil
}

// checkAttributes keeps the values of hidden attributes and never returns nil.
func (s *Service) checkAttributes(ctx context.Context, values settings.AttributeValues, stored *Plan, problems *errs.Problems) (settings.AttributeValues, error) {
	if values == nil {
		values = settings.AttributeValues{}
	}
	var previous settings.AttributeValues
	if stored != nil {
		previous = stored.Attributes
	}
	result, err := s.settings.CheckAttributes(ctx, settings.LevelPlan, settings.AttributeInput{Values: values, Previous: previous})
	if err != nil {
		return nil, err
	}
	*problems = append(*problems, result.Problems...)
	if result.Values == nil {
		return settings.AttributeValues{}, nil
	}
	return result.Values, nil
}
