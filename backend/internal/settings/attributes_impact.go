package settings

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// AttributeImpact says, before a change of the Shell attribute id is saved, what it would affect (FR-021,
// US2-5, D-044): a marking that becomes needed makes the live records of its level without a value
// incomplete — counted as MissingAttributes counts, so records where it is hidden, or a dependent without
// options for the chosen parent (D-034), are not; a type without options removes its options. The body is
// checked as UpdateAttribute checks it, except the version; nothing is saved.
func (s *Service) AttributeImpact(ctx context.Context, a actor.Actor, id string, update ShellAttributeUpdate) (ChangeImpact, error) {
	impact := ChangeImpact{Warnings: []ImpactWarning{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, stored, u, err := s.attributeToChange(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		change := f.planAttributeChange(stored, update, u, &problems)
		if err := problems.Err(); err != nil {
			return err
		}
		changed := change.attribute
		if n := f.becomeIncomplete(stored, changed, u.refs); n > 0 {
			nouns, verb, keep := levelNouns[changed.Level], "have", plansKeepStatus
			if n == 1 {
				verb = "has"
			}
			if !publishes(changed.Level) {
				keep = activitiesKeepStatus
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{
				Kind: ImpactBecomesIncomplete, Message: fmt.Sprintf(incompleteMessage, plural(n, nouns[0], nouns[1]), verb, keep), Count: n,
			})
		}
		if n := len(change.removed); n > 0 {
			message := fmt.Sprintf(optionRemovedMessage, change.removed[0].Label)
			if n > 1 {
				labels := []string{}
				for _, o := range change.removed[:min(n, maxListedLabels)] {
					labels = append(labels, o.Label)
				}
				listed := strings.Join(labels, ", ")
				if n > maxListedLabels {
					listed += fmt.Sprintf(" and %d more", n-maxListedLabels)
				}
				message = fmt.Sprintf(optionsRemovedMessage, n, listed)
			}
			impact.Warnings = append(impact.Warnings, ImpactWarning{Kind: ImpactOptionsRemoved, Message: message, Count: n})
		}
		return nil
	})
	return impact, err
}

// becomeIncomplete counts the live records of the attribute's level that changing before into after makes
// incomplete (FR-021): when its marking goes from optional to needed, the records that MissingAttributes
// would list it for — not those where it is hidden by its condition, nor a dependent without options for
// the chosen parent (D-034).
func (f formData) becomeIncomplete(before, after Attribute, refs []Reference) int {
	if before.Marking != MarkingOptional || after.Marking == MarkingOptional || !after.Active {
		return 0
	}
	form := f
	form.level = after.Level
	form.attributes = displayOrder(append(filter(f.levelAttributes(after.Level), func(a Attribute) bool { return a.ID != after.ID }), after))
	planValues := map[string]AttributeValues{}
	for _, ref := range refs {
		if ref.Level == LevelPlan {
			planValues[ref.PlanID] = ref.Attributes
		}
	}
	field, n := "attributes."+after.ID, 0
	for _, ref := range refs {
		if ref.InBin || ref.Level != after.Level {
			continue
		}
		for _, m := range form.missing(ref.Attributes, f.businessUnitOf(planValues[ref.PlanID])) {
			if m.Field == field {
				n++
				break
			}
		}
	}
	return n
}
