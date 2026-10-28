package campaign

import "context"

// PurchaseRequests answers in_use, naming them, when purchase requests draw on a line item of the campaign (032).
type PurchaseRequests interface {
	RefuseCampaignMove(ctx context.Context, campaignID string) error
}

func (s *Service) UsePurchaseRequests(p PurchaseRequests) {
	s.purchases = p
}

func (s *Service) refuseDrawnOn(ctx context.Context, campaignID string, otherPlan bool) error {
	if s.purchases == nil || !otherPlan {
		return nil
	}
	return s.purchases.RefuseCampaignMove(ctx, campaignID)
}
