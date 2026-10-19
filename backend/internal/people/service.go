package people

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
)

// themes are the values a person can choose for Preferences.Theme.
var themes = []string{themeSystem, themeLight, themeDark}

// Service holds the rules about people, teams, team memberships and preferences.
type Service struct {
	tx      store.Transactor
	repo    Repository
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock

	// The hooks of later features (usage.go), set by internal/app before the API serves requests.
	plans      PlanSource     // nil until feature 004: no plan uses a team or has an owner
	usages     []PersonUsage  // in registration order
	currencies CurrencySource // OnlyUSD until feature 002
}

// NewService returns the people service. It keeps the records in repo, gives them readable IDs with
// generator, logs their changes and deletes teams through hist, works in transactions of tx and reads the
// time from now. Until UseCurrencies is called, US dollars are the only currency.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, now: now, currencies: OnlyUSD}
}

// UsePlans tells people where the plans are (feature 004), for the plans a team is used by, the owner
// warnings of membership changes and the plans a person owns (D-060).
func (s *Service) UsePlans(src PlanSource) {
	s.plans = src
}

// AddPersonUsage adds one count of what a person owns or approves (features 005, 006, 008), shown in the
// person's details after the plans they own (FR-004).
func (s *Service) AddPersonUsage(u PersonUsage) {
	s.usages = append(s.usages, u)
}

// UseCurrencies tells people which currencies exist (feature 002), for display currencies and their rates.
func (s *Service) UseCurrencies(src CurrencySource) {
	s.currencies = src
}

// allPlans returns every plan of the PlanSource, stored and in the recycle bin, inside the caller's
// transaction; none before feature 004 is wired.
func (s *Service) allPlans(ctx context.Context) ([]PlanUse, error) {
	if s.plans == nil {
		return nil, nil
	}
	return s.plans(ctx)
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
		if update.HomeScope != nil {
			// The default, My teams, is stored as no value, so choosing it again saves nothing (009 D-172).
			changed.HomeScope = *update.HomeScope
			if changed.HomeScope == HomeScopeMyTeams {
				changed.HomeScope = ""
			}
		}
		if err := validatePreferences(changed, update); err != nil {
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

// validatePreferences checks the rules of data model §2 and reports every problem at once. The home scope is
// checked as it was sent: My teams or Everything (009 FR-003).
func validatePreferences(p Preferences, update PreferencesUpdate) error {
	var problems errs.Problems
	if !slices.Contains(themes, p.Theme) {
		problems.Add("theme", "Choose system, light or dark.")
	}
	if update.HomeScope != nil && *update.HomeScope != HomeScopeMyTeams && *update.HomeScope != HomeScopeEverything {
		problems.Add("homeScope", chooseHomeScopeMessage)
	}
	return problems.Err()
}

// The checks below are shared by the forms of people and teams. Each adds its problem to problems, so
// that every problem of a request is reported at once (SC-007).

// checkLength adds message for field when text, already trimmed, has more than limit characters
// (characters, not bytes: "é" counts as one).
func checkLength(problems *errs.Problems, field, text string, limit int, message string) {
	if utf8.RuneCountInString(text) > limit {
		problems.Add(field, message)
	}
}

// checkVersionSent adds the version problem when an update or status action sent no version, or one below
// 1: the version check cannot work without it (001 research R5).
func checkVersionSent(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

// checkStatusUpdate reports the problems of the body of a status action, {active, version}: both are needed.
func checkStatusUpdate(update StatusUpdate) error {
	var problems errs.Problems
	if update.Active == nil {
		problems.Add("active", chooseStatusMessage)
	}
	checkVersionSent(&problems, update.Version)
	return problems.Err()
}

// statusVerb is the log verb of a status action: "reactivated" when the record becomes active,
// "deactivated" when it becomes inactive.
func statusVerb(active bool) string {
	if active {
		return verbReactivated
	}
	return verbDeactivated
}

// uniqueIDs returns ids without repeats, in the order they were sent: a repeated id in a request is
// ignored, never a second membership (003 data model §3). Never nil.
func uniqueIDs(ids []string) []string {
	list := []string{}
	for _, id := range ids {
		if !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}
