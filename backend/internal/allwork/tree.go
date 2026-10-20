package allwork

import (
	"cmp"
	"slices"
	"strings"
	"unicode/utf8"

	"northstar/internal/platform/money"
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

// The matching of data model §3 (FR-005 – FR-008, D-113).

// levelRank orders the levels top-down.
var levelRank = map[Level]int{LevelPlan: 0, LevelCampaign: 1, LevelInitiative: 2, LevelActivity: 3, LevelLineItem: 4}

// filter is one active filter of FR-004: the levels it is tested on, top-down, and the test of a record of
// such a level.
type filter struct {
	levels []Level
	meets  func(n *Node) bool
	test   func(t *tnode) bool // when set, tests the record with its children instead of meets (016 line-item attributes)
}

// progressValues are the Approval progress values of the Campaign / initiative status filter; the others
// are Timing values (domain model §2.2).
var progressValues = []string{"noActivities", "inProgress", "allApproved"}

// filtersOf returns the active filters of q (FR-004). An empty list is no filter, and so is Include.
func filtersOf(q Query) []filter {
	plans := []Level{LevelPlan}
	activities := []Level{LevelActivity}
	var filters []filter
	add := func(values []string, levels []Level, meets func(n *Node) bool) {
		if len(values) > 0 {
			filters = append(filters, filter{levels: levels, meets: meets})
		}
	}
	add(q.Plans, plans, func(n *Node) bool { return slices.Contains(q.Plans, n.ID) })
	add(q.PlanStatuses, plans, func(n *Node) bool { return slices.Contains(q.PlanStatuses, n.Status) })
	add(q.ContainerStatuses, []Level{LevelCampaign, LevelInitiative}, containerStatusTest(q.ContainerStatuses))
	add(q.ActivityStatuses, activities, func(n *Node) bool { return slices.Contains(q.ActivityStatuses, n.Status) })
	add(q.Owners, []Level{LevelPlan, LevelCampaign, LevelInitiative, LevelActivity}, func(n *Node) bool {
		return n.Owner != nil && slices.Contains(q.Owners, n.Owner.ID)
	})
	add(q.Teams, plans, func(n *Node) bool { return containsAny(n.TeamIDs, q.Teams) })
	add(q.BusinessUnits, plans, func(n *Node) bool { return containsAny(n.BusinessUnitIDs, q.BusinessUnits) })
	add(q.ActivityTypes, activities, func(n *Node) bool { return slices.Contains(q.ActivityTypes, n.ActivityTypeID) })
	switch q.Strategic {
	case StrategicOnly:
		filters = append(filters, filter{levels: plans, meets: func(n *Node) bool { return n.Strategic }})
	case StrategicExclude:
		filters = append(filters, filter{levels: plans, meets: func(n *Node) bool { return !n.Strategic }})
	}
	// 016 FR-037, FR-039: the plan type and plan currency of plans, and the Shell-attribute filters.
	add(q.PlanTypes, plans, func(n *Node) bool { return slices.Contains(q.PlanTypes, n.PlanType) })
	add(q.PlanCurrencies, plans, func(n *Node) bool { return slices.Contains(q.PlanCurrencies, n.PlanCurrency) })
	for _, a := range q.attrs {
		filters = append(filters, a.filter())
	}
	return filters
}

// containerStatusTest is the test of the Campaign / initiative status filter: the values of one group —
// Approval progress, Timing — combine with OR, the two groups with AND; a record without dates has no
// Timing (D-113).
func containerStatusTest(values []string) func(n *Node) bool {
	var progress, timing []string
	for _, v := range values {
		if slices.Contains(progressValues, v) {
			progress = append(progress, v)
		} else {
			timing = append(timing, v)
		}
	}
	return func(n *Node) bool {
		return (len(progress) == 0 || slices.Contains(progress, n.Status)) &&
			(len(timing) == 0 || slices.Contains(timing, n.Timing))
	}
}

// containsAny reports whether list holds any of values.
func containsAny(list, values []string) bool {
	return slices.ContainsFunc(list, func(v string) bool { return slices.Contains(values, v) })
}

// holds reports whether the filter holds for the record t (data model §3.1): on a level it is tested on,
// the record must meet it; below those levels, the nearest ancestor at a tested level must; above them, a
// record below must be a match. Children are marked before their parents, so matchBelow is known.
func (f filter) holds(t *tnode) bool {
	if slices.Contains(f.levels, t.n.Level) {
		return f.on(t)
	}
	if levelRank[f.levels[0]] < levelRank[t.n.Level] {
		for a := t.parent; a != nil; a = a.parent {
			if slices.Contains(f.levels, a.n.Level) {
				return f.on(a)
			}
		}
		return false
	}
	return t.matchBelow
}

// on tests the record t of a level the filter is tested on.
func (f filter) on(t *tnode) bool {
	if f.test != nil {
		return f.test(t)
	}
	return f.meets(t.n)
}

// applyFilters marks every record of t a match, a context row or hidden, for the filters, the search and
// the Show levels choice of q (data model §3).
func applyFilters(t *tree, q Query) {
	filters := filtersOf(q)
	if len(q.Finance) > 0 {
		filters = append(filters, financeFilter(q.Finance, t)) // feature 015 (FR-024)
	}
	for _, p := range t.plans {
		mark(p, filters, q.Search, strings.ToLower(q.Search), false)
	}
	switch q.ShowLevels {
	case ShowCompleteBranches:
		for _, p := range t.plans {
			keepCompleteBranches(p)
		}
	case ShowCampaignsWithoutInitiatives:
		for _, p := range t.plans {
			keepWithoutChildren(p, LevelCampaign)
		}
	case ShowInitiativesWithoutActivities:
		for _, p := range t.plans {
			keepWithoutChildren(p, LevelInitiative)
		}
	}
}

// mark marks n and everything below it, children first (data model §3.2, §3.3). A record is a match when
// it meets the search — its own name, ID, external name or description holds the text, one of its line
// items' does (activities), or an ancestor's does (ancestorHit) — and every filter holds for it. It is
// shown when it is a match or has a shown record below it. Line items take the marks of their activity.
func mark(n *tnode, filters []filter, text, lowerText string, ancestorHit bool) {
	n.hit = nil
	if lowerText != "" {
		n.hit = searchHit(n.n, text, lowerText)
	}
	inherited := ancestorHit || n.hit != nil
	n.matchBelow, n.shown = false, false
	lineItemHit := false
	for _, c := range n.children {
		if c.n.Level == LevelLineItem {
			c.hit = nil
			if lowerText != "" {
				c.hit = searchHit(c.n, text, lowerText)
				lineItemHit = lineItemHit || c.hit != nil
			}
			continue
		}
		mark(c, filters, text, lowerText, inherited)
		n.matchBelow = n.matchBelow || c.match || c.matchBelow
		n.shown = n.shown || c.shown
	}
	n.match = lowerText == "" || inherited || lineItemHit
	for _, f := range filters {
		if !n.match {
			break
		}
		n.match = f.holds(n)
	}
	n.shown = n.shown || n.match
	if n.n.Level == LevelActivity {
		for _, li := range n.children {
			li.match, li.shown = n.match, n.shown
		}
	}
}

// keepCompleteBranches hides every plan, campaign and initiative without a shown activity below it
// (Complete branches only, FR-007) and reports whether n reaches one.
func keepCompleteBranches(n *tnode) bool {
	if n.n.Level == LevelActivity {
		return n.shown
	}
	reaches := false
	for _, c := range n.children {
		if keepCompleteBranches(c) {
			reaches = true
		}
	}
	if !reaches {
		n.shown = false
	}
	return reaches
}

// keepWithoutChildren keeps under the plan p only the records of level that have no child at all and match
// the filters on their level, with the records above them as context rows (Campaigns without initiatives,
// Initiatives without activities, FR-007); everything else is hidden.
func keepWithoutChildren(p *tnode, level Level) {
	var visit func(n *tnode) bool
	visit = func(n *tnode) bool {
		if n.n.Level == level {
			keep := n.match && len(n.children) == 0
			n.match, n.shown = keep, keep
			for _, c := range n.children {
				hide(c)
			}
			return keep
		}
		n.match, n.shown = false, false
		for _, c := range n.children {
			if visit(c) {
				n.shown = true
			}
		}
		return n.shown
	}
	visit(p)
}

// hide hides n and everything below it.
func hide(n *tnode) {
	n.match, n.shown = false, false
	for _, c := range n.children {
		hide(c)
	}
}

// searchHit returns the fields of n that contain the search text, ignoring letter case, and for a hit in
// the external name or description an excerpt of about 60 characters around it (FR-006); nil without a hit.
func searchHit(n *Node, text, lowerText string) *SearchHit {
	hit := &SearchHit{Fields: []string{}}
	for _, field := range []struct{ name, value string }{
		{"name", n.Name}, {"id", n.ID}, {"externalName", n.ExternalName}, {"description", n.Description},
	} {
		if !strings.Contains(strings.ToLower(field.value), lowerText) {
			continue
		}
		hit.Fields = append(hit.Fields, field.name)
		if hit.Excerpt == "" && (field.name == "externalName" || field.name == "description") {
			hit.Excerpt = excerptAround(field.value, text)
		}
	}
	if len(hit.Fields) == 0 {
		return nil
	}
	return hit
}

// excerptLength is the length of a search excerpt, in characters.
const excerptLength = 60

// excerptAround returns about 60 characters of value around the first place that holds text, ignoring
// letter case, on one line, with "…" where it was cut.
func excerptAround(value, text string) string {
	runes := []rune(strings.Join(strings.Fields(value), " "))
	if len(runes) <= excerptLength {
		return string(runes)
	}
	n := utf8.RuneCountInString(text)
	at := 0
	for i := 0; i+n <= len(runes); i++ {
		if strings.EqualFold(string(runes[i:i+n]), text) {
			at = i
			break
		}
	}
	end := min(len(runes), max(0, at-(excerptLength-n)/2)+excerptLength)
	start := max(0, end-excerptLength)
	excerpt := string(runes[start:end])
	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(runes) {
		excerpt += "…"
	}
	return excerpt
}

// countOf returns "Showing N of M <noun>" (FR-008, data model §3.5): activities — or campaigns, or
// initiatives, under the last two Show levels choices — with N the shown matches of that level and M every
// record of that level in the year's plans.
func countOf(t *tree, q Query, m matches) Count {
	switch q.ShowLevels {
	case ShowCampaignsWithoutInitiatives:
		return Count{Shown: m.campaigns, Total: t.total[LevelCampaign], Noun: "campaigns"}
	case ShowInitiativesWithoutActivities:
		return Count{Shown: m.initiatives, Total: t.total[LevelInitiative], Noun: "initiatives"}
	}
	return Count{Shown: m.activities.activities, Total: t.total[LevelActivity], Noun: "activities"}
}
