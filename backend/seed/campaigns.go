package seed

import (
	"context"

	"northstar/internal/campaigns"
	"northstar/internal/platform/date"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// demoCampaign is one campaign of 005 data model §9 with its initiatives. Dates are YYYY-MM-DD; "" means
// no dates. attributes sets the Shell attribute values; nil means none.
type demoCampaign struct {
	plan, name, owner, start, end, budget string
	attributes                            func(v *values)
	initiatives                           []demoInitiative
}

// demoInitiative is one initiative of 005 data model §9.
type demoInitiative struct {
	name, owner, start, end, budget string
	attributes                      func(v *values)
}

// clusterAndRegion sets the Lubricants – Europe cluster and one of its regions.
func clusterAndRegion(region string) func(v *values) {
	return func(v *values) {
		v.choices("Cluster", "Lubricants – Europe").choices("Region", region)
	}
}

// notFIT4 sets the initiative's FIT4 initiative to No.
func notFIT4(v *values) {
	v.yesNo("FIT4 initiative", false)
}

// demoCampaigns are C-1001 … C-1004 with I-1001 … I-1008, in ID order. Retail growth has no cluster and no
// region on purpose ("2 fields needed", 004's publish example); Fleet cards has no dates; Brand platform
// ends after its plan's year (R4) and its initiatives' budgets exceed its own (over-allocated by 50,000.00).
var demoCampaigns = []demoCampaign{
	{plan: "FY2026 Lubricants Europe", name: "Helix launch", owner: "Priya Raman", start: "2026-01-01", end: "2026-12-31",
		budget: "700000.00", attributes: func(v *values) {
			clusterAndRegion("LUBES – DACH")(v)
			v.yesNo("Insight session completed", true).yesNo("Workshop completed", true).yesNo("Global GM sign-off completed", false)
		}, initiatives: []demoInitiative{
			{name: "Digital & social", owner: "Priya Raman", start: "2026-01-01", end: "2026-06-30", budget: "300000.00",
				attributes: func(v *values) {
					v.choices("Growth ambition", "LUBRICANTS – Premium growth").choice("Function", "Marketing").choice("Focus area", "Brand")
					notFIT4(v)
				}},
			{name: "Print & outdoor", owner: "Priya Raman", start: "2026-01-01", end: "2026-12-31", budget: "250000.00", attributes: notFIT4},
		}},
	{plan: "FY2026 Lubricants Europe", name: "Rimula fleet", owner: "Daniel Okoro", start: "2026-02-01", end: "2026-12-31",
		budget: "450000.00", attributes: clusterAndRegion("LUBES – BENELUX"), initiatives: []demoInitiative{
			{name: "Events", owner: "Daniel Okoro", start: "2026-03-01", end: "2026-09-30", budget: "200000.00", attributes: notFIT4},
			{name: "Retail events", owner: "Daniel Okoro", start: "2026-06-01", end: "2026-12-31", budget: "150000.00", attributes: notFIT4},
		}},
	{plan: "FY2026 Mobility China", name: "Retail growth", owner: "Mei Lin", start: "2026-01-01", end: "2026-12-31",
		budget: "600000.00", initiatives: []demoInitiative{
			{name: "Station promotions", owner: "Mei Lin", start: "2026-03-01", end: "2026-09-30", budget: "350000.00", attributes: notFIT4},
			{name: "Fleet cards", owner: "Mei Lin", budget: "200000.00", attributes: notFIT4},
		}},
	{plan: "FY2027 Global Brand", name: "Brand platform", owner: "Alex Morgan", start: "2027-01-01", end: "2028-03-31",
		budget: "300000.00", attributes: clusterAndRegion("LUBES – UK & IRELAND"), initiatives: []demoInitiative{
			{name: "Brand film", owner: "Alex Morgan", start: "2027-01-01", end: "2027-06-30", budget: "250000.00", attributes: notFIT4},
			{name: "Sponsorships", owner: "Alex Morgan", start: "2027-04-01", end: "2027-12-31", budget: "100000.00", attributes: notFIT4},
		}},
}

// addCampaigns creates the four demo campaigns and their eight initiatives through the campaigns
// repository.
func (d *demo) addCampaigns(ctx context.Context) error {
	repo := campaigns.NewJSONRepository()
	d.campaigns, d.initiatives = map[string]string{}, map[string]string{}
	for _, c := range demoCampaigns {
		var r resolver
		planID := r.id(d.plans, "plan", c.plan)
		ownerID := r.id(d.people, "person", c.owner)
		if r.err != nil {
			return r.err
		}
		start, end, budget, attributes, err := d.planningFields(c.start, c.end, c.budget, settings.LevelCampaign, c.attributes)
		if err != nil {
			return err
		}
		id, err := d.next(ctx, ids.PrefixCampaign)
		if err != nil {
			return err
		}
		campaign := campaigns.Campaign{
			ID: id, PlanID: planID, Name: c.name, OwnerID: &ownerID, StartDate: start, EndDate: end,
			Budget: budget, Attributes: attributes, Meta: d.meta(),
		}
		if err := repo.PutCampaign(ctx, campaign); err != nil {
			return err
		}
		d.campaigns[c.plan+"/"+c.name] = id

		for _, i := range c.initiatives {
			initiative, err := d.initiativeOf(i, id)
			if err != nil {
				return err
			}
			if initiative.ID, err = d.next(ctx, ids.PrefixInitiative); err != nil {
				return err
			}
			if err := repo.PutInitiative(ctx, initiative); err != nil {
				return err
			}
			d.initiatives[c.name+"/"+i.name] = initiative.ID
		}
	}
	return nil
}

// initiativeOf builds the stored initiative i of the campaign campaignID, without its ID, looking up its
// owner by name. NorthStar creates it at the seed time.
func (d *demo) initiativeOf(i demoInitiative, campaignID string) (campaigns.Initiative, error) {
	ownerID, err := find(d.people, "person", i.owner)
	if err != nil {
		return campaigns.Initiative{}, err
	}
	start, end, budget, attributes, err := d.planningFields(i.start, i.end, i.budget, settings.LevelInitiative, i.attributes)
	if err != nil {
		return campaigns.Initiative{}, err
	}
	return campaigns.Initiative{
		CampaignID: campaignID, Name: i.name, OwnerID: &ownerID, StartDate: start, EndDate: end,
		Budget: budget, Attributes: attributes, Meta: d.meta(),
	}, nil
}

// planningFields turns the texts of a seeded campaign or initiative into its stored dates, USD budget and
// Shell attribute values ("" dates or budget: none).
func (d *demo) planningFields(start, end, budget string, level settings.Level, attributes func(v *values)) (
	*date.Date, *date.Date, *money.Money, settings.AttributeValues, error) {
	startDate, endDate, err := dates(start, end)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var amount *money.Money
	if budget != "" {
		m, err := usd(budget)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		amount = &m
	}
	v := d.newValues(level)
	if attributes != nil {
		attributes(v)
	}
	values, err := v.done()
	return startDate, endDate, amount, values, err
}

// dates parses a start and an end date; "" is no date.
func dates(start, end string) (*date.Date, *date.Date, error) {
	var parsed [2]*date.Date
	for i, text := range []string{start, end} {
		if text == "" {
			continue
		}
		d, err := date.Parse(text)
		if err != nil {
			return nil, nil, err
		}
		parsed[i] = &d
	}
	return parsed[0], parsed[1], nil
}
