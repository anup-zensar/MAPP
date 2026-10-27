package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
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

// splitPlan is a split worked out: what it changes, and the problems that stop it.
type splitPlan struct {
	original  Activity
	pc        planContext
	target    planContext // the initiative of the new activity
	name      string
	moved     []LineItemBrief // the ticked line items, in the activity's order
	kept      []LineItemBrief
	amount    *money.Money // the planned cost that moves; nil when the original has none
	restart   bool         // the original is Submitted or Approved
	problems  errs.Problems
	sidesKnow bool // the target and the amount are known: the sides can be worked out
}

// planSplit checks a split of the activity id (data-model §3.6) and works out what it changes, inside the
// caller's transaction. Every problem is collected; a missing activity is not found.
func (s *Service) planSplit(ctx context.Context, id string, in SplitRequest) (splitPlan, error) {
	original, pc, err := s.load(ctx, id)
	if err != nil {
		return splitPlan{}, err
	}
	p := splitPlan{original: original, pc: pc, restart: inApproval(original)}

	targetKnown := false
	target, found, err := s.contextOf(ctx, strings.TrimSpace(in.InitiativeID))
	switch {
	case err != nil:
		return splitPlan{}, err
	case !found || target.plan.ID != pc.plan.ID:
		p.problems.Add("initiativeId", chooseAnotherMessage)
	case target.initiative.ID != original.InitiativeID && !datesFit(original, target.initiative):
		p.problems.Add("initiativeId", datesDontFit(original, target.initiative))
	default:
		p.target, targetKnown = target, true
	}

	p.name = strings.TrimSpace(in.Name)
	switch {
	case p.name == "":
		p.problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(p.name) > maxNameLength:
		p.problems.Add("name", nameTooLongMessage)
	case targetKnown:
		siblings, err := s.repo.ListByInitiative(ctx, target.initiative.ID)
		if err != nil {
			return splitPlan{}, err
		}
		if other, taken := namedLike(siblings, p.name, ""); taken {
			p.problems.Add("name", fmt.Sprintf(nameTakenFormat, target.initiative.Name, other.Name))
		}
	}

	items, err := s.lineItems.Items(ctx, id)
	if err != nil {
		return splitPlan{}, err
	}
	chosen := map[string]bool{}
	for _, liID := range in.LineItemIDs {
		liID = strings.TrimSpace(liID)
		switch {
		case chosen[liID]:
			p.problems.Add("lineItemIds", msgSplitLineItemTwice)
		case !slices.ContainsFunc(items, func(li LineItemBrief) bool { return li.ID == liID }):
			p.problems.Add("lineItemIds", fmt.Sprintf(msgSplitNotOfActivity, liID))
		}
		chosen[liID] = true
	}
	for _, li := range items {
		if chosen[li.ID] {
			p.moved = append(p.moved, li)
		} else {
			p.kept = append(p.kept, li)
		}
	}

	amountKnown := true
	amount, amountProblem := splitAmount(in.PlannedCost, original.PlannedCost, pc.plan.Currency)
	if amountProblem != "" {
		p.problems.Add("plannedCost", amountProblem)
		amountKnown = false
	}
	p.amount = amount
	if len(p.moved) == 0 && (amount == nil || amount.Amount.Sign() == 0) && amountProblem == "" {
		p.problems.Add("lineItemIds", msgSplitNothing)
	}
	if p.restart && !in.RestartApproval {
		p.problems.Add("restartApproval", restartApprovalMessage)
	}
	p.sidesKnow = targetKnown && amountKnown
	return p, nil
}

// splitAmount reads the planned cost to move (data-model §3.6): 0.00 up to the original's planned cost, in the
// plan currency; none (nil, "" or 0) when the original has no planned cost. It returns the amount — nil when
// the original has none — or the problem.
func splitAmount(text *string, original *money.Money, currency string) (*money.Money, string) {
	typed := ""
	if text != nil {
		typed = strings.TrimSpace(*text)
	}
	var amount money.Amount
	if typed != "" {
		parsed, err := money.ParseAmount(typed)
		if err != nil {
			return nil, enterAmountMessage
		}
		amount = parsed
	}
	if original == nil {
		if !amount.IsZero() {
			return nil, msgSplitNoCost
		}
		return nil, ""
	}
	if amount.Sign() < 0 || amount.Cmp(original.Amount) > 0 {
		return nil, fmt.Sprintf(msgSplitTooMuch, original.Format())
	}
	return &money.Money{Amount: amount, Currency: currency}, ""
}

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
		if in.Version == nil || *in.Version < 1 {
			p.problems.Add("version", reloadMessage)
		}
		if err := p.problems.Err(); err != nil {
			return err
		}
		if err := p.original.CheckVersion(*in.Version, noun); err != nil {
			return err
		}

		created := newSplitActivity(p)
		if created.ID, err = s.ids.Next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		created.Meta = record.NewMeta(a, s.now())
		if err := s.repo.Put(ctx, created); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: created.ID, RecordName: created.Name, Action: history.ActionCreated,
			Details: fmt.Sprintf(splitFromDetails, p.original.Name, p.original.ID),
		}); err != nil {
			return err
		}
		movedIDs := make([]string, 0, len(p.moved))
		for _, li := range p.moved {
			movedIDs = append(movedIDs, li.ID)
		}
		if len(movedIDs) > 0 {
			if err := s.lineItems.MoveToActivity(ctx, a, movedIDs, created.ID); err != nil {
				return err
			}
		}

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
			return err
		}
		var changes []history.Change
		if !samePlannedCost(before.PlannedCost, after.PlannedCost) {
			changes = append(changes, history.Change{Field: "plannedCost", Old: mustJSON(before.PlannedCost), New: mustJSON(after.PlannedCost)})
		}
		entry, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: after.ID, RecordName: after.Name, Action: history.ActionChanged, Verb: splitVerb,
			Changes: changes, Details: fmt.Sprintf(splitIntoDetails, created.Name, created.ID, movedText(s, len(p.moved), p.amount)),
		})
		if err != nil {
			return err
		}
		if p.restart {
			if err = s.logRestart(ctx, a, restartLog{before: before, after: after, changes: entry.Changes, details: splitRestartDetails}); err != nil {
				return err
			}
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
