package settings

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// GetOption returns the option id as it is stored now, with its parent options, usage and locks (D-211):
// the web app reads it after a version conflict. Every signed-in person may read it (FR-008), so a is not
// used yet.
func (s *Service) GetOption(ctx context.Context, a actor.Actor, id string) (OptionView, error) {
	var view OptionView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, stored, attribute, err := s.optionToChange(ctx, id)
		if err != nil {
			return err
		}
		view, err = s.optionViewOf(ctx, f, stored, attribute)
		return err
	})
	return view, err
}

// optionViewOf returns the option o of the attribute a as the API answers it, reading the usage index.
// f must hold the parent attribute's options.
func (s *Service) optionViewOf(ctx context.Context, f formData, o Option, a Attribute) (OptionView, error) {
	u, err := s.usage(ctx)
	if err != nil {
		return OptionView{}, err
	}
	return f.optionView(o, a, u), nil
}

// The statuses an option list can be filtered by; "" means all.
const (
	statusAll      = "all"
	statusActive   = "active"
	statusInactive = "inactive"
)

// ListOptions returns the options of the attribute attributeID in display order (FR-018, FR-020): those
// whose label contains filter.Q ignoring letter case and surrounding spaces ("0220" finds "CN04000220",
// SC-007), of the status asked, at most filter.Limit (1 to 500) after skipping filter.Offset; the page's
// total counts every match. Each option has its parent options and its own usage (US2-3).
func (s *Service) ListOptions(ctx context.Context, a actor.Actor, attributeID string, filter OptionFilter) (OptionPage, error) {
	var problems errs.Problems
	if filter.Limit < 1 || filter.Limit > maxOptionLimit {
		problems.Add("limit", limitMessage)
	}
	if filter.Offset < 0 {
		problems.Add("offset", offsetMessage)
	}
	if filter.Status != "" && filter.Status != statusAll && filter.Status != statusActive && filter.Status != statusInactive {
		problems.Add("status", optionStatusMessage)
	}
	if err := problems.Err(); err != nil {
		return OptionPage{}, err
	}

	page := OptionPage{Items: []OptionView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		f, err := s.loadForm(ctx, "")
		if err != nil {
			return err
		}
		attribute, found := f.byID[attributeID]
		if !found {
			return errs.NotFound(attributeGoneMessage)
		}
		u, err := s.usage(ctx)
		if err != nil {
			return err
		}
		search := strings.ToLower(strings.TrimSpace(filter.Q))
		var matches []Option
		for _, o := range f.options[attributeID] {
			if (filter.Status == statusActive && !o.Active) || (filter.Status == statusInactive && o.Active) {
				continue
			}
			if strings.Contains(strings.ToLower(o.Label), search) {
				matches = append(matches, o)
			}
		}
		page.Total = len(matches)
		for _, o := range matches[min(filter.Offset, len(matches)):min(filter.Offset+filter.Limit, len(matches))] {
			page.Items = append(page.Items, f.optionView(o, attribute, u))
		}
		return nil
	})
	return page, err
}

// optionView returns the option o of the attribute a as the API answers it.
func (f formData) optionView(o Option, a Attribute, u *usageIndex) OptionView {
	parents := []SettingRef{}
	for _, id := range o.ParentOptionIDs {
		p := f.optionByID[id]
		parents = append(parents, SettingRef{ID: id, Name: p.Label, Active: p.Active})
	}
	return OptionView{
		ID: o.ID, AttributeID: o.AttributeID, Label: o.Label, Position: o.Position, ParentOptions: parents,
		Active: o.Active, Usage: u.option(o, a), Locks: f.optionLocks(o, a), Meta: o.Meta,
	}
}

// optionLocks says what can't be done with the option o of the attribute a, and why (design system 6.6,
// D-211): the last option of a choice attribute can't be deleted (FR-018), as DeleteOption answers. An
// option that f does not hold yet (one just created) counts as one more option. Never nil.
func (f formData) optionLocks(o Option, a Attribute) []Lock {
	count := len(f.options[a.ID])
	if _, held := f.optionByID[o.ID]; !held {
		count++
	}
	if a.IsChoice() && count <= 1 {
		return []Lock{{Field: "delete", Reason: fmt.Sprintf(lastOptionLockMessage, a.Label, o.Label)}}
	}
	return []Lock{}
}
