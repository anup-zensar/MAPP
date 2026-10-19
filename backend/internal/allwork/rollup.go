package allwork

import (
	"northstar/internal/platform/money"
)

// Roll-ups, summary cards and totals (data model §5; FR-003, FR-013, FR-014, FR-017). Every sum is made in
// USD with exact decimals; each value shown is converted once to the display currency, half up, just
// before it is returned, so totals never drift through rounding.

// figures are the money and counts a row adds up, in USD. A nil amount is "—": nothing below has one.
type figures struct {
	activities int           // activities counted
	planned    *money.Amount // planned cost; an empty one counts as 0.00, a sum of only empty ones is "—"
	lineItems  *money.Amount // the line items' USD planned costs
	unassigned *money.Amount // activities: the balance; containers: the sum of the positive balances (D-116)
	negative   int           // activities with a negative balance (line items above the planned cost)
	inq, mql   *int64
	revenue    *money.Amount
}

// activityFigures are an activity's own values.
func activityFigures(a *tnode) figures {
	f := figures{
		activities: 1, planned: a.n.PlannedCost, lineItems: a.lineItemsUSD, unassigned: a.balance,
		inq: a.n.INQ, mql: a.n.MQL, revenue: a.n.Revenue,
	}
	if a.balance != nil && a.balance.Sign() < 0 {
		f.negative = 1
	}
	return f
}

// addActivity adds an activity's figures to a container's: its balance counts only when positive, a
// negative one is counted apart (D-116).
func (f *figures) addActivity(a figures) {
	f.activities++
	f.planned = addAmount(f.planned, a.planned)
	f.lineItems = addAmount(f.lineItems, a.lineItems)
	if a.unassigned != nil {
		positive := *a.unassigned
		if positive.Sign() < 0 {
			positive = money.Amount{}
			f.negative++
		}
		f.unassigned = addAmount(f.unassigned, &positive)
	}
	f.inq = addCount(f.inq, a.inq)
	f.mql = addCount(f.mql, a.mql)
	f.revenue = addAmount(f.revenue, a.revenue)
}

// addContainer adds a container's figures to its parent's.
func (f *figures) addContainer(c figures) {
	f.activities += c.activities
	f.planned = addAmount(f.planned, c.planned)
	f.lineItems = addAmount(f.lineItems, c.lineItems)
	f.unassigned = addAmount(f.unassigned, c.unassigned)
	f.negative += c.negative
	f.inq = addCount(f.inq, c.inq)
	f.mql = addCount(f.mql, c.mql)
	f.revenue = addAmount(f.revenue, c.revenue)
}

// rollUp works out the roll-ups of every record (FR-013): an activity's own values; a container's sums over
// the matching activities shown below it (fig) and over every activity below it (all).
func rollUp(t *tree) {
	for _, p := range t.plans {
		rollUpNode(p)
	}
}

// rollUpNode works out the roll-ups of n and everything below it.
func rollUpNode(n *tnode) {
	switch n.n.Level {
	case LevelActivity:
		n.fig = activityFigures(n)
		n.all = n.fig
		return
	case LevelLineItem:
		return
	}
	n.fig, n.all = figures{}, figures{}
	for _, c := range n.children {
		rollUpNode(c)
		if c.n.Level == LevelActivity {
			n.all.addActivity(c.fig)
			if c.shown && c.match {
				n.fig.addActivity(c.fig)
			}
			continue
		}
		n.all.addContainer(c.all)
		n.fig.addContainer(c.fig)
	}
}

// filtered reports whether filters hide some activities below the container n (FR-013).
func (t *tnode) filtered() bool {
	return t.isContainer() && t.fig.activities != t.all.activities
}

// matches are the shown matches of the year, as the summary cards and the totals row count them (FR-003,
// FR-014).
type matches struct {
	campaigns, initiatives int
	activities             figures       // the matching activities, added up as a container would
	planBudget             *money.Amount // the budgets of the matching plans
}

// matchesOf adds up the shown matches of t.
func matchesOf(t *tree) matches {
	var m matches
	for _, n := range t.byID {
		if !n.shown || !n.match {
			continue
		}
		switch n.n.Level {
		case LevelPlan:
			m.planBudget = addAmount(m.planBudget, n.n.Budget)
		case LevelCampaign:
			m.campaigns++
		case LevelInitiative:
			m.initiatives++
		case LevelActivity:
			m.activities.addActivity(activityFigures(n))
		}
	}
	return m
}

// summary returns the six summary cards (FR-003): the numbers of matching campaigns, initiatives and
// activities, their planned cost, the matching plans' budgets and the positive balances with the count of
// negative ones; equal to the totals row by construction (FR-014, SC-003).
func (m matches) summary(c Currency) Summary {
	return Summary{
		Campaigns:             m.campaigns,
		Initiatives:           m.initiatives,
		Activities:            m.activities.activities,
		PlannedCost:           c.optional(m.activities.planned),
		PlanBudget:            c.optional(m.planBudget),
		UnassignedPlannedCost: c.optional(m.activities.unassigned),
		NegativeBalanceCount:  m.activities.negative,
	}
}

// money converts an amount in USD to the display currency, half up to the cent (FR-017).
func (c Currency) money(usd money.Amount) money.Money {
	return money.Money{Amount: money.FromUSD(usd, c.Rate), Currency: c.Code}
}

// optional converts an amount in USD, or gives nil for no amount.
func (c Currency) optional(usd *money.Amount) *money.Money {
	if usd == nil {
		return nil
	}
	m := c.money(*usd)
	return &m
}
