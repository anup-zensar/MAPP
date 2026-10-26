package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"mapp/internal/platform/logging"
)

func TestHandler_AddsRequestAttributesAndTraceID_024_FR006(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	logger := logging.New(&out, slog.LevelInfo)
	traceID := trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	ctx := logging.WithAttrs(t.Context(), slog.String("request_id", "R-1"), slog.String("actor_id", "U-1006"))
	ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID}))

	logger.InfoContext(ctx, "request refused")
	logger.InfoContext(t.Context(), "outside a request")

	records := decode(t, &out)
	if stamp, _ := records[0]["time"].(string); !strings.HasSuffix(stamp, "Z") {
		t.Errorf("time = %q, want UTC", stamp)
	}
	want := map[string]any{"request_id": "R-1", "actor_id": "U-1006", "trace_id": traceID.String()}
	for key, value := range want {
		if records[0][key] != value {
			t.Errorf("request record %s = %v, want %v", key, records[0][key], value)
		}
		if _, found := records[1][key]; found {
			t.Errorf("record outside a request has %s", key)
		}
	}
}

func TestWithAttrs_SiblingsKeepTheirOwnAttributes_024_FR006(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	logger := logging.New(&out, slog.LevelInfo)
	parent := t.Context()
	for _, key := range []string{"request_id", "route", "actor_id"} {
		parent = logging.WithAttrs(parent, slog.String(key, key+"-value"))
	}
	first := logging.WithAttrs(parent, slog.String("step", "first"))
	second := logging.WithAttrs(parent, slog.String("step", "second"))

	logger.InfoContext(first, "step")
	logger.InfoContext(second, "step")

	records := decode(t, &out)
	if records[0]["step"] != "first" || records[1]["step"] != "second" {
		t.Errorf("steps = %v, %v; want first, second", records[0]["step"], records[1]["step"])
	}
}

func decode(t *testing.T, out *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	for line := range strings.Lines(out.String()) {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("log line %q is not JSON: %v", line, err)
		}
		records = append(records, record)
	}
	if len(records) != 2 {
		t.Fatalf("log has %d records, want 2:\n%s", len(records), out.String())
	}
	return records
}
