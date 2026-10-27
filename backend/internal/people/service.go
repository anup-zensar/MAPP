package people

import (
	"context"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

var themes = []string{themeSystem, themeLight, themeDark}

// planIDPattern checks only the form: an unknown or deleted plan's id is harmless.
var planIDPattern = regexp.MustCompile(`^P-\d+$`)

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock

	// Set by internal/app before the API serves requests; nil plans: no plan uses a team or has an owner.
	plans      PlanSource
	usages     []PersonUsage
	currencies CurrencySource
	access     AccessHooks
}

// NewService knows US dollars as the only currency until UseCurrencies is called.
func NewService(tools history.Tools, repo Store) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, currencies: OnlyUSD}
}

func (s *Service) UsePlans(src PlanSource) {
	s.plans = src
}

func (s *Service) AddPersonUsage(u PersonUsage) {
	s.usages = append(s.usages, u)
}

func (s *Service) UseCurrencies(src CurrencySource) {
	s.currencies = src
}

func (s *Service) allPlans(ctx context.Context) ([]PlanUse, error) {
	if s.plans == nil {
		return nil, nil
	}
	return s.plans(ctx)
}

// ActivePeople takes no actor: the sign-in page lists them before anyone has signed in.
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

// UpdatePreferences changes only the fields sent; preferences have no version and no log entry.
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
		if update.DisplayCurrency != nil {
			changed.DisplayCurrency = *update.DisplayCurrency
		}
		if update.DateFormat != nil {
			changed.DateFormat = *update.DateFormat
		}
		if update.HomeScope != nil {
			// My teams, the default, is stored as no value, so choosing it again saves nothing.
			changed.HomeScope = *update.HomeScope
			if changed.HomeScope == HomeScopeMyTeams {
				changed.HomeScope = ""
			}
		}
		if update.HistoryIncludeInside != nil {
			changed.HistoryIncludeInside = *update.HistoryIncludeInside
		}
		if update.PinnedPlanIDs != nil {
			// A repeated id pins the plan once, keeping the order the person pinned in.
			changed.PinnedPlanIDs = uniqueIDs(*update.PinnedPlanIDs)
		}
		var currencies []Currency
		if changed.DisplayCurrency != current.DisplayCurrency {
			if currencies, err = s.currencies(ctx); err != nil {
				return err
			}
		}
		if err := validatePreferences(current, changed, update, currencies); err != nil {
			return err
		}

		// Saved only when something changed, so a person who changed nothing keeps having no record.
		if !changed.equal(current) {
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

// DisplayCurrencies includes the person's current one even when it was set inactive after they chose it.
func (s *Service) DisplayCurrencies(ctx context.Context, a actor.Actor) ([]DisplayCurrencyOption, error) {
	var options []DisplayCurrencyOption
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		prefs, err := s.preferences(ctx, a.ID)
		if err != nil {
			return err
		}
		currencies, err := s.currencies(ctx)
		if err != nil {
			return err
		}
		options = []DisplayCurrencyOption{}
		for _, c := range currencies {
			if c.Active || c.Code == prefs.DisplayCurrency {
				options = append(options, DisplayCurrencyOption{Code: c.Code, Name: c.Name, Active: c.Active})
			}
		}
		slices.SortFunc(options, func(x, y DisplayCurrencyOption) int { return strings.Compare(x.Code, y.Code) })
		return nil
	})
	if err != nil {
		return nil, err
	}
	return options, nil
}

func (s *Service) preferences(ctx context.Context, personID string) (Preferences, error) {
	prefs, found, err := s.repo.GetPreferences(ctx, personID)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return defaultPreferences(personID), nil
	}
	return prefs.withDefaults(), nil
}

// validatePreferences lets a person keep a display currency set inactive, but not choose it again.
func validatePreferences(current, changed Preferences, update PreferencesUpdate, currencies []Currency) error {
	var problems errs.Problems
	if !slices.Contains(themes, changed.Theme) {
		problems.Add("theme", "Choose system, light or dark.")
	}
	activeCurrency := func(c Currency) bool { return c.Active && c.Code == changed.DisplayCurrency }
	if changed.DisplayCurrency != current.DisplayCurrency && !slices.ContainsFunc(currencies, activeCurrency) {
		problems.Add("displayCurrency", chooseCurrencyMessage)
	}
	if !slices.Contains(DateFormats, changed.DateFormat) {
		problems.Add("dateFormat", chooseDateFormatMessage)
	}
	if update.HomeScope != nil && *update.HomeScope != HomeScopeMyTeams && *update.HomeScope != HomeScopeEverything {
		problems.Add("homeScope", chooseHomeScopeMessage)
	}
	if update.PinnedPlanIDs != nil {
		if slices.ContainsFunc(changed.PinnedPlanIDs, func(id string) bool { return !planIDPattern.MatchString(id) }) {
			problems.Add("pinnedPlanIds", pinPlanIDsMessage)
		}
		if len(changed.PinnedPlanIDs) > maxPinnedPlans {
			problems.Add("pinnedPlanIds", pinLimitMessage)
		}
	}
	return problems.Err()
}

func checkLength(problems *errs.Problems, field, text string, limit int, message string) {
	if utf8.RuneCountInString(text) > limit {
		problems.Add(field, message)
	}
}

func checkVersionSent(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

func checkStatusUpdate(update StatusUpdate) error {
	var problems errs.Problems
	if update.Active == nil {
		problems.Add("active", chooseStatusMessage)
	}
	checkVersionSent(&problems, update.Version)
	return problems.Err()
}

func statusVerb(active bool) string {
	if active {
		return verbReactivated
	}
	return verbDeactivated
}

// uniqueIDs keeps the order the ids were sent; never nil.
func uniqueIDs(ids []string) []string {
	list := []string{}
	for _, id := range ids {
		if !slices.Contains(list, id) {
			list = append(list, id)
		}
	}
	return list
}
