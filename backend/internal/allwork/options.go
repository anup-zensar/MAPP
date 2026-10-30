package allwork

import (
	"cmp"
	"slices"
	"strings"
)

// The choices of the filter bar and the column chooser, as GET /api/v1/all-work/options answers them.

var groupLabels = []struct{ group, label string }{
	{GroupStandard, "Standard"},
	{GroupPlan, "Plan"},
	{GroupCampaign, "Campaign"},
	{GroupInitiative, "Initiative"},
	{GroupActivity, "Activity"},
	{GroupLineItem, "Line item"},
	{GroupFinancials, "Financials"},
}

// optionsOf returns the year's choices: years newest first, then plans, owners and teams (own first) by name.
func optionsOf(year, currentYear int, years []int, data YearData, c Choices, catalogue []Column) Options {
	o := Options{
		Year: year, CurrentYear: currentYear, Years: []int{currentYear}, Plans: []PlanOption{}, Owners: []Option{},
		Teams: slices.Clone(c.Teams), BusinessUnits: orNone(c.BusinessUnits), ActivityTypes: orNone(c.ActivityTypes),
		ColumnGroups: columnGroupsOf(catalogue), DefaultColumns: defaultColumnsIn(catalogueOf(catalogue)),
		YearFrom: year, YearTo: year, PlanTypes: slices.Clone(planTypeValues()), PlanCurrencies: []string{},
		Attributes: []FilterableAttribute{},
	}
	for _, y := range years {
		if !slices.Contains(o.Years, y) {
			o.Years = append(o.Years, y)
		}
	}
	slices.SortFunc(o.Years, func(x, y int) int { return cmp.Compare(y, x) })
	owners := map[string]bool{}
	for _, n := range data.Nodes {
		if n.Level == LevelPlan {
			o.Plans = append(o.Plans, PlanOption{ID: n.ID, Label: n.Name, Status: n.Status, Strategic: n.Strategic,
				Year: yearOf(&n, year), PlanType: n.PlanType, Currency: n.PlanCurrency})
			if n.PlanCurrency != "" && !slices.Contains(o.PlanCurrencies, n.PlanCurrency) {
				o.PlanCurrencies = append(o.PlanCurrencies, n.PlanCurrency)
			}
		}
		if n.Level != LevelLineItem && n.Owner != nil && !owners[n.Owner.ID] {
			owners[n.Owner.ID] = true
			o.Owners = append(o.Owners, Option{ID: n.Owner.ID, Label: n.Owner.Name, Inactive: n.Owner.Inactive})
		}
	}
	slices.SortFunc(o.Plans, func(x, y PlanOption) int { return compareNames(x.Label, x.ID, y.Label, y.ID) })
	slices.Sort(o.PlanCurrencies)
	slices.SortFunc(o.Owners, func(x, y Option) int { return compareNames(x.Label, x.ID, y.Label, y.ID) })
	if o.Teams == nil {
		o.Teams = []Option{}
	}
	slices.SortFunc(o.Teams, func(x, y Option) int {
		if x.Mine != y.Mine {
			if x.Mine {
				return -1
			}
			return 1
		}
		return compareNames(x.Label, x.ID, y.Label, y.ID)
	})
	return o
}

// filterableAttributes are the active choice or yes/no attributes, top-down in Settings' order, with their options.
func filterableAttributes(attributes []Attribute) []FilterableAttribute {
	list := []FilterableAttribute{}
	for _, a := range attributes {
		if !a.filterable() {
			continue
		}
		f := FilterableAttribute{ID: a.ID, Label: a.Label, Level: a.Level, Type: a.Type, Values: []Option{}}
		if a.Type == TypeYesNo {
			f.Values = append(f.Values, Option{ID: AttrYes, Label: labelYes}, Option{ID: AttrNo, Label: labelNo})
		}
		for _, o := range a.Options {
			if !o.Inactive {
				f.Values = append(f.Values, o)
			}
		}
		list = append(list, f)
	}
	return list
}

func columnGroupsOf(catalogue []Column) []ColumnGroup {
	groups := []ColumnGroup{}
	for _, g := range groupLabels {
		group := ColumnGroup{Group: g.group, Label: g.label, Columns: []Column{}}
		for _, c := range catalogue {
			if c.Group == g.group {
				group.Columns = append(group.Columns, c)
			}
		}
		if len(group.Columns) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

func compareNames(xName, xID, yName, yID string) int {
	return cmp.Or(strings.Compare(strings.ToLower(xName), strings.ToLower(yName)), compareIDs(xID, yID))
}

func orNone(options []Option) []Option {
	if options == nil {
		return []Option{}
	}
	return options
}
