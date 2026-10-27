package seed_test

import (
	"cmp"
	"net/http"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/apitest"
)

// The deleted initiative's snapshots are records of features 005–007 as they store them: the restore preview
// works, and Restore brings the initiative back with its activity and line item, under Rimula fleet (011
// FR-018, FR-021).
func TestSeed_DeletedInitiative_Restores_011(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")

	response := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin/BIN-1001/restore-preview", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("restore preview = %d %s", response.Status, response.Body)
	}
	var preview history.RestorePreview
	response.Decode(t, &preview)
	if len(preview.Destination) != 2 || preview.Destination[1].Name != "Rimula fleet" || len(preview.ParentEntries) != 0 {
		t.Errorf("preview = %+v, want back into FY2026 Lubricants Europe › Rimula fleet", preview)
	}

	response = priya.Do(t, http.MethodPost, "/api/v1/recycle-bin/BIN-1001/restore", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("restore = %d %s", response.Status, response.Body)
	}
	for path, want := range map[string]string{
		"/api/v1/initiatives/I-1011": "Trade shows", "/api/v1/activities/A-1013": "Fleet expo stand", "/api/v1/line-items/L-1011": "Stand build",
	} {
		response := priya.Do(t, http.MethodGet, path, nil)
		var got struct {
			Name     string `json:"name"`
			LineItem struct {
				Name string `json:"name"`
			} `json:"lineItem"` // GET /api/v1/line-items/{id} answers the line item with its activity
		}
		if response.Status == http.StatusOK {
			response.Decode(t, &got)
		}
		if response.Status != http.StatusOK || cmp.Or(got.Name, got.LineItem.Name) != want {
			t.Errorf("GET %s after the restore = %d %s, want %s", path, response.Status, response.Body, want)
		}
	}
}
