package app

import (
	"mapp/internal/bulk"
	"mapp/internal/history"
)

// bulkArea builds feature 012's bulk actions after the planning areas and approvals they call.
func (w *wiring) bulkArea() error {
	s := w.set
	planning := bulkPlanning{
		planningEdits: w.planningEdits(), plans: s.Plans, people: s.People, settings: w.settings,
		workflows: activityWorkflows{repo: s.Approvals}, approvals: w.approvals,
	}
	tools := history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	mount(w.signedIn, bulk.NewHandler(bulk.NewService(tools, s.Bulk, planning)))
	return nil
}

func (w *wiring) planningEdits() planningEdits {
	return planningEdits{
		campaignsRepo: w.set.Campaigns, activitiesRepo: w.set.Activities, lineItemsRepo: w.set.LineItems,
		campaigns: w.campaigns, activities: w.activities, lineItems: w.lineItems,
		targets: planningTargets{db: w.set.Tx, plans: w.set.Plans, campaigns: w.set.Campaigns, activities: w.set.Activities,
			lineItems: w.set.LineItems, approvals: w.set.Approvals},
	}
}
