package workbook

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var testChoices = Choices{
	People:        []Person{{ID: "U-1", Email: "priya.raman@shell.com", Active: true}, {ID: "U-2", Email: "gone@shell.com"}},
	ActivityTypes: []Option{{ID: "AT-1", Label: "Event", Active: true}, {ID: "AT-2", Label: "Print", Active: false}},
	ModelName:     "Lubricants 1.0",
	Currencies:    []Option{{ID: "EUR", Label: "EUR", Active: true}},
}

func TestParse_CellFormats_012D138(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, field, text, want, problem string
	}{
		{name: "an ISO date", field: fieldStart, text: "2026-03-12", want: "2026-03-12"},
		{name: "a written date", field: fieldStart, text: "12 Mar 2026", want: "2026-03-12"},
		{name: "a date that does not exist", field: fieldStart, text: "31 Feb 2026", problem: msgNotADate},
		{name: "an amount with a thousands separator", field: fieldPlanned, text: "12,500.5", want: "12500.50"},
		{name: "a negative amount", field: fieldPlanned, text: "-5", problem: msgNegative},
		{name: "more than two decimals", field: fieldPlanned, text: "1.234", problem: msgNotAnAmount},
		{name: "yes as y", field: fieldOngoing, text: "Y", want: "Yes"},
		{name: "neither yes nor no", field: fieldOngoing, text: "maybe", problem: msgYesNo},
		{name: "a person by e-mail in capitals", field: fieldOwner, text: "Priya.Raman@shell.com", want: "priya.raman@shell.com"},
		{name: "an inactive person", field: fieldOwner, text: "gone@shell.com", problem: msgNoPerson},
		{name: "a type by name", field: fieldType, text: "event", want: "Event"},
		{name: "a type no longer active", field: fieldType, text: "Print", problem: "'Print' is no longer allowed for Activity type"},
		{name: "a type the model does not allow", field: fieldType, text: "Teleport", problem: "'Teleport' is not an activity type the model 'Lubricants 1.0' allows"},
		{name: "a currency in lower case", field: fieldCurrency, text: "eur", want: "EUR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := column{heading: map[string]string{fieldType: "Activity type"}[tt.field], field: tt.field}
			v := Values{Attributes: map[string]json.RawMessage{}}

			problem := parse(c, tt.text, &v, testChoices)

			if problem != tt.problem {
				t.Fatalf("parse(%q) problem = %q, want %q", tt.text, problem, tt.problem)
			}
			if got := format(c, v, testChoices); tt.problem == "" && got != tt.want {
				t.Errorf("parse(%q) then format = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestParseAttribute_ChoicesAndSplits_012R10(t *testing.T) {
	t.Parallel()
	options := []Option{{ID: "SO-1", Label: "China", Active: true}, {ID: "SO-2", Label: "Hong Kong", Active: true},
		{ID: "SO-3", Label: "Macau", Active: false}}
	tests := []struct {
		name, typ, text, want, problem, back string
	}{
		{name: "a single choice", typ: typeSingle, text: "hong kong", want: `"SO-2"`},
		{name: "an inactive option", typ: typeSingle, text: "Macau", problem: "'Macau' is no longer allowed for Country"},
		{name: "several choices", typ: typeMultiple, text: "China; Hong Kong", want: `["SO-1","SO-2"]`},
		{name: "an unknown choice", typ: typeMultiple, text: "China; Mars", problem: "Not an option of Country"},
		{name: "a split with shares", typ: typeSplit, text: "China 60; Hong Kong 40",
			want: `[{"optionId":"SO-1","share":"60.00"},{"optionId":"SO-2","share":"40.00"}]`, back: "China 60; Hong Kong 40"},
		{name: "a split without shares is even", typ: typeSplit, text: "China; Hong Kong",
			want: `[{"optionId":"SO-1","share":"50.00"},{"optionId":"SO-2","share":"50.00"}]`, back: "China 50; Hong Kong 50"},
		{name: "a split with some shares", typ: typeSplit, text: "China 60; Hong Kong", problem: msgSharesAll},
		{name: "a share above 100", typ: typeSplit, text: "China 160", problem: msgShares},
		{name: "a whole number", typ: typeNumber, text: "42", want: `42`},
		{name: "not a whole number", typ: typeNumber, text: "4.2", problem: msgWholeNumber},
		{name: "yes", typ: typeYesNo, text: "true", want: `true`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := Attribute{ID: "SA-1", Label: "Country", Type: tt.typ, Options: options}

			raw, problem := parseAttribute(a, tt.text)

			if diff := cmp.Diff([]string{tt.want, tt.problem}, []string{string(raw), problem}); diff != "" {
				t.Errorf("parseAttribute(%q) (-want +got):\n%s", tt.text, diff)
			}
			if tt.back != "" && formatAttribute(a, raw) != tt.back {
				t.Errorf("formatAttribute = %q, want %q", formatAttribute(a, raw), tt.back)
			}
		})
	}
}
