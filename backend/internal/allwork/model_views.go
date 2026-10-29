package allwork

import "time"

type Preferences struct {
	Columns []string `json:"columns"`
	Sort    *Sort    `json:"sort"`
}

type FilterValue struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

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
	BCPIR             string        `json:"bcpir,omitempty"`
	ShowLevels        string        `json:"showLevels"`
	PlanTypes         []string      `json:"planTypes"`
	PlanCurrencies    []string      `json:"planCurrencies"`
	Attributes        []AttrFilter  `json:"attributes"`
}

type Expansion struct {
	Mode    string   `json:"mode"`
	Toggled []string `json:"toggled"`
}

type ViewState struct {
	View           string           `json:"view"`
	Year           int              `json:"year"`
	Filters        Filters          `json:"filters"`
	Search         string           `json:"search"`
	Columns        []string         `json:"columns"`
	Sort           *Sort            `json:"sort"`
	Expansion      Expansion        `json:"expansion"`
	ShowUnassigned bool             `json:"showUnassigned"`
	Period         string           `json:"period"`
	CalendarLayout string           `json:"calendarLayout"`
	YearFrom       int              `json:"yearFrom"`
	YearTo         int              `json:"yearTo"`
	ComparePhased  bool             `json:"comparePhased"`
	Breakdown      *BreakdownChoice `json:"breakdown"`
}

type BreakdownChoice struct {
	By     string  `json:"by"`
	ThenBy *string `json:"thenBy"`
}

type SavedView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"isDefault"`
	State     ViewState `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SavedViewPage struct {
	Items []SavedView `json:"items"`
	Total int         `json:"total"`
}

type SavedViewCreate struct {
	Name      string     `json:"name"`
	State     *ViewState `json:"state"`
	IsDefault bool       `json:"isDefault"`
}

type SavedViewUpdate struct {
	Name      *string    `json:"name"`
	State     *ViewState `json:"state"`
	IsDefault *bool      `json:"isDefault"`
}

type StoredView struct {
	ID        string    `json:"id"`
	PersonID  string    `json:"personId"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"isDefault"`
	State     ViewState `json:"state"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (v StoredView) view() SavedView {
	return SavedView{ID: v.ID, Name: v.Name, IsDefault: v.IsDefault, State: v.State, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type PreferencesRecord struct {
	PersonID  string    `json:"personId"`
	Columns   []string  `json:"columns"`
	Sort      *Sort     `json:"sort"`
	UpdatedAt time.Time `json:"updatedAt"`
}
