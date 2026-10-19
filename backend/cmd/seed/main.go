// Command seed loads NorthStar's demo data into an empty data folder (FR-028, research R22). From the
// backend folder, with the API stopped:
//
//	go run ./cmd/seed                   # into DATA_DIR, or the folder "data"
//	go run ./cmd/seed -data <folder>    # into another folder
//	go run ./cmd/seed -reset            # delete the current data first
//
// The API reads its data folder only when it starts and would later overwrite the new files from its
// memory, so the seed refuses to run while an API answers on APP_PORT. It never runs in production.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"northstar/internal/app"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/store"
	"northstar/seed"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run loads the demo data as the command-line arguments args and the environment (read with getenv) ask,
// and prints what it did to out. It refuses, before changing anything, in production, while the API is
// running, and when the folder already has data and -reset was not given.
func run(args []string, getenv func(string) string, out io.Writer) error {
	cfg, err := app.LoadConfig(getenv) // the API's own settings: APP_ENV, APP_PORT and DATA_DIR
	if err != nil {
		return err
	}

	// A wrong option stops the program with a description of the options, as Go's own tools do.
	flags := flag.NewFlagSet("seed", flag.ExitOnError)
	dataDir := flags.String("data", cfg.DataDir, "the data folder to load the demo data into")
	reset := flags.Bool("reset", false, "delete the current data first (the folder's *.json, *.jsonl and *.tmp files)")
	flags.Parse(args) // returns only when the options are right
	if flags.NArg() > 0 {
		return fmt.Errorf("Unexpected argument '%s' — the options are -data <folder> and -reset.", flags.Arg(0))
	}

	if cfg.Env == "production" {
		return errors.New("The demo data is never loaded when APP_ENV is production.")
	}
	if apiIsRunning(cfg.Port) {
		return errors.New("The NorthStar API is running — stop it, then run the seed again.")
	}
	if *reset {
		if err := deleteData(*dataDir); err != nil {
			return err
		}
	}

	db, err := store.Open(*dataDir) // creates the folder when it is missing
	if err != nil {
		return err
	}
	err = seed.Load(context.Background(), db, clock.System)
	if errors.Is(err, seed.ErrNotEmpty) {
		return fmt.Errorf("The data folder %s already has data. Run go run ./cmd/seed -reset to replace it (this deletes the current data).", *dataDir)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Loaded the demo data into %s: 6 people.\n", *dataDir)
	return nil
}

// apiIsRunning reports whether a NorthStar API answers GET /api/health on port within one second: 200 OK
// with its health answer, {"status": "ok", ...}. Another program that happens to use the port does not
// count, whatever it answers — an error, or a web page with 200 OK.
func apiIsRunning(port int) bool {
	client := http.Client{Timeout: time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	if err != nil {
		return false // nothing answered
	}
	defer response.Body.Close()
	var health struct {
		Status string `json:"status"`
	}
	return response.StatusCode == http.StatusOK &&
		json.NewDecoder(response.Body).Decode(&health) == nil && health.Status == "ok"
}

// deleteData deletes the data files in the folder dir: its *.json, *.jsonl and *.tmp files. Other files
// and folders stay. A folder that does not exist has nothing to delete.
func deleteData(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && isDataFile(entry.Name()) {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// isDataFile reports whether name is a file of the JSON store: a table (.json), an append-only table
// (.jsonl) or the temporary file of an interrupted save (.tmp).
func isDataFile(name string) bool {
	switch filepath.Ext(name) {
	case ".json", ".jsonl", ".tmp":
		return true
	}
	return false
}
