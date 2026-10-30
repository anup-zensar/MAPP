package dashboard

import (
	"cmp"
	"slices"
	"strings"

	"mapp/internal/allwork"
	"mapp/internal/platform/deck"
	"mapp/internal/platform/money"
)

// planDeck is a plan's file: its title page, its summary by month, its campaigns and its activities.
func (in *input) planDeck() deck.Deck {
	b, d := in.board, in.display
	p := b.plans[0]
	v := in.view()
	row := v.Plans[0]
	file := in.newDeck(p.Name)
	file.Pages = []deck.Page{
		{Title: p.Name, Subtitle: strings.Join([]string{"Plan", yearText(in.query.Year), word(p.Status), word(p.PlanType)}, " · "),
			Cards: figureCards(row.Figures), Notes: append(in.planNotes(p, row), in.madeNotes()...)},
		monthsPage("Planned and actual by month", v.Months, in.query.Currency),
		campaignsPage(b.planCampaigns(p.ID, d), in.query.Currency),
		{Title: wordActivities, Subtitle: p.Name, Table: in.activitiesTable()},
	}
	return file
}

func (in *input) planNotes(p *allwork.Node, row PlanRow) []string {
	markets := make([]string, 0, len(row.Teams))
	for _, t := range row.Teams {
		markets = append(markets, t.Name)
	}
	notes := []string{"Markets: " + cmp.Or(strings.Join(markets, ", "), none)}
	if p.Owner != nil {
		notes = append(notes, "Owner: "+p.Owner.Name)
	}
	if !row.FinanceShown {
		notes = append(notes, financeNote)
	}
	return notes
}

func campaignsPage(campaigns *PlanCampaigns, currency string) deck.Page {
	chart := &deck.Chart{Series: []deck.Series{{Name: labelBudget, Colour: deck.ChartBudget},
		{Name: labelPlanned, Colour: deck.ChartPlanned}, {Name: labelActual, Colour: deck.ChartActual}}}
	table := &deck.Table{Columns: figureColumns(deck.Column{Title: labelCampaign, Weight: 4})}
	var largest money.Amount
	for _, c := range campaigns.Items {
		values := []*money.Money{c.Figures.Budget, c.Figures.PlannedCost, c.Figures.Actual}
		chart.Categories = append(chart.Categories, c.Campaign.Name)
		for i, m := range values {
			cents := int64(0)
			if m != nil {
				cents = m.Amount.Cents()
				largest = maxAmount(largest, m.Amount)
			}
			chart.Series[i].Values = append(chart.Series[i].Values, cents)
		}
		table.Rows = append(table.Rows, deck.Row{Cells: append([]string{c.Campaign.Name}, figureCells(c.Figures)...)})
	}
	chart.MaxLabel = "Scale up to " + money.Money{Amount: largest, Currency: currency}.Format()
	return deck.Page{Title: wordCampaigns, Subtitle: "Budget, planned cost and actual by campaign", Chart: chart, Table: table}
}

// activitiesTable lists the plan's activities in its tree's order with their line items' GSAP figures.
func (in *input) activitiesTable() *deck.Table {
	b, d := in.board, in.display
	names := map[string]string{}
	for _, c := range b.byLevel[allwork.LevelCampaign] {
		names[c.ID] = c.Name
	}
	activities := slices.Clone(b.byLevel[allwork.LevelActivity])
	slices.SortStableFunc(activities, func(x, y *allwork.Node) int { return cmp.Compare(x.Order, y.Order) })
	spent := b.activityFinance()
	table := &deck.Table{Columns: []deck.Column{{Title: "Activity", Weight: 4}, {Title: labelCampaign, Weight: 3},
		{Title: wordStatus, Weight: 2}, {Title: "Dates", Weight: 4}, {Title: labelPlanned, Weight: 3, Align: deck.Right},
		{Title: labelCommitted, Weight: 3, Align: deck.Right}, {Title: labelActual, Weight: 3, Align: deck.Right}}}
	for _, a := range activities {
		s := spent[a.ID]
		table.Rows = append(table.Rows, deck.Row{Cells: []string{a.Name, names[b.campaignOf[a.ID]], word(a.Status),
			datesText(a.StartDate, a.EndDate, a.Ongoing), moneyText(optionalShown(d, a.PlannedCost)),
			moneyText(optionalShown(d, s.committed)), moneyText(optionalShown(d, s.actual))}})
	}
	return table
}

// activityFinance adds up each activity's line items' GSAP figures, in the plans whose finance counts.
func (b *board) activityFinance() map[string]sums {
	out := map[string]sums{}
	for _, a := range b.byLevel[allwork.LevelActivity] {
		if b.finance[a.PlanID] {
			out[a.ID] = b.start(a)
		}
	}
	for _, li := range b.byLevel[allwork.LevelLineItem] {
		if s, counted := out[li.ParentID]; counted {
			out[li.ParentID] = s.plus(b.spent[li.ID])
		}
	}
	return out
}
