package home

import (
	"context"
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// Sources is everything Home reads, as functions (009 plan "Inputs", D-171, D-196): internal/app fills them
// from the services of features 003–008 (home_sources.go), and the service tests from fakes. Home never
// reads another feature's storage, so every figure is the one its own feature calculates (FR-017). Each
// function joins the caller's transaction, which ctx carries. Records that are deleted are not stored, so
// no function returns them (FR-019).
type Sources struct {
	// MyTeams returns the teams the person a belongs to, active or not (003).
	MyTeams func(ctx context.Context, a actor.Actor) ([]Team, error)
	// Plans returns every plan with its figures in US dollars, as the Plans list and the plan page work
	// them out (004, with the roll-ups of 005–007).
	Plans func(ctx context.Context, a actor.Actor) ([]Plan, error)
	// Containers returns the campaigns and initiatives whose owner is ownerID, in every plan, with their
	// figures in US dollars as their pages work them out (005).
	Containers func(ctx context.Context, a actor.Actor, ownerID string) ([]Container, error)
	// Activities returns the activities whose owner is ownerID, in every plan, as their pages show them
	// (006, line items of 007), with the decline of each Declined one (008).
	Activities func(ctx context.Context, a actor.Actor, ownerID string) ([]Activity, error)
	// WaitingForMe returns the Waiting for me tab of the Approvals page for a: the Submitted activities
	// whose current level lists a, oldest submission first (008).
	WaitingForMe func(ctx context.Context, a actor.Actor) ([]Waiting, error)
	// DisplayCurrency returns a's display currency with its current rate (003, rates of 002). A display
	// currency set inactive stays in use with its rate (D-201, D-063).
	DisplayCurrency func(ctx context.Context, a actor.Actor) (Currency, error)
}

// MissingField is a field still needed to publish or submit (contract MissingField).
type MissingField = errs.MissingField

// Team is a team as Home reads it.
type Team struct {
	ID, Name string
}

// Plan is a plan with its figures, all amounts in US dollars.
type Plan struct {
	ID, Name                                            string
	Year                                                int
	StrategicAOP                                        bool
	Status                                              string        // "draft" or "published"
	OwnerID                                             string        // "" when none
	Teams                                               []Team        // by name
	Budget                                              *money.Amount // nil when none
	PlannedCost                                         *money.Amount // the roll-up (domain model §4.2); nil when nothing below has one
	RemainingBudget                                     *money.Amount // budget − Σ campaign budgets; nil without a budget
	CampaignCount, ActivityCount, ApprovedActivityCount int           // approved: Approved, In progress or Complete (§2.2)
	Missing                                             []MissingField
	ItemsBelow                                          int // records below that need attention
	UpdatedAt                                           time.Time
}

// Container is a campaign or an initiative, all amounts in US dollars.
type Container struct {
	Type            string // "campaign" or "initiative"
	ID, Name        string
	Plan            errs.RecordRef
	Path            []errs.RecordRef // its ancestors, plan first
	Budget          *money.Amount    // nil when none
	PlannedCost     *money.Amount    // the roll-up; nil when nothing below has one
	RemainingBudget *money.Amount    // campaigns: budget − Σ initiative budgets; nil without a budget and for initiatives
	Missing         []MissingField
	ItemsBelow      int // records below that need attention
	UpdatedAt       time.Time
}

// Activity is an activity as its page shows it, amounts in US dollars.
type Activity struct {
	ID, Name                   string
	Plan, Campaign, Initiative errs.RecordRef
	Status                     string // stored: created, submitted, declined or approved
	DisplayStatus              string // shown (domain model §2.3): also inProgress or complete
	StartDate, EndDate         *date.Date
	Ongoing                    bool
	PlannedCost                *money.Amount  // nil when not set
	Unassigned                 *money.Amount  // planned cost − line items (§4.3); nil with neither
	Missing                    []MissingField // its own fields still needed to submit
	LineItemsNeedingAttention  int
	Decline                    *Decline // the decision that declined it, while Declined
	UpdatedAt                  time.Time
}

// Decline is who declined an activity, when, and why.
type Decline struct {
	By      actor.Actor
	At      time.Time
	Comment string
}

// Waiting is one row of the Waiting for me tab.
type Waiting struct {
	ActivityID, ActivityName string
	Plan                     errs.RecordRef
	Owner                    *actor.Actor  // nil when the activity has none
	PlannedCost              *money.Amount // nil when not set
	Level, Levels            int           // "Level 1 of 2"
	WaitingSince             time.Time     // when the round reached its current level
}

// Currency is a display currency and its rate in units per 1 USD.
type Currency struct {
	Code string
	Rate money.Rate
}
