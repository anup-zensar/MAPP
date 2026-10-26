package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Without these timeouts a slow or malicious client could hold a connection open forever.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

// Serve serves until ctx ends, then lets the requests in progress finish within shutdownTimeout, or fails.
func Serve(ctx context.Context, listener net.Listener, handler http.Handler, shutdownTimeout time.Duration) error {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(slog.Default().Handler(), slog.LevelError),
	}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()

	select {
	case err := <-served:
		return fmt.Errorf("serve on %s: %w", listener.Addr(), err)
	case <-ctx.Done():
	}
	if err := shutdown(ctx, server, shutdownTimeout); err != nil {
		return err
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve on %s: %w", listener.Addr(), err)
	}
	return nil
}

// shutdown starts after ctx has ended, so it keeps the context's values but not its cancellation.
func shutdown(ctx context.Context, server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shut down http server: %w", err)
	}
	return nil
}
