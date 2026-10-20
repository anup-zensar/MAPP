package seed_test

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"

	"northstar/internal/activities"
	"northstar/internal/campaigns"
	"northstar/internal/plans"
	"northstar/internal/platform/date"
	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// The demo activities of 006 plan "Demo data", A-1001 … A-1010, in their initiatives and statuses, with
// their types, owners, dates, planned costs and workflows. (A-1011, which goes to the recycle bin with its
// initiative, comes with feature 011's seed step.)
func TestLoad_Activities_FR003(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, a := range d.activities {
		cost := "no cost"
		if a.PlannedCost != nil {
			cost = a.PlannedCost.Format()
		}
		when := dates(a.StartDate, a.EndDate)
		if a.Ongoing {
			when = "ongoing from " + date.Format(*a.StartDate)
		}
		got = append(got, fmt.Sprintf("%s %s %s %s %s %q %s %s %q", a.ID, a.Name, a.InitiativeID, a.Status,
			a.ActivityTypeID, a.OwnerID, when, cost, a.ApprovalWorkflowID))
	}
	want := []string{
		`A-1001 Launch film I-1001 created AT-1004 "U-1002" 01 Mar – 31 May 2026 120,000.00 USD "WF-1001"`,
		`A-1002 Social always-on I-1001 approved AT-1003 "U-1002" 01 Jan – 30 Jun 2026 80,000.00 USD "WF-1001"`,
		`A-1003 Billboard winter wave I-1002 approved AT-1001 "U-1002" 05 Jan – 28 Feb 2026 45,000.00 USD "WF-1001"`,
		`A-1004 Trade magazine ads I-1002 submitted AT-1004 "U-1002" 01 Apr – 30 Sep 2026 30,000.00 USD "WF-1001"`,
		`A-1005 Fleet roadshow I-1004 declined AT-1002 "U-1002" 01 Jul – 30 Sep 2026 60,000.00 USD "WF-1001"`,
		`A-1006 Christmas trade gifting I-1004 approved AT-1002 "U-1002" 01 Nov – 15 Dec 2026 25,000.00 USD "WF-1001"`,
		`A-1007 Partner co-marketing fund I-1003 approved AT-1005 "U-1002" ongoing from 01 Mar 2026 100,000.00 USD "WF-1001"`,
		`A-1008 Grand opening Shanghai I-1005 created AT-1002 "U-1004" 01 Apr – 30 Jun 2026 40,000.00 USD "WF-1002"`,
		`A-1009 Fleet card launch I-1006 submitted AT-1006 "U-1004" 01 May – 31 Dec 2026 150,000.00 USD "WF-1002"`,
		`A-1010 Lunar new year promo I-1005 created AT-1003 "" no dates no cost ""`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("activities =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	launch, fleetCard := d.activities[0], d.activities[8]
	if launch.INQ == nil || *launch.INQ != 250_000 || launch.SetLaterStagesMyself || launch.MQL != nil || launch.Revenue != nil {
		t.Errorf("Launch film goals: inq %v, typed %v", launch.INQ, launch.SetLaterStagesMyself)
	}
	if !fleetCard.SetLaterStagesMyself || *fleetCard.MQL != 2_000 || fleetCard.Revenue.Format() != "400,000.00 USD" {
		t.Errorf("Fleet card launch goals: typed %v, mql %v, revenue %v", fleetCard.SetLaterStagesMyself, fleetCard.MQL, fleetCard.Revenue)
	}

	today := date.Today(loadedAt)
	billboard, gifting, fund := d.activities[2], d.activities[5], d.activities[6]
	if !billboard.EndDate.Before(today) || !gifting.StartDate.After(today) || fund.StartDate.After(today) || fund.EndDate != nil {
		t.Error("the dates no longer show Complete, Approved (not started) and In progress on the test date")
	}
}

// The approval workflows 006 seeds in 008's shape: WF-1001 with Daniel Okoro, WF-1002 with Carlos Mendes,
// then Alex Morgan (D-103, D-201; 008 data model §8); 017's approvals step adds WF-1003 (017 FR-037).
func TestLoad_ApprovalWorkflows_D103(t *testing.T) {
	d := loadDemo(t)
	var got []string
	for _, w := range d.workflows {
		var levels []string
		for _, l := range w.Levels {
			levels = append(levels, l.Name+strings.Join(l.ApproverIDs, ","))
		}
		got = append(got, fmt.Sprintf("%s %s %v %s", w.ID, w.Name, w.Active, strings.Join(levels, " → ")))
		if w.Description == "" {
			t.Errorf("workflow %s has no description", w.ID)
		}
	}
	want := []string{"WF-1001 Lubricants EU approval true U-1003", "WF-1002 China BC approval workflow true U-1005 → U-1001",
		"WF-1003 Standard marketing approval true Line manager → Marketing administratorU-1001 → Mobility second checkU-1003,U-1001"}
	if !slices.Equal(got, want) {
		t.Errorf("workflows = %q, want %q", got, want)
	}
}

// Each activity's type is allowed by its plan's model (R7), and its dates lie inside its initiative's dates
// when the initiative has both — for an ongoing activity, its start (R3).
func TestLoad_ActivitiesFitTheirPlan_R3_R7(t *testing.T) {
	d := loadDemo(t)
	for _, a := range d.activities {
		initiative, campaign, plan := d.pathOf(t, a)
		model := settingsModel(t, d, plan.ModelID)
		if !model.Allows(a.ActivityTypeID) {
			t.Errorf("%s: type %s is not allowed by %s", a.Name, a.ActivityTypeID, model.Name)
		}
		if initiative.StartDate == nil || a.StartDate == nil {
			continue
		}
		outside := a.StartDate.Before(*initiative.StartDate) || a.StartDate.After(*initiative.EndDate)
		if !a.Ongoing {
			outside = outside || a.EndDate.After(*initiative.EndDate)
		}
		if outside {
			t.Errorf("%s is outside %s › %s (%s)", a.Name, campaign.Name, initiative.Name, dates(initiative.StartDate, initiative.EndDate))
		}
	}
}

// The ready-to-submit gaps of the demo story: Launch film misses Company code, Cost centre and PIR ("3
// fields needed"); Grand opening Shanghai misses its Cost centre and splits only 90.00 % over countries;
// Lunar new year promo misses every needed attribute; every other activity has every shown needed-to-submit
// attribute (the Lubricants group on the Lubricants plan), splits at 100.00 % (006 plan "Demo data").
func TestLoad_ActivityAttributes_FR014(t *testing.T) {
	d := loadDemo(t)
	want := map[string][]string{
		"Launch film":            {"Company code", "Cost centre", "PIR"},
		"Grand opening Shanghai": {"Country / OU 90.00 %", "Cost centre"},
		"Lunar new year promo": {"Channel / audience", "Country / OU", "Company code", "Cost centre", "Global campaign",
			"Growth ambition", "PIR", "FIT4 initiative", "FIT4 ID"},
	}
	for _, a := range d.activities {
		_, _, plan := d.pathOf(t, a)
		var businessUnit []string
		if value := plan.Attributes[d.attributeID(t, settings.LevelPlan, "Business unit")]; value != nil {
			_ = json.Unmarshal(value, &businessUnit)
		}
		var missing []string
		for _, attribute := range d.attributes {
			if attribute.Level != settings.LevelActivity || attribute.Marking != settings.MarkingNeededToSubmit {
				continue
			}
			shown := len(attribute.Condition) == 0 ||
				slices.ContainsFunc(attribute.Condition, func(o string) bool { return slices.Contains(businessUnit, o) })
			value, filled := a.Attributes[attribute.ID]
			switch {
			case !shown:
			case !filled:
				missing = append(missing, attribute.Label)
			case attribute.Type == settings.TypeMultipleChoiceSplit:
				var shares []settings.SplitShare
				if err := json.Unmarshal(value, &shares); err != nil {
					t.Fatalf("%s: %s: %v", a.Name, attribute.Label, err)
				}
				var parts []money.Percent
				for _, s := range shares {
					parts = append(parts, s.Share)
				}
				if total := money.SumPercent(parts...); total.Cmp(money.Hundred) != 0 {
					missing = append(missing, attribute.Label+" "+total.Format())
				}
			}
		}
		if !slices.Equal(missing, want[a.Name]) {
			t.Errorf("%s misses %q, want %q", a.Name, missing, want[a.Name])
		}
	}
}

// pathOf returns the initiative, campaign and plan of an activity.
func (d demoData) pathOf(t *testing.T, a activities.Activity) (campaigns.Initiative, campaigns.Campaign, plans.Plan) {
	t.Helper()
	i := slices.IndexFunc(d.initiatives, func(i campaigns.Initiative) bool { return i.ID == a.InitiativeID })
	if i < 0 {
		t.Fatalf("%s: no initiative %s", a.Name, a.InitiativeID)
	}
	initiative := d.initiatives[i]
	c := slices.IndexFunc(d.campaigns, func(c campaigns.Campaign) bool { return c.ID == initiative.CampaignID })
	if c < 0 {
		t.Fatalf("%s: no campaign %s", a.Name, initiative.CampaignID)
	}
	campaign := d.campaigns[c]
	p := slices.IndexFunc(d.plans, func(p plans.Plan) bool { return p.ID == campaign.PlanID })
	if p < 0 {
		t.Fatalf("%s: no plan %s", a.Name, campaign.PlanID)
	}
	return initiative, campaign, d.plans[p]
}

// settingsModel returns the stored model with id.
func settingsModel(t *testing.T, d demoData, id string) settings.Model {
	t.Helper()
	i := slices.IndexFunc(d.models, func(m settings.Model) bool { return m.ID == id })
	if i < 0 {
		t.Fatalf("no model %s", id)
	}
	return d.models[i]
}
