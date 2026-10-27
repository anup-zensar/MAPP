package settings

import (
	"encoding/json"
	"reflect"
	"testing"

	"mapp/internal/platform/errs"
)

// A dependent attribute offers only options linked to a selected parent option: one that does not belong
// is refused when chosen (US3-1, FR-025).
func TestCheckAttributes_DependentOptionsFilteredByParent_FR025(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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

// A dependent attribute whose chosen parent options have no active options linked is never missing: it
// cannot be filled in (D-034).
func TestCheckAttributes_NoOptionsForParentNotMissing_D034(t *testing.T) {
	t.Parallel()
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
