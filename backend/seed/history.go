package seed

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/settings"
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
	planStatusWords     = map[plans.Status]string{plans.StatusDraft: "Draft", plans.StatusPublished: "Published"}
	activityStatusWords = map[activities.Status]string{
		activities.StatusCreated: "Created", activities.StatusSubmitted: "Submitted",
		activities.StatusDeclined: "Declined", activities.StatusApproved: "Approved",
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
		name: "Fleet expo stand", status: activities.StatusCreated, activityType: "Event", owner: "Priya Raman",
		start: "2026-06-01", end: "2026-06-30", plannedCost: "18000.00", workflow: lubricantsFlow,
		attributes: lubricantsActivity("Germany", "DE03", "DE03000150", "Rimula", "FIT4-0049")}
	binLineItem = demoLineItem{plan: lubricantsPlan, activity: "Fleet expo stand", name: "Stand build", currency: "USD",
		plannedCost: "18000.00", method: lineitems.SpreadEvenly, spendType: "BTL – Event",
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
	if h.records, err = loadTree(ctx); err != nil {
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

	repo := history.NewJSONRepository()
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
	list, err := people.NewJSONRepository().List(ctx)
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

// treeRecord is one stored planning record, with its path.
type treeRecord struct {
	kind     kindLabel
	id, name string
	ownerID  string
	path     []history.RecordRef
	plan     *plans.Plan          // set for plans
	activity *activities.Activity // set for activities
}

// loadTree reads every stored plan, campaign, initiative, activity and line item, in tree order: each plan,
// then each of its campaigns followed by everything under it, and so on down to the line items, by ID at
// every level, so parents always come before their children.
func loadTree(ctx context.Context) ([]treeRecord, error) {
	planList, err := plans.NewJSONRepository().List(ctx)
	if err != nil {
		return nil, err
	}
	campaignRepo := campaigns.NewJSONRepository()
	campaignList, err := campaignRepo.ListCampaigns(ctx)
	if err != nil {
		return nil, err
	}
	initiativeList, err := campaignRepo.ListInitiatives(ctx)
	if err != nil {
		return nil, err
	}
	activityList, err := activities.NewJSONRepository().List(ctx)
	if err != nil {
		return nil, err
	}
	lineItemList, err := lineitems.NewJSONRepository().List(ctx)
	if err != nil {
		return nil, err
	}
	// The repositories list records by ID (the store's order), so every level is in ID order.
	var records []treeRecord
	visit := func(r treeRecord) []history.RecordRef {
		records = append(records, r)
		return append(slices.Clone(r.path), ref(r.kind, r.id, r.name))
	}
	for i := range planList {
		p := &planList[i]
		planPath := visit(treeRecord{kind: planningKinds.plan, id: p.ID, name: p.Name, ownerID: p.OwnerID, path: []history.RecordRef{}, plan: p})
		for _, c := range campaignList {
			if c.PlanID != p.ID {
				continue
			}
			campaignPath := visit(treeRecord{kind: planningKinds.campaign, id: c.ID, name: c.Name, ownerID: deref(c.OwnerID), path: planPath})
			for _, in := range initiativeList {
				if in.CampaignID != c.ID {
					continue
				}
				initiativePath := visit(treeRecord{kind: planningKinds.initiative, id: in.ID, name: in.Name, ownerID: deref(in.OwnerID), path: campaignPath})
				for j := range activityList {
					a := &activityList[j]
					if a.InitiativeID != in.ID {
						continue
					}
					activityPath := visit(treeRecord{kind: planningKinds.activity, id: a.ID, name: a.Name, ownerID: a.OwnerID,
						path: initiativePath, activity: a})
					for _, li := range lineItemList {
						if li.ActivityID == a.ID {
							visit(treeRecord{kind: planningKinds.lineItem, id: li.ID, name: li.Name, ownerID: li.OwnerID, path: activityPath})
						}
					}
				}
			}
		}
	}
	return records, nil
}

// deref returns the text p points at, or "" for nil.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// addCreated keeps one "created" entry per seeded planning record, by its owner, spread evenly between 14
// and 10 days before the seed time in tree order, so parents come before their children (step 1).
func (h *demoHistory) addCreated(context.Context) error {
	first, last := h.at.Add(-createdFrom), h.at.Add(-createdUntil)
	step := time.Duration(0)
	if n := len(h.records); n > 1 {
		step = last.Sub(first) / time.Duration(n-1)
	}
	for i, r := range h.records {
		at := wholeMinuteFrom(first.Add(time.Duration(i) * step))
		h.add(at, h.ownerOrAlex(r.ownerID), history.ActionCreated, string(history.ActionCreated), r.kind, r.id, r.name,
			r.path, nil, "", "")
	}
	return nil
}

// wholeMinuteFrom returns t, or the next whole minute when t has seconds: a time on the minute that is never
// earlier than t, so the first entry stays within the 14 days whatever the seed time's seconds are.
func wholeMinuteFrom(t time.Time) time.Time {
	if minute := t.Truncate(time.Minute); minute.Before(t) {
		return minute.Add(time.Minute)
	}
	return t
}

// addPublished keeps one "published" entry for every Published plan, by its owner, 7 days before (step 2).
func (h *demoHistory) addPublished(context.Context) error {
	for _, r := range h.records {
		if r.plan == nil || r.plan.Status != plans.StatusPublished {
			continue
		}
		change, err := statusChange(plans.StatusDraft, plans.StatusPublished, planStatusWords)
		if err != nil {
			return err
		}
		h.add(h.at.Add(-publishedAgo), h.ownerOrAlex(r.ownerID), history.ActionStatusChanged, "published",
			r.kind, r.id, r.name, r.path, []history.Change{change}, "", "")
	}
	return nil
}

// statusChange is the change of a record's "status" field from one stored status to another, with the
// words people read.
func statusChange[S ~string](from, to S, words map[S]string) (history.Change, error) {
	oldValue, err := json.Marshal(from)
	if err != nil {
		return history.Change{}, err
	}
	newValue, err := json.Marshal(to)
	if err != nil {
		return history.Change{}, err
	}
	return history.Change{Field: "status", Label: "Status", Old: oldValue, New: newValue, OldText: words[from], NewText: words[to]}, nil
}

// addApprovalActions keeps one "statusChanged" entry for every approval action of 008's seed in the last 14
// days, as 008 logs it on the activity: the verb ("submitted", "re-submitted", "approved", "approved level
// 1 of", "declined"), the actor, the time, the comment, the round's details and the status change (step 3).
// Rounds that ended before the log starts are left out; the activity's own approval history keeps them.
func (h *demoHistory) addApprovalActions(ctx context.Context) error {
	repo := approvals.NewJSONRepository()
	rounds, err := repo.ListRounds(ctx)
	if err != nil {
		return err
	}
	actions, err := repo.ListActions(ctx)
	if err != nil {
		return err
	}
	roundByID := map[string]approvals.Round{}
	for _, round := range rounds {
		roundByID[round.ID] = round
	}
	activityByID := map[string]treeRecord{}
	for _, r := range h.records {
		if r.activity != nil {
			activityByID[r.id] = r
		}
	}

	// Every action in time order, so each one knows the status the activity had before it.
	// The repository lists them by ID; a stable sort keeps that order for actions of the same moment.
	slices.SortStableFunc(actions, func(x, y approvals.Action) int { return x.At.Compare(y.At) })
	logStart := h.at.Add(-HistoryDays * day)
	status := map[string]activities.Status{} // activity ID → its status after the actions so far
	for _, action := range actions {
		round, found := roundByID[action.RoundID]
		activity, stored := activityByID[action.ActivityID]
		if !found || !stored {
			return fmt.Errorf("seed: approval action %s has no round or activity", action.ID)
		}
		before := cmp.Or(status[action.ActivityID], activities.StatusCreated)
		after, verb, details := approvalStep(action, round, before)
		status[action.ActivityID] = after
		if action.At.Before(logStart) {
			continue
		}
		changes := []history.Change{}
		if after != before {
			change, err := statusChange(before, after, activityStatusWords)
			if err != nil {
				return err
			}
			changes = append(changes, change)
		}
		h.add(action.At, action.Actor, history.ActionStatusChanged, verb, activity.kind, activity.id, activity.name,
			activity.path, changes, action.Comment, details)
	}
	return nil
}

// approvalStep returns what one approval action did, as 008 logs it (approvals rounds.go): the activity's
// status after it, the verb and the details.
func approvalStep(action approvals.Action, round approvals.Round, before activities.Status) (activities.Status, string, string) {
	approverNames := func(level int) string {
		list := make([]string, 0, len(round.Levels[level-1].Approvers))
		for _, a := range round.Levels[level-1].Approvers {
			list = append(list, a.Name)
		}
		return strings.Join(list, ", ")
	}
	switch action.Action {
	case approvals.ActionSubmitted, approvals.ActionResubmitted:
		verb := "submitted"
		if action.Action == approvals.ActionResubmitted {
			verb = "re-submitted"
		}
		return activities.StatusSubmitted, verb,
			fmt.Sprintf("Round %d · %s · waiting for level 1 (%s)", round.Number, round.WorkflowName, approverNames(1))
	}
	decided := fmt.Sprintf("Round %d · level %d of %d", round.Number, action.Level, len(round.Levels))
	suffix := ""
	if len(action.OnBehalfOf) > 0 {
		names := make([]string, 0, len(action.OnBehalfOf))
		for _, a := range action.OnBehalfOf {
			names = append(names, a.Name)
		}
		suffix = "; on behalf of " + strings.Join(names, ", ")
	}
	switch {
	case action.Action == approvals.ActionDeclined:
		return activities.StatusDeclined, "declined", decided + suffix
	case action.Level == len(round.Levels):
		return activities.StatusApproved, "approved", decided + suffix
	default:
		next := action.Level + 1
		return before, fmt.Sprintf("approved level %d of", action.Level),
			fmt.Sprintf("%s · now waiting for level %d (%s)%s", decided, next, approverNames(next), suffix)
	}
}

// addCostChange keeps one "changed" entry by Priya Raman 5 days before: the planned cost of her first
// FY2026 Lubricants Europe activity with a planned cost went up by 5,000.00 USD to what is stored (step 4).
func (h *demoHistory) addCostChange(context.Context) error {
	var r resolver
	priyaID := r.id(h.demo.people, "person", "Priya Raman")
	planID := r.id(h.plans, "plan", lubricantsPlan)
	if r.err != nil {
		return r.err
	}
	i := slices.IndexFunc(h.records, func(rec treeRecord) bool {
		return rec.activity != nil && rec.ownerID == priyaID && rec.activity.PlannedCost != nil && rec.path[0].ID == planID
	})
	if i < 0 {
		return fmt.Errorf("seed: Priya Raman has no activity with a planned cost in %s", lubricantsPlan)
	}
	activity := h.records[i]
	now := *activity.activity.PlannedCost
	by, err := money.ParseAmount(costChangedBy)
	if err != nil {
		return err
	}
	before := money.Money{Amount: now.Amount.Sub(by), Currency: now.Currency}
	oldValue, err := json.Marshal(before)
	if err != nil {
		return err
	}
	newValue, err := json.Marshal(now)
	if err != nil {
		return err
	}
	change := history.Change{Field: "plannedCost", Label: "Planned cost", Old: oldValue, New: newValue,
		OldText: before.Format(), NewText: now.Format()}
	h.add(h.at.Add(-costChangedAgo), h.people[priyaID], history.ActionChanged, string(history.ActionChanged),
		activity.kind, activity.id, activity.name, activity.path, []history.Change{change}, "", "")
	return nil
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

	initiative, err := h.initiativeOf(binInitiative, campaignID)
	if err != nil {
		return history.BinEntry{}, err
	}
	if initiative.ID, err = h.next(ctx, ids.PrefixInitiative); err != nil {
		return history.BinEntry{}, err
	}
	initiative.Meta = meta

	activity, err := h.activityOf(binActivity, initiative.ID)
	if err != nil {
		return history.BinEntry{}, err
	}
	if activity.ID, err = h.next(ctx, ids.PrefixActivity); err != nil {
		return history.BinEntry{}, err
	}
	activity.Meta = record.NewMeta(priya, created.Add(binCreatedSpread))

	currency, found, err := settings.NewJSONRepository().GetCurrency(ctx, binLineItem.currency)
	if err != nil || !found {
		return history.BinEntry{}, cmpErr(err, fmt.Errorf("seed: there is no currency %s", binLineItem.currency))
	}
	lineItem, err := h.lineItemFields(binLineItem, activity, currency.Rate)
	if err != nil {
		return history.BinEntry{}, fmt.Errorf("seed: line item %s: %w", binLineItem.name, err)
	}
	if lineItem.ID, err = h.next(ctx, ids.PrefixLineItem); err != nil {
		return history.BinEntry{}, err
	}
	lineItem.Meta = record.NewMeta(priya, created.Add(2*binCreatedSpread))

	path := []history.RecordRef{ref(planningKinds.plan, planID, lubricantsPlan), ref(planningKinds.campaign, campaignID, binActivity.campaign)}
	initiativePath := append(slices.Clone(path), ref(planningKinds.initiative, initiative.ID, initiative.Name))
	activityPath := append(slices.Clone(initiativePath), ref(planningKinds.activity, activity.ID, activity.Name))
	h.add(initiative.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.initiative,
		initiative.ID, initiative.Name, path, nil, "", "")
	h.add(activity.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.activity,
		activity.ID, activity.Name, initiativePath, nil, "", "")
	h.add(lineItem.Meta.CreatedAt, priya, history.ActionCreated, string(history.ActionCreated), planningKinds.lineItem,
		lineItem.ID, lineItem.Name, activityPath, nil, "", "")

	snapshots := make([]history.Snapshot, 0, 3)
	for _, s := range []struct {
		kind               kindLabel
		id, name, parentID string
		data               any
	}{
		{planningKinds.initiative, initiative.ID, initiative.Name, campaignID, initiative},
		{planningKinds.activity, activity.ID, activity.Name, initiative.ID, activity},
		{planningKinds.lineItem, lineItem.ID, lineItem.Name, activity.ID, lineItem},
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
