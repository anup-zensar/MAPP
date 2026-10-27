package approval

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

const (
	endDateField = "endDate"
	statusLabel  = "Status"
)

// parseDay reads a date "YYYY-MM-DD"; false when it is empty or not a date.
func parseDay(text string) (date.Date, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return date.Date{}, false
	}
	d, err := date.Parse(text)
	return d, err == nil
}

// delegationName is a delegation's name in the log: "<person> → <delegate>" (data-model §7).
func delegationName(dl Delegation, d directory) string {
	return d.person(dl.PersonID).Name + " → " + d.person(dl.DelegateID).Name
}

// delegationLog is a delegation as the activity log compares it: what people read, without the bookkeeping.
type delegationLog struct {
	DelegateID  string     `json:"delegateId"`
	StartDate   date.Date  `json:"startDate"`
	EndDate     date.Date  `json:"endDate"`
	Note        string     `json:"note"`
	CancelledAt *time.Time `json:"cancelledAt"`
}

// logView returns dl as the activity log compares it.
func (dl Delegation) logView() delegationLog {
	return delegationLog{DelegateID: dl.DelegateID, StartDate: dl.StartDate, EndDate: dl.EndDate, Note: dl.Note, CancelledAt: dl.CancelledAt}
}

// delegationKindOf is the history kind of approval delegations: they are logged (017 FR-035) but never moved
// to the recycle bin, so Remove is never called.
func (s *Service) delegationKindOf() history.Kind {
	return history.Kind{
		Name: delegationKind, Singular: delegationSingular, Plural: "approval delegations",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			dl, found, err := s.repo.GetDelegation(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			d, err := s.directory(ctx)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(dl)
			if err != nil {
				return history.Snapshot{}, false, err
			}
			return history.Snapshot{ID: dl.ID, Name: delegationName(dl, d), Data: data}, true, nil
		},
		Remove: func(context.Context, string) error { return errs.NotAllowed(delegationEndedMessage) },
		Put: func(ctx context.Context, snapshot history.Snapshot) error {
			var dl Delegation
			if err := json.Unmarshal(snapshot.Data, &dl); err != nil {
				return err
			}
			return s.repo.PutDelegation(ctx, dl)
		},
		Describe: s.describeDelegation,
	}
}

// describeDelegation gives the labels and texts of a delegation's changed fields in the activity log (data
// model §7): the delegate's name, the dates as "01 Nov 2026", the note and the status.
func (s *Service) describeDelegation(ctx context.Context, field string, value any) (string, string, error) {
	switch field {
	case "delegateId":
		id, _ := value.(string)
		d, err := s.directory(ctx)
		if err != nil {
			return "", "", err
		}
		return "Delegate", d.person(id).Name, nil
	case "startDate", endDateField:
		label := "From"
		if field == endDateField {
			label = "To"
		}
		text, _ := value.(string)
		if day, ok := parseDay(text); ok {
			text = date.Format(day)
		}
		return label, text, nil
	case "note":
		return "Note", "", nil
	case "cancelledAt":
		if value == nil {
			return statusLabel, "", nil
		}
		return statusLabel, cancelledStatusText, nil
	}
	return "", "", nil
}
