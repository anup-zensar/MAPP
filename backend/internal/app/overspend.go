package app

import (
	"mapp/internal/history"
	"mapp/internal/overspend"
)

// overspendArea builds Settings › Overspend control after Settings, whose exchange rates the guard needs for
// global budgets (feature 037); the planning areas built after it read its default and limits.
func (w *wiring) overspendArea() error {
	w.overspend = overspend.NewService(history.Tools{Tx: w.set.Tx, IDs: w.generator, Log: w.history, Now: w.now},
		w.set.Overspend)
	w.overspend.UseRates(w.settings)
	w.guard.UseRates(w.settings)
	if err := w.register(w.overspend.Kinds()); err != nil {
		return err
	}
	mount(w.signedIn, overspend.NewHandler(w.overspend))
	return nil
}

func (w *wiring) useOverspend() {
	w.plans.UseOverspendDefaults(w.overspend)
	w.campaigns.UseOverspendDefaults(w.overspend)
	w.activities.UseOverspend(w.overspend)
	w.approvals.UseApprovalOverspend(w.overspend)
}
