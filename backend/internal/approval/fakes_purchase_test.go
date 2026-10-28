package approval

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// purchaseEvent is one call of RoundChanged.
type purchaseEvent struct {
	kind    SubjectKind
	outcome Outcome
}

// fakePurchases stands in for feature 032: purchase requests in memory with 013's status mapping.
type fakePurchases struct {
	clock    *testClock
	facts    map[string]PurchaseRequestFacts
	problems map[string][]SubmitProblem
	events   map[string][]purchaseEvent
	order    []string
}

func (f *fakePurchases) PurchaseRequest(_ context.Context, id string) (PurchaseRequestFacts, bool, error) {
	facts, found := f.facts[id]
	return facts, found, nil
}

func (f *fakePurchases) ListInScope(_ context.Context, scope authz.Scope, ids []string) ([]PurchaseRequestFacts, error) {
	list := []PurchaseRequestFacts{}
	for _, id := range f.order {
		if facts := f.facts[id]; slices.Contains(ids, id) && scope.Sees(facts.Access) {
			list = append(list, facts)
		}
	}
	return list, nil
}

func (f *fakePurchases) ReadyCheck(_ context.Context, _ actor.Actor, id string, _ SubjectKind) (ReadyCheck, error) {
	return ReadyCheck{Problems: slices.Clone(f.problems[id])}, nil
}

func (f *fakePurchases) RoundChanged(_ context.Context, a actor.Actor, id string, kind SubjectKind, outcome Outcome) (PurchaseRequestFacts, error) { //nolint:revive // argument-limit 5: the port agreed with feature 032
	facts, found := f.facts[id]
	if !found {
		return PurchaseRequestFacts{}, fmt.Errorf("fake: purchase request %s is not stored", id)
	}
	f.events[id] = append(f.events[id], purchaseEvent{kind: kind, outcome: outcome})
	facts.Status = statusAfter(facts.Status, kind, outcome)
	facts.Touch(a, f.clock.Now())
	f.facts[id] = facts
	return facts, nil
}

// statusAfter is client 013's status model as 032 maps the events.
func statusAfter(status string, kind SubjectKind, outcome Outcome) string {
	modification := kind == KindModifiedPurchaseRequest
	switch outcome { //nolint:exhaustive // restarted and cancelled never reach purchase requests
	case OutcomeApproved:
		return StatusApproved
	case OutcomeOpen:
		if modification || status == StatusSubmitted {
			return StatusInReview
		}
		return StatusSubmitted
	case OutcomeDeclined, OutcomeRecalled, OutcomeSuperseded:
		return endedStatus(status, modification, outcome)
	}
	return status
}

func endedStatus(status string, modification bool, outcome Outcome) string {
	switch {
	case modification && outcome == OutcomeSuperseded:
		return status
	case modification:
		return StatusApproved
	case outcome == OutcomeDeclined:
		return StatusDeclined
	}
	return StatusCreated
}

// usePurchases connects a fake feature 032 and its history kind.
func (f *fixture) usePurchases(t *testing.T) *fakePurchases {
	t.Helper()
	prs := &fakePurchases{clock: f.clock, facts: map[string]PurchaseRequestFacts{}, problems: map[string][]SubmitProblem{},
		events: map[string][]purchaseEvent{}}
	err := f.history.Register(history.Kind{
		Name: string(KindPurchaseRequest), Singular: "purchase request", Plural: "purchase requests",
		Snapshot: func(_ context.Context, id string) (history.Snapshot, bool, error) {
			facts, found := prs.facts[id]
			return history.Snapshot{ID: id, Name: facts.Name}, found, nil
		},
		Remove: func(context.Context, string) error { return nil },
		Put:    func(context.Context, history.Snapshot) error { return nil },
	})
	if err != nil {
		t.Fatalf("Register(purchaseRequest): %v", err)
	}
	f.service.UsePurchaseRequests(prs)
	return prs
}

// purchaseRequest is a stand-in PR-… of plan P-1001 owned by owner, with its MoA approver and workflows.
type purchaseRequest struct {
	id, owner, moa, workflow, modification, amount string
}

func (f *fixture) addPurchaseRequest(t *testing.T, prs *fakePurchases, pr purchaseRequest) {
	t.Helper()
	var amount *money.Money
	if pr.amount != "" {
		usd := money.USD(mustAmount(t, pr.amount))
		amount = &usd
	}
	prs.facts[pr.id] = PurchaseRequestFacts{
		ID: pr.id, Name: "Agency fee " + pr.id, Status: StatusCreated, OwnerID: pr.owner, MoAApproverID: pr.moa,
		Amount: amount, WorkflowID: pr.workflow, ModificationWorkflowID: pr.modification,
		Path: []errs.RecordRef{{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"}},
		Meta: record.NewMeta(actor.System, start), Plan: PlanFacts{ID: "P-1001", TeamIDs: []string{"T-1001"}},
		Access: authz.PlanFacts{ID: "P-1001", TeamIDs: []string{"T-1001"}},
	}
	prs.order = append(prs.order, pr.id)
}

func (f *fixture) putKindWorkflow(t *testing.T, id string, kind SubjectKind, levels ...Level) {
	t.Helper()
	f.putWorkflow(t, id, string(kind)+" "+id, nil, nil, levels...)
	f.write(t, func(ctx context.Context) error {
		w, _, err := f.service.repo.GetWorkflow(ctx, id)
		if err != nil {
			return err
		}
		w.ObjectType = kind
		return f.service.repo.PutWorkflow(ctx, w)
	})
}

func (f *fixture) decideSubject(t *testing.T, a actor.Actor, action ActionKind, comment string) (Section, error) {
	t.Helper()
	round, _ := openRound(f.rounds(t, purchaseRequestA))
	return f.service.DecideSubject(t.Context(), a, KindPurchaseRequest, purchaseRequestA, SubjectDecision{Action: action,
		DecisionInput: DecisionInput{RoundID: round.ID, Level: version(round.CurrentLevel), Comment: comment}})
}
