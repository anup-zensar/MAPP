package allwork

import (
	"slices"
	"strings"
	"unicode/utf8"
)

var levelRank = map[Level]int{LevelPlan: 0, LevelCampaign: 1, LevelInitiative: 2, LevelActivity: 3, LevelLineItem: 4}

type filter struct {
	levels []Level
	meets  func(n *Node) bool
	test   func(t *tnode) bool
}

var progressValues = []string{"noActivities", statusInProgress, "allApproved"}

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
	filters = append(filters, bcpirFilter(q)...)
	// 016 FR-037, FR-039: the plan type and plan currency of plans, and the Shell-attribute filters.
	add(q.PlanTypes, plans, func(n *Node) bool { return slices.Contains(q.PlanTypes, n.PlanType) })
	add(q.PlanCurrencies, plans, func(n *Node) bool { return slices.Contains(q.PlanCurrencies, n.PlanCurrency) })
	for _, a := range q.attrs {
		filters = append(filters, a.filter())
	}
	return filters
}

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

func containsAny(list, values []string) bool {
	return slices.ContainsFunc(list, func(v string) bool { return slices.Contains(values, v) })
}

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

func (f filter) on(t *tnode) bool {
	if f.test != nil {
		return f.test(t)
	}
	return f.meets(t.n)
}

func applyFilters(t *tree, q Query) {
	filters := filtersOf(q)
	if len(q.Finance) > 0 {
		filters = append(filters, financeFilter(q.Finance, t))
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

const excerptLength = 60

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

func countOf(t *tree, q Query, m matches) Count {
	switch q.ShowLevels {
	case ShowCampaignsWithoutInitiatives:
		return Count{Shown: m.campaigns, Total: t.total[LevelCampaign], Noun: "campaigns"}
	case ShowInitiativesWithoutActivities:
		return Count{Shown: m.initiatives, Total: t.total[LevelInitiative], Noun: "initiatives"}
	}
	return Count{Shown: m.activities.activities, Total: t.total[LevelActivity], Noun: nounActivities}
}
