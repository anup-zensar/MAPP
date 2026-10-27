package allwork

import (
	"context"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

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

// sameSort reports whether two sorts are the same (both none, or the same column and direction).
func sameSort(x, y *Sort) bool {
	if x == nil || y == nil {
		return x == y
	}
	return *x == *y
}
