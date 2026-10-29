package purchaserequest

import (
	"context"

	"mapp/internal/platform/money"
)

// Summary is a purchase request as feature 036's dashboards count it, its amount in USD as the Purchase requests
// page counts it.
type Summary struct {
	ID             string
	PlanID         string
	Status         Status
	AmountUSD      money.Amount
	WaitingForGSAP bool
}

type Summaries struct {
	SwitchedOn bool
	Items      []Summary
}

// Summaries returns the purchase requests on the line items lineItemIDs that the person acting may see.
func (s *Service) Summaries(ctx context.Context, lineItemIDs []string) (Summaries, error) {
	out := Summaries{Items: []Summary{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		config, err := s.repo.Configuration(ctx)
		if err != nil {
			return err
		}
		out.SwitchedOn = config.PurchaseSupport
		r := &reader{}
		if r.rates, err = s.deps.Currencies.Rates(ctx); err != nil {
			return err
		}
		prs, err := s.repo.OnLineItems(ctx, viewScope(ctx), lineItemIDs)
		if err != nil {
			return err
		}
		for _, pr := range prs {
			out.Items = append(out.Items, Summary{ID: pr.ID, PlanID: pr.PlanID, Status: pr.Status,
				AmountUSD: r.usd(pr.Lines), WaitingForGSAP: waiting(pr)})
		}
		return nil
	})
	return out, err
}
