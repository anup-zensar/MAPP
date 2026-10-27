package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// Split activity (feature 016, FR-026 – FR-031, data-model §3.6, D-267): one activity becomes two inside its
// plan. Whole line items move to the new activity and keep their IDs; part of the planned cost moves with
// them; goals stay on the original; the new activity remembers where it came from (splitFromId). Splitting a
// Submitted or Approved activity follows the re-approval rule of domain model §2.5. No undo.

// The messages of a split (016 plan "User-visible messages", data-model §3.6).
const (
	splitSuffix           = " (split)"
	msgSplitNothing       = "Choose line items or an amount to move."
	msgSplitTooMuch       = "Move at most %s, the activity's planned cost."
	msgSplitNoCost        = "This activity has no planned cost."
	msgSplitNotOfActivity = "%s is not a line item of this activity."
	msgSplitLineItemTwice = "Choose each line item once."
	splitVerb             = "split"
	splitIntoDetails      = "Into '%s' (%s): %s"
	splitFromDetails      = "Split from '%s' (%s)"
	splitRestartDetails   = "Approval restarted because the activity was split."
)

// Split splits the activity id, all or nothing (FR-028 – FR-030): the new activity — a copy of the original's
// type, owner, dates, ongoing, workflow, description, external name and Shell attributes, with the chosen name,
// the moved planned cost, no goals, status Created and the split link — the ticked line items moved to it with
// their IDs, values and spreads, the original's planned cost lowered by the moved amount, and the original back
// to Created when it was Submitted or Approved (FR-029). Each change is logged: "split" on the original,
// "created … Split from" on the new activity, "moved" on each line item.
func (s *Service) Split(ctx context.Context, a actor.Actor, id string, in SplitRequest) (SplitResult, error) {
	var result SplitResult
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		p, err := s.planSplit(ctx, id, in)
		if err != nil {
			return err
		}
		if err = p.refusal(in.Version); err != nil {
			return err
		}
		created, err := s.createSplitOff(ctx, a, p)
		if err != nil {
			return err
		}
		after, err := s.lowerOriginal(ctx, a, p, created)
		if err != nil {
			return err
		}
		if result.Activity, err = s.viewOf(ctx, after, p.pc); err != nil {
			return err
		}
		result.NewActivity, err = s.viewOf(ctx, created, p.target)
		return err
	})
	if err != nil {
		return SplitResult{}, err
	}
	if result.Activity, err = s.shown(ctx, a, result.Activity); err != nil {
		return SplitResult{}, err
	}
	if result.NewActivity, err = s.shown(ctx, a, result.NewActivity); err != nil {
		return SplitResult{}, err
	}
	return result, nil
}

// refusal stops the split p before it saves anything: its problems, a missing or an out-of-date version.
func (p *splitPlan) refusal(version *int) error {
	if version == nil || *version < 1 {
		p.problems.Add("version", reloadMessage)
	}
	if err := p.problems.Err(); err != nil {
		return err
	}
	return p.original.CheckVersion(*version, noun)
}

func (s *Service) createSplitOff(ctx context.Context, a actor.Actor, p splitPlan) (Activity, error) {
	created := newSplitActivity(p)
	var err error
	if created.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
		return Activity{}, err
	}
	created.Meta = record.NewMeta(a, s.now())
	if err = s.repo.Put(ctx, created); err != nil {
		return Activity{}, err
	}
	if _, err = s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: created.ID, RecordName: created.Name, Action: history.ActionCreated,
		Details: fmt.Sprintf(splitFromDetails, p.original.Name, p.original.ID),
	}); err != nil {
		return Activity{}, err
	}
	if len(p.moved) == 0 {
		return created, nil
	}
	movedIDs := make([]string, 0, len(p.moved))
	for _, li := range p.moved {
		movedIDs = append(movedIDs, li.ID)
	}
	if err = s.lineItems.MoveToActivity(ctx, a, movedIDs, created.ID); err != nil {
		return Activity{}, err
	}
	return created, nil
}

// lowerOriginal saves the original after the split p: less planned cost, and Created when approval restarts.
func (s *Service) lowerOriginal(ctx context.Context, a actor.Actor, p splitPlan, created Activity) (Activity, error) {
	before := p.original
	after := before
	if p.amount != nil && before.PlannedCost != nil {
		left := money.Money{Amount: before.PlannedCost.Amount.Sub(p.amount.Amount), Currency: before.PlannedCost.Currency}
		after.PlannedCost = &left
	}
	if p.restart {
		after.Status = StatusCreated
	}
	after.Touch(a, s.now())
	if err := s.repo.Put(ctx, after); err != nil {
		return Activity{}, err
	}
	var changes []history.Change
	if !samePlannedCost(before.PlannedCost, after.PlannedCost) {
		changes = append(changes, history.Change{Field: fieldPlannedCost, Old: mustJSON(before.PlannedCost), New: mustJSON(after.PlannedCost)})
	}
	entry, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Verb: splitVerb,
		Changes: changes, Details: fmt.Sprintf(splitIntoDetails, created.Name, created.ID, movedText(s, len(p.moved), p.amount)),
	})
	if err != nil {
		return Activity{}, err
	}
	if p.restart {
		if err = s.logRestart(ctx, a, restartLog{before: before, after: after, changes: entry.Changes, details: splitRestartDetails}); err != nil {
			return Activity{}, err
		}
	}
	return after, nil
}

// newSplitActivity is the new activity of a split (FR-028), without its ID and record facts.
func newSplitActivity(p splitPlan) Activity {
	o := p.original
	created := Activity{
		InitiativeID: p.target.initiative.ID, Name: p.name, ExternalName: o.ExternalName, ActivityTypeID: o.ActivityTypeID,
		OwnerID: o.OwnerID, StartDate: clonePtr(o.StartDate), EndDate: clonePtr(o.EndDate), Ongoing: o.Ongoing,
		ApprovalWorkflowID: o.ApprovalWorkflowID, Description: o.Description, Status: StatusCreated, SplitFromID: o.ID,
		Attributes: cloneValues(o.Attributes),
	}
	if p.amount != nil {
		moved := *p.amount
		created.PlannedCost = &moved
	}
	return created
}

// cloneValues returns a copy of Shell attribute values that shares nothing with them; never nil.
func cloneValues(values settings.AttributeValues) settings.AttributeValues {
	clone := settings.AttributeValues{}
	for id, v := range values {
		clone[id] = append(json.RawMessage{}, v...)
	}
	return clone
}

// movedText says what moved: "1 line item and 19,750.00 GBP", "2 line items", "5,000.00 GBP".
func movedText(s *Service, lineItems int, amount *money.Money) string {
	var parts []string
	if with := s.history.CountsText(map[string]int{lineItemKind: lineItems}); with != "" {
		parts = append(parts, strings.TrimPrefix(with, "with ")) // "with 1 line item" → "1 line item"
	}
	if amount != nil && !amount.Amount.IsZero() {
		parts = append(parts, amount.Format())
	}
	return strings.Join(parts, " and ")
}

// samePlannedCost reports whether two planned costs are the same.
func samePlannedCost(x, y *money.Money) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return *x == *y
}
