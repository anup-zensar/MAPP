package activity

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// PreviewSplit returns both activities as the split of in would leave them, with every problem the split
// would answer, and saves nothing (FR-027). A wrong version is not a problem here: the split itself answers it.
func (s *Service) PreviewSplit(ctx context.Context, _ actor.Actor, id string, in SplitRequest) (SplitPreview, error) {
	preview := SplitPreview{Problems: []errs.FieldError{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.planSplit(ctx, id, in)
		if err != nil {
			return err
		}
		for _, problem := range p.problems {
			preview.Problems = append(preview.Problems, errs.FieldError{Field: problem.Field, Message: problem.Message})
		}
		if p.sidesKnow {
			original, created := p.sides()
			preview.Original, preview.Created = &original, &created
		}
		return nil
	})
	if err != nil {
		return SplitPreview{}, err
	}
	return preview, nil
}

// sides works out both activities as the split would leave them (FR-027): planned costs, line items, the
// unassigned planned cost in the plan currency (red when negative) and the status.
func (p splitPlan) sides() (original, created SplitSide) {
	var remaining *money.Money
	if p.original.PlannedCost != nil {
		left := money.Money{Amount: p.original.PlannedCost.Amount, Currency: p.pc.plan.Currency}
		if p.amount != nil {
			left.Amount = left.Amount.Sub(p.amount.Amount)
		}
		remaining = &left
	}
	status := p.original.Status
	if p.restart {
		status = StatusCreated
	}
	original = side(SplitSide{Name: p.original.Name, InitiativeName: p.pc.initiative.Name, PlannedCost: remaining, Status: status},
		p.kept, p.pc.plan.Currency)
	created = side(SplitSide{Name: p.name, InitiativeName: p.target.initiative.Name, PlannedCost: p.amount, Status: StatusCreated},
		p.moved, p.pc.plan.Currency)
	return original, created
}

// side is one activity of a split, named, placed and with its planned cost and status as in, with items as its
// line items: its unassigned planned cost is the planned cost minus its line items in the plan currency.
func side(in SplitSide, items []LineItemBrief, currency string) SplitSide {
	planned := in.PlannedCost
	s := SplitSide{Name: in.Name, InitiativeName: in.InitiativeName, PlannedCost: planned, LineItemNames: []string{}, Status: in.Status,
		UnassignedState: unassignedNoPlannedCost}
	var assigned money.Amount
	for _, li := range items {
		s.LineItemNames = append(s.LineItemNames, li.Name)
		assigned = assigned.Add(li.PlannedCostPlan.Amount)
	}
	if planned == nil {
		return s
	}
	balance := money.Money{Amount: planned.Amount.Sub(assigned), Currency: currency}
	s.UnassignedPlannedCost = &balance
	switch balance.Amount.Sign() {
	case 1:
		s.UnassignedState = unassignedLeft
	case 0:
		s.UnassignedState = unassignedAllAssigned
	default:
		s.UnassignedState = unassignedExceeded
	}
	return s
}
