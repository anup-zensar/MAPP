package financials

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
)

// Query is the filter set, sort and page of the Financials page (FR-021), as GET /api/financials/transactions
// receives it. The address of the page holds the same parameters, so a link reproduces the view.
type Query struct {
	Type     string // "all" or a Type: the tab
	Search   string // trimmed; "" for none
	Year     int    // 0: the record's plan year with RecordID, otherwise the current year
	PlanID   string
	RecordID string
	Link     string // all, linked or notLinked
	Sort     string // postingDate, documentNumber, type, vendor, amount or linkedTo
	Desc     bool
	Page     int
	PageSize int
}

// The values of the list's parameters (contract FinancialType, FinancialLinkFilter, FinancialSort,
// FinancialPageSize).
const (
	filterAll       = "all"
	filterLinked    = "linked"
	filterNotLinked = "notLinked"

	sortPostingDate    = "postingDate"
	sortDocumentNumber = "documentNumber"
	sortType           = "type"
	sortVendor         = "vendor"
	sortAmount         = "amount"
	sortLinkedTo       = "linkedTo"
)

var (
	typeValues     = []string{filterAll, string(TypePurchaseOrder), string(TypeInvoice), string(TypePayment), string(TypeAccrual)}
	linkValues     = []string{filterAll, filterLinked, filterNotLinked}
	pageSizeValues = []int{50, 100, 200}
	sortForm       = regexp.MustCompile(`^(postingDate|documentNumber|type|vendor|amount|linkedTo):(asc|desc)$`)
	planIDForm     = regexp.MustCompile(`^P-[0-9]+$`)
	recordIDForm   = regexp.MustCompile(`^[PCIAL]-[0-9]+$`)
)

// The limits of a query.
const (
	firstYear       = 2000
	lastYear        = 2100
	maxSearchLength = 200
)

// ParseQuery reads the list's parameters, reporting every wrong one at once with the parameter's name as
// the field (as D-219). Missing parameters take their defaults: tab All, every link state, newest first,
// page 1 of 50.
func ParseQuery(values url.Values) (Query, error) {
	var problems errs.Problems
	q := Query{Type: filterAll, Link: filterAll, Sort: sortPostingDate, Desc: true, Page: 1, PageSize: 50}
	if v := values.Get("type"); v != "" {
		if !slices.Contains(typeValues, v) {
			problems.Add("type", msgChooseListed)
		} else {
			q.Type = v
		}
	}
	q.Search = strings.TrimSpace(values.Get("q"))
	if utf8.RuneCountInString(q.Search) > maxSearchLength {
		problems.Add("q", msgSearchTooLong)
	}
	if v := values.Get("year"); v != "" {
		year, err := strconv.Atoi(v)
		if err != nil || year < firstYear || year > lastYear {
			problems.Add("year", msgChooseYear)
		} else {
			q.Year = year
		}
	}
	if q.PlanID = values.Get("planId"); q.PlanID != "" && !planIDForm.MatchString(q.PlanID) {
		problems.Add("planId", msgChoosePlan)
	}
	if q.RecordID = values.Get("recordId"); q.RecordID != "" && !recordIDForm.MatchString(q.RecordID) {
		problems.Add("recordId", msgChooseRecord)
	}
	if v := values.Get("link"); v != "" {
		if !slices.Contains(linkValues, v) {
			problems.Add("link", msgChooseListed)
		} else {
			q.Link = v
		}
	}
	if v := values.Get("sort"); v != "" {
		if !sortForm.MatchString(v) {
			problems.Add("sort", msgChooseSort)
		} else {
			column, direction, _ := strings.Cut(v, ":")
			q.Sort, q.Desc = column, direction == "desc"
		}
	}
	if v := values.Get("page"); v != "" {
		page, err := strconv.Atoi(v)
		if err != nil || page < 1 {
			problems.Add("page", msgChoosePage)
		} else {
			q.Page = page
		}
	}
	if v := values.Get("pageSize"); v != "" {
		size, err := strconv.Atoi(v)
		if err != nil || !slices.Contains(pageSizeValues, size) {
			problems.Add("pageSize", msgChoosePageSize)
		} else {
			q.PageSize = size
		}
	}
	if err := problems.Err(); err != nil {
		return Query{}, err
	}
	return q, nil
}

// ParseScope reads Home's scope: anything but myTeams or everything is validation_failed on scope (as
// 009's ParseScope).
func ParseScope(text string) (Scope, error) {
	switch scope := Scope(text); scope {
	case ScopeMyTeams, ScopeEverything:
		return scope, nil
	}
	return "", fieldProblem("scope", msgChooseScope)
}
