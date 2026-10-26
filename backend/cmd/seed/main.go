// Command seed creates and migrates the database at DATABASE_URL and loads MAPP's demo data into it (FR-028):
// go run ./cmd/seed, -reset to empty every table first, or -if-empty to leave a database with data alone.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"mapp/internal/app"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/database"
	"mapp/internal/stores"
	"mapp/seed"
)

// message is an error the person at the terminal reads as it is: a sentence, unlike Go's error strings.
type message string

func (m message) Error() string {
	return string(m)
}

type options struct {
	reset   bool
	ifEmpty bool
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, out io.Writer) error {
	cfg, err := app.LoadConfig(getenv)
	if err != nil {
		return err
	}
	opts, err := parseArgs(args)
	if err != nil {
		return err
	}
	if err = refuse(cfg); err != nil {
		return err
	}
	loaded, err := load(context.Background(), cfg.DatabaseURL, opts)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, loaded)
	return err
}

func parseArgs(args []string) (options, error) {
	flags := flag.NewFlagSet("seed", flag.ExitOnError)
	reset := flags.Bool("reset", false, "empty every table first (this deletes the current data)")
	ifEmpty := flags.Bool("if-empty", false, "leave a database that already has data as it is, and succeed")
	_ = flags.Parse(args) // ExitOnError: a wrong option stops the program with a description of the options
	if flags.NArg() > 0 {
		return options{}, message(fmt.Sprintf("Unexpected argument '%s' — the options are -reset and -if-empty.", flags.Arg(0)))
	}
	if *reset && *ifEmpty {
		return options{}, message("Choose -reset or -if-empty, not both.")
	}
	return options{reset: *reset, ifEmpty: *ifEmpty}, nil
}

func refuse(cfg app.Config) error {
	if cfg.Env == "production" {
		return message("The demo data is never loaded when APP_ENV is production.")
	}
	if apiIsRunning(cfg.Port) {
		return message("The MAPP API is running — stop it, then run the seed again.")
	}
	return nil
}

func load(ctx context.Context, url string, opts options) (string, error) {
	db, err := prepare(ctx, url, opts.reset)
	if err != nil {
		return "", err
	}
	defer db.Close()
	err = seed.Load(ctx, stores.Postgres(db), clock.System)
	switch {
	case errors.Is(err, seed.ErrNotEmpty) && opts.ifEmpty:
		return "The database already has data; it was left as it is.", nil
	case errors.Is(err, seed.ErrNotEmpty):
		return "", message("The database already has data. Run go run ./cmd/seed -reset to replace it (this deletes the current data).")
	case err != nil:
		return "", err
	}
	return fmt.Sprintf("Loaded the demo data: %s.", seed.Loaded), nil
}

func prepare(ctx context.Context, url string, reset bool) (*database.DB, error) {
	if err := database.CreateIfMissing(ctx, url); err != nil {
		return nil, message(fmt.Sprintf("The database at DATABASE_URL cannot be created (%v) — start PostgreSQL, "+
			"with a role that may create databases.", err))
	}
	db, err := database.Open(ctx, url)
	if err != nil {
		return nil, err
	}
	err = database.Migrate(ctx, db)
	if err == nil && reset {
		err = database.Empty(ctx, db)
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// apiIsRunning reports whether a MAPP API answers /healthz on port with 204; another program does not count.
func apiIsRunning(port int) bool {
	client := http.Client{Timeout: time.Second}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/healthz", port), nil)
	if err != nil {
		return false
	}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer func() { _ = response.Body.Close() }()
	return response.StatusCode == http.StatusNoContent
}
