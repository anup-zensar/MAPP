package lineitem

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const plannedBelowRequestedMessage = "Purchase requests %s already request %s %s of this line item; the planned cost cannot be lower."

type PurchaseRequests interface {
	// Requested returns what purchase requests (032) request of the line item in currency at today's rates, and which.
	Requested(ctx context.Context, lineItemID, currency string) (money.Amount, []string, error)
}

func (s *Service) UsePurchaseRequests(p PurchaseRequests) {
	s.purchases = p
}

// requestedRule refuses a planned cost or currency that leaves the line item below what purchase requests request.
func (s *Service) requestedRule(ctx context.Context, problems *errs.Problems, id string, after LineItem) error {
	if s.purchases == nil || after.Currency == "" {
		return nil
	}
	requested, prs, err := s.purchases.Requested(ctx, id, after.Currency)
	if err != nil || len(prs) == 0 || after.PlannedCost.Cmp(requested) >= 0 {
		return err
	}
	problems.Add("plannedCost", fmt.Sprintf(plannedBelowRequestedMessage, strings.Join(prs, ", "), requested.Format(), after.Currency))
	return nil
}
