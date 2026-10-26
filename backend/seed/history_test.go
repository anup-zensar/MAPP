package seed_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/lineitems"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/seed"
)

const day = 24 * time.Hour

// demoLog reads the activity log and the recycle bin the seed wrote.
func demoLog(t *testing.T, db *store.DB) ([]history.Entry, []history.BinEntry) {
	t.Helper()
	var entries []history.Entry
	var bin []history.BinEntry
	read(t, db, func(ctx context.Context) error {
		repo := history.NewJSONRepository()
		var err error
		if entries, err = repo.Entries(ctx); err != nil {
			return err
		}
		bin, err = repo.BinEntries(ctx)
		return err
	})
	return entries, bin
}

// entryText is an entry as a failure message shows it.
func entryText(e history.Entry) string {
	return fmt.Sprintf("%s %s %s %s %s %s '%s'", e.ID, e.At.Format(time.RFC3339), e.Actor.Name, e.Verb, e.RecordType, e.RecordID, e.RecordName)
}

// The seed's demo history (011 plan "Demo data", T019): every entry is dated within the 14 days before the
// seed time, in time order, with sequence numbers that follow time; the recycle bin holds one unresolved
// entry, the initiative "Trade shows" with 1 activity and 1 line item, whose records are not stored and
// whose restore state is restorable (D-188, FR-015, FR-017).
func TestSeed_HistoryAndRecycleBin_011(t *testing.T) {
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
		_, initiativeStored, err := campaigns.NewJSONRepository().GetInitiative(ctx, "I-1011")
		if err != nil {
			return err
		}
		_, activityStored, err := activities.NewJSONRepository().Get(ctx, "A-1013")
		if err != nil {
			return err
		}
		_, lineItemStored, err := lineitems.NewJSONRepository().Get(ctx, "L-1011")
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
	response := priya.Do(t, http.MethodGet, "/api/recycle-bin", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/recycle-bin = %d %s", response.Status, response.Body)
	}
	var page history.BinPage
	response.Decode(t, &page)
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].RestoreState != history.Restorable ||
		len(page.Items[0].Records) != 3 || page.Items[0].RecordTypePlural != "initiatives" {
		t.Errorf("recycle bin = %+v, want the one restorable entry with its 3 records", page)
	}
}

