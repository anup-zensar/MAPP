package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/activity"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/record"
)

// HistoryDays is how far back the demo history goes: every activity-log entry of the seed is dated within
// the 14 days before the seed time (011 plan "Demo data").
const HistoryDays = 14

// The record kinds of the planning levels, with their mid-sentence names, as features 004–007 register them
// with history (the kinds.go file of each feature).
var planningKinds = struct{ plan, campaign, initiative, activity, lineItem kindLabel }{
	plan:       kindLabel{"plan", "plan"},
	campaign:   kindLabel{"campaign", "campaign"},
	initiative: kindLabel{"initiative", "initiative"},
	activity:   kindLabel{"activity", "activity"},
	lineItem:   kindLabel{"lineItem", "line item"},
}

// kindLabel is a history kind's name and its mid-sentence name.
type kindLabel struct {
	name, singular string
}

// The words of the stored statuses, as the activity log shows them (004's plan statuses, 006's activity
// statuses).
var (
	planStatusWords     = map[plan.Status]string{plan.StatusDraft: "Draft", plan.StatusPublished: "Published"}
	activityStatusWords = map[activity.Status]string{
		activity.StatusCreated: "Created", activity.StatusSubmitted: "Submitted",
		activity.StatusDeclined: "Declined", activity.StatusApproved: "Approved",
	}
)

// The deleted initiative of the demo story (011 plan "Demo data", step 5): "Trade shows" in Rimula fleet of
// FY2026 Lubricants Europe, with its activity "Fleet expo stand" and that activity's line item "Stand
// build". Priya Raman created them 12 days before the seed time and deleted the initiative 2 days before,
// so they are only in the recycle bin.
var (
	binInitiative = demoInitiative{name: "Trade shows", owner: "Priya Raman", start: "2026-02-01", end: "2026-12-31",
		budget: "20000.00", attributes: notFIT4}
	binActivity = demoActivity{plan: lubricantsPlan, campaign: "Rimula fleet", initiative: "Trade shows",
		name: "Fleet expo stand", status: activity.StatusCreated, activityType: "Event", owner: "Priya Raman",
		start: "2026-06-01", end: "2026-06-30", plannedCost: "18000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("Germany", "DE03", "DE03000150", "Rimula", "FIT4-0049")}
	binLineItem = demoLineItem{plan: lubricantsPlan, activity: "Fleet expo stand", name: "Stand build", currency: "USD",
		plannedCost: "18000.00", method: lineitem.SpreadEvenly, spendType: "BTL – Event",
		glCode: "7250040 – Events and sponsorship", budget: "2026 Lubricants Budget", item: "Europe – Events", requestSwbs: true}
)

// The times of the demo story, before the seed time.
const (
	createdFrom      = HistoryDays * day // the first seeded record was created 14 days before
	createdUntil     = 10 * day          // the last one 10 days before: before the first approval action logged
	publishedAgo     = 7 * day           // the Published plans were published 7 days before
	costChangedAgo   = 5 * day           // Priya Raman changed a planned cost 5 days before
	costChangedBy    = "5000.00"         // … by this many US dollars, up to the stored planned cost
	binCreatedAgo    = 12 * day          // Priya Raman created the deleted initiative 12 days before
	binDeletedAgo    = 2 * day           // … and deleted it 2 days before
	binCreatedSpread = time.Minute       // its activity a minute later, its line item a minute after that
)

// addHistory writes the demo history (feature 011, D-188, D-202), the last step of Load: about two weeks of
// activity-log entries and one recycle-bin entry, so the Activity log, the Recycle bin and Home's Recent
// changes are not empty in a demo. It writes straight into the history repository, not through the history
// service (whose kinds the API registers): it builds every entry first, sorts them by time and only then
// appends them, so the sequence numbers follow time. The entries are, as 011 plan "Demo data" lists them:
//
//  1. one "created" entry for every seeded plan, campaign, initiative, activity and line item, by its owner
//     (Alex Morgan when it has none), between 14 and 10 days before, parents before children;
//  2. one "published" entry for every Published plan, by its owner, 7 days before;
//  3. one "statusChanged" entry for every approval action of 008's seed in those 14 days, with 008's verb,
//     actor, time, comment and details (older rounds, approved weeks before, are not in the log);
//  4. one "changed" entry by Priya Raman 5 days before: the planned cost of her first FY2026 Lubricants
//     Europe activity with one, 5,000.00 USD lower before;
//  5. the deleted initiative "Trade shows" (binInitiative): "created" entries 12 days before, then its
//     recycle-bin entry and "deleted" entry 2 days before. Its records get the next IDs but are never stored.
func (d *demo) addHistory(ctx context.Context) error {
	var r resolver
	priyaID := r.id(d.people, "person", "Priya Raman")
	alexID := r.id(d.people, "person", "Alex Morgan")
	if r.err != nil {
		return r.err
	}
	h := &demoHistory{demo: d, people: map[string]actor.Actor{}}
	err := h.loadPeople(ctx)
	if err != nil {
		return err
	}
	h.fallbackOwner = h.people[alexID]
	if h.records, err = h.loadTree(ctx); err != nil {
		return err
	}

	steps := []func(context.Context) error{h.addCreated, h.addPublished, h.addApprovalActions, h.addCostChange}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}
	bin, err := h.addDeletedInitiative(ctx, h.people[priyaID])
	if err != nil {
		return err
	}

	repo := d.stores.History
	// Time order; entries of the same moment keep the order they were made in (parents before children).
	slices.SortStableFunc(h.entries, func(x, y history.Entry) int { return x.At.Compare(y.At) })
	for _, e := range h.entries {
		seq, err := d.ids.NextNumber(ctx, ids.PrefixLogEntry)
		if err != nil {
			return err
		}
		e.ID, e.Seq = fmt.Sprintf("%s-%d", ids.PrefixLogEntry, seq), seq // the form of ids.Generator.Next: LOG-1001
		if err := repo.AppendEntry(ctx, e); err != nil {
			return err
		}
	}
	return repo.PutBinEntry(ctx, bin)
}

