package activity

import (
	"fmt"

	"mapp/internal/campaign"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// Messages of the plan's "User-visible messages" table about dates.
const (
	invalidDateMessage   = "Enter a valid date."
	endBeforeStartFormat = "End date must be on or after the start date (%s)."
	outsideFormat        = "Dates must be inside the initiative's dates (%s)."
	afterLatestFormat    = "The end date can't be after %s — 10 years after the plan's year."
)

// latestEndDate is the latest end date of an activity of a plan of year: 31 December of year + 10 (R5).
func latestEndDate(year int) date.Date {
	return date.YearEnd(year + 10)
}

// parseDate reads a date of a request: nil or "" is no date; anything else must be YYYY-MM-DD, otherwise
// the problem is added to problems under field.
func parseDate(problems *errs.Problems, field string, text *string) *date.Date {
	if text == nil || *text == "" {
		return nil
	}
	parsed, err := date.Parse(*text)
	if err != nil {
		problems.Add(field, invalidDateMessage)
		return nil
	}
	return &parsed
}

// checkDates adds the problems of an activity's dates to problems (FR-009, FR-010): the end on or after
// the start; both inside the initiative's dates when the initiative has both (R3) — for an ongoing activity
// only the start; the end no later than latestEnd (R5). end is ignored when ongoing.
func checkDates(problems *errs.Problems, start, end *date.Date, ongoing bool, initiative campaign.Initiative, latestEnd date.Date) {
	if ongoing {
		end = nil
	}
	if start != nil && end != nil && end.Before(*start) {
		problems.Add("endDate", fmt.Sprintf(endBeforeStartFormat, date.Format(*start)))
	}
	if initiative.StartDate != nil && initiative.EndDate != nil {
		message := fmt.Sprintf(outsideFormat, date.FormatRangeFull(*initiative.StartDate, *initiative.EndDate))
		if start != nil && !inside(*start, initiative) {
			problems.Add("startDate", message)
		}
		if end != nil && !inside(*end, initiative) {
			problems.Add("endDate", message)
		}
	}
	if end != nil && end.After(latestEnd) {
		problems.Add("endDate", fmt.Sprintf(afterLatestFormat, date.Format(latestEnd)))
	}
}

// datesFit reports whether the activity's dates lie inside the initiative's, when the initiative has both
// dates (R3); for an ongoing activity only the start counts.
func datesFit(a Activity, initiative campaign.Initiative) bool {
	if initiative.StartDate == nil || initiative.EndDate == nil {
		return true
	}
	if a.StartDate != nil && !inside(*a.StartDate, initiative) {
		return false
	}
	return a.Ongoing || a.EndDate == nil || inside(*a.EndDate, initiative)
}

// inside reports whether d lies within the initiative's dates, both included; the initiative has both.
func inside(d date.Date, initiative campaign.Initiative) bool {
	return !d.Before(*initiative.StartDate) && !d.After(*initiative.EndDate)
}
