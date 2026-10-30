package activity

import (
	"context"
	"fmt"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

const provisionalPrefix = "new-activity-"

// Importer creates the activities of one workbook import with the rules of Create and stores them on Flush.
type Importer struct {
	s       *Service
	a       actor.Actor
	places  map[string]*importPlace
	pending []Activity
}

type importPlace struct {
	pc       planContext
	siblings []Activity
	defaults ImportDefaults
}

// ImportDefaults are what a new activity of an initiative starts with, as the new form fills it in.
type ImportDefaults struct {
	Defaults
	TypeCosts map[string]money.Money
}

func (s *Service) NewImporter(a actor.Actor) *Importer {
	return &Importer{s: s, a: a, places: map[string]*importPlace{}}
}

// Defaults returns the defaults of a new activity under initiativeID.
func (im *Importer) Defaults(ctx context.Context, initiativeID string) (ImportDefaults, error) {
	p, err := im.place(ctx, initiativeID)
	if err != nil {
		return ImportDefaults{}, err
	}
	return p.defaults, nil
}

// Create checks in as Create does and keeps the activity for Flush; it returns a provisional id.
func (im *Importer) Create(ctx context.Context, initiativeID string, in Input) (string, error) {
	p, err := im.place(ctx, initiativeID)
	if err != nil {
		return "", err
	}
	checker := *im.s
	checker.repo = placeSiblings{Store: im.s.repo, place: p}
	var problems errs.Problems
	activity, err := checker.checkFields(ctx, &problems, fieldCheck{pc: p.pc, in: in})
	if err != nil {
		return "", err
	}
	if err = problems.Err(); err != nil {
		return "", err
	}
	activity.ID = fmt.Sprintf("%s%d", provisionalPrefix, len(im.pending)+1)
	activity.Status, activity.Meta = StatusCreated, record.NewMeta(im.a, im.s.now())
	p.siblings = append(p.siblings, activity)
	im.pending = append(im.pending, activity)
	return activity.ID, nil
}

// Flush stores the kept activities under readable ids, logs each and maps every provisional id to its own.
func (im *Importer) Flush(ctx context.Context) (map[string]string, error) {
	if len(im.pending) == 0 {
		return map[string]string{}, nil
	}
	fresh, err := im.s.ids.NextIDs(ctx, ids.PrefixActivity, len(im.pending))
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
	if err = im.logCreated(ctx); err != nil {
		return nil, err
	}
	im.renameSiblings(readable)
	im.pending = nil
	return readable, nil
}

func (im *Importer) logCreated(ctx context.Context) error {
	for _, activity := range im.pending {
		if _, err := im.s.history.Log(ctx, im.a, history.LogInput{
			Kind: kindName, RecordID: activity.ID, RecordName: activity.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (im *Importer) renameSiblings(readable map[string]string) {
	for _, p := range im.places {
		for i := range p.siblings {
			if id, found := readable[p.siblings[i].ID]; found {
				p.siblings[i].ID = id
			}
		}
	}
}

// Forget has the initiatives read again, after a change made beside the importer; nothing may be kept.
func (im *Importer) Forget() {
	clear(im.places)
}

func (im *Importer) place(ctx context.Context, initiativeID string) (*importPlace, error) {
	if p, found := im.places[initiativeID]; found {
		return p, nil
	}
	pc, found, err := im.s.contextOf(ctx, initiativeID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errs.NotFound(initiativeGoneMessage)
	}
	p := &importPlace{pc: pc}
	if p.siblings, err = im.s.repo.ListByInitiative(ctx, initiativeID); err != nil {
		return nil, err
	}
	if p.defaults, err = im.s.importDefaults(ctx, pc); err != nil {
		return nil, err
	}
	im.places[initiativeID] = p
	return p, nil
}

func (s *Service) importDefaults(ctx context.Context, pc planContext) (ImportDefaults, error) {
	active, err := s.activeWorkflows(ctx)
	if err != nil {
		return ImportDefaults{}, err
	}
	defaults, err := s.defaults(ctx, pc, active)
	if err != nil {
		return ImportDefaults{}, err
	}
	costs, err := s.typeCosts(ctx, pc)
	if err != nil {
		return ImportDefaults{}, err
	}
	return ImportDefaults{Defaults: *defaults, TypeCosts: costs}, nil
}

func (s *Service) activeWorkflows(ctx context.Context) ([]WorkflowOption, error) {
	workflows, err := s.workflows.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	var active []WorkflowOption
	for _, w := range workflows {
		if w.Active {
			active = append(active, WorkflowOption{ID: w.ID})
		}
	}
	return active, nil
}

func (s *Service) typeCosts(ctx context.Context, pc planContext) (map[string]money.Money, error) {
	types, err := s.settings.ActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := s.settings.Rates(ctx)
	if err != nil {
		return nil, err
	}
	costs := map[string]money.Money{}
	for _, t := range types {
		if cost := inPlanCurrency(t.DefaultPlannedCost, pc.plan.Currency, rates); t.Active && pc.model.Allows(t.ID) && cost != nil {
			costs[t.ID] = *cost
		}
	}
	return costs, nil
}

type placeSiblings struct {
	Store
	place *importPlace
}

func (p placeSiblings) ListByInitiative(context.Context, string) ([]Activity, error) {
	return p.place.siblings, nil
}
