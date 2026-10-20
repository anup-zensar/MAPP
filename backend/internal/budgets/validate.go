package budgets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// The limits of a global budget (014 data model §1.4).
const (
	maxNameLength          = 120 // characters
	maxDescriptionLength   = 2000
	firstYearBeforeCurrent = 1 // a global budget's year: the current year − 1 …
	lastYearAfterCurrent   = 5 // … to the current year + 5, as plans (FR-029)
)

// The messages of global budgets and allocations (014 plan "User-visible messages", numbers in brackets).
const (
	msgEnterName            = "Enter a name for the global budget."
	msgNameTooLong          = "Use 120 characters or fewer."
	msgNameTaken            = "A global budget called '%s' already exists." // 24
	msgChooseYear           = "Choose a year from %d to %d."
	msgYearLocked           = "The year can't change once the global budget is saved." // 25
	msgEnterAmount          = "Enter an amount of 0 or more."                          // 26
	msgTwoDecimals          = "Use at most 2 decimals."
	msgAmountInUSD          = "Enter the amount in USD."
	msgChooseOwner          = "Choose a person who can manage global budgets for all markets." // 27
	msgDescriptionTooLong   = "Use 2,000 characters or fewer."
	msgReload               = "Reload the page and try again."
	msgChoosePlan           = "Choose a plan."
	msgPlanOfYear           = "Choose a plan of %d."                                 // 28
	msgPlanAllocated        = "'%s' already has an allocation from '%s'."            // 29
	msgAmountInPlanCurrency = "Enter the amount in %s, the plan's currency."         // consolidation D-303
	msgInUseOne             = "'%s' has an allocation to 1 plan. Remove it first."   // 30
	msgInUseMany            = "'%s' has allocations to %d plans. Remove them first." // 30
	msgChooseListYear       = "Choose a year such as 2026."
	msgBudgetGone           = "This global budget no longer exists — someone may have deleted it."
	msgAllocationGone       = "This allocation no longer exists — someone may have removed it."
)

// The comments of the plan's log entries when an allocation sets or stops setting its budget (014 data model
// §3), and the restore notes of an allocation that comes back with its plan (message 31).
const (
	commentAllocated     = "Allocated from '%s'"
	commentRemoved       = "Allocation from '%s' removed"
	noteBudgetDeleted    = "The allocation was not restored because its global budget was deleted." // 31
	notePlanReallocated  = "The allocation was not restored because the plan has another allocation."
	undoNameTakenMessage = "Can't undo — there is now another global budget called '%s'. Restore it from the recycle bin instead."
	renamedNote          = "Renamed to '%s' because the name was taken."
)

// inUseMessage refuses to delete a global budget that still has n allocations (message 30).
func inUseMessage(name string, n int) string {
	if n == 1 {
		return fmt.Sprintf(msgInUseOne, name)
	}
	return fmt.Sprintf(msgInUseMany, name, n)
}

// years returns the years a new global budget may have: the current year − 1 to + 5.
func (s *Service) years() (first, last int) {
	current := s.now().Year()
	return current - firstYearBeforeCurrent, current + lastYearAfterCurrent
}

// checkYear checks the year of a new global budget and returns it (0 when it is wrong).
func (s *Service) checkYear(year *int, problems *errs.Problems) int {
	first, last := s.years()
	if year == nil || *year < first || *year > last {
		problems.Add("year", fmt.Sprintf(msgChooseYear, first, last))
		return 0
	}
	return *year
}

// checkName checks a global budget name, already trimmed: required, at most 120 characters, and no other
// stored global budget (than exceptID) with the same name ignoring letter case (message 24).
func (s *Service) checkName(ctx context.Context, name, exceptID string, problems *errs.Problems) error {
	switch {
	case name == "":
		problems.Add("name", msgEnterName)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", msgNameTooLong)
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

// budgetNamed returns the global budget among all, other than exceptID, whose name is name ignoring letter
// case, and false when there is none. Names are stored trimmed.
func budgetNamed(all []GlobalBudget, name, exceptID string) (GlobalBudget, bool) {
	for _, b := range all {
		if b.ID != exceptID && strings.EqualFold(b.Name, name) {
			return b, true
		}
	}
	return GlobalBudget{}, false
}

// checkDescription checks a description, already trimmed: at most 2,000 characters.
func checkDescription(description string, problems *errs.Problems) {
	if utf8.RuneCountInString(description) > maxDescriptionLength {
		problems.Add("description", msgDescriptionTooLong)
	}
}

// checkOwner checks the owner of a global budget: an active person who holds globalBudgets.manage for All
// markets (FR-029, message 27).
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

// checkAmount checks a money input on field: sent, in currency (wrongCurrency otherwise), an amount with at
// most two decimals, 0 or more (message 26). It returns the amount, and false when there is a problem.
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

// checkVersion adds the problem of a version that was not sent on field.
func checkVersion(version *int, field string, problems *errs.Problems) {
	if version == nil || *version < 1 {
		problems.Add(field, msgReload)
	}
}
