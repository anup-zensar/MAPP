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
	unreadableMessage = "SP&A One couldn't read this request. Reload the page and try again."
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
