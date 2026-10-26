package apibench

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/lineitems"
	"mapp/internal/people"
)

// newLineItem is the line item given to a template activity that has none.
func newLineItem() map[string]any {
	return map[string]any{fieldName: "Volume media", fieldCurrency: usd, "plannedCost": "1000.00",
		"spread": map[string]any{"method": "evenly"}, "attributes": map[string]any{}}
}

const (
	demoLubricantsPlan = "P-1001"
	demoPlanYear       = 2026
	globalTeam         = "Global Marketing Ops"
)

// lubricantsPlans grows a copy of the demo Lubricants plan into the template — 2 campaigns of 5 initiatives
// of 10 activities with 4 line items each — and copies it, through the API as a planner would.
func (v *volume) lubricantsPlans() {
	template := v.copyPlan(demoLubricantsPlan, TemplatePlan, demoPlanYear)
	v.growTemplate(template)
	for n := 2; n <= lubricantsPlans; n++ {
		v.copyPlan(template, fmt.Sprintf("Volume Lubricants %03d", n), demoPlanYear)
	}
	v.progress("%d Lubricants plans", lubricantsPlans)
	big := v.copyPlan(template, BigPlan, demoPlanYear)
	for _, c := range v.campaignsOf(big) {
		for n := 2; n <= bigPlanCampaigns; n++ {
			body := map[string]any{fieldPlanID: big, fieldName: fmt.Sprintf("%s %d", c.Name, n)}
			v.alex.Do(v.tb, http.MethodPost, "/api/v1/campaigns/"+c.ID+"/copy", body)
		}
	}
	v.progress("the plan of %d activities", len(v.activitiesOf(big)))
}

func (v *volume) growTemplate(planID string) {
	for _, i := range v.initiativesOf(planID) {
		have, err := v.set.Activities.ListByInitiative(v.ctx, i.ID)
		v.must(err)
		model := v.fullest(have)
		for range activitiesPerTarget - len(have) {
			v.alex.Do(v.tb, http.MethodPost, "/api/v1/activities/"+model.ID+"/copy", map[string]any{"initiativeId": i.ID})
		}
	}
	for _, a := range v.activitiesOf(planID) {
		items, err := v.set.LineItems.ListByActivity(v.ctx, a.ID)
		v.must(err)
		first := ""
		if len(items) > 0 {
			first = items[0].ID
		} else {
			v.alex.Do(v.tb, http.MethodPost, "/api/v1/activities/"+a.ID+"/line-items", newLineItem()).Decode(v.tb, &struct {
				ID *string `json:"id"`
			}{ID: &first})
		}
		for n := max(len(items), 1); n < lineItemsPerTarget; n++ {
			v.alex.Do(v.tb, http.MethodPost, "/api/v1/line-items/"+first+"/duplicate", map[string]any{"restartApproval": false})
		}
	}
	for _, c := range v.campaignsOf(planID) {
		have, err := v.set.Campaigns.InitiativesOfCampaign(v.ctx, c.ID)
		v.must(err)
		for n := len(have) + 1; n <= initiativesPerGroup; n++ {
			body := map[string]any{"campaignId": c.ID, fieldName: fmt.Sprintf("%s %d", have[0].Name, n)}
			v.alex.Do(v.tb, http.MethodPost, "/api/v1/initiatives/"+have[0].ID+"/copy", body)
		}
	}
	v.progress("template %s: %d activities, %d line items", planID, len(v.activitiesOf(planID)), len(v.lineItemsOf(planID)))
}

// globalPlans copies the template into Global Marketing Ops' plans, 50 a year from 2027 to 2030.
func (v *volume) globalPlans() {
	template := v.planNamed(TemplatePlan)
	first := v.copyPlan(template, fmt.Sprintf("Volume Global %d 001", firstGlobalYear), firstGlobalYear)
	v.moveToGlobalTeam(first)
	for year := firstGlobalYear; year <= lastGlobalYear; year++ {
		if year > firstGlobalYear {
			first = v.copyPlan(first, fmt.Sprintf("Volume Global %d 001", year), year)
		}
		for n := 2; n <= globalPlansPerYear; n++ {
			v.copyPlan(first, fmt.Sprintf("Volume Global %d %03d", year, n), year)
		}
		v.progress("Global plans of %d", year)
	}
}

func (v *volume) moveToGlobalTeam(planID string) {
	teams, err := v.set.People.ListTeams(v.ctx)
	v.must(err)
	team := teams[slices.IndexFunc(teams, func(t people.Team) bool { return t.Name == globalTeam })]
	v.write(func(ctx context.Context, _ func(string) string) error {
		p, _, err := v.set.Plans.Get(ctx, planID)
		if err != nil {
			return err
		}
		p.TeamIDs, p.OwnerID = []string{team.ID}, alexID
		return v.set.Plans.Put(ctx, p)
	})
}

func (v *volume) fullest(list []activities.Activity) activities.Activity {
	var fullest activities.Activity
	most := -1
	for _, a := range list {
		items, err := v.set.LineItems.ListByActivity(v.ctx, a.ID)
		v.must(err)
		if len(items) > most {
			fullest, most = a, len(items)
		}
	}
	return fullest
}

func (v *volume) planNamed(name string) string {
	all, err := v.set.Plans.List(v.ctx)
	v.must(err)
	for _, p := range all {
		if p.Name == name {
			return p.ID
		}
	}
	v.tb.Fatalf("apibench: no plan named %s", name)
	return ""
}

func (v *volume) campaignsOf(planID string) []campaigns.Campaign {
	list, err := v.set.Campaigns.CampaignsOfPlan(v.ctx, planID)
	v.must(err)
	return list
}

func (v *volume) initiativesOf(planID string) []campaigns.Initiative {
	var campaignIDs []string
	for _, c := range v.campaignsOf(planID) {
		campaignIDs = append(campaignIDs, c.ID)
	}
	list, err := v.set.Campaigns.InitiativesOfCampaigns(v.ctx, campaignIDs)
	v.must(err)
	return list
}

func (v *volume) activitiesOf(planID string) []activities.Activity {
	var initiativeIDs []string
	for _, i := range v.initiativesOf(planID) {
		initiativeIDs = append(initiativeIDs, i.ID)
	}
	list, err := v.set.Activities.ListByInitiatives(v.ctx, initiativeIDs)
	v.must(err)
	return list
}

func (v *volume) lineItemsOf(planID string) []lineitems.LineItem {
	var activityIDs []string
	for _, a := range v.activitiesOf(planID) {
		activityIDs = append(activityIDs, a.ID)
	}
	list, err := v.set.LineItems.ListByActivities(v.ctx, activityIDs)
	v.must(err)
	return list
}
