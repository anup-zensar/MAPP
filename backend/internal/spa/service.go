package spa

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

const (
	msgNoFinanceAccess = "You don't have access to financial data in this plan."
	msgPlannedOutside  = "%s of planned cost falls outside %d and is not counted."
	msgSpendOutside    = "Transactions posted outside %d are not counted: committed %s, actual spend %s."
	msgPlanEmpty       = "This plan has no activities yet."
	msgCampaignEmpty   = "This campaign has no activities yet."
)

type Service struct {
	tx   Transactor
	deps Deps
}

func NewService(tx Transactor, deps Deps) *Service {
	return &Service{tx: tx, deps: deps}
}

// inputs are what one view is worked out from.
type inputs struct {
	plan      Plan
	tree      tree
	scope     *Record // nil for the whole plan
	campaigns []*Record
	dimension Dimension
	dims      []Dimension
	sees      bool
	show      shower
}

// View is the SP&A view of the plan planID or of its campaign q.CampaignID (FR-005 – FR-022), read in one
// transaction: a plan the person does not see, or a campaign not in it, is not found.
func (s *Service) View(ctx context.Context, a actor.Actor, planID string, q Query) (View, error) {
	var view View
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		in, err := s.inputsOf(ctx, a, planID, q)
		if err != nil {
			return err
		}
		view, err = s.build(ctx, in)
		return err
	})
	return view, err
}

func (s *Service) inputsOf(ctx context.Context, a actor.Actor, planID string, q Query) (inputs, error) {
	in, err := s.scopeOf(ctx, planID, q.CampaignID)
	if err != nil {
		return inputs{}, err
	}
	if in.dims, err = s.deps.Planning.Dimensions(ctx); err != nil {
		return inputs{}, err
	}
	if in.dimension, err = chosen(in.dims, q.Dimension); err != nil {
		return inputs{}, err
	}
	if in.sees, err = s.deps.Finance.SeesFinance(ctx, a, in.plan.ID); err != nil {
		return inputs{}, err
	}
	display, err := s.deps.Displays.Display(ctx, a.ID)
	in.show = shower{display: display}
	return in, err
}

// scopeOf reads the plan and its records, and finds the campaign campaignID among them ("" for none).
func (s *Service) scopeOf(ctx context.Context, planID, campaignID string) (inputs, error) {
	p, found, err := s.deps.Planning.Plan(ctx, planID)
	if err != nil || !found {
		return inputs{}, notFound(err, "plan")
	}
	records, err := s.deps.Planning.Records(ctx, p)
	if err != nil {
		return inputs{}, err
	}
	in := inputs{plan: p, tree: newTree(records)}
	in.campaigns = in.tree.kids(p.ID, LevelCampaign)
	if campaignID == "" {
		return in, nil
	}
	if in.scope = in.tree.byID[campaignID]; in.scope == nil || in.scope.Level != LevelCampaign {
		return inputs{}, errs.NotFound(authz.NotFoundMessage("campaign"))
	}
	return in, nil
}

func notFound(err error, record string) error {
	if err != nil {
		return err
	}
	return errs.NotFound(authz.NotFoundMessage(record))
}

// chosen is the dimension key asks for, the first offered for "".
func chosen(dims []Dimension, key string) (Dimension, error) {
	for _, d := range dims {
		if d.Key == key || key == "" {
			return d, nil
		}
	}
	if key == "" {
		return Dimension{Key: "", Label: ""}, nil
	}
	var problems errs.Problems
	problems.Add(fieldDimension, msgChooseListed)
	return Dimension{}, problems.Err()
}

