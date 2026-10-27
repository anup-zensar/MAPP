package campaign

import (
	"context"
	"maps"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// Service holds the rules of campaigns and initiatives (005 plan, "Service"). Every method takes the actor,
// runs in one transaction and reports every problem of a save at once. No method checks roles: every
// signed-in person may do everything (FR-031, D-004); the actor is where those checks will go.
type Service struct {
	tx      store.Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	deps    Deps
	// displayRates reads the reader's display currency (UseDisplayRates, D-220); nil shows USD.
	displayRates DisplayRates
}

// NewService returns the campaigns service. It keeps the records in repo, gives them readable IDs, logs their
// changes, deletes them, works in transactions and reads the time with tools, and reads what it needs from other
// features from deps.
func NewService(tools history.Tools, repo Store, deps Deps) *Service {
	deps.Plans = inPlanCurrency{deps.Plans}
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now, deps: deps}
}

// UpdateCampaign saves the campaign form of the campaign id (FR-018), whatever the plan's status. The
// version must be the one the person started from. New dates must still contain every initiative's dates
// (FR-007; checked only when the dates change, D-095). A save that changes nothing answers the campaign as
// it is, with no new version and no log entry; otherwise one "changed" entry lists every change (FR-030).
func (s *Service) UpdateCampaign(ctx context.Context, a actor.Actor, id string, in CampaignUpdate) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, found, err := s.repo.GetCampaign(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(campaignGoneMessage)
		}
		if in.Version != nil && *in.Version >= 1 {
			if err := stored.CheckVersion(*in.Version, campaignSingular); err != nil {
				return err
			}
		}
		plan, found, err := s.deps.Plans.Plan(ctx, stored.PlanID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		siblings, err := s.repo.CampaignsOfPlan(ctx, stored.PlanID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
			plan.Currency, func(name string) string { return campaignNameTaken(siblings, plan, name, stored.ID) })
		if err := s.checkOwner(ctx, &problems, f.OwnerID, stored.OwnerID); err != nil {
			return err
		}
		if datesChanged(stored.StartDate, stored.EndDate, f.StartDate, f.EndDate) && f.StartDate != nil && f.EndDate != nil {
			initiatives, err := s.repo.InitiativesOfCampaign(ctx, stored.ID)
			if err != nil {
				return err
			}
			children := make([]dated, 0, len(initiatives))
			for _, i := range initiatives {
				children = append(children, dated{name: i.Name, start: i.StartDate, end: i.EndDate})
			}
			checkChildrenInside(&problems, "initiatives", children, *f.StartDate, *f.EndDate)
		}
		values, err := s.checkAttributes(ctx, &problems, settings.LevelCampaign, plan, in.Attributes, stored.Attributes)
		if err != nil {
			return err
		}
		checkVersionSent(&problems, in.Version)
		if err := problems.Err(); err != nil {
			return err
		}

		updated := stored
		updated.Name, updated.Description, updated.OwnerID = f.Name, f.Description, f.OwnerID
		updated.StartDate, updated.EndDate, updated.Budget, updated.Attributes = f.StartDate, f.EndDate, f.Budget, values
		if updated.Budget == nil { // a removed budget removes its phasing (016 FR-022)
			updated.BudgetPhasing = rollup.NotPhased
		}
		withoutValues := func(c Campaign) Campaign { c.Attributes, c.BudgetPhasing = nil, rollup.Phasing{}; return c }
		changes, err := recordChanges(withoutValues(stored), withoutValues(updated), stored.Attributes, updated.Attributes)
		if err != nil {
			return err
		}
		changes = append(changes, phasingChanges(stored.BudgetPhasing, updated.BudgetPhasing, stored.Budget, updated.Budget, plan)...)
		if len(changes) > 0 {
			updated.Touch(a, s.now())
			if err := s.repo.PutCampaign(ctx, updated); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: campaignKind, RecordID: updated.ID, RecordName: updated.Name, Action: history.ActionChanged, Changes: changes,
			}); err != nil {
				return err
			}
		}
		view, err = s.campaignView(ctx, id)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// UpdateInitiative saves the initiative form of the initiative id (FR-018). When the dates change (D-095)
