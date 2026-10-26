// Command api runs the MAPP API on http://127.0.0.1:<APP_PORT>. From the backend folder:
//
//	go run ./cmd/api
//
// It reads its settings from the environment (see app.LoadConfig), writes its log as JSON lines on standard
// output and serves until Ctrl+C or SIGTERM, letting the requests in progress finish before it stops.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mapp/internal/app"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/database"
	"mapp/internal/platform/logging"
	"mapp/internal/stores"
)

func main() {
	slog.SetDefault(logging.New(os.Stdout, slog.LevelInfo))
	if err := run(context.Background()); err != nil {
		slog.Error("MAPP API stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := app.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
	slog.SetDefault(logging.New(os.Stdout, cfg.LogLevel))
	db, err := openDatabase(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	api, err := app.New(cfg, stores.Postgres(db), clock.System)
	if err != nil {
		return err
	}
	listener, err := listen(cfg.Host, cfg.Port)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "MAPP API listening", "url", "http://"+listener.Addr().String(),
		"environment", cfg.Env, "demo_sign_in", cfg.DemoMode())
	served := app.Serve(ctx, listener, api.Handler, cfg.ShutdownTimeout)
	closing, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	return errors.Join(served, api.Close(closing))
}

type message string

func (m message) Error() string {
	return string(m)
}

func openDatabase(ctx context.Context, url string) (*database.DB, error) {
	db, err := database.Open(ctx, url)
	if err != nil {
		return nil, message(fmt.Sprintf("The database at DATABASE_URL cannot be reached (%v) — start PostgreSQL, "+
			"then run go run ./cmd/seed.", err))
	}
	pending, err := database.HasPending(ctx, db)
	if err == nil && pending {
		err = message("The database is behind the latest migration — run go run ./cmd/seed first.")
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// listen opens port on host: 127.0.0.1, so that other computers cannot reach the API, or 0.0.0.0 inside the
// container, whose port compose publishes on 127.0.0.1 only.
func listen(host string, port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if isPortInUse(err) {
		return nil, fmt.Errorf("Port %d is already in use — stop the other program or set APP_PORT.", port)
	}
	return listener, err
}

// Windows' error numbers for a port that another program holds: WSAEADDRINUSE (in use), and WSAEACCES
// (another program listens on the port on all addresses, or Windows has reserved the port). On Windows,
// Go's syscall.EADDRINUSE matches neither.
const (
	windowsAddressInUse = 10048
	windowsAccessDenied = 10013
)

// isPortInUse reports whether err says that another program, or Windows, already holds the port.
func isPortInUse(err error) bool {
	var errno syscall.Errno
	return errors.Is(err, syscall.EADDRINUSE) ||
		(errors.As(err, &errno) && (errno == windowsAddressInUse || errno == windowsAccessDenied))
}
