package allwork

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
)

// The rules of All work's personal settings (data model §1): the remembered columns and, later in this file,
// the saved views. Both are personal settings like D-038 (D-121): no version, no activity-log entry, no
// recycle bin, and nobody but their owner reads or changes them.

// Messages of the plan's validation table (API).
const (
	msgEachColumnOnce = "Each column can be chosen once."
	msgEnterViewName  = "Enter a name for the view."
	msgViewNameLong   = "Use 60 characters or fewer."
	msgViewNameTaken  = "You already have a view called '%s'."
	msgViewState      = "The view can't be saved — reload the page and try again."
	msgViewGone       = "This view no longer exists — it may have been deleted in another tab."
)

// maxViewName is the longest name of a saved view, in characters (FR-029).
const maxViewName = 60

// ListViews returns a's saved views, sorted by name ignoring letter case; nobody else's (FR-030).
func (s *Service) ListViews(ctx context.Context, a actor.Actor) (SavedViewPage, error) {
	page := SavedViewPage{Items: []SavedView{}}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		own, err := s.repo.ListViews(ctx, a.ID)
		if err != nil {
			return err
		}
		sortViews(own)
		for _, v := range own {
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

// CreateView saves a's current state under a name (FR-029): the name is required, trimmed, at most 60
// characters and unique among a's views ignoring letter case; the state is checked; a view made the default
// takes that place from a's other default in the same Write (FR-031). Every problem is reported at once.
// It is a personal setting: not logged (D-121).
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

// UpdateView changes a's own view id (FR-030, FR-031): a new name (the rules of CreateView), a new state
// (Update view), or whether it is the default; only the parts sent change. Another person's view, or one that
// is gone, is not found, never revealed.
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

// DeleteView deletes a's own view id and answers it, so the web app's Undo can save it again (FR-031). It is
// a personal setting: not to the recycle bin and not logged (D-121).
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

// ownView returns a's view id, or not found when it is gone or someone else's (FR-030).
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

// saveView stores v; when it is the default, the person's other default view stops being one (FR-031).
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

// checkViewName returns the trimmed name, adding its problems (FR-029): empty, longer than 60 characters, or
// the name of another of the person's views than exceptID, ignoring letter case.
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

// checkState returns the state of a view with its lists never nil and Name left out of its columns, adding
// the one state problem when a value is not one of its choices (data model §1.2): the view, the year, the
// filters with fixed values, the search, the columns, the sort, the expansion, the period and the calendar
// layout.
func checkState(problems *errs.Problems, s ViewState) ViewState {
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
		allIn(f.PlanStatuses, planStatusValues) && allIn(f.ContainerStatuses, containerStatusValues) &&
		allIn(f.ActivityStatuses, activityStatusValues) && slices.Contains(strategicValues, f.Strategic) &&
		slices.Contains(showLevelsValues, f.ShowLevels) && utf8.RuneCountInString(s.Search) <= maxSearchLength &&
		(s.Sort == nil || (s.Sort.Column != "" && validDirection(s.Sort.Direction))) &&
		slices.Contains([]string{"default", "allExpanded", "allCollapsed"}, s.Expansion.Mode) &&
		slices.Contains(periodValues, s.Period) && slices.Contains([]string{"timeline", "month"}, s.CalendarLayout)
	if !valid {
		problems.Add("state", msgViewState)
	}
	return s
}

// allIn reports whether every value is one of allowed.
func allIn(values, allowed []string) bool {
	return !slices.ContainsFunc(values, func(v string) bool { return !slices.Contains(allowed, v) })
}

// orNoValues returns values, or an empty list for nil.
func orNoValues(values []FilterValue) []FilterValue {
	if values == nil {
		return []FilterValue{}
	}
	return values
}

// orNoKeys returns keys, or an empty list for nil.
func orNoKeys(keys []string) []string {
	if keys == nil {
		return []string{}
	}
	return keys
}

// sortViews orders views by name ignoring letter case, then by ID.
func sortViews(views []StoredView) {
	slices.SortFunc(views, func(x, y StoredView) int { return compareNames(x.Name, x.ID, y.Name, y.ID) })
}

// Preferences returns a's remembered columns, their order and the sort, or the defaults when a never changed
// them: the default columns and the plan tree's order (FR-019, FR-021).
func (s *Service) Preferences(ctx context.Context, a actor.Actor) (Preferences, error) {
	var p Preferences
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		p, err = s.preferences(ctx, a.ID)
		return err
	})
	if err != nil {
		return Preferences{}, err
	}
	return p, nil
}

// preferences returns the preferences of personID, or the defaults, inside the caller's transaction.
func (s *Service) preferences(ctx context.Context, personID string) (Preferences, error) {
	stored, found, err := s.repo.GetPreferences(ctx, personID)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return Preferences{Columns: slices.Clone(DefaultColumns)}, nil
	}
	columns := stored.Columns
	if columns == nil {
		columns = []string{}
	}
	return Preferences{Columns: columns, Sort: stored.Sort}, nil
}

// UpdatePreferences remembers a's columns, their order and the sort (FR-021): at most 100 columns, each
// once; Name is always shown and never stored; a sort names a column of the grid and a direction. Columns
// that no longer exist are kept, because an attribute may come back. Every problem is reported at once.
// Nothing is saved when nothing changed, and nothing is logged (D-121).
func (s *Service) UpdatePreferences(ctx context.Context, a actor.Actor, p Preferences) (Preferences, error) {
	columns := slices.DeleteFunc(slices.Clone(p.Columns), func(key string) bool { return key == nameColumn })
	if columns == nil {
		columns = []string{}
	}
	var problems errs.Problems
	checkColumns(&problems, "columns", columns)
	var saved Preferences
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		if p.Sort != nil {
			columns, err := s.catalogue(ctx)
			if err != nil {
				return err
			}
			if _, known := catalogueOf(columns)[p.Sort.Column]; !known || !validDirection(p.Sort.Direction) {
				problems.Add("sort", msgChooseSort)
			}
		}
		if err := problems.Err(); err != nil {
			return err
		}
		current, err := s.preferences(ctx, a.ID)
		if err != nil {
			return err
		}
		next := Preferences{Columns: columns, Sort: p.Sort}
		if slices.Equal(current.Columns, next.Columns) && sameSort(current.Sort, next.Sort) {
			saved = current
			return nil
		}
		saved = next
		return s.repo.PutPreferences(ctx, PreferencesRecord{PersonID: a.ID, Columns: columns, Sort: p.Sort, UpdatedAt: s.now()})
	})
	if err != nil {
		return Preferences{}, err
	}
	return saved, nil
}

// checkColumns adds the problems of a list of column keys under field: more than 100, or a key listed twice.
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

// validDirection reports whether direction is asc or desc.
func validDirection(direction string) bool {
	return direction == SortAscending || direction == SortDescending
}

// sameSort reports whether two sorts are the same (both none, or the same column and direction).
func sameSort(x, y *Sort) bool {
	if x == nil || y == nil {
		return x == y
	}
	return *x == *y
}
