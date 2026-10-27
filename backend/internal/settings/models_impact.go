package settings

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// ModelImpact says, before a change of the model id is saved, what it would affect (FR-011, FR-012, D-044):
// changed rates or deal size recalculate the goals of the live plans using it; each activity type no longer
// allowed is kept by the live activities of those plans that use it. The body is checked as UpdateModel
// checks it, except the version; nothing is saved.
func (s *Service) ModelImpact(ctx context.Context, a actor.Actor, id string, update ModelUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		m, err := s.storedModel(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		changed, err := s.applyModel(ctx, m, update, &problems)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}
		refs, err := s.references(ctx)
		if err != nil {
			return err
		}
		plans, activities := modelInUse(refs, m.ID)
		if goalsChange(m, changed) && len(plans) > 0 {
			impact.Warnings = append(impact.Warnings, ImpactWarning{
				Kind:    ImpactGoalsRecalculated,
				Message: fmt.Sprintf(goalsRecalculatedMessage, plural(len(plans), "plan", "plans"), plural(activities, "activity", "activities")),
				Count:   len(plans),
			})
		}
		for _, typeID := range m.ActivityTypeIDs {
			if slices.Contains(changed.ActivityTypeIDs, typeID) {
				continue
			}
			n := liveActivitiesOfType(refs, typeID, plans)
			if n == 0 {
				continue
			}
			t, _, err := s.repo.GetActivityType(ctx, typeID)
			if err != nil {
				return err
			}
			message := fmt.Sprintf(typeRemovedMessage, plural(n, "activity", "activities"), t.Name)
			if n == 1 {
				message = fmt.Sprintf(typeRemovedOneMessage, t.Name)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactActivityTypeRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}

// goalsChange reports whether a change of a model changes the goals it calculates: its rates or its deal
// size (FR-011).
func goalsChange(before, after Model) bool {
	return before.InqToMqlRate != after.InqToMqlRate || before.MqlToDealRate != after.MqlToDealRate ||
		before.AverageDealSize != after.AverageDealSize
}

// modelInUse returns the live plans using the model modelID and the number of live activities in them.
// Impact statements count live records only: records in the recycle bin are recalculated when they come
// back (data model §10).
func modelInUse(refs []Reference, modelID string) (plans map[string]bool, activities int) {
	plans = map[string]bool{}
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelPlan && ref.ModelID == modelID {
			plans[ref.PlanID] = true
		}
	}
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelActivity && plans[ref.PlanID] {
			activities++
		}
	}
	return plans, activities
}

// liveActivitiesOfType counts the live activities of the activity type typeID in the plans.
func liveActivitiesOfType(refs []Reference, typeID string, plans map[string]bool) int {
	n := 0
	for _, ref := range refs {
		if !ref.InBin && ref.Level == LevelActivity && ref.ActivityTypeID == typeID && plans[ref.PlanID] {
			n++
		}
	}
	return n
}
