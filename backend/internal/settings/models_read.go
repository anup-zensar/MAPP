package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ListModels returns every model, active and inactive, by name ignoring letter case, each with its
// allowed activity types and its usage (FR-002, FR-003). Every signed-in person may see them (FR-008).
func (s *Service) ListModels(ctx context.Context, a actor.Actor) ([]ModelView, error) {
	var views []ModelView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		models, err := s.repo.ListModels(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(models, func(x, y Model) int { return compareNames(x.Name, y.Name) })
		views, err = s.modelViews(ctx, models)
		return err
	})
	return views, err
}

// GetModel returns the model id with its usage and the live plans using it (US1-3).
func (s *Service) GetModel(ctx context.Context, a actor.Actor, id string) (ModelView, error) {
	var view ModelView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.modelView(ctx, m)
		return err
	})
	return view, err
}

// storedModel reads the model id, or answers not found.
func (s *Service) storedModel(ctx context.Context, id string) (Model, error) {
	m, found, err := s.repo.GetModel(ctx, id)
	if err != nil {
		return Model{}, err
	}
	if !found {
		return Model{}, errs.NotFound(modelGoneMessage)
	}
	return m, nil
}

// modelView returns the model m as the API answers it.
func (s *Service) modelView(ctx context.Context, m Model) (ModelView, error) {
	views, err := s.modelViews(ctx, []Model{m})
	if err != nil {
		return ModelView{}, err
	}
	return views[0], nil
}

// modelViews returns models as the API answers them, with their allowed activity types by name and their
// usage, read once for all of them.
func (s *Service) modelViews(ctx context.Context, models []Model) ([]ModelView, error) {
	types, err := s.repo.ListActivityTypes(ctx)
	if err != nil {
		return nil, err
	}
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	typesByID := map[string]ActivityType{}
	for _, t := range types {
		typesByID[t.ID] = t
	}
	views := make([]ModelView, 0, len(models))
	for _, m := range models {
		refs := []SettingRef{}
		for _, id := range m.ActivityTypeIDs {
			if t, found := typesByID[id]; found {
				refs = append(refs, SettingRef{ID: t.ID, Name: t.Name, Active: t.Active})
			}
		}
		slices.SortStableFunc(refs, func(x, y SettingRef) int { return compareNames(x.Name, y.Name) })
		views = append(views, ModelView{
			ID: m.ID, Name: m.Name, InqToMqlRate: m.InqToMqlRate, MqlToDealRate: m.MqlToDealRate,
			AverageDealSize: m.AverageDealSize, ActivityTypes: refs, Active: m.Active, Usage: u.model(m.ID), Meta: m.Meta,
		})
	}
	return views, nil
}
