package app_test

import (
	"log/slog"
	"strings"
	"testing"
	"time"

	"mapp/internal/app"
	"mapp/internal/platform/telemetry"
)

func TestLoadConfig_Defaults(t *testing.T) {
	t.Parallel()
	cfg, err := app.LoadConfig(environment(nil))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	want := app.Config{
		Env: "local", Host: "127.0.0.1", Port: 8080, DatabaseURL: "postgres://mapp@127.0.0.1:5432/mapp?sslmode=disable", DemoPassword: "mapp",
		LogLevel: slog.LevelInfo, ShutdownTimeout: 10 * time.Second,
		Telemetry: telemetry.Config{Traces: telemetry.None, Metrics: telemetry.None},
	}
	if cfg != want {
		t.Errorf("LoadConfig = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfig_ReadsEveryVariable(t *testing.T) {
	t.Parallel()
	cfg, err := app.LoadConfig(environment(map[string]string{
		"APP_ENV":               "development",
		"APP_HOST":              "0.0.0.0",
		"APP_PORT":              "18080",
		"DATABASE_URL":          "postgres://planner@127.0.0.1:5433/mapp_dev",
		"DEMO_PASSWORD":         "open sesame",
		"LOG_LEVEL":             "debug",
		"SHUTDOWN_TIMEOUT":      "25s",
		"OTEL_TRACES_EXPORTER":  "console",
		"OTEL_METRICS_EXPORTER": "console",
	}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	want := app.Config{
		Env: "development", Host: "0.0.0.0", Port: 18080, DatabaseURL: "postgres://planner@127.0.0.1:5433/mapp_dev", DemoPassword: "open sesame",
		LogLevel: slog.LevelDebug, ShutdownTimeout: 25 * time.Second,
		Telemetry: telemetry.Config{Traces: telemetry.Console, Metrics: telemetry.Console},
	}
	if cfg != want {
		t.Errorf("LoadConfig = %+v, want %+v", cfg, want)
	}
}

// Invalid values stop the API with the plan's messages; no exporter may send telemetry off this computer.
func TestLoadConfig_RefusesInvalidValues_024_FR012(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		variable string
		value    string
		message  string
	}{
		{"unknown environment", "APP_ENV", "staging", "APP_ENV must be local, development or production (it is 'staging')."},
		{"environment in capitals", "APP_ENV", "Production", "APP_ENV must be local, development or production (it is 'Production')."},
		{"host other than loopback or the container's all-addresses", "APP_HOST", "192.168.1.5", "APP_HOST must be 127.0.0.1 or 0.0.0.0 (it is '192.168.1.5')."},
		{"port 0", "APP_PORT", "0", "APP_PORT must be a number from 1 to 65535 (it is '0')."},
		{"port above 65535", "APP_PORT", "65536", "APP_PORT must be a number from 1 to 65535 (it is '65536')."},
		{"port that is not a number", "APP_PORT", "eighty", "APP_PORT must be a number from 1 to 65535 (it is 'eighty')."},
		{"blank demo password", "DEMO_PASSWORD", "   ", "DEMO_PASSWORD can't be blank."},
		{"unknown log level", "LOG_LEVEL", "loud", "LOG_LEVEL must be debug, info, warn or error (it is 'loud')."},
		{"shutdown timeout that is not a duration", "SHUTDOWN_TIMEOUT", "forever", shutdownMessage("forever")},
		{"shutdown timeout below 1s", "SHUTDOWN_TIMEOUT", "999ms", shutdownMessage("999ms")},
		{"shutdown timeout above 60s", "SHUTDOWN_TIMEOUT", "61s", shutdownMessage("61s")},
		{"traces sent to a collector", "OTEL_TRACES_EXPORTER", "otlp", "OTEL_TRACES_EXPORTER must be none or console (it is 'otlp')."},
		{"metrics sent to a collector", "OTEL_METRICS_EXPORTER", "otlp", "OTEL_METRICS_EXPORTER must be none or console (it is 'otlp')."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.LoadConfig(environment(map[string]string{tt.variable: tt.value}))

			if err == nil || err.Error() != tt.message {
				t.Errorf("LoadConfig = %v, want %q", err, tt.message)
			}
		})
	}
}

func TestLoadConfig_LimitsAreAccepted_024_FR012(t *testing.T) {
	t.Parallel()
	limits := map[string][]string{"APP_PORT": {"1", "65535"}, "SHUTDOWN_TIMEOUT": {"1s", "60s"}}
	for variable, values := range limits {
		for _, value := range values {
			if _, err := app.LoadConfig(environment(map[string]string{variable: value})); err != nil {
				t.Errorf("%s=%s: %v", variable, value, err)
			}
		}
	}
}

// Every invalid value is reported at once, so one start is enough to see what to fix.
func TestLoadConfig_ReportsEveryProblem_024_FR012(t *testing.T) {
	t.Parallel()
	_, err := app.LoadConfig(environment(map[string]string{
		"APP_ENV":              "test",
		"APP_PORT":             "-1",
		"DEMO_PASSWORD":        " ",
		"LOG_LEVEL":            "loud",
		"SHUTDOWN_TIMEOUT":     "forever",
		"OTEL_TRACES_EXPORTER": "otlp",
	}))

	want := []string{
		"APP_ENV must be local, development or production (it is 'test').",
		"APP_PORT must be a number from 1 to 65535 (it is '-1').",
		"DEMO_PASSWORD can't be blank.",
		"LOG_LEVEL must be debug, info, warn or error (it is 'loud').",
		shutdownMessage("forever"),
		"OTEL_TRACES_EXPORTER must be none or console (it is 'otlp').",
	}
	if err == nil || err.Error() != strings.Join(want, "\n") {
		t.Errorf("LoadConfig = %v, want the six messages, one per line", err)
	}
}

// Demo sign-in exists only in the local and development environments (FR-009).
func TestDemoMode_OnlyLocalAndDevelopment_FR009(t *testing.T) {
	t.Parallel()
	for env, want := range map[string]bool{"local": true, "development": true, "production": false} {
		cfg, err := app.LoadConfig(environment(map[string]string{"APP_ENV": env}))
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if got := cfg.DemoMode(); got != want {
			t.Errorf("APP_ENV=%s: DemoMode() = %v, want %v", env, got, want)
		}
	}
}

func shutdownMessage(value string) string {
	return "SHUTDOWN_TIMEOUT must be a duration from 1s to 60s, such as 10s (it is '" + value + "')."
}

// environment returns a getenv function that reads from values instead of the real environment.
func environment(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}
