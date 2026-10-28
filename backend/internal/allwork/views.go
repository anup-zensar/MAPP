package allwork

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

const (
	msgEachColumnOnce = "Each column can be chosen once."
	msgEnterViewName  = "Enter a name for the view."
	msgViewNameLong   = "Use 60 characters or fewer."
	msgViewNameTaken  = "You already have a view called '%s'."
	msgViewState      = "The view can't be saved — reload the page and try again."
	msgViewGone       = "This view no longer exists — it may have been deleted in another tab."
)

const maxViewName = 60

func (s *Service) ListViews(ctx context.Context, a actor.Actor) (SavedViewPage, error) {
	page := SavedViewPage{Items: []SavedView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		own, err := s.repo.ListViews(ctx, a.ID)
		if err != nil {
			return err
		}
		sortViews(own)
		for _, v := range own {
			v.State = migrateState(v.State)
			page.Items = append(page.Items, v.view())
		}
		page.Total = len(page.Items)
		return nil
	})
	if err != nil {
		return SavedViewPage{}, err
	}
	return page, nil
}

func (s *Service) CreateView(ctx context.Context, a actor.Actor, in SavedViewCreate) (SavedView, error) {
	var created SavedView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		own, err := s.repo.ListViews(ctx, a.ID)
		if err != nil {
			return err
		}
		var problems errs.Problems
		name := checkViewName(&problems, in.Name, own, "")
		var state ViewState
		if in.State == nil {
			problems.Add("state", msgViewState)
		} else {
			state = checkState(&problems, *in.State)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		id, err := s.ids.Next(ctx, ids.PrefixSavedView)
		if err != nil {
			return err
		}
		now := s.now()
		v := StoredView{ID: id, PersonID: a.ID, Name: name, IsDefault: in.IsDefault, State: state, CreatedAt: now, UpdatedAt: now}
		if err := s.saveView(ctx, v, own); err != nil {
			return err
		}
		created = v.view()
		return nil
	})
	if err != nil {
		return SavedView{}, err
	}
	return created, nil
}

func (s *Service) UpdateView(ctx context.Context, a actor.Actor, id string, in SavedViewUpdate) (SavedView, error) {
	var updated SavedView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		v, err := s.ownView(ctx, a, id)
		if err != nil {
			return err
		}
		own, err := s.repo.ListViews(ctx, a.ID)
		if err != nil {
			return err
		}
		var problems errs.Problems
		if in.Name != nil {
			v.Name = checkViewName(&problems, *in.Name, own, id)
		}
		if in.State != nil {
			v.State = checkState(&problems, *in.State)
		}
		if err := problems.Err(); err != nil {
			return err
		}
		if in.IsDefault != nil {
			v.IsDefault = *in.IsDefault
		}
		v.UpdatedAt = s.now()
		if err := s.saveView(ctx, v, own); err != nil {
			return err
		}
		updated = v.view()
		return nil
	})
	if err != nil {
		return SavedView{}, err
	}
	return updated, nil
}

func (s *Service) DeleteView(ctx context.Context, a actor.Actor, id string) (SavedView, error) {
	var deleted SavedView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		v, err := s.ownView(ctx, a, id)
		if err != nil {
			return err
		}
		deleted = v.view()
		return s.repo.DeleteView(ctx, id)
	})
	if err != nil {
		return SavedView{}, err
	}
	return deleted, nil
}

func (s *Service) ownView(ctx context.Context, a actor.Actor, id string) (StoredView, error) {
	v, found, err := s.repo.GetView(ctx, id)
	if err != nil {
		return StoredView{}, err
	}
	if !found || v.PersonID != a.ID {
		return StoredView{}, errs.NotFound(msgViewGone)
	}
	return v, nil
}

func (s *Service) saveView(ctx context.Context, v StoredView, own []StoredView) error {
	if v.IsDefault {
		for _, other := range own {
			if other.ID != v.ID && other.IsDefault {
				other.IsDefault, other.UpdatedAt = false, v.UpdatedAt
				if err := s.repo.PutView(ctx, other); err != nil {
					return err
				}
			}
		}
	}
	return s.repo.PutView(ctx, v)
}

