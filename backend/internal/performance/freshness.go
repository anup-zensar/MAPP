package performance

import "time"

// graceOf allows one extra period, so a feed that is merely a little late does not alarm anyone.
func graceOf(c Cadence) time.Duration {
	switch c {
	case CadenceDaily:
		return 2 * 24 * time.Hour
	case CadenceWeekly:
		return 9 * 24 * time.Hour
	case CadenceMonthly:
		return 35 * 24 * time.Hour
	case CadenceOnDemand:
		return 0
	}
	return 0
}

// FreshnessAt is derived, never stored: a stored value would go stale by itself (FR-005).
func (v SourceView) FreshnessAt(now time.Time) Freshness {
	if v.Status == StatusInactive {
		return FreshnessInactive
	}
	if v.LastLoadedAt.IsZero() {
		return FreshnessStale
	}
	if v.Cadence == CadenceOnDemand {
		return FreshnessCurrent
	}
	if now.Sub(v.LastLoadedAt) > graceOf(v.Cadence) {
		return FreshnessStale
	}
	return FreshnessCurrent
}
