// Package apitest runs the real NorthStar API inside a test and checks every request and every answer
// against the contract, backend/api/openapi.yaml (research R11). Every endpoint gets a contract test
// written with it (constitution V):
//
//	server := apitest.New(t)
//	response := server.Do(t, http.MethodGet, "/api/health", nil)
//	if response.Status != http.StatusOK { ... }
//
//	priya := server.SignIn(t, "priya.raman@northstar.demo")
//	response = priya.Do(t, http.MethodGet, "/api/me", nil) // sent with Priya's session cookie
//
// The API runs on a temporary data folder holding the demo people of the seed, with a fixed clock that
// starts at 12 Mar 2026 09:00 UTC and moves only with Advance. Requests are handed to the API directly,
// without a network port.
//
// Only external test packages (package <name>_test) import apitest, because it builds the whole API.
package apitest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/pb33f/libopenapi-validator/config"
	openapierrors "github.com/pb33f/libopenapi-validator/errors"

	"northstar/api"
	"northstar/internal/app"
	"northstar/internal/auth"
	"northstar/internal/platform/store"
	"northstar/seed"
)

// startTime is where the clock of every test server starts: 12 Mar 2026 09:00 UTC.
var startTime = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// Server is the real API, built by app.New, for one test. It is not meant for parallel subtests: Advance
// must not run while a request is being served.
type Server struct {
	handler      http.Handler
	db           *store.DB // the data folder, for GrantRole
	contract     validator.Validator
	now          time.Time
	demoPassword string // what SignIn sends
}

// Option changes how New builds the server.
type Option func(*settings)

type settings struct {
	env         string // APP_ENV
	dataDir     string // empty: a new temporary folder
	withoutSeed bool   // do not load the demo people
}

// WithEnv starts the API in the environment env ("local", "development" or "production") instead of
// "local". Demo sign-in is off in "production".
func WithEnv(env string) Option {
	return func(s *settings) { s.env = env }
}

// WithDataDir starts the API on dir instead of a new temporary folder, so that two servers can share one
// data folder (for example a session made in "local" and tried in "production").
func WithDataDir(dir string) Option {
	return func(s *settings) { s.dataDir = dir }
}

// WithoutSeed starts the API without the demo people: an empty data folder, as before the demo data is
// loaded.
func WithoutSeed() Option {
	return func(s *settings) { s.withoutSeed = true }
}

// New builds the API for test t. The data folder starts with the demo people of the seed, created at the
// clock's start time, unless WithoutSeed is given or the folder already has data (a folder shared through
// WithDataDir is used as it is). It stops the test when the API cannot start.
func New(t testing.TB, opts ...Option) *Server {
	t.Helper()
	s := settings{env: "local"}
	for _, opt := range opts {
		opt(&s)
	}
	if s.dataDir == "" {
		s.dataDir = t.TempDir()
	}

	// The API's own configuration code, with these environment variables.
	variables := map[string]string{"APP_ENV": s.env, "DATA_DIR": s.dataDir}
	cfg, err := app.LoadConfig(func(name string) string { return variables[name] })
	if err != nil {
		t.Fatalf("apitest: configuration: %v", err)
	}
	db, err := store.Open(s.dataDir)
	if err != nil {
		t.Fatalf("apitest: opening the data folder: %v", err)
	}

	server := &Server{db: db, contract: newContractValidator(t), now: startTime, demoPassword: cfg.DemoPassword}
	if !s.withoutSeed {
		err := seed.Load(t.Context(), db, server.clock)
		if err != nil && !errors.Is(err, seed.ErrNotEmpty) { // a folder with data is used as it is
			t.Fatalf("apitest: loading the demo data: %v", err)
		}
	}
	built, err := app.New(cfg, db, server.clock)
	if err != nil {
		t.Fatalf("apitest: starting the API: %v", err)
	}
	server.handler = built.Handler
	return server
}

// The contract validator is built once per test binary and shared by every test server: building it
// from the whole contract takes seconds, and no test runs in parallel with another.
var (
	contractOnce    sync.Once
	sharedContract  validator.Validator
	contractFailure error
)