func checkViewName(problems *errs.Problems, name string, own []StoredView, exceptID string) string {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		problems.Add("name", msgEnterViewName)
	case utf8.RuneCountInString(name) > maxViewName:
		problems.Add("name", msgViewNameLong)
	default:
		for _, v := range own {
			if v.ID != exceptID && strings.EqualFold(v.Name, name) {
				problems.Add("name", fmt.Sprintf(msgViewNameTaken, v.Name))
				break
			}
		}
	}
	return name
}

func checkState(problems *errs.Problems, s ViewState) ViewState {
	s = migrateState(s)
	f := &s.Filters
	f.Plans, f.Owners, f.Teams = orNoValues(f.Plans), orNoValues(f.Owners), orNoValues(f.Teams)
	f.BusinessUnits, f.ActivityTypes = orNoValues(f.BusinessUnits), orNoValues(f.ActivityTypes)
	f.PlanStatuses, f.ContainerStatuses, f.ActivityStatuses = orNoKeys(f.PlanStatuses), orNoKeys(f.ContainerStatuses), orNoKeys(f.ActivityStatuses)
	s.Columns = slices.DeleteFunc(orNoKeys(s.Columns), func(key string) bool { return key == nameColumn })
	s.Expansion.Toggled = orNoKeys(s.Expansion.Toggled)
	var state errs.Problems
	checkColumns(&state, "state", s.Columns)
	valid := len(state) == 0 &&
		slices.Contains(viewValues, s.View) && s.Year >= firstYear && s.Year <= lastYear &&
		s.YearTo >= s.YearFrom && s.YearTo-s.YearFrom < maxRangeYears && allIn(f.PlanTypes, planTypeValues) &&
		!slices.ContainsFunc(f.Attributes, func(a AttrFilter) bool { return a.AttributeID == "" || len(a.Values) == 0 }) &&
		(s.Breakdown == nil || s.Breakdown.By != "") &&
		allIn(f.PlanStatuses, planStatusValues) && allIn(f.ContainerStatuses, containerStatusValues) &&
		allIn(f.ActivityStatuses, activityStatusValues) && slices.Contains(strategicValues, f.Strategic) &&
		(f.BCPIR == "" || slices.Contains(strategicValues, f.BCPIR)) &&
		slices.Contains(showLevelsValues, f.ShowLevels) && utf8.RuneCountInString(s.Search) <= maxSearchLength &&
		(s.Sort == nil || (s.Sort.Column != "" && validDirection(s.Sort.Direction))) &&
		slices.Contains([]string{"default", "allExpanded", "allCollapsed"}, s.Expansion.Mode) &&
		slices.Contains(periodValues, s.Period) && slices.Contains([]string{"timeline", "month"}, s.CalendarLayout)
	if !valid {
		problems.Add("state", msgViewState)
	}
	return s
}

func migrateState(s ViewState) ViewState {
	switch {
	case s.YearFrom == 0 && s.YearTo == 0:
		s.YearFrom, s.YearTo = s.Year, s.Year
	case s.YearFrom == 0:
		s.YearFrom = s.YearTo
	case s.YearTo == 0:
		s.YearTo = s.YearFrom
	}
	s.Year = s.YearFrom
	f := &s.Filters
	f.PlanTypes, f.PlanCurrencies = orNoKeys(f.PlanTypes), orNoKeys(f.PlanCurrencies)
	if f.Attributes == nil {
		f.Attributes = []AttrFilter{}
	}
	return s
}

func allIn(values, allowed []string) bool {
	return !slices.ContainsFunc(values, func(v string) bool { return !slices.Contains(allowed, v) })
}

func orNoValues(values []FilterValue) []FilterValue {
	if values == nil {
		return []FilterValue{}
	}
	return values
}

func orNoKeys(keys []string) []string {
	if keys == nil {
		return []string{}
	}
	return keys
}

func sortViews(views []StoredView) {
	slices.SortFunc(views, func(x, y StoredView) int { return compareNames(x.Name, x.ID, y.Name, y.ID) })
}

func checkColumns(problems *errs.Problems, field string, columns []string) {
	if len(columns) > maxColumns {
		problems.Add(field, msgTooManyCols)
	}
	seen := make(map[string]bool, len(columns))
	for _, key := range columns {
		if seen[key] {
			problems.Add(field, msgEachColumnOnce)
			return
		}
		seen[key] = true
	}
}

func validDirection(direction string) bool {
	return direction == SortAscending || direction == SortDescending
}
