// Command api runs the NorthStar API on http://127.0.0.1:<APP_PORT>. From the backend folder:
//
//	go run ./cmd/api
//
// It reads APP_ENV, APP_PORT, DATA_DIR and DEMO_PASSWORD (see app.LoadConfig), loads the data folder and
// serves until Ctrl+C, letting the requests in progress finish before it stops.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"northstar/internal/app"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/store"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run starts the API and serves until Ctrl+C. An error stops the program with its message.
func run() error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil))) // the server log (research R24)

	cfg, err := app.LoadConfig(os.Getenv)
	if err != nil {
		return err
	}
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

	interrupted, stopListening := signal.NotifyContext(context.Background(), os.Interrupt) // Ctrl+C
	defer stopListening()

	server := &http.Server{
		Handler:           api.Handler,
		ReadHeaderTimeout: 10 * time.Second, // a client that never finishes its headers cannot block a connection
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	fmt.Println(startupLine(cfg))

	select {
	case err := <-served:
		return err // the server stopped by itself
	case <-interrupted.Done():
	}
	// Stop accepting requests and give the ones in progress up to 10 seconds to finish.
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdown)
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

// startupLine is what the API prints once it is ready.
func startupLine(cfg app.Config) string {
	demoSignIn := "off"
	if cfg.DemoMode() {
		demoSignIn = "on"
	}
	return fmt.Sprintf("NorthStar API listening on http://127.0.0.1:%d (environment: %s, demo sign-in %s)",
		cfg.Port, cfg.Env, demoSignIn)
}
