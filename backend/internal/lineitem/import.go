package lineitem

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// provisionalPrefix starts the id an imported line item has until Flush gives it its readable one.
const provisionalPrefix = "new-line-item-"

// Importer creates the line items of one plan workbook import with the checks of Create, reading each
// activity's facts and line items once, and stores them together on Flush (038 FR-008).
type Importer struct {
	s       *Service
	a       actor.Actor
	places  map[string]*importPlace
	pending []LineItem
}

type importPlace struct {
	facts    ActivityFacts
	siblings []LineItem
}

func (s *Service) NewImporter(a actor.Actor) *Importer {
	return &Importer{s: s, a: a, places: map[string]*importPlace{}}
}

// Defaults returns what a new line item of activityID starts with, as the form fills it in: the activity's
// owner when active, the plan's default spread and year, Request SWBS code on and the preselected finance
// budget item.
func (im *Importer) Defaults(ctx context.Context, activityID string) (Defaults, error) {
	p, err := im.place(ctx, activityID)
	if err != nil {
		return Defaults{}, err
	}
	d, err := im.s.defaults(ctx, p.facts, p.siblings, UnassignedBalance{})
	if err != nil {
		return Defaults{}, err
	}
	items, err := im.s.deps.FinanceBudgets.Items(ctx)
	if err != nil {
		return Defaults{}, err
	}
	d.FinanceBudgetItemID, d.FinanceBudgetItemPreselected = preselectedItem(p.facts, items)
	return d, nil
}

// Create checks in as Create does and keeps the line item for Flush; it returns a provisional id.
func (im *Importer) Create(ctx context.Context, activityID string, in Input) (string, error) {
	p, err := im.place(ctx, activityID)
	if err != nil {
		return "", err
	}
	li, problems, err := im.s.check(ctx, p.facts, p.siblings, in, nil)
	if err != nil {
		return "", err
	}
	if err = problems.Err(); err != nil {
		return "", err
	}
	li.ID = fmt.Sprintf("%s%d", provisionalPrefix, len(im.pending)+1)
	li.ActivityID, li.Meta = p.facts.ID, record.NewMeta(im.a, im.s.now())
	p.siblings = append(p.siblings, li)
	im.pending = append(im.pending, li)
	return li.ID, nil
}

// Flush stores the kept line items with their readable ids, logs each as created and returns the readable
// id of each provisional one.
func (im *Importer) Flush(ctx context.Context) (map[string]string, error) {
	if len(im.pending) == 0 {
		return map[string]string{}, nil
	}
	fresh, err := im.s.ids.NextIDs(ctx, ids.PrefixLineItem, len(im.pending))
	if err != nil {
		return nil, err
	}
	readable := make(map[string]string, len(fresh))
	for i := range im.pending {
		readable[im.pending[i].ID], im.pending[i].ID = fresh[i], fresh[i]
	}
	if err = im.s.repo.AddMany(ctx, im.pending); err != nil {
		return nil, err
	}
	if _, err = im.s.repo.ListByIDs(ctx, fresh); err != nil { // one read the log entries below find kept
		return nil, err
	}
	for _, li := range im.pending {
		if _, err = im.s.history.Log(ctx, im.a, history.LogInput{
			Kind: lineItemKind, RecordID: li.ID, RecordName: li.Name, Action: history.ActionCreated,
		}); err != nil {
			return nil, err
		}
	}
	for _, p := range im.places {
		for i := range p.siblings {
			if id, found := readable[p.siblings[i].ID]; found {
				p.siblings[i].ID = id
			}
		}
	}
	im.pending = nil
	return readable, nil
}

// Forget has the activities read again, after a change made beside the importer; nothing may be kept.
func (im *Importer) Forget() {
	clear(im.places)
}

func (im *Importer) place(ctx context.Context, activityID string) (*importPlace, error) {
	if p, found := im.places[activityID]; found {
		return p, nil
	}
	f, err := im.s.facts(ctx, activityID)
	if err != nil {
		return nil, err
	}
	p := &importPlace{facts: f}
	if p.siblings, err = im.s.repo.ListByActivity(ctx, activityID); err != nil {
		return nil, err
	}
	im.places[activityID] = p
	return p, nil
}
