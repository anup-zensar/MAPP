// Package apitest runs the real MAPP API inside a test, in memory on a copy of the demo data with a clock
// fixed at 12 Mar 2026 09:00 UTC, and checks every request and answer against backend/api/openapi.yaml.
package apitest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	validator "github.com/pb33f/libopenapi-validator"
	openapierrors "github.com/pb33f/libopenapi-validator/errors"

	"mapp/internal/app"
	"mapp/internal/auth"
	"mapp/internal/platform/fake"
	"mapp/internal/stores"
	"mapp/internal/stores/storestest"
)

var startTime = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

var errStorageDown = errors.New("apitest: the storage is down")

// Server is one test's API. Advance must not run while a request is being served.
type Server struct {
	handler      http.Handler
	set          stores.Set
	contract     validator.Validator
	now          time.Time
	demoPassword string
}

type Option func(*settings)

type settings struct {
	env            string
	db             *fake.DB
	withoutSeed    bool
	withoutHistory bool
	storageDown    bool
}

// WithEnv starts the API in env ("local", "development" or "production"); demo sign-in is off in production.
func WithEnv(env string) Option {
	return func(s *settings) { s.env = env }
}

// WithDB runs the API on db as it is, so two servers can share their data.
func WithDB(db *fake.DB) Option {
	return func(s *settings) { s.db = db }
}

func WithStorageDown() Option {
	return func(s *settings) { s.storageDown = true }
}

func WithoutSeed() Option {
	return func(s *settings) { s.withoutSeed = true }
}

func WithoutDemoHistory() Option {
	return func(s *settings) { s.withoutHistory = true }
}

func New(t testing.TB, opts ...Option) *Server {
	t.Helper()
	s := settings{env: "local"}
	for _, opt := range opts {
		opt(&s)
	}
	variables := map[string]string{"APP_ENV": s.env}
	cfg, err := app.LoadConfig(func(name string) string { return variables[name] })
	if err != nil {
		t.Fatalf("apitest: configuration: %v", err)
	}
	set := storestest.New(s.dataOf(t))
	if s.storageDown {
		set.Ready = func(context.Context) error { return errStorageDown }
	}
	server := &Server{set: set, contract: newContractValidator(t), now: startTime, demoPassword: cfg.DemoPassword}
	built, err := app.New(cfg, set, server.clock)
	if err != nil {
		t.Fatalf("apitest: starting the API: %v", err)
	}
	server.handler = built.Handler
	return server
}

func (s settings) dataOf(tb testing.TB) *fake.DB {
	tb.Helper()
	switch {
	case s.db != nil:
		return s.db
	case s.withoutSeed:
		return fake.NewDB()
	}
	return DemoDB(tb, s.withoutHistory)
}

// The contract validator is built once per test binary and shared by every test server (contract.go).
var (
	contractOnce    sync.Once
	sharedContract  validator.Validator
	contractFailure error
)

func newContractValidator(t testing.TB) validator.Validator {
	t.Helper()
	contractOnce.Do(func() { sharedContract, contractFailure = buildContract() })
	if contractFailure != nil {
		t.Fatalf("apitest: %v", contractFailure)
	}
	return sharedContract
}

func (s *Server) clock() time.Time {
	return s.now
}

func (s *Server) Advance(d time.Duration) {
	s.now = s.now.Add(d)
}

type Client struct {
	server *Server
	cookie *http.Cookie
}

func (s *Server) SignIn(t testing.TB, email string) *Client {
	t.Helper()
	response := s.Do(t, http.MethodPost, "/api/v1/auth/sign-in",
		map[string]string{"email": email, "password": s.demoPassword})
	cookie := response.Cookie(auth.CookieName)
	if response.Status != http.StatusOK || cookie == nil {
		t.Fatalf("apitest: signing in as %s: %d %s", email, response.Status, response.Body)
	}
	return &Client{server: s, cookie: cookie}
}

func (c *Client) Do(t testing.TB, method, path string, body any, opts ...RequestOption) *Response {
	t.Helper()
	withSession := append([]RequestOption{WithCookie(c.cookie)}, opts...)
	return c.server.Do(t, method, path, body, withSession...)
}

type RequestOption func(*request)

type request struct {
	skipRequestValidation bool
	skipValidation        bool
	rawBody               []byte
	host                  string
	cookies               []*http.Cookie
}

// SkipRequestValidation sends a request that breaks the contract on purpose; the answer is still checked.
func SkipRequestValidation() RequestOption {
	return func(r *request) { r.skipRequestValidation = true }
}

func SkipValidation() RequestOption {
	return func(r *request) { r.skipValidation = true }
}

func RawBody(body []byte) RequestOption {
	return func(r *request) { r.rawBody = body }
}

func WithCookie(c *http.Cookie) RequestOption {
	return func(r *request) { r.cookies = append(r.cookies, c) }
}

func WithHost(host string) RequestOption {
	return func(r *request) { r.host = host }
}

type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Do sends a request, body encoded as JSON, and reports a request or answer that breaks the contract with
// t.Errorf, so the test goes on and can report more.
func (s *Server) Do(t testing.TB, method, path string, body any, opts ...RequestOption) *Response {
	t.Helper()
	options := request{host: "127.0.0.1"}
	for _, opt := range opts {
		opt(&options)
	}
	payload := options.rawBody
	if payload == nil && body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatalf("apitest: encoding the request body: %v", err)
		}
	}
	if !options.skipValidation && !options.skipRequestValidation {
		// A request of its own: the validator reads the body.
		if valid, problems := s.contract.ValidateHttpRequest(newRequest(method, path, payload, options)); !valid {
			t.Errorf("apitest: %s %s: the request does not match the contract:%s", method, path, describe(problems))
		}
	}

	sent := newRequest(method, path, payload, options)
	recorder := httptest.NewRecorder()
	s.handler.ServeHTTP(recorder, sent)
	answer := recorder.Result()
	answerBody, err := io.ReadAll(answer.Body)
	if err != nil {
		t.Fatalf("apitest: reading the answer: %v", err)
	}
	if !options.skipValidation {
		answer.Body = io.NopCloser(bytes.NewReader(answerBody))
		if valid, problems := s.contract.ValidateHttpResponse(sent, answer); !valid {
			t.Errorf("apitest: %s %s: the answer %d %s does not match the contract:%s",
				method, path, answer.StatusCode, answerBody, describe(problems))
		}
	}
	return &Response{Status: answer.StatusCode, Header: answer.Header, Body: answerBody}
}

func newRequest(method, path string, payload []byte, options request) *http.Request {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	r := httptest.NewRequest(method, "http://"+options.host+path, body)
	if payload != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range options.cookies {
		r.AddCookie(cookie)
	}
	return r
}

func describe(problems []*openapierrors.ValidationError) string {
	var text strings.Builder
	for _, problem := range problems {
		text.WriteString("\n  - " + problem.Error())
	}
	return text.String()
}

func (r *Response) Cookie(name string) *http.Cookie {
	for _, line := range r.Header.Values("Set-Cookie") {
		if cookie, err := http.ParseSetCookie(line); err == nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func (r *Response) Decode(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("apitest: the answer %s does not fit %T: %v", r.Body, v, err)
	}
}
