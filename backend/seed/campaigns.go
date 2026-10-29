package seed

import (
	"cmp"
	"context"

	"mapp/internal/campaign"
	"mapp/internal/platform/date"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// demoCampaign is one campaign of 005 data model §9 with its initiatives. Dates are YYYY-MM-DD; "" means
// no dates. attributes sets the Shell attribute values; nil means none.
type demoCampaign struct {
	plan, name, owner, start, end, budget string
	attributes                            func(v *values)
	initiatives                           []demoInitiative
	phasing                               []string // 016: a custom phasing as month, amount pairs; nil for not phased
}

type demoInitiative struct {
	name, owner, start, end, budget string
	attributes                      func(v *values)
}

func clusterAndRegion(region string) func(v *values) {
	return func(v *values) {
		v.choices("Cluster", "Lubricants – Europe").choices("Region", region)
	}
}

func notFIT4(v *values) {
	v.yesNo("FIT4 initiative", false)
}

// demoCampaigns are C-1001 … C-1005 with I-1001 … I-1010 in ID order; C-1005 is the GBP plan's of 016, phased in four
// months. Retail growth has no cluster or region on purpose ("2 fields needed", 004); Fleet cards has no dates; Brand
// platform ends after its plan's year (R4) and its initiatives' budgets exceed its own by 50,000.00.
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
	{plan: ukPlan, name: "Helix UK & Ireland", owner: "Priya Raman", start: "2027-01-01", end: "2027-12-31",
		budget: "400000.00", attributes: clusterAndRegion("LUBES – UK & IRELAND"),
		phasing: []string{"2027-03", "100000.00", "2027-06", "100000.00", "2027-09", "100000.00", "2027-11", "100000.00"},
		initiatives: []demoInitiative{
			{name: "UK", owner: "Priya Raman", start: "2027-01-01", end: "2027-12-31", budget: "250000.00", attributes: notFIT4},
			{name: "Ireland", owner: "Priya Raman", start: "2027-01-01", end: "2027-12-31", budget: "150000.00", attributes: notFIT4},
		}},
}

func (d *demo) addCampaigns(ctx context.Context) error {
	d.campaigns, d.initiatives = map[string]string{}, map[string]string{}
	return d.putCampaigns(ctx, demoCampaigns)
}

func (d *demo) putCampaigns(ctx context.Context, list []demoCampaign) error {
	repo := d.stores.Campaigns
	for _, c := range list {
		var r resolver
		planID := r.id(d.plans, "plan", c.plan)
		ownerID := r.id(d.people, "person", c.owner)
		if r.err != nil {
			return r.err
		}
		currency := d.planCurrencies[c.plan]
		start, end, budget, attributes, err := d.planningFields(c.start, c.end, c.budget, currency, settings.LevelCampaign, c.attributes)
		if err != nil {
			return err
		}
		id, err := d.next(ctx, ids.PrefixCampaign)
		if err != nil {
			return err
		}
		phasing, err := customPhasing(c.phasing)
		if err != nil {
			return err
		}
		camp := campaign.Campaign{
			ID: id, PlanID: planID, Name: c.name, OwnerID: &ownerID, StartDate: start, EndDate: end,
			Budget: budget, Attributes: attributes, BudgetPhasing: phasing, Meta: d.meta(),
		}
		if err := repo.PutCampaign(ctx, camp); err != nil {
			return err
		}
		d.campaigns[c.plan+"/"+c.name] = id
		if err := d.putInitiatives(ctx, c, id, currency); err != nil {
			return err
		}
	}
	return nil
}

func (d *demo) putInitiatives(ctx context.Context, c demoCampaign, campaignID, currency string) error {
	repo := d.stores.Campaigns
	for _, i := range c.initiatives {
		initiative, err := d.initiativeOf(i, campaignID, currency)
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
	return nil
}

func (d *demo) initiativeOf(i demoInitiative, campaignID, currency string) (campaign.Initiative, error) {
	ownerID, err := find(d.people, "person", i.owner)
	if err != nil {
		return campaign.Initiative{}, err
	}
	start, end, budget, attributes, err := d.planningFields(i.start, i.end, i.budget, currency, settings.LevelInitiative, i.attributes)
	if err != nil {
		return campaign.Initiative{}, err
	}
	return campaign.Initiative{
		CampaignID: campaignID, Name: i.name, OwnerID: &ownerID, StartDate: start, EndDate: end,
		Budget: budget, Attributes: attributes, BudgetPhasing: rollup.NotPhased, Meta: d.meta(),
	}, nil
}

// planningFields reads "" dates or budget as none and puts amounts in the plan's currency (016).
func (d *demo) planningFields(start, end, budget, currency string, level settings.Level, attributes func(v *values)) (
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
		m.Currency = cmp.Or(currency, money.BaseCurrency)
		amount = &m
	}
	v := d.newValues(level)
	if attributes != nil {
		attributes(v)
	}
	values, err := v.done()
	return startDate, endDate, amount, values, err
}

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

// customPhasing is a custom budget phasing of month, amount pairs (016 FR-020); not phased without pairs.
func customPhasing(pairs []string) (rollup.Phasing, error) {
	if len(pairs) == 0 {
		return rollup.NotPhased, nil
	}
	phasing := rollup.Phasing{Method: rollup.MethodCustom, Months: []rollup.PhasingMonth{}}
	for i := 0; i+1 < len(pairs); i += 2 {
		amount, err := money.ParseAmount(pairs[i+1])
		if err != nil {
			return rollup.Phasing{}, err
		}
		phasing.Months = append(phasing.Months, rollup.PhasingMonth{Month: pairs[i], Amount: amount})
	}
	return phasing, nil
}
