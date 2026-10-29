package financials

import (
	"context"
)

// SpendByMonth is the Committed and Paid of lineItemIDs, whose finance the caller may see, in USD by posting month
// ("2026-03"), at current rates as Facts sums them; a currency no longer in Settings is left out (036, D-358).
func (s *Service) SpendByMonth(ctx context.Context, lineItemIDs []string) (map[string]map[string]Facts, error) {
	spend := map[string]map[string]Facts{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.SpendOnLineItems(ctx, lineItemIDs)
		if err != nil {
			return err
		}
		rates, err := s.rates.Rates(ctx)
		if err != nil {
			return err
		}
		for i := range stored {
			t := &stored[i]
			usd := usdOf(t, rates)
			if usd == nil {
				continue
			}
			months := spend[t.LineItemID]
			if months == nil {
				months = map[string]Facts{}
				spend[t.LineItemID] = months
			}
			month := t.PostingDate.String()[:7]
			f := months[month]
			f.add(t, usd, "")
			months[month] = f
		}
		return nil
	})
	return spend, err
}

func (t *Transaction) spend() bool {
	return t.Type == TypePurchaseOrder || t.Type == TypePayment
}
