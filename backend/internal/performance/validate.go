package performance

import (
	"strconv"
	"strings"
	"time"
)

// Fault: a row carries every fault it has, not the first (FR-011); the review row's reason is the headline.
type Fault struct {
	Column  string `json:"column"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Cell struct {
	Column string
	Value  string
}

type RowInput struct {
	Reference string
	Date      Cell
	Stage     Cell
	Count     Cell
	Amount    Cell
	Currency  Cell
}

// Validate reports every fault; today and the known sets come from the caller, so the rules stay pure.
func Validate(in RowInput, today time.Time, stages, currencies map[string]bool) []Fault {
	var faults []Fault
	if strings.TrimSpace(in.Reference) == "" {
		faults = append(faults, Fault{Column: "reference", Code: faultMissingReference, Message: msgReferenceNeeded})
	}
	faults = append(faults, dateFaults(in.Date, today)...)
	faults = append(faults, countFaults(in.Count)...)
	faults = append(faults, moneyFaults(in.Amount, in.Currency, currencies)...)
	if in.Stage.Value != "" && !stages[strings.ToLower(in.Stage.Value)] {
		faults = append(faults, Fault{Column: in.Stage.Column, Code: faultUnknownStage, Message: msgStageUnknown})
	}
	return faults
}

func dateFaults(c Cell, today time.Time) []Fault {
	if strings.TrimSpace(c.Value) == "" {
		return []Fault{{Column: c.Column, Code: faultEmptyColumn, Message: msgDateNeeded}}
	}
	on, err := time.Parse(time.DateOnly, strings.TrimSpace(c.Value))
	if err != nil {
		return []Fault{{Column: c.Column, Code: faultInvalidDate, Message: msgDateInvalid}}
	}
	if on.After(today) {
		return []Fault{{Column: c.Column, Code: faultFutureDate, Message: msgDateFuture}}
	}
	return nil
}

// countFaults reports negative and fractional separately, because "-1.5" is both (FR-011).
func countFaults(c Cell) []Fault {
	raw := strings.TrimSpace(c.Value)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return []Fault{{Column: c.Column, Code: faultInvalidCount, Message: msgCountInvalid}}
	}
	var faults []Fault
	if value < 0 {
		faults = append(faults, Fault{Column: c.Column, Code: faultNegativeCount, Message: msgCountNegative})
	}
	if value != float64(int64(value)) {
		faults = append(faults, Fault{Column: c.Column, Code: faultFractionalCount, Message: msgCountFractional})
	}
	return faults
}

func moneyFaults(amount, currency Cell, known map[string]bool) []Fault {
	raw := strings.TrimSpace(amount.Value)
	if raw == "" {
		return nil
	}
	var faults []Fault
	if _, err := strconv.ParseFloat(raw, 64); err != nil {
		faults = append(faults, Fault{Column: amount.Column, Code: faultInvalidAmount, Message: msgAmountInvalid})
	}
	code := strings.ToUpper(strings.TrimSpace(currency.Value))
	if code == "" || !known[code] {
		faults = append(faults, Fault{Column: currency.Column, Code: faultUnknownCurrency, Message: msgCurrencyUnknown})
	}
	return faults
}
