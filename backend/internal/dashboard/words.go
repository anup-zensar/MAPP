package dashboard

import (
	"strconv"

	"mapp/internal/allwork"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

const (
	product     = "SP&A One"
	none        = "—"
	dayLayout   = "2 Jan 2006"
	financeNote = "You can't see financial data"
)

const (
	statusCreated        = "created"
	statusSubmitted      = "submitted"
	statusDeclined       = "declined"
	statusApproved       = "approved"
	statusInProgress     = "inProgress"
	statusComplete       = "complete"
	typeOperational      = "operational"
	typeStrategic        = "strategic"
	typeCrossMarket      = "crossMarket"
	wordApproved         = "Approved"
	wordDeclined         = "Declined"
	wordPlans            = "Plans"
	wordCampaigns        = "Campaigns"
	wordActivities       = "Activities"
	wordStatus           = "Status"
	wordAmount           = "Amount"
	wordPurchaseRequests = "Purchase requests"
	wordLeftToPlan       = "Left to plan"
)

var words = map[string]string{ //nolint:gochecknoglobals // the words people read for the contract's values
	"draft": "Draft", "published": "Published",
	typeOperational: "Operational", typeStrategic: "Strategic", typeCrossMarket: "Cross-market",
	statusCreated: "Created", statusSubmitted: "Submitted", statusDeclined: wordDeclined, statusApproved: wordApproved,
	statusInProgress: "In progress", statusComplete: "Complete", "inReview": "In review", "closed": "Closed",
	string(allwork.LevelPlan): wordPlans, string(allwork.LevelCampaign): wordCampaigns,
	string(allwork.LevelInitiative): "Initiatives", string(allwork.LevelActivity): wordActivities,
	string(allwork.LevelLineItem): "Line items",
}

func word(value string) string {
	if w, found := words[value]; found {
		return w
	}
	return value
}

func moneyText(m *money.Money) string {
	if m == nil {
		return none
	}
	return m.Format()
}

func percentText(p *int) string {
	if p == nil {
		return none
	}
	return strconv.Itoa(*p) + "%"
}

func dayText(d date.Date) string {
	return d.Time().Format(dayLayout)
}

func datesText(start, end *date.Date, ongoing bool) string {
	switch {
	case start == nil:
		return none
	case ongoing || end == nil:
		return "Ongoing from " + dayText(*start)
	}
	return dayText(*start) + " – " + dayText(*end)
}
