package allwork

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

const (
	msgChooseYear    = "Choose a year from 2000 to 2100."
	msgChooseView    = "Choose grid, calendar or budget."
	msgChooseListed  = "Choose one of the listed values."
	msgChooseSort    = "Choose a column of the grid to sort by."
	msgSearchTooLong = "Use 200 characters or fewer."
	msgTooManyCols   = "Choose at most 100 columns."
	msgYearRange     = "Choose at most 5 years."
	msgYearOrder     = "Choose a last year on or after the first year."
	msgChooseAttr    = "Choose a Shell attribute and its values, such as SA-1022:SO-1301,none."
	msgChooseCcy     = "Choose currencies by their codes, such as GBP."
)

const (
	firstYear       = 2000
	lastYear        = 2100
	maxSearchLength = 200
	maxColumns      = 100
	maxRangeYears   = 5
)

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

func ParseQuery(values url.Values) (Query, error) {
	var problems errs.Problems
	q := parseQuery(values, &problems)
	if err := problems.Err(); err != nil {
		return Query{}, err
	}
	return q, nil
}

const (
	FormatXLSX = "xlsx"
	FormatCSV  = "csv"
)

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

func parseQuery(values url.Values, problems *errs.Problems) Query {
	q := Query{
		View:       ViewGrid,
		Strategic:  StrategicInclude,
		BCPIR:      StrategicInclude,
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
	q.BCPIR = single(values, "bcpir", strategicValues, StrategicInclude, msgChooseListed, problems)
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

const nameColumn = "name"

func list(values url.Values, name string) []string {
	kept := []string{}
	for _, v := range values[name] {
		if v = strings.TrimSpace(v); v != "" && !slices.Contains(kept, v) {
			kept = append(kept, v)
		}
	}
	return kept
}

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
