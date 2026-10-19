package web_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/platform/errs"
	"northstar/internal/platform/web"
)

// The bodies of the API's own refusals, as the web app receives them.
const (
	otherSiteBody  = `{"code":"bad_request","message":"This request came from another site and was refused.","fields":[]}`
	unreadableBody = `{"code":"bad_request","message":"NorthStar couldn't read this request. Reload the page and try again.","fields":[]}`
	internalBody   = `{"code":"internal_error","message":"Something went wrong on our side. Please try again.","fields":[]}`
)

// Requests that a browser marks as coming from another site or origin are refused, so another web page
// on this computer cannot act for the signed-in person. Reads (GET) change nothing and stay allowed
// (research R6).
func TestMiddleware_RefusesCrossOriginRequests(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		header  string
		value   string
		refused bool
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
			handler, called := countingHandler()
			r := newRequest(tt.method, "/api/test", "")
			if tt.header != "" {
				r.Header.Set(tt.header, tt.value)
			}

			w := serve(web.Middleware(handler), r)

			if tt.refused {
				assertAnswer(t, w, http.StatusBadRequest, otherSiteBody)
				if *called {
					t.Error("the handler ran for a refused request")
				}
			} else if w.Code != http.StatusOK || !*called {
				t.Errorf("status %d, handler ran %v; want 200 and the handler to run", w.Code, *called)
			}
		})
	}
}

// Only requests addressed to this computer are answered, which protects against DNS rebinding
// (research R6). Any port is fine.
func TestMiddleware_RefusesOtherHosts(t *testing.T) {
	tests := []struct {
		host    string
		refused bool
	}{
		{"127.0.0.1:8080", false},
		{"127.0.0.1", false},
		{"localhost:5173", false},
		{"LOCALHOST", false},
		{"[::1]:8080", false},
		{"[::1]", false},
		{"evil.example", true},
		{"evil.example:8080", true},
		{"127.0.0.1.evil.example", true},
		{"localhost.evil.example:8080", true},
		{"", true},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			handler, called := countingHandler()
			r := newRequest(http.MethodGet, "/api/test", "")
			r.Host = tt.host

			w := serve(web.Middleware(handler), r)

			if tt.refused {
				assertAnswer(t, w, http.StatusBadRequest, otherSiteBody)
				if *called {
					t.Error("the handler ran for a refused request")
				}
			} else if w.Code != http.StatusOK || !*called {
				t.Errorf("status %d, handler ran %v; want 200 and the handler to run", w.Code, *called)
			}
		})
	}
}

// A panic in a handler answers a plain-language 500, never a stack trace (FR-020).
func TestMiddleware_PanicAnswersInternalError_FR020(t *testing.T) {
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })

	w := serve(web.Middleware(panicking), newRequest(http.MethodGet, "/api/test", ""))

	assertAnswer(t, w, http.StatusInternalServerError, internalBody)
}

// A panic after the answer has started cannot change it any more: nothing is added to what was sent.
func TestMiddleware_PanicAfterTheAnswerStartedAddsNothing(t *testing.T) {
	startedThenPanics := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		web.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
		panic("boom")
	})

	w := serve(web.Middleware(startedThenPanics), newRequest(http.MethodGet, "/api/test", ""))

	assertAnswer(t, w, http.StatusOK, `{"ok":true}`)
}

