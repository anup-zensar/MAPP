// Package allwork is All work (feature 010): a read-only, cross-plan view of one year — every plan,
// campaign, initiative, activity and line item — as a Grid, a Calendar and a Budget & cost view under one
// filter bar, with summary cards, saved views and export.
//
// All work never changes a business record (FR-002). It reads the records of features 002–008 through one
// consumer interface, Source (source.go), which internal/app implements over those features (D-196), so the
// values the owning features calculate — statuses, completeness, USD amounts, spreads, goals — are reused,
// never re-implemented. Everything All work adds — matching, roll-ups, sorting, periods, conversion to the
// display currency — is a pure function of the year's nodes (tree.go, rollup.go, columns.go, …), worked out
// on the server for every request (D-114). The only records it stores are two kinds of personal settings:
// saved views and the remembered columns (D-121).
package allwork

import (
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// Level is the level of a row (contract AllWorkLevel): a record's level, or an unassigned planned cost row.
type Level string

// The levels, top-down, and the system row under an activity.
const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
	LevelUnassigned Level = "unassigned"
)

// Tone is the colour family of a status chip (contract AllWorkTone, design system §4).
type Tone string

// The tones.
const (
	ToneNeutral  Tone = "neutral"
	ToneReview   Tone = "review"
	TonePositive Tone = "positive"
	ToneActive   Tone = "active"
	ToneDone     Tone = "done"
	ToneProblem  Tone = "problem"
)

// The views of the page (contract AllWorkView).
const (
	ViewGrid     = "grid"
	ViewCalendar = "calendar"
	ViewBudget   = "budget"
)

// The roles of a row (FR-005).
const (
	RoleMatch   = "match"
	RoleContext = "context"
)

// The kinds of a cell (contract AllWorkCell.kind) and of a column (AllWorkColumn.kind).
const (
	KindEmpty    = "empty"
	KindText     = "text"
	KindMoney    = "money"
	KindNumber   = "number"
	KindPercent  = "percent"
	KindDate     = "date"
	KindDateTime = "dateTime"
	KindStatus   = "status"
)

// Status is a status chip: the value, its word, its colour family and an optional detail (AllWorkStatus).
type Status struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Tone   Tone   `json:"tone"`
	Detail string `json:"detail,omitempty"`
}

// Cell is one value of the grid (AllWorkCell). Kind says which field holds it; KindEmpty is "—" on the
// column's own level. A cell of another level's column is not a Cell but null.
type Cell struct {
	Kind       string       `json:"kind"`
	Text       string       `json:"text,omitempty"` // text cells, and the word of status cells
	Money      *money.Money `json:"money,omitempty"`
	Number     *int64       `json:"number,omitempty"`
	Percent    string       `json:"percent,omitempty"` // whole per cent, "80"
	Date       date.Date    `json:"date,omitzero"`
	DateTime   *time.Time   `json:"dateTime,omitempty"`
	Tone       Tone         `json:"tone,omitempty"`
	Inactive   bool         `json:"inactive,omitempty"`   // an inactive option, type or person: "(inactive)"
	Unfiltered *Cell        `json:"unfiltered,omitempty"` // the roll-up without filters (FR-013)
}

// SearchHit says which of a row's own fields contain the search text (AllWorkSearchHit).
type SearchHit struct {
	Fields  []string `json:"fields"`            // name, id, externalName, description
	Excerpt string   `json:"excerpt,omitempty"` // about 60 characters around the first hit in a long text
}

// Owner is a row's owner (AllWorkOwner).
type Owner struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Inactive bool   `json:"inactive"`
}

