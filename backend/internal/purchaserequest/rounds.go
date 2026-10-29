package purchaserequest

import (
	"context"
	"fmt"

	"mapp/internal/approval"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type roundEvent struct {
	kind  RoundKind
	event RoundEvent
}

// Facts is a purchase request as the approval engine sees it (brief §8.2): Amount is the USD total, nil
// without lines; Path is the plan … activity of its first line.
type Facts struct {
	ID, Name                           string
	Status                             Status
	OwnerID, MoAApproverID             string
	Amount                             *money.Amount
	WorkflowID, ModificationWorkflowID string
	Path                               []errs.RecordRef
	Plan                               authz.PlanFacts
	Overages                           []approval.Overage
	record.Meta
}

func (s *Service) factsByID(ctx context.Context, id string) (Facts, bool, error) {
	pr, found, err := s.repo.Get(ctx, id)
	if err != nil || !found {
		return Facts{}, found, err
	}
	f, err := s.factsFrom(ctx, pr)
	return f, true, err
}

// factsInScope returns the facts of the purchase requests among ids that plans (plans.view) or the person's
// purchaseRequests.list scope sees, by id.
func (s *Service) factsInScope(ctx context.Context, plans authz.Scope, ids []string) ([]Facts, error) {
	scope := ViewScope{Plans: plans, List: authz.PermissionScopeOf(ctx, authz.PurchaseRequestsList)}
	prs, err := s.repo.ListInScope(ctx, scope, ids)
	if err != nil {
		return nil, err
	}
	list := make([]Facts, 0, len(prs))
	for _, pr := range prs {
		f, err := s.factsFrom(ctx, pr)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, nil
}

func (s *Service) factsFrom(ctx context.Context, pr PurchaseRequest) (Facts, error) {
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return Facts{}, err
	}
	rates, err := s.deps.Currencies.Rates(ctx)
	if err != nil {
		return Facts{}, err
	}
	r := reader{rates: rates, facts: facts}
	f := Facts{ID: pr.ID, Name: pr.Name, Status: pr.Status, OwnerID: pr.OwnerID, MoAApproverID: pr.MoAApproverID,
		WorkflowID: pr.WorkflowID, ModificationWorkflowID: pr.ModificationWorkflowID, Plan: pr.planFacts(facts), Meta: pr.Meta}
	_, f.Path = r.plan(pr)
	f.Overages = overagesOf(pr, facts)
	if len(pr.Lines) > 0 {
		total := r.usd(pr.Lines)
		f.Amount = &total
	}
	return f, nil
}

func (s *Service) readyCheck(ctx context.Context, _ actor.Actor, id string, kind RoundKind) ([]Problem, error) {
	pr, err := s.stored(ctx, id)
	if err != nil {
		return nil, err
	}
	facts, err := s.factsOf(ctx, pr)
	if err != nil {
		return nil, err
	}
	return s.gates(ctx, pr, facts, kind)
}

// roundChanged sets the purchase request's status for a round event of the approval engine (Status Model,
// data-model §3.5), keeps or restores its approved version and routes it once approved. It writes no log entry
// of its own but the routing's; the engine logs the decision. While Purchase Support is off it refuses.
func (s *Service) roundChanged(ctx context.Context, a actor.Actor, id string, e roundEvent) (Facts, error) {
	kind, event := e.kind, e.event
	if _, err := s.switchedOn(ctx); err != nil {
		return Facts{}, err
	}
	pr, err := s.stored(ctx, id)
	if err != nil {
		return Facts{}, err
	}
	if kind == KindModification {
		pr, err = s.modificationEvent(ctx, a, pr, event)
	} else {
		pr, err = s.firstRoundEvent(ctx, a, pr, event)
	}
	if err != nil {
		return Facts{}, err
	}
	pr.Touch(a, s.now())
	if err = s.repo.Put(ctx, pr); err != nil {
		return Facts{}, err
	}
	return s.factsFrom(ctx, pr)
}

func (s *Service) firstRoundEvent(ctx context.Context, a actor.Actor, pr PurchaseRequest, event RoundEvent) (PurchaseRequest, error) {
	if !firstRoundAllows(pr.Status, event) {
		return PurchaseRequest{}, unexpected(pr, KindPurchaseRequest, event)
	}
	switch event {
	case EventOpen:
		pr.Status = opened(pr.Status)
	case EventDeclined:
		pr.Status = StatusDeclined
	case EventApproved:
		pr.Status, pr.Approved = StatusApproved, pr.snapshot(s.now())
		return pr, s.route(ctx, a, &pr, routing{typ: RoutingOriginal})
	case EventSuperseded, EventRecalled, EventCancelled:
		pr.Status = StatusCreated
	}
	return pr, nil
}

func firstRoundAllows(status Status, event RoundEvent) bool {
	inRound := status == StatusSubmitted || status == StatusInReview
	if event == EventOpen {
		return inRound || status == StatusCreated || status == StatusDeclined
	}
	return inRound
}

func opened(status Status) Status {
	if status == StatusSubmitted || status == StatusInReview {
		return StatusInReview
	}
	return StatusSubmitted
}

func (s *Service) modificationEvent(ctx context.Context, a actor.Actor, pr PurchaseRequest, event RoundEvent) (PurchaseRequest, error) {
	pending := pr.Status == StatusInReview && pr.ChangePending
	if !pending && (event != EventOpen || pr.Status != StatusApproved) {
		return PurchaseRequest{}, unexpected(pr, KindModification, event)
	}
	switch event {
	case EventOpen:
		pr.Status, pr.ChangePending = StatusInReview, true
	case EventApproved:
		previous := pr.Approved
		pr.Status, pr.ChangePending, pr.Approved = StatusApproved, false, pr.snapshot(s.now())
		return pr, s.route(ctx, a, &pr, routing{typ: RoutingModification, previous: previous})
	case EventDeclined, EventRecalled, EventCancelled:
		return s.restore(ctx, pr)
	case EventSuperseded:
	}
	return pr, nil
}

// restore brings back the last approved values of a change that was declined or recalled: lines the
// change added go, with their reservation.
func (s *Service) restore(ctx context.Context, pr PurchaseRequest) (PurchaseRequest, error) {
	v := pr.Approved
	pr.Name, pr.Description, pr.OwnerID, pr.MoAApproverID = v.Name, v.Description, v.OwnerID, v.MoAApproverID
	pr.Attributes, pr.Vendor = v.Attributes, v.Vendor
	kept := make([]Line, 0, len(pr.Lines))
	for _, l := range pr.Lines {
		before, found := v.approvedLine(l.ID)
		if !found {
			if err := s.repo.DeleteLine(ctx, l.ID); err != nil {
				return PurchaseRequest{}, err
			}
			continue
		}
		l.Currency, l.Amount, l.DeliveryDate, l.OverallLimit = before.Currency, before.Amount, before.DeliveryDate, before.OverallLimit
		l.Description, l.Attributes = before.Description, before.Attributes
		kept = append(kept, l)
	}
	pr.Lines, pr.Status, pr.ChangePending = kept, StatusApproved, false
	return pr, nil
}

func unexpected(pr PurchaseRequest, kind RoundKind, event RoundEvent) error {
	return fmt.Errorf("purchaserequest: %s round event %s does not apply to %s in status %s", kind, event, pr.ID, pr.Status)
}
