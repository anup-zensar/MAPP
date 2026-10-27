package allwork

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// Messages of the plan's validation table (API) for the query of a view.
const (
	msgChooseYear    = "Choose a year from 2000 to 2100."
	msgChooseView    = "Choose grid, calendar or budget."
	msgChooseListed  = "Choose one of the listed values."
	msgChooseSort    = "Choose a column of the grid to sort by."
	msgSearchTooLong = "Use 200 characters or fewer."
	msgTooManyCols   = "Choose at most 100 columns."
	msgYearRange     = "Choose at most 5 years."                        // 016 FR-034
	msgYearOrder     = "Choose a last year on or after the first year." // 016 FR-034
	msgChooseAttr    = "Choose a Shell attribute and its values, such as SA-1022:SO-1301,none."
	msgChooseCcy     = "Choose currencies by their codes, such as GBP."
)

// The limits of a query (data model §1.2).
const (
	firstYear       = 2000
	lastYear        = 2100
	maxSearchLength = 200
	maxColumns      = 100
	maxRangeYears   = 5 // 016 FR-034: a range of at most 5 years
)

// The choices of the filters with fixed values (FR-004).
var (
	planStatusValues      = []string{"draft", "published"}
	containerStatusValues = []string{"noActivities", "inProgress", "allApproved", "notStarted", "running", "finished"}
	activityStatusValues  = []string{"created", "submitted", "declined", "approved", "inProgress", "complete"}
	strategicValues       = []string{StrategicInclude, StrategicOnly, StrategicExclude}
	showLevelsValues      = []string{ShowEverything, ShowCompleteBranches, ShowCampaignsWithoutInitiatives, ShowInitiativesWithoutActivities}
	periodValues          = []string{PeriodMonth, PeriodQuarter, PeriodYear}
	viewValues            = []string{ViewGrid, ViewCalendar, ViewBudget, ViewBreakdown}
	planTypeValues        = []string{"operational", "strategic", "crossMarket"} // 016 FR-032
	amountsInValues       = []string{AmountsInDisplay, AmountsInPlan}
)

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

// ParseExport reads the query of GET /api/v1/all-work/export: the query of the view, the format (xlsx or csv,
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
		AmountsIn:  AmountsInDisplay,
	}
	q.YearFrom, q.YearTo = parseRange(values, problems)
	q.Year = q.YearFrom
	q.View = single(values, "view", viewValues, ViewGrid, msgChooseView, problems)
	q.Plans = list(values, "plan")
	q.PlanStatuses = listOf(values, "planStatus", planStatusValues, problems)
	q.ContainerStatuses = listOf(values, "containerStatus", containerStatusValues, problems)
	q.ActivityStatuses = listOf(values, "activityStatus", activityStatusValues, problems)
	q.Owners = list(values, "owner")
	q.Teams = list(values, "team")
	q.BusinessUnits = list(values, "businessUnit")
	q.ActivityTypes = list(values, "activityType")
	q.Finance = listOf(values, "finance", financeValues, problems)
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
	q.PlanTypes = listOf(values, "planType", planTypeValues, problems)
	q.PlanCurrencies = list(values, "planCurrency")
	for _, code := range q.PlanCurrencies {
		if !isCurrencyCode(code) {
			problems.Add("planCurrency", msgChooseCcy)
			q.PlanCurrencies = []string{}
			break
		}
	}
	q.Attributes = parseAttrs(values, problems)
	switch values.Get("comparePhased") {
	case "", "false":
	case "true":
		q.ComparePhased = true
	default:
		problems.Add("comparePhased", msgChooseListed)
	}
	q.By, q.ThenBy = strings.TrimSpace(values.Get("by")), strings.TrimSpace(values.Get("thenBy"))
	q.AmountsIn = single(values, "amountsIn", amountsInValues, AmountsInDisplay, msgChooseListed, problems)
	return q
}

// parseRange reads the years of a view (016 FR-034, D-269): yearFrom and yearTo, or year as the alias of a
// one-year range; yearTo is yearFrom when not sent, and at most yearFrom + 4. 0, 0 means the current year.
func parseRange(values url.Values, problems *errs.Problems) (from, to int) {
	from, to = yearParam(values, "yearFrom", problems), yearParam(values, "yearTo", problems)
	if year := yearParam(values, "year", problems); year != 0 && from == 0 && to == 0 {
		from, to = year, year
	}
	if from == 0 {
		from = to
	}
	if to == 0 {
		to = from
	}
	switch {
	case to < from:
		problems.Add("yearTo", msgYearOrder)
		to = from
	case to-from >= maxRangeYears:
		problems.Add("yearTo", msgYearRange)
		to = from
	}
	return from, to
}

// yearParam reads the year parameter name: 0 when not sent or wrong (a wrong one adds its problem).
func yearParam(values url.Values, name string, problems *errs.Problems) int {
	text := values.Get(name)
	if text == "" {
		return 0
	}
	year, err := strconv.Atoi(text)
	if err != nil || year < firstYear || year > lastYear {
		problems.Add(name, msgChooseYear)
		return 0
	}
	return year
}

// parseAttrs reads the Shell-attribute filters "<attributeId>:<value>,<value>" (016 FR-039): each attribute
// once, its values in order, each once. A filter that is not of that form adds one problem.
func parseAttrs(values url.Values, problems *errs.Problems) []AttrFilter {
	filters := []AttrFilter{}
	for _, text := range list(values, "attr") {
		id, rest, found := strings.Cut(text, ":")
		id = strings.TrimSpace(id)
		var chosen []string
		for _, v := range strings.Split(rest, ",") {
			if v = strings.TrimSpace(v); v != "" && !slices.Contains(chosen, v) {
				chosen = append(chosen, v)
			}
		}
		if !found || id == "" || len(chosen) == 0 {
			problems.Add("attr", msgChooseAttr)
			return []AttrFilter{}
		}
		if i := slices.IndexFunc(filters, func(f AttrFilter) bool { return f.AttributeID == id }); i >= 0 {
			for _, v := range chosen {
				if !slices.Contains(filters[i].Values, v) {
					filters[i].Values = append(filters[i].Values, v)
				}
			}
			continue
		}
		filters = append(filters, AttrFilter{AttributeID: id, Values: chosen})
	}
	return filters
}

// isCurrencyCode reports whether code is three capital letters.
func isCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
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
