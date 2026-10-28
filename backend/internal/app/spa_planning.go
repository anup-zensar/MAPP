package app

import (
	"context"

	"mapp/internal/allwork"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/spa"
	"mapp/internal/stores"
)

// The SP&A view's dimensions (013 D-336): activity type, then the active choice and yes/no attributes of
// activities and line items, Channel / audience first. The label is Shell's attribute in Settings.
const (
	spaChannelLabel      = "Channel / audience"
	spaActivityTypeKey   = "activityType"
	spaActivityTypeLabel = "Activity type"
	spaAttributePrefix   = "attr:"
)

// spaPlanning reads the plan through All work's plan reader, so the SP&A view gets the very values the plan
// tree, All work and Budget by month use: USD amounts, 007's spreads, 006's display status, 016's attributes.
type spaPlanning struct {
	plans  stores.PlanStore
	rates  func(ctx context.Context) (money.Rates, error)
	source allWorkSource
	now    clock.Clock
}

func (p spaPlanning) Plan(ctx context.Context, planID string) (spa.Plan, bool, error) {
	stored, found, err := p.plans.GetInScope(ctx, authz.ScopeOf(ctx), planID)
	if err != nil || !found {
		return spa.Plan{}, false, err
	}
	rates, err := p.rates(ctx)
	return spaPlanOf(stored, rates), true, err
}

func (p spaPlanning) Records(ctx context.Context, pl spa.Plan) ([]spa.Record, error) {
	data, err := p.source.PlanYear(ctx, pl.ID, pl.Year, date.Today(p.now()))
	if err != nil {
		return nil, err
	}
	labels, err := p.labels(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]spa.Record, 0, len(data.Nodes))
	for _, n := range data.Nodes {
		if n.Level != allwork.LevelPlan {
			records = append(records, labels.record(n))
		}
	}
	return records, nil
}

func (p spaPlanning) Dimensions(ctx context.Context) ([]spa.Dimension, error) {
	attributes, err := p.source.Attributes(ctx)
	if err != nil {
		return nil, err
	}
	activityType := spa.Dimension{Key: spaActivityTypeKey, Label: spaActivityTypeLabel, Level: spa.LevelActivity}
	var channel, others []spa.Dimension
	for _, a := range attributes {
		if !groupable(a) {
			continue
		}
		d := spa.Dimension{Key: spaAttributePrefix + a.ID, Label: a.Label, Level: spa.Level(a.Level)}
		if d.Level == spa.LevelActivity && a.Label == spaChannelLabel {
			d.Channel = true
			channel = append(channel, d)
			continue
		}
		others = append(others, d)
	}
	return append(append(channel, activityType), others...), nil
}

// groupable is an active choice or yes/no attribute of activities or line items (All work's Breakdown, D-270).
func groupable(a allwork.Attribute) bool {
	switch a.Type {
	case allwork.TypeSingleChoice, allwork.TypeMultipleChoice, allwork.TypeMultipleChoiceSplit, allwork.TypeYesNo:
		return a.Active && (a.Level == allwork.LevelActivity || a.Level == allwork.LevelLineItem)
	}
	return false
}

// spaLabels names what the nodes refer to by ID: activity types and attribute options.
type spaLabels struct {
	types   map[string]string
	options map[string]string
	yesNo   map[allwork.Level][]string // the yes/no attributes of each level
}

func (p spaPlanning) labels(ctx context.Context) (spaLabels, error) {
	l := spaLabels{types: map[string]string{}, options: map[string]string{}, yesNo: map[allwork.Level][]string{}}
	types, err := p.source.settings.ActivityTypes(ctx)
	if err != nil {
		return l, err
	}
	for _, t := range types {
		l.types[t.ID] = t.Name
	}
	attributes, err := p.source.Attributes(ctx)
	for _, a := range attributes {
		if a.Type == allwork.TypeYesNo {
			l.yesNo[a.Level] = append(l.yesNo[a.Level], a.ID)
		}
		for _, o := range a.Options {
			l.options[o.ID] = o.Label
		}
	}
	return l, err
}

func (l spaLabels) record(n allwork.Node) spa.Record {
	r := spa.Record{
		Level: spa.Level(n.Level), ID: n.ID, ParentID: n.ParentID, Order: n.Order, Name: n.Name, StartDate: n.StartDate,
		Months: n.Months, PlannedCost: n.PlannedCost, Values: l.values(n),
	}
	if n.Owner != nil {
		r.Owner = n.Owner.Name
	}
	switch n.Level {
	case allwork.LevelActivity:
		r.Status, r.ActivityType = n.Status, l.types[n.ActivityTypeID]
	case allwork.LevelCampaign:
		r.Budget = n.Budget
	case allwork.LevelLineItem:
		_, r.Linked = n.Fields["lineItem.financeBudgetItem"]
		for _, m := range n.MonthsUSD {
			r.Spread = append(r.Spread, spa.MonthAmount{Month: m.Month, USD: m.USD})
		}
	case allwork.LevelPlan, allwork.LevelInitiative, allwork.LevelUnassigned:
	}
	return r
}

// values are the node's attribute values by dimension key; an empty yes/no reads as No, as everywhere.
func (l spaLabels) values(n allwork.Node) map[string][]spa.Value {
	values := map[string][]spa.Value{}
	if name := l.types[n.ActivityTypeID]; n.Level == allwork.LevelActivity && name != "" {
		values[spaActivityTypeKey] = []spa.Value{{Label: name}}
	}
	for _, id := range l.yesNo[n.Level] {
		values[spaAttributePrefix+id] = []spa.Value{{Label: yesNoLabel(n.Attrs[id].Yes)}}
	}
	for id, v := range n.Attrs {
		key := spaAttributePrefix + id
		for _, option := range v.OptionIDs {
			value := spa.Value{Label: l.options[option]}
			if share, split := v.Shares[option]; split {
				value.Share = &share
			}
			values[key] = append(values[key], value)
		}
	}
	return values
}

func yesNoLabel(yes *bool) string {
	if yes != nil && *yes {
		return "Yes"
	}
	return "No"
}
