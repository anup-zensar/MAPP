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
	"mapp/internal/platform/logging"
	"mapp/internal/platform/store"
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
	db, err := store.Open(cfg.DataDir)
	if err != nil {
		return err
	}
	api, err := app.New(cfg, db, clock.System)
	if err != nil {
		return err
	}
	listener, err := listen(cfg.Port)
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

// listen opens port on 127.0.0.1 only, so that other computers cannot reach the API.
func listen(port int) (net.Listener, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
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
