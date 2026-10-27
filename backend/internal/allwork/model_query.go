package allwork

// The values of the single-choice filters and switches (contract parameters).
const (
	StrategicInclude = "include"
	StrategicOnly    = "only"
	StrategicExclude = "exclude"

	ShowEverything                   = "everything"
	ShowCompleteBranches             = "completeBranches"
	ShowCampaignsWithoutInitiatives  = "campaignsWithoutInitiatives"
	ShowInitiativesWithoutActivities = "initiativesWithoutActivities"

	PeriodMonth   = "month"
	PeriodQuarter = "quarter"
	PeriodYear    = "year" // 016 FR-035

	// The value of the attribute filters that means Not filled in, and the answers of a yes/no attribute (016
	// FR-039).
	AttrNotFilledIn = "none"
	AttrYes         = "yes"
	AttrNo          = "no"

	// AmountsInDisplay and AmountsInPlan say the currency of a Breakdown (016 FR-042).
	AmountsInDisplay = "display"
	AmountsInPlan    = "plan"
)

// ViewBreakdown is the Breakdown view (016 FR-040): GET /api/v1/all-work answers it as the Grid, the groups come
// from GET /api/v1/all-work/breakdown, and the export holds the group table.
const ViewBreakdown = "breakdown"

// AttrFilter is one Shell-attribute filter (contract AllWorkAttributeFilter, 016 FR-039): the attribute and
// the values that match: option ids, yes, no, or none for Not filled in.
type AttrFilter struct {
	AttributeID string   `json:"attributeId"`
	Values      []string `json:"values"`
}

// Query is one view's filter set and display choices, as GET /api/v1/all-work receives them (FR-004). Only the
// filters of FR-004 exist (FR-010, D-207): any other parameter is ignored.
type Query struct {
	Year              int    // the first year of the range (yearFrom); 0: the current year
	View              string // grid, calendar, budget or breakdown
	Plans             []string
	PlanStatuses      []string
	ContainerStatuses []string
	ActivityStatuses  []string
	Owners            []string
	Teams             []string
	BusinessUnits     []string
	ActivityTypes     []string
	Finance           []string // the Financials filter of feature 015 (FR-024): finance.go's values
	Strategic         string   // include, only or exclude
	ShowLevels        string
	Search            string   // trimmed
	Columns           []string // in display order; nil when the query names none: the default columns
	Sort              *Sort    // nil: the plan-tree order
	Period            string   // month, quarter or year

	// 016: the range of years (0: the current year; YearTo - YearFrom <= 4), the plan type and plan currency
	// filters (tested on plans), the Shell-attribute filters (tested on each attribute's level), the phased
	// budget beside Budget & cost, and the groupings and currency of the Breakdown.
	YearFrom, YearTo int
	PlanTypes        []string
	PlanCurrencies   []string
	Attributes       []AttrFilter
	ComparePhased    bool
	By, ThenBy       string
	AmountsIn        string // display or plan

	attrs []attrTest // the attribute filters resolved with Settings' attributes (build)
}
