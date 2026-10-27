package allwork

import (
	"fmt"
	"strings"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
)

// The rule tests build their years by hand, dated 18 Oct 2026 (plan "Tests"), and read them through an
// in-memory Source. seedYear reproduces the seed story of docs/product/demo-users-and-data.md.

// fixtureToday is the date of every pure rule test.
var fixtureToday = date.MustParse("2026-10-18")

// fixtureNow is the moment of every pure rule test.
var fixtureNow = time.Date(2026, 10, 18, 14, 5, 0, 0, time.UTC)

// The people of the fixtures.
var (
	priya  = &Owner{ID: "U-1002", Name: "Priya Raman"}
	daniel = &Owner{ID: "U-1003", Name: "Daniel Okoro"}
	mei    = &Owner{ID: "U-1004", Name: "Mei Lin"}
	sara   = &Owner{ID: "U-1006", Name: "Sara Novak", Inactive: true}
)

// yearBuilder builds the nodes of one year. Each record gets the next order among its siblings; a line
// item without months of its own is spread evenly over its activity's months, or cannot be spread while the
// activity has no dates.
type yearBuilder struct {
	nodes  []Node
	index  map[string]int // node id → its index in nodes
	orders map[string]int // parent id → the next order
	custom map[string]bool
}

// newYear returns an empty year.
func newYear() *yearBuilder {
	return &yearBuilder{index: map[string]int{}, orders: map[string]int{}, custom: map[string]bool{}}
}

// option changes a node being built.
type option func(*Node)

// add adds a node under parentID and returns its id.
func (b *yearBuilder) add(level Level, parentID, id, name string, opts []option) string {
	n := Node{Level: level, ID: id, ParentID: parentID, Name: name, Order: b.orders[parentID],
		CreatedAt: fixtureNow.Add(-48 * time.Hour), UpdatedAt: fixtureNow.Add(-24 * time.Hour)}
	b.orders[parentID]++
	n.PlanID = id
	if parentID != "" {
		n.PlanID = b.node(parentID).PlanID
	}
	switch level {
	case LevelPlan:
		n.Status = "draft"
	case LevelCampaign, LevelInitiative:
		n.Status = "noActivities"
	case LevelActivity:
		n.Status = "created"
	case LevelLineItem:
		n.Swbs = "notRequested"
	}
	for _, opt := range opts {
		opt(&n)
	}
	b.index[id] = len(b.nodes)
	b.nodes = append(b.nodes, n)
	return id
}

// node returns the node id built so far.
func (b *yearBuilder) node(id string) *Node {
	i, found := b.index[id]
	if !found {
		panic("fixture: no node " + id)
	}
	return &b.nodes[i]
}

func (b *yearBuilder) plan(id, name string, opts ...option) string {
	return b.add(LevelPlan, "", id, name, opts)
}

func (b *yearBuilder) campaign(planID, id, name string, opts ...option) string {
	return b.add(LevelCampaign, planID, id, name, opts)
}

func (b *yearBuilder) initiative(campaignID, id, name string, opts ...option) string {
	return b.add(LevelInitiative, campaignID, id, name, opts)
}

func (b *yearBuilder) activity(initiativeID, id, name string, opts ...option) string {
	return b.add(LevelActivity, initiativeID, id, name, opts)
}

// lineItem adds a line item costing usd US dollars (its local amount is the same in USD unless local says
// otherwise).
func (b *yearBuilder) lineItem(activityID, id, name, usd string, opts ...option) string {
	amount := amountOf(usd)
	all := append([]option{func(n *Node) {
		n.PlannedCost = &amount
		n.LocalPlannedCost = &money.Money{Amount: amount, Currency: "USD"}
	}}, opts...)
	return b.add(LevelLineItem, activityID, id, name, all)
}

// months gives a line item its own month amounts in USD (a custom spread).
func (b *yearBuilder) months(lineItemID string, amounts ...string) {
	n := b.node(lineItemID)
	n.MonthsUSD = []MonthAmount{}
	for i := 0; i+1 < len(amounts); i += 2 {
		n.MonthsUSD = append(n.MonthsUSD, MonthAmount{Month: amounts[i], USD: amountOf(amounts[i+1])})
	}
	b.custom[lineItemID] = true
}

// data returns the year: activities get their months, line items their spread.
func (b *yearBuilder) data() YearData {
	nodes := make([]Node, len(b.nodes))
	copy(nodes, b.nodes)
	months := map[string][]string{}
	for i := range nodes {
		n := &nodes[i]
		if n.Level == LevelActivity {
			n.Months = activityMonths(n)
			months[n.ID] = n.Months
		}
	}
	for i := range nodes {
		n := &nodes[i]
		if n.Level != LevelLineItem || b.custom[n.ID] {
			continue
		}
		n.MonthsUSD = []MonthAmount{}
		of := months[n.ParentID]
		if len(of) == 0 {
			n.UnspreadUSD = *n.PlannedCost
			continue
		}
		for j, part := range n.PlannedCost.SplitEvenly(len(of)) {
			n.MonthsUSD = append(n.MonthsUSD, MonthAmount{Month: of[j], USD: part})
		}
	}
	return YearData{Nodes: nodes}
}

// activityMonths returns the months of an activity's dates as 007 works them out: the start month only
// when ongoing; none without dates.
func activityMonths(n *Node) []string {
	if n.StartDate == nil || (!n.Ongoing && n.EndDate == nil) {
		return []string{}
	}
	start := n.StartDate.Time()
	if n.Ongoing {
		return []string{start.Format("2006-01")}
	}
	var list []string
	for m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC); !m.After(n.EndDate.Time()); m = m.AddDate(0, 1, 0) {
		list = append(list, m.Format("2006-01"))
	}
	return list
}