// Row is one row of a view (AllWorkRow), in final order. Money is in the display currency.
type Row struct {
	Key         string         `json:"key"`       // the record ID, or "<activity ID>:unassigned"
	ParentKey   *string        `json:"parentKey"` // null for plans
	Level       Level          `json:"level"`
	RecordID    string         `json:"recordId"` // the record the row opens (an unassigned row: its activity)
	PlanID      string         `json:"planId"`
	ActivityID  string         `json:"activityId,omitempty"` // line items and unassigned rows
	Depth       int            `json:"depth"`
	Name        string         `json:"name"`
	Role        string         `json:"role"`
	SearchHit   *SearchHit     `json:"searchHit,omitempty"`
	Strategic   *bool          `json:"strategic,omitempty"` // plans only: the AOP tag
	Status      *Status        `json:"status"`              // null for unassigned rows
	Timing      *Status        `json:"timing,omitempty"`    // campaigns and initiatives with dates
	StartDate   date.Date      `json:"startDate,omitzero"`
	EndDate     date.Date      `json:"endDate,omitzero"`
	Ongoing     bool           `json:"ongoing"`
	Owner       *Owner         `json:"owner,omitempty"`
	PlannedCost *money.Money   `json:"plannedCost,omitempty"`
	Balance     *money.Money   `json:"balance,omitempty"` // activities: their unassigned planned cost
	Budget      *money.Money   `json:"budget,omitempty"`  // plans, campaigns, initiatives: their own budget
	HasChildren bool           `json:"hasChildren"`
	Filtered    bool           `json:"filtered,omitempty"` // containers: filters hide records below (FR-013)
	Cells       []*Cell        `json:"cells"`              // one per column of Result.Columns; nil = blank
	Periods     []*money.Money `json:"periods,omitempty"`  // Budget & cost: one per Result.Periods; nil = "—"
	Total       *money.Money   `json:"total"`              // Budget & cost: the row's total; null otherwise
	// 016: the plan's year (plan rows, FR-034), the plan type of the record's plan (FR-032) and, with Compare
	// with phased budget, the phased budget per period and the periods where planned cost is above it (FR-038).
	Year         *int           `json:"year,omitempty"`
	PlanType     string         `json:"planType,omitempty"`
	PhasedBudget []*money.Money `json:"phasedBudget,omitempty"`
	OverPeriods  []int          `json:"overPeriods,omitempty"`
}

// Column is one column of the catalogue (AllWorkColumn).
type Column struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Group    string  `json:"group"` // standard, plan, campaign, initiative, activity, lineItem
	Kind     string  `json:"kind"`
	Levels   []Level `json:"levels"` // the levels on which the column has values
	Sortable bool    `json:"sortable"`
	Inactive bool    `json:"inactive"`
}

// ColumnGroup is one group of the column chooser (AllWorkColumnGroup).
type ColumnGroup struct {
	Group   string   `json:"group"`
	Label   string   `json:"label"`
	Columns []Column `json:"columns"`
}

// Period is one period column of Budget & cost (AllWorkPeriod).
type Period struct {
	Key   string `json:"key"` // 2026-01 … 2026-12, Q1-2026 … Q4-2026, 2026, other or noDates (016 FR-035)
	Label string `json:"label"`
}

// Summary is the six summary cards (AllWorkSummary, FR-003). Money is null when nothing has a value.
type Summary struct {
	Campaigns             int          `json:"campaigns"`
	Initiatives           int          `json:"initiatives"`
	Activities            int          `json:"activities"`
	PlannedCost           *money.Money `json:"plannedCost"`
	PlanBudget            *money.Money `json:"planBudget"`
	UnassignedPlannedCost *money.Money `json:"unassignedPlannedCost"`
	NegativeBalanceCount  int          `json:"negativeBalanceCount"`
}

// Count is "Showing N of M <noun>" (AllWorkCount, FR-008).
type Count struct {
	Shown int    `json:"shown"`
	Total int    `json:"total"`
	Noun  string `json:"noun"` // activities, campaigns or initiatives
}

// Totals is the totals row (AllWorkTotals, FR-014).
type Totals struct {
	Activities  int            `json:"activities"`
	Cells       []*Cell        `json:"cells"`
	PlannedCost *money.Money   `json:"plannedCost"`
	PlanBudget  *money.Money   `json:"planBudget"`
	Periods     []*money.Money `json:"periods,omitempty"`
	Total       *money.Money   `json:"total"`
}

