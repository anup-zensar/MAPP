package plans_test

import (
	"net/http"
	"testing"

	"mapp/internal/platform/apitest"
)

// Deleting a plan answers its recycle-bin entry, so the web app can offer Undo; Undo brings it back with
// the same ID and status (FR-029, FR-030, US6 scenarios 3 and 4).
func TestDeletePlan_AndUndo_FR029_FR030(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	server.GrantRole(t, priyasEmail, "globalPlanner", "all") // 014: deleting plans

	deleted := priya.Do(t, http.MethodDelete, "/api/v1/plans/P-1001", nil)
	inBin := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001", nil)
	var result struct {
		RecycleBinEntry struct {
			ID              string `json:"id"`
			RecordType      string `json:"recordType"`
			RecordTypeLabel string `json:"recordTypeLabel"`
			RecordID        string `json:"recordId"`
			RecordName      string `json:"recordName"`
		} `json:"recycleBinEntry"`
	}
	deleted.Decode(t, &result)
	undone := priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/"+result.RecycleBinEntry.ID+"/undo", nil)
	back := priya.Do(t, http.MethodGet, "/api/v1/plans/P-1001", nil)
	gone := priya.Do(t, http.MethodDelete, "/api/v1/plans/P-9999", nil)

	if deleted.Status != http.StatusOK || result.RecycleBinEntry.RecordType != "plan" ||
		result.RecycleBinEntry.RecordTypeLabel != "plan" || result.RecycleBinEntry.RecordID != "P-1001" ||
		result.RecycleBinEntry.RecordName != "FY2026 Lubricants Europe" {
		t.Errorf("delete = %d %s", deleted.Status, deleted.Body)
	}
	assertAnswer(t, inBin, http.StatusNotFound, planGoneBody)
	assertFields(t, undone, http.StatusOK, map[string]any{
		"recordType": "plan", "recordId": "P-1001", "recordName": "FY2026 Lubricants Europe", "details": "",
	})
	assertFields(t, back, http.StatusOK, map[string]any{"id": "P-1001", "status": "published", "version": 1})
	assertAnswer(t, gone, http.StatusNotFound, planGoneBody)
}
