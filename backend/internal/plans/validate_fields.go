package plans

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

// text returns a sent text without its surrounding spaces; "" when it was not sent (null).
func text(s *string) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(*s)
}

// checkName checks a plan name, already trimmed: required, at most 120 characters, and no other stored
// plan (than exceptID) with the same name ignoring letter case (data model §1). A plan in the recycle bin is
// not stored, so its name is free.
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

// planNamed returns the plan among all, other than exceptID, whose name is name ignoring letter case, and
// false when there is none. Names are stored trimmed, so surrounding spaces are ignored too.
func planNamed(all []Plan, name, exceptID string) (Plan, bool) {
	for _, p := range all {
		if p.ID != exceptID && strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Plan{}, false
}

// years returns the years a new plan may have: the current year − 1 to + 5 (FR-006).
func (s *Service) years() (first, last int) {
	current := s.now().Year()
	return current - firstYearBeforeCurrent, current + lastYearAfterCurrent
}

// checkYear checks the year of a new plan and returns it (0 when it is wrong).
func (s *Service) checkYear(year *int, problems *errs.Problems) int {
	first, last := s.years()
	if year == nil || *year < first || *year > last {
		problems.Add("year", fmt.Sprintf(msgChooseYear, first, last))
		return 0
	}
	return *year
}

// checkModel checks the model: required, known, and active unless the plan already has it.
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

// checkBudget checks the plan budget, when one is sent: in USD, an amount with at most two decimals, not
// negative. It returns the budget to store; nil for none or a wrong one.
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

// checkFinanceBudget checks the default finance budget, when one is chosen: known and of the plan's year
// (FR-007). Finance budgets have no status (D-193).
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

// checkAttributes has Settings check the plan's Shell attribute values (FR-008) and returns the values to
// store; never nil. Values of attributes that are hidden are kept (R9).
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
