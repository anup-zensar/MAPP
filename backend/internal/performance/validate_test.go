package performance_test

import (
	"testing"
	"time"

	"mapp/internal/performance"
)

var (
	today    = time.Date(2026, 3, 12, 0, 0, 0, 0, time.UTC)
	knownCcy = map[string]bool{"USD": true, "EUR": true}
)

type cells struct{ ref, date, inq, revenue, ccy string }

// loadRow builds a row of a source mapping one INQ column and one revenue column, the wide shape FR-006
// allows: the mapping names each measure and the cell carries only its figure.
func loadRow(c cells) performance.Row {
	return performance.Row{
		Reference: performance.Cell{Column: "reference", Value: c.ref},
		Date:      performance.Cell{Column: "date", Value: c.date},
		Currency:  performance.Cell{Column: "currency", Value: c.ccy},
		Readings: []performance.Reading{
			{Measure: performance.Measure{Stage: "inq"}, Cell: performance.Cell{Column: "inq", Value: c.inq}},
			{Measure: performance.Measure{Stage: "revenue"}, Cell: performance.Cell{Column: "revenue", Value: c.revenue}},
		},
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

	if got := performance.Validate(loadRow(cells{ref: "r1", date: "2026-02-01", inq: "5"}), today, knownCcy); len(got) != 0 {
		t.Errorf("a good row reported %+v, want none", got)
	}
}

func TestValidate_EveryFaultAtOnce_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(
		loadRow(cells{ref: "", date: "not-a-date", inq: "-1.5", revenue: "abc", ccy: ""}), today, knownCcy))

	for _, want := range []string{"missingReference", "invalidDate", "negativeCount", "fractionalCount",
		"invalidAmount", "unconfiguredCurrency"} {
		if !got[want] {
			t.Errorf("%s is not among the faults reported: %v", want, got)
		}
	}
}

func TestValidate_NegativeAndFractionalAreBothReported_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(loadRow(cells{ref: "r1", date: "2026-02-01", inq: "-1.5"}), today, knownCcy))

	if !got["negativeCount"] || !got["fractionalCount"] {
		t.Errorf("-1.5 reported %v, want both negativeCount and fractionalCount", got)
	}
}

func TestValidate_CountsThatAreNotWholeNumbers_FR011(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"a count beyond what the column holds": "9223372036854775808",
		"a count in exponent form":             "1e5",
		"a count that is not a number at all":  "NaN",
		"a hexadecimal count":                  "0x10",
		"a count with a stray suffix":          "12abc",
	}

	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := codesOf(performance.Validate(loadRow(cells{ref: "r1", date: "2026-02-01", inq: raw}), today, knownCcy)); !got["invalidCount"] {
				t.Errorf("Validate(%q) reported %v, want invalidCount", raw, got)
			}
		})
	}
}

func TestValidate_ATrailingZeroIsStillAWholeCount_FR011(t *testing.T) {
	t.Parallel()

	if got := performance.Validate(loadRow(cells{ref: "r1", date: "2026-02-01", inq: "7.00"}), today, knownCcy); len(got) != 0 {
		t.Errorf("7.00 reported %+v, want no fault: it is a whole number", got)
	}
}

func TestValidate_MoneyIsReadExactlyNotAsAFloat_FR014(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(
		loadRow(cells{ref: "r1", date: "2026-02-01", revenue: "1234.567", ccy: "USD"}), today, knownCcy))

	if !got["invalidAmount"] {
		t.Errorf("an amount with three decimals reported %v, want invalidAmount", got)
	}
}

func TestValidate_FutureDateRefused_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(loadRow(cells{ref: "r1", date: "2026-06-01", inq: "1"}), today, knownCcy))

	if !got["futureDate"] {
		t.Errorf("a date after today reported %v, want futureDate", got)
	}
}

func TestValidate_AmountWithoutAConfiguredCurrency_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(
		loadRow(cells{ref: "r1", date: "2026-02-01", revenue: "1234.50", ccy: "GBP"}), today, knownCcy))

	if !got["unconfiguredCurrency"] {
		t.Errorf("an amount in an unconfigured currency reported %v, want unconfiguredCurrency", got)
	}
}

func TestValidate_ARowFillingNoMeasureColumnIsRefused_FR011(t *testing.T) {
	t.Parallel()

	got := codesOf(performance.Validate(loadRow(cells{ref: "r1", date: "2026-02-01"}), today, knownCcy))

	if !got["noMeasure"] {
		t.Errorf("a row filling no measure column reported %v, want noMeasure", got)
	}
}

func TestValidate_AReferenceTooLongForItsColumn_FR011(t *testing.T) {
	t.Parallel()

	long := make([]byte, 201)
	for i := range long {
		long[i] = 'r'
	}
	got := codesOf(performance.Validate(loadRow(cells{ref: string(long), date: "2026-02-01", inq: "1"}), today, knownCcy))

	if !got["referenceTooLong"] {
		t.Errorf("a 201-character reference reported %v, want referenceTooLong", got)
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
