package plans

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"slices"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/authz"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
	"northstar/internal/settings"
)

// msgReload asks for a version that was not sent.
const msgReload = "Reload the page and try again."

// Update saves the whole plan form from the version the person started from (FR-018 – FR-023, D-078): the
// year never changes and the status stays, so a Published plan stays Published. Every problem is reported
// at once, a missing version among them; a save from an out-of-date version is refused naming who changed
// the plan and when. The budget of a plan that a global budget allocates to is read-only: a save that changes
// it is refused with 409 not_allowed (014 FR-031). A save that changes nothing creates no version and no log
// entry; otherwise one "changed" entry lists every changed field, each Shell attribute on its own.
func (s *Service) Update(ctx context.Context, a actor.Actor, id string, update PlanUpdate) (PlanView, error) {
	var view PlanView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		var problems errs.Problems
		changed, err := s.check(ctx, formOfUpdate(update), &stored, &problems)
		if err != nil {
			return err
		}
		checkTeams(ctx, authz.PlansEdit, stored.TeamIDs, AccessFacts(changed), &problems) // 014 FR-013
		if update.Version == nil || *update.Version < 1 {
			problems.Add("version", msgReload)
		}
		if changed.Currency != stored.Currency { // 016 FR-012: only while nothing below has an amount
			facts, err := s.facts(ctx, []string{stored.ID})
			if err != nil {
				return err
			}
			if lock := currencyLock(*facts[stored.ID]); lock != nil {
				problems.Add("currency", *lock)
			}
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if err := stored.CheckVersion(*update.Version, kindSingular); err != nil {
			return err
		}
		if err := s.refuseAllocatedBudget(ctx, stored, changed); err != nil { // 014 FR-031
			return err
		}
		changes, err := planChanges(stored, changed)
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			view, err = s.view(ctx, a, stored)
			return err
		}
		changed.Touch(a, s.now())
		if err := s.repo.Put(ctx, changed); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: kindName, RecordID: changed.ID, RecordName: changed.Name, Action: history.ActionChanged, Changes: changes,
		}); err != nil {
			return err
		}
		view, err = s.view(ctx, a, changed)
		return err
	})
	if err != nil {
		return PlanView{}, err
	}
	return view, nil
}

// planChanges lists what changed between two versions of a plan, for the activity log: every changed field
// as history.Diff finds it — the budget as money in the plan currency (016) — then each Shell attribute
// whose value changed as a field "attributes.<id>", sorted by attribute id (D-048), then the phasing's
// method and months when the phasing itself changed (016 FR-023).
func planChanges(before, after Plan) ([]history.Change, error) {
	withoutBefore, withoutAfter := before, after
	withoutBefore.Attributes, withoutAfter.Attributes = nil, nil
	withoutBefore.Budget, withoutAfter.Budget = nil, nil
	withoutBefore.BudgetPhasing, withoutAfter.BudgetPhasing = rollup.Phasing{}, rollup.Phasing{}
	changes, err := history.Diff(withoutBefore, withoutAfter)
	if err != nil {
		return nil, err
	}
	if oldBudget, newBudget := budgetValue(before), budgetValue(after); !bytes.Equal(oldBudget, newBudget) {
		changes = append(changes, history.Change{Field: "budget", Old: oldBudget, New: newBudget})
		slices.SortStableFunc(changes, func(x, y history.Change) int { return cmp.Compare(x.Field, y.Field) }) // as Diff orders them
	}
	ids := slices.Collect(maps.Keys(before.Attributes))
	for id := range after.Attributes {
		if _, inBefore := before.Attributes[id]; !inBefore {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		if !sameValue(before.Attributes[id], after.Attributes[id]) {
			changes = append(changes, history.Change{Field: "attributes." + id, Old: before.Attributes[id], New: after.Attributes[id]})
		}
	}
	if !reflect.DeepEqual(before.BudgetPhasing.Normalised(), after.BudgetPhasing.Normalised()) {
		changes = append(changes, rollup.PhasingChanges(before.BudgetPhasing, after.BudgetPhasing, before.Budget, after.Budget,
			after.Year, after.Currency)...)
	}
	return changes, nil
}

// budgetValue is the logged value of a plan's budget: money in the plan currency, or null (016).
func budgetValue(p Plan) json.RawMessage {
	if p.Budget == nil {
		return json.RawMessage("null")
	}
	encoded, _ := json.Marshal(money.Money{Amount: *p.Budget, Currency: p.Currency}) // always encodable
	return encoded
}

// sameValue reports whether two stored attribute values are the same: equal JSON, or both missing.
func sameValue(x, y json.RawMessage) bool {
	var compactX, compactY bytes.Buffer
	errX, errY := json.Compact(&compactX, x), json.Compact(&compactY, y)
	if errX != nil || errY != nil { // a missing value is no JSON
		return errX != nil && errY != nil
	}
	return bytes.Equal(compactX.Bytes(), compactY.Bytes())
}

// ChangeEffects says, before anything is saved, what a new model and new Shell attribute values would
// change on the plan id (D-077): the activities whose type the new model does not allow, which keep it until
// changed (FR-019, R7), and the attribute groups the new business unit would hide — their values are kept
// — or show again (FR-020, R9). It changes nothing.
func (s *Service) ChangeEffects(ctx context.Context, a actor.Actor, id string, request ChangeEffectsRequest) (ChangeEffects, error) {
	effects := ChangeEffects{
		DisallowedActivities: []errs.RecordRef{}, HiddenGroups: []AttributeGroupChange{}, ShownGroups: []AttributeGroupChange{},
	}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		p, err := s.stored(ctx, id)
		if err != nil {
			return err
		}
		if modelID := text(request.ModelID); modelID != "" && modelID != p.ModelID {
			if effects.DisallowedActivities, err = s.disallowedActivities(ctx, p.ID, modelID); err != nil {
				return err
			}
			effects.ModelChanged = true
		}
		before, err := s.settings.BusinessUnit(ctx, p.Attributes)
		if err != nil {
			return err
		}
		after, err := s.settings.BusinessUnit(ctx, request.Attributes)
		if err != nil {
			return err
		}
		if effects.HiddenGroups, err = s.groupChanges(ctx, before, after); err != nil {
			return err
		}
		effects.ShownGroups, err = s.groupChanges(ctx, after, before)
		return err
	})
	if err != nil {
		return ChangeEffects{}, err
	}
	return effects, nil
}

