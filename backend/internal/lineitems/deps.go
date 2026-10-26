package lineitems

import (
	"cmp"
	"context"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/settings"
)

// Line items read activities, plans, Settings and people, while those features read line-item numbers. To
// avoid import cycles, this package declares what it needs as the small interfaces below (D-196);
// internal/app implements them over the other features and hands them to NewService. Every method runs
// inside the caller's transaction, which ctx carries.

// Deps are the other features line items read.
type Deps struct {
	Activities     Activities
	Currencies     Currencies
	FinanceBudgets FinanceBudgets
	Attributes     Attributes
	People         People
}

// Activities is what line items need from activities (feature 006) and their plans (feature 004).
type Activities interface {
	// Facts returns the activity activityID with its plan's facts, and false when the activity is not
	// stored (deleted, or never there).
	Facts(ctx context.Context, activityID string) (ActivityFacts, bool, error)
	// FactsOf returns the facts of each stored activity among activityIDs, reading each level once for all.
	FactsOf(ctx context.Context, activityIDs []string) (map[string]ActivityFacts, error)
	// RestartApproval sends a Submitted or Approved activity back to Created (006 FR-027; 008 ends the open
	// round), logging the reasons, inside the caller's Write.
	RestartApproval(ctx context.Context, a actor.Actor, activityID string, reasons []string) error
}

// ActivityFacts are what the line-item rules need to know about an activity and its plan.
type ActivityFacts struct {
	ID, Name             string
	OwnerID              string // "" when the activity has no owner
	Status               string // stored status: "created", "submitted", "declined" or "approved"
	StartDate, EndDate   *date.Date
	Ongoing              bool
	PlannedCost          *money.Amount    // in PlanCurrency (016); nil when not set
	Path                 []errs.RecordRef // plan, campaign, initiative, top level first
	PlanYear             int
	DefaultSpread        SpreadMethod    // the plan's default line-item spread
	DefaultFinanceBudget string          // the name of the plan's default finance budget; "" when none
	BusinessUnit         []string        // the plan's Business unit option ids: which Shell attributes are shown
	Access               authz.PlanFacts // the plan as access decisions need it (feature 014)
	PlanCurrency         string          // 016 FR-011: the plan currency; "" reads as USD
}

// planCurrency is the plan currency of the activity's amounts: USD when the facts have none (016).
func (f ActivityFacts) planCurrency() string {
	return cmp.Or(f.PlanCurrency, money.BaseCurrency)
}

// restartsApproval reports whether a change that matters for approval sends the activity back to Created
// (FR-031): it is Submitted or Approved.
func (f ActivityFacts) restartsApproval() bool {
	return f.Status == "submitted" || f.Status == "approved"
}

// Currencies are the currencies of Settings (feature 002), active or not.
type Currencies interface {
	// Currency returns the currency with the ISO code, active or not, and false when there is none.
	Currency(ctx context.Context, code string) (settings.Currency, bool, error)
	// Currencies returns every currency, active or not.
	Currencies(ctx context.Context) ([]settings.Currency, error)
	// Rates returns the current rate of every currency (016 FR-002).
	Rates(ctx context.Context) (money.Rates, error)
	// CurrentRates returns every currency's current rate with its date and source (016 FR-009).
	CurrentRates(ctx context.Context) ([]settings.RateInfo, error)
}

// BudgetItem is a finance budget item with its budget's year and name (feature 002).
type BudgetItem struct {
	ID         string
	Year       int
	BudgetName string
	Name       string
	Code       string
	Active     bool
}

// FinanceBudgets are the finance budget items of Settings (feature 002), active or not.
type FinanceBudgets interface {
	// Item returns the finance budget item with id, active or not, and false when there is none.
	Item(ctx context.Context, id string) (BudgetItem, bool, error)
	// Items returns every finance budget item of every year, active or not.
	Items(ctx context.Context) ([]BudgetItem, error)
}

// Attributes are feature 002's rules for the line-item Shell attributes (Spend type, GL code, …).
type Attributes interface {
	// Check checks the values of a save (002's CheckAttributes at level lineItem): its Problems block saving
	// (fields "attributes.<id>"), its Values are what to store.
	Check(ctx context.Context, in settings.AttributeInput) (settings.AttributeResult, error)
	// Missing lists the visible, active, needed-to-submit line-item attributes without a value, in Settings
	// order (002's MissingAttributes; D-034).
	Missing(ctx context.Context, values settings.AttributeValues, businessUnit []string) ([]settings.MissingField, error)
	// Describe gives the label and the text of one attribute value for the activity log; field is
	// "attributes.<id>" (002's DescribeAttribute).
	Describe(ctx context.Context, field string, value any) (label, text string, err error)
}

// People are the people of feature 003.
type People interface {
	// Person returns the person with id as records show them, active or not, and false when there is none.
	Person(ctx context.Context, id string) (people.PersonRef, bool, error)
	// DisplayRateOf returns the display currency the person personID shows amounts in, with its rate in
	// units per 1 USD (USD and "1" when they have not chosen one; 003 FR-023, D-213).
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}
