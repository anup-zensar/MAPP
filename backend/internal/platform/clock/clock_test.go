package clock_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/clock"
)

// Timestamps are stored in UTC (constitution VI).
func TestSystem_IsUTC(t *testing.T) {
	if loc := clock.System().Location(); loc != time.UTC {
		t.Errorf("Location = %v, want UTC", loc)
	}
}

// The time is cut to whole milliseconds, so a stored timestamp has at most 3 fractional digits and reads
// back exactly as it was written (research R10).
func TestSystem_SurvivesJSONRoundTrip(t *testing.T) {
	now := clock.System()

	body, err := json.Marshal(now)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if _, fraction, found := strings.Cut(strings.Trim(string(body), `"Z`), "."); found && len(fraction) > 3 {
		t.Errorf("JSON %s has %d fractional digits, want at most 3", body, len(fraction))
	}

	var back time.Time
	if err := json.Unmarshal(body, &back); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if !back.Equal(now) {
		t.Errorf("read back %v, want %v", back, now)
	}
}
