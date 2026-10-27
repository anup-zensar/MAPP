package activity

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
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
	targetKnown, err := s.checkSplitTarget(ctx, &p, in.InitiativeID)
	if err != nil {
		return splitPlan{}, err
	}
	if err = s.checkSplitName(ctx, &p, in.Name, targetKnown); err != nil {
		return splitPlan{}, err
	}
	if err = s.chooseLineItems(ctx, &p, id, in.LineItemIDs); err != nil {
		return splitPlan{}, err
	}
	amountKnown := p.checkAmount(in.PlannedCost)
	if p.restart && !in.RestartApproval {
		p.problems.Add("restartApproval", restartApprovalMessage)
	}
	p.sidesKnow = targetKnown && amountKnown
	return p, nil
}

// checkSplitTarget sets p's target and reports true when the initiative can take the new activity.
func (s *Service) checkSplitTarget(ctx context.Context, p *splitPlan, initiativeID string) (bool, error) {
	target, found, err := s.contextOf(ctx, strings.TrimSpace(initiativeID))
	switch {
	case err != nil:
		return false, err
	case !found || target.plan.ID != p.pc.plan.ID:
		p.problems.Add("initiativeId", chooseAnotherMessage)
	case target.initiative.ID != p.original.InitiativeID && !datesFit(p.original, target.initiative):
		p.problems.Add("initiativeId", datesDontFit(p.original, target.initiative))
	default:
		p.target = target
		return true, nil
	}
	return false, nil
}

// checkSplitName sets the new activity's name; whether it is free is known only once the target is.
func (s *Service) checkSplitName(ctx context.Context, p *splitPlan, name string, targetKnown bool) error {
	p.name = strings.TrimSpace(name)
	switch {
	case p.name == "":
		p.problems.Add("name", enterNameMessage)
	case utf8.RuneCountInString(p.name) > maxNameLength:
		p.problems.Add("name", nameTooLongMessage)
	case targetKnown:
		siblings, err := s.repo.ListByInitiative(ctx, p.target.initiative.ID)
		if err != nil {
			return err
		}
		if other, taken := namedLike(siblings, p.name, ""); taken {
			p.problems.Add("name", fmt.Sprintf(nameTakenFormat, p.target.initiative.Name, other.Name))
		}
	}
	return nil
}

func (s *Service) chooseLineItems(ctx context.Context, p *splitPlan, id string, chosenIDs []string) error {
	items, err := s.lineItems.Items(ctx, id)
	if err != nil {
		return err
	}
	chosen := map[string]bool{}
	for _, liID := range chosenIDs {
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
	return nil
}

// checkAmount sets the planned cost that moves and reports whether it is known; a split must move something.
func (p *splitPlan) checkAmount(text *string) bool {
	amount, problem := splitAmount(text, p.original.PlannedCost, p.pc.plan.Currency)
	p.amount = amount
	if problem != "" {
		p.problems.Add(fieldPlannedCost, problem)
		return false
	}
	if len(p.moved) == 0 && (amount == nil || amount.Amount.Sign() == 0) {
		p.problems.Add("lineItemIds", msgSplitNothing)
	}
	return true
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
