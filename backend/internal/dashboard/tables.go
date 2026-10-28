package dashboard

import (
	"cmp"
	"slices"
	"strings"

	"mapp/internal/allwork"
	"mapp/internal/people"
	"mapp/internal/platform/money"
)

const listLength = 5

// markets adds every plan to the row of each of its teams, so a plan of two teams counts in both rows.
func (b *board) markets(teams map[string]people.TeamRef, d money.Display) []MarketRow {
	type market struct {
		plans int
		sums  sums
	}
	byTeam := map[string]*market{}
	for _, p := range b.plans {
		for _, id := range p.TeamIDs {
			m := byTeam[id]
			if m == nil {
				m = &market{}
				byTeam[id] = m
			}
			m.plans++
			m.sums = m.sums.plus(b.own[p.ID])
		}
	}
	rows := make([]MarketRow, 0, len(byTeam))
	for id, m := range byTeam {
		rows = append(rows, MarketRow{Team: teamRef(teams, id), PlanCount: m.plans, Figures: m.sums.figures(d)})
	}
	slices.SortFunc(rows, func(x, y MarketRow) int { return byName(x.Team.Name, y.Team.Name, x.Team.ID, y.Team.ID) })
	return rows
}

func (b *board) planRows(teams map[string]people.TeamRef, d money.Display) []PlanRow {
	rows := make([]PlanRow, 0, len(b.plans))
	for _, p := range b.plans {
		row := PlanRow{Plan: PlanRef{ID: p.ID, Name: p.Name, Status: p.Status, PlanType: p.PlanType},
			Teams: make([]people.TeamRef, 0, len(p.TeamIDs)), Figures: b.own[p.ID].figures(d), FinanceShown: b.finance[p.ID]}
		for _, id := range p.TeamIDs {
			row.Teams = append(row.Teams, teamRef(teams, id))
		}
		slices.SortFunc(row.Teams, func(x, y people.TeamRef) int { return byName(x.Name, y.Name, x.ID, y.ID) })
		rows = append(rows, row)
	}
	return rows
}

// campaignLists are the campaigns most over budget and those with the most budget left to plan, five each.
func (b *board) campaignLists(d money.Display) (over, left []CampaignRow) {
	type ranked struct {
		node *allwork.Node
		diff money.Amount
	}
	var overs, lefts []ranked
	for _, c := range b.byLevel[allwork.LevelCampaign] {
		s := b.own[c.ID]
		if s.budget == nil {
			continue
		}
		planned := money.Amount{}
		if s.planned != nil {
			planned = *s.planned
		}
		switch diff := planned.Sub(*s.budget); diff.Sign() {
		case 1:
			overs = append(overs, ranked{c, diff})
		case -1:
			lefts = append(lefts, ranked{c, diff.Neg()})
		}
	}
	rows := func(list []ranked) []CampaignRow {
		slices.SortFunc(list, func(x, y ranked) int {
			return cmp.Or(y.diff.Cmp(x.diff), byName(x.node.Name, y.node.Name, x.node.ID, y.node.ID))
		})
		out := []CampaignRow{}
		for _, r := range list[:min(len(list), listLength)] {
			out = append(out, b.campaignRow(r.node, d, optionalShown(d, &r.diff)))
		}
		return out
	}
	return rows(overs), rows(lefts)
}

func (b *board) campaignRow(c *allwork.Node, d money.Display, difference *money.Money) CampaignRow {
	plan := RecordRef{ID: c.PlanID}
	for _, p := range b.plans {
		if p.ID == c.PlanID {
			plan.Name = p.Name
		}
	}
	return CampaignRow{Campaign: RecordRef{ID: c.ID, Name: c.Name}, Plan: plan, Figures: b.own[c.ID].figures(d),
		Difference: difference}
}

// planCampaigns lists every campaign of the plan in the plan's order; nil when the plan is not on the board.
func (b *board) planCampaigns(planID string, d money.Display) *PlanCampaigns {
	if planID == "" || !b.inView(planID) {
		return nil
	}
	var campaigns []*allwork.Node
	for _, c := range b.byLevel[allwork.LevelCampaign] {
		if c.PlanID == planID {
			campaigns = append(campaigns, c)
		}
	}
	slices.SortStableFunc(campaigns, func(x, y *allwork.Node) int { return cmp.Compare(x.Order, y.Order) })
	out := &PlanCampaigns{Items: make([]CampaignRow, 0, len(campaigns))}
	for _, c := range campaigns {
		row := b.campaignRow(c, d, nil)
		out.Plan, out.Items = row.Plan, append(out.Items, row)
	}
	if len(campaigns) == 0 {
		out.Plan = b.campaignRow(&allwork.Node{PlanID: planID}, d, nil).Plan
	}
	return out
}

func teamRef(teams map[string]people.TeamRef, id string) people.TeamRef {
	if t, found := teams[id]; found {
		return t
	}
	return people.TeamRef{ID: id, Name: id}
}

func byName(x, y, xID, yID string) int {
	return cmp.Or(strings.Compare(strings.ToLower(x), strings.ToLower(y)), strings.Compare(xID, yID))
}
