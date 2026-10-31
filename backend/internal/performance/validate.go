package performance

import (
	"strconv"
	"strings"
	"time"

	"mapp/internal/platform/money"
)

const maxReferenceLength = 200

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

// Validate reports every fault. Nothing here reads a clock or a database, so each rule is testable alone.
func Validate(row Row, today time.Time, currencies map[string]bool) []Fault {
	faults := referenceFaults(row.Reference)
	faults = append(faults, dateFaults(row.Date, today)...)
	return append(faults, readingFaults(row, currencies)...)
}

func referenceFaults(c Cell) []Fault {
	raw := strings.TrimSpace(c.Value)
	switch {
	case raw == "":
		return []Fault{{Column: c.Column, Code: faultMissingReference, Message: msgReferenceNeeded}}
	case len(raw) > maxReferenceLength:
		return []Fault{{Column: c.Column, Code: faultReferenceTooLong, Message: msgReferenceTooLong}}
	}
	return nil
}

func dateFaults(c Cell, today time.Time) []Fault {
	raw := strings.TrimSpace(c.Value)
	if raw == "" {
		return []Fault{{Column: c.Column, Code: faultEmptyColumn, Message: msgDateNeeded}}
	}
	on, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return []Fault{{Column: c.Column, Code: faultInvalidDate, Message: msgDateInvalid}}
	}
	if on.After(today) {
		return []Fault{{Column: c.Column, Code: faultFutureDate, Message: msgDateFuture}}
	}
	return nil
}

// readingFaults checks each measure column the row fills, and refuses a row that fills none: a line saying
// nothing is a mistake, not a set of zeroes.
func readingFaults(row Row, currencies map[string]bool) []Fault {
	filled := row.filled()
	if len(filled) == 0 {
		return []Fault{{Column: firstColumn(row.Readings), Code: faultNoMeasure, Message: msgMeasureNeeded}}
	}
	var faults []Fault
	for _, reading := range filled {
		if reading.money() {
			faults = append(faults, moneyFaults(reading.Cell, row.Currency, currencies)...)
			continue
		}
		faults = append(faults, countFaults(reading.Cell)...)
	}
	return faults
}

func firstColumn(readings []Reading) string {
	if len(readings) == 0 {
		return ""
	}
	return readings[0].Column
}

// countFaults reports negative and fractional separately, because "-1.5" is both (FR-011). The count is read
// as an integer, never a float, so a value the bigint column cannot hold is refused here rather than there.
func countFaults(c Cell) []Fault {
	whole, fraction, hasPoint := strings.Cut(strings.TrimSpace(c.Value), ".")
	value, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || (hasPoint && !digitsOnly(fraction)) {
		return []Fault{{Column: c.Column, Code: faultInvalidCount, Message: msgCountInvalid}}
	}
	var faults []Fault
	if value < 0 || strings.HasPrefix(whole, "-") {
		faults = append(faults, Fault{Column: c.Column, Code: faultNegativeCount, Message: msgCountNegative})
	}
	if hasPoint && strings.Trim(fraction, "0") != "" {
		faults = append(faults, Fault{Column: c.Column, Code: faultFractionalCount, Message: msgCountFractional})
	}
	return faults
}

func wholeCount(raw string) (int64, bool) {
	whole, _, _ := strings.Cut(strings.TrimSpace(raw), ".")
	value, err := strconv.ParseInt(whole, 10, 64)
	return value, err == nil
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	return strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' }) < 0
}

func moneyFaults(amount, currency Cell, known map[string]bool) []Fault {
	var faults []Fault
	if _, err := money.ParseAmount(strings.TrimSpace(amount.Value)); err != nil {
		faults = append(faults, Fault{Column: amount.Column, Code: faultInvalidAmount, Message: msgAmountInvalid})
	}
	code := strings.ToUpper(strings.TrimSpace(currency.Value))
	if code == "" || !known[code] {
		faults = append(faults, Fault{Column: currency.Column, Code: faultUnknownCurrency, Message: msgCurrencyUnknown})
	}
	return faults
}
