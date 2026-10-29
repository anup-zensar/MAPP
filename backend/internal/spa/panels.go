package spa

import "mapp/internal/platform/money"

const msgNoBudget = "No budget set"

// The purchase-request statuses (032) as the procurement panel groups them (D-335).
const (
	statusCreated   = "created"
	statusSubmitted = "submitted"
	statusInReview  = "inReview"
	statusApproved  = "approved"
	statusClosed    = "closed"
	statusDeclined  = "declined"
)

// budgetPanelOf is FR-020 for the scope's budget (nil for none).
func budgetPanelOf(acts []*activity, budget *money.Amount, sees bool, s shower) BudgetPanel {
	planned := sumOf(acts, plannedOf)
	var linked money.Amount
	for _, a := range acts {
		for _, li := range a.lineItems {
			if li.rec.Linked {
				linked = linked.Add(li.planned)
			}
		}
	}
	panel := BudgetPanel{
		PlannedCost: s.money(planned), Committed: s.financial(sees, sumOf(acts, committedOf)),
		Linked: s.money(linked), NotLinked: s.money(planned.Sub(linked)),
	}
	if budget == nil || budget.IsZero() {
		panel.NoBudgetReason = text(msgNoBudget)
		return panel
	}
	shown := s.money(*budget)
	panel.Budget = &shown
	panel.PlannedVsBudgetPercent = percentOf(planned.Sub(*budget), *budget, percentsDecimals)
	panel.OverBudget = planned.Cmp(*budget) > 0
	return panel
}

// procurementOf is FR-021: the purchase requests on the scope's line items, counting only those lines.
func procurementOf(acts []*activity, prs []PurchaseRequest, sees bool, s shower) ProcurementPanel {
	var approved, pending, draft money.Amount
	items := make([]PurchaseRequestRow, 0, len(prs))
	for _, pr := range prs {
		var amount money.Amount
		for _, l := range pr.Lines {
			amount = amount.Add(l.USD)
		}
		switch pr.Status {
		case statusApproved, statusClosed:
			approved = approved.Add(amount)
		case statusSubmitted, statusInReview:
			pending = pending.Add(amount)
		case statusCreated:
			draft = draft.Add(amount)
		}
		items = append(items, PurchaseRequestRow{ID: pr.ID, Name: pr.Name, Vendor: pr.Vendor, Status: pr.Status,
			Amount: s.money(amount), Counted: pr.Status != statusDeclined})
	}
	return ProcurementPanel{
		Count: len(items), Approved: s.money(approved), PendingApproval: s.money(pending), Draft: s.money(draft),
		Invoiced: s.financial(sees, sumOf(acts, invoicedOf)), Items: items,
	}
}

func lineItemIDsOf(acts []*activity) []string {
	ids := []string{}
	for _, a := range acts {
		for _, li := range a.lineItems {
			ids = append(ids, li.rec.ID)
		}
	}
	return ids
}
