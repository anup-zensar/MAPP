package seed

import (
	"context"
	"fmt"

	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/platform/ids"
	"mapp/internal/settings"
)

// demoWorkflows are WF-1001 and WF-1002 in feature 008's shape (D-103, D-201); each level lists its approvers by name.
var demoWorkflows = []struct {
	name, description string
	levels            [][]string
}{
	{"Lubricants EU approval", "Lubricants Europe activities. The marketing team leader approves.",
		[][]string{{danielOkoro}}},
	{"China BC approval workflow", "Mobility China activities: the business champion first, then the marketing administrator.",
		[][]string{{carlosMendes}, {alexMorgan}}},
}

// demoActivity holds dates as YYYY-MM-DD and amounts as texts, "" for none; an ongoing one has only a start date.
type demoActivity struct {
	plan, campaign, initiative, name string
	status                           activity.Status
	activityType, owner              string
	start, end                       string
	ongoing                          bool
	plannedCost                      string
	workflow                         string
	inq                              int64
	typedMQL                         int64 // with typedRevenue: "Set later stages myself"
	typedRevenue                     string
	attributes                       func(v *values)
}

const (
	lubricantsPlan = "FY2026 Lubricants Europe"
	mobilityPlan   = "FY2026 Mobility China"
	lubricantsFlow = "Lubricants EU approval"
	mobilityFlow   = "China BC approval workflow"
)

// lubricantsActivity fills every attribute needed to submit, with one country and one brand at 100.00 %.
func lubricantsActivity(country, companyCode, costCentre, brand, fit4ID string) func(v *values) {
	return func(v *values) {
		v.choices("Channel / audience", "External").split("Country / OU", country, "100.00").
			choice("Company code", companyCode).choice("Cost centre", costCentre).
			choices("Global campaign", "Helix Ultra global platform").choices("Growth ambition", "LUBRICANTS – Premium growth").
			choice("PIR", "No").choice("FIT4 initiative", "Not FIT4").text("FIT4 ID", fit4ID).
			split("Brands", brand, "100.00").choices("Global key accounts", "Account A – Automotive OEM").
			choices("Sector", "Automotive").choices("Sub-sector", "Passenger cars").choice("Product launch", "No")
	}
}

