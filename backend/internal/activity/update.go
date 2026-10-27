package activity

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
		after, err := s.checkUpdate(ctx, pc, before, update)
		if err != nil {
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
		after, err = s.saveEdit(ctx, a, edit{before: before, after: after, changes: changes, restartFields: restartFields})
		if err != nil {
			return err
		}
		view, err = s.viewOf(ctx, after, pc)
		return err
	})
	if err != nil {
		return View{}, err
	}
	return s.shown(ctx, a, view)
}

// checkUpdate refuses an out-of-date version, then checks every field of update and the version together,
// and returns the activity as it would be saved.
func (s *Service) checkUpdate(ctx context.Context, pc planContext, before Activity, update Update) (Activity, error) {
	if update.Version != nil && *update.Version >= 1 {
		if err := before.CheckVersion(*update.Version, noun); err != nil {
			return Activity{}, err
		}
	}
	var problems errs.Problems
	after, err := s.checkFields(ctx, &problems, fieldCheck{pc: pc, in: update.fields(), before: &before})
	if err != nil {
		return Activity{}, err
	}
	if update.Version == nil || *update.Version < 1 {
		problems.Add("version", reloadMessage)
	}
	return after, problems.Err()
}

// edit is a save that changes an activity: the activity before and after it, its changes, and the changed
// fields, in words, that restart its approval.
type edit struct {
	before, after Activity
	changes       []history.Change
	restartFields []string
}

// saveEdit saves and logs e, back in Created when it restarts the approval, tells the line items when the
// dates changed, and returns the activity as saved.
func (s *Service) saveEdit(ctx context.Context, a actor.Actor, e edit) (Activity, error) {
	after := e.after
	if len(e.restartFields) > 0 {
		after.Status = StatusCreated
	}
	after.Touch(a, s.now())
	if err := s.repo.Put(ctx, after); err != nil {
		return Activity{}, err
	}
	entry, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Changes: e.changes,
	})
	if err != nil {
		return Activity{}, err
	}
	if datesChanged(e.before, after) {
		if err = s.lineItems.DatesChanged(ctx, a, e.before, after); err != nil {
			return Activity{}, err
		}
	}
	if len(e.restartFields) > 0 {
		details := fmt.Sprintf(approvalRestartedFormat, joinWords(e.restartFields))
		if err = s.logRestart(ctx, a, restartLog{before: e.before, after: after, changes: entry.Changes, details: details}); err != nil {
			return Activity{}, err
		}
	}
	return after, nil
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