// they must lie inside the campaign's (R3, FR-006) and still contain every activity's dates — only the
// start of an ongoing activity (FR-007).
func (s *Service) UpdateInitiative(ctx context.Context, a actor.Actor, id string, in InitiativeUpdate) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		stored, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		if in.Version != nil && *in.Version >= 1 {
			if err := stored.CheckVersion(*in.Version, initiativeSingular); err != nil {
				return err
			}
		}
		campaign, found, err := s.repo.GetCampaign(ctx, stored.CampaignID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(campaignGoneMessage)
		}
		plan, found, err := s.deps.Plans.Plan(ctx, campaign.PlanID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		siblings, err := s.repo.InitiativesOfCampaign(ctx, campaign.ID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
			plan.Currency, func(name string) string { return initiativeNameTaken(siblings, campaign, name, stored.ID) })
		if err := s.checkOwner(ctx, &problems, f.OwnerID, stored.OwnerID); err != nil {
			return err
		}
		if datesChanged(stored.StartDate, stored.EndDate, f.StartDate, f.EndDate) {
			checkInsideCampaign(&problems, campaign, f.StartDate, f.EndDate)
			if f.StartDate != nil && f.EndDate != nil {
				if err := s.checkActivitiesInside(ctx, &problems, stored.ID, *f.StartDate, *f.EndDate); err != nil {
					return err
				}
			}
		}
		values, err := s.checkAttributes(ctx, &problems, settings.LevelInitiative, plan, in.Attributes, stored.Attributes)
		if err != nil {
			return err
		}
		checkVersionSent(&problems, in.Version)
		if err := problems.Err(); err != nil {
			return err
		}

		updated := stored
		updated.Name, updated.Description, updated.OwnerID = f.Name, f.Description, f.OwnerID
		updated.StartDate, updated.EndDate, updated.Budget, updated.Attributes = f.StartDate, f.EndDate, f.Budget, values
		if updated.Budget == nil { // a removed budget removes its phasing (016 FR-022)
			updated.BudgetPhasing = rollup.NotPhased
		}
		withoutValues := func(i Initiative) Initiative { i.Attributes, i.BudgetPhasing = nil, rollup.Phasing{}; return i }
		changes, err := recordChanges(withoutValues(stored), withoutValues(updated), stored.Attributes, updated.Attributes)
		if err != nil {
			return err
		}
		changes = append(changes, phasingChanges(stored.BudgetPhasing, updated.BudgetPhasing, stored.Budget, updated.Budget, plan)...)
		if len(changes) > 0 {
			updated.Touch(a, s.now())
			if err := s.repo.PutInitiative(ctx, updated); err != nil {
				return err
			}
			if _, err := s.history.Log(ctx, a, history.LogInput{
				Kind: initiativeKind, RecordID: updated.ID, RecordName: updated.Name, Action: history.ActionChanged, Changes: changes,
			}); err != nil {
				return err
			}
		}
		view, err = s.initiativeView(ctx, id)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

// checkOwner adds a problem when the owner is newly chosen and is not an active person (FR-002). An owner
// the record already has stays, even when inactive (spec edge case).
func (s *Service) checkOwner(ctx context.Context, problems *errs.Problems, ownerID, storedOwnerID *string) error {
	if ownerID == nil || (storedOwnerID != nil && *storedOwnerID == *ownerID) {
		return nil
	}
	person, found, err := s.deps.People.Get(ctx, *ownerID)
	if err != nil {
		return err
	}
	if !found || !person.Active {
		problems.Add("ownerId", chooseActivePersonMessage)
	}
	return nil
}

// checkAttributes checks the Shell attribute values sent for level with Settings' rules (FR-002, FR-009,
// R8, R9), adding their problems to problems, and returns the values to store, never nil.
func (s *Service) checkAttributes(ctx context.Context, problems *errs.Problems, level settings.Level, plan PlanFacts,
	values, stored settings.AttributeValues) (settings.AttributeValues, error) {
	businessUnit, err := s.deps.Settings.BusinessUnit(ctx, plan.Attributes)
	if err != nil {
		return nil, err
	}
	result, err := s.deps.Settings.CheckAttributes(ctx, level, settings.AttributeInput{
		Values: orEmpty(values), Previous: orEmpty(stored), BusinessUnit: businessUnit,
	})
	if err != nil {
		return nil, err
	}
	*problems = append(*problems, result.Problems...)
	if result.Values != nil {
		return result.Values, nil
	}
	return maps.Clone(orEmpty(values)), nil
}

// checkActivitiesInside applies FR-007 to the activities of the initiative id for new dates start – end.
func (s *Service) checkActivitiesInside(ctx context.Context, problems *errs.Problems, id string, start, end date.Date) error {
	activities, err := s.deps.Activities.Summaries(ctx, []string{id})
	if err != nil {
		return err
	}
	children := make([]dated, 0, len(activities))
	for _, a := range activities {
		children = append(children, dated{name: a.Name, start: a.StartDate, end: a.EndDate, ongoing: a.Ongoing})
	}
	checkChildrenInside(problems, "activities", children, start, end)
	return nil
}
