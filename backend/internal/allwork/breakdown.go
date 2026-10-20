package allwork

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/money"
	"northstar/internal/rollup"
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
)

// The keys of the groups that are not values.
const (
	keyNotFilledIn   = "notFilledIn"
	keyNotOnLineItem = "notOnLineItem"
	keyNoOwner       = "noOwner"
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

// Grouping is one grouping the Breakdown offers (contract AllWorkGrouping).
type Grouping struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Level     *string `json:"level"` // the attribute's level; null for the built-in groupings
	HasBudget bool    `json:"hasBudget"`

	attribute *Attribute // attribute groupings
}

// BreakdownSubgroup is a "then by" group inside a group (contract AllWorkBreakdownSubgroup).
type BreakdownSubgroup struct {
	Key               string       `json:"key"`
	Label             string       `json:"label"`
	Activities        int          `json:"activities"`
	LineItems         int          `json:"lineItems"`
	PlannedCost       money.Money  `json:"plannedCost"`
	Share             string       `json:"share"`
	Budget            *money.Money `json:"budget"`
	BudgetUsedPercent *int         `json:"budgetUsedPercent"`
	OverBudget        bool         `json:"overBudget"`
}

// BreakdownGroup is one group of the Breakdown, with its "then by" groups (contract AllWorkBreakdownGroup).
type BreakdownGroup struct {
	Key               string              `json:"key"`
	Label             string              `json:"label"`
	Activities        int                 `json:"activities"`
	LineItems         int                 `json:"lineItems"`
	PlannedCost       money.Money         `json:"plannedCost"`
	Share             string              `json:"share"`
	Budget            *money.Money        `json:"budget"`
	BudgetUsedPercent *int                `json:"budgetUsedPercent"`
	OverBudget        bool                `json:"overBudget"`
	Children          []BreakdownSubgroup `json:"children"`
}

// Breakdown is the Breakdown of a filter set (contract AllWorkBreakdown).
type Breakdown struct {
	By           Grouping         `json:"by"`
	ThenBy       *Grouping        `json:"thenBy"`
	Currency     string           `json:"currency"`
	Groups       []BreakdownGroup `json:"groups"`
	Total        BreakdownGroup   `json:"total"`
	CountsInEach bool             `json:"countsInEach"`
	Notes        []string         `json:"notes"`
	Groupings    []Grouping       `json:"groupings"`
}

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

// breakdownWork is one Breakdown being worked out.
type breakdownWork struct {
	in           input
	t            *tree
	by           Grouping
	thenBy       *Grouping
	planCurrency string // the plan currency of amountsIn=plan; "" for the display currency
	units        map[string][]string

	countsInEach bool
	withoutPC    money.Amount // line items of activities without a planned cost (line-item groupings)
}

// piece is a part of an activity's planned cost that goes to groups: the whole activity, one of its line items,
// or its unassigned planned cost (line-item groupings).
type piece struct {
	activity   *tnode
	lineItem   *tnode // nil for the whole activity and the unassigned part
	unassigned bool
	amount     money.Amount
}

// value is one group of a piece with the share it gets: nil for the full amount.
type value struct {
	key, label string
	order      int
	share      *money.Percent
}

// accum adds up one group.
type accum struct {
	value
	activities map[*tnode]bool
	lineItems  map[*tnode]bool
	planned    money.Amount
	children   map[string]*accum
}

// newAccum returns an empty group of v.
func newAccum(v value) *accum {
	return &accum{value: v, activities: map[*tnode]bool{}, lineItems: map[*tnode]bool{}, children: map[string]*accum{}}
}

// add counts the piece p with the amount a in the group.
func (g *accum) add(p piece, a money.Amount) {
	g.planned = g.planned.Add(a)
	g.activities[p.activity] = true
	if p.lineItem != nil {
		g.lineItems[p.lineItem] = true
	}
}

