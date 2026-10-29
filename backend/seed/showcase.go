package seed

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"mapp/internal/activity"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
)

// showcaseJSON is one tree, plan down to line item, so the two showcase plans read as the plans they are.
//
//go:embed showcase.json
var showcaseJSON []byte

// WithShowcase adds two complete 2026 plans after the demo story, so no demo ID moves; the tests load without it.
func WithShowcase() Option {
	return func(o *options) { o.showcase = true }
}

const ShowcaseLoaded = "3 plans, 7 campaigns, 12 initiatives, 20 activities, 39 line items, 9 approval rounds"

type showcaseFile struct {
	Presets map[string]attributeSet
	Plans   []showcasePlan
}

type attributeSet struct {
	Choice  map[string]string
	Choices map[string][]string
	Split   map[string][]string
	YesNo   map[string]bool
	Text    map[string]string
}

type showcasePlan struct {
	Name, Model, Team, Owner, Budget, Status, GoalStage, GoalAmount, FinanceBudget, Description, Overspend string
	Year                                                                                                   int
	BusinessUnits                                                                                          []string
	SMP                                                                                                    bool
	Campaigns                                                                                              []showcaseCampaign
}

type showcaseCampaign struct {
	Name, Owner, Start, End, Budget string
	Attributes                      attributeSet
	Initiatives                     []showcaseInitiative
}

type showcaseInitiative struct {
	Name, Owner, Start, End, Budget string
	Attributes                      attributeSet
	Activities                      []showcaseActivity
}

type showcaseActivity struct {
	Name, Type, Status, Owner, Start, End, PlannedCost, Workflow, Preset, Revenue string
	INQ, MQL                                                                      int64
	Attributes                                                                    attributeSet
	Approval                                                                      *showcaseApproval
	LineItems                                                                     []showcaseLineItem
}

type showcaseApproval struct {
	SubmittedBy string
	DaysAgo     int
	Approvers   []string
}

type showcaseLineItem struct {
	Name, Currency, PlannedCost, Spread, SpendType, GLCode, Budget, Item string
}

type showcaseLists struct {
	presets    map[string]attributeSet
	plans      []demoPlan
	campaigns  []demoCampaign
	activities []demoActivity
	lineItems  []demoLineItem
	rounds     []demoRound
}

func (s attributeSet) apply(v *values) {
	for label, option := range s.Choice {
		v.choice(label, option)
	}
	for label, options := range s.Choices {
		v.choices(label, options...)
	}
	for label, shares := range s.Split {
		v.split(label, shares...)
	}
	for label, yes := range s.YesNo {
		v.yesNo(label, yes)
	}
	for label, text := range s.Text {
		v.text(label, text)
	}
}

func (d *demo) addShowcase(ctx context.Context) error {
	var file showcaseFile
	if err := json.Unmarshal(showcaseJSON, &file); err != nil {
		return fmt.Errorf("seed: showcase.json: %w", err)
	}
	lists := &showcaseLists{presets: file.Presets}
	for _, p := range file.Plans {
		lists.addPlan(p)
	}
	steps := []func(context.Context) error{
		func(ctx context.Context) error { return d.putPlans(ctx, lists.plans) },
		func(ctx context.Context) error { return d.putCampaigns(ctx, lists.campaigns) },
		func(ctx context.Context) error { return d.putActivities(ctx, lists.activities) },
		func(ctx context.Context) error { return d.putLineItems(ctx, lists.lineItems) },
		func(ctx context.Context) error { return d.putRounds(ctx, lists.rounds) },
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *showcaseLists) addPlan(p showcasePlan) {
	l.plans = append(l.plans, demoPlan{
		name: p.Name, year: p.Year, model: p.Model, team: p.Team, owner: p.Owner, budget: p.Budget,
		status: plan.Status(p.Status), businessUnits: p.BusinessUnits, smp: p.SMP, goalStage: plan.GoalStage(p.GoalStage),
		goalAmount: p.GoalAmount, financeBudget: p.FinanceBudget, description: p.Description, overspend: p.Overspend,
	})
	for _, c := range p.Campaigns {
		campaign := demoCampaign{plan: p.Name, name: c.Name, owner: c.Owner, start: c.Start, end: c.End, budget: c.Budget,
			attributes: c.Attributes.apply}
		for _, in := range c.Initiatives {
			campaign.initiatives = append(campaign.initiatives, demoInitiative{name: in.Name, owner: in.Owner, start: in.Start,
				end: in.End, budget: in.Budget, attributes: in.Attributes.apply})
			for _, a := range in.Activities {
				l.addActivity(demoActivity{plan: p.Name, campaign: c.Name, initiative: in.Name}, a)
			}
		}
		l.campaigns = append(l.campaigns, campaign)
	}
}

func (l *showcaseLists) addActivity(at demoActivity, a showcaseActivity) {
	preset := l.presets[a.Preset]
	at.name, at.status, at.activityType, at.owner = a.Name, activity.Status(a.Status), a.Type, a.Owner
	at.start, at.end, at.plannedCost, at.workflow = a.Start, a.End, a.PlannedCost, a.Workflow
	at.inq, at.typedMQL, at.typedRevenue = a.INQ, a.MQL, a.Revenue
	at.attributes = func(v *values) {
		preset.apply(v)
		a.Attributes.apply(v)
	}
	l.activities = append(l.activities, at)
	for _, li := range a.LineItems {
		l.lineItems = append(l.lineItems, demoLineItem{plan: at.plan, activity: a.Name, name: li.Name, currency: li.Currency,
			plannedCost: li.PlannedCost, method: lineitem.SpreadMethod(li.Spread), spendType: li.SpendType, glCode: li.GLCode,
			budget: li.Budget, item: li.Item, requestSwbs: true})
	}
	if a.Approval != nil {
		l.rounds = append(l.rounds, a.Approval.round(at.plan, a.Name))
	}
}

func (s showcaseApproval) round(planName, activityName string) demoRound {
	r := demoRound{plan: planName, activity: activityName, submittedBy: s.SubmittedBy,
		submittedAgo: time.Duration(s.DaysAgo) * day}
	for i, by := range s.Approvers {
		r.decisions = append(r.decisions, demoDecision{by: by, ago: time.Duration(s.DaysAgo-1-i) * day})
	}
	return r
}
