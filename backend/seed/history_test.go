package seed_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/activity/activitytest"
	"mapp/internal/campaign/campaigntest"
	"mapp/internal/history"
	"mapp/internal/lineitem/lineitemtest"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/record"
	"mapp/seed"
)

const day = 24 * time.Hour

// The demo history is dated within the 14 days before the seed time, in time order; the recycle bin holds the
// initiative "Trade shows", restorable, with 1 activity and 1 line item that are no longer stored.
func TestSeed_HistoryAndRecycleBin_011(t *testing.T) {
	t.Parallel()
	db := loadedStore(t)
	entries, bin := demoLog(t, db)

	if len(entries) == 0 {
		t.Fatal("the seed wrote no activity-log entries")
	}
	logStart := loadedAt.Add(-seed.HistoryDays * day)
	for i, e := range entries {
		if e.Seq != 1001+i || e.ID != fmt.Sprintf("LOG-%d", e.Seq) {
			t.Errorf("entry %d: id %s seq %d, want LOG-%d", i+1, e.ID, e.Seq, 1001+i)
		}
		if e.At.Before(logStart) || e.At.After(loadedAt) {
			t.Errorf("%s is not within the 14 days before the seed time", entryText(e))
		}
		if i > 0 && e.At.Before(entries[i-1].At) {
			t.Errorf("%s comes after %s, which is later", entryText(e), entryText(entries[i-1]))
		}
		if e.Changes == nil || e.Path == nil {
			t.Errorf("%s: changes %v and path %v, want lists", entryText(e), e.Changes, e.Path)
		}
	}

	if len(bin) != 1 {
		t.Fatalf("%d recycle-bin entries, want 1: %+v", len(bin), bin)
	}
	b := bin[0]
	wantContents := []history.ContentCount{
		{RecordType: "activity", Label: "activity", Count: 1}, {RecordType: "lineItem", Label: "line item", Count: 1},
	}
	if b.ID != "BIN-1001" || b.RecordType != "initiative" || b.RecordTypeLabel != "initiative" || b.RecordID != "I-1011" ||
		b.RecordName != "Trade shows" || b.Resolution != nil || b.DeletedBy.Name != "Priya Raman" ||
		!b.DeletedAt.Equal(loadedAt.Add(-2*day)) || !slices.Equal(b.Contents, wantContents) {
		t.Errorf("recycle-bin entry = %+v\nwant BIN-1001, initiative I-1011 'Trade shows', unresolved, deleted by Priya Raman "+
			"2 days before with 1 activity and 1 line item", b)
	}
	wantPath := []history.RecordRef{{Type: "plan", ID: "P-1001", Name: "FY2026 Lubricants Europe"},
		{Type: "campaign", ID: "C-1002", Name: "Rimula fleet"}}
	if !slices.Equal(b.Path, wantPath) || b.Parent == nil || *b.Parent != wantPath[1] {
		t.Errorf("recycle-bin entry path %+v parent %+v, want %+v", b.Path, b.Parent, wantPath)
	}
	var snapshots []string
	for _, s := range b.Snapshots {
		snapshots = append(snapshots, fmt.Sprintf("%s %s %s < %s", s.Kind, s.ID, s.Name, s.ParentID))
	}
	wantSnapshots := []string{"initiative I-1011 Trade shows < C-1002", "activity A-1013 Fleet expo stand < I-1011",
		"lineItem L-1011 Stand build < A-1013"}
	if !slices.Equal(snapshots, wantSnapshots) {
		t.Errorf("snapshots = %q, want %q", snapshots, wantSnapshots)
	}
	read(t, db, func(ctx context.Context) error {
		_, initiativeStored, err := campaigntest.NewStore(db).GetInitiative(ctx, "I-1011")
		if err != nil {
			return err
		}
		_, activityStored, err := activitytest.NewStore(db).Get(ctx, "A-1013")
		if err != nil {
			return err
		}
		_, lineItemStored, err := lineitemtest.NewStore(db).Get(ctx, "L-1011")
		if err != nil {
			return err
		}
		if initiativeStored || activityStored || lineItemStored {
			t.Errorf("stored: initiative %v, activity %v, line item %v; the deleted records are only in the recycle bin",
				initiativeStored, activityStored, lineItemStored)
		}
		return nil
	})

	deletes := slices.DeleteFunc(slices.Clone(entries), func(e history.Entry) bool { return e.Action != history.ActionDeleted })
	if len(deletes) != 1 || deletes[0].RecordID != "I-1011" || deletes[0].Details != "with 1 activity and 1 line item" ||
		deletes[0].Actor.Name != "Priya Raman" || !deletes[0].At.Equal(b.DeletedAt) || !slices.Equal(deletes[0].Path, wantPath) {
		t.Errorf("deleted entries = %+v, want one: Priya Raman deleted initiative 'Trade shows' with 1 activity and 1 line item", deletes)
	}

	// Through the API, as the Recycle bin page reads it.
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")
	response := priya.Do(t, http.MethodGet, "/api/v1/recycle-bin", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/recycle-bin = %d %s", response.Status, response.Body)
	}
	var page history.BinPage
	response.Decode(t, &page)
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].RestoreState != history.Restorable ||
		len(page.Items[0].Records) != 3 || page.Items[0].RecordTypePlural != "initiatives" {
		t.Errorf("recycle bin = %+v, want the one restorable entry with its 3 records", page)
	}
}

