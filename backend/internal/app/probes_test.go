package app_test

import (
	"net/http"
	"os"
	"testing"

	"mapp/internal/platform/apitest"
)

const unavailableBody = `{"error":{"code":"internal_error","message":"Something went wrong on our side. Please try again.","fields":[]}}`

// Both probes answer 204 without a session or a body while the API can use its data (022 FR-007, FR-008).
func TestProbes_LiveAndReady_022_FR007_FR008(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)

	for _, path := range []string{"/healthz", "/readyz"} {
		response := server.Do(t, http.MethodGet, path, nil)

		if response.Status != http.StatusNoContent || len(response.Body) != 0 {
			t.Errorf("GET %s = %d %q, want 204 without a body", path, response.Status, response.Body)
		}
	}
}

// Without its data folder the API is not ready (503), and still alive: liveness checks nothing else.
func TestReadyz_DataFolderGone_Answers503_022_FR008(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	server := apitest.New(t, apitest.WithDataDir(dir))
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("removing the data folder: %v", err)
	}

	ready := server.Do(t, http.MethodGet, "/readyz", nil)
	live := server.Do(t, http.MethodGet, "/healthz", nil)

	if ready.Status != http.StatusServiceUnavailable || string(ready.Body) != unavailableBody {
		t.Errorf("GET /readyz = %d %s\nwant 503 %s", ready.Status, ready.Body, unavailableBody)
	}
	if live.Status != http.StatusNoContent {
		t.Errorf("GET /healthz = %d %s, want 204", live.Status, live.Body)
	}
}
