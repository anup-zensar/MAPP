package campaign

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

const (
	maxNameLength        = 120
	maxDescriptionLength = 2000
)

// Messages of the plan's "API validation and error messages" table. <…> parts are filled in with fmt; dates
// are written with date.Format ("01 Mar 2026", D-099).
const (
	enterNameMessage           = "Enter a name."
	nameTooLongMessage         = "Use 120 characters or fewer."
	campaignNameTakenMessage   = "'%s' already has a campaign called '%s'."    // plan name, other campaign's name
	initiativeNameTakenMessage = "'%s' already has an initiative called '%s'." // campaign name, other initiative's name
	descriptionTooLongMessage  = "Use 2,000 characters or fewer."
	chooseActivePersonMessage  = "Choose an active person."
	invalidDateMessage         = "Enter a valid date."
	enterEndDateMessage        = "Enter an end date too, or clear the start date."
	enterStartDateMessage      = "Enter a start date too, or clear the end date."
	endBeforeStartMessage      = "End date must be on or after the start date (%s)."
	outsideCampaignMessage     = "Dates must be inside the campaign's dates (%s)."
	childOutsideMessage        = "%s would be outside these dates. Change its dates first, or choose dates that include it."
	childrenOutsideMessage     = "These %s would be outside these dates: %s. Change their dates first, or choose dates that include them."
	negativeBudgetMessage      = "Budget can't be negative."
	budgetAmountMessage        = "Enter an amount in %s with at most 2 decimals, such as 12500 or 12500.50."
	budgetCurrencyMessage      = "Enter the amount in %s, the plan's currency." // 016 FR-011
	reloadMessage              = "Reload the page and try again."
	campaignGoneMessage        = "This campaign no longer exists — someone may have deleted it."
	initiativeGoneMessage      = "This initiative no longer exists — someone may have deleted it."
	planGoneMessage            = "This plan no longer exists — someone may have deleted it."
)

type form struct {
	Name, Description  string
	OwnerID            *string
	StartDate, EndDate *date.Date
	Budget             *money.Input
	Attributes         settings.AttributeValues
}

type checked struct {
	Name, Description  string
	OwnerID            *string
	StartDate, EndDate *date.Date
	Budget             *money.Money
}

func checkForm(problems *errs.Problems, f form, currency string, nameTaken func(name string) string) checked {
	c := checked{
		Description: strings.TrimSpace(f.Description),
		OwnerID:     f.OwnerID,
		StartDate:   f.StartDate,
		EndDate:     f.EndDate,
	}
	if c.OwnerID != nil && *c.OwnerID == "" {
		c.OwnerID = nil // "" clears the owner, as null does
	}

	c.Name = checkName(problems, f.Name, nameTaken)
	if utf8.RuneCountInString(c.Description) > maxDescriptionLength {
		problems.Add("description", descriptionTooLongMessage)
	}

	c.StartDate, c.EndDate = checkDates(problems, c.StartDate, c.EndDate)
	c.Budget = checkBudget(problems, f.Budget, currency)
	return c
}

func checkDates(problems *errs.Problems, start, end *date.Date) (*date.Date, *date.Date) {
	startValid, endValid := validDate(problems, "startDate", start), validDate(problems, "endDate", end)
	if !startValid || !endValid {
		return nil, nil
	}
	switch {
	case start != nil && end == nil:
		problems.Add("endDate", enterEndDateMessage)
	case start == nil && end != nil:
		problems.Add("startDate", enterStartDateMessage)
	case start != nil && end.Before(*start):
		problems.Add("endDate", fmt.Sprintf(endBeforeStartMessage, date.Format(*start)))
	}
	return start, end
}

func validDate(problems *errs.Problems, field string, d *date.Date) bool {
	if d != nil && d.IsZero() {
		problems.Add(field, invalidDateMessage)
		return false
	}
	return true
}

func checkName(problems *errs.Problems, name string, nameTaken func(name string) string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	case record.CheckSingleLine(name) != "":
		problems.Add("name", record.CheckSingleLine(name))
	default:
		if message := nameTaken(name); message != "" {
			problems.Add("name", message)
		}
	}
	return name
}

func checkBudget(problems *errs.Problems, in *money.Input, currency string) *money.Money {
	if in == nil {
		return nil
	}
	budget, err := money.ParseInput(*in)
	switch {
	case err != nil:
		problems.Add("budget", fmt.Sprintf(budgetAmountMessage, currency))
		return nil
	case budget.Currency != currency:
		problems.Add("budget", fmt.Sprintf(budgetCurrencyMessage, currency))
		return nil
	}
	if budget.Amount.Sign() < 0 {
		problems.Add("budget", negativeBudgetMessage)
		return nil
	}
	return &budget
}

func campaignNamed(campaigns []Campaign, name, exceptID string) (Campaign, bool) {
	for _, c := range campaigns {
		if c.ID != exceptID && strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return Campaign{}, false
}

func initiativeNamed(initiatives []Initiative, name, exceptID string) (Initiative, bool) {
	for _, i := range initiatives {
		if i.ID != exceptID && strings.EqualFold(i.Name, name) {
			return i, true
		}
	}
	return Initiative{}, false
}

func checkInsideCampaign(problems *errs.Problems, campaign Campaign, start, end *date.Date) {
	if campaign.StartDate == nil || campaign.EndDate == nil || start == nil || end == nil {
		return // a campaign without dates does not limit its initiatives (spec edge case)
	}
	if start.Before(*campaign.StartDate) || end.After(*campaign.EndDate) {
		problems.Add("startDate", fmt.Sprintf(outsideCampaignMessage, date.FormatRangeFull(*campaign.StartDate, *campaign.EndDate)))
	}
}

type dated struct {
	name       string
	start, end *date.Date
	ongoing    bool
}

func checkChildrenInside(problems *errs.Problems, plural string, children []dated, start, end date.Date) {
	var outside []string
	for _, child := range children {
		if child.start == nil {
			continue
		}
		var out bool
		var dates string
		if child.ongoing || child.end == nil {
			out = child.start.Before(start) || child.start.After(end)
			dates = "ongoing from " + date.Format(*child.start)
		} else {
			out = child.start.Before(start) || child.end.After(end)
			dates = date.FormatRange(*child.start, *child.end)
		}
		if out {
			outside = append(outside, fmt.Sprintf("%s (%s)", child.name, dates))
		}
	}
	switch len(outside) {
	case 0:
	case 1:
		problems.Add("startDate", fmt.Sprintf(childOutsideMessage, outside[0]))
	default:
		problems.Add("startDate", fmt.Sprintf(childrenOutsideMessage, plural, strings.Join(outside, ", ")))
	}
}

func datesChanged(oldStart, oldEnd, newStart, newEnd *date.Date) bool {
	return !sameDate(oldStart, newStart) || !sameDate(oldEnd, newEnd)
}

func sameDate(a, b *date.Date) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func checkVersionSent(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

func checkVersionIfSent(m record.Meta, version *int, noun string) error {
	if version == nil || *version < 1 {
		return nil
	}
	return m.CheckVersion(*version, noun)
}

func campaignNameTaken(siblings []Campaign, plan PlanFacts, name, exceptID string) string {
	if other, taken := campaignNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(campaignNameTakenMessage, plan.Name, other.Name)
	}
	return ""
}

func initiativeNameTaken(siblings []Initiative, campaign Campaign, name, exceptID string) string {
	if other, taken := initiativeNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(initiativeNameTakenMessage, campaign.Name, other.Name)
	}
	return ""
}
