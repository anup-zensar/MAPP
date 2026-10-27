package settings

import (
	"reflect"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
)

// A type counts the activities of that type and the models allowing it, those in the recycle bin too; its
// details list the stored models allowing it (US5-2, FR-003).
func TestActivityTypeUsage_ActivitiesAndModels_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	lubricants := f.createModel(t, modelInput("Lubricants 1.0", event.ID))
	old := f.createModel(t, modelInput("Old", event.ID))
	if _, err := f.service.DeleteModel(t.Context(), alex, old.ID); err != nil {
		t.Fatalf("DeleteModel: %v", err)
	}
	f.refs = []Reference{
		activityRef("A-1001", "P-1001", event.ID, false, nil), activityRef("A-1002", "P-1001", event.ID, false, nil),
		activityRef("A-1003", "P-1001", event.ID, false, nil), activityRef("A-1004", "P-1002", event.ID, true, nil),
	}

	view, err := f.service.GetActivityType(t.Context(), alex, event.ID)

	want := Usage{InUse: true, Summary: "Used by 4 activities (1 in the recycle bin) · Allowed in 2 models (1 in the recycle bin)",
		Parts: []UsagePart{
			{Kind: "activity", Text: "Used by 4 activities (1 in the recycle bin)", Count: 4, InRecycleBin: 1},
			{Kind: "model", Text: "Allowed in 2 models (1 in the recycle bin)", Count: 2, InRecycleBin: 1},
		}, Records: []history.RecordRef{}}
	if err != nil || !reflect.DeepEqual(view.Usage, want) {
		t.Errorf("usage = %+v, %v\nwant %+v", view.Usage, err, want)
	}
	if !reflect.DeepEqual(view.Models, []SettingRef{{ID: lubricants.ID, Name: "Lubricants 1.0", Active: true}}) {
		t.Errorf("models = %+v", view.Models)
	}
}

// A type in use — by activities, also in the recycle bin, or allowed in a model — cannot be deleted: the
// answer says where it is used and to set it inactive instead (FR-004, SC-004).
func TestDeleteActivityType_InUseRefused_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	event := f.addType(t, "Event", true)
	webinar := f.addType(t, "Webinar", true)
	f.createModel(t, modelInput("Lubricants 1.0", webinar.ID))
	f.refs = []Reference{activityRef("A-1001", "P-1001", event.ID, true, nil)}

	_, used := f.service.DeleteActivityType(t.Context(), alex, event.ID)
	_, allowed := f.service.DeleteActivityType(t.Context(), alex, webinar.ID)
	_, gone := f.service.DeleteActivityType(t.Context(), alex, "AT-9999")

	assertError(t, used, errs.InUse("'Event' is in use (used by 1 activity (1 in the recycle bin)) — set it inactive instead."))
	assertError(t, allowed, errs.InUse("'Webinar' is in use (allowed in 1 model) — set it inactive instead."))
	assertError(t, gone, errs.NotFound("This activity type no longer exists — someone may have deleted it."))
	if types, _ := f.service.ActivityTypes(t.Context()); len(types) != 2 {
		t.Errorf("activity types = %d, want both kept", len(types))
	}
}
