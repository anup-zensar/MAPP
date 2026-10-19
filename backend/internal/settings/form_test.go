package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"northstar/internal/history"
	"northstar/internal/platform/errs"
)

// newFormFixture stores a small form of every kind of attribute, like the seed's (data model §13): the
// Business unit and a Mobility-only yes/no on the plan, Cluster and Region (depending on Cluster) on the
// campaign, every type on the activity in three sections, and Spend type and GL code on the line item.
func newFormFixture(t *testing.T) (*fixture, testForm) {
	t.Helper()
	f := newTestService(t)
	tf := f.addAttributes(t,
		testAttribute{level: LevelPlan, label: "Business unit", typ: TypeMultipleChoice, marking: MarkingNeededToPublish,
			options: []string{"Mobility", "Lubricants"}},
		testAttribute{level: LevelPlan, label: "SMP", typ: TypeYesNo, condition: []string{"Mobility"}},
		testAttribute{level: LevelCampaign, label: "Cluster", typ: TypeMultipleChoice, marking: MarkingNeededToPublish,
			options: []string{"Lubricants – Europe", "Retail – China", "Retail – India"}},
		testAttribute{level: LevelCampaign, label: "Region", typ: TypeMultipleChoice, marking: MarkingNeededToPublish, parent: "Cluster",
			options: []string{"LUBES – UK", "LUBES – DACH", "RETAIL – CHINA"},
			links:   map[string][]string{"LUBES – UK": {"Lubricants – Europe"}, "LUBES – DACH": {"Lubricants – Europe"}, "RETAIL – CHINA": {"Retail – China"}}},
		testAttribute{level: LevelActivity, label: "Country / OU", typ: TypeMultipleChoiceSplit, marking: MarkingNeededToSubmit,
			options: []string{"China", "India", "United Kingdom"}},
		testAttribute{level: LevelActivity, label: "Cost centre", typ: TypeSingleChoice, marking: MarkingNeededToSubmit,
			options: []string{"CN04000220", "GB01000105"}, inactiveOptions: []string{"GB01000105"}},
		testAttribute{level: LevelActivity, label: "FIT4 ID", typ: TypeText, marking: MarkingNeededToSubmit},
		testAttribute{level: LevelActivity, label: "Innovation", typ: TypeLongText, section: "Mobility attributes", condition: []string{"Mobility"}},
		testAttribute{level: LevelActivity, label: "Budget size", typ: TypeNumber},
		testAttribute{level: LevelActivity, label: "Brands", typ: TypeMultipleChoiceSplit, marking: MarkingNeededToSubmit,
			section: "Lubricants attributes", condition: []string{"Lubricants"}, options: []string{"Helix", "Rimula"}},
		testAttribute{level: LevelActivity, label: "Old field", typ: TypeSingleChoice, marking: MarkingNeededToSubmit, inactive: true,
			options: []string{"A", "B"}},
		testAttribute{level: LevelActivity, label: "PIR flag", typ: TypeYesNo},
		testAttribute{level: LevelLineItem, label: "Spend type", typ: TypeSingleChoice, marking: MarkingNeededToSubmit, options: []string{"Digital", "TV"}},
		testAttribute{level: LevelLineItem, label: "GL code", typ: TypeSingleChoice, marking: MarkingNeededToSubmit, parent: "Spend type",
			options: []string{"7250080", "7250010"}, links: map[string][]string{"7250080": {"Digital"}, "7250010": {"TV"}}},
	)
	return f, tf
}

// check runs CheckAttributes and stops the test when it fails.
func (f *fixture) check(t *testing.T, level Level, in AttributeInput) AttributeResult {
	t.Helper()
	result, err := f.service.CheckAttributes(t.Context(), level, in)
	if err != nil {
		t.Fatalf("CheckAttributes: %v", err)
	}
	return result
}

