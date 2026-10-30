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
	fieldApprovalMode = "approvalOverPlannedCost"
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

// SaveConfiguration needs settings.manage.
func (s *Service) SaveConfiguration(ctx context.Context, a actor.Actor, in ConfigurationInput) (ConfigurationView, error) {
	if err := authz.Require(ctx, authz.SettingsManage); err != nil {
		return ConfigurationView{}, err
	}
	if err := in.problems(); err != nil {
		return ConfigurationView{}, err
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
		before = before.normalised()
		after := in.applied(before)
		if after == before {
			view = configurationView(ctx, before)
			return nil
		}
		view, err = s.saveChange(ctx, a, before, after)
		return err
	})
	return view, err
}

func (in ConfigurationInput) problems() error {
	var problems errs.Problems
	if _, ok := ParseMode(in.DefaultMode); !ok {
		problems.Add(fieldDefaultMode, MsgChooseMode)
	}
	if in.ApprovalOverPlannedCost != nil {
		if _, ok := ParseMode(*in.ApprovalOverPlannedCost); !ok {
			problems.Add(fieldApprovalMode, MsgChooseMode)
		}
	}
	return problems.Err()
}

func (in ConfigurationInput) applied(before Configuration) Configuration {
	after := before
	after.DefaultMode, _ = ParseMode(in.DefaultMode)
	if in.ApprovalOverPlannedCost != nil {
		after.ApprovalOverPlannedCost, _ = ParseMode(*in.ApprovalOverPlannedCost)
	}
	return after
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
	view := ConfigurationView{DefaultMode: c.DefaultMode.Normalised(),
		ApprovalOverPlannedCost: c.ApprovalOverPlannedCost.Normalised(), Version: c.Version,
		CanChange: authz.AvailableAnywhere(ctx, authz.SettingsManage)}
	if !c.UpdatedAt.IsZero() && c.UpdatedBy.ID != actor.System.ID {
		at, by := c.UpdatedAt, c.UpdatedBy
		view.UpdatedAt, view.UpdatedBy = &at, &by
	}
	return view
}

// RefusesOverPlannedCost joins the caller's transaction.
func (s *Service) RefusesOverPlannedCost(ctx context.Context) (bool, error) {
	c, err := s.repo.Configuration(ctx)
	return c.ApprovalOverPlannedCost.Refuses(), err
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
