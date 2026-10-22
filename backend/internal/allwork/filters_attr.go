package allwork

import (
	"slices"

	"mapp/internal/platform/money"
)

// Shell-attribute filters (feature 016, FR-039, D-270): under More filters, any active choice or yes/no Shell
// attribute; its values — options, Yes, No — and Not filled in can be chosen, several at once. Each filter is
// tested on its attribute's level with the matching rules of 010 FR-005 (D-113): records of that level must
// meet it, records below take their ancestor's answer, records above need a match below. A % split value
// matches when its share is above 0; a value hidden by its Business unit condition counts as not filled in.
// Line items are never tested themselves (D-113), so a line-item attribute is tested on activities: an
// activity meets it when one of its line items does.

// attrTest is an attribute filter resolved with Settings' attribute and the Business units of the plans.
type attrTest struct {
	attribute Attribute
	values    []string
	units     map[string][]string // plan id → its Business unit options
}

// resolveAttrs returns the attribute filters of q that name a filterable attribute, with what their tests need;
// filters of attributes that no longer exist or cannot be filtered are ignored (FR-031).
func resolveAttrs(filters []AttrFilter, attributes []Attribute, nodes []Node) []attrTest {
	units := map[string][]string{}
	for _, n := range nodes {
		if n.Level == LevelPlan {
			units[n.ID] = n.BusinessUnitIDs
		}
	}
	tests := []attrTest{}
	for _, f := range filters {
		i := slices.IndexFunc(attributes, func(a Attribute) bool { return a.ID == f.AttributeID })
		if i < 0 || !attributes[i].filterable() || len(f.Values) == 0 {
			continue
		}
		tests = append(tests, attrTest{attribute: attributes[i], values: f.Values, units: units})
	}
	return tests
}

// filter returns the filter of the attribute test: tested on the attribute's level, or on activities for a
// line-item attribute.
func (a attrTest) filter() filter {
	if a.attribute.Level == LevelLineItem {
		return filter{levels: []Level{LevelActivity}, meets: func(*Node) bool { return false }, test: func(t *tnode) bool {
			if len(t.children) == 0 {
				return slices.Contains(a.values, AttrNotFilledIn)
			}
			return slices.ContainsFunc(t.children, func(li *tnode) bool { return a.meets(li.n) })
		}}
	}
	return filter{levels: []Level{a.attribute.Level}, meets: a.meets}
}

// meets reports whether the record n has one of the chosen values (FR-039): an option it holds (a % split's
// with a share above 0), Yes or No, or Not filled in when it has no value or the value is hidden by the
// attribute's Business unit condition.
func (a attrTest) meets(n *Node) bool {
	v, filled := valueOf(a.attribute, n, a.units[n.PlanID])
	return slices.ContainsFunc(a.values, func(want string) bool {
		switch want {
		case AttrNotFilledIn:
			return !filled
		case AttrYes, AttrNo:
			return filled && v.Yes != nil && *v.Yes == (want == AttrYes)
		}
		return filled && slices.Contains(v.OptionIDs, want) && (v.Shares == nil || shareOf(v, want).Hundredths() > 0)
	})
}

// valueOf returns the value of the attribute on the record n of a plan with the Business unit options units,
// and false when it is not filled in: no value, or hidden by the attribute's condition.
func valueOf(a Attribute, n *Node, units []string) (AttrValue, bool) {
	v, found := n.Attrs[a.ID]
	if !found || !a.shownFor(units) || (len(v.OptionIDs) == 0 && v.Yes == nil) {
		return AttrValue{}, false
	}
	return v, true
}

// shareOf returns the share of the option in a % split value; 0 when it has none.
func shareOf(v AttrValue, optionID string) money.Percent {
	if v.Shares == nil {
		return money.Percent{}
	}
	return v.Shares[optionID]
}