// newContractValidator returns the validator for backend/api/openapi.yaml. Format assertions are on, so
// that a "date-time" really is one; security is not checked here, because the API's own sign-in
// middleware enforces it and has its own tests.
func newContractValidator(t testing.TB) validator.Validator {
	t.Helper()
	contractOnce.Do(func() {
		document, err := libopenapi.NewDocument(api.OpenAPI)
		if err != nil {
			contractFailure = fmt.Errorf("reading the contract: %w", err)
			return
		}
		contract, problems := validator.NewValidator(document,
			config.WithFormatAssertions(), config.WithoutSecurityValidation())
		if len(problems) > 0 {
			contractFailure = fmt.Errorf("building the contract validator: %w", errors.Join(problems...))
			return
		}
		sharedContract = contract
	})
	if contractFailure != nil {
		t.Fatalf("apitest: %v", contractFailure)
	}
	return sharedContract
}

// clock is the API's clock: the fixed time the test has reached.
func (s *Server) clock() time.Time {
	return s.now
}

// Advance moves the API's clock forward by d, for example past the end of a session or of Undo.
func (s *Server) Advance(d time.Duration) {
	s.now = s.now.Add(d)
}

// Client sends requests as one signed-in person: see SignIn.
type Client struct {
	server *Server
	cookie *http.Cookie // the person's session cookie
}

// SignIn signs in as the demo person with e-mail email, through POST /api/auth/sign-in with the demo
// password, and returns a client that sends that person's session cookie. It stops the test when the
// sign-in fails.
func (s *Server) SignIn(t testing.TB, email string) *Client {
	t.Helper()
	response := s.Do(t, http.MethodPost, "/api/auth/sign-in",
		map[string]string{"email": email, "password": s.demoPassword})
	cookie := response.Cookie(auth.CookieName)
	if response.Status != http.StatusOK || cookie == nil {
		t.Fatalf("apitest: signing in as %s: %d %s", email, response.Status, response.Body)
	}
	return &Client{server: s, cookie: cookie}
}

// Do sends a request like Server.Do, with the person's session cookie.
func (c *Client) Do(t testing.TB, method, path string, body any, opts ...RequestOption) *Response {
	t.Helper()
	withSession := append([]RequestOption{WithCookie(c.cookie)}, opts...) // a new list: opts stays as it was
	return c.server.Do(t, method, path, body, withSession...)
}

// RequestOption changes one request sent by Do.
type RequestOption func(*request)

type request struct {
	skipRequestValidation bool
	skipValidation        bool
	rawBody               []byte
	host                  string
	cookies               []*http.Cookie
}

// SkipRequestValidation sends a request that breaks the contract on purpose, such as a body with a
// missing field or a wrong type. The answer is still checked.
func SkipRequestValidation() RequestOption {
	return func(r *request) { r.skipRequestValidation = true }
}

// SkipValidation checks neither the request nor the answer, for a path or method that is not in the
// contract.
func SkipValidation() RequestOption {
	return func(r *request) { r.skipValidation = true }
}

// RawBody sends body exactly as given, as application/json, for example malformed JSON. It replaces the
// body argument of Do.
func RawBody(body []byte) RequestOption {
	return func(r *request) { r.rawBody = body }
}

// WithCookie sends cookie c with the request.
func WithCookie(c *http.Cookie) RequestOption {
	return func(r *request) { r.cookies = append(r.cookies, c) }
}

// WithHost sends the request to host (the Host header) instead of 127.0.0.1.
func WithHost(host string) RequestOption {
	return func(r *request) { r.host = host }
}

// Response is the API's answer to one request.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Do sends a request to the API and returns the answer. body, when not nil, is sent as JSON. The request
// and the answer are checked against the contract; a mismatch fails the test, which goes on so that it
// can report more.
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

	// The request is built twice: the validator reads the body of the one it checks.
	if !options.skipValidation && !options.skipRequestValidation {
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

// newRequest builds the request Do sends: JSON when there is a payload, to the chosen host, with cookies.
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

// describe lists the validator's findings, one per line.
func describe(problems []*openapierrors.ValidationError) string {
	var text strings.Builder
	for _, problem := range problems {
		text.WriteString("\n  - " + problem.Error())
	}
	return text.String()
}

// Cookie returns the cookie called name that the answer set, or nil when it set none.
func (r *Response) Cookie(name string) *http.Cookie {
	for _, line := range r.Header.Values("Set-Cookie") {
		if cookie, err := http.ParseSetCookie(line); err == nil && cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// Decode reads the answer's JSON body into v. It stops the test when the body does not fit v.
func (r *Response) Decode(t testing.TB, v any) {
	t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		t.Fatalf("apitest: the answer %s does not fit %T: %v", r.Body, v, err)
	}
}
