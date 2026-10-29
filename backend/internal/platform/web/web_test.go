package web_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/logging"
	"mapp/internal/platform/web"
)

const (
	otherSiteBody  = `{"error":{"code":"bad_request","message":"This request came from another site and was refused.","fields":[]}}`
	unreadableBody = `{"error":{"code":"bad_request","message":"SP\u0026A One couldn't read this request. Reload the page and try again.","fields":[]}}`
	internalBody   = `{"error":{"code":"internal_error","message":"Something went wrong on our side. Please try again.","fields":[]}}`
	notFoundBody   = `{"error":{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}}`
	testPath       = "/api/v1/test"
	uploadPath     = "/api/v1/upload"
)

// Requests that a browser marks as coming from another site or origin are refused; reads stay allowed.
func TestEngine_RefusesCrossOriginRequests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, method, header, value string
		refused                     bool
	}{
		{"POST from another site", http.MethodPost, "Sec-Fetch-Site", "same-site", true},
		{"PATCH from another site", http.MethodPatch, "Sec-Fetch-Site", "cross-site", true},
		{"POST from another origin, older browser", http.MethodPost, "Origin", "http://127.0.0.1:3000", true},
		{"POST from the same origin", http.MethodPost, "Sec-Fetch-Site", "same-origin", false},
		{"POST without browser headers", http.MethodPost, "", "", false},
		{"GET from another site", http.MethodGet, "Sec-Fetch-Site", "cross-site", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newRequest(tt.method, testPath, "")
			if tt.header != "" {
				r.Header.Set(tt.header, tt.value)
			}
			assertRefused(t, r, tt.refused)
		})
	}
}

// Only requests addressed to this computer are answered (DNS rebinding); any port is fine.
func TestEngine_RefusesOtherHosts(t *testing.T) {
	t.Parallel()
	hosts := map[string]bool{
		"127.0.0.1:8080": false, "127.0.0.1": false, "localhost:5173": false, "LOCALHOST": false,
		"[::1]:8080": false, "[::1]": false, "evil.example": true, "evil.example:8080": true,
		"127.0.0.1.evil.example": true, "localhost.evil.example:8080": true, "": true,
	}
	for host, refused := range hosts {
		t.Run(host, func(t *testing.T) {
			t.Parallel()
			r := newRequest(http.MethodGet, testPath, "")
			r.Host = host
			assertRefused(t, r, refused)
		})
	}
}

func TestEngine_PanicAnswersInternalError_024_FR009(t *testing.T) {
	t.Parallel()
	w := serve(t, func(*gin.Context) { panic("boom") }, newRequest(http.MethodGet, testPath, ""))

	assertAnswer(t, w, http.StatusInternalServerError, internalBody)
}

func TestEngine_PanicAfterTheAnswerStartedAddsNothing(t *testing.T) {
	t.Parallel()
	startedThenPanics := func(c *gin.Context) {
		web.Respond(c, http.StatusOK, map[string]bool{"ok": true})
		panic("boom")
	}

	w := serve(t, startedThenPanics, newRequest(http.MethodGet, testPath, ""))

	assertAnswer(t, w, http.StatusOK, `{"ok":true}`)
}

// Every answer, refusals, panics and unknown paths included, forbids caching and content sniffing.
func TestEngine_APIHeadersOnEveryAnswer(t *testing.T) {
	t.Parallel()
	otherHost := newRequest(http.MethodGet, testPath, "")
	otherHost.Host = "evil.example"
	crossSite := newRequest(http.MethodPost, testPath, "")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")
	answers := map[string]*httptest.ResponseRecorder{
		"normal answer":      serve(t, ok, newRequest(http.MethodGet, testPath, "")),
		"refused host":       serve(t, ok, otherHost),
		"refused cross-site": serve(t, ok, crossSite),
		"panic":              serve(t, func(*gin.Context) { panic("boom") }, newRequest(http.MethodGet, testPath, "")),
		"unknown path":       serve(t, ok, newRequest(http.MethodGet, "/api/v1/nowhere", "")),
	}
	for name, w := range answers {
		if got := w.Header().Get("Cache-Control"); got != "no-store" {
			t.Errorf("%s: Cache-Control = %q, want no-store", name, got)
		}
		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: X-Content-Type-Options = %q, want nosniff", name, got)
		}
	}
}

func TestEngine_UnknownPathsAndMethodsAnswerJSONNotFound(t *testing.T) {
	t.Parallel()
	for _, r := range []*http.Request{
		newRequest(http.MethodGet, "/api/v1/nowhere", ""),
		newRequest(http.MethodGet, testPath+"/", ""),
		newRequest(http.MethodDelete, testPath, ""),
	} {
		assertAnswer(t, serve(t, ok, r), http.StatusNotFound, notFoundBody)
	}
}

func TestEngine_RequestIDMadeByTheServer_024_FR005(t *testing.T) {
	t.Parallel()
	sent := newRequest(http.MethodGet, testPath, "")
	sent.Header.Set("X-Request-ID", "chosen-by-the-client")

	first := serve(t, ok, sent).Header().Get("X-Request-ID")
	second := serve(t, ok, newRequest(http.MethodGet, testPath, "")).Header().Get("X-Request-ID")

	if first == "" || first == "chosen-by-the-client" || first == second {
		t.Errorf("request IDs %q and %q, want two different IDs made by the server", first, second)
	}
}

