package lineitems

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The history kind of line items and its parent, feature 006's activity (data model §7.1).
const (
	lineItemKind     = "lineItem"
	activityKind     = "activity"
	lineItemSingular = "line item"
)

// spreadLabels are the names of the spread methods on screens and in the log (design system §8).
var spreadLabels = map[string]string{
	string(SpreadEvenly):     "Evenly",
	string(SpreadFirstMonth): "All in first month",
	string(SpreadLastMonth):  "All in last month",
	string(SpreadCustom):     "Custom",
}

// Kinds returns the history kind of line items, for internal/app to register after feature 006's kind
// "activity" (data model §7.1). With it, history logs line-item changes with their path, deletes a line item
// (and an activity with its line items) to the recycle bin and brings it back.
func (s *Service) Kinds() []history.Kind {
	return []history.Kind{{
		Name:           lineItemKind,
		Singular:       lineItemSingular,
		Plural:         "line items",
		Parent:         activityKind,
		Snapshot:       s.snapshot,
		ChildIDs:       s.lineItemIDs,
		Remove:         s.repo.Delete,
		Put:            s.put,
		AllChildIDs:    s.allLineItemIDs,
		SnapshotsOf:    s.snapshotsOf,
		RemoveAll:      s.repo.DeleteMany,
		PutAll:         s.putAll,
		Describe:       s.describe,
		PrepareRestore: s.prepareRestore,
		AfterRestore:   s.afterRestore,
	}}
}

// Messages and notes of a line item that comes back (plan, "History notes").
const (
	undoNameTakenMessage = "Can't undo — there is now another line item called '%s'. Restore it from the recycle bin instead."
	renamedNote          = "Renamed to '%s' because the name was taken."
	restartedNote        = "Activity '%s' needs approval again because a line item came back."
	restoredReason       = "Line item '%s' (%s) restored: %s %s"
)

// prepareRestore applies the name rule of a line item that comes back (data model §7.1). Names stay unique
// in the activity: an Undo of a name that is taken meanwhile is refused (D-037), a Restore renames to "<name>
// (restored)", " (restored 2)", … and says so.
func (s *Service) prepareRestore(ctx context.Context, rc history.RestoreContext, snapshot *history.Snapshot) ([]string, error) {
	var li LineItem
	if err := json.Unmarshal(snapshot.Data, &li); err != nil {
		return nil, err
	}
	siblings, err := s.repo.ListByActivity(ctx, li.ActivityID) // the line item itself is not stored now
	if err != nil {
		return nil, err
	}
	var notes []string
	if other, taken := named(siblings, li.Name); taken {
		if rc.Undo {
			return nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, other.Name))
		}
		li.Name = uniqueName(li.Name, restoredNameSuffix, siblings, 0) // nothing but a missing parent blocks a restore
		li.Touch(rc.Actor, rc.Now)
		data, err := json.Marshal(li)
		if err != nil {
			return nil, err
		}
		snapshot.Name, snapshot.Data = li.Name, data
		notes = append(notes, fmt.Sprintf(renamedNote, li.Name))
	}
	return notes, nil
}

// afterRestore applies FR-033 once the line item the person deleted is back, after an Undo or a Restore:
// history calls it only when the line item itself was deleted, never when it comes back with its activity
// (or a level above). When its activity is Submitted or Approved now, bringing it back counts as adding one:
// the activity goes back to Created and a note says so (D-147). An Undo right after a delete that ended an
// approval finds the activity Created, so it never brings that approval back.
func (s *Service) afterRestore(ctx context.Context, rc history.RestoreContext, top history.Snapshot) ([]string, error) {
	var li LineItem
	if err := json.Unmarshal(top.Data, &li); err != nil {
		return nil, err
	}
	f, err := s.facts(ctx, li.ActivityID)
	if err != nil || !f.restartsApproval() {
		return nil, err
	}
	reason := fmt.Sprintf(restoredReason, li.Name, li.ID, li.PlannedCost.Format(), li.Currency)
	if err := s.deps.Activities.RestartApproval(ctx, rc.Actor, f.ID, []string{reason}); err != nil {
		return nil, err
	}
	return []string{fmt.Sprintf(restartedNote, f.Name)}, nil
}