// LeftOut is a value of the query that no longer exists and was ignored (AllWorkLeftOut, FR-031).
type LeftOut struct {
	Filter string `json:"filter"` // plan, owner, team, businessUnit, activityType, column, sort
	ID     string `json:"id"`
}

// Result is everything one view shows for a filter set (AllWorkResult).
type Result struct {
	Year            int       `json:"year"` // the first year of the range (016)
	YearFrom        int       `json:"yearFrom"`
	YearTo          int       `json:"yearTo"`
	Today           date.Date `json:"today"`
	View            string    `json:"view"`
	DisplayCurrency string    `json:"displayCurrency"`
	GeneratedAt     time.Time `json:"generatedAt"`
	Columns         []Column  `json:"columns"` // the requested columns that exist, in order (grid only)
	Periods         []Period  `json:"periods"` // Budget & cost only
	Rows            []Row     `json:"rows"`
	Summary         Summary   `json:"summary"`
	Count           Count     `json:"count"`
	Totals          Totals    `json:"totals"`
	LeftOut         []LeftOut `json:"leftOut"`
}

// Option is one choice of a filter (AllWorkOption).
type Option struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Inactive bool   `json:"inactive"`
	Mine     bool   `json:"mine,omitempty"` // teams only: the signed-in person belongs to it
}

// PlanOption is one plan of the Plan filter (AllWorkPlanOption).
type PlanOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"` // draft or published
	Strategic bool   `json:"strategic"`
	Year      int    `json:"year"`     // 016
	PlanType  string `json:"planType"` // 016
	Currency  string `json:"currency"` // 016
}

// FilterableAttribute is a Shell attribute the attribute filters and the Breakdown offer (contract
// AllWorkFilterableAttribute, 016 FR-039): its options, or Yes and No.
type FilterableAttribute struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Level  Level    `json:"level"`
	Type   string   `json:"type"`
	Values []Option `json:"values"`
}

// Options are the choices of the filter bar for a year and the column catalogue (AllWorkOptions).
type Options struct {
	Year           int           `json:"year"`
	CurrentYear    int           `json:"currentYear"`
	Years          []int         `json:"years"` // years with plans and the current year, newest first
	Plans          []PlanOption  `json:"plans"`
	Owners         []Option      `json:"owners"`
	Teams          []Option      `json:"teams"`
	BusinessUnits  []Option      `json:"businessUnits"`
	ActivityTypes  []Option      `json:"activityTypes"`
	ColumnGroups   []ColumnGroup `json:"columnGroups"`
	DefaultColumns []string      `json:"defaultColumns"`
	// 016: the range, the plan types, the plan currencies of the range's plans and the filterable attributes.
	YearFrom       int                   `json:"yearFrom"`
	YearTo         int                   `json:"yearTo"`
	PlanTypes      []string              `json:"planTypes"`
	PlanCurrencies []string              `json:"planCurrencies"`
	Attributes     []FilterableAttribute `json:"attributes"`
}

// RecordLocation is a row's record that still exists (AllWorkRecordLocation).
type RecordLocation struct {
	RecordID   string `json:"recordId"`
	Level      Level  `json:"level"`
	Name       string `json:"name"`
	PlanID     string `json:"planId"`
	ActivityID string `json:"activityId,omitempty"` // line items: their activity
}

// Sort is a column and a direction (AllWorkSort): children are ordered within each parent (FR-015).
type Sort struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // asc or desc
}

// The sort directions.
const (
	SortAscending  = "asc"
	SortDescending = "desc"
)

// Preferences are a person's remembered columns, their order and the sort (AllWorkPreferences, FR-021).
type Preferences struct {
	Columns []string `json:"columns"`
	Sort    *Sort    `json:"sort"`
}

