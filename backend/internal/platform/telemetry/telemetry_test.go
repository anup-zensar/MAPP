package telemetry_test

import (
	"bytes"
	"strings"
	"testing"

	"mapp/internal/platform/telemetry"
)

func TestStart_OnlyConsoleWritesAndTraceIDsAlwaysExist_024_FR008(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		cfg         telemetry.Config
		wantSpans   bool
		wantMetrics bool
	}{
		{name: "the default exports nothing", cfg: telemetry.Config{}},
		{
			name:      "console traces, no metrics",
			cfg:       telemetry.Config{Traces: telemetry.Console, Metrics: telemetry.None},
			wantSpans: true,
		},
		{
			name:      "console traces and metrics",
			cfg:       telemetry.Config{Traces: telemetry.Console, Metrics: telemetry.Console},
			wantSpans: true, wantMetrics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var console bytes.Buffer
			tel, err := telemetry.Start(tt.cfg, &console)
			if err != nil {
				t.Fatalf("Start: %v", err)
			}
			_, span := tel.Tracer.Tracer("test").Start(t.Context(), "probe span")
			traceID := span.SpanContext().TraceID()
			span.End()
			counter, err := tel.Meter.Meter("test").Int64Counter("probe.requests")
			if err != nil {
				t.Fatalf("Int64Counter: %v", err)
			}
			counter.Add(t.Context(), 1)

			if err := tel.Shutdown(t.Context()); err != nil {
				t.Fatalf("Shutdown: %v", err)
			}

			if !traceID.IsValid() {
				t.Errorf("trace ID %s is not valid, want one for the logs", traceID)
			}
			written := console.String()
			if got := strings.Contains(written, "probe span"); got != tt.wantSpans {
				t.Errorf("span written = %v, want %v; console:\n%s", got, tt.wantSpans, written)
			}
			if got := strings.Contains(written, "probe.requests"); got != tt.wantMetrics {
				t.Errorf("metric written = %v, want %v; console:\n%s", got, tt.wantMetrics, written)
			}
		})
	}
}
