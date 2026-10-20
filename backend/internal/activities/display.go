package activities

import (
	"context"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/money"
)

// The amounts an activity shows — its planned cost, its unassigned planned cost and its revenue goal — are
// answered a second time in the reader's display currency (D-061, D-220): `display` on the activity and on
// each plan-tree row. Each is converted once from USD, half up to the cent. The USD fields stay as they
// are, because the activity form types the planned cost in USD, compares it with the saved one for the
// re-approval question (FR-027) and the web app never converts.

// DisplayRates gives the display currency a person shows amounts in, with its rate in units per 1 USD
// (feature 003). internal/app hands people's service to UseDisplayRates; without it amounts stay in USD.
type DisplayRates interface {
	DisplayRateOf(ctx context.Context, personID string) (people.DisplayRate, error)
}

// UseDisplayRates sets where the display currency of the reader is read (D-220).
func (s *Service) UseDisplayRates(rates DisplayRates) {
	s.displayRates = rates
}

// Display is an activity's amounts in the reader's display currency (ActivityDisplay).
type Display struct {
	PlannedCost           *money.Money `json:"plannedCost"`
	UnassignedPlannedCost *money.Money `json:"unassignedPlannedCost"`
	Revenue               *money.Money `json:"revenue"` // the revenue goal; always null on a plan-tree row
}

// displayFor returns the display currency of the person a, read in a transaction of its own (or the
// caller's).
// The planned costs are in their plan currency (016 FR-013): they are converted from it with the current rates.
func (s *Service) displayFor(ctx context.Context, a actor.Actor) (money.Display, error) {
	d := money.DisplayUSD
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		rates, err := s.settings.Rates(ctx)
		if err != nil {
			return err
		}
		if s.displayRates != nil {
			rate, err := s.displayRates.DisplayRateOf(ctx, a.ID)
			if err != nil {
				return err
			}
			d = money.DisplayIn(rate.Code, rate.Rate)
		}
		d = d.WithRates(rates)
		return nil
	})
	return d, err
}

// shown returns the activity with its amounts also in a's display currency.
func (s *Service) shown(ctx context.Context, a actor.Actor, view View) (View, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return View{}, err
	}
	return displayView(d, view), nil
}

// shownSummaries returns the plan-tree rows with their amounts also in a's display currency.
func (s *Service) shownSummaries(ctx context.Context, a actor.Actor, list []Summary) ([]Summary, error) {
	d, err := s.displayFor(ctx, a)
	if err != nil {
		return nil, err
	}
	shown := make([]Summary, len(list))
	for i, row := range list {
		shown[i] = displaySummary(d, row)
	}
	return shown, nil
}

// displayView fills the activity's display amounts from its USD amounts.
func displayView(d money.Display, v View) View {
	v.Display = Display{
		PlannedCost:           d.Optional(v.PlannedCost),
		UnassignedPlannedCost: d.Optional(v.UnassignedPlannedCost),
		Revenue:               d.Optional(v.Goals.Revenue),
	}
	return v
}

// displaySummary fills a plan-tree row's display amounts from its USD amounts.
func displaySummary(d money.Display, row Summary) Summary {
	row.Display = Display{
		PlannedCost:           d.Optional(row.PlannedCost),
		UnassignedPlannedCost: d.Optional(row.UnassignedPlannedCost),
	}
	return row
}
