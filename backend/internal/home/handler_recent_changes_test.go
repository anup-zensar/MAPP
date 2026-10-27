package home_test

import (
	"net/http"
	"slices"
	"testing"

	"mapp/internal/platform/apitest"
)

// recentChange is the part of an ActivityLogEntry these tests read.
type recentChange struct {
	Action     string `json:"action"`
	RecordType string `json:"recordType"`
	RecordID   string `json:"recordId"`
	RecordName string `json:"recordName"`
	Actor      struct {
		Name string `json:"name"`
	} `json:"actor"`
	RecordNow struct {
		State string `json:"state"`
	} `json:"recordNow"`
}

// isBinnedDelete reports whether c is actorName's delete of the activity recordID, now in the recycle bin.
func (c recentChange) isBinnedDelete(actorName, recordID string) bool {
	return c.Action == "deleted" && c.RecordType == "activity" && c.RecordID == recordID &&
		c.Actor.Name == actorName && c.RecordNow.State == "inRecycleBin"
}

// recentChangesPage is a RecentChangesPage as these tests read it.
type recentChangesPage struct {
	Items []recentChange `json:"items"`
	Total int            `json:"total"`
}

// Priya deletes an activity of her team's plan: her Recent changes start with it, its record now in the
// recycle bin; Mei, in another team, sees it only under Everything; another scope is refused (FR-015,
// FR-016, 011 FR-008).
func TestHomeRecentChanges_Contract(t *testing.T) { //nolint:gocyclo // 12: one delete seen from each scope and person, then the refused scopes
	t.Parallel()
	server := apitest.New(t)
	server.GrantRole(t, meisEmail, "globalPlanner", "all") // 014: every market's plans, for Everything
	priya := server.SignIn(t, priyasEmail)
	mei := server.SignIn(t, meisEmail)

	if deleted := priya.Do(t, http.MethodDelete, "/api/v1/activities/A-1002", nil); deleted.Status != http.StatusOK {
		t.Fatalf("DELETE /api/v1/activities/A-1002 = %d %s", deleted.Status, deleted.Body)
	}
	priyas := get[recentChangesPage](t, priya, "/api/v1/home/recent-changes?scope=myTeams")
	meis := get[recentChangesPage](t, mei, "/api/v1/home/recent-changes?scope=myTeams")
	meisEverything := get[recentChangesPage](t, mei, "/api/v1/home/recent-changes?scope=everything")

	if priyas.Total < 1 || len(priyas.Items) == 0 || len(priyas.Items) > 10 {
		t.Fatalf("Priya's page = %+v, want the delete first", priyas)
	}
	first := priyas.Items[0]
	if !first.isBinnedDelete("Priya Raman", "A-1002") {
		t.Errorf("Priya's newest change = %+v, want her delete of A-1002, in the recycle bin", first)
	}
	if slices.ContainsFunc(meis.Items, func(c recentChange) bool { return c.RecordID == "A-1002" }) {
		t.Errorf("Mei's My teams page = %+v, want nothing of the Lubricants Europe plan", meis)
	}
	if len(meisEverything.Items) == 0 || meisEverything.Items[0].RecordID != "A-1002" {
		t.Errorf("Mei's Everything page = %+v, want the delete first", meisEverything)
	}

	refused := `{"error":{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"scope","message":"Choose My teams or Everything."}]}}`
	for _, path := range []string{"/api/v1/home/recent-changes?scope=all", "/api/v1/home/recent-changes"} {
		response := priya.Do(t, http.MethodGet, path, nil, apitest.SkipRequestValidation())
		if response.Status != http.StatusBadRequest || string(response.Body) != refused {
			t.Errorf("GET %s = %d %s\nwant 400 %s", path, response.Status, response.Body, refused)
		}
	}
}
