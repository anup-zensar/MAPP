package performance_test

import (
	"testing"
	"time"

	"mapp/internal/performance"
)

var (
	today      = time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	knownStage = map[string]bool{"inq": true, "mql": true}
	knownCcy   = map[string]bool{"USD": true, "EUR": true}
)

type cells struct{ ref, date, stage, count, amount, ccy string }

func row(c cells) performance.RowInput {
	return performance.RowInput{
		Reference: c.ref,
		Date:      performance.Cell{Column: "date", Value: c.date},
		Stage:     performance.Cell{Column: "stage", Value: c.stage},
		Count:     performance.Cell{Column: "count", Value: c.count},
		Amount:    performance.Cell{Column: "amount", Value: c.amount},
		Currency:  performance.Cell{Column: "currency", Value: c.ccy},
	}
}

func codesOf(faults []performance.Fault) map[string]bool {
	out := map[string]bool{}
	for _, f := range faults {
		out[f.Code] = true
	}
	return out
}

func TestValidate_AGoodRowHasNoFaults_FR011(t *testing.T) {
	t.Parallel()

	if got := performance.Validate(row(cells{ref: "r1", date: "2026-02-01", stage: "inq", count: "5", amount: "", ccy: ""}), today, knownStage, knownCcy); len(got) != 0 {
		t.Errorf("a good row reported %+v, want none", got)
	}
}

func TestValidate_EveryFaultAtOnce_FR011(t *testing.T) {
	t.Parallel()

	// No reference, an unreadable date, a count that is both negative and fractional, an unreadable amount
	// with no currency, and a stage the model does not define.
	got := codesOf(performance.Validate(row(cells{ref: "", date: "not-a-date", stage: "deals", count: "-1.5", amount: "abc", ccy: ""}), today, knownStage, knownCcy))

	for _, want := range []string{"missingReference", "invalidDate", "negativeCount", "fractionalCount",
		"invalidAmount", "unconfiguredCurrency", "unknownStage"} {
		if !got[want] {
			t.Errorf("%s is not among the faults reported: %v", want, got)
		}
	}
}

func TestValidate_NegativeAndFractionalAreBothReported_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(row(cells{ref: "r1", date: "2026-02-01", stage: "inq", count: "-1.5", amount: "", ccy: ""}), today, knownStage, knownCcy))

	if !got["negativeCount"] || !got["fractionalCount"] {
		t.Errorf("-1.5 reported %v, want both negativeCount and fractionalCount", got)
	}
}

func TestValidate_FutureDateRefused_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(row(cells{ref: "r1", date: "2026-06-01", stage: "inq", count: "1", amount: "", ccy: ""}), today, knownStage, knownCcy))

	if !got["futureDate"] {
		t.Errorf("a date after today reported %v, want futureDate", got)
	}
}

func TestValidate_AmountWithoutAConfiguredCurrency_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(row(cells{ref: "r1", date: "2026-02-01", stage: "inq", count: "", amount: "1234.50", ccy: "GBP"}), today, knownStage, knownCcy))

	if !got["unconfiguredCurrency"] {
		t.Errorf("an amount in an unconfigured currency reported %v, want unconfiguredCurrency", got)
	}
}

func TestPersonalDataIn_RefusesTheWholeFileNamingEveryColumn_FR012(t *testing.T) {
	t.Parallel()

	found := performance.PersonalDataIn([]string{"Reference", "Date", "Email", "Contact Phone", "INQ"})

	if len(found) != 2 {
		t.Fatalf("PersonalDataIn() found %v, want both Email and Contact Phone named", found)
	}
}

func TestPersonalDataIn_AnOrdinaryFilePasses_FR012(t *testing.T) {
	t.Parallel()

	if found := performance.PersonalDataIn([]string{"Reference", "Date", "Stage", "Count"}); len(found) != 0 {
		t.Errorf("an ordinary file was refused for %v", found)
	}
}
