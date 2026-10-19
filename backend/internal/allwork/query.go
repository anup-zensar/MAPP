package allwork

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"northstar/internal/platform/errs"
)

// Messages of the plan's validation table (API) for the query of a view.
const (
	msgChooseYear    = "Choose a year from 2000 to 2100."
	msgChooseView    = "Choose grid, calendar or budget."
	msgChooseListed  = "Choose one of the listed values."
	msgChooseSort    = "Choose a column of the grid to sort by."
	msgSearchTooLong = "Use 200 characters or fewer."
	msgTooManyCols   = "Choose at most 100 columns."
)

// The limits of a query (data model §1.2).
const (
	firstYear       = 2000
	lastYear        = 2100
	maxSearchLength = 200
	maxColumns      = 100
)

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
)

// The choices of the filters with fixed values (FR-004).
var (
	planStatusValues      = []string{"draft", "published"}
	containerStatusValues = []string{"noActivities", "inProgress", "allApproved", "notStarted", "running", "finished"}
	activityStatusValues  = []string{"created", "submitted", "declined", "approved", "inProgress", "complete"}
	strategicValues       = []string{StrategicInclude, StrategicOnly, StrategicExclude}
	showLevelsValues      = []string{ShowEverything, ShowCompleteBranches, ShowCampaignsWithoutInitiatives, ShowInitiativesWithoutActivities}
	periodValues          = []string{PeriodMonth, PeriodQuarter}
	viewValues            = []string{ViewGrid, ViewCalendar, ViewBudget}
)

// Query is one view's filter set and display choices, as GET /api/all-work receives them (FR-004). Only the
// filters of FR-004 exist (FR-010, D-207): any other parameter is ignored.
type Query struct {
	Year              int    // 0: the current year
	View              string // grid, calendar or budget
	Plans             []string
	PlanStatuses      []string
	ContainerStatuses []string
	ActivityStatuses  []string
	Owners            []string
	Teams             []string
	BusinessUnits     []string
	ActivityTypes     []string
	Strategic         string // include, only or exclude
	ShowLevels        string
	Search            string   // trimmed
	Columns           []string // in display order; nil when the query names none: the default columns
	Sort              *Sort    // nil: the plan-tree order
	Period            string   // month or quarter
}

// ParseQuery reads the query of a view from the parameters of the request (FR-004), reporting every wrong
// parameter at once with the messages of the plan. Repeated parameters give several values, in order, each
// once; empty values are ignored. A sort names a column and a direction; whether the column exists is
// decided with the catalogue (an unknown one is left out, FR-031).
func ParseQuery(values url.Values) (Query, error) {
	var problems errs.Problems
	q := parseQuery(values, &problems)
	if err := problems.Err(); err != nil {
		return Query{}, err
	}
	return q, nil
}

// The formats of an export (FR-032).
const (
	FormatXLSX = "xlsx"
	FormatCSV  = "csv"
)

// ParseExport reads the query of GET /api/all-work/export: the query of the view, the format (xlsx or csv,
// required) and whether the unassigned planned cost rows are included (the screen's switch, default true).
// Every wrong parameter is reported at once.
func ParseExport(values url.Values) (q Query, format string, unassigned bool, err error) {
	var problems errs.Problems
	q = parseQuery(values, &problems)
	format = values.Get("format")
	if format != FormatXLSX && format != FormatCSV {
		problems.Add("format", msgChooseListed)
	}
	switch values.Get("unassigned") {
	case "", "true":
		unassigned = true
	case "false":
	default:
		problems.Add("unassigned", msgChooseListed)
	}
	if err := problems.Err(); err != nil {
		return Query{}, "", false, err
	}
	return q, format, unassigned, nil
}

// parseQuery reads the query of a view, adding its problems.
func parseQuery(values url.Values, problems *errs.Problems) Query {
	q := Query{
		View:       ViewGrid,
		Strategic:  StrategicInclude,
		ShowLevels: ShowEverything,
		Period:     PeriodMonth,
	}
	if text := values.Get("year"); text != "" {
		year, err := strconv.Atoi(text)
		if err != nil || year < firstYear || year > lastYear {
			problems.Add("year", msgChooseYear)
		} else {
			q.Year = year
		}
	}
	q.View = single(values, "view", viewValues, ViewGrid, msgChooseView, problems)
	q.Plans = list(values, "plan")
	q.PlanStatuses = listOf(values, "planStatus", planStatusValues, problems)
	q.ContainerStatuses = listOf(values, "containerStatus", containerStatusValues, problems)
	q.ActivityStatuses = listOf(values, "activityStatus", activityStatusValues, problems)
	q.Owners = list(values, "owner")
	q.Teams = list(values, "team")
	q.BusinessUnits = list(values, "businessUnit")
	q.ActivityTypes = list(values, "activityType")
	q.Strategic = single(values, "strategic", strategicValues, StrategicInclude, msgChooseListed, problems)
	q.ShowLevels = single(values, "showLevels", showLevelsValues, ShowEverything, msgChooseListed, problems)
	q.Period = single(values, "period", periodValues, PeriodMonth, msgChooseListed, problems)
	q.Search = strings.TrimSpace(values.Get("search"))
	if utf8.RuneCountInString(q.Search) > maxSearchLength {
		problems.Add("search", msgSearchTooLong)
	}
	if _, given := values["column"]; given {
		q.Columns = slices.DeleteFunc(list(values, "column"), func(key string) bool { return key == nameColumn })
		if len(q.Columns) > maxColumns {
			problems.Add("column", msgTooManyCols)
		}
	}
	if text := values.Get("sort"); text != "" {
		column, direction, found := strings.Cut(text, ":")
		if !found || column == "" || (direction != SortAscending && direction != SortDescending) {
			problems.Add("sort", msgChooseSort)
		} else {
			q.Sort = &Sort{Column: column, Direction: direction}
		}
	}
	return q
}

// nameColumn is the Name column: always shown, never a column key (FR-019).
const nameColumn = "name"

// list returns the non-empty values of the parameter name, in order, each once; never nil.
func list(values url.Values, name string) []string {
	kept := []string{}
	for _, v := range values[name] {
		if v = strings.TrimSpace(v); v != "" && !slices.Contains(kept, v) {
			kept = append(kept, v)
		}
	}
	return kept
}

// listOf is list for a parameter with fixed values: one problem when any value is not one of allowed.
func listOf(values url.Values, name string, allowed []string, problems *errs.Problems) []string {
	kept := list(values, name)
	for _, v := range kept {
		if !slices.Contains(allowed, v) {
			problems.Add(name, msgChooseListed)
			return []string{}
		}
	}
	return kept
}

// single returns the value of a single-choice parameter, or fallback when it is not given; a value that is
// not one of allowed adds message.
func single(values url.Values, name string, allowed []string, fallback, message string, problems *errs.Problems) string {
	v := values.Get(name)
	if v == "" {
		return fallback
	}
	if !slices.Contains(allowed, v) {
		problems.Add(name, message)
		return fallback
	}
	return v
}
