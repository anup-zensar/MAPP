package settings

import (
	"fmt"
	"reflect"
	"testing"
)

// Before new rates or deal size are saved, the plans and activities of a model in use are stated (FR-011,
// US1-4); typed goals stay, as the message says.
func TestModelImpact_RatesInUse_FR011(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, false, nil), planRef("P-1002", "B", m.ID, false, nil), planRef("P-1003", "C", m.ID, true, nil)}
	for i := range 31 {
		f.refs = append(f.refs, activityRef(fmt.Sprintf("A-%d", 1001+i), "P-1001", event.ID, false, nil))
	}

	impact, err := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{MqlToDealRate: text("6"), Version: version(1)})

	want := []ImpactWarning{{Kind: "goalsRecalculated", Count: 2, Message: "Used by 2 plans (31 activities). Their calculated MQL and " +
		"revenue goals will be recalculated; typed goals ('Set later stages myself') are kept."}}
	if err != nil || !reflect.DeepEqual(impact.Warnings, want) {
		t.Errorf("impact = %+v, %v\nwant %+v", impact, err, want)
	}
	if stored, _, _ := f.service.Model(t.Context(), m.ID); stored.MqlToDealRate.String() != "5.00" {
		t.Errorf("the impact check saved the rate: %s", stored.MqlToDealRate)
	}
	_, invalid := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{MqlToDealRate: text("0")})
	assertError(t, invalid, problems("mqlToDealRate", "Enter a rate above 0 % and at most 100 %, with at most two decimals."))
}

// A model nothing uses has nothing to confirm; a change of its name alone neither (FR-011).
func TestModelImpact_Unused_NoWarning_FR011(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	m := f.createModel(t, modelInput("Model", event.ID))
	used := f.createModel(t, modelInput("Used", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", used.ID, false, nil)}

	unused, err1 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{InqToMqlRate: text("3"), Version: version(1)})
	renamed, err2 := f.service.ModelImpact(t.Context(), alex, used.ID, ModelUpdate{Name: text("Used 2"), Version: version(1)})

	if err1 != nil || err2 != nil || len(unused.Warnings) != 0 || len(renamed.Warnings) != 0 {
		t.Errorf("impacts = %+v (%v), %+v (%v); want no warnings", unused, err1, renamed, err2)
	}
}

// Removing an allowed activity type states how many activities in the model's plans use it; they keep it
// (US1-6, FR-012, R7).
func TestModelImpact_RemovedTypeInUse_FR012(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)
	m := f.createModel(t, modelInput("Lubricants 1.0", event.ID, digital.ID))
	other := f.createModel(t, modelInput("Other", event.ID))
	f.refs = []Reference{planRef("P-1001", "A", m.ID, false, nil), planRef("P-1002", "B", other.ID, false, nil)}
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005"} {
		f.refs = append(f.refs, activityRef(id, "P-1001", event.ID, false, nil))
	}
	f.refs = append(f.refs,
		activityRef("A-1006", "P-1002", event.ID, false, nil), // another model's plan
		activityRef("A-1007", "P-1001", event.ID, true, nil),  // in the recycle bin
		activityRef("A-1008", "P-1001", digital.ID, false, nil))

	five, err1 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{digital.ID}})
	one, err2 := f.service.ModelImpact(t.Context(), alex, m.ID, ModelUpdate{ActivityTypeIDs: &[]string{event.ID}})

	wantFive := []ImpactWarning{{Kind: "activityTypeRemoved", Count: 5, Message: "5 activities use 'Event'. They keep it; new activities can't choose it."}}
	wantOne := []ImpactWarning{{Kind: "activityTypeRemoved", Count: 1, Message: "1 activity uses 'Digital media'. It keeps it; new activities can't choose it."}}
	if err1 != nil || !reflect.DeepEqual(five.Warnings, wantFive) {
		t.Errorf("removing Event: %+v, %v", five, err1)
	}
	if err2 != nil || !reflect.DeepEqual(one.Warnings, wantOne) {
		t.Errorf("removing Digital media: %+v, %v", one, err2)
	}
}