// demoHistory is the state of addHistory: the people by ID and the entries made so far, not yet sorted.
type demoHistory struct {
	*demo
	people        map[string]actor.Actor // person ID → the person as an actor
	fallbackOwner actor.Actor            // who created a record without an owner: Alex Morgan
	records       []treeRecord           // the stored planning records, in tree order
	entries       []history.Entry
}

// loadPeople reads the seeded people, to name the actors of the entries.
func (h *demoHistory) loadPeople(ctx context.Context) error {
	list, err := h.stores.People.List(ctx)
	if err != nil {
		return err
	}
	for _, p := range list {
		h.people[p.ID] = p.Actor()
	}
	return nil
}

// ownerOrAlex returns the person ownerID as an actor, or Alex Morgan when the record has no owner.
func (h *demoHistory) ownerOrAlex(ownerID string) actor.Actor {
	if a, found := h.people[ownerID]; found {
		return a
	}
	return h.fallbackOwner
}

// ref points at a record of kind k for an entry's path.
func ref(k kindLabel, id, name string) history.RecordRef {
	return history.RecordRef{Type: k.name, ID: id, Name: name}
}

// add keeps one entry about the record id of kind k, without its ID and sequence number.
func (h *demoHistory) add(at time.Time, by actor.Actor, action history.Action, verb string, k kindLabel, id, name string,
	path []history.RecordRef, changes []history.Change, comment, details string) {
	if changes == nil {
		changes = []history.Change{}
	}
	h.entries = append(h.entries, history.Entry{
		At: at, Actor: by, Action: action, Verb: verb, RecordType: k.name, RecordTypeLabel: k.singular,
		RecordID: id, RecordName: name, Path: slices.Clone(path), Changes: changes, Comment: comment, Details: details,
	})
}