// A level's form has its sections in the order of their first attribute, each attribute — active and
// inactive — with its options in display order; a section whose attributes all have a condition is
// conditional (FR-019, FR-024).
func TestForm_SectionsInOrderOfFirstAttribute_FR019_FR024(t *testing.T) {
	f, tf := newFormFixture(t)

	form, err := f.service.Form(t.Context(), alex, LevelActivity)
	if err != nil {
		t.Fatalf("Form: %v", err)
	}

	var sections []string
	var conditional []bool
	var labels [][]string
	for _, s := range form.Sections {
		sections = append(sections, s.Name)
		conditional = append(conditional, s.Conditional)
		var names []string
		for _, a := range s.Attributes {
			names = append(names, a.Label)
		}
		labels = append(labels, names)
	}
	wantLabels := [][]string{
		{"Country / OU", "Cost centre", "FIT4 ID", "Budget size", "Old field", "PIR flag"}, {"Innovation"}, {"Brands"},
	}
	if !reflect.DeepEqual(sections, []string{"Shell attributes", "Mobility attributes", "Lubricants attributes"}) ||
		!reflect.DeepEqual(conditional, []bool{false, true, true}) || !reflect.DeepEqual(labels, wantLabels) {
		t.Errorf("sections = %q %v %q", sections, conditional, labels)
	}
	businessUnit := tf.attribute(LevelPlan, "Business unit").ID
	if form.BusinessUnitAttributeID == nil || *form.BusinessUnitAttributeID != businessUnit || len(form.BusinessUnitOptions) != 2 {
		t.Errorf("business unit = %v %+v", form.BusinessUnitAttributeID, form.BusinessUnitOptions)
	}
	costCentre := form.Sections[0].Attributes[1]
	wantOptions := []FormOption{
		{ID: tf.option(LevelActivity, "Cost centre", "CN04000220"), Label: "CN04000220", Active: true, ParentOptionIDs: []string{}},
		{ID: tf.option(LevelActivity, "Cost centre", "GB01000105"), Label: "GB01000105", Active: false, ParentOptionIDs: []string{}},
	}
	if !reflect.DeepEqual(costCentre.Options, wantOptions) || costCentre.ParentAttributeID != nil {
		t.Errorf("Cost centre = %+v", costCentre)
	}
	region := mustForm(t, f, LevelCampaign).Sections[0].Attributes[1]
	if region.ParentAttributeID == nil || *region.ParentAttributeID != tf.attribute(LevelCampaign, "Cluster").ID {
		t.Errorf("Region depends on %v", region.ParentAttributeID)
	}
}

// mustForm returns the form of level.
func mustForm(t *testing.T, f *fixture, level Level) FormView {
	t.Helper()
	form, err := f.service.Form(t.Context(), alex, level)
	if err != nil {
		t.Fatalf("Form(%s): %v", level, err)
	}
	return form
}

// Every value must have the shape of its attribute's type and stay within its limits; values are stored
// trimmed and empty ones are left out (FR-016, data model §9).
func TestCheckAttributes_ShapesAndLimits_FR016(t *testing.T) {
	f, tf := newFormFixture(t)
	id := func(label string) string { return tf.attribute(LevelActivity, label).ID }
	china := tf.option(LevelActivity, "Country / OU", "China")
	tests := []struct {
		name, attribute, value, message string
	}{
		{"single choice, unknown option", "Cost centre", `"SO-9999"`, "Choose one of the options of Cost centre."},
		{"single choice, another attribute's option", "Cost centre", `"` + china + `"`, "Choose one of the options of Cost centre."},
		{"single choice, a list", "Cost centre", `["x"]`, "Choose one of the options of Cost centre."},
		{"split, share 0", "Country / OU", `[{"optionId":"` + china + `","share":"0"}]`, "Enter a share above 0 % with at most two decimals."},
		{"split, three decimals", "Country / OU", `[{"optionId":"` + china + `","share":"33.333"}]`, "Enter a share above 0 % with at most two decimals."},
		{"split, an option twice", "Country / OU", `[{"optionId":"` + china + `","share":"50"},{"optionId":"` + china + `","share":"50"}]`, "Each option can appear only once."},
		{"text, a number", "FIT4 ID", `12`, "Enter text."},
		{"text, 121 characters", "FIT4 ID", `"` + strings.Repeat("x", 121) + `"`, "Use 120 characters or fewer."},
		{"long text, 2,001 characters", "Innovation", `"` + strings.Repeat("é", 2001) + `"`, "Use 2,000 characters or fewer."},
		{"number, not whole", "Budget size", `12.5`, "Enter a whole number between -999,999,999,999 and 999,999,999,999."},
		{"number, too large", "Budget size", `1000000000000`, "Enter a whole number between -999,999,999,999 and 999,999,999,999."},
		{"number, as text", "Budget size", `"12"`, "Enter a whole number between -999,999,999,999 and 999,999,999,999."},
		{"yes/no, a text", "PIR flag", `"yes"`, "Choose Yes or No."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := f.check(t, LevelActivity, AttributeInput{Values: values(id(tt.attribute), tt.value)})

			want := []errs.FieldError{{Field: "attributes." + id(tt.attribute), Message: tt.message}}
			if !reflect.DeepEqual(result.Problems, want) {
				t.Errorf("problems = %+v, want %+v", result.Problems, want)
			}
		})
	}

	result := f.check(t, LevelActivity, AttributeInput{Values: values(
		id("FIT4 ID"), `"  FIT4-0042 "`, id("Innovation"), `"   "`, id("Budget size"), `-999999999999`, id("PIR flag"), `false`,
		id("Country / OU"), `[{"optionId":"`+china+`","share":"100"}]`, id("Cost centre"), `""`,
		tf.attribute(LevelCampaign, "Cluster").ID, `[]`, "SA-9999", `"x"`,
	)})
	wantProblems := []errs.FieldError{
		{Field: "attributes." + tf.attribute(LevelCampaign, "Cluster").ID, Message: "Cluster belongs to the Campaign form."},
		{Field: "attributes.SA-9999", Message: "This Shell attribute no longer exists — reload the page."},
	}
	wantValues := values(id("FIT4 ID"), `"FIT4-0042"`, id("Budget size"), `-999999999999`, id("PIR flag"), `false`,
		id("Country / OU"), `[{"optionId":"`+china+`","share":"100.00"}]`)
	if !reflect.DeepEqual(result.Problems, wantProblems) || !reflect.DeepEqual(result.Values, wantValues) {
		t.Errorf("result = %+v\nwant problems %+v and values %s", result, wantProblems, wantValues)
	}
}

