package lineitems

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The longest name and description, in characters.
const (
	maxNameLength        = 120
	maxDescriptionLength = 2000
)

// maxAmount is the largest amount a line item takes: 999,999,999,999.99 (plan, "Validation messages").
var maxAmount = money.FromCents(99_999_999_999_999)

// Messages of the plan's "Validation messages" table.
const (
	enterNameMessage          = "Enter a name for the line item."
	nameTooLongMessage        = "Use 120 characters or fewer."
	nameTakenMessage          = "'%s' already has a line item called '%s'."
	choosePersonMessage       = "Choose a person from the list."
	inactivePersonMessage     = "Choose an active person — %s is inactive."
	descriptionTooLongMessage = "Use 2,000 characters or fewer."
	chooseCurrencyMessage     = "Choose a currency."
	inactiveCurrencyMessage   = "Choose an active currency — %s is no longer active."
	enterCostMessage          = "Enter the planned cost."
	enterAmountMessage        = "Enter an amount, such as 12500.00."
	twoDecimalsMessage        = "Enter an amount with at most two decimals."
	belowTrillionMessage      = "Enter an amount below one trillion."
	negativeCostMessage       = "Planned cost can't be negative."
	negativeContractedMessage = "Contracted amount can't be negative."
	negativeMonthMessage      = "Month amounts can't be negative."
	monthRangeMessage         = "Choose a month from Jan %d to Dec %d."
	monthTwiceMessage         = "Each month can appear only once."
	chooseSpreadMessage       = "Choose how to spread the planned cost over time."
	budgetYearMessage         = "Choose %d or a year with finance budget items."
	yearFirstMessage          = "Choose the finance budget year first."
	chooseItemMessage         = "Choose a finance budget item from the list."
	itemOfYearMessage         = "Choose a finance budget item of %d."
	inactiveItemMessage       = "Choose an active finance budget item — '%s' is no longer active."
)

// spreadMethods are the four spread methods (design system §8).
var spreadMethods = []SpreadMethod{SpreadEvenly, SpreadFirstMonth, SpreadLastMonth, SpreadCustom}

// input returns the form fields of an update, which are those of a create.
func (u Update) input() Input {
	return Input{
		Name: u.Name, OwnerID: u.OwnerID, Description: u.Description, Currency: u.Currency,
		PlannedCost: u.PlannedCost, Spread: u.Spread, Attributes: u.Attributes, ContractedAmount: u.ContractedAmount,
		RequestSwbs: u.RequestSwbs, FinanceBudgetYear: u.FinanceBudgetYear, FinanceBudgetItemID: u.FinanceBudgetItemID,
		RestartApproval: u.RestartApproval,
	}
}