// result works out the groups, the totals and the notes.
func (w *breakdownWork) result(groupings []Grouping) Breakdown {
	byLineItems := w.by.level() == LevelLineItem || (w.thenBy != nil && w.thenBy.level() == LevelLineItem)
	groups := map[string]*accum{}
	total := newAccum(value{key: "total", label: "Total"})
	var activities []*tnode
	for _, a := range w.matchingActivities() {
		activities = append(activities, a)
		for _, p := range w.pieces(a, byLineItems) {
			total.add(p, p.amount)
			for _, part := range w.split(p.amount, w.values(p, w.by)) {
				g := groups[part.key]
				if g == nil {
					g = newAccum(part.value)
					groups[part.key] = g
				}
				g.add(p, part.amount)
				if w.thenBy == nil {
					continue
				}
				for _, sub := range w.split(part.amount, w.values(p, *w.thenBy)) {
					c := g.children[sub.key]
					if c == nil {
						c = newAccum(sub.value)
						g.children[sub.key] = c
					}
					c.add(p, sub.amount)
				}
			}
		}
	}
	result := Breakdown{
		By: w.by, ThenBy: w.thenBy, Currency: w.currency(), Groups: []BreakdownGroup{}, CountsInEach: w.countsInEach,
		Notes: []string{}, Groupings: groupings,
	}
	for _, g := range sortedAccums(groups) {
		group := w.group(g, total.planned, w.by, byLineItems, nil)
		group.Children = []BreakdownSubgroup{}
		for _, c := range sortedAccums(g.children) {
			sub := w.group(c, total.planned, *w.thenBy, byLineItems, g)
			group.Children = append(group.Children, BreakdownSubgroup{
				Key: sub.Key, Label: sub.Label, Activities: sub.Activities, LineItems: sub.LineItems, PlannedCost: sub.PlannedCost,
				Share: sub.Share, Budget: sub.Budget, BudgetUsedPercent: sub.BudgetUsedPercent, OverBudget: sub.OverBudget,
			})
		}
		result.Groups = append(result.Groups, group)
	}
	result.Total = w.totalGroup(total, activities)
	if w.countsInEach {
		result.Notes = append(result.Notes, noteCountsInEach)
	}
	if !w.withoutPC.IsZero() {
		result.Notes = append(result.Notes, fmt.Sprintf(noteLineItemsNoPC, w.money(w.withoutPC).Format()))
	}
	return result
}

