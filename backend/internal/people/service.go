package people

import (
	"context"
	"slices"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/store"
)

// themes are the values a person can choose for Preferences.Theme.
var themes = []string{themeSystem, themeLight, themeDark}

// Service holds the rules about people and their preferences.
type Service struct {
	tx   store.Transactor
	repo Repository
}

// NewService returns the people service, which keeps its data in repo and runs in transactions of tx.
func NewService(tx store.Transactor, repo Repository) *Service {
	return &Service{tx: tx, repo: repo}
}

// ActivePeople returns the people who can sign in, sorted by name, as the sign-in page lists them
// (FR-001). It takes no actor because the sign-in page needs it before anyone has signed in. The list is
// empty, never nil, when no demo data was loaded (FR-028).
func (s *Service) ActivePeople(ctx context.Context) ([]DemoPerson, error) {
	var everyone []Person
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		everyone, err = s.repo.List(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Stable, so that two people with the same name keep the order of their ids.
	slices.SortStableFunc(everyone, func(a, b Person) int { return strings.Compare(a.Name, b.Name) })
	active := make([]DemoPerson, 0, len(everyone))
	for _, p := range everyone {
		if p.Active {
			active = append(active, p.demoPerson())
		}
	}
	return active, nil
}

// CurrentUser returns the signed-in person a with their preferences: what the sidebar and the account menu
// show (FR-012). A person who no longer exists must sign in again.
func (s *Service) CurrentUser(ctx context.Context, a actor.Actor) (CurrentUser, error) {
	var user CurrentUser
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		person, found, err := s.repo.Get(ctx, a.ID)
		if err != nil {
			return err
		}
		if !found {
			return errs.SignInRequired()
		}
		prefs, err := s.preferences(ctx, a.ID)
		if err != nil {
			return err
		}
		user = person.currentUser(prefs)
		return nil
	})
	if err != nil {
		return CurrentUser{}, err
	}
	return user, nil
}

// UpdatePreferences changes the preferences of the signed-in person a (FR-011, FR-014): only the fields
// that were sent change. It returns the preferences after the change. Preferences are not business
// records, so the change has no version and is not written to the activity log (D-038).
func (s *Service) UpdatePreferences(ctx context.Context, a actor.Actor, update PreferencesUpdate) (PreferencesView, error) {
	var view PreferencesView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		current, err := s.preferences(ctx, a.ID)
		if err != nil {
			return err
		}

		changed := current
		if update.Theme != nil {
			changed.Theme = *update.Theme
		}
		if update.SidebarCollapsed != nil {
			changed.SidebarCollapsed = *update.SidebarCollapsed
		}
		if err := validatePreferences(changed); err != nil {
			return err
		}

		// Saved only when something changed, so a person who changed nothing keeps having no record.
		if changed != current {
			if err := s.repo.PutPreferences(ctx, changed); err != nil {
				return err
			}
		}
		view = changed.view()
		return nil
	})
	if err != nil {
		return PreferencesView{}, err
	}
	return view, nil
}

// preferences returns the saved preferences of a person, or the defaults when they have saved none.
func (s *Service) preferences(ctx context.Context, personID string) (Preferences, error) {
	prefs, found, err := s.repo.GetPreferences(ctx, personID)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return defaultPreferences(personID), nil
	}
	return prefs, nil
}

// validatePreferences checks the rules of data model §2 and reports every problem at once.
func validatePreferences(p Preferences) error {
	var problems errs.Problems
	if !slices.Contains(themes, p.Theme) {
		problems.Add("theme", "Choose system, light or dark.")
	}
	return problems.Err()
}
