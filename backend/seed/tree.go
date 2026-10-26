package seed

import (
	"context"
	"slices"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/plans"
)

// treeRecord is one stored planning record with its path; plan or activity is set for those kinds.
type treeRecord struct {
	kind     kindLabel
	id, name string
	ownerID  string
	path     []history.RecordRef
	plan     *plans.Plan
	activity *activities.Activity
}

// loadTree lists every stored planning record in tree order — each plan, then each of its campaigns with
// everything under it, by ID at every level — so parents always come before their children.
func (d *demo) loadTree(ctx context.Context) ([]treeRecord, error) {
	planList, err := d.stores.Plans.List(ctx)
	if err != nil {
		return nil, err
	}
	t, err := d.readTree(ctx)
	if err != nil {
		return nil, err
	}
	for i := range planList {
		p := &planList[i]
		t.addCampaigns(p.ID, t.visit(treeRecord{kind: planningKinds.plan, id: p.ID, name: p.Name, ownerID: p.OwnerID,
			path: []history.RecordRef{}, plan: p}))
	}
	return t.records, nil
}

type planningTree struct {
	campaigns   []campaigns.Campaign
	initiatives []campaigns.Initiative
	activities  []activities.Activity
	lineItems   []lineitems.LineItem
	records     []treeRecord
}

func (d *demo) readTree(ctx context.Context) (*planningTree, error) {
	t := &planningTree{}
	var err error
	if t.campaigns, err = d.stores.Campaigns.ListCampaigns(ctx); err != nil {
		return nil, err
	}
	if t.initiatives, err = d.stores.Campaigns.ListInitiatives(ctx); err != nil {
		return nil, err
	}
	if t.activities, err = d.stores.Activities.List(ctx); err != nil {
		return nil, err
	}
	if t.lineItems, err = d.stores.LineItems.List(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *planningTree) visit(r treeRecord) []history.RecordRef {
	t.records = append(t.records, r)
	return append(slices.Clone(r.path), ref(r.kind, r.id, r.name))
}

func (t *planningTree) addCampaigns(planID string, path []history.RecordRef) {
	for _, c := range t.campaigns {
		if c.PlanID == planID {
			t.addInitiatives(c.ID, t.visit(treeRecord{kind: planningKinds.campaign, id: c.ID, name: c.Name,
				ownerID: deref(c.OwnerID), path: path}))
		}
	}
}

func (t *planningTree) addInitiatives(campaignID string, path []history.RecordRef) {
	for _, in := range t.initiatives {
		if in.CampaignID == campaignID {
			t.addActivities(in.ID, t.visit(treeRecord{kind: planningKinds.initiative, id: in.ID, name: in.Name,
				ownerID: deref(in.OwnerID), path: path}))
		}
	}
}

func (t *planningTree) addActivities(initiativeID string, path []history.RecordRef) {
	for j := range t.activities {
		a := &t.activities[j]
		if a.InitiativeID == initiativeID {
			t.addLineItems(a.ID, t.visit(treeRecord{kind: planningKinds.activity, id: a.ID, name: a.Name,
				ownerID: a.OwnerID, path: path, activity: a}))
		}
	}
}

func (t *planningTree) addLineItems(activityID string, path []history.RecordRef) {
	for _, li := range t.lineItems {
		if li.ActivityID == activityID {
			t.visit(treeRecord{kind: planningKinds.lineItem, id: li.ID, name: li.Name, ownerID: li.OwnerID, path: path})
		}
	}
}
