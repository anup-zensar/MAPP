package app

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"mapp/internal/platform/telemetry"
)

const (
	envLocal       = "local"
	envDevelopment = "development"
	envProduction  = "production"

	defaultDatabaseURL     = "postgres://mapp@127.0.0.1:5432/mapp?sslmode=disable"
	defaultLogLevel        = "info"
	defaultShutdownTimeout = "10s"
	minShutdownTimeout     = time.Second
	maxShutdownTimeout     = time.Minute
	maxPort                = 65535
)

// Config is the API's configuration, read from environment variables. Host is 127.0.0.1 except in the
// container image, which listens on 0.0.0.0 inside the container and is published on 127.0.0.1 only.
type Config struct {
	Env             string
	Host            string
	Port            int
	DatabaseURL     string
	DemoPassword    string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	Telemetry       telemetry.Config
	TrustedProxies  []string
}

type reader struct {
	getenv   func(name string) string
	problems []string
}

// LoadConfig reads the configuration with getenv; a variable that is not set takes its default. When values
// are invalid, the error lists every problem, one message per line, as the person starting the API reads it.
func LoadConfig(getenv func(name string) string) (Config, error) {
	env := &reader{getenv: getenv}
	cfg := Config{
		Env:             env.choice("APP_ENV", envLocal, envLocal, envDevelopment, envProduction),
		Host:            env.choice("APP_HOST", "127.0.0.1", "127.0.0.1", "0.0.0.0"),
		Port:            env.port("APP_PORT", "8080"),
		DatabaseURL:     env.text("DATABASE_URL", defaultDatabaseURL),
		DemoPassword:    env.notBlank("DEMO_PASSWORD", "mapp"),
		LogLevel:        env.logLevel("LOG_LEVEL"),
		ShutdownTimeout: env.shutdownTimeout("SHUTDOWN_TIMEOUT"),
		Telemetry: telemetry.Config{
			Traces:  env.exporter("OTEL_TRACES_EXPORTER"),
			Metrics: env.exporter("OTEL_METRICS_EXPORTER"),
		},
		TrustedProxies: env.proxies("TRUSTED_PROXIES"),
	}
	if len(env.problems) > 0 {
		return Config{}, errors.New(strings.Join(env.problems, "\n"))
	}
	return cfg, nil
}

// DemoMode reports whether demo sign-in and the developer pages are on: only in the local and
// development environments.
func (c Config) DemoMode() bool {
	return c.Env == envLocal || c.Env == envDevelopment
}

func (r *reader) text(name, fallback string) string {
	return cmp.Or(r.getenv(name), fallback)
}

func (r *reader) problem(format string, args ...any) {
	r.problems = append(r.problems, fmt.Sprintf(format, args...))
}

func (r *reader) choice(name, fallback string, allowed ...string) string {
	value := r.text(name, fallback)
	if !slices.Contains(allowed, value) {
		last := len(allowed) - 1
		r.problem("%s must be %s or %s (it is '%s').", name, strings.Join(allowed[:last], ", "), allowed[last], value)
	}
	return value
}

func (r *reader) port(name, fallback string) int {
	value := r.text(name, fallback)
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || number > maxPort {
		r.problem("%s must be a number from 1 to %d (it is '%s').", name, maxPort, value)
	}
	return number
}

func (r *reader) notBlank(name, fallback string) string {
	value := r.text(name, fallback)
	if strings.TrimSpace(value) == "" {
		r.problem("%s can't be blank.", name)
	}
	return value
}

func (r *reader) logLevel(name string) slog.Level {
	value := r.text(name, defaultLogLevel)
	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		r.problem("%s must be debug, info, warn or error (it is '%s').", name, value)
	}
	return level
}

func (r *reader) shutdownTimeout(name string) time.Duration {
	value := r.text(name, defaultShutdownTimeout)
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout < minShutdownTimeout || timeout > maxShutdownTimeout {
		r.problem("%s must be a duration from 1s to 60s, such as 10s (it is '%s').", name, value)
	}
	return timeout
}

func (r *reader) proxies(name string) []string {
	value := r.getenv(name)
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var list []string
	for part := range strings.SplitSeq(value, ",") {
		part = strings.TrimSpace(part)
		_, _, cidrErr := net.ParseCIDR(part)
		if net.ParseIP(part) == nil && cidrErr != nil {
			r.problem("%s must list IP addresses or CIDR ranges separated by commas (it is '%s').", name, value)
			return nil
		}
		list = append(list, part)
	}
	return list
}

func (r *reader) exporter(name string) telemetry.Exporter {
	return telemetry.Exporter(r.choice(name, string(telemetry.None), string(telemetry.None), string(telemetry.Console)))
}