// FilterValue is a chosen ID with its name when a view was saved, to name a value that no longer exists
// (AllWorkFilterValue, FR-031).
type FilterValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Filters are the filter choices of a saved view (AllWorkFilters, data model §1.3). An empty list means no
// filter.
type Filters struct {
	Plans             []FilterValue `json:"plans"`
	PlanStatuses      []string      `json:"planStatuses"`
	ContainerStatuses []string      `json:"containerStatuses"`
	ActivityStatuses  []string      `json:"activityStatuses"`
	Owners            []FilterValue `json:"owners"`
	Teams             []FilterValue `json:"teams"`
	BusinessUnits     []FilterValue `json:"businessUnits"`
	ActivityTypes     []FilterValue `json:"activityTypes"`
	Strategic         string        `json:"strategic"`
	ShowLevels        string        `json:"showLevels"`
	PlanTypes         []string      `json:"planTypes"`      // 016 FR-043
	PlanCurrencies    []string      `json:"planCurrencies"` // 016 FR-043
	Attributes        []AttrFilter  `json:"attributes"`     // 016 FR-043
}

// Expansion says which rows are open (AllWorkExpansion, data model §4).
type Expansion struct {
	Mode    string   `json:"mode"`    // default, allExpanded or allCollapsed
	Toggled []string `json:"toggled"` // row keys switched away from the mode
}

// ViewState is the whole state of the page that a saved view keeps (AllWorkViewState, FR-029).
type ViewState struct {
	View           string    `json:"view"`
	Year           int       `json:"year"`
	Filters        Filters   `json:"filters"`
	Search         string    `json:"search"`
	Columns        []string  `json:"columns"`
	Sort           *Sort     `json:"sort"`
	Expansion      Expansion `json:"expansion"`
	ShowUnassigned bool      `json:"showUnassigned"`
	Period         string    `json:"period"`
	CalendarLayout string    `json:"calendarLayout"`
	// 016 FR-043: the range (a view saved before 016 has only year: read as year–year), Compare with phased
	// budget and the Breakdown's groupings.
	YearFrom      int              `json:"yearFrom"`
	YearTo        int              `json:"yearTo"`
	ComparePhased bool             `json:"comparePhased"`
	Breakdown     *BreakdownChoice `json:"breakdown"`
}

// BreakdownChoice is the groupings of a saved Breakdown view (contract AllWorkBreakdownChoice).
type BreakdownChoice struct {
	By     string  `json:"by"`
	ThenBy *string `json:"thenBy"`
}

// SavedView is a saved view as the API returns it (SavedView): only its owner ever sees it (FR-030).
type SavedView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"isDefault"`
	State     ViewState `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SavedViewPage is the answer of GET /api/v1/saved-views (SavedViewPage).
type SavedViewPage struct {
	Items []SavedView `json:"items"`
	Total int         `json:"total"`
}

// SavedViewCreate is the body of POST /api/v1/saved-views (SavedViewCreate).
type SavedViewCreate struct {
	Name      string     `json:"name"`
	State     *ViewState `json:"state"`
	IsDefault bool       `json:"isDefault"`
}

// SavedViewUpdate is the body of PATCH /api/v1/saved-views/{viewId} (SavedViewUpdate): only the parts to
// change.
type SavedViewUpdate struct {
	Name      *string    `json:"name"`
	State     *ViewState `json:"state"`
	IsDefault *bool      `json:"isDefault"`
}

// StoredView is a saved view as stored in the table "saved_views" (data model §1.1). It is a personal
// setting, not a business record (D-121): no version, no activity-log entry, no recycle bin.
type StoredView struct {
	ID        string    `json:"id"`       // "SV-1001"
	PersonID  string    `json:"personId"` // the owner; never changes
	Name      string    `json:"name"`     // 1–60 characters, unique among the owner's views ignoring case
	IsDefault bool      `json:"isDefault"`
	State     ViewState `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// view returns the stored view as the API returns it.
func (v StoredView) view() SavedView {
	return SavedView{ID: v.ID, Name: v.Name, IsDefault: v.IsDefault, State: v.State, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

// PreferencesRecord is a person's All work preferences as stored in the table "all_work_preferences"
// (data model §1.4), keyed by person ID. Not a business record (D-121).
type PreferencesRecord struct {
	PersonID  string    `json:"personId"`
	Columns   []string  `json:"columns"`
	Sort      *Sort     `json:"sort"`
	UpdatedAt time.Time `json:"updatedAt"`
}
