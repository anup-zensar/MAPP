package purchaserequest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

func (s *Service) Requested(ctx context.Context, lineItemID, currency string) (money.Amount, []string, error) {
	b, err := s.balanceOf(ctx, []string{lineItemID})
	if err != nil {
		return money.Amount{}, nil, err
	}
	requested, err := b.requested(LineItemFacts{ID: lineItemID, Currency: currency}, "")
	if errors.Is(err, money.ErrUnknownCurrency) {
		return money.Amount{}, nil, nil
	}
	return requested, prIDs(b.claims), err
}

// RefuseDrawnOn refuses, naming a purchase request, to remove line items that one draws on; whoever
// asks, as every claim counts.
func (s *Service) RefuseDrawnOn(ctx context.Context, lineItemIDs []string) error {
	if len(lineItemIDs) == 0 {
		return nil
	}
	claims, err := s.repo.Claims(ctx, lineItemIDs)
	if err != nil || len(claims) == 0 {
		return err
	}
	return errs.InUse(fmt.Sprintf(msgDrawnOn, claims[0].PurchaseRequestID, claims[0].LineItemID))
}

// RefuseMove refuses to move line items to another plan while a purchase request draws on one.
func (s *Service) RefuseMove(ctx context.Context, lineItemIDs []string) error {
	if len(lineItemIDs) == 0 {
		return nil
	}
	claims, err := s.repo.Claims(ctx, lineItemIDs)
	if err != nil || len(claims) == 0 {
		return err
	}
	return errs.InUse(fmt.Sprintf(msgCampaignMove, strings.Join(prIDs(claims), ", ")))
}

func prIDs(claims []Claim) []string {
	ids := []string{}
	for _, c := range claims {
		if !slices.Contains(ids, c.PurchaseRequestID) {
			ids = append(ids, c.PurchaseRequestID)
		}
	}
	return ids
}
