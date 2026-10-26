package main

import (
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const apiRunning = "The MAPP API is running — stop it, then run the seed again."

func TestRun_RefusesInProduction_FR028(t *testing.T) {
	err := run(nil, environment(t, map[string]string{"APP_ENV": "production"}), io.Discard)

	want := "The demo data is never loaded when APP_ENV is production."
	if err == nil || err.Error() != want {
		t.Errorf("run = %v, want %q", err, want)
	}
}

func TestRun_RefusesWhileTheAPIIsRunning_FR028(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer api.Close()

	err := run([]string{"-reset"}, environment(t, map[string]string{"APP_PORT": portOf(api)}), io.Discard)

	if err == nil || err.Error() != apiRunning {
		t.Errorf("run = %v, want %q", err, apiRunning)
	}
}

// A single-page app answers /healthz with a page and 200 OK: the seed goes on, to the (closed) database.
func TestRun_IgnoresAnotherProgramOnThePort_FR028(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>Another app</title>"))
	}))
	defer other.Close()

	err := run(nil, environment(t, map[string]string{"APP_PORT": portOf(other)}), io.Discard)

	if err == nil || !strings.HasPrefix(err.Error(), "The database at DATABASE_URL cannot be created") {
		t.Errorf("run = %v, want the database's error", err)
	}
}

// A mistyped command such as "-reset mapp2", or two options that contradict each other, stop before the
// database is touched.
func TestRun_RefusesUnexpectedArguments(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"-reset", "mapp2"}, want: "Unexpected argument 'mapp2' — the options are -reset and -if-empty."},
		{args: []string{"-reset", "-if-empty"}, want: "Choose -reset or -if-empty, not both."},
	}
	for _, tt := range tests {
		err := run(tt.args, environment(t, nil), io.Discard)

		if err == nil || err.Error() != tt.want {
			t.Errorf("run(%q) = %v, want %q", tt.args, err, tt.want)
		}
	}
}

// environment points APP_PORT and DATABASE_URL at ports where nothing listens unless values says otherwise,
// so neither a developer's API nor their database changes a result.
func environment(t *testing.T, values map[string]string) func(string) string {
	t.Helper()
	all := map[string]string{
		"APP_PORT":     closedPort(t),
		"DATABASE_URL": "postgres://mapp@127.0.0.1:" + closedPort(t) + "/mapp?sslmode=disable&connect_timeout=1",
	}
	maps.Copy(all, values)
	return func(name string) string { return all[name] }
}

func closedPort(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("closing the listener: %v", err)
	}
	return strconv.Itoa(port)
}

func portOf(server *httptest.Server) string {
	return strconv.Itoa(server.Listener.Addr().(*net.TCPAddr).Port)
}