// Every seeded plan, campaign, initiative, activity and line item has exactly one "created" entry, by its
// owner (Alex Morgan when it has none), between 14 and 10 days before the seed time, after the entries of
// its parents; so have the three records of the deleted initiative, 12 days before (011 plan "Demo data"
// steps 1 and 5).
func TestSeed_History_CreatedEntries_011(t *testing.T) {
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

	owners := map[string]string{} // person ID → name
	for _, p := range d.people {
		owners[p.ID] = p.Name
	}
	ownerName := func(id string) string {
		if id == "" {
			return "Alex Morgan"
		}
		return owners[id]
	}
	type seeded struct{ kind, id, owner string }
	var records []seeded
	for _, p := range d.plans {
		records = append(records, seeded{"plan", p.ID, ownerName(p.OwnerID)})
	}
	for _, c := range d.campaigns {
		records = append(records, seeded{"campaign", c.ID, ownerName(*c.OwnerID)})
	}
	for _, i := range d.initiatives {
		records = append(records, seeded{"initiative", i.ID, ownerName(*i.OwnerID)})
	}
	for _, a := range d.activities {
		records = append(records, seeded{"activity", a.ID, ownerName(a.OwnerID)})
	}
	for _, l := range d.lineItems {
		records = append(records, seeded{"lineItem", l.ID, ownerName(l.OwnerID)})
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

// Every Published plan has one "published" entry by its owner 7 days before, Draft → Published; and Priya
// Raman changed the planned cost of her first FY2026 Lubricants Europe activity 5 days before, from 5,000.00
// USD less to what is stored now (011 plan "Demo data" steps 2 and 4).
func TestSeed_History_PublishedAndCostChange_011(t *testing.T) {
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

// Every approval action 008's seed dates within the 14 days has one "statusChanged" entry on its activity
// with the same actor, time and comment, 008's verb and details and the status change; older actions have
// none (011 plan "Demo data" step 3).
func TestSeed_History_ApprovalActions_011(t *testing.T) {
	d := loadDemo(t)
	entries, _ := demoLog(t, loadedStore(t))

	var got []string
	for _, e := range entries {
		if e.Action == history.ActionStatusChanged && e.RecordType == "activity" {
			status := ""
			if len(e.Changes) == 1 {
				status = " " + e.Changes[0].OldText + " → " + e.Changes[0].NewText
			}
			got = append(got, fmt.Sprintf("%s %s %s %s (%s)%s | %s | %s", loadedAt.Sub(e.At), e.Actor.Name, e.Verb, e.RecordName,
				e.RecordID, status, e.Details, e.Comment))
		}
	}
	split := "Please split the agency fee by country before re-submitting."
	budget := "The planned cost is above this quarter's budget — please reduce it or move part of it to Q3."
	want := []string{
		"216h0m0s Priya Raman submitted Trade magazine ads (A-1004) Created → Submitted | Round 1 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"192h0m0s Daniel Okoro declined Trade magazine ads (A-1004) Submitted → Declined | Round 1 · level 1 of 1 | " + split,
		"144h0m0s Priya Raman submitted Fleet roadshow (A-1005) Created → Submitted | Round 1 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"120h0m0s Daniel Okoro declined Fleet roadshow (A-1005) Submitted → Declined | Round 1 · level 1 of 1 | " + budget,
		"96h0m0s Mei Lin submitted Fleet card launch (A-1009) Created → Submitted | Round 1 · China BC approval workflow · waiting for level 1 (Carlos Mendes) | ",
		"48h0m0s Priya Raman re-submitted Trade magazine ads (A-1004) Declined → Submitted | Round 2 · Lubricants EU approval · waiting for level 1 (Daniel Okoro) | ",
		"2h0m0s Carlos Mendes approved level 1 of Fleet card launch (A-1009) | Round 1 · level 1 of 2 · now waiting for level 2 (Alex Morgan) | ",
	}
	if !slices.Equal(got, want) {
		t.Errorf("activity status entries:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	inWindow := 0
	for _, a := range d.actions {
		if !a.At.Before(loadedAt.Add(-14 * day)) {
			inWindow++
		}
	}
	if inWindow != len(want) {
		t.Errorf("008 seeds %d approval actions within the 14 days, the log has %d", inWindow, len(want))
	}
}

// The demo history comes out of the API as the Activity log and Home's Recent changes read it: the deleted
// initiative's entries point at the recycle bin, the summary counts the seeded entries (011 FR-002,
// FR-008; 009 FR-015). Priya reads every market's plans here (014: the log covers only what the reader may
// see), so the summary counts the whole seeded history.
func TestSeed_History_ThroughTheAPI_011(t *testing.T) {
	server := apitest.New(t)
	server.GrantRole(t, "priya.raman@shell.com", "viewer", "all")
	priya := server.SignIn(t, "priya.raman@shell.com")

	response := priya.Do(t, http.MethodGet, "/api/activity-log?recordId=I-1011&limit=10", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/activity-log = %d %s", response.Status, response.Body)
	}
	var log history.LogPage
	response.Decode(t, &log)
	if log.Total != 2 || len(log.Items) != 2 || log.Items[0].Action != history.ActionDeleted ||
		log.Items[0].RecordNow.State != history.RecordInRecycleBin || log.Items[0].RecordNow.BinEntryID != "BIN-1001" {
		t.Errorf("entries of I-1011 = %+v, want deleted and created, in the recycle bin BIN-1001", log)
	}

	week := loadedAt.Add(-7 * day).Format(time.RFC3339)
	today := time.Date(loadedAt.Year(), loadedAt.Month(), loadedAt.Day(), 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	response = priya.Do(t, http.MethodGet, "/api/activity-log/summary?todayStart="+today+"&weekStart="+week, nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/activity-log/summary = %d %s", response.Status, response.Body)
	}
	var summary history.LogSummary
	response.Decode(t, &summary)
	if summary.ChangesToday != 1 || summary.DeletionsThisWeek != 1 || summary.PeopleActiveThisWeek != 4 || summary.Total < 40 {
		t.Errorf("summary = %+v, want 1 change today, 1 deletion and 4 people active this week", summary)
	}

	response = priya.Do(t, http.MethodGet, "/api/home/recent-changes?scope=myTeams", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/home/recent-changes = %d %s", response.Status, response.Body)
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

// The deleted initiative's snapshots are records of features 005–007 as they store them: the restore preview
// works, and Restore brings the initiative back with its activity and line item, under Rimula fleet (011
// FR-018, FR-021).
func TestSeed_DeletedInitiative_Restores_011(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, "priya.raman@shell.com")

	response := priya.Do(t, http.MethodGet, "/api/recycle-bin/BIN-1001/restore-preview", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("restore preview = %d %s", response.Status, response.Body)
	}
	var preview history.RestorePreview
	response.Decode(t, &preview)
	if len(preview.Destination) != 2 || preview.Destination[1].Name != "Rimula fleet" || len(preview.ParentEntries) != 0 {
		t.Errorf("preview = %+v, want back into FY2026 Lubricants Europe › Rimula fleet", preview)
	}

	response = priya.Do(t, http.MethodPost, "/api/recycle-bin/BIN-1001/restore", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("restore = %d %s", response.Status, response.Body)
	}
	for path, want := range map[string]string{
		"/api/initiatives/I-1011": "Trade shows", "/api/activities/A-1013": "Fleet expo stand", "/api/line-items/L-1011": "Stand build",
	} {
		response := priya.Do(t, http.MethodGet, path, nil)
		var got struct {
			Name     string `json:"name"`
			LineItem struct {
				Name string `json:"name"`
			} `json:"lineItem"` // GET /api/line-items/{id} answers the line item with its activity
		}
		if response.Status == http.StatusOK {
			response.Decode(t, &got)
		}
		if response.Status != http.StatusOK || cmp.Or(got.Name, got.LineItem.Name) != want {
			t.Errorf("GET %s after the restore = %d %s, want %s", path, response.Status, response.Body, want)
		}
	}
}

// Whatever the seed time's seconds, every entry stays within the 14 days before it and in time order: the
// seed command runs at the computer's time, not on a whole minute (011 plan "Demo data").
func TestSeed_History_SeedTimeWithSeconds_011(t *testing.T) {
	at := time.Date(2026, 10, 20, 7, 41, 38, 512_000_000, time.UTC)
	db := openStore(t)
	if err := seed.Load(t.Context(), db, func() time.Time { return at }); err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, _ := demoLog(t, db)
	for i, e := range entries {
		if e.At.Before(at.Add(-seed.HistoryDays*day)) || e.At.After(at) {
			t.Errorf("%s is not within the 14 days before %s", entryText(e), at.Format(time.RFC3339Nano))
		}
		if i > 0 && e.At.Before(entries[i-1].At) {
			t.Errorf("%s comes after %s, which is later", entryText(e), entryText(entries[i-1]))
		}
	}
}

// Tests of the log's and the bin's rules can load the demo data without its history: the log and the bin
// start empty and the deleted initiative takes no IDs (seed.WithoutHistory).
func TestLoad_WithoutHistory_011(t *testing.T) {
	db := openStore(t)
	if err := seed.Load(t.Context(), db, fixedClock, seed.WithoutHistory()); err != nil {
		t.Fatalf("Load: %v", err)
	}
	entries, bin := demoLog(t, db)
	if len(entries) != 0 || len(bin) != 0 {
		t.Errorf("%d log entries and %d recycle-bin entries, want none", len(entries), len(bin))
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for prefix, want := range map[string]string{ids.PrefixInitiative: "I-1011", ids.PrefixActivity: "A-1013",
			ids.PrefixLineItem: "L-1011", ids.PrefixLogEntry: "LOG-1001", ids.PrefixBinEntry: "BIN-1001"} {
			if next, err := generator.Next(ctx, prefix); err != nil || next != want {
				t.Errorf("next %s ID = %s (%v), want %s", prefix, next, err, want)
			}
		}
		return errors.New("roll back") // leave the counters as the seed left them
	})
	if err == nil || err.Error() != "roll back" {
		t.Fatalf("Write: %v", err)
	}
}
