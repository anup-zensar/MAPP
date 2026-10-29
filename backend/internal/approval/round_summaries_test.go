package approval

import (
	"testing"
	"time"
)

// An open round waits since its current level began; a decided round keeps its submission and end (036 FR-016).
func TestRoundSummaries_036_FR016(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1", "Fleet card launch", mei.ID, chinaFlow)
	f.addActivity(t, "A-2", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, mei, "A-1", "")
	f.clock.Advance(24 * time.Hour)
	f.approve(t, carlos, "A-1", "")
	f.submit(t, priya, "A-2", "")
	f.clock.Advance(36 * time.Hour)
	f.approve(t, daniel, "A-2", "")

	list, err := f.service.RoundSummaries(t.Context())
	if err != nil {
		t.Fatalf("RoundSummaries: %v", err)
	}
	got := map[string]RoundSummary{}
	for _, s := range list {
		got[s.SubjectID] = s
	}
	checkRounds(t, got["A-1"], got["A-2"])
}

func checkRounds(t *testing.T, open, decided RoundSummary) {
	t.Helper()
	if open.Outcome != OutcomeOpen || open.WaitingSince == nil || !open.WaitingSince.Equal(start.Add(24*time.Hour)) {
		t.Errorf("A-1 = %+v, want open and waiting at level 2 since a day after the start", open)
	}
	if decided.Outcome != OutcomeApproved || decided.EndedAt == nil || decided.EndedAt.Sub(decided.SubmittedAt) != 36*time.Hour ||
		decided.WaitingSince != nil {
		t.Errorf("A-2 = %+v, want approved 36 hours after submission", decided)
	}
}

func TestAverageDecisionDays_HalfUpToOneDecimal(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		decided, seconds int64
		want             string
	}{
		{name: "nothing decided", decided: 0, seconds: 0, want: ""},
		{name: "a day and a half", decided: 2, seconds: 3 * 86400, want: "1.5"},
		{name: "rounded down", decided: 3, seconds: 100000, want: "0.4"},
		{name: "a half rounds up", decided: 1, seconds: 4320, want: "0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ""
			if average := AverageDecisionDays(tt.decided, tt.seconds); average != nil {
				got = *average
			}
			if got != tt.want {
				t.Errorf("AverageDecisionDays(%d, %d) = %q, want %q", tt.decided, tt.seconds, got, tt.want)
			}
		})
	}
}
