package app

import (
	"cmp"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Config is the API's configuration, read from environment variables (research R21). There is no host
// setting: the API always listens on 127.0.0.1 only.
type Config struct {
	Env          string // APP_ENV: "local" (the default), "development" or "production"
	Port         int    // APP_PORT: 1 to 65535, default 8080
	DataDir      string // DATA_DIR: the folder of the JSON store, default "data"
	DemoPassword string // DEMO_PASSWORD: the password all demo people share, default "northstar"; never blank
}

// DemoMode reports whether demo sign-in and the developer pages are on: only in the local and
// development environments (FR-009).
func (c Config) DemoMode() bool {
	return c.Env == "local" || c.Env == "development"
}

// LoadConfig reads the configuration with getenv, which is os.Getenv when the API runs (tests pass their
// own). A variable that is not set takes its default. When values are invalid, the error lists every
// problem, one message per line.
func LoadConfig(getenv func(name string) string) (Config, error) {
	// cmp.Or returns the first of its values that is not empty.
	cfg := Config{
		Env:          cmp.Or(getenv("APP_ENV"), "local"),
		DataDir:      cmp.Or(getenv("DATA_DIR"), "data"),
		DemoPassword: cmp.Or(getenv("DEMO_PASSWORD"), "northstar"),
	}
	var problems []error

	switch cfg.Env {
	case "local", "development", "production":
	default:
		problems = append(problems, fmt.Errorf("APP_ENV must be local, development or production (it is '%s').", cfg.Env))
	}

	port := cmp.Or(getenv("APP_PORT"), "8080")
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		problems = append(problems, fmt.Errorf("APP_PORT must be a number from 1 to 65535 (it is '%s').", port))
	}
	cfg.Port = number

	if strings.TrimSpace(cfg.DemoPassword) == "" {
		problems = append(problems, errors.New("DEMO_PASSWORD can't be blank."))
	}

	if err := errors.Join(problems...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
