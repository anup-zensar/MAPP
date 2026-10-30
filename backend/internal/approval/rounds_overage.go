package approval

import (
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/errs"
)

// ApprovalOverspend is Settings › Overspend control as approvals read it: whether approving is refused while an
// activity's line items exceed its planned cost.
type ApprovalOverspend interface {
	RefusesOverPlannedCost(ctx context.Context) (bool, error)
}

func (s *Service) UseApprovalOverspend(o ApprovalOverspend) {
	s.overspend = o
}

const (
	overageClause  = "line items of activity '%s' are %s above its planned cost"
	overageRefusal = " — ask the owner to lower them, or decline."
	overageLogged  = " · while line items were %s above planned cost"
	overageNamed   = " · while line items of activity '%s' were %s above planned cost"
)

func (s *Service) refusesOverPlannedCost(ctx context.Context) (bool, error) {
	if s.overspend == nil {
		return false, nil
	}
	return s.overspend.RefusesOverPlannedCost(ctx)
}

// overageRefused is "" unless the setting refuses an approval of the record's overages.
func (s *Service) overageRefused(ctx context.Context, sub subject) (string, error) {
	if len(sub.Overages) == 0 {
		return "", nil
	}
	refuses, err := s.refusesOverPlannedCost(ctx)
	if err != nil || !refuses {
		return "", err
	}
	clauses := make([]string, 0, len(sub.Overages))
	for _, o := range sub.Overages {
		clauses = append(clauses, fmt.Sprintf(overageClause, o.Activity.Name, o.OverBy.Format()))
	}
	text := strings.Join(clauses, "; ") + overageRefusal
	return strings.ToUpper(text[:1]) + text[1:], nil
}

func (s *Service) refuseOverage(ctx context.Context, dg deciding) error {
	if dg.kind != ActionApproved {
		return nil
	}
	refusal, err := s.overageRefused(ctx, dg.sub)
	if err != nil || refusal == "" {
		return err
	}
	return errs.NotAllowed(refusal)
}

// overagesOn shows the open round's overages and, when the setting refuses, why Approve is not offered.
func (s *Service) overagesOn(ctx context.Context, view *Section, sub subject) error {
	view.OverPlannedCost = append([]Overage{}, sub.Overages...)
	refusal, err := s.overageRefused(ctx, sub)
	if err != nil || refusal == "" || !view.Actions.Approve.Available {
		return err
	}
	view.Actions.Approve = ActionAvailability{Reason: refusal}
	return nil
}

func overageDetails(sub subject) string {
	var details string
	for _, o := range sub.Overages {
		if sub.Kind == KindActivity {
			details += fmt.Sprintf(overageLogged, o.OverBy.Format())
			continue
		}
		details += fmt.Sprintf(overageNamed, o.Activity.Name, o.OverBy.Format())
	}
	return details
}