// demoActivities are A-1001 … A-1012 in ID order, A-1011 and A-1012 the GBP plan's; history.go writes A-1013, which is
// in the recycle bin with its initiative.
var demoActivities = []demoActivity{
	{plan: lubricantsPlan, campaign: "Helix launch", initiative: "Digital & social", name: "Launch film",
		status: activity.StatusCreated, activityType: typeContentProduction, owner: priyaRaman,
		start: "2026-03-01", end: "2026-05-31", plannedCost: "120000.00", workflow: lubricantsFlow, inq: 250_000,
		// Company code, Cost centre and PIR left empty on purpose: "3 fields needed".
		attributes: func(v *values) {
			v.choices("Channel / audience", "External", "Partners").
				split("Country / OU", "Germany", "50.00", "United Kingdom", "30.00", "Netherlands", "20.00").
				choices("Global campaign", "Helix Ultra global platform").choices("Growth ambition", "LUBRICANTS – Premium growth").
				choice("FIT4 initiative", "FIT4 – Growth").text("FIT4 ID", "FIT4-0042").choices("Zero-based cost hierarchy", "LUBES - Growth drivers").
				split("Brands", "Helix", "100.00").choices("Global key accounts", "Account A – Automotive OEM").
				choices("Sector", "Automotive").choices("Sub-sector", "Passenger cars").choice("Product launch", "Yes – relaunch")
		}},
	{plan: lubricantsPlan, campaign: "Helix launch", initiative: "Digital & social", name: "Social always-on",
		status: activity.StatusApproved, activityType: typeDigitalMedia, owner: priyaRaman,
		start: "2026-01-01", end: "2026-06-30", plannedCost: "80000.00", workflow: lubricantsFlow, inq: 40_000,
		attributes: lubricantsActivity("Germany", "DE03", "DE03000150", "Helix", "FIT4-0043")},
	{plan: lubricantsPlan, campaign: "Helix launch", initiative: "Print & outdoor", name: "Billboard winter wave",
		status: activity.StatusApproved, activityType: typeAgencyFeesCreative, owner: priyaRaman,
		start: "2026-01-05", end: "2026-02-28", plannedCost: "45000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("United Kingdom", "GB01", "GB01000105", "Helix", "FIT4-0044")},
	{plan: lubricantsPlan, campaign: "Helix launch", initiative: "Print & outdoor", name: "Trade magazine ads",
		status: activity.StatusSubmitted, activityType: typeContentProduction, owner: priyaRaman,
		start: "2026-04-01", end: "2026-09-30", plannedCost: "30000.00", workflow: lubricantsFlow, inq: 20_000,
		attributes: lubricantsActivity("Netherlands", "NL02", "NL02000340", "Helix", "FIT4-0045")},
	{plan: lubricantsPlan, campaign: "Rimula fleet", initiative: "Retail events", name: "Fleet roadshow",
		status: activity.StatusDeclined, activityType: typeEvent, owner: priyaRaman,
		start: "2026-07-01", end: "2026-09-30", plannedCost: "60000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("Germany", "DE03", "DE03000150", "Rimula", "FIT4-0046")},
	{plan: lubricantsPlan, campaign: "Rimula fleet", initiative: "Retail events", name: "Christmas trade gifting",
		status: activity.StatusApproved, activityType: typeEvent, owner: priyaRaman,
		start: "2026-11-01", end: "2026-12-15", plannedCost: "25000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("United Kingdom", "GB01", "GB01000118", "Rimula", "FIT4-0047")},
	{plan: lubricantsPlan, campaign: "Rimula fleet", initiative: "Events", name: "Partner co-marketing fund",
		status: activity.StatusApproved, activityType: typeSponsorship, owner: priyaRaman,
		start: "2026-03-01", ongoing: true, plannedCost: "100000.00", workflow: lubricantsFlow,
		attributes: func(v *values) {
			lubricantsActivity("Netherlands", "NL02", "NL02000340", "Rimula", "FIT4-0048")(v)
			v.split("Brands", "Rimula", "60.00", "Tellus", "40.00")
		}},
	{plan: mobilityPlan, campaign: "Retail growth", initiative: "Station promotions", name: "Grand opening Shanghai",
		status: activity.StatusCreated, activityType: typeEvent, owner: meiLin,
		start: "2026-04-01", end: "2026-06-30", plannedCost: "40000.00", workflow: mobilityFlow,
		// The shares add up to 90.00 % and the cost centre is empty on purpose.
		attributes: func(v *values) {
			v.choices("Channel / audience", "External").split("Country / OU", "China", "60.00", "India", "30.00").
				choice("Company code", "CN04").choices("Global campaign", "Shell Recharge launch").
				choices("Growth ambition", "MOBILITY – Convenience retail").choice("PIR", "No").
				choice("FIT4 initiative", "Not FIT4").text("FIT4 ID", "FIT4-0108")
		}},
	{plan: mobilityPlan, campaign: "Retail growth", initiative: "Fleet cards", name: "Fleet card launch",
		status: activity.StatusSubmitted, activityType: typeLongTermContract, owner: meiLin,
		start: "2026-05-01", end: "2026-12-31", plannedCost: "150000.00", workflow: mobilityFlow,
		inq: 60_000, typedMQL: 2_000, typedRevenue: "400000.00",
		attributes: func(v *values) {
			v.choices("Channel / audience", "External", "Partners").split("Country / OU", "China", "100.00").
				choice("Company code", "CN04").choice("Cost centre", "CN04000231").choices("Global campaign", "Shell Recharge launch").
				choices("Growth ambition", "MOBILITY – Integrated offer").choice("PIR", "Yes").
				choice("FIT4 initiative", "FIT4 – Growth").text("FIT4 ID", "FIT4-0107")
		}},
	// Deliberately incomplete: owner, dates, planned cost, workflow and attributes empty.
	{plan: mobilityPlan, campaign: "Retail growth", initiative: "Station promotions", name: "Lunar new year promo",
		status: activity.StatusCreated, activityType: typeDigitalMedia},
	// The GBP plan's activities, planned costs in GBP.
	{plan: ukPlan, campaign: "Helix UK & Ireland", initiative: "UK", name: "UK launch event",
		status: activity.StatusCreated, activityType: typeEvent, owner: priyaRaman,
		start: "2027-03-01", end: "2027-05-31", plannedCost: "40000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("United Kingdom", "GB01", "GB01000105", "Helix", "FIT4-0049")},
	{plan: ukPlan, campaign: "Helix UK & Ireland", initiative: "Ireland", name: "Dealer conference",
		status: activity.StatusCreated, activityType: typeEvent, owner: priyaRaman,
		start: "2027-09-01", end: "2027-09-30", plannedCost: "25000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("United Kingdom", "GB01", "GB01000118", "Helix", "FIT4-0050")},
}

func (d *demo) addActivities(ctx context.Context) error {
	if err := d.addWorkflows(ctx); err != nil {
		return err
	}
	d.activities = map[string]string{}
	return d.putActivities(ctx, demoActivities)
}

func (d *demo) putActivities(ctx context.Context, list []demoActivity) error {
	repo := d.stores.Activities
	for _, a := range list {
		var r resolver
		initiativeID := r.id(d.initiatives, "initiative", a.campaign+"/"+a.initiative)
		if _, err := find(d.campaigns, "campaign", a.plan+"/"+a.campaign); err != nil && r.err == nil {
			r.err = err // the initiative must be in that plan's campaign
		}
		if r.err != nil {
			return r.err
		}
		act, err := d.activityOf(a, initiativeID)
		if err != nil {
			return err
		}
		if act.ID, err = d.next(ctx, ids.PrefixActivity); err != nil {
			return err
		}
		if err := repo.Put(ctx, act); err != nil {
			return err
		}
		d.activities[a.plan+"/"+a.name] = act.ID
	}
	return nil
}

func (d *demo) activityOf(a demoActivity, initiativeID string) (activity.Activity, error) {
	var r resolver
	act := activity.Activity{
		InitiativeID: initiativeID, Name: a.name,
		ActivityTypeID: r.id(d.settings.ActivityTypes, "activity type", a.activityType),
		Ongoing:        a.ongoing, Status: a.status, Meta: d.meta(),
	}
	if a.owner != "" {
		act.OwnerID = r.id(d.people, "person", a.owner)
	}
	if a.workflow != "" {
		act.ApprovalWorkflowID = r.id(d.workflows, "approval workflow", a.workflow)
	}
	if r.err != nil {
		return activity.Activity{}, r.err
	}
	start, end, cost, attributes, err := d.planningFields(a.start, a.end, a.plannedCost, d.planCurrencies[a.plan], settings.LevelActivity, a.attributes)
	if err != nil {
		return activity.Activity{}, fmt.Errorf("seed: activity %s: %w", a.name, err)
	}
	act.StartDate, act.EndDate, act.PlannedCost, act.Attributes = start, end, cost, attributes
	if a.inq > 0 {
		inq := a.inq
		act.INQ = &inq
	}
	if a.typedRevenue != "" {
		revenue, err := usd(a.typedRevenue)
		if err != nil {
			return activity.Activity{}, err
		}
		mql := a.typedMQL
		act.SetLaterStagesMyself, act.MQL, act.Revenue = true, &mql, &revenue
	}
	return act, nil
}

func (d *demo) addWorkflows(ctx context.Context) error {
	repo := d.stores.Approvals
	d.workflows = map[string]string{}
	for _, w := range demoWorkflows {
		var r resolver
		levels := make([]approval.Level, 0, len(w.levels))
		for _, approvers := range w.levels {
			level := approval.Level{ApproverIDs: []string{}}
			for _, name := range approvers {
				level.ApproverIDs = append(level.ApproverIDs, r.id(d.people, "person", name))
			}
			levels = append(levels, level)
		}
		if r.err != nil {
			return r.err
		}
		id, err := d.next(ctx, ids.PrefixApprovalWorkflow)
		if err != nil {
			return err
		}
		workflow := approval.Workflow{ID: id, Name: w.name, Description: w.description, Active: true, Levels: levels, Meta: d.meta()}
		if err := repo.PutWorkflow(ctx, workflow); err != nil {
			return err
		}
		d.workflows[w.name] = id
	}
	return nil
}
