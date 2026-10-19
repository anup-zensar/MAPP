package settings

import (
	"context"
	"slices"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// activityTypeGoneMessage answers a request about an activity type that no longer exists.
const activityTypeGoneMessage = "This activity type no longer exists — someone may have deleted it."

// ListActivityTypes returns every activity type, active and inactive, by name ignoring letter case, each
// with the models that allow it and its usage (FR-002, FR-003). Every signed-in person may see them
// (FR-008).
func (s *Service) ListActivityTypes(ctx context.Context, a actor.Actor) ([]ActivityTypeView, error) {
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
func (s *Service) GetActivityType(ctx context.Context, a actor.Actor, id string) (ActivityTypeView, error) {
	var view ActivityTypeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		t, found, err := s.repo.GetActivityType(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(activityTypeGoneMessage)
		}
		views, err := s.activityTypeViews(ctx, []ActivityType{t})
		if err != nil {
			return err
		}
		view = views[0]
		return nil
	})
	return view, err
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
