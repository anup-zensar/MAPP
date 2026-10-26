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
	t.Parallel()
	if loc := clock.System().Location(); loc != time.UTC {
		t.Errorf("Location = %v, want UTC", loc)
	}
}

// Whole milliseconds: JSON keeps 3 fractional digits, so a stored timestamp reads back exactly (research R10).
func TestSystem_SurvivesJSONRoundTrip(t *testing.T) {
	t.Parallel()
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
