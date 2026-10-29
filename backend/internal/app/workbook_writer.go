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

// workbookWriter is the workbook.Writer of one import (038 FR-008): campaigns, initiatives and updates go
// through their single operations; activities and line items are checked by their areas' importers, which
// read each parent once and store a sheet's records together. Each parent's permission is checked once.
type workbookWriter struct {
	p          workbookPlanning
	plan       authz.PlanFacts
	activities *activity.Importer
	lineItems  *lineitem.Importer
	permitted  map[string]error
	newOwners  map[string]string // the owner of each activity this import creates, by id
	readable   map[string]string // the readable id of each provisional one stored since the last Flush
}

var _ workbook.Writer = (*workbookWriter)(nil)

// Prepare starts the writer of an import into the plan wp, which Plan has read for a.
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

// Update stores what is kept first, so that the single edit checks against it, and has the importers read
// their parents again after it.
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

// Flush stores the activities and line items kept so far and returns the readable ids given since the last
// Flush.
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
	if len(created) > 0 { // one read: the new activities have no line items yet
		if _, err = w.p.lineItemsRepo.ListByActivities(ctx, slices.Collect(maps.Values(created))); err != nil {
			return err
		}
	}
	items, err := w.lineItems.Flush(ctx)
	maps.Copy(w.readable, created)
	maps.Copy(w.readable, items)
	return err
}

// permit is the parent's permit, checked once; an activity this import creates is seen through its plan and
// owner, having no approvers yet.
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
	w.permitted[key] = err // any other error than a refusal stops the import anyway
	return err
}
