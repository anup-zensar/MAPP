package people

import "context"

// The hooks below let later features tell people what they use (003 plan, "Constructor, hooks and
// wiring"), so that people never imports them. internal/app sets them before the API serves requests: 004
// UsePlans, 005/006/008 AddPersonUsage, 002 UseCurrencies. Every hook runs inside the people service's
// transaction, which its context carries.

// PlanUse is one plan as people needs it: its owner and teams, and whether it is in the recycle bin.
type PlanUse struct {
	ID, Name     string
	OwnerID      string   // "" when the plan has no owner
	TeamIDs      []string // the plan's teams
	InRecycleBin bool
}

// PlanSource lists every plan, stored and in the recycle bin (feature 004).
type PlanSource func(ctx context.Context) ([]PlanUse, error)

// PersonUsage counts the records of one type that name a person in one role ("owner" of activities,
// "approver" of approval workflows), records in the recycle bin not counted.
type PersonUsage struct {
	Role             string // "owner" or "approver"
	RecordType       string // "campaign", "initiative", "activity", "approvalWorkflow"
	Singular, Plural string // "activity" / "activities"
	Count            func(ctx context.Context, personID string) (int, error)
}

// Currency is a currency a person can show amounts in: its code, name, current rate in units per 1 USD as
// decimal text ("0.92"), and whether it is active (from feature 002).
type Currency struct {
	Code, Name, Rate string
	Active           bool
}

// CurrencySource lists every currency, active or not (feature 002, through an adapter in internal/app).
type CurrencySource func(ctx context.Context) ([]Currency, error)

// OnlyUSD is the CurrencySource until feature 002 is wired: US dollars only (spec Assumptions).
func OnlyUSD(context.Context) ([]Currency, error) {
	return []Currency{{Code: DefaultDisplayCurrency, Name: "US dollar", Rate: "1", Active: true}}, nil
}
