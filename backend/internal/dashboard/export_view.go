package dashboard

import (
	"strconv"

	"mapp/internal/platform/deck"
)

const (
	titleExecutive = "Executive dashboard"
	titleRegional  = "Regional dashboard"
	noneRow        = "None"
)

func (in *input) dashboardTitle() string {
	if in.kind == KindRegional && in.team != nil {
		return titleRegional + nameSeparator + in.team.Name
	}
	return titleExecutive
}

// dashboardDeck is a dashboard's file: the same panels as the screen, one or more pages each (FR-024).
func (in *input) dashboardDeck(v View, title string) deck.Deck {
	file := in.newDeck(title)
	planType := "All plan types"
	if v.Filters.PlanType != nil {
		planType = word(*v.Filters.PlanType) + " plans"
	}
	notes := []string{strconv.Itoa(v.PlanCount) + " plans, " + strconv.Itoa(v.PlansWithoutBudget) + " without a budget",
		"Financial figures cover " + strconv.Itoa(v.Finance.PlanCount) + " of " + strconv.Itoa(v.PlanCount) + " plans"}
	file.Pages = append(file.Pages, deck.Page{Title: title, Subtitle: yearText(v.Filters.Year) + " · " + planType,
		Cards: figureCards(v.Totals), Notes: append(notes, in.madeNotes()...)})
	if v.Kind == KindExecutive {
		file.Pages = append(file.Pages, marketsPage(v.Markets))
	}
	file.Pages = append(file.Pages, plansPage(v.Plans), monthsPage("Planned and actual by month", v.Months, v.Filters.Currency),
		campaignListPage("Campaigns most over budget", "Over budget by", v.OverBudget),
		campaignListPage("Campaigns with the most left to plan", "Left to plan", v.LeftToPlan))
	if v.PlanCampaigns != nil {
		file.Pages = append(file.Pages, campaignsPage(v.PlanCampaigns, v.Filters.Currency))
	}
	file.Pages = append(file.Pages, approvalsPage(v.Approvals), purchasesPage(v.PurchaseRequests), completenessPage(v.Completeness))
	if v.GlobalBudgets != nil {
		file.Pages = append(file.Pages, globalBudgetsPage(v.GlobalBudgets))
	}
	return file
}

func marketsPage(rows []MarketRow) deck.Page {
	table := &deck.Table{Columns: figureColumns(deck.Column{Title: "Market", Weight: 3}, deck.Column{Title: wordPlans, Weight: 1, Align: deck.Right})}
	for _, r := range rows {
		cells := append([]string{r.Team.Name, strconv.Itoa(r.PlanCount)}, figureCells(r.Figures)...)
		table.Rows = append(table.Rows, deck.Row{Cells: cells})
	}
	return withNone(deck.Page{Title: "By market", Subtitle: "A plan shared by two markets counts in both rows", Table: table})
}

func plansPage(rows []PlanRow) deck.Page {
	table := &deck.Table{Columns: figureColumns(deck.Column{Title: labelPlan, Weight: 4}, deck.Column{Title: wordStatus, Weight: 2})}
	for _, r := range rows {
		cells := append([]string{r.Plan.Name, word(r.Plan.Status)}, figureCells(r.Figures)...)
		table.Rows = append(table.Rows, deck.Row{Cells: cells})
	}
	return withNone(deck.Page{Title: "By plan", Table: table})
}

func campaignListPage(title, difference string, rows []CampaignRow) deck.Page {
	table := &deck.Table{Columns: []deck.Column{{Title: labelCampaign, Weight: 4}, {Title: labelPlan, Weight: 4},
		{Title: labelBudget, Weight: 3, Align: deck.Right}, {Title: labelPlanned, Weight: 3, Align: deck.Right},
		{Title: difference, Weight: 3, Align: deck.Right}}}
	for _, r := range rows {
		table.Rows = append(table.Rows, deck.Row{Cells: []string{r.Campaign.Name, r.Plan.Name, moneyText(r.Figures.Budget),
			moneyText(r.Figures.PlannedCost), moneyText(r.Difference)}})
	}
	return withNone(deck.Page{Title: title, Subtitle: "Up to five campaigns with a budget", Table: table})
}

