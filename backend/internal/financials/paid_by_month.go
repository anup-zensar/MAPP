package financials

import (
	"context"

	"mapp/internal/platform/money"
)

// PaidByMonth gives feature 036's dashboards the payments of each linked line item in USD by posting month
// ("2026-03"), at current rates as Paid is summed, currencies no longer in Settings left out. It checks no access.
func (s *Service) PaidByMonth(ctx context.Context) (map[string]map[string]money.Amount, error) {
	paid := map[string]map[string]money.Amount{}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.repo.ListTransactions(ctx)
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
			if t.Type != TypePayment || !t.linked() || usd == nil {
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
