package campaigns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// Service holds the rules of campaigns and initiatives (005 plan, "Service"). Every method takes the actor,
// runs in one transaction and reports every problem of a save at once. No method checks roles: every
// signed-in person may do everything (FR-031, D-004); the actor is where those checks will go.
type Service struct {
	tx      store.Transactor
	repo    Repository
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	deps    Deps
	// displayRates reads the reader's display currency (UseDisplayRates, D-220); nil shows USD.
	displayRates DisplayRates
}

// NewService returns the campaigns service. It keeps the records in repo, gives them readable IDs with
// generator, logs their changes and deletes them through hist, works in transactions of tx, reads the time
// from now and what it needs from other features from deps.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now clock.Clock, deps Deps) *Service {
	deps.Plans = inPlanCurrency{deps.Plans}
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, now: now, deps: deps}
}

// inPlanCurrency reads plans with their plan currency set: USD for a plan known without one (016 data-model
// §4.5), so every amount of the tree is labelled.
type inPlanCurrency struct {
	PlanSource
}

// Plan returns the plan id with its currency.
func (p inPlanCurrency) Plan(ctx context.Context, id string) (PlanFacts, bool, error) {
	plan, found, err := p.PlanSource.Plan(ctx, id)
	return withCurrency(plan), found, err
}

// PlansOfYear returns the plans of year with their currencies.
func (p inPlanCurrency) PlansOfYear(ctx context.Context, year int) ([]PlanFacts, error) {
	plans, err := p.PlanSource.PlansOfYear(ctx, year)
	for i := range plans {
		plans[i] = withCurrency(plans[i])
	}
	return plans, err
}

// withCurrency returns plan with USD as its currency when it has none.
func withCurrency(plan PlanFacts) PlanFacts {
	if plan.Currency == "" {
		plan.Currency = money.BaseCurrency
	}
	return plan
}

// ListPlanCampaigns returns the campaigns of the plan planID, each with its initiatives, in tree order
// (FR-020), and the plan's totals (FR-022).
func (s *Service) ListPlanCampaigns(ctx context.Context, a actor.Actor, planID string) (PlanCampaigns, error) {
	var result PlanCampaigns
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		plan, found, err := s.deps.Plans.Plan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		result, err = s.planCampaigns(ctx, plan)
		return err
	})
	if err != nil {
		return PlanCampaigns{}, err
	}
	return s.shownTree(ctx, a, result)
}

// GetCampaign returns the campaign id with its initiatives and every derived value (FR-016).
func (s *Service) GetCampaign(ctx context.Context, a actor.Actor, id string) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.campaignView(ctx, id)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
}

// GetInitiative returns the initiative id with every derived value (FR-016).
func (s *Service) GetInitiative(ctx context.Context, a actor.Actor, id string) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		var err error
		view, err = s.initiativeView(ctx, id)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
}

// CreateCampaign adds a campaign to the plan planID (FR-001 – FR-005). Only the name is required; the API
// stores what it receives (D-093): the defaults of FR-003 are filled in by the form. Dates outside the
// plan's year are allowed and only flagged (FR-008); a budget larger than what remains is allowed (D-015).
// The campaign gets a readable ID (FR-004) and a "created" log entry (FR-030).
func (s *Service) CreateCampaign(ctx context.Context, a actor.Actor, planID string, in CampaignInput) (CampaignView, error) {
	var view CampaignView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		plan, found, err := s.deps.Plans.Plan(ctx, planID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(planGoneMessage)
		}
		siblings, err := s.repo.CampaignsOfPlan(ctx, planID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
			plan.Currency, func(name string) string { return campaignNameTaken(siblings, plan, name, "") })
		if err := s.checkOwner(ctx, &problems, f.OwnerID, nil); err != nil {
			return err
		}
		values, err := s.checkAttributes(ctx, &problems, settings.LevelCampaign, plan, in.Attributes, nil)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixCampaign)
		if err != nil {
			return err
		}
		c := Campaign{
			ID: id, PlanID: planID, Name: f.Name, Description: f.Description, OwnerID: f.OwnerID,
			StartDate: f.StartDate, EndDate: f.EndDate, Budget: f.Budget, Attributes: values, BudgetPhasing: rollup.NotPhased,
			Meta: record.NewMeta(a, s.now()),
		}
		if err := s.repo.PutCampaign(ctx, c); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: campaignKind, RecordID: c.ID, RecordName: c.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.campaignView(ctx, c.ID)
		return err
	})
	if err != nil {
		return CampaignView{}, err
	}
	return s.shownCampaign(ctx, a, view)
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

