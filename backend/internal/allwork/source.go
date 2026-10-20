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
	// RatesUsed returns the current rate of every currency other than USD as an export's "About this export"
	// states it (016 FR-009): "1 USD = 0.79 GBP · rate of 19 Oct 2026 · GSAP import".
	RatesUsed(ctx context.Context) ([]string, error)
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

	Budget           *money.Amount // plans, campaigns, initiatives: their own budget, converted to USD (016); nil = none
	PlannedCost      *money.Amount // activities as typed, converted to USD (016; nil = none); line items converted to USD
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
	// Feature 016 (data-model §5.1): the plan's year, type and currency, the amounts in the plan currency and the
	// phased budget. Budget, PlannedCost, MonthsUSD and UnspreadUSD above stay USD.
	PlanYear        int            // every level: the year of the record's plan (a range spans several years)
	PlanType        string         // every level: the type of the record's plan (operational, strategic, crossMarket)
	PlanCurrency    string         // every level: the currency of the record's plan
	BudgetPlan      *money.Money   // plans, campaigns, initiatives: the budget in the plan currency; nil = none
	PlannedCostPlan *money.Money   // activities as typed, line items converted, in the plan currency; nil = none
	PhasingMethod   string         // plans, campaigns, initiatives: none, evenly or custom
	PhasedPlan      []money.Amount // plans, campaigns, initiatives: the 12 phased budgets of PlanYear in the plan currency; nil = not phased
	PhasedBudgetUSD []MonthAmount  // the same months in USD (money.ConvertParts); nil = not phased
	MonthsPlan      []PlanMonth    // line items: their spread in the plan currency, by month
	UnspreadPlan    money.Amount   // line items: the plan-currency amount not spread while the activity has no dates
	// Attrs are the record's own Shell attribute values of choice and yes/no attributes by attribute id, for the
	// attribute filters and the Breakdown (016 FR-039 – FR-041); a missing entry is not filled in.
	Attrs map[string]AttrValue
}

// AttrValue is a record's value of a choice or yes/no Shell attribute (016 data-model §5.1): the chosen
// options in their order, with the shares of a % split, or the yes/no answer (an empty yes/no reads as No).
type AttrValue struct {
	OptionIDs []string
	Shares    map[string]money.Percent // % split only: the share of each option
	Yes       *bool
}

// PlanMonth is a line item's amount in one month, in its plan's currency (016 FR-024).
type PlanMonth struct {
	Month  string // "2027-03"
	Amount money.Amount
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

// Attribute is a Shell attribute as a column of its level's group (data model §6.2), and, since 016, as an
// attribute filter and a Breakdown grouping (FR-039, FR-040).
type Attribute struct {
	ID     string
	Label  string
	Level  Level
	Number bool // a number attribute; the others show text
	Active bool
	// 016: the attribute's type (singleChoice, multipleChoice, multipleChoiceSplit, yesNo, …), the Business
	// unit options it is shown for (empty: always shown) and its options in Settings' order.
	Type      string
	Condition []string
	Options   []Option
}

// The attribute types the filters and the Breakdown use (002's AttributeType).
const (
	TypeSingleChoice        = "singleChoice"
	TypeMultipleChoice      = "multipleChoice"
	TypeMultipleChoiceSplit = "multipleChoiceSplit"
	TypeYesNo               = "yesNo"
)

// filterable reports whether the attribute can be a filter or a Breakdown grouping: an active choice or yes/no
// attribute (FR-039).
func (a Attribute) filterable() bool {
	switch a.Type {
	case TypeSingleChoice, TypeMultipleChoice, TypeMultipleChoiceSplit, TypeYesNo:
		return a.Active
	}
	return false
}

// shownFor reports whether the attribute is shown for a plan with the Business unit options units: always
// when it has no condition (002 FR-022).
func (a Attribute) shownFor(units []string) bool {
	return len(a.Condition) == 0 || containsAny(a.Condition, units)
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
	// FinancePlans are the plans whose financial data the person may see (015 consolidation, D-301): nil
	// for every plan. The rows of other plans show "—" in the Financials columns, never match the
	// Financials filter and add nothing to the totals.
	FinancePlans []string
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
	Year       int    // plans: their year (016, Budget by month); 0 for the other levels
}
