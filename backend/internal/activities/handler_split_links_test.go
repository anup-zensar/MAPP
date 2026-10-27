package activities_test

import (
	"net/http"
	"testing"

	"mapp/internal/activities"
	"mapp/internal/platform/apitest"
)

// When the original goes to the recycle bin, the new activity's link says so ("deleted"); a copy of the new
// activity carries no link (FR-031).
func TestSplitLinks_DeletedOriginal_FR031(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1))
	var result activities.SplitResult
	split.Decode(t, &result)

	deleted := priya.Do(t, http.MethodDelete, "/api/v1/activities/"+ukLaunchEvent, nil)
	assertStatus(t, deleted, http.StatusOK)
	var created activities.View
	priya.Do(t, http.MethodGet, "/api/v1/activities/"+result.NewActivity.ID, nil).Decode(t, &created)

	if created.SplitFrom == nil || *created.SplitFrom != (activities.SplitLink{ID: ukLaunchEvent, Name: "UK launch event", PlanID: "P-1004", Deleted: true}) {
		t.Errorf("split from = %+v, want UK launch event, deleted", created.SplitFrom)
	}
}

// A copy never carries the split link (FR-031, R13).
func TestCopy_DoesNotCarrySplitFrom_FR031(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	split := priya.Do(t, http.MethodPost, "/api/v1/activities/"+ukLaunchEvent+"/split",
		splitBody("Ireland roadshow", irelandInit, []string{agencyItem}, "19750.00", false, 1))
	var result activities.SplitResult
	split.Decode(t, &result)

	copied := priya.Do(t, http.MethodPost, "/api/v1/activities/"+result.NewActivity.ID+"/copy", map[string]any{"initiativeId": irelandInit})

	assertStatus(t, copied, http.StatusCreated)
	var duplicate activities.View
	copied.Decode(t, &duplicate)
	if duplicate.SplitFrom != nil || len(duplicate.SplitInto) != 0 {
		t.Errorf("copy links: from %+v, into %+v; want none", duplicate.SplitFrom, duplicate.SplitInto)
	}
}
