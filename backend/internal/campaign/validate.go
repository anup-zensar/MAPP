package campaign

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// The longest name and description, in characters (data model §1–§2).
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

// form is what the create and update forms of both levels send.
type form struct {
	Name, Description  string
	OwnerID            *string
	StartDate, EndDate *date.Date
	Budget             *money.Input
	Attributes         settings.AttributeValues
}

// checked is a form after its own checks: texts trimmed, an empty owner as none, the budget read. Values
// with a problem are left as sent; the problem stops the save anyway.
type checked struct {
	Name, Description  string
	OwnerID            *string
	StartDate, EndDate *date.Date
	Budget             *money.Money
}

// checkForm checks the fields that need nothing but the form (FR-005), adding every problem to problems:
// the name (and, through nameTaken, its uniqueness under the parent), the description, both dates or
// neither with the start on or before the end, and the budget (in the plan currency, at most 2 decimals, not
// negative — never compared with what remains, D-015). nameTaken returns the "already has" message for a
// valid name, or "".
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

	switch {
	case c.StartDate != nil && c.EndDate == nil:
		problems.Add("endDate", enterEndDateMessage)
	case c.StartDate == nil && c.EndDate != nil:
		problems.Add("startDate", enterStartDateMessage)
	case c.StartDate != nil && c.EndDate.Before(*c.StartDate):
		problems.Add("endDate", fmt.Sprintf(endBeforeStartMessage, date.Format(*c.StartDate)))
	}

	c.Budget = checkBudget(problems, f.Budget, currency)
	return c
}

// checkName checks a name of a form or a copy (FR-005, FR-026), adding its problem to problems: required, at
// most 120 characters, and — through nameTaken, which returns the "already has" message for a valid name, or
// "" — unique under the parent. It returns the name trimmed.
func checkName(problems *errs.Problems, name string, nameTaken func(name string) string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(name) > maxNameLength:
		problems.Add("name", nameTooLongMessage)
	default:
		if message := nameTaken(name); message != "" {
			problems.Add("name", message)
		}
	}
	return name
}

// checkBudget reads a budget: nil for none, otherwise in the plan currency (016 FR-011) with at most two
// decimals and not negative.
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

// campaignNamed returns the campaign among campaigns, other than exceptID, whose name is name ignoring
// letter case, and false when there is none. Names are stored trimmed, so surrounding spaces are ignored
// too.
func campaignNamed(campaigns []Campaign, name, exceptID string) (Campaign, bool) {
	for _, c := range campaigns {
		if c.ID != exceptID && strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return Campaign{}, false
}

// initiativeNamed is campaignNamed for initiatives.
func initiativeNamed(initiatives []Initiative, name, exceptID string) (Initiative, bool) {
	for _, i := range initiatives {
		if i.ID != exceptID && strings.EqualFold(i.Name, name) {
			return i, true
		}
	}
	return Initiative{}, false
}

// checkInsideCampaign applies R3 (FR-006): when the campaign has both dates and the initiative has both,
// the initiative's dates lie inside the campaign's. The message names the allowed range.
func checkInsideCampaign(problems *errs.Problems, campaign Campaign, start, end *date.Date) {
	if campaign.StartDate == nil || campaign.EndDate == nil || start == nil || end == nil {
		return // a campaign without dates does not limit its initiatives (spec edge case)
	}
	if start.Before(*campaign.StartDate) || end.After(*campaign.EndDate) {
		problems.Add("startDate", fmt.Sprintf(outsideCampaignMessage, date.FormatRangeFull(*campaign.StartDate, *campaign.EndDate)))
	}
}

// dated is a record below whose dates a new date range must contain (FR-007): an initiative, or an
// activity, where an ongoing one has only its start date checked.
type dated struct {
	name       string
	start, end *date.Date
	ongoing    bool
}

// checkChildrenInside applies FR-007: every child with dates must lie inside start – end (an ongoing child:
// its start only); undated children are ignored. One problem on startDate names every child outside, with
// its dates; plural is "initiatives" or "activities".
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

// datesChanged reports whether a save changes a record's start or end date (D-095: the date rules run only
// then).
func datesChanged(oldStart, oldEnd, newStart, newEnd *date.Date) bool {
	return !sameDate(oldStart, newStart) || !sameDate(oldEnd, newEnd)
}

// sameDate reports whether two optional dates are equal: both none, or the same day.
func sameDate(a, b *date.Date) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// checkVersionSent adds the version problem when an update sends no version, or one below 1: the version
// check cannot work then (FR-018).
func checkVersionSent(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

// campaignNameTaken returns the "already has" message when another campaign of plan than exceptID has name,
// ignoring letter case (FR-005); "" otherwise.
func campaignNameTaken(siblings []Campaign, plan PlanFacts, name, exceptID string) string {
	if other, taken := campaignNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(campaignNameTakenMessage, plan.Name, other.Name)
	}
	return ""
}

// initiativeNameTaken returns the "already has" message when another initiative of campaign than exceptID
// has name, ignoring letter case (FR-005); "" otherwise.
func initiativeNameTaken(siblings []Initiative, campaign Campaign, name, exceptID string) string {
	if other, taken := initiativeNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(initiativeNameTakenMessage, campaign.Name, other.Name)
	}
	return ""
}
