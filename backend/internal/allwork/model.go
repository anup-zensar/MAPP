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

type Level string

const (
	LevelPlan       Level = "plan"
	LevelCampaign   Level = "campaign"
	LevelInitiative Level = "initiative"
	LevelActivity   Level = "activity"
	LevelLineItem   Level = "lineItem"
	LevelUnassigned Level = "unassigned"
)

type Tone string

const (
	ToneNeutral  Tone = "neutral"
	ToneReview   Tone = "review"
	TonePositive Tone = "positive"
	ToneActive   Tone = "active"
	ToneDone     Tone = "done"
	ToneProblem  Tone = "problem"
)

const (
	ViewGrid     = "grid"
	ViewCalendar = "calendar"
	ViewBudget   = "budget"
)

const (
	RoleMatch   = "match"
	RoleContext = "context"
)

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

type Status struct {
	Value  string `json:"value"`
	Label  string `json:"label"`
	Tone   Tone   `json:"tone"`
	Detail string `json:"detail,omitempty"`
}

type Cell struct {
	Kind       string       `json:"kind"`
	Text       string       `json:"text,omitempty"`
	Money      *money.Money `json:"money,omitempty"`
	Number     *int64       `json:"number,omitempty"`
	Percent    string       `json:"percent,omitempty"`
	Date       date.Date    `json:"date,omitzero"`
	DateTime   *time.Time   `json:"dateTime,omitempty"`
	Tone       Tone         `json:"tone,omitempty"`
	Inactive   bool         `json:"inactive,omitempty"`
	Unfiltered *Cell        `json:"unfiltered,omitempty"`
}

type SearchHit struct {
	Fields  []string `json:"fields"`
	Excerpt string   `json:"excerpt,omitempty"`
}

type Owner struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Inactive bool   `json:"inactive"`
}

type Row struct {
	Key          string         `json:"key"`
	ParentKey    *string        `json:"parentKey"`
	Level        Level          `json:"level"`
	RecordID     string         `json:"recordId"`
	PlanID       string         `json:"planId"`
	ActivityID   string         `json:"activityId,omitempty"`
	Depth        int            `json:"depth"`
	Name         string         `json:"name"`
	Role         string         `json:"role"`
	SearchHit    *SearchHit     `json:"searchHit,omitempty"`
	Strategic    *bool          `json:"strategic,omitempty"`
	BCPIR        *bool          `json:"bcpir,omitempty"`
	Status       *Status        `json:"status"`
	Timing       *Status        `json:"timing,omitempty"`
	StartDate    date.Date      `json:"startDate,omitzero"`
	EndDate      date.Date      `json:"endDate,omitzero"`
	Ongoing      bool           `json:"ongoing"`
	Owner        *Owner         `json:"owner,omitempty"`
	PlannedCost  *money.Money   `json:"plannedCost,omitempty"`
	Balance      *money.Money   `json:"balance,omitempty"`
	Budget       *money.Money   `json:"budget,omitempty"`
	HasChildren  bool           `json:"hasChildren"`
	Filtered     bool           `json:"filtered,omitempty"`
	Cells        []*Cell        `json:"cells"`
	Periods      []*money.Money `json:"periods,omitempty"`
	Total        *money.Money   `json:"total"`
	Year         *int           `json:"year,omitempty"`
	PlanType     string         `json:"planType,omitempty"`
	PhasedBudget []*money.Money `json:"phasedBudget,omitempty"`
	OverPeriods  []int          `json:"overPeriods,omitempty"`
}

type Column struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Group    string  `json:"group"`
	Kind     string  `json:"kind"`
	Levels   []Level `json:"levels"`
	Sortable bool    `json:"sortable"`
	Inactive bool    `json:"inactive"`
}

type ColumnGroup struct {
	Group   string   `json:"group"`
	Label   string   `json:"label"`
	Columns []Column `json:"columns"`
}

type Period struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type Summary struct {
	Campaigns             int          `json:"campaigns"`
	Initiatives           int          `json:"initiatives"`
	Activities            int          `json:"activities"`
	PlannedCost           *money.Money `json:"plannedCost"`
	PlanBudget            *money.Money `json:"planBudget"`
	UnassignedPlannedCost *money.Money `json:"unassignedPlannedCost"`
	NegativeBalanceCount  int          `json:"negativeBalanceCount"`
}

type Count struct {
	Shown int    `json:"shown"`
	Total int    `json:"total"`
	Noun  string `json:"noun"`
}

const (
	nounActivities = "activities"
	nounActivity   = "activity"
)

type Totals struct {
	Activities  int            `json:"activities"`
	Cells       []*Cell        `json:"cells"`
	PlannedCost *money.Money   `json:"plannedCost"`
	PlanBudget  *money.Money   `json:"planBudget"`
	Periods     []*money.Money `json:"periods,omitempty"`
	Total       *money.Money   `json:"total"`
}

type LeftOut struct {
	Filter string `json:"filter"`
	ID     string `json:"id"`
}

type Result struct {
	Year            int       `json:"year"`
	YearFrom        int       `json:"yearFrom"`
	YearTo          int       `json:"yearTo"`
	Today           date.Date `json:"today"`
	View            string    `json:"view"`
	DisplayCurrency string    `json:"displayCurrency"`
	GeneratedAt     time.Time `json:"generatedAt"`
	Columns         []Column  `json:"columns"`
	Periods         []Period  `json:"periods"`
	Rows            []Row     `json:"rows"`
	Summary         Summary   `json:"summary"`
	Count           Count     `json:"count"`
	Totals          Totals    `json:"totals"`
	LeftOut         []LeftOut `json:"leftOut"`
}

type Option struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Inactive bool   `json:"inactive"`
	Mine     bool   `json:"mine,omitempty"`
}

type PlanOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	Strategic bool   `json:"strategic"`
	Year      int    `json:"year"`
	PlanType  string `json:"planType"`
	Currency  string `json:"currency"`
}

type FilterableAttribute struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Level  Level    `json:"level"`
	Type   string   `json:"type"`
	Values []Option `json:"values"`
}

type Options struct {
	Year           int                   `json:"year"`
	CurrentYear    int                   `json:"currentYear"`
	Years          []int                 `json:"years"`
	Plans          []PlanOption          `json:"plans"`
	Owners         []Option              `json:"owners"`
	Teams          []Option              `json:"teams"`
	BusinessUnits  []Option              `json:"businessUnits"`
	ActivityTypes  []Option              `json:"activityTypes"`
	ColumnGroups   []ColumnGroup         `json:"columnGroups"`
	DefaultColumns []string              `json:"defaultColumns"`
	YearFrom       int                   `json:"yearFrom"`
	YearTo         int                   `json:"yearTo"`
	PlanTypes      []string              `json:"planTypes"`
	PlanCurrencies []string              `json:"planCurrencies"`
	Attributes     []FilterableAttribute `json:"attributes"`
}

type RecordLocation struct {
	RecordID   string `json:"recordId"`
	Level      Level  `json:"level"`
	Name       string `json:"name"`
	PlanID     string `json:"planId"`
	ActivityID string `json:"activityId,omitempty"`
}

type Sort struct {
	Column    string `json:"column"`
	Direction string `json:"direction"`
}

const (
	SortAscending  = "asc"
	SortDescending = "desc"
)
