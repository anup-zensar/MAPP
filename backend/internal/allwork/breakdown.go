package allwork

import (
	"cmp"
	"context"
	"slices"
	"strconv"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// The Breakdown (feature 016, FR-040 – FR-042, data-model §3.5, D-270): the planned cost of the activities that
// match the filters, grouped by one grouping — a choice or yes/no Shell attribute of any level, or Plan, Plan
// type, Plan currency, Year, Team, Business unit, Activity type, Owner — and optionally then by a second. It
// is also the plan page's Planned cost by card (one plan, amounts in its plan currency).

// The messages of the Breakdown (016 plan "User-visible messages").
const (
	msgChooseBy        = "Choose what to break down by."
	msgChooseThenBy    = "Choose another grouping than the first."
	msgOnePlanForPlan  = "Choose one plan to show amounts in its currency."
	noteCountsInEach   = "Activities with several values count in each group; the total counts them once."
	noteLineItemsNoPC  = "Includes %s on line items of activities without a planned cost."
	labelNotFilledIn   = "Not filled in"
	labelNotOnLineItem = "Not on a line item"
	labelNoOwner       = "No owner"
	labelTotal         = "Total"
)

// The keys of the groups that are not values.
const (
	keyNotFilledIn   = "notFilledIn"
	keyNotOnLineItem = "notOnLineItem"
	keyNoOwner       = "noOwner"
	keyTotal         = "total"
)

// The built-in groupings (FR-040).
const (
	byPlan         = "plan"
	byPlanType     = "planType"
	byPlanCurrency = "planCurrency"
	byYear         = "year"
	byTeam         = "team"
	byBusinessUnit = "businessUnit"
	byActivityType = "activityType"
	byOwner        = "owner"
	attrPrefix     = "attr:"
)

// Breakdown returns the Breakdown of q for a (FR-040 – FR-042), read in one transaction: by is needed, thenBy
// must differ from it, and amountsIn=plan needs exactly one plan. Every problem is reported at once.
func (s *Service) Breakdown(ctx context.Context, a actor.Actor, q Query) (Breakdown, error) {
	var result Breakdown
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		in, err := s.input(ctx, a, q)
		if err != nil {
			return err
		}
		result, err = breakdownOf(in)
		return err
	})
	if err != nil {
		return Breakdown{}, err
	}
	return result, nil
}

// breakdownOf works out the Breakdown of in (data-model §3.5).
func breakdownOf(in input) (Breakdown, error) {
	groupings := groupingsOf(in.attributes)
	q := in.query
	var problems errs.Problems
	by, found := groupingNamed(groupings, q.By)
	if !found {
		problems.Add("by", msgChooseBy)
	}
	var thenBy *Grouping
	if q.ThenBy != "" {
		g, found := groupingNamed(groupings, q.ThenBy)
		switch {
		case !found:
			problems.Add("thenBy", msgChooseListed)
		case g.Key == q.By:
			problems.Add("thenBy", msgChooseThenBy)
		default:
			thenBy = &g
		}
	}
	in.query.View = ViewGrid
	b := build(in)
	planCurrency := ""
	if q.AmountsIn == AmountsInPlan {
		if len(b.query.Plans) != 1 {
			problems.Add("amountsIn", msgOnePlanForPlan)
		} else if p, found := b.tree.byID[b.query.Plans[0]]; found {
			planCurrency = cmp.Or(p.n.PlanCurrency, money.BaseCurrency)
		}
	}
	if err := problems.Err(); err != nil {
		return Breakdown{}, err
	}
	w := breakdownWork{in: in, t: b.tree, by: by, thenBy: thenBy, planCurrency: planCurrency, units: unitsOf(in.data.Nodes)}
	return w.result(groupings), nil
}

// groupingsOf returns every grouping the pickers offer (FR-040): the built-in ones, then the filterable Shell
// attributes level by level.
func groupingsOf(attributes []Attribute) []Grouping {
	list := []Grouping{
		{Key: byPlan, Label: "Plan", HasBudget: true},
		{Key: byPlanType, Label: "Plan type", HasBudget: true},
		{Key: byPlanCurrency, Label: "Plan currency", HasBudget: true},
		{Key: byYear, Label: "Year", HasBudget: true},
		{Key: byTeam, Label: "Team", HasBudget: true},
		{Key: byBusinessUnit, Label: "Business unit", HasBudget: true},
		{Key: byActivityType, Label: "Activity type"},
		{Key: byOwner, Label: "Owner"},
	}
	for i := range attributes {
		a := &attributes[i]
		if !a.filterable() {
			continue
		}
		level := string(a.Level)
		list = append(list, Grouping{
			Key: attrPrefix + a.ID, Label: a.Label, Level: &level,
			HasBudget: a.Level == LevelPlan || a.Level == LevelCampaign || a.Level == LevelInitiative, attribute: a,
		})
	}
	return list
}