// disallowedActivities lists the activities of the plan planID whose type the model modelID does not allow,
// from every level that can tell (006); an unknown model is a problem of the field modelId.
func (s *Service) disallowedActivities(ctx context.Context, planID, modelID string) ([]errs.RecordRef, error) {
	m, found, err := s.settings.Model(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errs.Validation(errs.FieldError{Field: "modelId", Message: msgUnknownModel})
	}
	activities := []errs.RecordRef{}
	for _, l := range s.levels {
		if l.ActivitiesWithTypeNotAllowed == nil {
			continue
		}
		refs, err := l.ActivitiesWithTypeNotAllowed(ctx, planID, m.ActivityTypeIDs)
		if err != nil {
			return nil, err
		}
		activities = append(activities, refs...)
	}
	return activities, nil
}

// groupChanges returns the Shell attributes, at every level, that are shown with the business unit from and
// hidden with the business unit to, grouped by the business-unit option that is in from but not in to, in
// the options' display order; each group lists its levels top-down and its attributes' labels in display
// order. Inactive attributes are never shown, so they are left out. Never nil.
func (s *Service) groupChanges(ctx context.Context, from, to []string) ([]AttributeGroupChange, error) {
	type group struct {
		option   string // the business-unit option's id
		position int    // its display position
		change   AttributeGroupChange
	}
	var groups []*group
	for _, level := range settings.Levels {
		attributes, err := s.settings.Attributes(ctx, level)
		if err != nil {
			return nil, err
		}
		for _, attribute := range attributes {
			if !attribute.Active || !shownFor(attribute, from) || shownFor(attribute, to) {
				continue
			}
			// Shown with from but not with to: one of its condition's options is in from, not in to.
			unit := attribute.Condition[slices.IndexFunc(attribute.Condition, func(id string) bool { return slices.Contains(from, id) })]
			i := slices.IndexFunc(groups, func(g *group) bool { return g.option == unit })
			if i < 0 {
				o, found, err := s.settings.Option(ctx, unit)
				if err != nil {
					return nil, err
				}
				if !found {
					o.Label = unit // an option that no longer exists is shown by its id
				}
				groups = append(groups, &group{option: unit, position: o.Position,
					change: AttributeGroupChange{Unit: o.Label, Levels: []string{}, Attributes: []string{}}})
				i = len(groups) - 1
			}
			change := &groups[i].change
			if !slices.Contains(change.Levels, string(level)) {
				change.Levels = append(change.Levels, string(level))
			}
			change.Attributes = append(change.Attributes, attribute.Label)
		}
	}
	slices.SortStableFunc(groups, func(x, y *group) int { return cmp.Compare(x.position, y.position) })
	changes := []AttributeGroupChange{}
	for _, g := range groups {
		changes = append(changes, g.change)
	}
	return changes, nil
}

// shownFor reports whether a Shell attribute is shown for a business unit: it has no condition, or the
// business unit includes one of its options (002 FR-023, the rule of settings.Visible).
func shownFor(attribute settings.Attribute, businessUnit []string) bool {
	return len(attribute.Condition) == 0 ||
		slices.ContainsFunc(attribute.Condition, func(id string) bool { return slices.Contains(businessUnit, id) })
}
