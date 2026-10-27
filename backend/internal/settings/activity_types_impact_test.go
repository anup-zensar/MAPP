package settings

import (
	"reflect"
	"testing"
)

// Before models are removed from a type, the warning says, per model, how many live activities of the
// type its live plans have; they keep the type (FR-012, R7). Adding models or renaming warns of nothing,
// the body is checked as the update checks it, and nothing is saved.
func TestActivityTypeImpact_RemovedModel_FR012(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	digital := f.addType(t, "Digital media", true)
	lubricants := f.createModel(t, modelInput("Lubricants 1.0", event.ID, digital.ID))
	mobility := f.createModel(t, modelInput("Mobility 1.0", event.ID, digital.ID))
	unused := f.createModel(t, modelInput("Unused", event.ID, digital.ID))
	f.refs = []Reference{planRef("P-1001", "A", lubricants.ID, false, nil), planRef("P-1002", "B", mobility.ID, false, nil)}
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.refs = append(f.refs, activityRef(id, "P-1001", event.ID, false, nil))
	}
	f.refs = append(f.refs,
		activityRef("A-1005", "P-1001", event.ID, true, nil),    // in the recycle bin
		activityRef("A-1006", "P-1001", digital.ID, false, nil), // another type
		activityRef("A-1007", "P-1002", event.ID, false, nil))

	impact, err := f.service.ActivityTypeImpact(t.Context(), alex, event.ID, ActivityTypeUpdate{ModelIDs: &[]string{}, Version: version(1)})
	added, err2 := f.service.ActivityTypeImpact(t.Context(), alex, event.ID, ActivityTypeUpdate{
		Name: text("Events"), ModelIDs: &[]string{lubricants.ID, mobility.ID, unused.ID}, Version: version(1),
	})
	f.createModel(t, modelInput("Only events", event.ID))
	_, invalid := f.service.ActivityTypeImpact(t.Context(), alex, event.ID, ActivityTypeUpdate{ModelIDs: &[]string{lubricants.ID}})

	want := []ImpactWarning{
		{Kind: "activityTypeRemoved", Count: 4, Message: "In plans using 'Lubricants 1.0', 4 activities use 'Event'. They keep it; new activities can't choose it."},
		{Kind: "activityTypeRemoved", Count: 1, Message: "In plans using 'Mobility 1.0', 1 activity uses 'Event'. It keeps it; new activities can't choose it."},
	}
	if err != nil || !reflect.DeepEqual(impact.Warnings, want) {
		t.Errorf("impact = %+v, %v\nwant %+v", impact, err, want)
	}
	if err2 != nil || len(added.Warnings) != 0 {
		t.Errorf("adding models and renaming = %+v, %v", added, err2)
	}
	assertError(t, invalid, problems("modelIds", "'Only events' needs at least one activity type, so 'Event' can't be removed from it."))
	if stored, _, _ := f.service.Model(t.Context(), lubricants.ID); !stored.Allows(event.ID) {
		t.Errorf("the impact check removed Event from Lubricants 1.0")
	}
}
