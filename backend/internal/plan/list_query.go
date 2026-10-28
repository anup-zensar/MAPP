package plan

import (
	"net/url"
	"strconv"

	"mapp/internal/platform/errs"
)

const (
	scopeField  = "scope"
	statusField = "status"
)

// parseListFilter reports every wrong parameter at once, in the order of the query: scope and status, then
// the years, then the range and the plan types.
func parseListFilter(query url.Values) (ListFilter, error) {
	filter := ListFilter{Scope: query.Get(scopeField), Status: query.Get(statusField), Q: query.Get("q"), Types: query["type"],
		BCPIR: query.Get("bcpir")}
	var yearProblems errs.Problems
	filter.Year = parseListYear(query, "year", &yearProblems)
	filter.YearFrom = parseListYear(query, "yearFrom", &yearProblems)
	filter.YearTo = parseListYear(query, "yearTo", &yearProblems)

	var problems errs.Problems
	others := filterProblems(filter)
	for _, p := range others {
		if isScopeOrStatus(p.Field) {
			problems.Add(p.Field, p.Message)
		}
	}
	problems = append(problems, yearProblems...)
	for _, p := range others {
		if !isScopeOrStatus(p.Field) {
			problems.Add(p.Field, p.Message)
		}
	}
	return filter, problems.Err()
}

// parseListYear adds a problem to problems when the parameter is not a whole number.
func parseListYear(query url.Values, name string, problems *errs.Problems) *int {
	text := query.Get(name)
	if text == "" {
		return nil
	}
	year, err := strconv.Atoi(text)
	if err != nil {
		problems.Add(name, msgChooseListYear)
		return nil
	}
	return &year
}

func isScopeOrStatus(field string) bool {
	return field == scopeField || field == statusField
}
