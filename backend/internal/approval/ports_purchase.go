package approval

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// PurchaseRequests is feature 032's side of approvals (spec 031 contracts/go-api.md), in the caller's transaction.
type PurchaseRequests interface {
	PurchaseRequest(ctx context.Context, id string) (PurchaseRequestFacts, bool, error)
	ListInScope(ctx context.Context, scope authz.Scope, ids []string) ([]PurchaseRequestFacts, error)
	ReadyCheck(ctx context.Context, a actor.Actor, id string, kind SubjectKind) (ReadyCheck, error)
	// RoundChanged lets 032 set the PR status for a round event; approvals writes the log entry.
	RoundChanged(ctx context.Context, a actor.Actor, id string, kind SubjectKind, outcome Outcome) (PurchaseRequestFacts, error)
}

type PurchaseRequestFacts struct {
	ID, Name, Status       string
	OwnerID, MoAApproverID string
	Amount                 *money.Money
	WorkflowID             string
	ModificationWorkflowID string
	Path                   []errs.RecordRef
	record.Meta
	Access   authz.PlanFacts
	Plan     PlanFacts
	Overages []Overage
}

const (
	StatusInReview = "inReview"
	StatusClosed   = "closed"
)

const (
	GroupPurchaseRequest     = "purchaseRequest"
	GroupPurchaseRequestLine = "purchaseRequestLine"
)

func (s *Service) UsePurchaseRequests(p PurchaseRequests) {
	s.purchases = p
}
