package financials

import (
	"context"

	"mapp/internal/platform/money"
)

// PaidByMonth is the payments of lineItemIDs, whose finance the caller may see, in USD by posting month, at
// current rates as Paid sums them; a currency no longer in Settings is left out (036 FR-014, D-257, D-356).
func (s *Service) PaidByMonth(ctx context.Context, lineItemIDs []string) (map[string]map[string]money.Amount, error) {
	paid := map[string]map[string]money.Amount{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.PaymentsOnLineItems(ctx, lineItemIDs)
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
			months := paid[t.LineItemID]
			if months == nil {
				months = map[string]money.Amount{}
				paid[t.LineItemID] = months
			}
			month := t.PostingDate.String()[:7]
			months[month] = months[month].Add(*usd)
		}
		return nil
	})
	return paid, err
}