// DeleteCampaign moves the campaign id, with its initiatives and everything below them, to the recycle bin
// (FR-028), and returns the recycle-bin entry so that the person can undo the delete (FR-029).
func (s *Service) DeleteCampaign(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetCampaign(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(campaignGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, campaignKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// CreateInitiative adds an initiative to the campaign campaignID (FR-001 – FR-006). Only the name is
// required; its dates must lie inside the campaign's when the campaign has both (R3, FR-006).
func (s *Service) CreateInitiative(ctx context.Context, a actor.Actor, campaignID string, in InitiativeInput) (InitiativeView, error) {
	var view InitiativeView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		campaign, found, err := s.repo.GetCampaign(ctx, campaignID)
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
		siblings, err := s.repo.InitiativesOfCampaign(ctx, campaignID)
		if err != nil {
			return err
		}

		var problems errs.Problems
		f := checkForm(&problems, form{in.Name, in.Description, in.OwnerID, in.StartDate, in.EndDate, in.Budget, in.Attributes},
			plan.Currency, func(name string) string { return initiativeNameTaken(siblings, campaign, name, "") })
		if err := s.checkOwner(ctx, &problems, f.OwnerID, nil); err != nil {
			return err
		}
		checkInsideCampaign(&problems, campaign, f.StartDate, f.EndDate)
		values, err := s.checkAttributes(ctx, &problems, settings.LevelInitiative, plan, in.Attributes, nil)
		if err != nil {
			return err
		}
		if err := problems.Err(); err != nil {
			return err
		}

		id, err := s.ids.Next(ctx, ids.PrefixInitiative)
		if err != nil {
			return err
		}
		i := Initiative{
			ID: id, CampaignID: campaignID, Name: f.Name, Description: f.Description, OwnerID: f.OwnerID,
			StartDate: f.StartDate, EndDate: f.EndDate, Budget: f.Budget, Attributes: values, BudgetPhasing: rollup.NotPhased,
			Meta: record.NewMeta(a, s.now()),
		}
		if err := s.repo.PutInitiative(ctx, i); err != nil {
			return err
		}
		if _, err := s.history.Log(ctx, a, history.LogInput{
			Kind: initiativeKind, RecordID: i.ID, RecordName: i.Name, Action: history.ActionCreated,
		}); err != nil {
			return err
		}
		view, err = s.initiativeView(ctx, i.ID)
		return err
	})
	if err != nil {
		return InitiativeView{}, err
	}
	return s.shownInitiative(ctx, a, view)
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

// DeleteInitiative moves the initiative id, with its activities and line items, to the recycle bin
// (FR-028), and returns the recycle-bin entry so that the person can undo the delete (FR-029).
func (s *Service) DeleteInitiative(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetInitiative(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(initiativeGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, initiativeKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// campaignView builds the view of the campaign id inside the caller's transaction: its plan's whole tree is
// built, because the campaign's remaining plan budget depends on every other campaign of the plan.
func (s *Service) campaignView(ctx context.Context, id string) (CampaignView, error) {
	c, found, err := s.repo.GetCampaign(ctx, id)
	if err != nil {
		return CampaignView{}, err
	}
	if !found {
		return CampaignView{}, errs.NotFound(campaignGoneMessage)
	}
	tree, err := s.treeOf(ctx, c.PlanID, campaignGoneMessage)
	if err != nil {
		return CampaignView{}, err
	}
	for _, view := range tree.Items {
		if view.ID == id {
			return view, nil
		}
	}
	return CampaignView{}, errs.NotFound(campaignGoneMessage)
}

// initiativeView builds the view of the initiative id inside the caller's transaction.
func (s *Service) initiativeView(ctx context.Context, id string) (InitiativeView, error) {
	i, found, err := s.repo.GetInitiative(ctx, id)
	if err != nil {
		return InitiativeView{}, err
	}
	if !found {
		return InitiativeView{}, errs.NotFound(initiativeGoneMessage)
	}
	campaign, err := s.campaignView(ctx, i.CampaignID)
	if err != nil {
		return InitiativeView{}, err
	}
	for _, view := range campaign.Initiatives {
		if view.ID == id {
			return view, nil
		}
	}
	return InitiativeView{}, errs.NotFound(initiativeGoneMessage)
}

// treeOf builds the campaign tree of the plan planID; a plan that is gone answers notFound, the message of
// the record that was asked for.
func (s *Service) treeOf(ctx context.Context, planID, notFound string) (PlanCampaigns, error) {
	plan, found, err := s.deps.Plans.Plan(ctx, planID)
	if err != nil {
		return PlanCampaigns{}, err
	}
	if !found {
		return PlanCampaigns{}, errs.NotFound(notFound)
	}
	return s.planCampaigns(ctx, plan)
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

// checkVersionSent adds the version problem when an update sends no version, or one below 1: the version
// check cannot work then (FR-018).
func checkVersionSent(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

// campaignNameTaken returns the "already has" message when another campaign of plan than exceptID has name,
// ignoring letter case (FR-005); "" otherwise.
func campaignNameTaken(siblings []Campaign, plan PlanFacts, name, exceptID string) string {
	if other, taken := campaignNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(campaignNameTakenMessage, plan.Name, other.Name)
	}
	return ""
}

// initiativeNameTaken returns the "already has" message when another initiative of campaign than exceptID
// has name, ignoring letter case (FR-005); "" otherwise.
func initiativeNameTaken(siblings []Initiative, campaign Campaign, name, exceptID string) string {
	if other, taken := initiativeNamed(siblings, name, exceptID); taken {
		return fmt.Sprintf(initiativeNameTakenMessage, campaign.Name, other.Name)
	}
	return ""
}

// recordChanges lists what a save changes (data model §8): every changed stored field of the record
// without its Shell attributes (history.Diff), then one change per changed attribute, field
// "attributes.<id>", sorted by attribute id. Values that are empty — missing, null, "", [] or {} — count as
// the same.
func recordChanges(before, after any, beforeValues, afterValues settings.AttributeValues) ([]history.Change, error) {
	changes, err := history.Diff(before, after)
	if err != nil {
		return nil, err
	}
	attributeIDs := map[string]bool{}
	for id := range beforeValues {
		attributeIDs[id] = true
	}
	for id := range afterValues {
		attributeIDs[id] = true
	}
	for _, id := range slices.Sorted(maps.Keys(attributeIDs)) {
		old, updated := compactJSON(beforeValues[id]), compactJSON(afterValues[id])
		if bytes.Equal(old, updated) || (emptyJSON(old) && emptyJSON(updated)) {
			continue
		}
		changes = append(changes, history.Change{Field: "attributes." + id, Old: old, New: updated})
	}
	return changes, nil
}

// compactJSON returns value without spaces, so that values sent with spaces compare equal; null for a
// missing value.
func compactJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage("null")
	}
	var b bytes.Buffer
	if err := json.Compact(&b, value); err != nil {
		return value // not JSON: compared as it is
	}
	return b.Bytes()
}

// emptyJSON reports whether a stored value counts as empty: null, "", [] or {}.
func emptyJSON(value json.RawMessage) bool {
	switch string(value) {
	case "null", `""`, "[]", "{}":
		return true
	}
	return false
}
