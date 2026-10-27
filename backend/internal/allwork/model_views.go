package allwork

import "time"

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
