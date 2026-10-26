package web_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"mapp/internal/platform/logging"
	"mapp/internal/platform/querycount"
)

func TestEngine_RequestRecordCountsTheStatementsSent_028_FR009(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(logging.New(&out, slog.LevelInfo))
	t.Cleanup(func() { slog.SetDefault(previous) })
	threeStatements := func(c *gin.Context) {
		for range 3 {
			querycount.Add(c.Request.Context())
		}
		c.Status(http.StatusNoContent)
	}

	serve(t, threeStatements, newRequest(http.MethodGet, testPath, ""))

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("log %q is not one JSON record: %v", out.String(), err)
	}
	if record["queries"] != float64(3) {
		t.Errorf("queries = %v, want 3", record["queries"])
	}
}
