package settings

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// Messages of the plan's "User-visible messages" table about moves.
const (
	firstInSectionMessage    = "'%s' is already first in its section."
	lastInSectionMessage     = "'%s' is already last in its section."
	firstSectionMessage      = "This section is already first."
	lastSectionMessage       = "This section is already last."
	businessUnitFirstMessage = "Business unit stays first on the plan form, because other attributes are shown or hidden by it."
)

// MoveAttribute moves the Shell attribute id, or its whole section, one place up or down on its level's
// form (FR-019, US2-6, data model §4.2), from the version of the attribute the person started at (FR-007).
// An attribute swaps places with its neighbour in its section; a section swaps with the previous or next
// section and the level is numbered 1…n again. A move past the edge, one that puts a dependent above its
// parent, and one that takes the Business unit attribute from the top of the plan form are refused on
// direction, and nothing changes. Each attribute whose place changed gets one "changed" entry (field
// position). It answers the level's attributes in their new order.
func (s *Service) MoveAttribute(ctx context.Context, a actor.Actor, id string, in MoveInput) ([]ShellAttributeView, error) {
	var problems errs.Problems
	if in.What != MoveAttribute && in.What != MoveSection {
		problems.Add("what", reloadMessage)
	}
	if in.Direction != MoveUp && in.Direction != MoveDown {
		problems.Add("direction", reloadMessage)
	}
	checkVersion(&problems, in.Version)
	if err := problems.Err(); err != nil {
		return nil, err
	}
	var views []ShellAttributeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, "")
		if err != nil {
			return err
		}
		stored, found := f.byID[id]
		if !found {
			return errs.NotFound(attributeGoneMessage)
		}
		if err := stored.CheckVersion(*in.Version, attributeNoun); err != nil {
			return err
		}
		level := f.levelAttributes(stored.Level)
		order, refusal := reordered(level, stored, in)
		if refusal == "" {
			refusal = orderRefusal(order)
		}
		if refusal != "" {
			return errs.Validation(errs.FieldError{Field: "direction", Message: refusal})
		}
		for _, changed := range repositioned(level, order, in.What == MoveSection) {
			change := history.Change{Field: "position", Old: jsonOf(f.byID[changed.ID].Position), New: jsonOf(changed.Position)}
			changed.Touch(a, s.now())
			if err := s.repo.PutAttribute(ctx, changed); err != nil {
				return err
			}
			if err := s.logChanged(ctx, a, attributeKind, changed.ID, changed.Label, []history.Change{change}, ""); err != nil {
				return err
			}
		}
		views, err = s.attributeList(ctx, &stored.Level)
		return err
	})
	return views, err
}

// reordered returns the attributes of level — in display order — in the order the move in of the attribute a
// gives, or the reason it cannot move: the Business unit attribute never moves, and nothing moves past the
// edge of its section or of the form.
func reordered(level []Attribute, a Attribute, in MoveInput) ([]Attribute, string) {
	step := -1
	if in.Direction == MoveDown {
		step = 1
	}
	order := slices.Clone(level)
	if in.What == MoveAttribute {
		if a.Role == RoleBusinessUnit {
			return nil, businessUnitLock
		}
		i := slices.IndexFunc(order, func(x Attribute) bool { return x.ID == a.ID })
		j := i + step
		if j < 0 || j >= len(order) || order[j].Section != a.Section {
			if step < 0 {
				return nil, fmt.Sprintf(firstInSectionMessage, a.Label)
			}
			return nil, fmt.Sprintf(lastInSectionMessage, a.Label)
		}
		order[i], order[j] = order[j], order[i]
		return order, ""
	}
	var sections []string
	for _, x := range level {
		if !slices.Contains(sections, x.Section) {
			sections = append(sections, x.Section)
		}
	}
	k := slices.Index(sections, a.Section)
	t := k + step
	switch {
	case t < 0:
		return nil, firstSectionMessage
	case t >= len(sections):
		return nil, lastSectionMessage
	}
	sections[k], sections[t] = sections[t], sections[k]
	order = order[:0]
	for _, section := range sections {
		order = append(order, filter(level, func(x Attribute) bool { return x.Section == section })...)
	}
	return order, ""
}

// orderRefusal returns why a level's attributes cannot have the order given, or "": the Business unit
// attribute comes first on the plan form, and every dependent comes after its parent (data model §4.2).
func orderRefusal(order []Attribute) string {
	for i, a := range order {
		if i > 0 && a.Role == RoleBusinessUnit {
			return businessUnitFirstMessage
		}
		if a.ParentAttributeID == "" {
			continue
		}
		if p := slices.IndexFunc(order, func(x Attribute) bool { return x.ID == a.ParentAttributeID }); p > i {
			return fmt.Sprintf(stayBelowMessage, a.Label, order[p].Label)
		}
	}
	return ""
}

// repositioned gives the attributes of order — the attributes of level, which is in display order, in a
// new order — the positions that make that order, and returns those whose position changes: a move inside
// a section swaps two positions; a section move, or two attributes on the same position, numbers the level
// 1…n.
func repositioned(level, order []Attribute, renumber bool) []Attribute {
	next := slices.Clone(order)
	for k := range next {
		if renumber {
			next[k].Position = k + 1
		} else {
			next[k].Position = level[k].Position
		}
	}
	if !renumber && !slices.EqualFunc(displayOrder(next), next, func(x, y Attribute) bool { return x.ID == y.ID }) {
		return repositioned(level, order, true)
	}
	changed := []Attribute{}
	for k, x := range next {
		if i := slices.IndexFunc(level, func(y Attribute) bool { return y.ID == x.ID }); level[i].Position != next[k].Position {
			changed = append(changed, x)
		}
	}
	return changed
}

// displayOrder returns the attributes of one level in display order (data model §4.2, FR-019): sections
// in the order of their lowest position, and inside a section by position, then by id. Forms, the
// preview and the Settings list all use it.
func displayOrder(attributes []Attribute) []Attribute {
	first := map[string]int{} // section → its lowest position
	for _, a := range attributes {
		if p, seen := first[a.Section]; !seen || a.Position < p {
			first[a.Section] = a.Position
		}
	}
	ordered := slices.Clone(attributes)
	slices.SortStableFunc(ordered, func(a, b Attribute) int {
		return cmp.Or(
			cmp.Compare(first[a.Section], first[b.Section]),
			cmp.Compare(a.Section, b.Section), // two sections starting at the same position stay apart
			cmp.Compare(a.Position, b.Position),
			compareIDs(a.ID, b.ID),
		)
	})
	return ordered
}

// optionOrder returns options in display order: by position, then by id.
func optionOrder(options []Option) []Option {
	ordered := slices.Clone(options)
	slices.SortStableFunc(ordered, compareOptions)
	return ordered
}

// compareOptions orders options for display: by position, then by id.
func compareOptions(a, b Option) int {
	return cmp.Or(cmp.Compare(a.Position, b.Position), compareIDs(a.ID, b.ID))
}

// ofLevel returns the attributes of level among attributes, in display order.
func ofLevel(attributes []Attribute, level Level) []Attribute {
	return displayOrder(filter(attributes, func(a Attribute) bool { return a.Level == level }))
}
