package app

import (
	"context"
	"errors"
	"time"

	"mapp/internal/financials"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/purchaserequest"
	"mapp/internal/spa"
)

// spaArea wires the SP&A view (013): a read-only view over the plan reader of All work, 015's figures and 032's
// purchase requests.
func (w *wiring) spaArea() {
	s := w.set
	service := spa.NewService(s.Tx, spa.Deps{
		Planning:  spaPlanning{plans: s.Plans, rates: w.settings.Rates, source: w.allWork, now: w.now},
		Finance:   spaFinance{financials: w.financials, access: financialAccess{tx: s.Tx, plans: s.Plans}},
		Purchases: spaPurchases{prs: w.purchases},
		Displays:  financialDisplays{people: w.people},
	})
	mount(w.signedIn, spa.NewHandler(service))
}

type spaFinance struct {
	financials *financials.Service
	access     financials.Access
}

// SeesFinance turns 015's refusal into "no": the view then shows planned figures only.
func (f spaFinance) SeesFinance(ctx context.Context, a actor.Actor, planID string) (bool, error) {
	err := f.access.SeeFinanceIn(ctx, a, planID)
	if _, refused := errors.AsType[*errs.Error](err); refused {
		return false, nil
	}
	return err == nil, err
}

func (f spaFinance) SpendByMonth(ctx context.Context) (map[string]map[string]spa.Spend, error) {
	facts, err := f.financials.FactsByLineItemMonth(ctx)
	spend := make(map[string]map[string]spa.Spend, len(facts))
	for id, months := range facts {
		spend[id] = make(map[string]spa.Spend, len(months))
		for month, m := range months {
			spend[id][month] = spa.Spend{Committed: m.Committed, Invoiced: m.Invoiced, Actual: m.Paid}
		}
	}
	return spend, err
}

func (f spaFinance) DataAsOf(ctx context.Context) (*time.Time, error) {
	return f.financials.DataAsOf(ctx)
}

type spaPurchases struct {
	prs *purchaserequest.Service
}

func (p spaPurchases) Drawn(ctx context.Context, lineItemIDs []string) ([]spa.PurchaseRequest, error) {
	drawn, err := p.prs.DrawnOnLineItems(ctx, lineItemIDs)
	list := make([]spa.PurchaseRequest, 0, len(drawn))
	for _, d := range drawn {
		pr := spa.PurchaseRequest{ID: d.ID, Name: d.Name, Vendor: d.Vendor, Status: string(d.Status), Lines: []spa.PurchaseLine{}}
		for _, l := range d.Lines {
			pr.Lines = append(pr.Lines, spa.PurchaseLine{LineItemID: l.LineItemID, USD: l.USD})
		}
		list = append(list, pr)
	}
	return list, err
}

func (p spaPurchases) CanRaise(ctx context.Context, pl spa.Plan) (authz.Availability, error) {
	return p.prs.RaiseAvailability(ctx, pl.Access)
}

// spaPlanOf returns the plan with its budget in USD at the current rate of its currency (016 FR-014).
func spaPlanOf(p plan.Plan, rates money.Rates) spa.Plan {
	view := spa.Plan{ID: p.ID, Name: p.Name, Year: p.Year, Status: string(p.Status), BCPIR: p.BCPIR, Access: plan.AccessFacts(p)}
	if p.Budget != nil {
		budget, err := rates.ToUSD(money.Money{Amount: *p.Budget, Currency: p.Normalised().Currency})
		if err != nil {
			budget = *p.Budget
		}
		view.Budget = &budget
	}
	return view
}