func (s *Service) build(ctx context.Context, in inputs) (View, error) {
	acts, outside, err := s.activitiesOf(ctx, in)
	if err != nil {
		return View{}, err
	}
	view := View{
		Plan:  PlanRef{ID: in.plan.ID, Name: in.plan.Name, Year: in.plan.Year, Status: in.plan.Status, BCPIR: in.plan.BCPIR},
		Scope: RecordRef{Type: LevelPlan, ID: in.plan.ID, Name: in.plan.Name}, Campaigns: refsOf(in.campaigns),
		Currency: in.show.display.Currency(), FinanceAccess: in.sees, Headline: headlineOf(acts, in.sees, in.show),
		Periods: periodsOf(acts, in.plan.Year, in.sees, in.show), Dimension: breakdownOf(acts, in.dimension, in.show),
		Dimensions: choicesOf(in.dims), Notes: notesOf(in, acts, outside),
	}
	budget := in.plan.Budget
	if in.scope != nil {
		view.Scope, budget = refOf(in.scope), in.scope.Budget
	}
	view.Budget = budgetPanelOf(acts, budget, in.sees, in.show)
	if err := s.addFinanceAndPurchases(ctx, in, acts, &view); err != nil {
		return View{}, err
	}
	view.EmptyMessage = emptyMessage(in, acts)
	return view, nil
}

// activitiesOf returns the scope's activities with their figures, and the transactions posted in other years.
func (s *Service) activitiesOf(ctx context.Context, in inputs) ([]*activity, Spend, error) {
	campaigns := in.campaigns
	if in.scope != nil {
		campaigns = []*Record{in.scope}
	}
	acts := in.tree.activitiesUnder(campaigns)
	byLineItem := map[string]map[string]Spend{}
	if in.sees {
		var err error
		if byLineItem, err = s.deps.Finance.SpendByMonth(ctx); err != nil {
			return nil, Spend{}, err
		}
	}
	var outside Spend
	for _, a := range acts {
		a.placePlanned(in.plan.Year)
		outside = outside.plus(a.placeSpend(in.plan.Year, byLineItem))
	}
	return acts, outside, nil
}

func (s *Service) addFinanceAndPurchases(ctx context.Context, in inputs, acts []*activity, view *View) error {
	raise, err := s.deps.Purchases.CanRaise(ctx, in.plan)
	if err != nil {
		return err
	}
	channel := ""
	for _, d := range in.dims {
		if d.Channel {
			channel = d.Key
		}
	}
	view.Activities = activityRows(acts, rowFacts{sees: in.sees, raise: raise, channel: channel, show: in.show})
	prs, err := s.deps.Purchases.Drawn(ctx, lineItemIDsOf(acts))
	if err != nil {
		return err
	}
	view.Procurement = procurementOf(acts, prs, in.sees, in.show)
	if !in.sees {
		view.FinanceNote = text(msgNoFinanceAccess)
		return nil
	}
	view.FinanceDataAsOf, err = s.deps.Finance.DataAsOf(ctx)
	return err
}

func notesOf(in inputs, acts []*activity, outside Spend) []string {
	notes := []string{}
	if planned := sumOf(acts, func(a *activity) money.Amount { return a.outside }); !planned.IsZero() {
		notes = append(notes, fmt.Sprintf(msgPlannedOutside, in.show.money(planned).Format(), in.plan.Year))
	}
	if !outside.Committed.IsZero() || !outside.Actual.IsZero() {
		notes = append(notes, fmt.Sprintf(msgSpendOutside, in.plan.Year, in.show.money(outside.Committed).Format(),
			in.show.money(outside.Actual).Format()))
	}
	return notes
}

func emptyMessage(in inputs, acts []*activity) *string {
	switch {
	case len(acts) > 0:
		return nil
	case in.scope != nil:
		return text(msgCampaignEmpty)
	}
	return text(msgPlanEmpty)
}

func refsOf(records []*Record) []RecordRef {
	refs := make([]RecordRef, 0, len(records))
	for _, r := range records {
		refs = append(refs, refOf(r))
	}
	return refs
}

func choicesOf(dims []Dimension) []DimensionChoice {
	list := make([]DimensionChoice, 0, len(dims))
	for _, d := range dims {
		list = append(list, DimensionChoice{Key: d.Key, Label: d.Label})
	}
	return list
}
