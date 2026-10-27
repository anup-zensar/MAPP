package settings

import (
	"reflect"
	"testing"
)

// The needed attributes are those marked Needed to publish at plan, campaign and initiative level and
// Needed to submit at activity and line-item level; optional ones and yes/no never block (FR-017).
func TestMissingAttributes_MarkingPerLevel_FR017(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	f, tf := newFormFixture(t)
	mobility := tf.option(LevelPlan, "Business unit", "Mobility")

	chosen, err1 := f.service.BusinessUnit(t.Context(), values(tf.attribute(LevelPlan, "Business unit").ID, `["`+mobility+`"]`))
	none, err2 := f.service.BusinessUnit(t.Context(), nil)

	if err1 != nil || err2 != nil || !reflect.DeepEqual(chosen, []string{mobility}) || none == nil || len(none) != 0 {
		t.Errorf("BusinessUnit = %q, %q (%v, %v)", chosen, none, err1, err2)
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