func approvalsPage(a Approvals) deck.Page {
	oldest := none
	if a.OldestWaitingSince != nil {
		oldest = a.OldestWaitingSince.UTC().Format(dayLayout)
	}
	average := none
	if a.AverageDecisionDays != nil {
		average = *a.AverageDecisionDays + " days"
	}
	table := &deck.Table{Columns: []deck.Column{{Title: "Activity status", Weight: 3}, {Title: wordActivities, Weight: 1, Align: deck.Right}}}
	for _, s := range a.ActivitiesByStatus {
		table.Rows = append(table.Rows, deck.Row{Cells: []string{word(s.Status), strconv.Itoa(s.Count)}})
	}
	return deck.Page{Title: "Approvals", Subtitle: "Activities and purchase requests", Table: table, Cards: []deck.Card{
		{Label: "Waiting for a decision", Value: strconv.Itoa(a.Waiting), Note: "Oldest since " + oldest},
		{Label: wordApproved, Value: strconv.Itoa(a.Approved)}, {Label: wordDeclined, Value: strconv.Itoa(a.Declined)},
		{Label: "Approved automatically", Value: strconv.Itoa(a.AutoApproved)},
		{Label: "Average time to decide", Value: average},
	}}
}

func purchasesPage(p PurchaseRequests) deck.Page {
	table := &deck.Table{Columns: []deck.Column{{Title: wordStatus, Weight: 3}, {Title: wordPurchaseRequests, Weight: 2, Align: deck.Right},
		{Title: wordAmount, Weight: 3, Align: deck.Right}}}
	for _, s := range p.ByStatus {
		table.Rows = append(table.Rows, deck.Row{Cells: []string{word(s.Status), strconv.Itoa(s.Count), s.Amount.Format()}})
	}
	table.Rows = append(table.Rows, deck.Row{Cells: []string{"Total", strconv.Itoa(p.Count), p.Amount.Format()}, Bold: true})
	page := deck.Page{Title: wordPurchaseRequests, Table: table, Cards: []deck.Card{
		{Label: wordPurchaseRequests, Value: strconv.Itoa(p.Count)}, {Label: wordAmount, Value: p.Amount.Format()},
		{Label: "Waiting for GSAP", Value: strconv.Itoa(p.WaitingForGSAP)}}}
	if !p.Available {
		page.Notes = []string{"Purchase requests are switched off in Settings"}
	}
	return page
}

func completenessPage(levels []Completeness) deck.Page {
	table := &deck.Table{Columns: []deck.Column{{Title: "Level", Weight: 3}, {Title: "Records", Weight: 2, Align: deck.Right},
		{Title: "All filled in", Weight: 2, Align: deck.Right}, {Title: "Share", Weight: 2, Align: deck.Right}}}
	for _, l := range levels {
		table.Rows = append(table.Rows, deck.Row{Cells: []string{word(l.Level), strconv.Itoa(l.Total), strconv.Itoa(l.Complete),
			percentText(l.Percent)}})
	}
	return deck.Page{Title: "Plan details", Subtitle: "Records with every needed field filled in", Table: table}
}

func globalBudgetsPage(budgets []GlobalBudget) deck.Page {
	table := &deck.Table{Columns: []deck.Column{{Title: "Global budget", Weight: 4}, {Title: wordAmount, Weight: 3, Align: deck.Right},
		{Title: "Allocated", Weight: 3, Align: deck.Right}, {Title: "Unallocated", Weight: 3, Align: deck.Right},
		{Title: labelPlanned, Weight: 3, Align: deck.Right}}}
	for _, g := range budgets {
		name := g.Name
		if g.OverAllocated {
			name += " (over-allocated)"
		}
		table.Rows = append(table.Rows, deck.Row{Cells: []string{name, g.Amount.Format(), g.Allocated.Format(),
			g.Unallocated.Format(), g.PlannedCost.Format()}})
	}
	return withNone(deck.Page{Title: "Global budgets", Table: table})
}

// withNone writes "None" in an empty table, so a page never looks unfinished.
func withNone(p deck.Page) deck.Page {
	if len(p.Table.Rows) == 0 {
		p.Table.Rows = []deck.Row{{Cells: []string{noneRow}}}
	}
	return p
}