// groupingNamed returns the grouping with key.
func groupingNamed(groupings []Grouping, key string) (Grouping, bool) {
	i := slices.IndexFunc(groupings, func(g Grouping) bool { return g.Key == key })
	if i < 0 {
		return Grouping{}, false
	}
	return groupings[i], true
}

// level returns the level the grouping's values come from: the attribute's, plans for the plan groupings,
// activities for Activity type and Owner.
func (g Grouping) level() Level {
	switch {
	case g.attribute != nil:
		return g.attribute.Level
	case g.Key == byActivityType || g.Key == byOwner:
		return LevelActivity
	}
	return LevelPlan
}

// unitsOf returns the Business unit options of every plan.
func unitsOf(nodes []Node) map[string][]string {
	units := map[string][]string{}
	for _, n := range nodes {
		if n.Level == LevelPlan {
			units[n.ID] = n.BusinessUnitIDs
		}
	}
	return units
}

// notFilledIn is the group of records without a value.
var notFilledIn = value{key: keyNotFilledIn, label: labelNotFilledIn, order: 1 << 30}

// values returns the groups of the piece p by the grouping g (data-model §3.5).
func (w *breakdownWork) values(p piece, g Grouping) []value {
	a := p.activity
	plan := planOf(a)
	switch g.Key {
	case byPlan:
		return []value{{key: plan.n.ID, label: plan.n.Name}}
	case byPlanType:
		return []value{{key: plan.n.PlanType, label: cmp.Or(planTypeWords[plan.n.PlanType], plan.n.PlanType)}}
	case byPlanCurrency:
		return []value{{key: plan.n.PlanCurrency, label: plan.n.PlanCurrency}}
	case byYear:
		year := strconv.Itoa(yearOf(plan.n, w.in.year))
		return []value{{key: year, label: year}}
	case byTeam:
		return w.optionValues(plan.n.TeamIDs, w.in.choices.Teams)
	case byBusinessUnit:
		return w.optionValues(plan.n.BusinessUnitIDs, w.in.choices.BusinessUnits)
	case byActivityType:
		return w.optionValues(nonEmpty(a.n.ActivityTypeID), w.in.choices.ActivityTypes)
	case byOwner:
		if a.n.Owner == nil {
			return []value{{key: keyNoOwner, label: labelNoOwner, order: 1 << 29}}
		}
		return []value{{key: a.n.Owner.ID, label: a.n.Owner.Name}}
	}
	attribute := g.attribute
	var record *tnode
	switch attribute.Level {
	case LevelLineItem:
		if p.lineItem == nil {
			return []value{{key: keyNotOnLineItem, label: labelNotOnLineItem, order: 1 << 31}}
		}
		record = p.lineItem
	case LevelActivity:
		record = a
	default:
		record = ancestorAt(a, attribute.Level)
	}
	if record == nil {
		return []value{notFilledIn}
	}
	return attributeValues(*attribute, record.n, w.units[record.n.PlanID])
}

// attributeValues returns the groups of a record by a Shell attribute: its options in their order (with the
// shares of a % split), Yes or No, or Not filled in.
func attributeValues(a Attribute, n *Node, units []string) []value {
	v, filled := valueOf(a, n, units)
	if !filled {
		return []value{notFilledIn}
	}
	if v.Yes != nil {
		if *v.Yes {
			return []value{{key: AttrYes, label: "Yes", order: 0}}
		}
		return []value{{key: AttrNo, label: "No", order: 1}}
	}
	list := make([]value, 0, len(v.OptionIDs))
	for _, id := range v.OptionIDs {
		val := value{key: id, label: id, order: len(a.Options)}
		if i := slices.IndexFunc(a.Options, func(o Option) bool { return o.ID == id }); i >= 0 {
			val.label, val.order = a.Options[i].Label, i
		}
		if v.Shares != nil {
			share := v.Shares[id]
			val.share = &share
		}
		list = append(list, val)
	}
	return list
}

// optionValues returns the groups of ids named by options (labels; an unknown id shows itself), or Not filled
// in without ids. Several ids count the amount in each.
func (w *breakdownWork) optionValues(ids []string, options []Option) []value {
	if len(ids) == 0 {
		return []value{notFilledIn}
	}
	list := make([]value, 0, len(ids))
	for _, id := range ids {
		val := value{key: id, label: id}
		if i := slices.IndexFunc(options, func(o Option) bool { return o.ID == id }); i >= 0 {
			val.label = options[i].Label
		}
		list = append(list, val)
	}
	return list
}

// nonEmpty returns id as a list, or none for "".
func nonEmpty(id string) []string {
	if id == "" {
		return nil
	}
	return []string{id}
}

// planOf returns the plan of the record t.
func planOf(t *tnode) *tnode {
	for t.parent != nil {
		t = t.parent
	}
	return t
}

// ancestorAt returns the record t or its ancestor at level; nil when there is none.
func ancestorAt(t *tnode, level Level) *tnode {
	for ; t != nil; t = t.parent {
		if t.n.Level == level {
			return t
		}
	}
	return nil
}
