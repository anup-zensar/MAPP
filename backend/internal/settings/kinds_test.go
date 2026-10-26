package settings

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"mapp/internal/history"
)

// The activity log names every field of a Settings record in words and writes its values as people read
// them (FR-006, data model §12): percentages, money, rates, the names of linked records, status words and
// the words of the attribute enums.
func TestKinds_DescribeTexts_FR006(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)
	item := f.addItem(t, 2026, "2026 Lubricants Budget", "Europe – Digital", "", true)
	tf := f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, options: []string{"Mobility", "Lubricants"}},
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice, options: []string{"Europe"}},
	)
	kinds := map[string]history.Kind{}
	for _, k := range f.service.Kinds() {
		kinds[k.Name] = k
	}
	tests := []struct {
		kind, field, value string // value as stored JSON
		label, text        string
	}{
		{modelKind, "name", `"Lubricants 1.0"`, "Name", ""},
		{modelKind, "inqToMqlRate", `"2.00"`, "INQ→MQL rate", "2.00 %"},
		{modelKind, "mqlToDealRate", `"5.00"`, "MQL→deal rate", "5.00 %"},
		{modelKind, "averageDealSize", `{"amount":"10000.00","currency":"USD"}`, "Average deal size", "10,000.00 USD"},
		{modelKind, "activityTypeIds", `["` + digital.ID + `","` + event.ID + `"]`, "Allowed activity types", "Digital media, Event"},
		{modelKind, "active", `false`, "Status", "Inactive"},
		{modelKind, "active", `true`, "Status", "Active"},
		{activityTypeKind, "defaultPlannedCost", `{"amount":"25000.00","currency":"USD"}`, "Default planned cost", "25,000.00 USD"},
		{activityTypeKind, "defaultPlannedCost", `null`, "Default planned cost", ""},
		{attributeKind, "level", `"lineItem"`, "Level", "Line item"},
		{attributeKind, "type", `"multipleChoiceSplit"`, "Type", "Multiple choice with % split"},
		{attributeKind, "marking", `"neededToSubmit"`, "Marking", "Needed to submit"},
		{attributeKind, "position", `3`, "Place on the form", ""},
		{attributeKind, "parentAttributeId", `"` + tf.attribute(LevelCampaign, "Cluster").ID + `"`, "Depends on", "Cluster"},
		{attributeKind, "condition", `["` + tf.option(LevelPlan, "Business unit", "Lubricants") + `"]`, "Shown when the business unit includes", "Lubricants"},
		{attributeKind, "condition", `[]`, "Shown when the business unit includes", "Always"},
		{optionKind, "parentOptionIds", `["` + tf.option(LevelCampaign, "Cluster", "Europe") + `"]`, "Linked to", "Europe"},
		{optionKind, "position", `2`, "Place in the list", ""},
		{currencyKind, "rate", `"0.92"`, "Rate (units per 1 USD)", "0.92"},
		{currencyKind, "rate", `"1000000"`, "Rate (units per 1 USD)", "1,000,000"},
		{budgetKind, "name", `"2026 Lubricants Budget"`, "Budget name", ""},
		{itemKind, "name", `"Europe – Digital"`, "Item name", ""},
		{itemKind, "budgetId", `"` + item.BudgetID + `"`, "Budget", "2026 Lubricants Budget"},
	}
	for _, tt := range tests {
		t.Run(tt.kind+" "+tt.field+" "+tt.value, func(t *testing.T) {
			decoder := json.NewDecoder(strings.NewReader(tt.value))
			decoder.UseNumber()
			var value any
			if err := decoder.Decode(&value); err != nil {
				t.Fatalf("decode: %v", err)
			}
			var label, text string
			err := f.db.Read(t.Context(), func(ctx context.Context) error {
				var err error
				label, text, err = kinds[tt.kind].Describe(ctx, tt.field, value)
				return err
			})
			if err != nil || label != tt.label || text != tt.text {
				t.Errorf("Describe = %q, %q, %v; want %q, %q", label, text, err, tt.label, tt.text)
			}
		})
	}
}
