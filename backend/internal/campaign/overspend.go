package campaign

import (
	"context"

	"mapp/internal/overspend"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const fieldOverspend = "overspend"

// OverspendDefaults gives the setting a new campaign or initiative starts with (037 FR-007).
type OverspendDefaults interface {
	DefaultMode(ctx context.Context) (overspend.Mode, error)
}

func (s *Service) UseOverspendDefaults(d OverspendDefaults) {
	s.overspendDefaults = d
}

func (s *Service) newOverspend(ctx context.Context, problems *errs.Problems, sent *string) (overspend.Mode, error) {
	if sent != nil {
		return parsedOverspend(problems, *sent), nil
	}
	if s.overspendDefaults == nil {
		return overspend.Warn, nil
	}
	return s.overspendDefaults.DefaultMode(ctx)
}

// changedOverspend is the setting an update stores (037 FR-003): a change needs the record's owner or a budget
// manager of the plan (403 otherwise).
func changedOverspend(ctx context.Context, problems *errs.Problems, sent *string, before ownedSetting) (overspend.Mode, error) {
	if sent == nil {
		return before.mode.Normalised(), nil
	}
	mode := parsedOverspend(problems, *sent)
	if mode == before.mode.Normalised() {
		return mode, nil
	}
	return mode, overspend.RequireChange(ctx, ownerOf(before.ownerID), before.plan.Access)
}

type ownedSetting struct {
	mode    overspend.Mode
	ownerID *string
	plan    PlanFacts
}

func parsedOverspend(problems *errs.Problems, sent string) overspend.Mode {
	mode, ok := overspend.ParseMode(sent)
	if !ok {
		problems.Add(fieldOverspend, overspend.MsgChooseMode)
	}
	return mode
}

func ownerOf(ownerID *string) string {
	if ownerID == nil {
		return ""
	}
	return *ownerID
}

func withOverspendAccess(ctx context.Context, view CampaignView, plan authz.PlanFacts) CampaignView {
	view.Overspend = overspend.ControlOf(ctx, view.Overspend.Mode, ownerOf(view.OwnerID), plan)
	for n, i := range view.Initiatives {
		view.Initiatives[n] = withInitiativeAccess(ctx, i, plan)
	}
	return view
}

func withInitiativeAccess(ctx context.Context, view InitiativeView, plan authz.PlanFacts) InitiativeView {
	view.Overspend = overspend.ControlOf(ctx, view.Overspend.Mode, ownerOf(view.OwnerID), plan)
	return view
}
