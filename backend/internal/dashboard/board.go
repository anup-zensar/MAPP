package dashboard

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"mapp/internal/allwork"
	"mapp/internal/platform/money"
)

// board is the records of the plans on one dashboard, indexed, with the sums of every plan and campaign.
type board struct {
	plans      []*allwork.Node // by name
	byLevel    map[allwork.Level][]*allwork.Node
	campaignOf map[string]string // initiative and activity id → campaign id
	finance    map[string]bool   // plans whose financial figures count
	own        map[string]sums   // plan and campaign id → its sums
	paid       map[string]money.Amount
}

// newBoard keeps the nodes of the plans that keep says to keep; financePlans nil means every plan's finance counts.
func newBoard(nodes []allwork.Node, keep func(plan *allwork.Node) bool, seesFinance bool, financePlans []string) *board {
	b := &board{byLevel: map[allwork.Level][]*allwork.Node{}, campaignOf: map[string]string{}, finance: map[string]bool{},
		own: map[string]sums{}, paid: map[string]money.Amount{}}
	kept := map[string]bool{}
	for i := range nodes {
		n := &nodes[i]
		if n.Level == allwork.LevelPlan && keep(n) {
			kept[n.ID] = true
			b.plans = append(b.plans, n)
			b.finance[n.ID] = seesFinance && (financePlans == nil || slices.Contains(financePlans, n.ID))
		}
	}
	for i := range nodes {
		if n := &nodes[i]; kept[n.PlanID] {
			b.byLevel[n.Level] = append(b.byLevel[n.Level], n)
		}
	}
	slices.SortFunc(b.plans, func(x, y *allwork.Node) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.Name), strings.ToLower(y.Name)), strings.Compare(x.ID, y.ID))
	})
	b.index()
	b.addUp()
	return b
}

func (b *board) index() {
	for _, c := range b.byLevel[allwork.LevelCampaign] {
		b.campaignOf[c.ID] = c.ID
	}
	for _, level := range []allwork.Level{allwork.LevelInitiative, allwork.LevelActivity} {
		for _, n := range b.byLevel[level] {
			b.campaignOf[n.ID] = b.campaignOf[n.ParentID]
		}
	}
}

// addUp gives every plan and campaign its budget, its activities' planned cost and its line items' GSAP figures.
func (b *board) addUp() {
	for _, p := range b.plans {
		b.own[p.ID] = b.start(p)
	}
	for _, c := range b.byLevel[allwork.LevelCampaign] {
		b.own[c.ID] = b.start(c)
	}
	for _, a := range b.byLevel[allwork.LevelActivity] {
		b.addTo(a, sums{planned: a.PlannedCost})
	}
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		if b.finance[li.PlanID] && li.Finance != nil {
			b.addTo(li, sums{committed: &li.Finance.Committed})
		}
	}
}

// countPaid makes each line item's payments posted in year its actual, as 013's Actual spend counts (D-357).
func (b *board) countPaid(paid map[string]map[string]money.Amount, year int) {
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		if !b.finance[li.PlanID] {
			continue
		}
		var actual money.Amount
		for month, amount := range paid[li.ID] {
			if inYear(month, year) {
				actual = actual.Add(amount)
			}
		}
		b.paid[li.ID] = actual
		b.addTo(li, sums{actual: &actual})
	}
}

func inYear(month string, year int) bool {
	return month[:4] == strconv.Itoa(year)
}

func (b *board) start(n *allwork.Node) sums {
	s := sums{budget: n.Budget}
	if b.finance[n.PlanID] {
		zero := money.Amount{}
		s.committed, s.actual = &zero, &zero
	}
	return s
}

func (b *board) addTo(n *allwork.Node, s sums) {
	b.own[n.PlanID] = b.own[n.PlanID].plus(s)
	campaign := b.campaignOf[n.ID]
	if n.Level == allwork.LevelLineItem {
		campaign = b.campaignOf[n.ParentID]
	}
	if campaign != "" {
		b.own[campaign] = b.own[campaign].plus(s)
	}
}

func (b *board) total() sums {
	var t sums
	for _, p := range b.plans {
		t = t.plus(b.own[p.ID])
	}
	return t
}

func (b *board) financeCount() int {
	count := 0
	for _, p := range b.plans {
		if b.finance[p.ID] {
			count++
		}
	}
	return count
}

func (b *board) inView(planID string) bool {
	_, found := b.finance[planID]
	return found
}

func (b *board) teamIDs() []string {
	ids := []string{}
	for _, p := range b.plans {
		for _, id := range p.TeamIDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (b *board) lineItemIDs() []string {
	ids := make([]string, 0, len(b.byLevel[allwork.LevelLineItem]))
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		ids = append(ids, li.ID)
	}
	return ids
}

func (b *board) financeLineItemIDs() []string {
	ids := []string{}
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		if b.finance[li.PlanID] {
			ids = append(ids, li.ID)
		}
	}
	return ids
}
