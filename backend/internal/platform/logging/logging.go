// Package logging writes the API's log as JSON lines and adds the request's attributes and trace ID to every
// record logged with the request's context (constitution B7).
package logging

import (
	"context"
	"io"
	"log/slog"
	"slices"

	"go.opentelemetry.io/otel/trace"
)

type attrsKey struct{}

type contextHandler struct {
	next slog.Handler
}

func New(w io.Writer, level slog.Leveler) *slog.Logger {
	options := &slog.HandlerOptions{Level: level, ReplaceAttr: utcTime}
	return slog.New(contextHandler{next: slog.NewJSONHandler(w, options)})
}

func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	existing, _ := ctx.Value(attrsKey{}).([]slog.Attr)
	// Clip makes append copy, so a child context never writes into its parent's attributes.
	return context.WithValue(ctx, attrsKey{}, append(slices.Clip(existing), attrs...))
}

func (h contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h contextHandler) Handle(ctx context.Context, record slog.Record) error {
	out := record.Clone()
	if attrs, ok := ctx.Value(attrsKey{}).([]slog.Attr); ok {
		out.AddAttrs(attrs...)
	}
	if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
		out.AddAttrs(slog.String("trace_id", span.TraceID().String()))
	}
	return h.next.Handle(ctx, out)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{next: h.next.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{next: h.next.WithGroup(name)}
}

func utcTime(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey {
		return slog.Time(slog.TimeKey, attr.Value.Time().UTC())
	}
	return attr
}
