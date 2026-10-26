package lineitems

import (
	"context"
	"encoding/json"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Split activity (feature 016, FR-026 – FR-030, D-267) moves whole line items from one activity to another of
// the same plan. It is the only change of a line item's activity: everything else — ID, values, spread, SWBS
// request — stays, and the SWBS request status follows the new activity because it is worked out from the
// activity's status on read (FR-029).

// Brief is a line item in short, as the Split dialog lists it: its planned cost in its own currency and in
// the plan currency at the current rates (FR-026).
type Brief struct {
	ID, Name        string
	PlannedCost     money.Money // in the line item's currency
	PlannedCostPlan money.Money // in the plan currency
}

// Briefs returns the line items of the activity activityID in creation order, each with its planned cost in
// the plan currency (FR-026), inside the caller's transaction.
func (s *Service) Briefs(ctx context.Context, activityID string) ([]Brief, error) {
	briefs := []Brief{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.facts(ctx, activityID)
		if err != nil {
			return err
		}
		items, err := s.repo.ListByActivity(ctx, activityID)
		if err != nil {
			return err
		}
		views, err := s.views(ctx, f, items)
		if err != nil {
			return err
		}
		for _, v := range views {
			briefs = append(briefs, Brief{ID: v.ID, Name: v.Name, PlannedCost: v.PlannedCost, PlannedCostPlan: v.PlannedCostPlan})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return briefs, nil
}

// msgNotOfActivity refuses a line item that does not belong to the activity being split (data-model §3.6).
const msgNotOfActivity = "%s is not a line item of this activity."

// MoveToActivity moves the line items lineItemIDs to the activity toActivityID inside the caller's Write
// (FR-028): only their activity changes; each is touched and logged "moved" with the old and the new activity
// (FR-030). A line item that is not stored is refused, nothing is moved.
func (s *Service) MoveToActivity(ctx context.Context, a actor.Actor, lineItemIDs []string, toActivityID string) error {
	return s.tx.Write(ctx, func(ctx context.Context) error {
		for _, id := range lineItemIDs {
			li, found, err := s.repo.Get(ctx, id)
			if err != nil {
				return err
			}
			if !found {
				return errs.Validation(errs.FieldError{Field: "lineItemIds", Message: fmt.Sprintf(msgNotOfActivity, id)})
			}
			if li.ActivityID == toActivityID {
				continue
			}
			from := li.ActivityID
			li.ActivityID = toActivityID
			li.Touch(a, s.now())
			if err := s.repo.Put(ctx, li); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: lineItemKind, RecordID: li.ID, RecordName: li.Name, Action: history.ActionMoved,
				Changes: []history.Change{{Field: "activityId", Old: jsonText(from), New: jsonText(toActivityID)}},
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// jsonText encodes a text as a JSON string.
func jsonText(text string) json.RawMessage {
	encoded, _ := json.Marshal(text) // a string always encodes
	return encoded
}
