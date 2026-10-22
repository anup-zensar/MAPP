package app_test

import (
	"strings"
	"testing"

	"mapp/internal/app"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg, err := app.LoadConfig(environment(nil))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	want := app.Config{Env: "local", Port: 8080, DataDir: "data", DemoPassword: "mapp"}
	if cfg != want {
		t.Errorf("LoadConfig = %+v, want %+v", cfg, want)
	}
}

func TestLoadConfig_ReadsEveryVariable(t *testing.T) {
	cfg, err := app.LoadConfig(environment(map[string]string{
		"APP_ENV":       "development",
		"APP_PORT":      "18080",
		"DATA_DIR":      "C:/temp/mapp-data",
		"DEMO_PASSWORD": "open sesame",
	}))
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	want := app.Config{Env: "development", Port: 18080, DataDir: "C:/temp/mapp-data", DemoPassword: "open sesame"}
	if cfg != want {
		t.Errorf("LoadConfig = %+v, want %+v", cfg, want)
	}
}

// Invalid values stop the API with the plan's messages.
func TestLoadConfig_RefusesInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
		message  string
	}{
		{"unknown environment", "APP_ENV", "staging", "APP_ENV must be local, development or production (it is 'staging')."},
		{"environment in capitals", "APP_ENV", "Production", "APP_ENV must be local, development or production (it is 'Production')."},
		{"port 0", "APP_PORT", "0", "APP_PORT must be a number from 1 to 65535 (it is '0')."},
		{"port above 65535", "APP_PORT", "65536", "APP_PORT must be a number from 1 to 65535 (it is '65536')."},
		{"port that is not a number", "APP_PORT", "eighty", "APP_PORT must be a number from 1 to 65535 (it is 'eighty')."},
		{"blank demo password", "DEMO_PASSWORD", "   ", "DEMO_PASSWORD can't be blank."},
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

func TestLoadConfig_PortLimitsAreAccepted(t *testing.T) {
	for _, port := range []string{"1", "65535"} {
		if _, err := app.LoadConfig(environment(map[string]string{"APP_PORT": port})); err != nil {
			t.Errorf("APP_PORT=%s: %v", port, err)
		}
	}
}

// Every invalid value is reported at once, so one start is enough to see what to fix.
func TestLoadConfig_ReportsEveryProblem(t *testing.T) {
	_, err := app.LoadConfig(environment(map[string]string{
		"APP_ENV":       "test",
		"APP_PORT":      "-1",
		"DEMO_PASSWORD": " ",
	}))

	want := []string{
		"APP_ENV must be local, development or production (it is 'test').",
		"APP_PORT must be a number from 1 to 65535 (it is '-1').",
		"DEMO_PASSWORD can't be blank.",
	}
	if err == nil || err.Error() != strings.Join(want, "\n") {
		t.Errorf("LoadConfig = %v, want the three messages, one per line", err)
	}
}

// Demo sign-in exists only in the local and development environments (FR-009).
func TestDemoMode_OnlyLocalAndDevelopment_FR009(t *testing.T) {
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

// environment returns a getenv function that reads from values instead of the real environment.
func environment(values map[string]string) func(string) string {
	return func(name string) string { return values[name] }
}