// A dependent attribute offers only options linked to a selected parent option: one that does not belong
// is refused when chosen (US3-1, FR-025).
func TestCheckAttributes_DependentOptionsFilteredByParent_FR025(t *testing.T) {
	f, tf := newFormFixture(t)
	cluster, region := tf.attribute(LevelCampaign, "Cluster").ID, tf.attribute(LevelCampaign, "Region").ID
	retailChina := `["` + tf.option(LevelCampaign, "Cluster", "Retail – China") + `"]`
	before := values(cluster, retailChina)

	fits := f.check(t, LevelCampaign, AttributeInput{Previous: before,
		Values: values(cluster, retailChina, region, `["`+tf.option(LevelCampaign, "Region", "RETAIL – CHINA")+`"]`)})
	wrong := f.check(t, LevelCampaign, AttributeInput{Previous: before,
		Values: values(cluster, retailChina, region, `["`+tf.option(LevelCampaign, "Region", "LUBES – UK")+`"]`)})

	if len(fits.Problems) != 0 || len(fits.Missing) != 0 {
		t.Errorf("fitting region: %+v", fits)
	}
	want := []errs.FieldError{{Field: "attributes." + region,
		Message: "'LUBES – UK' doesn't belong to the chosen cluster — choose another region."}}
	if !reflect.DeepEqual(wrong.Problems, want) {
		t.Errorf("problems = %+v, want %+v", wrong.Problems, want)
	}
}

// When a parent value changes, the dependent values that no longer fit are cleared and listed, and the
// rest is kept (US3-4, FR-025).
func TestCheckAttributes_ParentChangeClearsDependent_FR025(t *testing.T) {
	f, tf := newFormFixture(t)
	cluster, region := tf.attribute(LevelCampaign, "Cluster").ID, tf.attribute(LevelCampaign, "Region").ID
	europe, china := tf.option(LevelCampaign, "Cluster", "Lubricants – Europe"), tf.option(LevelCampaign, "Cluster", "Retail – China")
	uk, dach, retail := tf.option(LevelCampaign, "Region", "LUBES – UK"), tf.option(LevelCampaign, "Region", "LUBES – DACH"),
		tf.option(LevelCampaign, "Region", "RETAIL – CHINA")
	before := values(cluster, `["`+europe+`"]`, region, `["`+uk+`","`+dach+`"]`)

	all := f.check(t, LevelCampaign, AttributeInput{Previous: before, Values: values(cluster, `["`+china+`"]`, region, `["`+uk+`","`+dach+`"]`)})
	some := f.check(t, LevelCampaign, AttributeInput{Previous: before,
		Values: values(cluster, `["`+china+`"]`, region, `["`+uk+`","`+retail+`"]`)})

	wantCleared := []ClearedValue{{AttributeID: region, Label: "Region", OptionIDs: []string{uk, dach}, OptionLabels: []string{"LUBES – UK", "LUBES – DACH"}}}
	wantMissing := []MissingField{{Field: "attributes." + region, Label: "Region"}}
	if !reflect.DeepEqual(all.Cleared, wantCleared) || !reflect.DeepEqual(all.Values, values(cluster, `["`+china+`"]`)) ||
		!reflect.DeepEqual(all.Missing, wantMissing) || len(all.Problems) != 0 {
		t.Errorf("all cleared: %+v", all)
	}
	if len(some.Cleared) != 1 || !reflect.DeepEqual(some.Values[region], json.RawMessage(`["`+retail+`"]`)) {
		t.Errorf("one cleared: %+v", some)
	}
}

