// Package clock provides "now" in a form tests can replace.
//
// Services never call time.Now themselves: they receive a Clock, which is System when the API runs and
// a fixed time in tests (constitution V: tests do not depend on the real clock).
package clock

import "time"

// Clock returns the current time.
type Clock func() time.Time

// System returns the real current time in UTC, cut to whole milliseconds. Timestamps are stored in UTC
// (constitution VI), and JSON keeps at most 3 fractional digits of a second (research R10), so a time
// read back from storage equals the time that was written.
func System() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}
