package app_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"mapp/internal/app"
)

// closeSignal tells the test when Shutdown has begun: it closes its listeners first.
type closeSignal struct {
	net.Listener
	once   sync.Once
	closed chan struct{}
}

func (l *closeSignal) Close() error {
	l.once.Do(func() { close(l.closed) })
	return l.Listener.Close()
}

// blockingHandler answers 204 once released, after telling the test the request has arrived.
type blockingHandler struct {
	arrived chan struct{}
	release chan struct{}
}

func (h blockingHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	close(h.arrived)
	<-h.release
	w.WriteHeader(http.StatusNoContent)
}

func TestServe_DrainsTheRequestInProgress_024_FR011(t *testing.T) {
	t.Parallel()
	listener := listenForTest(t)
	handler := blockingHandler{arrived: make(chan struct{}), release: make(chan struct{})}
	ctx, stop := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- app.Serve(ctx, listener, handler, time.Minute) }()
	answered := make(chan int, 1)
	go func() { answered <- get(t, listener) }()

	<-handler.arrived
	stop()
	<-listener.closed
	select {
	case err := <-served:
		t.Fatalf("Serve returned %v while a request was in progress", err)
	default:
	}
	close(handler.release)

	if status := <-answered; status != http.StatusNoContent {
		t.Errorf("the request in progress answered %d, want %d", status, http.StatusNoContent)
	}
	if err := <-served; err != nil {
		t.Errorf("Serve = %v, want nil after a graceful shutdown", err)
	}
}

func TestServe_DeadlinePassed_ReturnsError_024_FR011(t *testing.T) {
	t.Parallel()
	listener := listenForTest(t)
	handler := blockingHandler{arrived: make(chan struct{}), release: make(chan struct{})}
	ctx, stop := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- app.Serve(ctx, listener, handler, time.Millisecond) }()
	answered := make(chan int, 1)
	go func() { answered <- get(t, listener) }()

	<-handler.arrived
	stop()
	err := <-served
	close(handler.release)
	<-answered

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve = %v, want the shutdown deadline exceeded", err)
	}
}

func listenForTest(t *testing.T) *closeSignal {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return &closeSignal{Listener: listener, closed: make(chan struct{})}
}

// get returns the status of GET / on listener, or 0 when the request fails.
func get(t *testing.T, listener net.Listener) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
		"http://"+listener.Addr().String()+"/", nil)
	if err != nil {
		return 0
	}
	client := &http.Client{}
	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil {
		return 0
	}
	if err := response.Body.Close(); err != nil {
		return 0
	}
	return response.StatusCode
}
