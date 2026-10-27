package campaign

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Activities is what campaigns needs from activities (005 plan, deps.go; D-195): their roll-up facts and
// the copy of an initiative's activities. Feature 006 implements it through an adapter in internal/app;
// until then NoActivities is used. Every method runs inside the caller's transaction.
type Activities interface {
	// Summaries returns one summary per activity under the given initiatives (any order).
	Summaries(ctx context.Context, initiativeIDs []string) ([]ActivitySummary, error)
	// CopyAll copies every activity of fromInitiativeID, with its line items, into toInitiativeID (R13: new
	// IDs, Created, no approvals, comments, history or GSAP values), moving every date by yearShift years
	// (target plan year − source plan year), inside the caller's Write; it writes no log entries.
	CopyAll(ctx context.Context, a actor.Actor, fromInitiativeID, toInitiativeID string, yearShift int) (ActivityCopyCounts, error)
	// CopyMany is CopyAll for many initiatives at once, initiative by initiative in the order of pairs.
	CopyMany(ctx context.Context, a actor.Actor, pairs []CopyPair, yearShift int) (ActivityCopyCounts, error)
}

type CopyPair struct {
	FromID, ToID string
}

// ActivitySummary is one activity as campaigns rolls it up.
type ActivitySummary struct {
	ID, InitiativeID, Name, TypeID string
	PlannedCost                    *money.Money // USD; nil when empty
	Approved                       bool         // stored status Approved (shown Approved, In progress or Complete)
	StartDate, EndDate             *date.Date   // EndDate nil for ongoing activities
	Ongoing                        bool
	INQ, MQL                       *int64       // as feature 006 shows them (calculated or typed)
	Revenue                        *money.Money // USD
	Incomplete                     bool         // its own completeness is not "All filled in"
	LineItems                      int
	IncompleteLineItems            []errs.RecordRef
}

// ActivityCopyCounts is how many records a CopyAll copied.
type ActivityCopyCounts struct {
	Activities, LineItems int
}

// NoActivities is the Activities of a build without feature 006: no activities, nothing to copy.
type NoActivities struct{}

// Summaries returns no activities.
func (NoActivities) Summaries(context.Context, []string) ([]ActivitySummary, error) {
	return []ActivitySummary{}, nil
}

// CopyAll copies nothing.
func (NoActivities) CopyAll(context.Context, actor.Actor, string, string, int) (ActivityCopyCounts, error) {
	return ActivityCopyCounts{}, nil
}

func (NoActivities) CopyMany(context.Context, actor.Actor, []CopyPair, int) (ActivityCopyCounts, error) {
	return ActivityCopyCounts{}, nil
}

// The compiler checks here that NoActivities implements Activities.
var _ Activities = NoActivities{}
