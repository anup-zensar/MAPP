package overspend

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
)

const (
	configurationID   = "overspend"
	configurationKind = "overspendConfiguration"
	settingsName      = "Overspend control"
	fieldDefaultMode  = "defaultMode"
	configurationNoun = "setting"
)

// Service holds Settings › Overspend control and answers what the planning services ask of it: the setting new
// budgets start with and the limits above an activity.
type Service struct {
	tx      Transactor
	repo    Store
	history *history.Service
	now     clock.Clock
	budgets *reader
}

func NewService(tools history.Tools, repo Store) *Service {
	return &Service{tx: tools.Tx, repo: repo, history: tools.Log, now: tools.Now, budgets: &reader{repo: repo}}
}

func (s *Service) UseRates(r Rates) {
	s.budgets.rates = r
}

func (s *Service) Configuration(ctx context.Context, _ actor.Actor) (ConfigurationView, error) {
	var view ConfigurationView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.repo.Configuration(ctx)
		view = configurationView(ctx, c)
		return err
	})
	return view, err
}

// SaveConfiguration changes the setting new budgets start with (FR-006); it needs settings.manage.
func (s *Service) SaveConfiguration(ctx context.Context, a actor.Actor, in ConfigurationInput) (ConfigurationView, error) {
	if err := authz.Require(ctx, authz.SettingsManage); err != nil {
		return ConfigurationView{}, err
	}
	mode, ok := ParseMode(in.DefaultMode)
	if !ok {
		return ConfigurationView{}, errs.Validation(errs.FieldError{Field: fieldDefaultMode, Message: MsgChooseMode})
	}
	var view ConfigurationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, err := s.repo.Configuration(ctx)
		if err != nil {
			return err
		}
		if err = before.CheckVersion(in.Version, configurationNoun); err != nil {
			return err
		}
		after := before
		after.DefaultMode = mode
		if after.DefaultMode == before.DefaultMode.Normalised() {
			view = configurationView(ctx, before)
			return nil
		}
		view, err = s.saveChange(ctx, a, before, after)
		return err
	})
	return view, err
}

func (s *Service) saveChange(ctx context.Context, a actor.Actor, before, after Configuration) (ConfigurationView, error) {
	after.Touch(a, s.now())
	if err := s.repo.PutConfiguration(ctx, after); err != nil {
		return ConfigurationView{}, err
	}
	changes, err := history.Diff(before, after)
	if err != nil {
		return ConfigurationView{}, err
	}
	_, err = s.history.Log(ctx, a, history.LogInput{Kind: configurationKind, RecordID: configurationID,
		RecordName: settingsName, Action: history.ActionChanged, Changes: changes})
	return configurationView(ctx, after), err
}

func configurationView(ctx context.Context, c Configuration) ConfigurationView {
	return ConfigurationView{DefaultMode: c.DefaultMode.Normalised(), Version: c.Version, UpdatedAt: c.UpdatedAt,
		UpdatedBy: c.UpdatedBy, CanChange: authz.AvailableAnywhere(ctx, authz.SettingsManage)}
}

// DefaultMode is the setting a new budget starts with (FR-007); it joins the caller's transaction.
func (s *Service) DefaultMode(ctx context.Context) (Mode, error) {
	c, err := s.repo.Configuration(ctx)
	return c.DefaultMode.Normalised(), err
}

// Limits lists the budgets set to Refuse above an activity at path, with what is left to plan in each
// (FR-014); it joins the caller's transaction.
func (s *Service) Limits(ctx context.Context, path Path) ([]Limit, error) {
	budgets, err := s.budgets.all(ctx)
	if err != nil {
		return nil, err
	}
	rates, err := s.budgets.currentRates(ctx)
	if err != nil {
		return nil, err
	}
	return LimitsOf(budgets, path, rates)
}
