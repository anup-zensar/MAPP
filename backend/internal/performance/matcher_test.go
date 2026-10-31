package performance_test

import (
	"testing"

	"mapp/internal/performance"
)

func reaching(pairs map[performance.MatchedOn]map[string][]string) performance.Resolved {
	return performance.Resolved(pairs)
}

func allApproved(string) bool  { return true }
func noneApproved(string) bool { return false }

func TestDecide_PrecedenceAndOutcomes_FR018_FR019_FR022(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		keys     performance.RowKeys
		resolved performance.Resolved
		want     performance.Outcome
		wantOn   performance.MatchedOn
		wantID   string
	}{
		{
			name: "FR-018 an activity id alone matches and is recorded as the key used",
			keys: performance.RowKeys{ActivityID: "A-1"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{
				performance.MatchedActivityID: {"A-1": {"A-1"}}}),
			want: performance.OutcomeMatched, wantOn: performance.MatchedActivityID, wantID: "A-1",
		},
		{
			name: "FR-018 with no activity id the media code is the key used",
			keys: performance.RowKeys{MediaCode: "9000001"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{
				performance.MatchedMediaCode: {"9000001": {"A-2"}}}),
			want: performance.OutcomeMatched, wantOn: performance.MatchedMediaCode, wantID: "A-2",
		},
		{
			name: "FR-018 two keys that agree match, recorded against the higher one",
			keys: performance.RowKeys{ActivityID: "A-3", ExternalName: "spring"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{
				performance.MatchedActivityID:   {"A-3": {"A-3"}},
				performance.MatchedExternalName: {"spring": {"A-3"}}}),
			want: performance.OutcomeMatched, wantOn: performance.MatchedActivityID, wantID: "A-3",
		},
		{
			name: "FR-019 two keys that disagree never guess",
			keys: performance.RowKeys{ActivityID: "A-4", MediaCode: "9000009"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{
				performance.MatchedActivityID: {"A-4": {"A-4"}},
				performance.MatchedMediaCode:  {"9000009": {"A-5"}}}),
			want: performance.OutcomeUnmatched,
		},
		{
			name: "FR-019 one key reaching two activities is ambiguous",
			keys: performance.RowKeys{ExternalName: "spring"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{
				performance.MatchedExternalName: {"spring": {"A-6", "A-7"}}}),
			want: performance.OutcomeUnmatched,
		},
		{
			name:     "046-C20 a present key that resolves to nothing does NOT fall through to a lower key",
			keys:     performance.RowKeys{ActivityID: "A-typo", MediaCode: "9000001"},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{performance.MatchedMediaCode: {"9000001": {"A-8"}}}),
			want:     performance.OutcomeUnmatched,
		},
		{
			name:     "046-C3 a row carrying no key at all is unattributed, not unmatched",
			keys:     performance.RowKeys{},
			resolved: reaching(map[performance.MatchedOn]map[string][]string{}),
			want:     performance.OutcomeUnattributed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := performance.Decide(tt.keys, tt.resolved, allApproved)

			if got.Outcome != tt.want {
				t.Fatalf("Decide() outcome = %s (%s), want %s", got.Outcome, got.Reason, tt.want)
			}
			if tt.wantID != "" && got.ActivityID != tt.wantID {
				t.Errorf("Decide() matched %s, want %s", got.ActivityID, tt.wantID)
			}
			if tt.wantOn != "" && got.MatchedOn != tt.wantOn {
				t.Errorf("Decide() recorded %s as the key used, want %s", got.MatchedOn, tt.wantOn)
			}
		})
	}
}

func TestDecide_UnapprovedActivityHoldsTheFact_FR020(t *testing.T) {
	t.Parallel()
	keys := performance.RowKeys{ActivityID: "A-1"}
	resolved := reaching(map[performance.MatchedOn]map[string][]string{
		performance.MatchedActivityID: {"A-1": {"A-1"}}})

	held := performance.Decide(keys, resolved, noneApproved)
	counted := performance.Decide(keys, resolved, allApproved)

	if held.Outcome != performance.OutcomeHeld {
		t.Errorf("an unapproved activity gave %s, want held", held.Outcome)
	}
	if held.ActivityID != "A-1" {
		t.Errorf("a held fact lost its activity: %q", held.ActivityID)
	}
	if counted.Outcome != performance.OutcomeMatched {
		t.Errorf("an approved activity gave %s, want matched", counted.Outcome)
	}
}

func TestDecide_DisagreementNamesBothCandidates_FR019(t *testing.T) {
	t.Parallel()

	got := performance.Decide(
		performance.RowKeys{ActivityID: "A-4", MediaCode: "9000009"},
		reaching(map[performance.MatchedOn]map[string][]string{
			performance.MatchedActivityID: {"A-4": {"A-4"}},
			performance.MatchedMediaCode:  {"9000009": {"A-5"}}}),
		allApproved)

	if len(got.Candidates) != 2 {
		t.Fatalf("Decide() named %v, want both candidates so a person can see the clash", got.Candidates)
	}
}

// Every row must land in exactly one of the five outcomes, whatever it carries — that is what makes the
// summary's five counts add up to rows read by construction rather than by a later check (FR-022).
func TestDecide_AlwaysExactlyOneOfTheFiveOutcomes_FR022(t *testing.T) {
	t.Parallel()
	values := []string{"", "known", "ambiguous", "other", "missing"}
	resolved := reaching(map[performance.MatchedOn]map[string][]string{
		performance.MatchedActivityID:    {"known": {"A-1"}, "ambiguous": {"A-1", "A-2"}, "other": {"A-9"}},
		performance.MatchedMediaCode:     {"known": {"A-1"}, "ambiguous": {"A-3", "A-4"}, "other": {"A-9"}},
		performance.MatchedIntegrationID: {"known": {"A-1"}, "other": {"A-9"}},
		performance.MatchedExternalName:  {"known": {"A-1"}, "other": {"A-9"}},
	})
	five := map[performance.Outcome]bool{
		performance.OutcomeMatched: true, performance.OutcomeHeld: true,
		performance.OutcomeUnattributed: true, performance.OutcomeRejected: true,
		performance.OutcomeUnmatched: true,
	}

	seen := map[performance.Outcome]int{}
	for _, keys := range everyCombination(values) {
		got := performance.Decide(keys, resolved, allApproved)
		if !five[got.Outcome] {
			t.Fatalf("Decide(%+v) gave outcome %q, which is not one of the five", keys, got.Outcome)
		}
		if got.Outcome != performance.OutcomeMatched && got.Reason == "" {
			t.Fatalf("Decide(%+v) gave %s with no reason; every non-match must state one", keys, got.Outcome)
		}
		seen[got.Outcome]++
	}
	for _, want := range []performance.Outcome{
		performance.OutcomeMatched, performance.OutcomeUnattributed, performance.OutcomeUnmatched} {
		if seen[want] == 0 {
			t.Errorf("no combination produced %s, so the sweep does not exercise it", want)
		}
	}
}

func everyCombination(values []string) []performance.RowKeys {
	var all []performance.RowKeys
	for _, a := range values {
		for _, m := range values {
			for _, i := range values {
				for _, e := range values {
					all = append(all, performance.RowKeys{
						ActivityID: a, MediaCode: m, IntegrationID: i, ExternalName: e})
				}
			}
		}
	}
	return all
}
