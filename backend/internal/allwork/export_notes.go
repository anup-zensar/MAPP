package allwork

import (
	"strconv"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
)

const (
	labelNone = "None"
	labelYear = "Year"
)

func (e exporter) notes(a actor.Actor) [][2]string {
	r := e.b.result
	notes := [][2]string{{"View", viewNames[r.View]}, {labelYear, strconv.Itoa(r.Year)}}
	if r.YearTo > r.YearFrom && r.YearFrom != 0 {
		notes[1] = [2]string{"Years", yearsText(r, " – ")}
	}
	chips := e.chips()
	if len(chips) == 0 {
		chips = []string{labelNone}
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
		search = labelNone
	}
	now := e.in.now.UTC()
	if showsFinance(e.b.result.Columns) {
		notes = append(notes, financeNote(e.in.data.FinanceAsOf))
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
	chips = append(chips, bcpirChips(q)...)
	add("Plan type", q.PlanTypes, func(v string) string { return planTypeWords[v] })
	add("Plan currency", q.PlanCurrencies, func(v string) string { return v })
	for _, f := range q.attrs {
		labels := make([]string, 0, len(f.values))
		for _, v := range f.values {
			labels = append(labels, attributeValueLabel(f.attribute, v))
		}
		chips = append(chips, f.attribute.Label+": "+strings.Join(labels, ", "))
	}
	return chips
}

func attributeValueLabel(a Attribute, v string) string {
	switch v {
	case AttrNotFilledIn:
		return labelNotFilledIn
	case AttrYes:
		return labelYes
	case AttrNo:
		return labelNo
	}
	return optionLabel(a.Options)(v)
}

func chipLabel(family map[string]chip) func(string) string {
	return func(v string) string {
		if c, found := family[v]; found {
			return c.label
		}
		return v
	}
}

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
