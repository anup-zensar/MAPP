package purchaserequest

import (
	"context"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const configurationID = "purchase-requests"

// Configuration is Settings › Purchase requests, with whether the person may change it.
func (s *Service) Configuration(ctx context.Context, _ actor.Actor) (ConfigurationView, error) {
	var view ConfigurationView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		c, err := s.repo.Configuration(ctx)
		view = configurationView(ctx, c)
		return err
	})
	return view, err
}

// SaveConfiguration changes Purchase Support and Lines on approved purchase requests (needs settings.manage).
func (s *Service) SaveConfiguration(ctx context.Context, a actor.Actor, in ConfigurationInput) (ConfigurationView, error) {
	if err := authz.Require(ctx, authz.SettingsManage); err != nil {
		return ConfigurationView{}, err
	}
	switch in.LinesOnApprovedPurchaseRequests {
	case NotAllowed, WithModificationApproval, WithoutApproval:
	default:
		return ConfigurationView{}, errs.Validation(errs.FieldError{Field: "linesOnApprovedPurchaseRequests", Message: msgLinesOnApproved})
	}
	var view ConfigurationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		before, err := s.repo.Configuration(ctx)
		if err != nil {
			return err
		}
		if err = before.CheckVersion(in.Version, "setting"); err != nil {
			return err
		}
		after := before
		after.PurchaseSupport, after.LinesOnApproved = in.PurchaseSupport, in.LinesOnApprovedPurchaseRequests
		after.Touch(a, s.now())
		if err = s.repo.PutConfiguration(ctx, after); err != nil {
			return err
		}
		changes, err := history.Diff(before, after)
		if err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{Kind: configurationKind, RecordID: configurationID,
			RecordName: settingsName, Action: history.ActionChanged, Changes: changes})
		view = configurationView(ctx, after)
		return err
	})
	return view, err
}

// configurationView has no last change while only SP&A One has saved the setting.
func configurationView(ctx context.Context, c Configuration) ConfigurationView {
	view := ConfigurationView{PurchaseSupport: c.PurchaseSupport, LinesOnApprovedPurchaseRequests: c.LinesOnApproved,
		Version: c.Version, CanChange: authz.AvailableAnywhere(ctx, authz.SettingsManage)}
	if !c.UpdatedAt.IsZero() && c.UpdatedBy.ID != actor.System.ID {
		at, by := c.UpdatedAt, c.UpdatedBy
		view.UpdatedAt, view.UpdatedBy = &at, &by
	}
	return view
}
