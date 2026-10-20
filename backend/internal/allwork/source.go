package allwork

import (
	"context"
	"time"

	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
)

// Source reads everything All work shows, with the values the owning features calculate (plan, "allwork.
// Source"). It is the only coupling to features 002–008: internal/app implements it over their packages
// (D-196), and the tests of this package use a Source built in memory. Every method runs inside the caller's
// transaction, which ctx carries, and never computes a business rule of its own.
type Source interface {
	// Years returns the years that have plans, in any order.
	Years(ctx context.Context) ([]int, error)
	// Year returns one Node per plan of year and per campaign, initiative, activity and line item under
	// them, never records in the recycle bin. today is the day statuses and timings are worked out for.
	// columns are the column keys the view shows or sorts by: a value only other columns show (a plan's
	// completeness, goals) may be left out, because the owning features work them out record by record.
	Year(ctx context.Context, year int, today date.Date, columns []string) (YearData, error)
	// Choices returns the values of the filters that do not come from the year's records: every team (the
	// person's own marked Mine), the options of the plan-level Business unit attribute, every activity type
	// and every person, active and inactive.
	Choices(ctx context.Context, personID string) (Choices, error)
	// Attributes returns every Shell attribute, active and inactive, level by level top-down and in
	// Settings' display order within a level: the attribute columns of the grid (FR-018).
	Attributes(ctx context.Context) ([]Attribute, error)
	// Currency returns the person's display currency and its current rate (D-010, feature 003).
	Currency(ctx context.Context, personID string) (Currency, error)
	// Locate returns the stored record recordID — a plan, campaign, initiative, activity or line item — and
	// false when it is not stored.
	Locate(ctx context.Context, recordID string) (Location, bool, error)
	// InRecycleBin reports whether the record recordID is in the recycle bin: in an entry not yet undone or
	// restored.
	InRecycleBin(ctx context.Context, recordID string) (bool, error)
}

// YearData is the input of every calculation: the nodes of one year's plans, in any order.
type YearData struct {
	Nodes []Node
	// FinanceAsOf is when the newest GSAP data was loaded (feature 015, FR-008), for the export's note; nil
	// when none ever was.
	FinanceAsOf *time.Time
}

// Node is one plan, campaign, initiative, activity or line item of the year (data model §2), with the
// values its owning feature calculates. Amounts are USD; All work converts them once, after summing.
type Node struct {
	Level    Level
	ID       string
	ParentID string // "" for plans
	PlanID   string
	Order    int // position among its siblings in the plan tree (005, 006) or in creation order (line items)

	Name         string
	ExternalName string // activities
	Description  string
	Owner        *Owner // nil when the record has none

	StartDate *date.Date // campaigns, initiatives, activities (plans: none, the whole year)
	EndDate   *date.Date // nil for ongoing activities
	Ongoing   bool
	Months    []string // activities: the months of their dates, "2026-03" (007's ActivityMonths); empty without dates

	// Status is the status value the owning feature works out (domain model §2): the plan status (draft,
	// published); a campaign's or initiative's Approval progress (noActivities, inProgress, allApproved); an
	// activity's display status (created … complete). Line items have none: their status is their
	// completeness (FR-011).
	Status   string
	Approved int    // campaigns and initiatives: activities below that are approved, of Total below
	Total    int    // campaigns and initiatives: activities below
	Timing   string // campaigns and initiatives: notStarted, running or finished; "" without dates
	Missing  int    // the record's own needed fields still empty (completeness)
	Swbs     string // line items: their SWBS request (notRequested, waitingForApproval, readyToSend)

	Budget           *money.Amount // plans, campaigns, initiatives: their own budget (USD); nil = none
	PlannedCost      *money.Amount // activities as typed (nil = none); line items converted to USD
	LocalPlannedCost *money.Money  // line items: their planned cost in their own currency
	MonthsUSD        []MonthAmount // line items: their spread in USD, by month (007's spread functions)
	UnspreadUSD      money.Amount  // line items: the USD amount that cannot be spread while the activity has no dates

	INQ, MQL *int64        // activities: goals as shown (006)
	Revenue  *money.Amount // activities: revenue goal as shown, USD

	CreatedAt, UpdatedAt time.Time

	Strategic       bool     // plans: Strategic / AOP plan
	TeamIDs         []string // plans: their teams
	BusinessUnitIDs []string // plans: the options of their Business unit
	ActivityTypeID  string   // activities

	Fields map[string]Value // the level-specific columns (data model §6.2) by column key; missing = empty

	// Finance is a line item's figures from its GSAP transactions (feature 015, finance.go); nil for a line
	// item without transactions. Activities and containers add up their line items'.
	Finance *FinanceFacts
}

// Value is the value of a level-specific column of one record, as its owning feature stores it, with the
// names of what it refers to. At most one part is set; none is an empty value ("—").
type Value struct {
	Text    string        // a text, or a name (inactive when Inactive)
	Number  *int64        // a whole number
	USD     *money.Amount // an amount in USD, shown in the display currency
	Money   *money.Money  // an amount shown in its own currency (a line item's local amounts)
	Yes     *bool         // a yes/no value
	Choices []Choice      // the options of a choice value, in order; with shares for a % split
	// Inactive marks a Text that names an inactive record: a model, activity type, workflow or budget item.
	Inactive bool
}

// Choice is one option of a choice value.
type Choice struct {
	Label    string
	Share    *money.Percent // a % split's share; nil otherwise
	Inactive bool
}

// Attribute is a Shell attribute as a column of its level's group (data model §6.2).
type Attribute struct {
	ID     string
	Label  string
	Level  Level
	Number bool // a number attribute; the others show text
	Active bool
}

// Choices are the values of the filters that do not come from the year's records (FR-004). Each Option
// has its ID, its name and whether it is inactive; Settings' order is kept.
type Choices struct {
	Teams         []Option // every team; Mine marks the person's own
	BusinessUnits []Option // the options of the plan-level Business unit attribute, in Settings' order
	ActivityTypes []Option // every activity type, by name
	People        []Option // every person: the owners a query or a saved view may name
	// SeesFinance says whether the person may see financial data (feature 015, FR-028): without it, the
	// Financials column group and filter are not offered and the figures are not worked out.
	SeesFinance bool
}

// MonthAmount is a line item's amount in one month, in USD.
type MonthAmount struct {
	Month string // "2026-03"
	USD   money.Amount
}

// Currency is a display currency with its rate in units per 1 USD.
type Currency struct {
	Code string
	Rate money.Rate
}

// usd is the display currency of a person who never chose one.
var usd = Currency{Code: money.BaseCurrency, Rate: money.One}

// Location is a stored record as Locate finds it.
type Location struct {
	Level      Level
	Name       string
	PlanID     string
	ActivityID string // line items
}
