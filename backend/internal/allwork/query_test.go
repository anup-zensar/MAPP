package allwork

import (
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"mapp/internal/platform/errs"
)

// Every parameter of the query is read (FR-004): single values, repeated values in order and once each,
// the column order, and the sort split into column and direction; the search is trimmed.
func TestParseQuery_AllParameters_FR004(t *testing.T) {
	values := url.Values{
		"year": {"2027"}, "view": {"budget"}, "plan": {"P-1001", "P-1002", "P-1001"}, "planStatus": {"published"},
		"containerStatus": {"inProgress", "running"}, "activityStatus": {"declined", "approved"},
		"owner": {"U-1003"}, "team": {"T-1001"}, "businessUnit": {"SO-1002"}, "activityType": {"AT-1005"},
		"strategic": {"only"}, "showLevels": {"completeBranches"}, "search": {"  Launch film "},
		"column": {"plannedCost", "id", "attr.SA-1003", "id"}, "sort": {"plannedCost:desc"}, "period": {"quarter"},
		"finance": {"invoiced", "hasPurchaseOrder", "invoiced"},
	}

	q, err := ParseQuery(values)

	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	want := Query{
		Year: 2027, View: ViewBudget, Plans: []string{"P-1001", "P-1002"}, PlanStatuses: []string{"published"},
		ContainerStatuses: []string{"inProgress", "running"}, ActivityStatuses: []string{"declined", "approved"},
		Owners: []string{"U-1003"}, Teams: []string{"T-1001"}, BusinessUnits: []string{"SO-1002"},
		ActivityTypes: []string{"AT-1005"}, Finance: []string{"invoiced", "hasPurchaseOrder"}, Strategic: StrategicOnly,
		ShowLevels: ShowCompleteBranches,
		Search:     "Launch film", Columns: []string{"plannedCost", "id", "attr.SA-1003"},
		Sort: &Sort{Column: "plannedCost", Direction: SortDescending}, Period: PeriodQuarter,
		// 016: year is the alias of the range year–year; the new filters are empty, amounts in the display currency.
		YearFrom: 2027, YearTo: 2027, PlanTypes: []string{}, PlanCurrencies: []string{}, Attributes: []AttrFilter{},
		AmountsIn: AmountsInDisplay,
	}
	if !reflect.DeepEqual(q, want) {
		t.Errorf("query = %+v\nwant    %+v", q, want)
	}
}

// Without parameters the query is the standard view: the current year, the grid, no filters, Include, Everything,
// by month, the default columns (Columns nil) and the plan-tree order (FR-004, FR-009).
func TestParseQuery_Defaults_FR004(t *testing.T) {
	q, err := ParseQuery(url.Values{})

	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if q.Year != 0 || q.View != ViewGrid || q.Strategic != StrategicInclude || q.ShowLevels != ShowEverything ||
		q.Period != PeriodMonth || q.Columns != nil || q.Sort != nil || len(q.Plans) != 0 || q.Search != "" {
		t.Errorf("query = %+v, want the standard view", q)
	}
	// A column parameter without columns asks for none: only Name.
	none, _ := ParseQuery(url.Values{"column": {""}})
	if none.Columns == nil || len(none.Columns) != 0 {
		t.Errorf("columns = %#v, want an empty list", none.Columns)
	}
}

// Every wrong parameter is reported at once, each with the plan's message (FR-004).
func TestParseQuery_AllProblemsAtOnce_FR004(t *testing.T) {
	values := url.Values{
		"year": {"1999"}, "view": {"table"}, "planStatus": {"archived"}, "containerStatus": {"late"},
		"activityStatus": {"lost"}, "strategic": {"maybe"}, "showLevels": {"some"}, "period": {"week"},
		"search": {strings.Repeat("x", 201)}, "sort": {"plannedCost:up"}, "column": manyColumns(101),
	}

	_, err := ParseQuery(values)

	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeValidationFailed {
		t.Fatalf("err = %v, want validation_failed", err)
	}
	want := []errs.FieldError{
		{Field: "year", Message: "Choose a year from 2000 to 2100."},
		{Field: "view", Message: "Choose grid, calendar or budget."},
		{Field: "planStatus", Message: "Choose one of the listed values."},
		{Field: "containerStatus", Message: "Choose one of the listed values."},
		{Field: "activityStatus", Message: "Choose one of the listed values."},
		{Field: "strategic", Message: "Choose one of the listed values."},
		{Field: "showLevels", Message: "Choose one of the listed values."},
		{Field: "period", Message: "Choose one of the listed values."},
		{Field: "search", Message: "Use 200 characters or fewer."},
		{Field: "column", Message: "Choose at most 100 columns."},
		{Field: "sort", Message: "Choose a column of the grid to sort by."},
	}
	if !reflect.DeepEqual(apiErr.Fields, want) {
		t.Errorf("problems = %+v\nwant       %+v", apiErr.Fields, want)
	}
	// A year that is not a number, and a sort without a direction, are refused too.
	_, err = ParseQuery(url.Values{"year": {"next"}, "sort": {"plannedCost"}})
	if !errors.As(err, &apiErr) || len(apiErr.Fields) != 2 {
		t.Errorf("err = %v, want problems on year and sort", err)
	}
}

// The filter bar has only the filters of FR-004 (FR-010, D-207): a Shell-attribute filter is an unknown
// parameter, which changes nothing.
func TestFilter_OnlyFR004Filters_FR010(t *testing.T) {
	plain, _ := ParseQuery(url.Values{"owner": {"U-1003"}})
	withAttribute, err := ParseQuery(url.Values{"owner": {"U-1003"}, "attr.SA-1003": {"SO-1010"}, "cluster": {"SO-1010"}})

	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if !reflect.DeepEqual(plain, withAttribute) {
		t.Errorf("an attribute filter changed the query: %+v, want %+v", withAttribute, plain)
	}
	b := seedYear()
	if got, want := rowKeys(view(b, withAttribute).Rows), rowKeys(view(b, plain).Rows); !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q, want %q", got, want)
	}
}

// manyColumns returns n column keys.
func manyColumns(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "attr.SA-" + strconv.Itoa(1001+i)
	}
	return keys
}