// Every seeded planning record has one "created" entry by its owner (Alex Morgan when it has none), 14 to 10
// days before the seed time and after its parents', which its created line matches (038 FR-060, L-8); the
// deleted initiative's three records, 12 days before.
func TestSeed_History_CreatedEntries_011_038_FR060_L8(t *testing.T) {
	t.Parallel()
	d := loadDemo(t)
	entries, _ := demoLog(t, loadedStore(t))

	created := map[string]history.Entry{}
	for _, e := range entries {
		if e.Action != history.ActionCreated {
			continue
		}
		if _, twice := created[e.RecordID]; twice {
			t.Errorf("%s: a second created entry", entryText(e))
		}
		created[e.RecordID] = e
		if e.Verb != "created" || len(e.Changes) != 0 || e.Details != "" {
			t.Errorf("%s: verb %q, changes %v, details %q; want created, none, none", entryText(e), e.Verb, e.Changes, e.Details)
		}
		for _, parent := range e.Path {
			if p, found := created[parent.ID]; !found || p.Seq > e.Seq {
				t.Errorf("%s comes before the created entry of its %s %s", entryText(e), parent.Type, parent.ID)
			}
		}
	}

	owners := map[string]string{}
	for _, p := range d.people {
		owners[p.ID] = p.Name
	}
	ownerName := func(id string) string {
		if id == "" {
			return "Alex Morgan"
		}
		return owners[id]
	}
	type seeded struct {
		kind, id, owner string
		meta            record.Meta
	}
	var records []seeded
	for _, p := range d.plans {
		records = append(records, seeded{"plan", p.ID, ownerName(p.OwnerID), p.Meta})
	}
	for _, c := range d.campaigns {
		records = append(records, seeded{"campaign", c.ID, ownerName(*c.OwnerID), c.Meta})
	}
	for _, i := range d.initiatives {
		records = append(records, seeded{"initiative", i.ID, ownerName(*i.OwnerID), i.Meta})
	}
	for _, a := range d.activities {
		records = append(records, seeded{"activity", a.ID, ownerName(a.OwnerID), a.Meta})
	}
	for _, l := range d.lineItems {
		records = append(records, seeded{"lineItem", l.ID, ownerName(l.OwnerID), l.Meta})
	}
	for _, r := range records {
		e, found := created[r.id]
		switch {
		case !found:
			t.Errorf("%s %s has no created entry", r.kind, r.id)
		case e.RecordType != r.kind || e.Actor.Name != r.owner:
			t.Errorf("%s, want a %s created by %s", entryText(e), r.kind, r.owner)
		case e.At.Before(loadedAt.Add(-14*day)) || e.At.After(loadedAt.Add(-10*day)):
			t.Errorf("%s is not between 14 and 10 days before the seed time", entryText(e))
		case !r.meta.CreatedAt.Equal(e.At) || r.meta.CreatedBy.ID != e.Actor.ID: // 038 FR-060 (L-8)
			t.Errorf("%s %s: created %s by %s, its first History entry says %s", r.kind, r.id, r.meta.CreatedAt,
				r.meta.CreatedBy.Name, entryText(e))
		}
	}
	for _, id := range []string{"I-1011", "A-1013", "L-1011"} {
		e, found := created[id]
		if !found || e.Actor.Name != "Priya Raman" || e.At.Sub(loadedAt.Add(-12*day)) > 2*time.Minute || e.At.Before(loadedAt.Add(-12*day)) {
			t.Errorf("created entry of %s = %+v, want Priya Raman 12 days before", id, e)
		}
	}
	if want := len(records) + 3; len(created) != want {
		t.Errorf("%d created entries, want %d", len(created), want)
	}
}