// The options of the builder.

func owner(o *Owner) option { return func(n *Node) { n.Owner = o } }

func budget(usd string) option {
	return func(n *Node) { a := amountOf(usd); n.Budget = &a }
}

func planned(usd string) option {
	return func(n *Node) { a := amountOf(usd); n.PlannedCost = &a }
}

func dates(start, end string) option {
	return func(n *Node) {
		s, e := date.MustParse(start), date.MustParse(end)
		n.StartDate, n.EndDate = &s, &e
	}
}

func ongoingFrom(start string) option {
	return func(n *Node) { s := date.MustParse(start); n.StartDate, n.Ongoing = &s, true }
}

func status(value string) option { return func(n *Node) { n.Status = value } }

func progress(value string, approved, total int) option {
	return func(n *Node) { n.Status, n.Approved, n.Total = value, approved, total }
}

func timing(value string) option { return func(n *Node) { n.Timing = value } }

func missing(count int) option { return func(n *Node) { n.Missing = count } }

func goals(inq, mql int64, revenue string) option {
	return func(n *Node) { r := amountOf(revenue); n.INQ, n.MQL, n.Revenue = &inq, &mql, &r }
}

func local(amount, currency string) option {
	return func(n *Node) { n.LocalPlannedCost = &money.Money{Amount: amountOf(amount), Currency: currency} }
}

func strategic() option { return func(n *Node) { n.Strategic = true } }

func teams(ids ...string) option { return func(n *Node) { n.TeamIDs = ids } }

func units(ids ...string) option { return func(n *Node) { n.BusinessUnitIDs = ids } }

func activityType(id string) option { return func(n *Node) { n.ActivityTypeID = id } }

func describe(text string) option { return func(n *Node) { n.Description = text } }

func externalName(text string) option { return func(n *Node) { n.ExternalName = text } }

func swbs(value string) option { return func(n *Node) { n.Swbs = value } }

// ptr returns a pointer to v.
func ptr[T any](v T) *T { return &v }

// field sets the value of a level-specific column.
func field(key string, v Value) option {
	return func(n *Node) {
		if n.Fields == nil {
			n.Fields = map[string]Value{}
		}
		n.Fields[key] = v
	}
}

// choices returns a choice value of the labels; a label ending in "!" is an inactive option.
func choices(labels ...string) Value {
	v := Value{}
	for _, l := range labels {
		inactive := strings.HasSuffix(l, "!")
		v.Choices = append(v.Choices, Choice{Label: strings.TrimSuffix(l, "!"), Inactive: inactive})
	}
	return v
}

// split returns a % split of label, share pairs.
func split(pairs ...string) Value {
	v := Value{}
	for i := 0; i+1 < len(pairs); i += 2 {
		share, err := money.ParsePercent(pairs[i+1])
		if err != nil {
			panic(err)
		}
		inactive := strings.HasSuffix(pairs[i], "!")
		v.Choices = append(v.Choices, Choice{Label: strings.TrimSuffix(pairs[i], "!"), Share: &share, Inactive: inactive})
	}
	return v
}

// amountOf parses an amount of the fixtures.
func amountOf(text string) money.Amount {
	a, err := money.ParseAmount(text)
	if err != nil {
		panic(fmt.Sprintf("fixture: amount %q: %v", text, err))
	}
	return a
}

// fixtureAttributes are the Shell attributes of the fixtures: Cluster on campaigns, an inactive Region, a
// number on activities and a % split of countries.
var fixtureAttributes = []Attribute{
	{ID: "SA-1001", Label: "Business unit", Level: LevelPlan, Active: true},
	{ID: "SA-1003", Label: "Cluster", Level: LevelCampaign, Active: true},
	{ID: "SA-1004", Label: "Region", Level: LevelCampaign},
	{ID: "SA-1020", Label: "Country / OU", Level: LevelActivity, Active: true},
	{ID: "SA-1021", Label: "Expected leads", Level: LevelActivity, Number: true, Active: true},
}

// fixtureChoices are the teams, business units, activity types and people of the fixtures, for personID:
// Priya belongs to Lubricants Europe.
func fixtureChoices(personID string) Choices {
	return Choices{
		Teams: []Option{
			{ID: "T-1001", Label: "Lubricants Europe", Mine: personID == priya.ID},
			{ID: "T-1002", Label: "Mobility China", Mine: personID == mei.ID},
			{ID: "T-1003", Label: "Global Marketing Ops", Inactive: true},
		},
		BusinessUnits: []Option{{ID: "SO-1001", Label: "Mobility"}, {ID: "SO-1002", Label: "Lubricants"}},
		ActivityTypes: []Option{
			{ID: "AT-1001", Label: "Content production"}, {ID: "AT-1002", Label: "Digital media"},
			{ID: "AT-1003", Label: "Agency fees – Creative"}, {ID: "AT-1004", Label: "Sponsorship"},
			{ID: "AT-1005", Label: "Event"}, {ID: "AT-1006", Label: "MOBILITY – Long-term contract", Inactive: true},
		},
		People: []Option{
			{ID: priya.ID, Label: priya.Name}, {ID: daniel.ID, Label: daniel.Name}, {ID: mei.ID, Label: mei.Name},
			{ID: sara.ID, Label: sara.Name, Inactive: true},
		},
		SeesFinance: true, // feature 015: the people of the fixtures may see financial data
	}
}
