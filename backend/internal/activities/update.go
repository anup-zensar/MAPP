package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// Update saves an edited activity (FR-026 – FR-028). update holds every editable field and the version the
// person started from: a missing version is reported with the other problems; an out-of-date one is
// refused with version_conflict. A save that changes nothing answers the activity unchanged. On a
// Submitted or Approved activity, a change of planned cost, start date, end date or ongoing needs
// RestartApproval (D-106) and returns it to Created; other edits keep the status.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, update Update) (View, error) {
	var view View
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, pc, err := s.load(ctx, id)
		if err != nil {
			return err
		}
		if update.Version != nil && *update.Version >= 1 {
			if err := before.CheckVersion(*update.Version, noun); err != nil {
				return err
			}
		}
		var problems errs.Problems
		after, err := s.checkFields(ctx, &problems, pc, update.fields(), &before)
		if err != nil {
			return err
		}
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", reloadMessage)
		}
		if err := problems.Err(); err != nil {
			return err
		}

		changes, err := diffActivity(before, after)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.viewOf(ctx, before, pc)
			return err
		}
		restartFields := reapprovalFields(before, changes)
		if len(restartFields) > 0 && !update.RestartApproval {
			return errs.Validation(errs.FieldError{Field: "restartApproval", Message: restartApprovalMessage})
		}
		if len(restartFields) > 0 {
			after.Status = StatusCreated
		}
		after.Touch(a, s.now())
		if err := s.repo.Put(ctx, after); err != nil {
			return err
		}
		entry, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: changes,
		})
		if err != nil {
			return err
		}
		if datesChanged(before, after) {
			if err := s.lineItems.DatesChanged(ctx, a, before, after); err != nil {
				return err
			}
		}
		if len(restartFields) > 0 {
			details := fmt.Sprintf(approvalRestartedFormat, joinWords(restartFields))
			if err := s.logRestart(ctx, a, before, after, entry.Changes, details); err != nil {
				return err
			}
		}
		view, err = s.viewOf(ctx, after, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// fields returns the editable fields of an update, as a create sends them.
func (u Update) fields() Input {
	return Input{
		Name: u.Name, ExternalName: u.ExternalName, ActivityTypeID: u.ActivityTypeID, OwnerID: u.OwnerID,
		StartDate: u.StartDate, EndDate: u.EndDate, Ongoing: u.Ongoing, PlannedCost: u.PlannedCost,
		ApprovalWorkflowID: u.ApprovalWorkflowID, Description: u.Description, Attributes: u.Attributes, Goals: u.Goals,
	}
}

// datesChanged reports whether the start date, end date or ongoing differ.
func datesChanged(before, after Activity) bool {
	return !sameDate(before.StartDate, after.StartDate) || !sameDate(before.EndDate, after.EndDate) ||
		before.Ongoing != after.Ongoing
}

// sameDate reports whether two optional dates are the same.
func sameDate(x, y *date.Date) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Compare(*y) == 0
}

// diffActivity returns the changed fields of an activity, with one change per Shell attribute
// ("attributes.<id>") instead of one for all of them, sorted by field.
func diffActivity(before, after Activity) ([]history.Change, error) {
	b, x := before, after
	b.Attributes, x.Attributes = nil, nil
	changes, err := history.Diff(b, x)
	if err != nil {
		return nil, err
	}
	attributeIDs := map[string]bool{}
	for id := range before.Attributes {
		attributeIDs[id] = true
	}
	for id := range after.Attributes {
		attributeIDs[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(attributeIDs)) {
		oldValue, newValue := before.Attributes[id], after.Attributes[id]
		if sameJSON(oldValue, newValue) || (isEmptyValue(oldValue) && isEmptyValue(newValue)) {
			continue
		}
		changes = append(changes, history.Change{Field: "attributes." + id, Old: orNull(oldValue), New: orNull(newValue)})
	}
	slices.SortStableFunc(changes, func(x, y history.Change) int { return strings.Compare(x.Field, y.Field) })
	return changes, nil
}

// orNull returns value, or JSON null for no value.
func orNull(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	return value
}
