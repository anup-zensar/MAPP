package history

import (
	"context"
	"strings"
)

type noteKey struct{}

// WithNote returns a copy of ctx whose activity-log entries end their details with note, such as "Bulk change
// of 20 activities (BLK-1001)"; an inner WithNote replaces an outer one.
func WithNote(ctx context.Context, note string) context.Context {
	return context.WithValue(ctx, noteKey{}, note)
}

func withNote(ctx context.Context, details string) string {
	note, _ := ctx.Value(noteKey{}).(string)
	switch {
	case note == "":
		return details
	case strings.TrimSpace(details) == "":
		return note
	}
	return strings.TrimRight(details, ". ") + ". " + note
}
