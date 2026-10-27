package allwork

import (
	"strconv"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

// notes returns the notes before the data (FR-034): the view, the year, one line per active filter (or
// None), the search, Show levels, the display currency, who exported it and when.
func (e exporter) notes(a actor.Actor) [][2]string {
	r := e.b.result
	notes := [][2]string{{"View", viewNames[r.View]}, {"Year", strconv.Itoa(r.Year)}}
	if r.YearTo > r.YearFrom && r.YearFrom != 0 { // 016 FR-034: the range
		notes[1] = [2]string{"Years", yearsText(r, " – ")}
	}
	chips := e.chips()
	if len(chips) == 0 {
		chips = []string{"None"}
	}
	for i, chip := range chips {
		label := ""
		if i == 0 {
			label = "Filters"
		}
		notes = append(notes, [2]string{label, chip})
	}
	search := e.b.query.Search
	if search == "" {
		search = "None"
	}
	now := e.in.now.UTC()
	if showsFinance(e.b.result.Columns) {
		notes = append(notes, financeNote(e.in.data.FinanceAsOf)) // feature 015 (FR-023)
	}
	notes = append(notes,
		[2]string{"Search", search},
		[2]string{"Show levels", showLevelsWords[e.b.query.ShowLevels]},
		[2]string{"Display currency", r.DisplayCurrency})
	notes = append(notes, e.rateNotes()...)
	return append(notes,
		[2]string{"Exported by", a.Name},
		[2]string{"Exported at", date.Format(date.Today(now)) + ", " + now.Format("15:04") + " UTC"},
	)
}

// rateNotes are the rates the export's amounts were converted with (016 FR-009), one line each.
func (e exporter) rateNotes() [][2]string {
	notes := [][2]string{}
	for i, line := range e.in.rateNotes {
		label := ""
		if i == 0 {
			label = "Rates used"
		}
		notes = append(notes, [2]string{label, line})
	}
	return notes
}

// chips returns the active filters as their chips read, "<Filter>: <value>", in the order of the filter bar
// (FR-008).
func (e exporter) chips() []string {
	q := e.b.query
	var chips []string
	add := func(filter string, values []string, label func(string) string) {
		for _, v := range values {
			chips = append(chips, filter+": "+label(v))
		}
	}
	add("Plan", q.Plans, func(id string) string {
		if n, found := e.b.tree.byID[id]; found {
			return n.n.Name
		}
		return id
	})
	add("Plan status", q.PlanStatuses, chipLabel(planStatuses))
	add("Campaign / initiative status", q.ContainerStatuses, func(v string) string {
		if c, found := progressStatuses[v]; found {
			return c.label
		}
		return chipLabel(timingStatuses)(v)
	})
	add("Activity status", q.ActivityStatuses, chipLabel(activityStatuses))
	add("Owner", q.Owners, optionLabel(e.in.choices.People))
	add("Team", q.Teams, optionLabel(e.in.choices.Teams))
	add("Business unit", q.BusinessUnits, optionLabel(e.in.choices.BusinessUnits))
	add("Activity type", q.ActivityTypes, optionLabel(e.in.choices.ActivityTypes))
	add("Financials", q.Finance, func(v string) string { return financeWords[v] })
	switch q.Strategic {
	case StrategicOnly:
		chips = append(chips, "Strategic / AOP plans: Only")
	case StrategicExclude:
		chips = append(chips, "Strategic / AOP plans: Exclude")
	}
	add("Plan type", q.PlanTypes, func(v string) string { return planTypeWords[v] }) // 016 FR-037
	add("Plan currency", q.PlanCurrencies, func(v string) string { return v })
	for _, f := range q.attrs { // 016 FR-039: "Brands: Helix, Not filled in"
		labels := make([]string, 0, len(f.values))
		for _, v := range f.values {
			labels = append(labels, attributeValueLabel(f.attribute, v))
		}
		chips = append(chips, f.attribute.Label+": "+strings.Join(labels, ", "))
	}
	return chips
}

// attributeValueLabel is a value of an attribute filter as its chip reads it.
func attributeValueLabel(a Attribute, v string) string {
	switch v {
	case AttrNotFilledIn:
		return labelNotFilledIn
	case AttrYes:
		return "Yes"
	case AttrNo:
		return "No"
	}
	return optionLabel(a.Options)(v)
}

// chipLabel returns the word of a status value of family.
func chipLabel(family map[string]chip) func(string) string {
	return func(v string) string {
		if c, found := family[v]; found {
			return c.label
		}
		return v
	}
}

// optionLabel returns the name of an option ID among options.
func optionLabel(options []Option) func(string) string {
	return func(id string) string {
		for _, o := range options {
			if o.ID == id {
				return o.Label
			}
		}
		return id
	}
}
