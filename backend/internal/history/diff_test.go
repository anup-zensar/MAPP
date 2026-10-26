package history

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/record"
)

// Changes are listed by field name, whatever the order of the record's own fields, so the log always reads
// the same (FR-022).
func TestDiff_SortedByField_FR022(t *testing.T) {
	t.Parallel()
	type activity struct { // the fields are deliberately not in alphabetical order
		Title  string `json:"title"`
		Budget string `json:"budget"`
		Owner  string `json:"owner"`
	}
	before := activity{Title: "Launch", Budget: "100", Owner: "U-1002"}
	after := activity{Title: "Launch film", Budget: "250", Owner: "U-1003"}

	got, err := Diff(before, after)

	want := []Change{
		{Field: "budget", Old: raw(`"100"`), New: raw(`"250"`)},
		{Field: "owner", Old: raw(`"U-1002"`), New: raw(`"U-1003"`)},
		{Field: "title", Old: raw(`"Launch"`), New: raw(`"Launch film"`)},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %s, %v\nwant %s", changesText(got), err, changesText(want))
	}
}

// The bookkeeping fields change with every save, so they are never listed as changed fields: a save that
// changed only them changed nothing (data model "Bookkeeping").
func TestDiff_IgnoresMeta_FR022(t *testing.T) {
	t.Parallel()
	type folder struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		record.Meta
	}
	before := folder{ID: "SF-1001", Name: "Ideas", Meta: record.NewMeta(priya, start)}
	touched := before
	touched.Touch(daniel, start.Add(time.Hour))

	got, err := Diff(before, touched)
	if err != nil || got == nil || len(got) != 0 {
		t.Errorf("only the bookkeeping changed: Diff = %s, %v; want an empty list", changesText(got), err)
	}

	renamed := touched
	renamed.Name = "Ideas 2"
	got, err = Diff(before, renamed)

	want := []Change{{Field: "name", Old: raw(`"Ideas"`), New: raw(`"Ideas 2"`)}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("renamed: Diff = %s, %v\nwant %s", changesText(got), err, changesText(want))
	}
}

// null, a missing field, "", [] and {} are all the same empty value: going from one of them to another is
// not a change. Filling in an empty field is, and false is a value, not an empty one (FR-022).
func TestDiff_NullAndEmptyAreEqual_FR022(t *testing.T) {
	t.Parallel()
	empty := map[string]any{"a": nil, "b": "", "c": []string{}, "d": map[string]string{}}
	alsoEmpty := map[string]any{"a": "", "b": []string{}, "c": map[string]string{}} // and "d" is missing

	for _, pair := range [][2]map[string]any{{empty, alsoEmpty}, {alsoEmpty, empty}} {
		if got, err := Diff(pair[0], pair[1]); err != nil || len(got) != 0 {
			t.Errorf("Diff(%v, %v) = %s, %v; want no change", pair[0], pair[1], changesText(got), err)
		}
	}

	got, err := Diff(empty, map[string]any{"b": "Filled in", "e": false})

	want := []Change{
		{Field: "b", Old: raw(`""`), New: raw(`"Filled in"`)},
		{Field: "e", Old: raw(`null`), New: raw(`false`)},
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %s, %v\nwant %s", changesText(got), err, changesText(want))
	}
}

// A nested object or list is compared as one value: a change inside it lists the whole old and new value.
// The order of a list matters (FR-022).
func TestDiff_NestedValuesAsAWhole_FR022(t *testing.T) {
	t.Parallel()
	before := map[string]any{
		"budget": map[string]string{"amount": "12500.00", "currency": "USD"},
		"tags":   []string{"Digital", "Print"},
	}
	after := map[string]any{
		"budget": map[string]string{"amount": "12500.00", "currency": "EUR"},
		"tags":   []string{"Digital", "Print"},
	}

	got, err := Diff(before, after)

	want := []Change{{
		Field: "budget",
		Old:   raw(`{"amount":"12500.00","currency":"USD"}`),
		New:   raw(`{"amount":"12500.00","currency":"EUR"}`),
	}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Diff = %s, %v\nwant %s", changesText(got), err, changesText(want))
	}

	reordered := map[string]any{"budget": before["budget"], "tags": []string{"Print", "Digital"}}
	got, err = Diff(before, reordered)

	want = []Change{{Field: "tags", Old: raw(`["Digital","Print"]`), New: raw(`["Print","Digital"]`)}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("reordered list: Diff = %s, %v\nwant %s", changesText(got), err, changesText(want))
	}
}

// Diff compares records, which are JSON objects; anything else is a mistake in the calling code.
func TestDiff_RefusesValuesThatAreNotRecords(t *testing.T) {
	t.Parallel()
	if got, err := Diff("Ideas", "Ideas 2"); err == nil {
		t.Errorf("Diff of two strings = %s, want an error", changesText(got))
	}
}

// Without a label from the kind, a field's JSON name is written in words (data model §5).
func TestFieldLabel_FieldNameInWords_FR022(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"name":          "Name",
		"plannedCost":   "Planned cost",
		"sysGenBalance": "Sys gen balance",
		"imageURL":      "Image url",
		"q1Budget":      "Q1 budget",
		"":              "",
	}
	for field, want := range tests {
		if got := fieldLabel(field); got != want {
			t.Errorf("fieldLabel(%q) = %q, want %q", field, got, want)
		}
	}
}

// Without a text from the kind, a value is written as people read it: text as it is, Yes or No, numbers as
// written, lists joined with commas, and "Not filled in" for every empty value (data model §5). An object
// has no default wording, so it is shown as its JSON.
func TestDefaultText_FR022(t *testing.T) {
	t.Parallel()
	tests := []struct {
		stored, want string
	}{
		{`"Launch film"`, "Launch film"},
		{`true`, "Yes"},
		{`false`, "No"},
		{`12500.50`, "12500.50"},
		{`3`, "3"},
		{`["Digital","Print"]`, "Digital, Print"},
		{`[1,2.50]`, "1, 2.50"},
		{`null`, "Not filled in"},
		{``, "Not filled in"}, // a value that is missing
		{`""`, "Not filled in"},
		{`[]`, "Not filled in"},
		{`{}`, "Not filled in"},
		{`{"amount":"1.00","currency":"USD"}`, `{"amount":"1.00","currency":"USD"}`},
	}
	for _, tt := range tests {
		value, err := decode(raw(tt.stored))
		if err != nil {
			t.Fatalf("decode(%s): %v", tt.stored, err)
		}

		if got := defaultText(value); got != tt.want {
			t.Errorf("defaultText(%s) = %q, want %q", tt.stored, got, tt.want)
		}
	}
}

// changesText writes changes one per line, so that a failure shows the stored values as text.
func changesText(changes []Change) string {
	var text strings.Builder
	for _, c := range changes {
		fmt.Fprintf(&text, "\n  %s %q: %s → %s (%q → %q)", c.Field, c.Label, c.Old, c.New, c.OldText, c.NewText)
	}
	if text.Len() == 0 {
		return "[]"
	}
	return text.String()
}