// restoredNameSuffix is the suffix of the n-th restored name: " (restored)", " (restored 2)", …
func restoredNameSuffix(n int) string {
	if n == 1 {
		return " (restored)"
	}
	return fmt.Sprintf(" (restored %d)", n)
}

// snapshot reads the line item id as history keeps it: the stored line item, its name, and its activity as
// its parent.
func (s *Service) snapshot(ctx context.Context, id string) (history.Snapshot, bool, error) {
	li, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return history.Snapshot{}, false, err
	}
	data, err := json.Marshal(li)
	if err != nil {
		return history.Snapshot{}, false, err
	}
	return history.Snapshot{ID: li.ID, Name: li.Name, ParentID: li.ActivityID, Data: data}, true, nil
}

// lineItemIDs lists the ids of the line items of the activity activityID.
func (s *Service) lineItemIDs(ctx context.Context, activityID string) ([]string, error) {
	items, err := s.repo.ListByActivity(ctx, activityID)
	if err != nil {
		return nil, err
	}
	list := make([]string, 0, len(items))
	for _, li := range items {
		list = append(list, li.ID)
	}
	return list, nil
}

// put stores the line item of a snapshot again, exactly as the snapshot holds it.
func (s *Service) put(ctx context.Context, snapshot history.Snapshot) error {
	var li LineItem
	if err := json.Unmarshal(snapshot.Data, &li); err != nil {
		return err
	}
	return s.repo.Put(ctx, li)
}

// describe gives the label of a changed line-item field and the text of a value, for the activity log
// (data model §7.3): the owner's name, amounts as "46,000.00", the spread in words, the finance budget item
// as "2026 Lubricants Budget › Europe – Digital", and each Shell attribute as feature 002 describes it. An
// empty label or text means the default.
func (s *Service) describe(ctx context.Context, field string, value any) (label, text string, err error) {
	if strings.HasPrefix(field, "attributes.") {
		return s.deps.Attributes.Describe(ctx, field, value)
	}
	stored, _ := value.(string)
	switch field {
	case "activityId": // a split's move (016 FR-030): "'UK launch event' (A-1011)"
		if stored == "" {
			return "Activity", "", nil
		}
		f, found, err := s.deps.Activities.Facts(ctx, stored)
		if err != nil || !found {
			return "Activity", stored, nil
		}
		return "Activity", fmt.Sprintf("'%s' (%s)", f.Name, stored), nil
	case "ownerId":
		if stored == "" {
			return "Owner", "", nil
		}
		person, found, err := s.deps.People.Person(ctx, stored)
		if err != nil || !found {
			return "Owner", "", err
		}
		return "Owner", person.Name, nil
	case "plannedCost":
		return "Planned cost", amountText(stored), nil
	case "contractedAmount":
		return "Contracted amount", amountText(stored), nil
	case "spread":
		return "Spread over time", spreadText(value), nil
	case "requestSwbs":
		return "Request SWBS code from GSAP", "", nil
	case "financeBudgetYear":
		return "Finance budget year", "", nil
	case "financeBudgetItemId":
		if stored == "" {
			return "Finance budget item", "", nil
		}
		item, found, err := s.deps.FinanceBudgets.Item(ctx, stored)
		if err != nil || !found {
			return "Finance budget item", "", err
		}
		return "Finance budget item", item.BudgetName + " › " + item.Name, nil
	}
	return "", "", nil // name, description, currency: the defaults read well
}

// amountText writes a stored amount as people read it: "46000.00" → "46,000.00"; "" for no amount.
func amountText(stored string) string {
	amount, err := money.ParseAmount(stored)
	if stored == "" || err != nil {
		return ""
	}
	return amount.Format()
}

// spreadText writes a stored spread as people read it: "Evenly", or for a custom spread its months,
// "Custom: Feb 2026 2,000.00 · Mar 2026 3,333.33".
func spreadText(value any) string {
	spread, _ := value.(map[string]any)
	method, _ := spread["method"].(string)
	label := spreadLabels[method]
	months, _ := spread["months"].([]any)
	if method != string(SpreadCustom) || len(months) == 0 {
		return label
	}
	parts := make([]string, 0, len(months))
	for _, m := range months {
		month, _ := m.(map[string]any)
		name, _ := month["month"].(string)
		amount, _ := month["amount"].(string)
		parts = append(parts, monthLabel(name)+" "+amountText(amount))
	}
	return label + ": " + strings.Join(parts, " · ")
}
