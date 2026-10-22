package campaigns

import (
	"context"
	"reflect"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
)

// Budget by month of campaigns and initiatives (016 FR-020 – FR-025, D-265): the same rules as the plan's,
// over the plan's year in the plan currency. A campaign whose dates leave the plan's year is still phased over
// the plan's year only.

// SetCampaignBudgetPhasing saves the phasing of the campaign id's budget, from the version the person started
// at, whatever the plan's status (FR-023). Every changed month is logged; a save that changes nothing writes
// no version and no log entry. A Custom phasing that does not add up is saved and becomes a needed-to-publish
// problem of the campaign (FR-022).
func (s *Service) SetCampaignBudgetPhasing(ctx context.Context, a actor.Actor, id string, in rollup.PhasingInput) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		c, plan, err := s.campaignAndPlan(ctx, id)
		if err != nil {
			return err
		}
		phasing, err := checkPhasingSave(in, c.Budget, plan, c.Meta.CheckVersion, campaignSingular)
		if err != nil {
			return err
		}
		if changes := phasingChanges(c.BudgetPhasing, phasing, c.Budget, c.Budget, plan); len(changes) > 0 {
			c.BudgetPhasing = phasing
			c.Touch(a, s.now())
			if err := s.repo.PutCampaign(ctx, c); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionChanged, Changes: changes,
			}); err != nil {
				return err
			}
		}
		view, err = s.campaignView(ctx, c.ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// SetInitiativeBudgetPhasing saves the phasing of the initiative id's budget, as SetCampaignBudgetPhasing does
// for a campaign.
func (s *Service) SetInitiativeBudgetPhasing(ctx context.Context, a actor.Actor, id string, in rollup.PhasingInput) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		i, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		c, plan, err := s.campaignAndPlan(ctx, i.CampaignID)
		if err != nil {
			return err
		}
		phasing, err := checkPhasingSave(in, i.Budget, plan, i.Meta.CheckVersion, initiativeSingular)
		if err != nil {
			return err
		}
		if changes := phasingChanges(i.BudgetPhasing, phasing, i.Budget, i.Budget, plan); len(changes) > 0 {
			i.BudgetPhasing = phasing
			i.Touch(a, s.now())
			if err := s.repo.PutInitiative(ctx, i); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, ParentID: c.ID, Action: history.ActionChanged, Changes: changes,
			}); err != nil {
				return err
			}
		}
		view, err = s.initiativeView(ctx, i.ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

// checkPhasingSave checks a phasing save of a record with budget in plan (data-model §3.3) and its version:
// every problem at once, then a version conflict.
func checkPhasingSave(in rollup.PhasingInput, budget *money.Money, plan PlanFacts, checkVersion func(int, string) error,
	noun string) (rollup.Phasing, error) {
	phasing, problems := rollup.ParsePhasing(in, amountOf(budget), plan.Year)
	checkVersionSent(&problems, in.Version)
	if err := problems.Err(); err != nil {
		return rollup.Phasing{}, err
	}
	return phasing, checkVersion(*in.Version, noun)
}

// phasingChanges lists the logged changes of a record's phasing — its method and each changed month — when
// the phasing itself changed; none otherwise (a budget change re-phases Evenly without a phasing entry).
func phasingChanges(before, after rollup.Phasing, budgetBefore, budgetAfter *money.Money, plan PlanFacts) []history.Change {
	if reflect.DeepEqual(before.Normalised(), after.Normalised()) {
		return nil
	}
	return rollup.PhasingChanges(before, after, amountOf(budgetBefore), amountOf(budgetAfter), plan.Year, plan.Currency)
}