// An attribute whose condition is not met is hidden: it keeps its value and is never missing (US3-3,
// FR-026); at plan level the business unit comes from the values themselves.
func TestCheckAttributes_HiddenKeptNotMissing_FR026(t *testing.T) {
	f, tf := newFormFixture(t)
	brands, helix := tf.attribute(LevelActivity, "Brands").ID, tf.option(LevelActivity, "Brands", "Helix")
	mobility, lubricants := tf.option(LevelPlan, "Business unit", "Mobility"), tf.option(LevelPlan, "Business unit", "Lubricants")
	half := `[{"optionId":"` + helix + `","share":"50"}]`

	hidden := f.check(t, LevelActivity, AttributeInput{Values: values(brands, half), BusinessUnit: []string{mobility}})
	shown := f.check(t, LevelActivity, AttributeInput{Values: values(), BusinessUnit: []string{lubricants}})
	plan := f.check(t, LevelPlan, AttributeInput{Values: values(tf.attribute(LevelPlan, "Business unit").ID, `["`+mobility+`"]`)})

	if hasMissing(hidden.Missing, brands) || hidden.Values[brands] == nil {
		t.Errorf("hidden Brands: missing %+v, values %s", hidden.Missing, hidden.Values)
	}
	if !hasMissing(shown.Missing, brands) {
		t.Errorf("shown Brands not missing: %+v", shown.Missing)
	}
	if len(plan.Missing) != 0 { // SMP is shown now, but yes/no is never missing
		t.Errorf("plan missing = %+v", plan.Missing)
	}
}

// An inactive attribute keeps its value read-only and never blocks; an inactive option the record already
// has stays and counts as filled in (FR-027).
func TestCheckAttributes_InactiveKeptNeverBlocks_FR027(t *testing.T) {
	f, tf := newFormFixture(t)
	old, a, b := tf.attribute(LevelActivity, "Old field").ID, tf.option(LevelActivity, "Old field", "A"), tf.option(LevelActivity, "Old field", "B")
	costCentre, inactive := tf.attribute(LevelActivity, "Cost centre").ID, tf.option(LevelActivity, "Cost centre", "GB01000105")
	before := values(old, `"`+a+`"`, costCentre, `"`+inactive+`"`)

	kept := f.check(t, LevelActivity, AttributeInput{Previous: before, Values: values(costCentre, `"`+inactive+`"`)})
	changed := f.check(t, LevelActivity, AttributeInput{Previous: before, Values: values(old, `"`+b+`"`)})

	if len(kept.Problems) != 0 || !reflect.DeepEqual(kept.Values[old], json.RawMessage(`"`+a+`"`)) ||
		hasMissing(kept.Missing, old) || hasMissing(kept.Missing, costCentre) {
		t.Errorf("kept: %+v", kept)
	}
	want := []errs.FieldError{{Field: "attributes." + old, Message: "Old field is inactive and can't be changed."}}
	if !reflect.DeepEqual(changed.Problems, want) {
		t.Errorf("problems = %+v, want %+v", changed.Problems, want)
	}
}

// An inactive option cannot be chosen newly (FR-004).
func TestCheckAttributes_NewInactiveOptionRefused_FR004(t *testing.T) {
	f, tf := newFormFixture(t)
	costCentre := tf.attribute(LevelActivity, "Cost centre").ID

	result := f.check(t, LevelActivity, AttributeInput{Values: values(costCentre, `"`+tf.option(LevelActivity, "Cost centre", "GB01000105")+`"`)})

	want := []errs.FieldError{{Field: "attributes." + costCentre, Message: "'GB01000105' is inactive — choose another option."}}
	if !reflect.DeepEqual(result.Problems, want) {
		t.Errorf("problems = %+v, want %+v", result.Problems, want)
	}
}

