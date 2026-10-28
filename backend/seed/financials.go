package seed

import (
	"context"

	"mapp/internal/financials"
	"mapp/internal/platform/money"
)

// addFinancials loads the simulated GSAP's demo data after approvals (D-306), without log entries (D-188): the
// SWBS run first (030), whose code Venue hire's transactions carry, then 015's transactions and their import.
func (d *demo) addFinancials(ctx context.Context) error {
	if err := d.addSwbsRun(ctx); err != nil {
		return err
	}
	items, err := d.stores.LineItems.List(ctx)
	if err != nil {
		return err
	}
	stored := storedLineItems{}
	for _, li := range items {
		stored[li.ID] = true
	}
	currencies, err := d.stores.Settings.ListCurrencies(ctx)
	if err != nil {
		return err
	}
	rates := money.Rates{}
	for _, c := range currencies {
		rates[c.Code] = c.Rate
	}
	_, err = financials.LoadDemo(ctx, d.stores.Financials, d.ids, financials.DemoLoad{Items: stored, Rates: rates, At: d.at})
	return err
}

type storedLineItems map[string]bool

func (s storedLineItems) Stored(id string) bool { return s[id] }
