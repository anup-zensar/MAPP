package activities

import (
	"mapp/internal/platform/date"
)

// displayStatus is the status shown for a on the day today (FR-021, D-091): an approved activity shows
// Complete after its end date (never when ongoing) and In progress from its start date; every other status
// shows as stored.
func displayStatus(a Activity, today date.Date) DisplayStatus {
	if a.Status != StatusApproved {
		return DisplayStatus(a.Status)
	}
	switch {
	case !a.Ongoing && a.EndDate != nil && today.After(*a.EndDate):
		return DisplayComplete
	case a.StartDate != nil && !today.Before(*a.StartDate):
		return DisplayInProgress
	}
	return DisplayApproved
}

// statusWords are the stored statuses as people read them (activity log, design system §4).
var statusWords = map[Status]string{
	StatusCreated: "Created", StatusSubmitted: "Submitted", StatusDeclined: "Declined", StatusApproved: "Approved",
}