func TestEngine_OneRequestRecordPerRequest_024_FR007(t *testing.T) {
	var out bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(logging.New(&out, slog.LevelInfo))
	t.Cleanup(func() { slog.SetDefault(previous) })

	w := serve(t, ok, newRequest(http.MethodGet, testPath, ""))

	var record map[string]any
	if err := json.Unmarshal(out.Bytes(), &record); err != nil {
		t.Fatalf("log %q is not one JSON record: %v", out.String(), err)
	}
	want := map[string]any{"msg": "request", "method": "GET", "path": testPath, "route": testPath,
		"status": float64(http.StatusOK), "request_id": w.Header().Get("X-Request-ID")}
	for key, value := range want {
		if record[key] != value {
			t.Errorf("%s = %v, want %v", key, record[key], value)
		}
	}
	if _, found := record["duration_ms"]; !found {
		t.Error("the request record has no duration_ms")
	}
}

func TestBindJSON_AcceptsAJSONBody(t *testing.T) {
	t.Parallel()
	r := newRequest(http.MethodPost, testPath, `{"name":"Ideas"}`+"\n")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	assertAnswer(t, serve(t, echoName, r), http.StatusOK, `{"name":"Ideas"}`)
}

// A body MAPP cannot read safely is refused with a message for the user; the detail goes only to the log.
func TestBindJSON_RefusesUnreadableBodies(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, contentType, body string }{
		{"plain text content type", "text/plain", `{"name":"Ideas"}`},
		{"no content type", "", `{"name":"Ideas"}`},
		{"data after the JSON value", "application/json", `{"name":"Ideas"} {"name":"Other"}`},
		{"garbage after the JSON value", "application/json", `{"name":"Ideas"}x`},
		{"unknown field", "application/json", `{"name":"Ideas","colour":"red"}`},
		{"not JSON", "application/json", `name=Ideas`},
		{"wrong type", "application/json", `{"name":42}`},
		{"empty body", "application/json", ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newRequest(http.MethodPost, testPath, tt.body)
			r.Header.Del("Content-Type")
			if tt.contentType != "" {
				r.Header.Set("Content-Type", tt.contentType)
			}

			assertAnswer(t, serve(t, echoName, r), http.StatusBadRequest, unreadableBody)
		})
	}
}

func TestLimitBody_AnOperationMayRaiseTheLimit_024_FR010(t *testing.T) {
	t.Parallel()
	w := serve(t, echoName, newRequest(http.MethodPost, uploadPath, bigName(2<<20)))

	if w.Code != http.StatusOK {
		t.Errorf("a 2 MB body to %s = %d %s, want 200", uploadPath, w.Code, w.Body.String()[:min(w.Body.Len(), 200)])
	}
}

func TestRespondError_WrappedErrorKeepsItsStatus(t *testing.T) {
	t.Parallel()
	err := fmt.Errorf("deleting folder: %w", errs.NotFound("This sample folder no longer exists — someone may have deleted it."))

	w := serve(t, failWith(err), newRequest(http.MethodGet, testPath, ""))

	assertAnswer(t, w, http.StatusNotFound,
		`{"error":{"code":"not_found","message":"This sample folder no longer exists — someone may have deleted it.","fields":[]}}`)
}

func TestRespondError_UnexpectedErrorHidesDetails_FR020(t *testing.T) {
	t.Parallel()
	w := serve(t, failWith(errors.New("open data/people.json: access is denied")), newRequest(http.MethodGet, testPath, ""))

	assertAnswer(t, w, http.StatusInternalServerError, internalBody)
}

func TestRespondError_RetryAfterInWholeSeconds_038_FR014(t *testing.T) {
	t.Parallel()
	err := errs.TooManyAttempts("Too many sign-in attempts. Try again in 15 minutes.", 14*time.Minute+500*time.Millisecond)

	w := serve(t, failWith(err), newRequest(http.MethodPost, testPath, ""))

	assertAnswer(t, w, http.StatusTooManyRequests,
		`{"error":{"code":"too_many_attempts","message":"Too many sign-in attempts. Try again in 15 minutes.","fields":[]}}`)
	if got := w.Header().Get("Retry-After"); got != "841" {
		t.Errorf("Retry-After = %q, want 841", got)
	}
}

func newRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

// serve answers r with the API's engine, whose GET, POST and PATCH /api/v1/test and POST /api/v1/upload
// (body limit 3 MB) run h.
func serve(t *testing.T, h gin.HandlerFunc, r *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	engine, err := web.NewEngine(web.Options{Tracer: tracenoop.NewTracerProvider(), Meter: metricnoop.NewMeterProvider(),
		BodyLimits: map[string]int64{"POST " + uploadPath: 3 << 20}})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch} {
		engine.Handle(method, testPath, h)
	}
	engine.POST(uploadPath, h)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, r)
	return w
}

func assertRefused(t *testing.T, r *http.Request, refused bool) {
	t.Helper()
	ran := false
	w := serve(t, func(c *gin.Context) { ran = true; ok(c) }, r)
	if refused {
		assertAnswer(t, w, http.StatusBadRequest, otherSiteBody)
		if ran {
			t.Error("the handler ran for a refused request")
		}
	} else if w.Code != http.StatusOK || !ran {
		t.Errorf("status %d, handler ran %v; want 200 and the handler to run", w.Code, ran)
	}
}

func ok(c *gin.Context) {
	web.Respond(c, http.StatusOK, map[string]bool{"ok": true})
}

func failWith(err error) gin.HandlerFunc {
	return func(c *gin.Context) { web.RespondError(c, err) }
}

func echoName(c *gin.Context) {
	var input struct {
		Name string `json:"name"`
	}
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, input)
}

func bigName(size int) string {
	return `{"name":"` + strings.Repeat("a", size) + `"}`
}

func assertAnswer(t *testing.T, w *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if w.Code != status {
		t.Errorf("status = %d, want %d", w.Code, status)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := w.Body.String(); got != body {
		t.Errorf("body =\n%s\nwant\n%s", got, body)
	}
}
