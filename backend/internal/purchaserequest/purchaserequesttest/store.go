// Package purchaserequesttest is the in-memory fake of the purchase requests store, for tests that build services
// or the whole API without a database; its scope rule is purchase_request_in_scope() in Go.
package purchaserequesttest

import (
	"context"
	"slices"

	"mapp/internal/approval"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/plan"
	"mapp/internal/plan/plantest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
)

const configurationID = "purchase-requests"

type configurationRow struct {
	ID string `json:"id"`
	purchaserequest.Configuration
}

type Store struct {
	requests      *fake.Table[purchaserequest.PurchaseRequest]
	lines         *fake.Table[purchaserequest.Line]
	routings      *fake.Table[purchaserequest.Routing]
	configuration *fake.Table[configurationRow]
	plans         plantest.Store
	approvals     approvaltest.Store
}

func NewStore(db *fake.DB) Store {
	return Store{
		requests:      fake.TableOf(db, "purchase_requests", func(pr purchaserequest.PurchaseRequest) string { return pr.ID }),
		lines:         fake.TableOf(db, "purchase_request_lines", func(l purchaserequest.Line) string { return l.ID }),
		routings:      fake.TableOf(db, "purchase_request_routings", func(r purchaserequest.Routing) string { return r.ID }),
		configuration: fake.TableOf(db, "purchase_request_configuration", func(c configurationRow) string { return c.ID }),
		plans:         plantest.NewStore(db),
		approvals:     approvaltest.NewStore(db),
	}
}

func (s Store) Get(ctx context.Context, id string) (purchaserequest.PurchaseRequest, bool, error) {
	pr, found, err := s.requests.Get(id)
	if err != nil || !found {
		return purchaserequest.PurchaseRequest{}, found, err
	}
	withLines, err := s.withLines(ctx, []purchaserequest.PurchaseRequest{pr})
	return withLines[0], err == nil, err
}

func (s Store) All(ctx context.Context) ([]purchaserequest.PurchaseRequest, error) {
	all, err := s.requests.List()
	if err != nil {
		return nil, err
	}
	return s.withLines(ctx, all)
}

func (s Store) withLines(_ context.Context, prs []purchaserequest.PurchaseRequest) ([]purchaserequest.PurchaseRequest, error) {
	lines, err := s.lines.List()
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(lines, func(a, b purchaserequest.Line) int { return a.Position - b.Position })
	list := make([]purchaserequest.PurchaseRequest, 0, len(prs))
	for _, pr := range prs {
		pr.Lines = []purchaserequest.Line{}
		for _, l := range lines {
			if l.PurchaseRequestID == pr.ID {
				pr.Lines = append(pr.Lines, l)
			}
		}
		list = append(list, pr)
	}
	return list, nil
}

func (s Store) GetInScope(ctx context.Context, scope purchaserequest.ViewScope, id string) (purchaserequest.PurchaseRequest, bool, error) {
	found, err := s.ListInScope(ctx, scope, []string{id})
	if err != nil || len(found) == 0 {
		return purchaserequest.PurchaseRequest{}, false, err
	}
	return found[0], true, nil
}

func (s Store) ListInScope(ctx context.Context, scope purchaserequest.ViewScope, ids []string) ([]purchaserequest.PurchaseRequest, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	return s.keep(ctx, all, func(pr purchaserequest.PurchaseRequest) (bool, error) {
		if !slices.Contains(ids, pr.ID) {
			return false, nil
		}
		return s.sees(ctx, scope, pr)
	})
}

func (s Store) OnLineItems(ctx context.Context, scope purchaserequest.ViewScope, lineItemIDs []string) ([]purchaserequest.PurchaseRequest, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	return s.keep(ctx, all, func(pr purchaserequest.PurchaseRequest) (bool, error) {
		on := slices.ContainsFunc(pr.Lines, func(l purchaserequest.Line) bool { return slices.Contains(lineItemIDs, l.LineItemID) })
		if !on {
			return false, nil
		}
		return s.sees(ctx, scope, pr)
	})
}

func (s Store) keep(_ context.Context, all []purchaserequest.PurchaseRequest,
	wanted func(purchaserequest.PurchaseRequest) (bool, error)) ([]purchaserequest.PurchaseRequest, error) {
	list := []purchaserequest.PurchaseRequest{}
	for _, pr := range all {
		ok, err := wanted(pr)
		if err != nil {
			return nil, err
		}
		if ok {
			list = append(list, pr)
		}
	}
	return list, nil
}

// sees is purchase_request_in_scope(): the plan in either scope, the owner, the MoA approver, or an approver
// named in one of its rounds.
func (s Store) sees(ctx context.Context, scope purchaserequest.ViewScope, pr purchaserequest.PurchaseRequest) (bool, error) {
	person := scope.Plans.PersonID
	if scope.Plans.Everything || scope.List.Everything || pr.OwnerID == person || (person != "" && pr.MoAApproverID == person) {
		return true, nil
	}
	p, err := s.planFacts(ctx, pr.PlanID)
	if err != nil {
		return false, err
	}
	if scope.Plans.Sees(p) || scope.List.Covers(p) {
		return true, nil
	}
	return s.namedApprover(ctx, pr.ID, person)
}

func (s Store) planFacts(ctx context.Context, planID string) (authz.PlanFacts, error) {
	p, _, err := s.plans.Get(ctx, planID)
	return plan.AccessFacts(p), err
}

func (s Store) namedApprover(ctx context.Context, prID, person string) (bool, error) {
	rounds, err := s.approvals.ListRounds(ctx)
	if err != nil {
		return false, err
	}
	for _, r := range rounds {
		if r.SubjectID == prID && names(r, person) {
			return true, nil
		}
	}
	return false, nil
}

func names(r approval.Round, person string) bool {
	for _, level := range r.Levels {
		if slices.ContainsFunc(level.Approvers, func(a actor.Actor) bool { return a.ID == person }) {
			return true
		}
	}
	return false
}

func (s Store) Claims(ctx context.Context, lineItemIDs []string) ([]purchaserequest.Claim, error) {
	all, err := s.All(ctx)
	if err != nil {
		return nil, err
	}
	claims := []purchaserequest.Claim{}
	for _, pr := range all {
		for _, l := range pr.Lines {
			if slices.Contains(lineItemIDs, l.LineItemID) {
				claims = append(claims, purchaserequest.Claim{LineID: l.ID, PurchaseRequestID: pr.ID, PurchaseRequest: pr.Name,
					LineItemID: l.LineItemID, Status: pr.Status, Amount: money.Money{Amount: l.Amount, Currency: l.Currency}})
			}
		}
	}
	return claims, nil
}

func (s Store) Put(ctx context.Context, pr purchaserequest.PurchaseRequest) error {
	lines := pr.Lines
	pr.Lines = nil
	if err := s.requests.Put(ctx, pr); err != nil {
		return err
	}
	return s.lines.Put(ctx, lines...)
}

func (s Store) Delete(ctx context.Context, id string) error {
	return s.requests.Delete(ctx, id)
}

func (s Store) GetLine(_ context.Context, id string) (purchaserequest.Line, bool, error) {
	return s.lines.Get(id)
}

func (s Store) PutLine(ctx context.Context, l purchaserequest.Line) error {
	return s.lines.Put(ctx, l)
}

func (s Store) DeleteLine(ctx context.Context, id string) error {
	return s.lines.Delete(ctx, id)
}