// check checks the form in of a line item of the activity f, inside the caller's transaction, and returns
// the fields to store with every problem at once (FR-009, SC-005). siblings are the activity's other line
// items (for the unique name); before is the stored line item of an update, nil for a new one: an inactive
// owner, currency, finance budget year or item it already has may stay (FR-011, FR-016), a new choice must
// be active. Only name, currency and planned cost are required (D-009).
func (s *Service) check(ctx context.Context, f ActivityFacts, siblings []LineItem, in Input, before *LineItem) (LineItem, errs.Problems, error) {
	var problems errs.Problems
	var li LineItem

	li.Name = strings.TrimSpace(in.Name)
	switch {
	case li.Name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(li.Name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	default:
		if other, taken := named(siblings, li.Name); taken {
			problems.Add("name", fmt.Sprintf(nameTakenMessage, f.Name, other.Name))
		}
	}

	if in.OwnerID != nil {
		li.OwnerID = strings.TrimSpace(*in.OwnerID)
	}
	if li.OwnerID != "" {
		person, found, err := s.deps.People.Person(ctx, li.OwnerID)
		switch {
		case err != nil:
			return LineItem{}, nil, err
		case !found:
			problems.Add("ownerId", choosePersonMessage)
		case !person.Active && (before == nil || before.OwnerID != li.OwnerID):
			problems.Add("ownerId", fmt.Sprintf(inactivePersonMessage, person.Name))
		}
	}

	li.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(li.Description) > maxDescriptionLength {
		problems.Add("description", descriptionTooLongMessage)
	}

	li.Currency = strings.TrimSpace(in.Currency)
	currency, found, err := s.deps.Currencies.Currency(ctx, li.Currency)
	switch {
	case err != nil:
		return LineItem{}, nil, err
	case li.Currency == "" || !found:
		problems.Add("currency", chooseCurrencyMessage)
	case !currency.Active && (before == nil || before.Currency != li.Currency):
		problems.Add("currency", fmt.Sprintf(inactiveCurrencyMessage, li.Currency))
	}

	if in.PlannedCost == nil || strings.TrimSpace(*in.PlannedCost) == "" {
		problems.Add("plannedCost", enterCostMessage)
	} else if amount, message := parseAmount(*in.PlannedCost, negativeCostMessage); message != "" {
		problems.Add("plannedCost", message)
	} else {
		li.PlannedCost = amount
	}

	if in.ContractedAmount != nil && strings.TrimSpace(*in.ContractedAmount) != "" {
		if amount, message := parseAmount(*in.ContractedAmount, negativeContractedMessage); message != "" {
			problems.Add("contractedAmount", message)
		} else {
			li.ContractedAmount = &amount
		}
	}

	li.Spread = checkSpread(&problems, in.Spread, f.PlanYear)

	var previous settings.AttributeValues
	if before != nil {
		previous = before.Attributes
	}
	attributes, err := s.deps.Attributes.Check(ctx, settings.AttributeInput{
		Values: orNoValues(in.Attributes), Previous: previous, BusinessUnit: f.BusinessUnit,
	})
	if err != nil {
		return LineItem{}, nil, err
	}
	problems = append(problems, attributes.Problems...)
	li.Attributes = orNoValues(attributes.Values)

	li.RequestSwbs = in.RequestSwbs
	if err := s.checkFinanceBudget(ctx, &problems, &li, in, f, before); err != nil {
		return LineItem{}, nil, err
	}
	return li, problems, nil
}

// checkSpread checks the spread of a request and returns it as stored: the custom months only for a custom
// spread, sorted by month, each a month from January of the plan's year − 1 to December of the plan's year
// + 10 at most once, with an amount of 0.00 or more (FR-018, D-152); an empty list for the other methods.
func checkSpread(problems *errs.Problems, in SpreadInput, planYear int) Spread {
	spread := Spread{Method: in.Method, Months: []MonthAmount{}}
	if !slices.Contains(spreadMethods, in.Method) {
		problems.Add("spread.method", chooseSpreadMessage)
		return spread
	}
	if in.Method != SpreadCustom {
		return spread
	}
	first, last := fmt.Sprintf("%04d-01", planYear-1), fmt.Sprintf("%04d-12", planYear+10)
	seen := map[string]bool{}
	for i, m := range in.Months {
		field := fmt.Sprintf("spread.months[%d]", i)
		month := strings.TrimSpace(m.Month)
		switch {
		case !monthText.MatchString(month) || month < first || month > last:
			problems.Add(field+".month", fmt.Sprintf(monthRangeMessage, planYear-1, planYear+10))
		case seen[month]:
			problems.Add(field+".month", monthTwiceMessage)
		}
		seen[month] = true
		amount, message := parseAmount(m.Amount, negativeMonthMessage)
		if message != "" {
			problems.Add(field+".amount", message)
		}
		spread.Months = append(spread.Months, MonthAmount{Month: month, Amount: amount})
	}
	slices.SortStableFunc(spread.Months, func(x, y MonthAmount) int { return strings.Compare(x.Month, y.Month) })
	return spread
}

// checkFinanceBudget checks the finance budget year and item of the form in and sets them on li (FR-015,
// FR-016, R12): the year is the plan's year, a year with active items, or the one already stored; the item
// needs a year first, must exist, belong to that year and, when newly chosen, be active.
func (s *Service) checkFinanceBudget(ctx context.Context, problems *errs.Problems, li *LineItem, in Input, f ActivityFacts, before *LineItem) error {
	itemID := ""
	if in.FinanceBudgetItemID != nil {
		itemID = strings.TrimSpace(*in.FinanceBudgetItemID)
	}
	if in.FinanceBudgetYear == nil && itemID == "" {
		return nil
	}
	items, err := s.deps.FinanceBudgets.Items(ctx)
	if err != nil {
		return err
	}

	if in.FinanceBudgetYear != nil {
		year := *in.FinanceBudgetYear
		kept := before != nil && before.FinanceBudgetYear != nil && *before.FinanceBudgetYear == year
		if year != f.PlanYear && !kept && !slices.Contains(yearsWithActiveItems(items), year) {
			problems.Add("financeBudgetYear", fmt.Sprintf(budgetYearMessage, f.PlanYear))
		}
		li.FinanceBudgetYear = &year
	}

	if itemID == "" {
		return nil
	}
	li.FinanceBudgetItemID = itemID
	i := slices.IndexFunc(items, func(item BudgetItem) bool { return item.ID == itemID })
	switch {
	case in.FinanceBudgetYear == nil:
		problems.Add("financeBudgetItemId", yearFirstMessage)
	case i < 0:
		problems.Add("financeBudgetItemId", chooseItemMessage)
	case items[i].Year != *in.FinanceBudgetYear:
		problems.Add("financeBudgetItemId", fmt.Sprintf(itemOfYearMessage, *in.FinanceBudgetYear))
	case !items[i].Active && (before == nil || before.FinanceBudgetItemID != itemID):
		problems.Add("financeBudgetItemId", fmt.Sprintf(inactiveItemMessage, items[i].Name))
	}
	return nil
}

// parseAmount reads an amount a person typed: a decimal number with at most two decimals (D-190), 0.00 or
// more, below one trillion. It returns the message of what is wrong, "" when the amount is fine;
// negativeMessage is the field's own message for a negative amount.
func parseAmount(text, negativeMessage string) (money.Amount, string) {
	amount, err := money.ParseAmount(strings.TrimSpace(text))
	switch {
	case errors.Is(err, money.ErrTooManyDecimals):
		return money.Amount{}, twoDecimalsMessage
	case errors.Is(err, money.ErrOutOfRange):
		return money.Amount{}, belowTrillionMessage
	case err != nil:
		return money.Amount{}, enterAmountMessage
	case amount.Sign() < 0:
		return money.Amount{}, negativeMessage
	case amount.Cmp(maxAmount) > 0:
		return money.Amount{}, belowTrillionMessage
	}
	return amount, ""
}

// named returns the line item among items whose name is name, ignoring letter case and surrounding spaces,
// and false when there is none.
func named(items []LineItem, name string) (LineItem, bool) {
	for _, li := range items {
		if strings.EqualFold(strings.TrimSpace(li.Name), strings.TrimSpace(name)) {
			return li, true
		}
	}
	return LineItem{}, false
}

// yearsWithActiveItems returns the years that have at least one active finance budget item, ascending.
func yearsWithActiveItems(items []BudgetItem) []int {
	years := []int{}
	for _, item := range items {
		if item.Active && !slices.Contains(years, item.Year) {
			years = append(years, item.Year)
		}
	}
	slices.Sort(years)
	return years
}

// orNoValues returns values, or empty values instead of nil: a stored line item never has nil attributes.
func orNoValues(values settings.AttributeValues) settings.AttributeValues {
	if values == nil {
		return settings.AttributeValues{}
	}
	return values
}
