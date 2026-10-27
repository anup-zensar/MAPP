package settings

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ActivityTypeImpact says, before a change of the activity type id is saved, what it would affect (FR-012,
// D-044): for each model that would no longer allow it, the live activities of the type in that model's
// live plans, which keep their type. The body is checked as UpdateActivityType checks it, except the
// version; nothing is saved.
func (s *Service) ActivityTypeImpact(ctx context.Context, _ actor.Actor, id string, update ActivityTypeUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		stored, err := s.storedActivityType(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		_, allowedIn, err := s.applyActivityType(ctx, stored, update, &problems)
		if err != nil {
			return err
		}
		if err = problems.Err(); err != nil {
			return err
		}
		refs, err := s.references(ctx)
		if err != nil {
			return err
		}
		for _, change := range allowedIn {
			if change.allow {
				continue
			}
			plans, _ := modelInUse(refs, change.model.ID)
			n := liveActivitiesOfType(refs, stored.ID, plans)
			if n == 0 {
				continue
			}
			message := fmt.Sprintf(removedFromModelMessage, change.model.Name, plural(n, "activity", "activities"), stored.Name)
			if n == 1 {
				message = fmt.Sprintf(removedFromModelOneMessage, change.model.Name, stored.Name)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactActivityTypeRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}
