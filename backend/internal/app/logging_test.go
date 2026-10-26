package app_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/logging"
)

var traceIDForm = regexp.MustCompile(`^[0-9a-f]{32}$`)

// Every record of a signed-in person's request carries its request ID, its trace ID and the person's ID, and
// no record names the person, an e-mail address or the session.
func TestRequestLogs_CarryRequestTraceAndActor_024_FR006(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(logging.New(&out, slog.LevelInfo))
	t.Cleanup(func() { slog.SetDefault(previous) })
	server := apitest.New(t)
	sara := server.SignIn(t, sarasEmail)

	refused := sara.Do(t, http.MethodDelete, "/api/v1/plans/P-1001", nil, apitest.SkipRequestValidation())
	read := sara.Do(t, http.MethodGet, "/api/v1/me", nil)

	records := recordsByRequest(t, out.String())
	for name, response := range map[string]*apitest.Response{"refused delete": refused, "read": read} {
		mine := records[response.Header.Get("X-Request-ID")]
		if len(mine) == 0 {
			t.Fatalf("%s: no log record carries its request ID", name)
		}
		for _, record := range mine {
			trace, _ := record["trace_id"].(string)
			if !traceIDForm.MatchString(trace) || record["actor_id"] != "U-1006" {
				t.Errorf("%s: record %v, want a trace_id and actor_id U-1006", name, record)
			}
		}
	}
	for _, personal := range []string{"Sara", "Novak", "@", "mapp_session"} {
		if strings.Contains(out.String(), personal) {
			t.Errorf("the log contains %q:\n%s", personal, out.String())
		}
	}
}

func recordsByRequest(t *testing.T, log string) map[string][]map[string]any {
	t.Helper()
	records := map[string][]map[string]any{}
	for line := range strings.Lines(log) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		id, _ := record["request_id"].(string)
		records[id] = append(records[id], record)
	}
	return records
}
