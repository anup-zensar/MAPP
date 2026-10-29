package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	"mapp/internal/platform/web"
)

const (
	controlBody = `{"error":{"code":"validation_failed","message":"Some fields need your attention.",` +
		`"fields":[{"field":"%s","message":"Remove the control characters from this text."}]}}`
	tooLargeBody = `{"error":{"code":"too_large","message":"This request is larger than 1 MB.","fields":[]}}`
)

func TestEngine_SecurityHeadersOnEveryAnswer_038_FR015(t *testing.T) {
	t.Parallel()
	otherHost := newRequest(http.MethodGet, testPath, "")
	otherHost.Host = "evil.example"
	answers := map[string]*httptest.ResponseRecorder{
		"normal answer": serve(t, ok, newRequest(http.MethodGet, testPath, "")),
		"refused host":  serve(t, ok, otherHost),
		"panic":         serve(t, func(*gin.Context) { panic("boom") }, newRequest(http.MethodGet, testPath, "")),
		"unknown path":  serve(t, ok, newRequest(http.MethodGet, "/api/v1/nowhere", "")),
	}
	want := map[string]string{
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"X-Content-Type-Options":  "nosniff",
	}
	for name, w := range answers {
		for header, value := range want {
			if got := w.Header().Get(header); got != value {
				t.Errorf("%s: %s = %q, want %q", name, header, got, value)
			}
		}
	}
}

func TestEngine_RefusesAForeignReferer_038_FR016_L14(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name, method, referer string
		refused               bool
	}{
		{"POST referred by another site", http.MethodPost, "http://evil.example/plans", true},
		{"PATCH referred by another port", http.MethodPatch, "http://127.0.0.1:3000/plans", true},
		{"POST with an unreadable referer", http.MethodPost, "://", true},
		{"POST referred by the same host", http.MethodPost, "http://127.0.0.1:8080/plans/P-1", false},
		{"GET referred by another site", http.MethodGet, "http://evil.example/plans", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := newRequest(tt.method, testPath, "")
			r.Header.Set("Referer", tt.referer)
			assertRefused(t, r, tt.refused)
		})
	}
}

func TestEngine_OriginWinsOverReferer_038_FR016(t *testing.T) {
	t.Parallel()
	r := newRequest(http.MethodPost, testPath, "")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.Header.Set("Referer", "http://evil.example/plans")

	assertRefused(t, r, false)
}

func TestBindJSON_RefusesControlCharacters_038_FR017_M3(t *testing.T) {
	t.Parallel()
	tests := []struct{ name, body, field string }{
		{"NUL in a nested field", `{"plannedCost":{"amount":"1\u0000"}}`, "plannedCost.amount"},
		{"bell in the name", `{"name":"Ide\u0007as"}`, "name"},
		{"delete character", "{\"name\":\"Ideas\x7f\"}", "name"},
		{"unpaired high surrogate", `{"name":"\ud800"}`, "name"},
		{"unpaired low surrogate", `{"description":"a\udc00b"}`, "description"},
		{"high surrogate before another escape", `{"name":"\ud800A"}`, "name"},
		{"NUL in a list", `{"recordIds":["A-1","A\u0000"]}`, "recordIds.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := serve(t, echoAny, newRequest(http.MethodPost, testPath, tt.body))

			assertAnswer(t, w, http.StatusBadRequest, strings.Replace(controlBody, "%s", tt.field, 1))
		})
	}
}

func TestBindJSON_AllowsOrdinaryText_038_FR017(t *testing.T) {
	t.Parallel()
	bodies := []string{
		`{"description":"Line one\n\tLine two\r\n"}`,
		`{"name":"Rocket 🚀 launch"}`,
		`{"name":"Rocket 🚀 — café \\ud800"}`,
		`{"recordIds":["A-1","A-2"],"plannedCost":{"amount":"12.50"},"yes":true,"n":null}`,
	}
	for _, body := range bodies {
		w := serve(t, echoAny, newRequest(http.MethodPost, testPath, body))
		if w.Code != http.StatusOK {
			t.Errorf("%s: status %d %s, want 200", body, w.Code, w.Body.String())
		}
	}
}

func TestEngine_ControlCharacterInAPathParameterIsNotFound_038_FR017_M3(t *testing.T) {
	t.Parallel()
	engine, err := web.NewEngine(web.Options{Tracer: tracenoop.NewTracerProvider(), Meter: metricnoop.NewMeterProvider()})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	ran := false
	engine.GET("/api/v1/things/:thingId", func(c *gin.Context) { ran = true; ok(c) })
	for _, path := range []string{"/api/v1/things/T-1%00", "/api/v1/things/T%0A1"} {
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, newRequest(http.MethodGet, path, ""))

		assertAnswer(t, w, http.StatusNotFound, notFoundBody)
	}
	if ran {
		t.Error("the handler ran for a path parameter with a control character")
	}
}

func TestBindJSON_BodyOverTheLimitIsTooLarge_038_FR054_L5(t *testing.T) {
	t.Parallel()
	w := serve(t, echoName, newRequest(http.MethodPost, testPath, bigName(1<<20)))

	assertAnswer(t, w, http.StatusRequestEntityTooLarge, tooLargeBody)
}

func TestEngine_ClientIsReportedOnlyByTrustedProxies_038_FR014(t *testing.T) {
	t.Parallel()
	engine, err := web.NewEngine(web.Options{Tracer: tracenoop.NewTracerProvider(), Meter: metricnoop.NewMeterProvider(),
		TrustedProxies: []string{"10.0.0.0/8"}})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	engine.GET(testPath, func(c *gin.Context) { web.Respond(c, http.StatusOK, c.ClientIP()) })
	tests := map[string]string{
		"127.0.0.1:5000": "203.0.113.5", "[::1]:5000": "203.0.113.5", "10.1.2.3:5000": "203.0.113.5",
		"192.0.2.1:5000": "192.0.2.1",
	}
	for remote, want := range tests {
		r := newRequest(http.MethodGet, testPath, "")
		r.RemoteAddr = remote
		r.Header.Set("X-Forwarded-For", "203.0.113.5")
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, r)

		if got := w.Body.String(); got != `"`+want+`"` {
			t.Errorf("from %s the client is %s, want %s", remote, got, want)
		}
	}
}

func TestNewEngine_RefusesAnUnreadableTrustedProxy_038_FR014(t *testing.T) {
	t.Parallel()
	_, err := web.NewEngine(web.Options{Tracer: tracenoop.NewTracerProvider(), Meter: metricnoop.NewMeterProvider(),
		TrustedProxies: []string{"not-an-address"}})

	if err == nil {
		t.Error("NewEngine accepted the trusted proxy not-an-address")
	}
}

func echoAny(c *gin.Context) {
	var input map[string]any
	if err := web.BindJSON(c, &input); err != nil {
		web.RespondError(c, err)
		return
	}
	web.Respond(c, http.StatusOK, input)
}
