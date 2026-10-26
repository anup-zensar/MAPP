package seed

import (
	"context"

	"mapp/internal/financials"
	"mapp/internal/platform/money"
)

// addFinancials loads the demo GSAP data of feature 015 (FR-031, data model §6): the 11 fictitious
// transactions of the demo feed, through the same matching rules as an import, and one import record "Demo
// data (simulated GSAP)" created by MAPP at the seed time. They link to Venue hire (L-1003), Speaker
// fees (L-1004) and Stand build (L-1006) only, plus one PO whose HX key matches no line item (D-306). The
// step runs after approvals and writes no activity-log entry (D-188).
func (d *demo) addFinancials(ctx context.Context) error {
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
	_, err = financials.LoadDemo(ctx, d.stores.Financials, d.ids, stored, rates, d.at)
	return err
}

// storedLineItems are the line items the seed stored, for matching: the seed has no recycle bin.
type storedLineItems map[string]bool

// Stored reports whether the line item id was stored.
func (s storedLineItems) Stored(id string) bool { return s[id] }
