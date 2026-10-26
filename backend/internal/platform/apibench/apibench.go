// Package apibench serves the real API on PostgreSQL holding the volume of the specs' scale targets, and
// measures the statements and the time each request takes. Only benchmarks import it.
package apibench

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mapp/internal/app"
	"mapp/internal/auth"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/dbtest"
	"mapp/internal/platform/querycount"
	"mapp/internal/stores"
)

const (
	alex  = "alex.morgan@shell.com"
	priya = "priya.raman@shell.com"
)

// Data is what a benchmark database holds: the demo data alone, or the volume of the scale targets as well.
type Data string

const (
	Demo   Data = "demo"
	Volume Data = "volume"
)

type Server struct {
	handler  http.Handler
	password string
	set      stores.Set
}

type Client struct {
	server *Server
	cookie *http.Cookie
}

type Response struct {
	Status  int
	Body    []byte
	Queries int64
}

type request struct {
	method, path string
	body         any
	cookie       *http.Cookie
}

// New serves the API for tb on its own copy of data's database, built on the first run.
func New(tb testing.TB, data Data) *Server {
	tb.Helper()
	slog.SetDefault(slog.New(slog.DiscardHandler))
	fill := loadDemo
	if data == Volume {
		fill = loadVolume
	}
	return serve(tb, dbtest.NewFilledDB(tb, string(data)+volumeVersion, func(db *database.DB) { fill(tb, db) }))
}

func serve(tb testing.TB, db *database.DB) *Server {
	tb.Helper()
	cfg, err := app.LoadConfig(func(string) string { return "" })
	if err != nil {
		tb.Fatalf("apibench: configuration: %v", err)
	}
	set := stores.Postgres(db)
	built, err := app.New(cfg, set, startTime)
	if err != nil {
		tb.Fatalf("apibench: starting the API: %v", err)
	}
	return &Server{handler: built.Handler, password: cfg.DemoPassword, set: set}
}

// startTime is the fixed time of every server: the seed's records and the volume are dated by it.
func startTime() time.Time {
	return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
}

func (s *Server) SignIn(tb testing.TB, email string) *Client {
	tb.Helper()
	sent := map[string]string{"email": email, "password": s.password}
	answer := s.do(tb.Context(), request{method: http.MethodPost, path: "/api/v1/auth/sign-in", body: sent})
	for _, line := range answer.Header().Values("Set-Cookie") {
		if cookie, err := http.ParseSetCookie(line); err == nil && cookie.Name == auth.CookieName {
			return &Client{server: s, cookie: cookie}
		}
	}
	tb.Fatalf("apibench: signing in as %s: %d %s", email, answer.Code, answer.Body)
	return nil
}

// Do sends one request; an answer outside 2xx stops tb.
func (c *Client) Do(tb testing.TB, method, path string, body any) Response {
	tb.Helper()
	answer := c.Send(tb, method, path, body)
	if answer.Status >= http.StatusMultipleChoices {
		tb.Fatalf("apibench: %s %s: %d %s", method, path, answer.Status, answer.Body)
	}
	return answer
}

// Send sends one request and returns the answer, whatever its status.
func (c *Client) Send(tb testing.TB, method, path string, body any) Response {
	tb.Helper()
	ctx := querycount.With(tb.Context())
	answer := c.server.do(ctx, request{method: method, path: path, body: body, cookie: c.cookie})
	got, err := io.ReadAll(answer.Body)
	if err != nil {
		tb.Fatalf("apibench: %s %s: reading the answer: %v", method, path, err)
	}
	return Response{Status: answer.Code, Body: got, Queries: querycount.Of(ctx)}
}

// Measure sends the request for every iteration of b and reports the statements per request beside the time.
func (c *Client) Measure(b *testing.B, method, path string, body any) {
	b.Helper()
	var queries int64
	for b.Loop() {
		queries += c.Do(b, method, path, body).Queries
	}
	b.ReportMetric(float64(queries)/float64(b.N), "queries/op")
}

func (s *Server) do(ctx context.Context, r request) *httptest.ResponseRecorder {
	var payload io.Reader
	if r.body != nil {
		encoded, err := json.Marshal(r.body)
		if err != nil {
			panic("apibench: a request body that cannot be encoded: " + err.Error())
		}
		payload = bytes.NewReader(encoded)
	}
	sent := httptest.NewRequestWithContext(ctx, r.method, "http://127.0.0.1"+r.path, payload)
	if r.body != nil {
		sent.Header.Set("Content-Type", "application/json")
	}
	if r.cookie != nil {
		sent.AddCookie(r.cookie)
	}
	w := httptest.NewRecorder()
	s.handler.ServeHTTP(w, sent)
	return w
}

// Decode reads the answer's JSON body into v; a body that does not fit stops tb.
func (r Response) Decode(tb testing.TB, v any) {
	tb.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		tb.Fatalf("apibench: decoding %s: %v", r.Body, err)
	}
}
