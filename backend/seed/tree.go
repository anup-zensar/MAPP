package seed

import (
	"context"
	"slices"
	"time"

	"mapp/internal/activity"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/record"
	"mapp/internal/stores"
)

// treeRecord is one stored planning record with its path; plan or activity is set for those kinds.
type treeRecord struct {
	kind     kindLabel
	id, name string
	ownerID  string
	path     []history.RecordRef
	plan     *plan.Plan
	activity *activity.Activity
	created  func(ctx context.Context, at time.Time, by actor.Actor) error // stores the record created at by
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
			path: []history.RecordRef{}, plan: p, created: stamp(&p.Meta, func(ctx context.Context) error {
				return d.stores.Plans.Put(ctx, *p)
			})}))
	}
	return t.records, nil
}

// stamp returns what makes the record of meta created at by, as its first History entry says.
func stamp(meta *record.Meta, put func(context.Context) error) func(context.Context, time.Time, actor.Actor) error {
	return func(ctx context.Context, at time.Time, by actor.Actor) error {
		if meta.Version == 1 {
			meta.UpdatedAt, meta.UpdatedBy = at, by
		}
		meta.CreatedAt, meta.CreatedBy = at, by
		return put(ctx)
	}
}

type planningTree struct {
	stores      stores.Set
	campaigns   []campaign.Campaign
	initiatives []campaign.Initiative
	activities  []activity.Activity
	lineItems   []lineitem.LineItem
	records     []treeRecord
}

func (d *demo) readTree(ctx context.Context) (*planningTree, error) {
	t := &planningTree{stores: d.stores}
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
	for i := range t.campaigns {
		c := &t.campaigns[i]
		if c.PlanID == planID {
			put := func(ctx context.Context) error { return t.stores.Campaigns.PutCampaign(ctx, *c) }
			t.addInitiatives(c.ID, t.visit(treeRecord{kind: planningKinds.campaign, id: c.ID, name: c.Name,
				ownerID: deref(c.OwnerID), path: path, created: stamp(&c.Meta, put)}))
		}
	}
}

func (t *planningTree) addInitiatives(campaignID string, path []history.RecordRef) {
	for i := range t.initiatives {
		in := &t.initiatives[i]
		if in.CampaignID == campaignID {
			put := func(ctx context.Context) error { return t.stores.Campaigns.PutInitiative(ctx, *in) }
			t.addActivities(in.ID, t.visit(treeRecord{kind: planningKinds.initiative, id: in.ID, name: in.Name,
				ownerID: deref(in.OwnerID), path: path, created: stamp(&in.Meta, put)}))
		}
	}
}

func (t *planningTree) addActivities(initiativeID string, path []history.RecordRef) {
	for j := range t.activities {
		a := &t.activities[j]
		if a.InitiativeID == initiativeID {
			put := func(ctx context.Context) error { return t.stores.Activities.Put(ctx, *a) }
			t.addLineItems(a.ID, t.visit(treeRecord{kind: planningKinds.activity, id: a.ID, name: a.Name,
				ownerID: a.OwnerID, path: path, activity: a, created: stamp(&a.Meta, put)}))
		}
	}
}

func (t *planningTree) addLineItems(activityID string, path []history.RecordRef) {
	for i := range t.lineItems {
		li := &t.lineItems[i]
		if li.ActivityID == activityID {
			put := func(ctx context.Context) error { return t.stores.LineItems.Put(ctx, *li) }
			t.visit(treeRecord{kind: planningKinds.lineItem, id: li.ID, name: li.Name, ownerID: li.OwnerID, path: path,
				created: stamp(&li.Meta, put)})
		}
	}
}
