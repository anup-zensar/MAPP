package activity

import (
	"context"
	"slices"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// Owned is an activity as feature 009's Home reads it (Go only): what its page shows of it, amounts in US
// dollars, with its own fields still needed to submit as the plan's completeness lists them.
type Owned struct {
	ID, Name                   string
	Plan, Campaign, Initiative errs.RecordRef
	Status                     Status
	DisplayStatus              DisplayStatus
	StartDate, EndDate         *date.Date
	Ongoing                    bool
	PlannedCost                *money.Money        // USD; nil when not set
	UnassignedPlannedCost      *money.Money        // USD; nil with neither a planned cost nor a line item
	Missing                    []errs.MissingField // the items of "This activity" as fields still needed
	LineItemsNeedingAttention  int
	UpdatedAt                  time.Time
}

// OwnedBy returns every stored activity whose owner is personID, in every plan, worked out as its page works
// it out (FR-019 – FR-024): for feature 009's Home, through an adapter in internal/app (D-196). It asks
// feature 007 once for all their line items, joins the caller's transaction and orders them by ID. Nobody
// ("") owns nothing.
func (s *Service) OwnedBy(ctx context.Context, a actor.Actor, personID string) ([]Owned, error) {
	owned := []Owned{}
	if personID == "" {
		return owned, nil
	}
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		all, err := s.repo.List(ctx)
		if err != nil {
			return err
		}
		mine := slices.DeleteFunc(all, func(act Activity) bool { return act.OwnerID != personID })
		facts, err := s.lineItemFacts(ctx, mine)
		if err != nil {
			return err
		}
		contexts := map[string]planContext{} // by initiative
		for _, act := range mine {
			pc, known := contexts[act.InitiativeID]
			if !known {
				if pc, known, err = s.contextOf(ctx, act.InitiativeID); err != nil {
					return err
				}
				if !known {
					continue // the initiative is gone: so is the activity
				}
				contexts[act.InitiativeID] = pc
			}
			d, err := s.storedDerive(ctx, act, pc, facts[act.ID])
			if err != nil {
				return err
			}
			owned = append(owned, Owned{
				ID: act.ID, Name: act.Name,
				Plan:       errs.RecordRef{Type: "plan", ID: pc.plan.ID, Name: pc.plan.Name},
				Campaign:   errs.RecordRef{Type: "campaign", ID: pc.campaign.ID, Name: pc.campaign.Name},
				Initiative: errs.RecordRef{Type: "initiative", ID: pc.initiative.ID, Name: pc.initiative.Name},
				Status:     act.Status, DisplayStatus: d.displayStatus, StartDate: act.StartDate, EndDate: act.EndDate,
				Ongoing: act.Ongoing, PlannedCost: act.PlannedCost, UnassignedPlannedCost: d.unassigned, Missing: d.missing,
				LineItemsNeedingAttention: d.completeness.LineItemsNeedingAttention, UpdatedAt: act.UpdatedAt,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return owned, nil
}