// matchingActivities returns the shown matching activities of the tree, in tree order (010 FR-005).
func (w *breakdownWork) matchingActivities() []*tnode {
	var list []*tnode
	var walk func(n *tnode)
	walk = func(n *tnode) {
		if n.n.Level == LevelActivity {
			if n.shown && n.match {
				list = append(list, n)
			}
			return
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, p := range w.t.plans {
		walk(p)
	}
	return list
}

// amountOf returns the planned cost of a record in the Breakdown's currency, before conversion: USD, or the
// plan currency; nil when it has none.
func (w *breakdownWork) amountOf(n *Node) *money.Amount {
	if w.planCurrency != "" {
		if n.PlannedCostPlan == nil {
			return nil
		}
		return &n.PlannedCostPlan.Amount
	}
	return n.PlannedCost
}

// pieces returns the parts of the activity a's planned cost (data-model §3.5): the whole activity, or, for a
// line-item grouping, each line item and the unassigned planned cost ("Not on a line item"). Line items of an
// activity without a planned cost count too, and are named in a note.
func (w *breakdownWork) pieces(a *tnode, byLineItems bool) []piece {
	planned := w.amountOf(a.n)
	if !byLineItems {
		var amount money.Amount
		if planned != nil {
			amount = *planned
		}
		return []piece{{activity: a, amount: amount}}
	}
	var list []piece
	var assigned money.Amount
	for _, li := range a.children {
		var amount money.Amount
		if cost := w.amountOf(li.n); cost != nil {
			amount = *cost
		}
		assigned = assigned.Add(amount)
		list = append(list, piece{activity: a, lineItem: li, amount: amount})
	}
	switch {
	case planned == nil:
		w.withoutPC = w.withoutPC.Add(assigned)
		if len(list) == 0 {
			list = append(list, piece{activity: a, unassigned: true})
		}
	case !planned.Sub(assigned).IsZero() || len(list) == 0:
		list = append(list, piece{activity: a, unassigned: true, amount: planned.Sub(assigned)})
	}
	return list
}

// part is the amount of a piece that goes to one group.
type part struct {
	value
	amount money.Amount
}

// split divides amount among the values (data-model §3.5): one value gets it all; a % split gives each value
// amount × share ÷ 100, half up, and the last the rest; several values without shares each get it all, and
// the Breakdown says so.
func (w *breakdownWork) split(amount money.Amount, values []value) []part {
	if len(values) == 1 {
		return []part{{value: values[0], amount: amount}}
	}
	parts := make([]part, 0, len(values))
	if values[0].share == nil {
		w.countsInEach = true
		for _, v := range values {
			parts = append(parts, part{value: v, amount: amount})
		}
		return parts
	}
	var given money.Amount
	for i, v := range values {
		a := amount.Sub(given)
		if i < len(values)-1 {
			a = shareOfAmount(amount, *v.share)
			given = given.Add(a)
		}
		parts = append(parts, part{value: v, amount: a})
	}
	return parts
}

// shareOfAmount is a × p ÷ 100, rounded half up (away from zero) to the cent.
func shareOfAmount(a money.Amount, p money.Percent) money.Amount {
	num := new(big.Int).Mul(big.NewInt(a.Cents()), big.NewInt(p.Hundredths()))
	return money.FromCents(roundHalfAway(num, big.NewInt(10000)))
}

// roundHalfAway divides num by den, rounding half away from zero.
func roundHalfAway(num, den *big.Int) int64 {
	negative := num.Sign()*den.Sign() < 0
	n, d := new(big.Int).Abs(num), new(big.Int).Abs(den)
	q, r := new(big.Int).QuoRem(n, d, new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(d) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if negative {
		q.Neg(q)
	}
	return q.Int64()
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

// sortedAccums returns the groups in order (data-model §3.5): by option order, then by name; Not filled in
// and Not on a line item last.
func sortedAccums(groups map[string]*accum) []*accum {
	list := make([]*accum, 0, len(groups))
	for _, g := range groups {
		list = append(list, g)
	}
	slices.SortFunc(list, func(x, y *accum) int {
		return cmp.Or(cmp.Compare(x.order, y.order), strings.Compare(strings.ToLower(x.label), strings.ToLower(y.label)),
			strings.Compare(x.key, y.key))
	})
	return list
}

// currency is the currency of the Breakdown's amounts.
func (w *breakdownWork) currency() string {
	if w.planCurrency != "" {
		return w.planCurrency
	}
	return w.in.currency.Code
}

// money returns an amount, summed in USD or the plan currency, as shown: converted once to the display
// currency, or as it is in the plan currency.
func (w *breakdownWork) money(a money.Amount) money.Money {
	if w.planCurrency != "" {
		return money.Money{Amount: a, Currency: w.planCurrency}
	}
	return w.in.currency.money(a)
}

// group returns the group g as answered (FR-040): its counts, planned cost, share of the total and, for a
// grouping with budgets, the budget of its records and the budget used. parent is the group of a "then by"
// group; nil for a group.
func (w *breakdownWork) group(g *accum, total money.Amount, by Grouping, byLineItems bool, parent *accum) BreakdownGroup {
	result := BreakdownGroup{
		Key: g.key, Label: g.label, Activities: len(g.activities), PlannedCost: w.money(g.planned), Share: shareText2(g.planned, total),
		Children: []BreakdownSubgroup{},
	}
	if by.level() == LevelLineItem {
		result.LineItems = len(g.lineItems)
	} else {
		for a := range g.activities {
			result.LineItems += len(a.children)
		}
	}
	if by.HasBudget && g.key != keyNotOnLineItem {
		if budget := w.budgetOf(g, by, parent); budget != nil {
			shown := w.money(*budget)
			result.Budget = &shown
			p, b := money.USD(g.planned), money.USD(*budget)
			result.BudgetUsedPercent = rollup.BudgetUsedPercent(&p, &b)
			result.OverBudget = g.planned.Cmp(*budget) > 0
		}
	}
	return result
}

// budgetOf returns the budgets of the records at the grouping's level whose value is in the group g (several
// values → each), counting only the records the filters show; within a "then by" group, only those of the
// parent group's activities' branches. nil when no such record has a budget.
func (w *breakdownWork) budgetOf(g *accum, by Grouping, parent *accum) *money.Amount {
	var budget *money.Amount
	for _, r := range w.recordsAt(by.level()) {
		if !r.shown || !slices.ContainsFunc(w.recordValues(r, by), func(v value) bool { return v.key == g.key }) {
			continue
		}
		if parent != nil && !w.inBranch(r, parent) {
			continue
		}
		budget = addAmount(budget, w.budgetAmount(r.n))
	}
	return budget
}

// inBranch reports whether one of the activities of the group is under the record r, or r is under one of
// them.
func (w *breakdownWork) inBranch(r *tnode, group *accum) bool {
	for a := range group.activities {
		for t := a; t != nil; t = t.parent {
			if t == r {
				return true
			}
		}
	}
	return false
}

// recordsAt returns every record of the level.
func (w *breakdownWork) recordsAt(level Level) []*tnode {
	var list []*tnode
	for _, t := range w.t.byID {
		if t.n.Level == level {
			list = append(list, t)
		}
	}
	return list
}

// recordValues returns the groups of the record r (a plan, campaign or initiative) by the grouping.
func (w *breakdownWork) recordValues(r *tnode, by Grouping) []value {
	if by.attribute != nil {
		return attributeValues(*by.attribute, r.n, w.units[r.n.PlanID])
	}
	return w.values(piece{activity: r}, by)
}

// budgetAmount is a record's budget in the Breakdown's currency: USD, or the plan currency.
func (w *breakdownWork) budgetAmount(n *Node) *money.Amount {
	if w.planCurrency != "" {
		if n.BudgetPlan == nil {
			return nil
		}
		return &n.BudgetPlan.Amount
	}
	return n.Budget
}

// totalGroup is the totals row (FR-040): every matching activity once, their line items, the total planned
// cost — equal to the Planned cost card — and, for a grouping with budgets, the budgets of the shown records of
// its level once each.
func (w *breakdownWork) totalGroup(total *accum, activities []*tnode) BreakdownGroup {
	result := BreakdownGroup{
		Key: "total", Label: "Total", Activities: len(activities), PlannedCost: w.money(total.planned),
		Share: shareText2(total.planned, total.planned), Children: []BreakdownSubgroup{},
	}
	for _, a := range activities {
		result.LineItems += len(a.children)
	}
	if w.by.HasBudget {
		var budget *money.Amount
		for _, r := range w.recordsAt(w.by.level()) {
			if r.shown {
				budget = addAmount(budget, w.budgetAmount(r.n))
			}
		}
		if budget != nil {
			shown := w.money(*budget)
			result.Budget = &shown
			p, b := money.USD(total.planned), money.USD(*budget)
			result.BudgetUsedPercent = rollup.BudgetUsedPercent(&p, &b)
			result.OverBudget = total.planned.Cmp(*budget) > 0
		}
	}
	return result
}

// shareText2 is part ÷ whole × 100, half up to two decimals ("37.50"); "0.00" when whole is 0.00.
func shareText2(part, whole money.Amount) string {
	if whole.IsZero() {
		return "0.00"
	}
	num := new(big.Int).Mul(big.NewInt(part.Cents()), big.NewInt(10000))
	hundredths := roundHalfAway(num, big.NewInt(whole.Cents()))
	sign := ""
	if hundredths < 0 {
		sign, hundredths = "-", -hundredths
	}
	return fmt.Sprintf("%s%d.%02d", sign, hundredths/100, hundredths%100)
}
