package settings

import (
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/history"
)

// A planning record's changes are logged field by field, its Shell attribute values attribute by
// attribute, sorted by field (FR-006, D-048).
func TestDiff_AttributeByAttribute_FR006(t *testing.T) {
	t.Parallel()
	type planning struct {
		Name       string          `json:"name"`
		Attributes AttributeValues `json:"attributes"`
	}
	before := planning{Name: "Launch", Attributes: values("SA-1003", `["SO-1"]`, "SA-1010", `"x"`, "SA-1020", `[]`)}
	after := planning{Name: "Launch film", Attributes: values("SA-1003", `["SO-1", "SO-2"]`, "SA-1005", `true`, "SA-1020", `null`)}

	changes, err := Diff(before, after)

	want := []history.Change{
		{Field: "attributes.SA-1003", Old: json.RawMessage(`["SO-1"]`), New: json.RawMessage(`["SO-1","SO-2"]`)},
		{Field: "attributes.SA-1005", Old: json.RawMessage(`null`), New: json.RawMessage(`true`)},
		{Field: "attributes.SA-1010", Old: json.RawMessage(`"x"`), New: json.RawMessage(`null`)},
		{Field: "name", Old: json.RawMessage(`"Launch"`), New: json.RawMessage(`"Launch film"`)},
	}
	if err != nil || !reflect.DeepEqual(changes, want) {
		t.Errorf("Diff = %+v, %v\nwant %+v", changes, err, want)
	}
}

// The activity log describes a Shell attribute value with the attribute's label and the value's words
// (FR-006, data model §12).
func TestDescribeAttribute_Texts_FR006(t *testing.T) {
	t.Parallel()
	f, tf := newFormFixture(t)
	id := func(label string) string { return "attributes." + tf.attribute(LevelActivity, label).ID }
	china, india := tf.option(LevelActivity, "Country / OU", "China"), tf.option(LevelActivity, "Country / OU", "India")
	tests := []struct {
		field string
		value any
		label string
		text  string
	}{
		{id("Cost centre"), tf.option(LevelActivity, "Cost centre", "CN04000220"), "Cost centre", "CN04000220"},
		{"attributes." + tf.attribute(LevelCampaign, "Cluster").ID,
			[]any{tf.option(LevelCampaign, "Cluster", "Retail – China"), tf.option(LevelCampaign, "Cluster", "Retail – India")},
			"Cluster", "Retail – China, Retail – India"},
		{id("Country / OU"), []any{map[string]any{"optionId": china, "share": "60.00"}, map[string]any{"optionId": india, "share": "40.00"}},
			"Country / OU", "China 60.00 %, India 40.00 %"},
		{id("PIR flag"), true, "PIR flag", "Yes"},
		{id("PIR flag"), false, "PIR flag", "No"},
		{id("FIT4 ID"), "FIT4-0042", "FIT4 ID", "FIT4-0042"},
		{id("Budget size"), json.Number("1200"), "Budget size", "1200"},
		{id("FIT4 ID"), nil, "FIT4 ID", ""},
		{"name", "Launch", "", ""},
	}
	for _, tt := range tests {
		label, text, err := f.service.DescribeAttribute(t.Context(), tt.field, tt.value)
		if err != nil || label != tt.label || text != tt.text {
			t.Errorf("DescribeAttribute(%s, %v) = %q, %q, %v; want %q, %q", tt.field, tt.value, label, text, err, tt.label, tt.text)
		}
	}
}
