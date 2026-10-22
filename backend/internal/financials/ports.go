package financials

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/money"
)

// The ports of financials (D-196): what it reads from other features, as consumer interfaces that
// internal/app implements (app/financials.go) and the tests fake. Every method runs inside the caller's
// transaction, which ctx carries.

// Planning reads the live planning tree of features 004–007: records in the recycle bin are not in it.
type Planning interface {
	// Nodes returns every live plan, campaign, initiative, activity and line item, each with its parent
	// and, among its siblings, its place in the plan tree.
	Nodes(ctx context.Context) ([]Node, error)
	// LineItemsInBin returns the ids of the line items held by a recycle-bin entry not yet undone or
	// restored: deleted themselves or with a record above them. Matching treats them as stored (data model
	// §4.2); a purged line item is neither live nor in the bin.
	LineItemsInBin(ctx context.Context) ([]string, error)
	// MyTeamPlanIDs returns the ids of the plans that have at least one of the person's teams (Home's My
	// teams, 009 FR-003).
	MyTeamPlanIDs(ctx context.Context, personID string) ([]string, error)
}

// Node is one live planning record.
type Node struct {
	Level    Level
	ID       string
	ParentID string // "" for plans
	PlanID   string
	Name     string
	PlanYear int // the year of the record's plan
	// Order is the record's place among its siblings: the plan tree's order for campaigns, initiatives and
	// activities (005 FR-020), creation order for line items. Plans are ordered by name.
	Order int
	// PlannedCost is in USD: an activity's planned cost as typed, a line item's converted at the current
	// rate (007); nil when there is none. Plans, campaigns and initiatives leave it nil: their planned cost
	// is the roll-up of their activities (D-090), worked out here.
	PlannedCost *money.Amount
}

// Rates returns the current exchange rate of every currency of Settings, active or not (002,
// settings.(*Service).Rates). A transaction converts at its currency's current rate (FR-010, D-303).
type Rates interface {
	Rates(ctx context.Context) (money.Rates, error)
}

// Displays returns the person's display currency with its rate (003, D-061).
type Displays interface {
	Display(ctx context.Context, personID string) (money.Display, error)
}

// Access decides who may see financial data, import GSAP extracts and link transactions (FR-028, D-252,
// D-301): feature 014's permissions finance.view (scoped), finance.import and finance.link (scoped to the
// line item's plan). Each check returns nil when allowed, or the 403 forbidden error with 014's message.
// internal/app implements it; AllowAll is the rule of the unit tests that do not test access.
type Access interface {
	// SeeFinance allows a to see financial data in at least one plan (Need finance.view).
	SeeFinance(ctx context.Context, a actor.Actor) error
	// SeeFinanceIn allows a to see the plan planID and its financial data (View + InPlan finance.view);
	// a plan outside a's access is refused with 014's message 4.
	SeeFinanceIn(ctx context.Context, a actor.Actor, planID string) error
	// Import allows a to import GSAP extracts (Need finance.import). Not-linked transactions are visible
	// only to people allowed to import.
	Import(ctx context.Context, a actor.Actor) error
	// Link allows a to link and unlink transactions in the plan planID (InPlan finance.link); "" asks
	// whether a may link in any plan.
	Link(ctx context.Context, a actor.Actor, planID string) error
	// VisiblePlanIDs returns nil for every plan, or the ids of the plans a can see and holds finance.view
	// in: lists, counts and totals leave out the transactions of other plans (014 FR-007).
	VisiblePlanIDs(ctx context.Context, a actor.Actor) ([]string, error)
}

// AllowAll is the Access of the unit tests that do not test access: everyone may do everything.
type AllowAll struct{}

// SeeFinance allows everyone.
func (AllowAll) SeeFinance(context.Context, actor.Actor) error { return nil }

// SeeFinanceIn allows everyone.
func (AllowAll) SeeFinanceIn(context.Context, actor.Actor, string) error { return nil }

// Import allows everyone.
func (AllowAll) Import(context.Context, actor.Actor) error { return nil }

// Link allows everyone.
func (AllowAll) Link(context.Context, actor.Actor, string) error { return nil }

// VisiblePlanIDs gives every plan.
func (AllowAll) VisiblePlanIDs(context.Context, actor.Actor) ([]string, error) { return nil, nil }

// LineItemStates answers whether a line item id is stored, live or in the recycle bin, for matching rules
// 1 and 2 (data model §4.2). The seed's demo load has no recycle bin and implements it over the line-items
// repository; the service builds one from Planning.
type LineItemStates interface {
	// Stored reports whether the line item id is live or in the recycle bin.
	Stored(id string) bool
}

// storedSet is a LineItemStates over a set of ids.
type storedSet map[string]bool

// Stored reports whether id is in the set.
func (s storedSet) Stored(id string) bool { return s[id] }
