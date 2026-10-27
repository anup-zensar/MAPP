package allwork

import (
	"cmp"
	"slices"
	"strings"

	"mapp/internal/platform/money"
)

// The tree of one year (data model §2–§4): the plans by name, each with its campaigns, initiatives,
// activities and line items in the plan tree's order. Matching (filters, search, Show levels) marks which
// records are shown and whether each is a match or only context; the roll-ups, rows and cells are worked
// out from those marks.

// tnode is one record of the tree.
type tnode struct {
	n        *Node
	parent   *tnode
	children []*tnode // campaigns, initiatives, activities or line items, in the plan tree's order
	depth    int      // plan 0, campaign 1, initiative 2, activity 3, line item 4

	match      bool       // meets every filter and the search (data model §3.2)
	matchBelow bool       // a record below it is a match
	shown      bool       // a match, or a parent of a shown record
	hit        *SearchHit // the record's own fields that contain the search text

	// Activities only (domain model §4.3).
	lineItemsUSD *money.Amount // the sum of its line items' USD planned costs; nil without line items
	balance      *money.Amount // its unassigned planned cost: planned cost − line items; nil with neither

	// Feature 015 (finance.go): the GSAP figures of a line item, or of an activity's line items; a zero
	// start for containers; nil when the reader may not see them. abovePlanned marks an activity committed
	// above its planned cost, itself or on one of its line items.
	finance      *FinanceFacts
	abovePlanned bool

	fig figures // roll-ups over the matching activities shown below (activities: their own values)
	all figures // the same over every activity below, for the "Filtered" mark (FR-013)
}

// role returns RoleMatch or RoleContext.
func (t *tnode) role() string {
	if t.match {
		return RoleMatch
	}
	return RoleContext
}

// isContainer reports whether the record is a plan, campaign or initiative.
func (t *tnode) isContainer() bool {
	return t.n.Level == LevelPlan || t.n.Level == LevelCampaign || t.n.Level == LevelInitiative
}

// tree is the year's records.
type tree struct {
	plans []*tnode          // sorted by name ignoring letter case
	byID  map[string]*tnode // every record of the tree
	total map[Level]int     // every record of each level in the year's plans
}

// buildTree builds the tree of nodes (FR-011): plans by name ignoring letter case (then ID), children in
// their plan-tree order (Node.Order, then ID), line items in creation order. A node whose parent is not
// among nodes is left out. Every activity gets the sum of its line items and its balance.
func buildTree(nodes []Node) *tree {
	t := &tree{byID: make(map[string]*tnode, len(nodes)), total: map[Level]int{}}
	all := make([]*tnode, len(nodes))
	for i := range nodes {
		all[i] = &tnode{n: &nodes[i]}
		t.byID[nodes[i].ID] = all[i]
	}
	for _, tn := range all {
		if tn.n.Level == LevelPlan {
			t.plans = append(t.plans, tn)
			continue
		}
		if parent, found := t.byID[tn.n.ParentID]; found {
			tn.parent = parent
			parent.children = append(parent.children, tn)
		}
	}
	slices.SortFunc(t.plans, func(x, y *tnode) int {
		return cmp.Or(strings.Compare(strings.ToLower(x.n.Name), strings.ToLower(y.n.Name)), cmp.Compare(x.n.ID, y.n.ID))
	})
	reached := make(map[string]*tnode, len(nodes))
	for _, p := range t.plans {
		t.prepare(p, 0, reached)
	}
	t.byID = reached
	return t
}

// prepare orders the children of n, sets depths, counts the records and works out the activities' line
// items and balances.
func (t *tree) prepare(n *tnode, depth int, reached map[string]*tnode) {
	n.depth = depth
	reached[n.n.ID] = n
	t.total[n.n.Level]++
	slices.SortStableFunc(n.children, func(x, y *tnode) int {
		return cmp.Or(cmp.Compare(x.n.Order, y.n.Order), compareIDs(x.n.ID, y.n.ID))
	})
	for _, c := range n.children {
		t.prepare(c, depth+1, reached)
	}
	if n.n.Level == LevelActivity {
		n.lineItemsUSD, n.balance = activityMoney(n)
	}
}

// activityMoney returns the sum of the activity's line items in USD (nil without line items) and its
// balance, the unassigned planned cost (domain model §4.3): planned cost (empty counts as 0.00) − line
// items; nil when it has neither a planned cost nor a line item, as feature 006 shows it.
func activityMoney(a *tnode) (lineItems, balance *money.Amount) {
	for _, li := range a.children {
		lineItems = addAmount(lineItems, li.n.PlannedCost)
		if lineItems == nil {
			zero := money.Amount{}
			lineItems = &zero
		}
	}
	if a.n.PlannedCost == nil && lineItems == nil {
		return nil, nil
	}
	var planned, assigned money.Amount
	if a.n.PlannedCost != nil {
		planned = *a.n.PlannedCost
	}
	if lineItems != nil {
		assigned = *lineItems
	}
	b := planned.Sub(assigned)
	return lineItems, &b
}

// compareIDs orders readable IDs as the store does: shorter first, then alphabetically (A-999 before
// A-1001).
func compareIDs(x, y string) int {
	return cmp.Or(cmp.Compare(len(x), len(y)), strings.Compare(x, y))
}

// addAmount returns total + a, where a nil a adds nothing and a nil total is "none yet": a sum of only
// empty values stays empty ("—").
func addAmount(total, a *money.Amount) *money.Amount {
	if a == nil {
		return total
	}
	sum := *a
	if total != nil {
		sum = total.Add(*a)
	}
	return &sum
}

// addCount is addAmount for whole numbers.
func addCount(total, n *int64) *int64 {
	if n == nil {
		return total
	}
	sum := *n
	if total != nil {
		sum += *total
	}
	return &sum
}
