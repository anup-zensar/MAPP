package workbook

import (
	"fmt"
	"strings"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

const (
	textYes = "Yes"
	textNo  = "No"
)

// The date formats a cell may hold besides the spreadsheet's own date cells, which read as YYYY-MM-DD (D-138).
var dateLayouts = []string{time.DateOnly, "2 Jan 2006", "2 January 2006", "02 Jan 2006"} //nolint:gochecknoglobals // fixed list, only read

// format writes a record's value of column c exactly as the import reads it back (FR-027).
func format(c column, v Values, ch Choices) string {
	if c.attr != nil {
		return formatAttribute(*c.attr, v.Attributes[c.attr.ID])
	}
	if text, isText := textField(c.field, &v); isText {
		return *text
	}
	if day, isDate := dateField(c.field, &v); isDate {
		return dateCell(*day)
	}
	if amount, isAmount := amountField(c.field, &v); isAmount {
		return amountCell(*amount)
	}
	switch c.field {
	case fieldOwner:
		return emailOf(ch.People, v.OwnerID)
	case fieldOngoing:
		return yesNo(v.Ongoing)
	case fieldType:
		return labelOf(ch.ActivityTypes, v.ActivityTypeID)
	case fieldWorkflow:
		return labelOf(ch.Workflows, v.WorkflowID)
	}
	return ""
}

func dateField(field string, v *Values) (**date.Date, bool) {
	switch field {
	case fieldStart:
		return &v.StartDate, true
	case fieldEnd:
		return &v.EndDate, true
	}
	return nil, false
}

func amountField(field string, v *Values) (**money.Amount, bool) {
	switch field {
	case fieldBudget:
		return &v.Budget, true
	case fieldPlanned:
		return &v.PlannedCost, true
	}
	return nil, false
}

// textField points at the field of v that is plain text: the name, the description, the external name or the
// currency.
func textField(field string, v *Values) (*string, bool) {
	switch field {
	case fieldName:
		return &v.Name, true
	case fieldDescription:
		return &v.Description, true
	case fieldExternal:
		return &v.ExternalName, true
	case fieldCurrency:
		return &v.Currency, true
	}
	return nil, false
}

// parse reads text into the field of column c; it answers the reason when the text is not a value of c.
func parse(c column, text string, v *Values, ch Choices) string {
	if c.attr != nil {
		return parseAttributeInto(*c.attr, text, v)
	}
	if field, isText := textField(c.field, v); isText && c.field != fieldCurrency {
		*field = text
		return ""
	}
	if day, isDate := dateField(c.field, v); isDate {
		return parseDate(text, day)
	}
	if amount, isAmount := amountField(c.field, v); isAmount {
		return parseAmount(text, amount)
	}
	switch c.field {
	case fieldOwner:
		return parsePerson(text, ch.People, &v.OwnerID)
	case fieldOngoing:
		return parseYesNo(text, &v.Ongoing)
	}
	return parseChoice(c, text, v, ch)
}

func parseAttributeInto(a Attribute, text string, v *Values) string {
	raw, problem := parseAttribute(a, text)
	if problem == "" {
		v.Attributes[a.ID] = raw
	}
	return problem
}

func parseChoice(c column, text string, v *Values, ch Choices) string {
	switch c.field {
	case fieldType:
		return parseOption(text, ch.ActivityTypes, &v.ActivityTypeID,
			optionCheck{heading: c.heading, unknown: fmt.Sprintf(msgTypeNotAllowed, text, ch.ModelName)})
	case fieldWorkflow:
		return parseOption(text, ch.Workflows, &v.WorkflowID, optionCheck{heading: c.heading, unknown: fmt.Sprintf(msgNotAnOption, c.heading)})
	case fieldCurrency:
		return parseOption(strings.ToUpper(text), ch.Currencies, &v.Currency, optionCheck{heading: c.heading, unknown: msgNotACurrency})
	}
	return ""
}

func dateCell(d *date.Date) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func amountCell(a *money.Amount) string {
	if a == nil {
		return ""
	}
	return a.String()
}

func yesNo(b bool) string {
	if b {
		return textYes
	}
	return textNo
}

func emailOf(people []Person, id string) string {
	for _, p := range people {
		if p.ID == id {
			return p.Email
		}
	}
	return id
}

func labelOf(options []Option, id string) string {
	for _, o := range options {
		if o.ID == id {
			return o.Label
		}
	}
	return id
}

func parseDate(text string, into **date.Date) string {
	*into = nil
	if text == "" {
		return ""
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, text); err == nil {
			d := date.New(t.Year(), t.Month(), t.Day())
			*into = &d
			return ""
		}
	}
	return msgNotADate
}

func parseAmount(text string, into **money.Amount) string {
	*into = nil
	if text == "" {
		return ""
	}
	if strings.HasPrefix(text, "-") {
		return msgNegative
	}
	a, err := money.ParseAmount(strings.ReplaceAll(text, ",", ""))
	if err != nil {
		return msgNotAnAmount
	}
	*into = &a
	return ""
}

func parseYesNo(text string, into *bool) string {
	switch strings.ToLower(text) {
	case "", "no", "n", "false", "0":
		*into = false
	case "yes", "y", "true", "1":
		*into = true
	default:
		return msgYesNo
	}
	return ""
}

func parsePerson(text string, people []Person, into *string) string {
	*into = ""
	if text == "" {
		return ""
	}
	for _, p := range people {
		if strings.EqualFold(p.Email, text) && p.Active {
			*into = p.ID
			return ""
		}
	}
	return msgNoPerson
}

// optionCheck names the column of a choice and the reason a text that is no option of it gets.
type optionCheck struct {
	heading, unknown string
}

// parseOption finds an option by label or ID, ignoring letter case.
func parseOption(text string, options []Option, into *string, check optionCheck) string {
	*into = ""
	if text == "" {
		return ""
	}
	for _, o := range options {
		if strings.EqualFold(o.Label, text) || o.ID == text {
			if !o.Active {
				return fmt.Sprintf(msgNoLongerAllowed, text, check.heading)
			}
			*into = o.ID
			return ""
		}
	}
	return check.unknown
}
