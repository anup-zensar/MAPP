package budget

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// The limits of a global budget (014 data model §1.4).
const (
	maxNameLength          = 120
	maxDescriptionLength   = 2000
	firstYearBeforeCurrent = 1
	lastYearAfterCurrent   = 5
)

// The messages of global budgets and allocations (014 plan "User-visible messages").
const (
	msgEnterName            = "Enter a name for the global budget."
	msgNameTooLong          = "Use 120 characters or fewer."
	msgNameTaken            = "A global budget called '%s' already exists."
	msgChooseYear           = "Choose a year from %d to %d."
	msgYearLocked           = "The year can't change once the global budget is saved."
	msgEnterAmount          = "Enter an amount of 0 or more."
	msgTwoDecimals          = "Use at most 2 decimals."
	msgAmountInUSD          = "Enter the amount in USD."
	msgChooseOwner          = "Choose a person who can manage global budgets for all markets."
	msgDescriptionTooLong   = "Use 2,000 characters or fewer."
	msgReload               = "Reload the page and try again."
	msgChoosePlan           = "Choose a plan."
	msgPlanOfYear           = "Choose a plan of %d."
	msgPlanAllocated        = "'%s' already has an allocation from '%s'."
	msgAmountInPlanCurrency = "Enter the amount in %s, the plan's currency."
	msgInUseOne             = "'%s' has an allocation to 1 plan. Remove it first."
	msgInUseMany            = "'%s' has allocations to %d plans. Remove them first."
	msgChooseListYear       = "Choose a year such as 2026."
	msgBudgetGone           = "This global budget no longer exists — someone may have deleted it."
	msgAllocationGone       = "This allocation no longer exists — someone may have removed it."
)

const (
	commentAllocated     = "Allocated from '%s'"
	commentRemoved       = "Allocation from '%s' removed"
	noteBudgetDeleted    = "The allocation was not restored because its global budget was deleted."
	notePlanReallocated  = "The allocation was not restored because the plan has another allocation."
	undoNameTakenMessage = "Can't undo — there is now another global budget called '%s'. Restore it from the recycle bin instead."
	renamedNote          = "Renamed to '%s' because the name was taken."
)

func inUseMessage(name string, n int) string {
	if n == 1 {
		return fmt.Sprintf(msgInUseOne, name)
	}
	return fmt.Sprintf(msgInUseMany, name, n)
}

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

// checkName checks a trimmed name: required, at most 120 characters, single-line, unique ignoring case.
func (s *Service) checkName(ctx context.Context, name, exceptID string, problems *errs.Problems) error {
	switch {
	case name == "":
		problems.Add("name", msgEnterName)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", msgNameTooLong)
	case record.CheckSingleLine(name) != "":
		problems.Add("name", record.CheckSingleLine(name))
	default:
		all, err := s.repo.ListBudgets(ctx)
		if err != nil {
			return err
		}
		if other, taken := budgetNamed(all, name, exceptID); taken {
			problems.Add("name", fmt.Sprintf(msgNameTaken, other.Name))
		}
	}
	return nil
}

func budgetNamed(all []GlobalBudget, name, exceptID string) (GlobalBudget, bool) {
	for _, b := range all {
		if b.ID != exceptID && strings.EqualFold(b.Name, name) {
			return b, true
		}
	}
	return GlobalBudget{}, false
}

func checkDescription(description string, problems *errs.Problems) {
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		problems.Add("description", msgDescriptionTooLong)
	}
}

func (s *Service) checkOwner(ctx context.Context, ownerID string, problems *errs.Problems) error {
	if ownerID == "" {
		problems.Add("ownerId", msgChooseOwner)
		return nil
	}
	refs, err := s.directory.PersonRefs(ctx, []string{ownerID})
	if err != nil {
		return err
	}
	if person, found := refs[ownerID]; !found || !person.Active {
		problems.Add("ownerId", msgChooseOwner)
		return nil
	}
	manages, err := s.access.CanManageAll(ctx, ownerID)
	if err != nil {
		return err
	}
	if !manages {
		problems.Add("ownerId", msgChooseOwner)
	}
	return nil
}

func checkAmount(in *money.Input, currency, wrongCurrency, field string, problems *errs.Problems) (money.Amount, bool) {
	if in == nil {
		problems.Add(field, msgEnterAmount)
		return money.Amount{}, false
	}
	amount, err := money.ParseAmount(strings.TrimSpace(in.Amount))
	switch {
	case in.Currency != currency:
		problems.Add(field, wrongCurrency)
	case errors.Is(err, money.ErrTooManyDecimals):
		problems.Add(field, msgTwoDecimals)
	case err != nil, amount.Sign() < 0:
		problems.Add(field, msgEnterAmount)
	default:
		return amount, true
	}
	return money.Amount{}, false
}

func checkVersion(version *int, field string, problems *errs.Problems) {
	if version == nil || *version < 1 {
		problems.Add(field, msgReload)
	}
}
