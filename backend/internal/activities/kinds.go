package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"northstar/internal/history"
	"northstar/internal/platform/date"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
)

// kindName is the history kind of activities (data model §5); feature 007's lineItem kind names it as its
// parent.
const kindName = "activity"

// parentKind is the kind of an activity's parent, registered by feature 005 before activities.
const parentKind = "initiative"

// Messages of the plan's "Undo refused / restore notes (API)" row.
const (
	undoNameTakenFormat = "Can't undo — there is now another activity called '%s'. Restore it from the recycle bin instead."
	renamedFormat       = "Renamed to '%s' because the name was taken."
	statusResetNote     = "Status reset to Created because it was restored from the recycle bin."
)

// labels are the activity-log labels of the activity's fields (data model §5).
var labels = map[string]string{
	"name": "Activity name", "externalName": "External name", "description": "Description",
	"activityTypeId": "Activity type", "ownerId": "Owner", "startDate": "Start date", "endDate": "End date",
	"ongoing": "Ongoing (no end date)", "setLaterStagesMyself": "Set later stages myself",
	"plannedCost": "Planned cost", "revenue": "Revenue", "approvalWorkflowId": "Approval workflow",
	"inq": "INQ", "mql": "MQL", "status": "Status", "initiativeId": "Initiative", "parent": "Initiative",
}

// Kinds returns the history kind of activities, for internal/app to register after feature 005's
// initiative kind (FR-031). With it, history deletes an activity with everything under it (feature 007's
// line items), brings it back, and logs its changes with readable labels and texts.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{
		Name:           kindName,
		Singular:       "activity",
		Plural:         "activities",
		Parent:         parentKind,
		Snapshot:       s.snapshot,
		ChildIDs:       s.childIDs,
		Remove:         s.repo.Delete,
		Put:            s.putSnapshot,
		Describe:       s.describe,
		PrepareRestore: s.prepareRestore,
	}}
}

// snapshot reads the activity id as history keeps it: the stored activity, its name, and its initiative as
// its parent.
func (s *Service) snapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	activity, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(activity)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: activity.ID, Name: activity.Name, ParentID: activity.InitiativeID, Data: data}, true, nil
}

// childIDs lists the ids of the activities of the initiative initiativeID.
func (s *Service) childIDs(ctx context.Context, initiativeID string) ([]string, error) {
	list, err := s.repo.ListByInitiative(ctx, initiativeID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, a := range list {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

// putSnapshot stores the activity of a snapshot again, exactly as the snapshot holds it.
func (s *Service) putSnapshot(ctx context.Context, snapshot history.Snapshot) error {
	var activity Activity
	if err := json.Unmarshal(snapshot.Data, &activity); err != nil {
		return err
	}
	return s.repo.Put(ctx, activity)
}

// prepareRestore applies the rules of activities just before one comes back (FR-031, data model §5). The
// name must still be free in its initiative: an Undo, which puts the activity back exactly, is refused
// with name_taken when it is not; a restore from the recycle bin renames it "<name> (restored)" instead. A
// restore from the recycle bin also returns it to Created (R14) and tells feature 008; an Undo keeps its
// status.
func (s *Service) prepareRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var activity Activity
	if err := json.Unmarshal(snapshot.Data, &activity); err != nil {
		return nil, err
	}
	siblings, err := s.repo.ListByInitiative(ctx, activity.InitiativeID)
	if err != nil {
		return nil, err
	}
	var notes []string
	changed := false
	if other, taken := namedLike(siblings, activity.Name, activity.ID); taken {
		if rc.Undo {
			return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenFormat, other.Name))
		}
		activity.Name = restoredName(siblings, activity.Name, activity.ID)
		notes = append(notes, fmt.Sprintf(renamedFormat, activity.Name))
		changed = true
	}
	if !rc.Undo {
		if activity.Status != StatusCreated {
			activity.Status = StatusCreated
			notes = append(notes, statusResetNote)
			changed = true
		}
		hookNotes, err := s.hooks.ActivityRestored(ctx, rc, activity.ID)
		if err != nil {
			return nil, err
		}
		notes = append(notes, hookNotes...)
	}
	if !changed {
		return notes, nil
	}
	activity.Touch(rc.Actor, rc.Now) // a changed record gets a new version
	data, err := json.Marshal(activity)
	if err != nil {
		return nil, err
	}
	snapshot.Name, snapshot.Data = activity.Name, data
	return notes, nil
}

// describe gives the activity log's label of field and the text of value (data model §5): type, owner and
// workflow names, dates as "01 Mar 2026", amounts as "120,000.00 USD", numbers with thousands separators,
// status words, and Shell attribute labels and option texts from Settings. An empty text means "use the
// default" ("Not filled in" for empty values).
func (s *Service) describe(ctx context.Context, field string, value any) (string, string, error) {
	label := labels[field]
	if label == "" {
		return s.settings.DescribeAttribute(ctx, field, value)
	}
	text, isText := value.(string)
	switch field {
	case "activityTypeId":
		if isText && text != "" {
			t, found, err := s.settings.ActivityType(ctx, text)
			if err != nil || !found {
				return label, "", err
			}
			return label, t.Name, nil
		}
	case "ownerId":
		ref, err := s.personRef(ctx, text)
		if err != nil || ref == nil {
			return label, "", err
		}
		return label, ref.Name, nil
	case "approvalWorkflowId":
		if isText && text != "" {
			w, found, err := s.workflows.GetWorkflow(ctx, text)
			if err != nil || !found {
				return label, "", err
			}
			return label, w.Name, nil
		}
	case "startDate", "endDate":
		if d, err := date.Parse(text); isText && err == nil {
			return label, date.Format(d), nil
		}
	case "plannedCost", "revenue":
		if m, ok := value.(map[string]any); ok {
			amount, err := money.ParseAmount(fmt.Sprint(m["amount"]))
			if err == nil {
				return label, money.Money{Amount: amount, Currency: fmt.Sprint(m["currency"])}.Format(), nil
			}
		}
	case "inq", "mql":
		if n, ok := value.(json.Number); ok {
			if whole, err := n.Int64(); err == nil {
				return label, groupedInt(whole), nil
			}
		}
	case "status":
		return label, statusWords[Status(text)], nil
	case "parent": // a move: the initiative, as {type, id, name}
		if ref, ok := value.(map[string]any); ok {
			id, _ := ref["id"].(string)
			name, _ := ref["name"].(string)
			path, err := s.initiativePath(ctx, id, name)
			return label, path, err
		}
	}
	return label, "", nil
}

// initiativePath names the initiative id with its campaign, as a move's log entry shows it: "Helix launch ›
// Digital & social"; name alone when the initiative or its campaign is gone.
func (s *Service) initiativePath(ctx context.Context, id, name string) (string, error) {
	initiative, found, err := s.hierarchy.GetInitiative(ctx, id)
	if err != nil || !found {
		return name, err
	}
	campaign, found, err := s.hierarchy.GetCampaign(ctx, initiative.CampaignID)
	if err != nil || !found {
		return initiative.Name, err
	}
	return campaign.Name + " › " + initiative.Name, nil
}

// groupedInt writes a whole number with thousands separators: 250000 → "250,000".
func groupedInt(n int64) string {
	digits := strconv.FormatInt(n, 10)
	sign := ""
	if n < 0 {
		sign, digits = "-", digits[1:]
	}
	for i := len(digits) - 3; i > 0; i -= 3 {
		digits = digits[:i] + "," + digits[i:]
	}
	return sign + digits
}
