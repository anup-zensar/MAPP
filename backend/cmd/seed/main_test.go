package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"northstar/internal/people"
	"northstar/internal/platform/store"
	"northstar/seed"
)

// The seed loads the demo data into an empty data folder, creating the folder when it is missing, and says
// what it loaded (FR-028).
func TestRun_LoadsTheDemoData_FR028(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data") // does not exist yet
	var out bytes.Buffer

	err := run([]string{"-data", dir}, environment(t, nil), &out)

	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if want := "Loaded the demo data into " + dir + ": " + seed.Loaded + ".\n"; out.String() != want {
		t.Errorf("printed %q, want %q", out.String(), want)
	}
	if got := len(storedPeople(t, dir)); got != 6 {
		t.Errorf("%d people stored, want 6", got)
	}
}

// Without -data the seed uses DATA_DIR, the folder the API uses (FR-028).
func TestRun_UsesDataDirByDefault_FR028(t *testing.T) {
	dir := t.TempDir()

	err := run(nil, environment(t, map[string]string{"DATA_DIR": dir}), io.Discard)

	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := len(storedPeople(t, dir)); got != 6 {
		t.Errorf("%d people stored in DATA_DIR, want 6", got)
	}
}

// A folder that already has data is refused without -reset, and nothing changes (FR-028).
func TestRun_RefusesAFolderWithData_FR028(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sample_folders.json", `{"SF-1001": {"id": "SF-1001"}}`)

	err := run([]string{"-data", dir}, environment(t, nil), io.Discard)

	want := "The data folder " + dir + " already has data. Run go run ./cmd/seed -reset to replace it (this deletes the current data)."
	if err == nil || err.Error() != want {
		t.Errorf("run = %v, want %q", err, want)
	}
	if got := fileNames(t, dir); !slices.Equal(got, []string{"sample_folders.json"}) {
		t.Errorf("files = %q, want only sample_folders.json", got)
	}
}

// -reset deletes only the data files (*.json, *.jsonl and *.tmp), then loads the demo data (FR-028).
func TestRun_ResetDeletesOnlyDataFiles_FR028(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sample_folders.json", `{}`)
	writeFile(t, dir, "activity_log.jsonl", "")
	writeFile(t, dir, "people.json.tmp", "half written")
	writeFile(t, dir, "notes.txt", "not a data file")
	if err := os.Mkdir(filepath.Join(dir, "backup.json"), 0o755); err != nil { // a folder, not a file
		t.Fatal(err)
	}

	err := run([]string{"-data", dir, "-reset"}, environment(t, nil), io.Discard)

	if err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{
		"activities.json", "activity_log.jsonl", "activity_types.json", "approval_actions.json", "approval_rounds.json",
		"approval_workflows.json", "backup.json", "campaigns.json", "counters.json", "currencies.json",
		"finance_budget_items.json", "finance_budgets.json", "initiatives.json", "line_items.json", "models.json", "notes.txt",
		"notifications.json", "people.json", "plans.json", "recycle_bin.json", "saved_views.json",
		"shell_attribute_options.json", "shell_attributes.json", "team_memberships.json", "teams.json",
	}
	if got := fileNames(t, dir); !slices.Equal(got, want) {
		t.Errorf("files = %q, want %q", got, want)
	}
}

// The demo data is never loaded in production, and nothing is created (FR-028).
func TestRun_RefusesInProduction_FR028(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")

	err := run([]string{"-data", dir}, environment(t, map[string]string{"APP_ENV": "production"}), io.Discard)

	want := "The demo data is never loaded when APP_ENV is production."
	if err == nil || err.Error() != want {
		t.Errorf("run = %v, want %q", err, want)
	}
	if _, statErr := os.Stat(dir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("the data folder was created (%v)", statErr)
	}
}

// While an API answers on APP_PORT the seed refuses before it deletes or loads anything, because the
// running API would overwrite the new files from its memory (FR-028, research R22).
func TestRun_RefusesWhileTheAPIIsRunning_FR028(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/health" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","environment":"local","demoSignIn":true}`))
	}))
	defer api.Close()
	port := strconv.Itoa(api.Listener.Addr().(*net.TCPAddr).Port)
	dir := t.TempDir()
	writeFile(t, dir, "sample_folders.json", `{}`)

	err := run([]string{"-data", dir, "-reset"}, environment(t, map[string]string{"APP_PORT": port}), io.Discard)

	want := "The NorthStar API is running — stop it, then run the seed again."
	if err == nil || err.Error() != want {
		t.Errorf("run = %v, want %q", err, want)
	}
	if got := fileNames(t, dir); !slices.Equal(got, []string{"sample_folders.json"}) {
		t.Errorf("files = %q, want sample_folders.json untouched", got)
	}
}

// Another program on APP_PORT that answers /api/health with a web page and 200 OK, as a single-page app
// does for every path, is not a NorthStar API: the seed goes on (FR-028).
func TestRun_IgnoresAnotherProgramOnThePort_FR028(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>Another app</title>"))
	}))
	defer other.Close()
	port := strconv.Itoa(other.Listener.Addr().(*net.TCPAddr).Port)
	dir := t.TempDir()

	err := run([]string{"-data", dir}, environment(t, map[string]string{"APP_PORT": port}), io.Discard)

	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if got := len(storedPeople(t, dir)); got != 6 {
		t.Errorf("%d people stored, want 6", got)
	}
}

// An argument that is not an option stops the seed before anything is deleted, so a mistyped command such
// as "-reset data2" never resets the default folder.
func TestRun_RefusesUnexpectedArguments(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "sample_folders.json", `{}`)

	err := run([]string{"-reset", "data2"}, environment(t, map[string]string{"DATA_DIR": dir}), io.Discard)

	want := "Unexpected argument 'data2' — the options are -data <folder> and -reset."
	if err == nil || err.Error() != want {
		t.Errorf("run = %v, want %q", err, want)
	}
	if got := fileNames(t, dir); !slices.Equal(got, []string{"sample_folders.json"}) {
		t.Errorf("files = %q, want sample_folders.json untouched", got)
	}
}

// environment returns a getenv function that reads values. APP_PORT is a port where nothing listens unless
// values sets it, so an API running on this computer, such as the developer's own, never changes a result.
func environment(t *testing.T, values map[string]string) func(string) string {
	t.Helper()
	all := map[string]string{"APP_PORT": closedPort(t)}
	maps.Copy(all, values)
	return func(name string) string { return all[name] }
}

// closedPort returns a port on 127.0.0.1 where nothing listens: one the system just gave out and took back.
func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("closing the listener: %v", err)
	}
	return strconv.Itoa(port)
}

// storedPeople returns the people stored in the data folder dir.
func storedPeople(t *testing.T, dir string) []people.Person {
	t.Helper()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	var list []people.Person
	err = db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		list, err = people.NewJSONRepository().List(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	return list
}

// fileNames returns the names in the folder dir, sorted.
func fileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir: %v", err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names // os.ReadDir sorts by name
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
