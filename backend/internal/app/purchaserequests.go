package app

import (
	"mapp/internal/history"
	"mapp/internal/purchaserequest"
)

// purchaseRequestsArea builds purchase requests (032) after the approval engine, which it plugs into both ways,
// and the simulated GSAP that answers routed purchase requests (033); the guards keep what they draw on.
func (w *wiring) purchaseRequestsArea() error {
	s := w.set
	tools := history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	prs := purchaserequest.NewService(s.PurchaseRequests, tools, purchaserequest.Deps{
		LineItems: prLineItems{set: s, settings: w.settings}, People: prPeople{repo: s.People, service: w.people},
		Currencies: w.settings, Attributes: prAttributes{settings: w.settings, repo: s.Settings},
		Workflows: prWorkflows{repo: s.Approvals}, Invoices: prInvoices{repo: s.Financials, rates: w.settings.Rates},
	})
	if err := w.register(prs.Kinds()); err != nil {
		return err
	}
	prs.UseApprovals(w.approvals)
	w.approvals.UsePurchaseRequests(prs)
	prs.UseGSAP(w.simulatedOrders())
	w.guardPurchaseRequests(prs)
	w.purchases = prs
	w.settings.AddReferenceSource(purchaseRequestReferences(s.PurchaseRequests, w.history))
	mount(w.signedIn, purchaserequest.NewHandler(prs), purchaserequest.NewApprovalHandler(prs), w.simulatedOrdersHandler(prs))
	return nil
}