// A % split may be saved with any total, but counts as filled in only at exactly 100.00 % (R10, FR-027).
func TestCheckAttributes_SplitMustTotal100ToCount_FR027(t *testing.T) {
	f, tf := newFormFixture(t)
	country := tf.attribute(LevelActivity, "Country / OU").ID
	china, india := tf.option(LevelActivity, "Country / OU", "China"), tf.option(LevelActivity, "Country / OU", "India")
	split := func(a, b string) string {
		return `[{"optionId":"` + china + `","share":"` + a + `"},{"optionId":"` + india + `","share":"` + b + `"}]`
	}

	ninety := f.check(t, LevelActivity, AttributeInput{Values: values(country, split("60", "30"))})
	hundred := f.check(t, LevelActivity, AttributeInput{Values: values(country, split("60", "40"))})

	want := MissingField{Field: "attributes." + country, Label: "Country / OU", Reason: "Shares add up to 90.00 % — they must add up to 100.00 %."}
	if len(ninety.Problems) != 0 || ninety.Values[country] == nil || ninety.Missing[0] != want {
		t.Errorf("90 %%: %+v", ninety)
	}
	if hasMissing(hundred.Missing, country) {
		t.Errorf("100 %%: still missing %+v", hundred.Missing)
	}
}

// A dependent attribute whose chosen parent options have no active options linked is never missing: it
// cannot be filled in (D-034).
func TestCheckAttributes_NoOptionsForParentNotMissing_D034(t *testing.T) {
	f, tf := newFormFixture(t)
	cluster, region := tf.attribute(LevelCampaign, "Cluster").ID, tf.attribute(LevelCampaign, "Region").ID

	india := f.check(t, LevelCampaign, AttributeInput{Values: values(cluster, `["`+tf.option(LevelCampaign, "Cluster", "Retail – India")+`"]`)})
	none := f.check(t, LevelCampaign, AttributeInput{Values: values()})

	if hasMissing(india.Missing, region) {
		t.Errorf("Region missing for Retail – India: %+v", india.Missing)
	}
	if !hasMissing(none.Missing, region) || !hasMissing(none.Missing, cluster) {
		t.Errorf("without a cluster: %+v", none.Missing)
	}
}

// The needed attributes are those marked Needed to publish at plan, campaign and initiative level and
// Needed to submit at activity and line-item level; optional ones and yes/no never block (FR-017).
func TestMissingAttributes_MarkingPerLevel_FR017(t *testing.T) {
	f, tf := newFormFixture(t)
	lubricants := tf.option(LevelPlan, "Business unit", "Lubricants")
	labels := func(level Level, v AttributeValues, businessUnit []string) []string {
		missing, err := f.service.MissingAttributes(t.Context(), level, v, businessUnit)
		if err != nil {
			t.Fatalf("MissingAttributes: %v", err)
		}
		var list []string
		for _, m := range missing {
			list = append(list, m.Label)
		}
		return list
	}

	plan := labels(LevelPlan, values(), nil)
	activity := labels(LevelActivity, values(tf.attribute(LevelActivity, "FIT4 ID").ID, `"F-1"`), []string{lubricants})
	lineItem := labels(LevelLineItem, values(tf.attribute(LevelLineItem, "Spend type").ID, `"`+tf.option(LevelLineItem, "Spend type", "TV")+`"`), nil)

	if !reflect.DeepEqual(plan, []string{"Business unit"}) {
		t.Errorf("plan = %q", plan)
	}
	if !reflect.DeepEqual(activity, []string{"Country / OU", "Cost centre", "Brands"}) {
		t.Errorf("activity = %q", activity)
	}
	if !reflect.DeepEqual(lineItem, []string{"GL code"}) {
		t.Errorf("line item = %q", lineItem)
	}
}

// The plan's business unit is the options of the Business unit attribute in its values (FR-023).
func TestBusinessUnit_FromPlanValues_FR023(t *testing.T) {
	f, tf := newFormFixture(t)
	mobility := tf.option(LevelPlan, "Business unit", "Mobility")

	chosen, err1 := f.service.BusinessUnit(t.Context(), values(tf.attribute(LevelPlan, "Business unit").ID, `["`+mobility+`"]`))
	none, err2 := f.service.BusinessUnit(t.Context(), nil)

	if err1 != nil || err2 != nil || !reflect.DeepEqual(chosen, []string{mobility}) || none == nil || len(none) != 0 {
		t.Errorf("BusinessUnit = %q, %q (%v, %v)", chosen, none, err1, err2)
	}
}

// A planning record's changes are logged field by field, its Shell attribute values attribute by
// attribute, sorted by field (FR-006, D-048).
func TestDiff_AttributeByAttribute_FR006(t *testing.T) {
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

// hasMissing reports whether missing lists the attribute id.
func hasMissing(missing []MissingField, id string) bool {
	for _, m := range missing {
		if m.Field == "attributes."+id {
			return true
		}
	}
	return false
}
