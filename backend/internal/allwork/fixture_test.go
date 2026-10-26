package allwork

import (
	"context"
	"fmt"
	"slices"
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

// seedYear is 2026 of the demo data on 18 Oct 2026: two plans, EUR and GBP line items, a balance of
// 20,000.00, one of 0.00, one of −500.00, an activity without dates and an ongoing one.
//
//	P-1001 FY2026 Lubricants Europe (Published, 1,200,000.00, Priya, T-1001, Lubricants)
//	  C-1001 Helix launch (700,000.00, Priya)
//	    I-1001 Digital & social (300,000.00)
//	      A-1001 Launch film (Created, Mar–May, 120,000.00; L-1001 50,000.00 USD, L-1002 46,000.00 EUR = 50,000.00)
//	      A-1002 Social always-on (Complete, Jan–Jun, 80,000.00; L-1003 15,800.00 GBP = 20,000.00, L-1004 5,000.00)
//	    I-1002 Print & outdoor (250,000.00)
//	      A-1003 Billboard winter wave (Complete, Jan–Feb, 45,000.00; L-1005 9,200.00 EUR = 10,000.00)
//	      A-1004 Trade magazine ads (Submitted, Apr–Sep, 30,000.00; L-1006 30,000.00 custom, deposit in Mar)
//	  C-1002 Rimula fleet (450,000.00, Daniel)
//	    I-1003 Events (200,000.00, Daniel)
//	      A-1005 Partner co-marketing fund (In progress, ongoing from Mar, 100,000.00)
//	    I-1004 Retail events (150,000.00, Daniel)
//	      A-1006 Fleet roadshow (Declined, Jul–Sep, 60,000.00)
//	      A-1007 Christmas trade gifting (Approved, Nov–Dec, 25,000.00, owner Daniel)
//	P-1002 FY2026 Mobility China (Draft, 800,000.00, Mei, T-1002, no business unit)
//	  C-1003 Retail growth (600,000.00, Mei)
//	    I-1005 Station promotions (350,000.00)
//	      A-1008 Grand opening Shanghai (Created, Apr–Jun, 40,000.00; L-1007 1,000.00, L-1008 39,500.00 → −500.00)
//	      A-1009 Lunar new year promo (Created, no dates, no planned cost, no owner)
//	    I-1006 Fleet cards (200,000.00, no dates)
//	      A-1010 Fleet card launch (Submitted, May–Dec, 150,000.00)
//	    I-1007 Loyalty (no activities)
func seedYear() *yearBuilder {
	b := newYear()
	lub := b.plan("P-1001", "FY2026 Lubricants Europe", status("published"), budget("1200000.00"), owner(priya),
		teams("T-1001"), units("SO-1002"), describe("Lubricants marketing in Europe"),
		field("plan.model", Value{Text: "Lubricants 1.0"}), field("attr.SA-1001", choices("Lubricants")))
	helix := b.campaign(lub, "C-1001", "Helix launch", owner(priya), dates("2026-01-01", "2026-12-31"),
		budget("700000.00"), progress("inProgress", 2, 4), timing("running"),
		field("attr.SA-1003", choices("Lubricants – Europe")), field("attr.SA-1004", choices("LUBES – DACH")))
	digital := b.initiative(helix, "I-1001", "Digital & social", owner(priya), dates("2026-01-01", "2026-06-30"),
		budget("300000.00"), progress("inProgress", 1, 2), timing("finished"))
	film := b.activity(digital, "A-1001", "Launch film", owner(priya), dates("2026-03-01", "2026-05-31"),
		planned("120000.00"), goals(250000, 5000, "2500000.00"), activityType("AT-1001"), externalName("Helix Ultra film"),
		describe("A hero film for the Helix Ultra relaunch"), field("activity.type", Value{Text: "Content production"}),
		field("attr.SA-1020", split("Germany", "60.00", "France", "40.00")), field("attr.SA-1021", Value{Number: ptr(int64(1200))}))
	b.lineItem(film, "L-1001", "Agency fee", "50000.00", owner(priya))
	b.lineItem(film, "L-1002", "Film production", "50000.00", owner(priya), local("46000.00", "EUR"),
		field("lineItem.localPlannedCost", Value{Money: &money.Money{Amount: amountOf("46000.00"), Currency: "EUR"}}),
		field("lineItem.currency", Value{Text: "EUR"}))
	social := b.activity(digital, "A-1002", "Social always-on", owner(priya), dates("2026-01-01", "2026-06-30"),
		planned("80000.00"), status("complete"), goals(40000, 800, "80000.00"), activityType("AT-1002"))
	b.lineItem(social, "L-1003", "Venue hire", "20000.00", owner(priya), local("15800.00", "GBP"))
	b.lineItem(social, "L-1004", "Speaker fees", "5000.00", owner(priya))
	printing := b.initiative(helix, "I-1002", "Print & outdoor", owner(priya), dates("2026-01-01", "2026-12-31"),
		budget("250000.00"), progress("inProgress", 1, 2), timing("running"))
	billboard := b.activity(printing, "A-1003", "Billboard winter wave", owner(priya), dates("2026-01-05", "2026-02-28"),
		planned("45000.00"), status("complete"), activityType("AT-1003"))
	b.lineItem(billboard, "L-1005", "Stand build", "10000.00", owner(priya), local("9200.00", "EUR"), swbs("readyToSend"))
	ads := b.activity(printing, "A-1004", "Trade magazine ads", owner(priya), dates("2026-04-01", "2026-09-30"),
		planned("30000.00"), status("submitted"), goals(20000, 400, "40000.00"), activityType("AT-1001"))
	b.lineItem(ads, "L-1006", "Media booking", "30000.00", owner(priya))
	b.months("L-1006", "2026-03", "3000.00", "2026-04", "4500.00", "2026-05", "4500.00", "2026-06", "4500.00",
		"2026-07", "4500.00", "2026-08", "4500.00", "2026-09", "4500.00")
	rimula := b.campaign(lub, "C-1002", "Rimula fleet", owner(daniel), dates("2026-02-01", "2026-12-31"),
		budget("450000.00"), progress("inProgress", 2, 3), timing("running"),
		field("attr.SA-1003", choices("Lubricants – Europe")), field("attr.SA-1004", choices("LUBES – BENELUX!")))
	events := b.initiative(rimula, "I-1003", "Events", owner(daniel), dates("2026-03-01", "2026-09-30"),
		budget("200000.00"), progress("allApproved", 1, 1), timing("finished"))
	b.activity(events, "A-1005", "Partner co-marketing fund", owner(priya), ongoingFrom("2026-03-01"),
		planned("100000.00"), status("inProgress"), activityType("AT-1004"))
	retail := b.initiative(rimula, "I-1004", "Retail events", owner(daniel), dates("2026-06-01", "2026-12-31"),
		budget("150000.00"), progress("inProgress", 1, 2), timing("running"))
	b.activity(retail, "A-1006", "Fleet roadshow", owner(priya), dates("2026-07-01", "2026-09-30"),
		planned("60000.00"), status("declined"), activityType("AT-1005"))
	b.activity(retail, "A-1007", "Christmas trade gifting", owner(daniel), dates("2026-11-01", "2026-12-15"),
		planned("25000.00"), status("approved"), activityType("AT-1005"))

	mob := b.plan("P-1002", "FY2026 Mobility China", budget("800000.00"), owner(mei), teams("T-1002"), missing(1))
	growth := b.campaign(mob, "C-1003", "Retail growth", owner(mei), dates("2026-01-01", "2026-12-31"),
		budget("600000.00"), progress("inProgress", 0, 3), timing("running"), missing(2))
	stations := b.initiative(growth, "I-1005", "Station promotions", owner(mei), dates("2026-03-01", "2026-09-30"),
		budget("350000.00"), progress("inProgress", 0, 2), timing("finished"))
	opening := b.activity(stations, "A-1008", "Grand opening Shanghai", owner(mei), dates("2026-04-01", "2026-06-30"),
		planned("40000.00"), activityType("AT-1005"), missing(1), field("attr.SA-1020", split("China", "70.00", "Macau!", "30.00")))
	b.lineItem(opening, "L-1007", "Station signage", "1000.00", owner(mei), missing(1))
	b.months("L-1007", "2026-04", "900.00")
	b.lineItem(opening, "L-1008", "Fleet card promotion", "39500.00", owner(mei), local("280450.00", "CNY"))
	b.activity(stations, "A-1009", "Lunar new year promo", activityType("AT-1002"), missing(5))
	cards := b.initiative(growth, "I-1006", "Fleet cards", owner(mei), budget("200000.00"), progress("inProgress", 0, 1),
		missing(1))
	b.activity(cards, "A-1010", "Fleet card launch", owner(mei), dates("2026-05-01", "2026-12-31"),
		planned("150000.00"), status("submitted"), activityType("AT-1006"),
		field("activity.type", Value{Text: "MOBILITY – Long-term contract", Inactive: true}))
	b.initiative(growth, "I-1007", "Loyalty", owner(sara), dates("2026-07-01", "2026-12-31"), timing("running"))
	return b
}

// fakeSource is an in-memory Source: its years, a display currency per person, the records of the recycle
// bin, and the number of times Year was read.
type fakeSource struct {
	years      map[int]YearData
	currencies map[string]Currency
	inBin      map[string]bool
	reads      int
}

// newFakeSource returns a Source with the year 2026 of b.
func newFakeSource(b *yearBuilder) *fakeSource {
	return &fakeSource{years: map[int]YearData{2026: b.data()}, currencies: map[string]Currency{}, inBin: map[string]bool{}}
}

func (f *fakeSource) Years(context.Context) ([]int, error) {
	years := []int{}
	for y := range f.years {
		years = append(years, y)
	}
	return years, nil
}

func (f *fakeSource) Choices(_ context.Context, personID string) (Choices, error) {
	return fixtureChoices(personID), nil
}

func (f *fakeSource) Attributes(context.Context) ([]Attribute, error) {
	return fixtureAttributes, nil
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

func (f *fakeSource) Year(_ context.Context, year int, _ date.Date, _ []string) (YearData, error) {
	f.reads++
	data, found := f.years[year]
	if !found {
		return YearData{Nodes: []Node{}}, nil
	}
	return data, nil
}

func (f *fakeSource) PlanYear(ctx context.Context, planID string, year int, today date.Date) (YearData, error) {
	data, err := f.Year(ctx, year, today, nil)
	data.Nodes = slices.DeleteFunc(slices.Clone(data.Nodes), func(n Node) bool { return n.PlanID != planID })
	return data, err
}

// RatesUsed answers the rates of the fixtures (016 FR-009).
func (f *fakeSource) RatesUsed(context.Context) ([]string, error) {
	return []string{"1 USD = 0.92 EUR · rate of 18 Oct 2026 · GSAP import", "1 USD = 0.79 GBP · rate of 18 Oct 2026 · Entered by hand"}, nil
}

func (f *fakeSource) Currency(_ context.Context, personID string) (Currency, error) {
	if c, found := f.currencies[personID]; found {
		return c, nil
	}
	return usd, nil
}

func (f *fakeSource) Locate(_ context.Context, recordID string) (Location, bool, error) {
	for _, data := range f.years {
		for _, n := range data.Nodes {
			if n.ID == recordID {
				l := Location{Level: n.Level, Name: n.Name, PlanID: n.PlanID}
				if n.Level == LevelLineItem {
					l.ActivityID = n.ParentID
				}
				return l, true, nil
			}
		}
	}
	return Location{}, false, nil
}

func (f *fakeSource) InRecycleBin(_ context.Context, recordID string) (bool, error) {
	return f.inBin[recordID], nil
}

// The compiler checks here that fakeSource implements Source.
var _ Source = (*fakeSource)(nil)

// view works out the view of q over the year of b on 18 Oct 2026, in US dollars unless cur says otherwise.
func view(b *yearBuilder, q Query, cur ...Currency) Result {
	return viewOf(b.data(), q, cur...).result
}

// viewOf is view over prepared data, with the tree.
func viewOf(data YearData, q Query, cur ...Currency) built {
	c := usd
	if len(cur) > 0 {
		c = cur[0]
	}
	if q.View == "" {
		q.View = ViewGrid
	}
	if q.Strategic == "" {
		q.Strategic = StrategicInclude
	}
	if q.ShowLevels == "" {
		q.ShowLevels = ShowEverything
	}
	if q.Period == "" {
		q.Period = PeriodMonth
	}
	return build(input{
		query: q, year: 2026, today: fixtureToday, now: fixtureNow, currency: c, data: data,
		choices: fixtureChoices(priya.ID), catalogue: catalogue(fixtureAttributes), attributes: fixtureAttributes,
	})
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

// matchEverything marks every record a shown match, as the view without filters does.
func matchEverything(t *tree) {
	for _, n := range t.byID {
		n.match, n.shown = true, true
	}
}

// rowKeys returns the keys of rows, in order.
func rowKeys(rows []Row) []string {
	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.Key)
	}
	return keys
}

// rowOf returns the row with key, and fails the test when there is none.
func rowOf(rows []Row, key string) Row {
	for _, r := range rows {
		if r.Key == key {
			return r
		}
	}
	panic("no row " + key)
}

// hasRow reports whether rows has the key.
func hasRow(rows []Row, key string) bool {
	for _, r := range rows {
		if r.Key == key {
			return true
		}
	}
	return false
}

// cellText writes a cell as a short text, for comparisons: "—", "50000.00 USD", "Draft/positive", nil as "".
func cellText(c *Cell) string {
	if c == nil {
		return ""
	}
	var text string
	switch c.Kind {
	case KindEmpty:
		text = "—"
	case KindMoney:
		text = c.Money.Amount.String() + " " + c.Money.Currency
	case KindNumber:
		text = fmt.Sprint(*c.Number)
	case KindPercent:
		text = c.Percent + " %"
	case KindDate:
		text = c.Date.String()
	case KindDateTime:
		text = c.DateTime.Format(time.RFC3339)
	case KindStatus:
		text = c.Text + "/" + string(c.Tone)
	default:
		text = c.Text
	}
	if c.Inactive {
		text += " (inactive)"
	}
	if c.Unfiltered != nil {
		text += " [all: " + cellText(c.Unfiltered) + "]"
	}
	return text
}

// money writes an optional amount as "1234.50 USD", "—" for none.
func moneyText(m *money.Money) string {
	if m == nil {
		return "—"
	}
	return m.Amount.String() + " " + m.Currency
}

// columnIndex returns the index of the column key in result's columns.
func columnIndex(r Result, key string) int {
	for i, c := range r.Columns {
		if c.Key == key {
			return i
		}
	}
	panic("no column " + key)
}

// cell returns the cell of column key in the row with rowKey.
func cell(r Result, rowKey, key string) string {
	return cellText(rowOf(r.Rows, rowKey).Cells[columnIndex(r, key)])
}
