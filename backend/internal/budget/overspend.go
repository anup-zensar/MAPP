package budget

import (
	"context"

	"mapp/internal/overspend"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
)

const fieldOverspend = "overspend"

// checkOverspend reads the overspend setting a request sends (037 FR-001): keep when none is sent, a field
// problem for anything but Warn and Refuse.
func checkOverspend(sent *string, keep overspend.Mode, problems *errs.Problems) overspend.Mode {
	if sent == nil {
		return keep
	}
	mode, ok := overspend.ParseMode(*sent)
	if !ok {
		problems.Add(fieldOverspend, overspend.MsgChooseMode)
	}
	return mode
}

func (s *Service) defaultOverspend(ctx context.Context) (overspend.Mode, error) {
	if s.overspend == nil {
		return overspend.Warn, nil
	}
	return s.overspend.DefaultMode(ctx)
}

func controlOf(b GlobalBudget, edit Availability) overspend.Control {
	return overspend.Control{Mode: b.Overspend.Normalised(),
		CanChange: authz.Availability{Available: edit.Available, Reason: edit.Reason}}
}
