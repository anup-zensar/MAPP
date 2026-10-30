package app

import (
	"context"
	"maps"
	"slices"

	"mapp/internal/access"
	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/workbook"
)

// workbookWriter imports activities and line items through their areas' importers, which read each parent once
// and store a sheet together; each parent's permission is checked once.
type workbookWriter struct {
	p          workbookPlanning
	plan       authz.PlanFacts
	activities *activity.Importer
	lineItems  *lineitem.Importer
	permitted  map[string]error
	newOwners  map[string]string
	readable   map[string]string
}

var _ workbook.Writer = (*workbookWriter)(nil)

func (p workbookPlanning) Prepare(ctx context.Context, a actor.Actor, wp workbook.Plan) (workbook.Writer, error) {
	stored, found, err := p.plans.Get(ctx, wp.ID)
	if err != nil || !found {
		return nil, readProblem(err, found)
	}
	return &workbookWriter{
		p: p, plan: plan.AccessFacts(stored), activities: p.activities.NewImporter(a), lineItems: p.lineItems.NewImporter(a),
		permitted: map[string]error{}, newOwners: map[string]string{}, readable: map[string]string{},
	}, nil
}

// Update stores what is kept first, so the single edit checks against it; the importers then re-read parents.
func (w *workbookWriter) Update(ctx context.Context, a actor.Actor, c workbook.Change) error {
	if err := w.store(ctx); err != nil {
		return err
	}
	err := w.p.Update(ctx, a, c)
	w.activities.Forget()
	w.lineItems.Forget()
	clear(w.permitted)
	return err
}

// Flush stores the activities and line items kept so far and returns the readable ids given since the last one.
func (w *workbookWriter) Flush(ctx context.Context) (map[string]string, error) {
	if err := w.store(ctx); err != nil {
		return nil, err
	}
	readable := w.readable
	w.readable = map[string]string{}
	return readable, nil
}

func (w *workbookWriter) store(ctx context.Context) error {
	created, err := w.activities.Flush(ctx)
	if err != nil {
		return err
	}
	for provisional, id := range created {
		w.newOwners[id] = w.newOwners[provisional]
		delete(w.newOwners, provisional)
	}
	if len(created) > 0 {
		if _, err = w.p.lineItemsRepo.ListByActivities(ctx, slices.Collect(maps.Values(created))); err != nil {
			return err
		}
	}
	items, err := w.lineItems.Flush(ctx)
	maps.Copy(w.readable, created)
	maps.Copy(w.readable, items)
	return err
}

// permit is checked once per parent; an activity this import creates has no approvers yet.
func (w *workbookWriter) permit(ctx context.Context, parentID string, p authz.Permission) error {
	key := parentID + " " + string(p)
	if err, done := w.permitted[key]; done {
		return err
	}
	var err error
	if owner, isNew := w.newOwners[parentID]; isNew {
		err = authz.RequireViewActivity(ctx, authz.ActivityFacts{Plan: w.plan, OwnerID: owner}, access.RecordActivity)
		if err == nil {
			err = authz.RequireIn(ctx, p, w.plan)
		}
	} else {
		err = w.p.permit(ctx, parentID, p)
	}
	w.permitted[key] = err
	return err
}