// Every answer tells the browser not to keep a copy (so signed-out users never see cached data) and not
// to guess content types (research R10).
func TestMiddleware_APIHeadersOnEveryAnswer(t *testing.T) {
	ok, _ := countingHandler()
	panicking := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") })
	otherHost := newRequest(http.MethodGet, "/api/test", "")
	otherHost.Host = "evil.example"
	crossSite := newRequest(http.MethodPost, "/api/test", "")
	crossSite.Header.Set("Sec-Fetch-Site", "cross-site")

	answers := map[string]*httptest.ResponseRecorder{
		"normal answer":      serve(web.Middleware(ok), newRequest(http.MethodGet, "/api/test", "")),
		"refused host":       serve(web.Middleware(ok), otherHost),
		"refused cross-site": serve(web.Middleware(ok), crossSite),
		"panic":              serve(web.Middleware(panicking), newRequest(http.MethodGet, "/api/test", "")),
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

func TestReadJSON_AcceptsAJSONBody(t *testing.T) {
	r := newRequest(http.MethodPost, "/api/test", `{"name":"Ideas"}`+"\n")
	r.Header.Set("Content-Type", "application/json; charset=utf-8")

	w := serve(http.HandlerFunc(echoName), r)

	assertAnswer(t, w, http.StatusOK, `{"name":"Ideas"}`)
}

// A body NorthStar cannot read safely is refused with a message for the user; the technical detail goes
// only to the server log (research R9, R10).
func TestReadJSON_RefusesUnreadableBodies(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{"plain text content type", "text/plain", `{"name":"Ideas"}`},
		{"no content type", "", `{"name":"Ideas"}`},
		{"data after the JSON value", "application/json", `{"name":"Ideas"} {"name":"Other"}`},
		{"garbage after the JSON value", "application/json", `{"name":"Ideas"}x`},
		{"unknown field", "application/json", `{"name":"Ideas","colour":"red"}`},
		{"not JSON", "application/json", `name=Ideas`},
		{"wrong type", "application/json", `{"name":42}`},
		{"empty body", "application/json", ``},
		{"larger than 1 MB", "application/json", `{"name":"` + strings.Repeat("a", 1<<20) + `"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRequest(http.MethodPost, "/api/test", tt.body)
			r.Header.Del("Content-Type")
			if tt.contentType != "" {
				r.Header.Set("Content-Type", tt.contentType)
			}

			w := serve(http.HandlerFunc(echoName), r)

			assertAnswer(t, w, http.StatusBadRequest, unreadableBody)
		})
	}
}

// An *errs.Error keeps its status and message even when it was wrapped on its way up, for example
// through a store transaction (research R9).
func TestWriteError_WrappedErrorKeepsItsStatus(t *testing.T) {
	err := fmt.Errorf("deleting folder: %w", errs.NotFound("This sample folder no longer exists — someone may have deleted it."))
	w := httptest.NewRecorder()

	web.WriteError(w, newRequest(http.MethodDelete, "/api/sample-folders/SF-1001", ""), err)

	assertAnswer(t, w, http.StatusNotFound,
		`{"code":"not_found","message":"This sample folder no longer exists — someone may have deleted it.","fields":[]}`)
}

// Any other error becomes a plain-language 500; its technical text never reaches the user (FR-020).
func TestWriteError_UnexpectedErrorHidesDetails_FR020(t *testing.T) {
	w := httptest.NewRecorder()

	web.WriteError(w, newRequest(http.MethodGet, "/api/test", ""), errors.New("open data/people.json: Access is denied."))

	assertAnswer(t, w, http.StatusInternalServerError, internalBody)
}

// Every error answer has "fields", an empty list when no field is wrong (research R9, FR-020).
func TestWriteError_FieldsAlwaysPresent_FR020(t *testing.T) {
	failures := map[string]error{
		"sign in required": errs.SignInRequired(),
		"version conflict": errs.VersionConflict("sample folder", "Daniel Okoro", time.Date(2026, 3, 12, 14, 32, 0, 0, time.UTC)),
		"unexpected":       errors.New("disk full"),
	}
	for name, err := range failures {
		w := httptest.NewRecorder()
		web.WriteError(w, newRequest(http.MethodGet, "/api/test", ""), err)

		var body map[string]json.RawMessage
		if decodeErr := json.Unmarshal(w.Body.Bytes(), &body); decodeErr != nil {
			t.Fatalf("%s: body %s is not JSON: %v", name, w.Body, decodeErr)
		}
		if got := string(body["fields"]); got != "[]" {
			t.Errorf("%s: fields = %s, want []", name, got)
		}
	}
}

func TestNotFound_AnswersJSON(t *testing.T) {
	w := httptest.NewRecorder()

	web.NotFound(w, newRequest(http.MethodGet, "/api/nowhere", ""))

	assertAnswer(t, w, http.StatusNotFound, `{"code":"not_found","message":"This page or record doesn't exist.","fields":[]}`)
}

func TestWriteJSON_StatusContentTypeAndBody(t *testing.T) {
	w := httptest.NewRecorder()

	web.WriteJSON(w, http.StatusCreated, map[string]string{"id": "SF-1001"})

	assertAnswer(t, w, http.StatusCreated, `{"id":"SF-1001"}`)
}

// The router records every pattern it registers, so a test can compare the API's routes with the
// contract (research R11).
func TestRouter_RecordsPatterns(t *testing.T) {
	ok, _ := countingHandler()
	router := web.NewRouter()
	router.Handle("GET /api/health", ok)
	router.Handle("POST /api/auth/sign-in", ok)

	got := router.Patterns()
	if want := []string{"GET /api/health", "POST /api/auth/sign-in"}; !slices.Equal(got, want) {
		t.Fatalf("Patterns() = %q, want %q", got, want)
	}

	got[0] = "changed by the caller"
	if again := router.Patterns(); again[0] != "GET /api/health" {
		t.Errorf("changing the returned list changed the router: %q", again)
	}
}

func TestRouter_ServesRegisteredPatterns(t *testing.T) {
	router := web.NewRouter()
	router.Handle("GET /api/sample-folders/{folderId}", func(w http.ResponseWriter, r *http.Request) {
		web.WriteJSON(w, http.StatusOK, map[string]string{"folderId": r.PathValue("folderId")})
	})

	w := serve(router, newRequest(http.MethodGet, "/api/sample-folders/SF-1001", ""))

	assertAnswer(t, w, http.StatusOK, `{"folderId":"SF-1001"}`)
}

// newRequest builds a request the way the web app sends it to the local API.
func newRequest(method, path, body string) *http.Request {
	r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

// serve sends r to h and returns the answer.
func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// countingHandler answers 200 {"ok":true} and records whether it ran.
func countingHandler() (http.HandlerFunc, *bool) {
	called := false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		web.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}), &called
}

// echoName reads {"name"} with ReadJSON and answers it back.
func echoName(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := web.ReadJSON(w, r, &input); err != nil {
		web.WriteError(w, r, err)
		return
	}
	web.WriteJSON(w, http.StatusOK, input)
}

// assertAnswer checks the status, the JSON content type and the exact body of an answer.
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
