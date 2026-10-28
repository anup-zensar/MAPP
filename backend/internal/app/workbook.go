package app

import (
	"mapp/internal/history"
	"mapp/internal/workbook"
)

// workbookArea builds feature 012's plan workbook after the planning areas it creates and updates through.
func (w *wiring) workbookArea() error {
	s := w.set
	planning := workbookPlanning{
		planningEdits: w.planningEdits(), plans: s.Plans, people: s.People, settings: w.settings,
		workflows: activityWorkflows{repo: s.Approvals},
	}
	tools := history.Tools{Tx: s.Tx, IDs: w.generator, Log: w.history, Now: w.now}
	mount(w.signedIn, workbook.NewHandler(workbook.NewService(tools, s.Workbook, planning)))
	return nil
}
