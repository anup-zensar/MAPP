package settings

import (
	"fmt"
	"testing"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/race"
)

// A 1,000-row GSAP rate file is reviewed, every row with its outcome, well within the 10 seconds of 016 SC-002
// (the page then shows the answer at once). Timed three times; the fastest counts.
func TestCheckRateImport_ThousandRows_016_SC002(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	f := importFixture(t)
	rows := []string{rateHeader}
	first := date.MustParse("2024-01-01")
	for day := range 500 {
		for _, code := range []string{"EUR", "GBP"} {
			rows = append(rows, fmt.Sprintf("M,USD,%s,%s,%d.%03d", code, first.AddDays(day), 1+day%9, day%1000))
		}
	}
	file := upload(rows...)

	elapsed := time.Hour
	var check RateImportCheck
	for range 3 {
		began := time.Now()
		var err error
		check, err = f.service.CheckRateImport(t.Context(), alex, file)
		elapsed = min(elapsed, time.Since(began))
		if err != nil {
			t.Fatalf("CheckRateImport: %v", err)
		}
	}

	if len(check.Rows) != 1000 {
		t.Errorf("reviewed %d rows, want 1000", len(check.Rows))
	}
	if elapsed > 10*time.Second {
		t.Errorf("the review took %s, want under 10 seconds", elapsed)
	}
	t.Logf("1,000 rows reviewed in %s", elapsed)
}
