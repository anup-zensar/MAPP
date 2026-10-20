// Package web holds the HTTP plumbing every handler shares: reading and writing JSON, turning errors into
// the one error format (research R9), the middleware around every request, and the router that records
// the API's routes.
//
// Every handler follows the same steps:
//
//	var input FolderInput
//	if err := web.ReadJSON(w, r, &input); err != nil {
//		web.WriteError(w, r, err)
//		return
//	}
//	a, _ := actor.From(r.Context()) // put there by the sign-in middleware
//	folder, err := h.service.CreateFolder(r.Context(), a, input)
//	if err != nil {
//		web.WriteError(w, r, err)
//		return
//	}
//	web.WriteJSON(w, http.StatusCreated, folder)
package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"northstar/internal/platform/errs"
)

// Messages of the plan's "User-visible messages" table.
const (
	unreadableMessage = "NorthStar couldn't read this request. Reload the page and try again."
	otherSiteMessage  = "This request came from another site and was refused."
	notFoundMessage   = "This page or record doesn't exist."
)

// maxBodyBytes is the largest request body the API reads (research R10).
const maxBodyBytes = 1 << 20 // 1 MB

// ReadJSON decodes the request's JSON body into dst. It accepts only a body sent as application/json, of
// at most 1 MB, without unknown fields and with nothing after the JSON value. Anything else returns a
// bad_request error with a message for the user; the technical detail goes only to the server log.
func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return ReadJSONUpTo(w, r, dst, maxBodyBytes)
}

// ReadJSONUpTo is ReadJSON for a body of at most limit bytes: the few operations that receive a file in their
// body (016's GSAP rate import: 3 MB for a 2 MB file in base64) raise the 1 MB of ReadJSON.
func ReadJSONUpTo(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	if err := decodeJSON(w, r, dst, limit); err != nil {
		slog.Warn("unreadable request body", "method", r.Method, "path", r.URL.Path, "error", err)
		return errs.BadRequest(unreadableMessage)
	}
	return nil
}

// decodeJSON does the work of ReadJSON and returns the technical reason when it refuses the body.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	contentType := r.Header.Get("Content-Type")
	if mediaType, _, err := mime.ParseMediaType(contentType); err != nil || mediaType != "application/json" {
		return fmt.Errorf("content type %q is not application/json", contentType)
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields() // a misspelt field must not be ignored silently
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	// Only spaces may follow the JSON value: a second value or any other data is refused.
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("data after the JSON value")
	}
	return nil
}

// WriteJSON answers with status and v encoded as JSON.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		// Only a programming mistake gets here: a value that JSON cannot encode.
		slog.Error("could not encode the answer", "error", err)
		status = http.StatusInternalServerError
		body, _ = json.Marshal(errs.Internal()) // cannot fail: errs.Error holds only strings and lists
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body) // a failed write means the client has gone; there is nobody left to tell
}

// WriteError answers with err in the API's error format. An *errs.Error, also when wrapped with %w on its
// way up, keeps its status and message. Any other error is logged and answered as internal_error, so
// technical details never reach the user (FR-020).
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var apiErr *errs.Error
	if !errors.As(err, &apiErr) {
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		apiErr = errs.Internal()
	}
	WriteJSON(w, apiErr.Status, apiErr)
}

// NotFound answers 404 not_found. The API answers it for every /api path or method it does not serve.
func NotFound(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, errs.NotFound(notFoundMessage))
}

// Middleware wraps every request to the API. In this order, it:
//
//   - recovers from a panic in a handler and answers internal_error (the details go to the log);
//   - logs one line per request: method, path, status and duration, never personal data (research R24);
//   - sets the API headers Cache-Control: no-store, so the browser never shows a signed-out person
//     cached data, and X-Content-Type-Options: nosniff (research R10);
//   - refuses, with bad_request, a request addressed to a host other than 127.0.0.1, localhost or [::1]
//     (protection against DNS rebinding) and a request a browser sent from another site or origin
//     (http.CrossOriginProtection), so other web pages cannot act for the signed-in person (research R6).
func Middleware(next http.Handler) http.Handler {
	crossOrigin := http.NewCrossOriginProtection()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		answer := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if problem := recover(); problem != nil {
				slog.Error("panic in handler", "method", r.Method, "path", r.URL.Path,
					"panic", problem, "stack", string(debug.Stack()))
				if !answer.started { // after the answer has started, nothing can be changed any more
					WriteError(answer, r, errs.Internal())
				}
			}
			slog.Info("request", "method", r.Method, "path", r.URL.Path,
				"status", answer.status, "duration", time.Since(start))
		}()

		// The headers are set before the checks, so that refusals and panics carry them too.
		answer.Header().Set("Cache-Control", "no-store")
		answer.Header().Set("X-Content-Type-Options", "nosniff")

		if !isLocalHost(r.Host) || crossOrigin.Check(r) != nil {
			WriteError(answer, r, errs.BadRequest(otherSiteMessage))
			return
		}
		next.ServeHTTP(answer, r)
	})
}

// isLocalHost reports whether a request's Host header, with or without a port, names this computer.
func isLocalHost(hostAndPort string) bool {
	host := hostAndPort
	if h, _, err := net.SplitHostPort(hostAndPort); err == nil {
		host = h
	}
	host = strings.TrimSuffix(strings.TrimPrefix(host, "["), "]") // "[::1]" sent without a port
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}

// statusRecorder passes everything on to the real ResponseWriter. It remembers the status code for the
// request log, and whether the answer has started, so that a panic is answered only when nothing was
// sent yet.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	started bool
}

// WriteHeader records the status code and sends it.
func (s *statusRecorder) WriteHeader(status int) {
	if !s.started {
		s.status, s.started = status, true
	}
	s.ResponseWriter.WriteHeader(status)
}

// Write records that the answer has started and sends b. Without an earlier WriteHeader, net/http sends
// 200, which is the status already recorded.
func (s *statusRecorder) Write(b []byte) (int, error) {
	s.started = true
	return s.ResponseWriter.Write(b)
}

// Wrap is middleware for a single route, such as the sign-in check: it takes a handler and returns one
// that does something before it. Packages receive it in their Routes(r, signedIn) method.
type Wrap func(http.HandlerFunc) http.HandlerFunc
