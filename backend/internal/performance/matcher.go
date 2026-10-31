package performance

// RowKeys: an empty field is no key at all, unlike a key present but resolving to nothing (046-C20).
type RowKeys struct {
	ActivityID    string
	MediaCode     string
	IntegrationID string
	ExternalName  string
}

func (k RowKeys) valueOf(on MatchedOn) string {
	switch on {
	case MatchedActivityID:
		return k.ActivityID
	case MatchedMediaCode:
		return k.MediaCode
	case MatchedIntegrationID:
		return k.IntegrationID
	case MatchedExternalName:
		return k.ExternalName
	case MatchedNone:
	}
	return ""
}

// Resolved is built once per load, one query per key type, never per row (research R2, plan R16).
type Resolved map[MatchedOn]map[string][]string

// Decision carries exactly one Outcome, so the five account for every row read (FR-022).
type Decision struct {
	ActivityID string
	MatchedOn  MatchedOn
	Outcome    Outcome
	Reason     string
	Candidates []string
}

// Decide is pure: no database, no clock, so every rule below is testable on its own (046-C1).
func Decide(keys RowKeys, resolved Resolved, approved func(activityID string) bool) Decision {
	present := presentKeys(keys)
	if len(present) == 0 {
		return Decision{Outcome: OutcomeUnattributed, Reason: reasonNoKey}
	}
	var reached []string
	for _, on := range present {
		hit := resolved[on][keys.valueOf(on)]
		if len(hit) == 0 {
			return Decision{Outcome: OutcomeUnmatched, Reason: reasonUnknown}
		}
		if len(hit) > 1 {
			return Decision{Outcome: OutcomeUnmatched, Reason: reasonAmbiguous, Candidates: hit}
		}
		reached = append(reached, hit[0])
	}
	for _, id := range reached[1:] {
		if id != reached[0] {
			return Decision{Outcome: OutcomeUnmatched, Reason: reasonDisagree, Candidates: distinct(reached)}
		}
	}
	decided := Decision{ActivityID: reached[0], MatchedOn: present[0], Outcome: OutcomeMatched}
	if !approved(decided.ActivityID) {
		decided.Outcome = OutcomeHeld
	}
	return decided
}

func presentKeys(keys RowKeys) []MatchedOn {
	var present []MatchedOn
	for _, on := range MatchOrder() {
		if keys.valueOf(on) != "" {
			present = append(present, on)
		}
	}
	return present
}

func distinct(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