// Each Published plan has one "published" entry by its owner 7 days before; Priya changed the planned cost of
// her first FY2026 Lubricants Europe activity 5 days before, from 5,000.00 USD less to what is stored now.
func TestSeed_History_PublishedAndCostChange_011(t *testing.T) {
	t.Parallel()
	entries, _ := demoLog(t, loadedStore(t))

	var published, changed []history.Entry
	for _, e := range entries {
		switch {
		case e.Action == history.ActionStatusChanged && e.RecordType == "plan":
			published = append(published, e)
		case e.Action == history.ActionChanged:
			changed = append(changed, e)
		}
	}
	if len(published) != 1 {
		t.Fatalf("plan status entries = %+v, want one: FY2026 Lubricants Europe published", published)
	}
	p := published[0]
	if p.RecordID != "P-1001" || p.Verb != "published" || p.Actor.Name != "Priya Raman" || !p.At.Equal(loadedAt.Add(-7*day)) ||
		len(p.Changes) != 1 || p.Changes[0].Field != "status" || p.Changes[0].Label != "Status" ||
		p.Changes[0].OldText != "Draft" || p.Changes[0].NewText != "Published" ||
		string(p.Changes[0].Old) != `"draft"` || string(p.Changes[0].New) != `"published"` {
		t.Errorf("published entry = %+v", p)
	}

	if len(changed) != 1 {
		t.Fatalf("changed entries = %+v, want one", changed)
	}
	c := changed[0]
	wantPath := []string{"FY2026 Lubricants Europe", "Helix launch", "Digital & social"}
	var path []string
	for _, r := range c.Path {
		path = append(path, r.Name)
	}
	if c.RecordID != "A-1001" || c.RecordName != "Launch film" || c.Actor.Name != "Priya Raman" ||
		!c.At.Equal(loadedAt.Add(-5*day)) || !slices.Equal(path, wantPath) || len(c.Changes) != 1 {
		t.Fatalf("changed entry = %+v, want Priya Raman's planned cost of A-1001 'Launch film' 5 days before", c)
	}
	change := c.Changes[0]
	if change.Field != "plannedCost" || change.Label != "Planned cost" || change.OldText != "115,000.00 USD" ||
		change.NewText != "120,000.00 USD" || !strings.Contains(string(change.Old), `"115000.00"`) ||
		!strings.Contains(string(change.New), `"120000.00"`) {
		t.Errorf("planned-cost change = %+v; old %s new %s", change, change.Old, change.New)
	}
}

// The deleted initiative's entries point at the recycle bin. Priya reads every market's plans here, because the
// log covers only what its reader may see, so the summary counts the whole seeded history.
func TestSeed_History_ThroughTheAPI_011(t *testing.T) {
	t.Parallel()
	server := apitest.New(t)
	server.GrantRole(t, "priya.raman@shell.com", "viewer", "all")
	priya := server.SignIn(t, "priya.raman@shell.com")

	response := priya.Do(t, http.MethodGet, "/api/v1/activity-log?recordId=I-1011&limit=10", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/activity-log = %d %s", response.Status, response.Body)
	}
	var log history.LogPage
	response.Decode(t, &log)
	if log.Total != 2 || len(log.Items) != 2 || log.Items[0].Action != history.ActionDeleted ||
		log.Items[0].RecordNow.State != history.RecordInRecycleBin || log.Items[0].RecordNow.BinEntryID != "BIN-1001" {
		t.Errorf("entries of I-1011 = %+v, want deleted and created, in the recycle bin BIN-1001", log)
	}

	week := loadedAt.Add(-7 * day).Format(time.RFC3339)
	today := time.Date(loadedAt.Year(), loadedAt.Month(), loadedAt.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	response = priya.Do(t, http.MethodGet, "/api/v1/activity-log/summary?todayStart="+today+"&weekStart="+week, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/activity-log/summary = %d %s", response.Status, response.Body)
	}
	var summary history.LogSummary
	response.Decode(t, &summary)
	if summary.ChangesToday != 1 || summary.DeletionsThisWeek != 1 || summary.PeopleActiveThisWeek != 4 || summary.Total < 40 {
		t.Errorf("summary = %+v, want 1 change today, 1 deletion and 4 people active this week", summary)
	}

	response = priya.Do(t, http.MethodGet, "/api/v1/home/recent-changes?scope=myTeams", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/v1/home/recent-changes = %d %s", response.Status, response.Body)
	}
	var recent struct {
		Items []history.EntryView `json:"items"`
		Total int                 `json:"total"`
	}
	response.Decode(t, &recent)
	if len(recent.Items) == 0 || recent.Items[0].RecordID != "I-1011" || recent.Items[0].Action != history.ActionDeleted {
		t.Errorf("Priya's recent changes = %+v, want the deleted initiative first", recent)
	}
}
