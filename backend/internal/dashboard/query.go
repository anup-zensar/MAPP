package dashboard

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"

	"mapp/internal/platform/errs"
)

const (
	minYear       = 2000
	maxYear       = 2100
	maxIDLength   = 40
	FormatPPTX    = "pptx"
	FormatPDF     = "pdf"
	msgYear       = "Choose a year from 2000 to 2100."
	msgPlanType   = "Choose Operational, Strategic or Cross-market."
	msgCurrency   = "Choose a currency of Settings."
	msgTeam       = "Choose a market."
	msgPlan       = "Choose a plan."
	msgFormat     = "Choose PowerPoint (pptx) or PDF (pdf)."
	fieldYear     = "year"
	fieldCurrency = "currency"
	fieldTeam     = "teamId"
)

var (
	currencyCode = regexp.MustCompile(`^[A-Z]{3}$`)
	planTypes    = []string{typeOperational, typeStrategic, typeCrossMarket} //nolint:gochecknoglobals // allowlist
)

// Query is the filters as asked; zero values take their defaults.
type Query struct {
	Year     int
	PlanType string
	Currency string
	TeamID   string
	PlanID   string
}

// ParseQuery reads the filters of a dashboard or an export, every wrong one reported at once.
func ParseQuery(v url.Values) (Query, error) {
	q := Query{PlanType: v.Get("planType"), Currency: v.Get(fieldCurrency), TeamID: v.Get(fieldTeam), PlanID: v.Get("planId")}
	year, yearOK := parseYear(v.Get(fieldYear))
	q.Year = year
	checks := []struct {
		ok             bool
		field, message string
	}{
		{yearOK, fieldYear, msgYear},
		{q.PlanType == "" || slices.Contains(planTypes, q.PlanType), "planType", msgPlanType},
		{q.Currency == "" || currencyCode.MatchString(q.Currency), fieldCurrency, msgCurrency},
		{len(q.TeamID) <= maxIDLength, fieldTeam, msgTeam},
		{len(q.PlanID) <= maxIDLength, "planId", msgPlan},
	}
	var problems []errs.FieldError
	for _, c := range checks {
		if !c.ok {
			problems = append(problems, errs.FieldError{Field: c.field, Message: c.message})
		}
	}
	if len(problems) > 0 {
		return Query{}, errs.Validation(problems...)
	}
	return q, nil
}

func parseYear(text string) (int, bool) {
	if text == "" {
		return 0, true
	}
	year, err := strconv.Atoi(text)
	return year, err == nil && year >= minYear && year <= maxYear
}

func ParseFormat(v url.Values) (string, error) {
	switch format := v.Get("format"); format {
	case FormatPPTX, FormatPDF:
		return format, nil
	}
	return "", errs.Validation(errs.FieldError{Field: "format", Message: msgFormat})
}
