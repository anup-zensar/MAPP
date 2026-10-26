// Package web is the API's HTTP transport on Gin: the engine and its middleware, strict JSON decoding, and
// the one place where errors become answers in the error envelope.
package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"mapp/internal/platform/errs"
)

const (
	unreadableMessage = "MAPP couldn't read this request. Reload the page and try again."
	otherSiteMessage  = "This request came from another site and was refused."
	notFoundMessage   = "This page or record doesn't exist."
)

const (
	maxBodyBytes = 1 << 20
	jsonType     = "application/json"
)

type errorEnvelope struct {
	Error *errs.Error `json:"error"`
}

type Wrap func(http.HandlerFunc) http.HandlerFunc

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return ReadJSONUpTo(w, r, dst, maxBodyBytes)
}

func ReadJSONUpTo(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	return readBody(r, http.MaxBytesReader(w, r.Body, limit), dst)
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	code, body, err := encode(status, v)
	if err != nil {
		slog.Error("could not encode the answer", "error", err)
	}
	w.Header().Set("Content-Type", jsonType)
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr, ok := errors.AsType[*errs.Error](err)
	if !ok {
		slog.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		apiErr = errs.Internal()
	}
	WriteJSON(w, apiErr.Status, errorEnvelope{Error: apiErr})
}

// readBody refuses, with bad_request, anything but one strict JSON value; the reason goes only to the log.
func readBody(r *http.Request, body io.Reader, dst any) error {
	if err := decode(r.Header.Get("Content-Type"), body, dst); err != nil {
		slog.WarnContext(r.Context(), "unreadable request body", "method", r.Method, "path", r.URL.Path, "error", err)
		return errs.BadRequest(unreadableMessage)
	}
	return nil
}

func decode(contentType string, body io.Reader, dst any) error {
	if mediaType, _, err := mime.ParseMediaType(contentType); err != nil || mediaType != jsonType {
		return fmt.Errorf("content type %q is not application/json", contentType)
	}
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decode the body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("data after the JSON value")
	}
	return nil
}

// encode falls back to the internal_error envelope when v cannot be encoded, a programming mistake.
func encode(status int, v any) (int, []byte, error) {
	body, err := json.Marshal(v)
	if err != nil {
		internal, _ := json.Marshal(errorEnvelope{Error: errs.Internal()}) // strings and lists only: cannot fail
		return http.StatusInternalServerError, internal, fmt.Errorf("encode the answer: %w", err)
	}
	return status, body, nil
}