// deref returns the text p points at, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// addDeletedInitiative builds the deleted initiative "Trade shows" with its activity and line item, keeps
// their "created" entries (12 days before) and the "deleted" entry (2 days before), and returns the
// recycle-bin entry that holds them (step 5). The records get the next IDs, so those IDs are never given
// out again, but are not stored: they are only in the recycle bin, as after a real delete.
func (h *demoHistory) addDeletedInitiative(ctx context.Context, priya actor.Actor) (history.BinEntry, error) {
	var r resolver
	planID := r.id(h.plans, "plan", lubricantsPlan)
	campaignID := r.id(h.campaigns, "campaign", binActivity.plan+"/"+binActivity.campaign)
	if r.err != nil {
		return history.BinEntry{}, r.err
	}
	created := h.at.Add(-binCreatedAgo)
	meta := record.NewMeta(priya, created)

	initiative, err := h.initiativeOf(binInitiative, campaignID, h.planCurrencies[binActivity.plan])
	if err != nil {
		return history.BinEntry{}, err
	}
	if initiative.ID, err = h.next(ctx, ids.PrefixInitiative); err != nil {
		return history.BinEntry{}, err
	}
	initiative.Meta = meta

	act, err := h.activityOf(binActivity, initiative.ID)
	if err != nil {
		return history.BinEntry{}, err
	}
	if act.ID, err = h.next(ctx, ids.PrefixActivity); err != nil {
		return history.BinEntry{}, err
	}
	act.Meta = record.NewMeta(priya, created.Add(binCreatedSpread))

	currency, found, err := h.stores.Settings.GetCurrency(ctx, binLineItem.currency)
	if err != nil || !found {
		return history.BinEntry{}, cmpErr(err, fmt.Errorf("seed: there is no currency %s", binLineItem.currency))
	}
	lineItem, err := h.lineItemFields(binLineItem, act, currency.Rate)
	if err != nil {
		return history.BinEntry{}, fmt.Errorf("seed: line item %s: %w", binLineItem.name, err)
	}
	if lineItem.ID, err = h.next(ctx, ids.PrefixLineItem); err != nil {
		return history.BinEntry{}, err
	}
	lineItem.Meta = record.NewMeta(priya, created.Add(2*binCreatedSpread))

	path := []history.RecordRef{ref(planningKinds.plan, planID, lubricantsPlan), ref(planningKinds.campaign, campaignID, binActivity.campaign)}
	initiativePath := append(slices.Clone(path), ref(planningKinds.initiative, initiative.ID, initiative.Name))
	activityPath := append(slices.Clone(initiativePath), ref(planningKinds.activity, act.ID, act.Name))
	h.add(initiative.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.initiative,
		initiative.ID, initiative.Name, path, nil, "", "")
	h.add(act.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.activity,
		act.ID, act.Name, initiativePath, nil, "", "")
	h.add(lineItem.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.lineItem,
		lineItem.ID, lineItem.Name, activityPath, nil, "", "")

	snapshots := make([]history.Snapshot, 0, 3)
	for _, s := range []struct {
		kind               kindLabel
		id, name, parentID string
		data               any
	}{
		{planningKinds.initiative, initiative.ID, initiative.Name, campaignID, initiative},
		{planningKinds.activity, act.ID, act.Name, initiative.ID, act},
		{planningKinds.lineItem, lineItem.ID, lineItem.Name, act.ID, lineItem},
	} {
		data, err := json.Marshal(s.data)
		if err != nil {
			return history.BinEntry{}, err
		}
		snapshots = append(snapshots, history.Snapshot{Kind: s.kind.name, ID: s.id, Name: s.name, ParentID: s.parentID, Data: data})
	}
	contents := []history.ContentCount{
		{RecordType: planningKinds.activity.name, Label: planningKinds.activity.singular, Count: 1},
		{RecordType: planningKinds.lineItem.name, Label: planningKinds.lineItem.singular, Count: 1},
	}
	deleted := h.at.Add(-binDeletedAgo)
	h.add(deleted, priya, history.ActionDeleted, string(history.ActionDeleted), planningKinds.initiative,
		initiative.ID, initiative.Name, path, nil, "", contentsDetails(contents))

	entryID, err := h.next(ctx, ids.PrefixBinEntry)
	if err != nil {
		return history.BinEntry{}, err
	}
	parent := path[len(path)-1]
	return history.BinEntry{
		ID: entryID, RecordType: planningKinds.initiative.name, RecordTypeLabel: planningKinds.initiative.singular,
		RecordID: initiative.ID, RecordName: initiative.Name, Path: path, Parent: &parent, DeletedAt: deleted,
		DeletedBy: priya, Contents: contents, Snapshots: snapshots,
	}, nil
}

// contentsDetails says what was under a deleted record, as the "deleted" entry of history's MoveToBin does:
// "with 1 activity and 1 line item".
func contentsDetails(contents []history.ContentCount) string {
	parts := make([]string, 0, len(contents))
	for _, c := range contents {
		parts = append(parts, fmt.Sprintf("%d %s", c.Count, c.Label))
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return "with " + parts[0]
	}
	return "with " + strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
}
