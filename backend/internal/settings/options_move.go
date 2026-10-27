package settings

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// MoveOption swaps the option id with its neighbour above or below in its attribute's list (FR-018), from
// the version the person started at. At the ends it is refused on direction. Both options get one
// "changed" entry (field position).
func (s *Service) MoveOption(ctx context.Context, a actor.Actor, id string, in OptionMoveInput) (OptionView, error) {
	var problems errs.Problems
	if in.Direction != MoveUp && in.Direction != MoveDown {
		problems.Add("direction", reloadMessage)
	}
	checkVersion(&problems, in.Version)
	if err := problems.Err(); err != nil {
		return OptionView{}, err
	}
	var view OptionView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		if err = stored.CheckVersion(*in.Version, optionNoun); err != nil {
			return err
		}
		options := f.options[attribute.ID]
		i := slices.IndexFunc(options, func(o Option) bool { return o.ID == id })
		j := i - 1
		if in.Direction == MoveDown {
			j = i + 1
		}
		switch {
		case j < 0:
			return errs.Validation(errs.FieldError{Field: "direction", Message: fmt.Sprintf(firstOptionMessage, stored.Label)})
		case j >= len(options):
			return errs.Validation(errs.FieldError{Field: "direction", Message: fmt.Sprintf(lastOptionMessage, stored.Label)})
		}
		order := slices.Clone(options)
		order[i], order[j] = order[j], order[i]
		for _, changed := range repositionedOptions(options, order) {
			change := history.Change{Field: "position", Old: jsonOf(f.optionByID[changed.ID].Position), New: jsonOf(changed.Position)}
			changed.Touch(a, s.now())
			if err = s.repo.PutOption(ctx, changed); err != nil {
				return err
			}
			if err = s.logChanged(ctx, a, optionKind, changed.ID, changed.Label, []history.Change{change}, ""); err != nil {
				return err
			}
			if changed.ID == id {
				stored = changed
			}
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// repositionedOptions gives the options of order — the options of one attribute, which are in display
// order, in a new order — the positions that make that order, and returns those whose position changes: a
// move swaps two positions, or, when both have the same position, numbers the list 1…n.
func repositionedOptions(options, order []Option) []Option {
	next := slices.Clone(order)
	for k := range next {
		next[k].Position = options[k].Position
	}
	if !slices.EqualFunc(optionOrder(next), next, func(x, y Option) bool { return x.ID == y.ID }) {
		for k := range next {
			next[k].Position = k + 1
		}
	}
	positions := make(map[string]int, len(options))
	for _, o := range options {
		positions[o.ID] = o.Position
	}
	changed := []Option{}
	for _, o := range next {
		if positions[o.ID] != o.Position {
			changed = append(changed, o)
		}
	}
	return changed
}
