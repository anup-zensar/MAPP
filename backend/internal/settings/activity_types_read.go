package settings

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ListActivityTypes returns every activity type, active and inactive, by name ignoring letter case, each
// with the models that allow it and its usage (FR-002, FR-003). Every signed-in person may see them
// (FR-008).
func (s *Service) ListActivityTypes(ctx context.Context, _ actor.Actor) ([]ActivityTypeView, error) {
	var views []ActivityTypeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		types, err := s.repo.ListActivityTypes(ctx)
		if err != nil {
			return err
		}
		slices.SortStableFunc(types, func(x, y ActivityType) int { return compareNames(x.Name, y.Name) })
		views, err = s.activityTypeViews(ctx, types)
		return err
	})
	return views, err
}

// GetActivityType returns the activity type id with its models and usage (US5-2).
func (s *Service) GetActivityType(ctx context.Context, _ actor.Actor, id string) (ActivityTypeView, error) {
	var view ActivityTypeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		t, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.activityTypeView(ctx, t)
		return err
	})
	return view, err
}

// storedActivityType reads the activity type id, or answers not found.
func (s *Service) storedActivityType(ctx context.Context, id string) (ActivityType, error) {
	t, found, err := s.repo.GetActivityType(ctx, id)
	if err != nil {
		return ActivityType{}, err
	}
	if !found {
		return ActivityType{}, errs.NotFound(activityTypeGoneMessage)
	}
	return t, nil
}

// activityTypeView returns the activity type t as the API answers it.
func (s *Service) activityTypeView(ctx context.Context, t ActivityType) (ActivityTypeView, error) {
	views, err := s.activityTypeViews(ctx, []ActivityType{t})
	if err != nil {
		return ActivityTypeView{}, err
	}
	return views[0], nil
}

// activityTypeViews returns activity types as the API answers them: each with the models allowing it, by
// name, and its usage.
func (s *Service) activityTypeViews(ctx context.Context, types []ActivityType) ([]ActivityTypeView, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(models, func(x, y Model) int { return compareNames(x.Name, y.Name) })
	u, err := s.usage(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]ActivityTypeView, 0, len(types))
	for _, t := range types {
		allowedIn := []SettingRef{}
		for _, m := range models {
			if m.Allows(t.ID) {
				allowedIn = append(allowedIn, SettingRef{ID: m.ID, Name: m.Name, Active: m.Active})
			}
		}
		views = append(views, ActivityTypeView{
			ID: t.ID, Name: t.Name, DefaultPlannedCost: t.DefaultPlannedCost, Models: allowedIn, Active: t.Active,
			Usage: u.activityType(t.ID), Meta: t.Meta,
		})
	}
	return views, nil
}
