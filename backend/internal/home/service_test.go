package home

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"mapp/internal/platform/race"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// A plan with two of the person's teams is covered once.
func TestPlansInScope_MyTeamsAndEverything_FR003_FR004(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{
		newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps),
		newPlan("P-1004", "FY2026 Joint plan", 2026, lubricantsEurope, mobilityChina),
	}
	s := newService(t, f)
	inScope := func(a actor.Actor, scope Scope) []string {
		ps, err := s.planScope(t.Context(), a)
		if err != nil {
			t.Fatalf("planScope: %v", err)
		}
		return planIDs(ps.in(scope))
	}

	if got := inScope(sara, ScopeMyTeams); !slices.Equal(got, []string{"P-1001", "P-1002", "P-1004"}) {
		t.Errorf("Sara, My teams = %q, want her two teams' plans, the joint plan once", got)
	}
	if got := inScope(priya, ScopeMyTeams); !slices.Equal(got, []string{"P-1001", "P-1004"}) {
		t.Errorf("Priya, My teams = %q", got)
	}
	if got := inScope(nina, ScopeMyTeams); len(got) != 0 {
		t.Errorf("Nina (no team), My teams = %q, want none", got)
	}
	if got := inScope(nina, ScopeEverything); !slices.Equal(got, []string{"P-1001", "P-1002", "P-1003", "P-1004"}) {
		t.Errorf("Everything = %q, want every plan", got)
	}
}

func TestScope_Invalid(t *testing.T) {
	t.Parallel()
	for _, text := range []string{"myTeams", "everything"} {
		if scope, err := ParseScope(text); err != nil || string(scope) != text {
			t.Errorf("ParseScope(%q) = %q, %v", text, scope, err)
		}
	}
	for _, text := range []string{"", "all", "MyTeams", "mine"} {
		_, err := ParseScope(text)

		assertError(t, err, scopeRefused)
	}
	s := newService(t, newFakeSources())
	_, err := s.Summary(t.Context(), priya, "all")
	assertError(t, err, scopeRefused)
}

func TestSummary_Counts_FR005(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	published := newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope)
	published.Status = "published"
	f.plans = []Plan{published, newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2027 Global Brand", 2027, globalMarketingOps)}
	plan := errs.RecordRef{Type: "plan", ID: "P-1001"} // the card counts by plan year
	f.activities[priya.ID] = []Activity{
		{ID: "A-1001", Plan: plan, Status: "created"}, {ID: "A-1004", Plan: plan, Status: "submitted"},
		{ID: "A-1005", Plan: plan, Status: "declined"}, {ID: "A-1002", Plan: plan, Status: "approved", DisplayStatus: "inProgress"},
	}
	f.waiting[priya.ID] = []Waiting{
		{ActivityID: "A-1009", WaitingSince: now.Add(-2 * time.Hour)}, // submitted first, but reached its level later
		{ActivityID: "A-1004", WaitingSince: now.Add(-50 * time.Hour)},
	}
	s := newService(t, f)

	mine, err := s.Summary(t.Context(), priya, "myTeams")
	all, err2 := s.Summary(t.Context(), priya, "everything")

	if err != nil || err2 != nil {
		t.Fatalf("Summary: %v, %v", err, err2)
	}
	if mine.Scope != ScopeMyTeams || mine.Plans != (PlanCounts{Count: 1, Published: 1, Draft: 0}) {
		t.Errorf("My teams: scope %q, plans %+v; want 1 published", mine.Scope, mine.Plans)
	}
	if all.Scope != ScopeEverything || all.Plans != (PlanCounts{Count: 3, Published: 1, Draft: 2}) {
		t.Errorf("Everything: scope %q, plans %+v; want 1 published, 2 draft", all.Scope, all.Plans)
	}
	if c := countsText(mine.ActivitiesOwned); c != "4 (1 declined, 1 submitted) in 2026–2026" {
		t.Errorf("activities owned = %s, want 4 with 1 declined and 1 submitted", c)
	}
	if w := mine.WaitingForMe; w.Count != 2 || w.OldestWaitingSince == nil || !w.OldestWaitingSince.Equal(now.Add(-50*time.Hour)) {
		t.Errorf("waiting for me = %+v, want 2, the oldest waiting for 50 hours", w)
	}
	empty, _ := s.Summary(t.Context(), daniel, "myTeams")
	if empty.WaitingForMe != (WaitingCounts{}) || empty.ActivitiesOwned != (ActivityCounts{}) {
		t.Errorf("Daniel: waiting %+v, owned %+v; want nothing", empty.WaitingForMe, empty.ActivitiesOwned)
	}
}

// Waiting for my approval is the Approvals page's Waiting for me tab, oldest first: the first 5, and the total.
func TestWaiting_SameListOldestFirst_FR013(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	for i := range 7 {
		f.waiting[daniel.ID] = append(f.waiting[daniel.ID], Waiting{
			ActivityID: fmt.Sprintf("A-10%02d", i), ActivityName: fmt.Sprintf("Activity %d", i), Plan: lubricantsRef,
			Owner: &priya, PlannedCost: usdPtr(t, "1000.00"), Level: 1, Levels: 1,
			WaitingSince: now.Add(-time.Duration(48-i) * time.Hour),
		})
	}
	f.waiting[daniel.ID][1].PlannedCost, f.waiting[daniel.ID][1].Owner = nil, nil
	f.currencies[daniel.ID] = Currency{Code: "EUR", Rate: rate(t, "0.92")}
	s := newService(t, f)

	page, err := s.Waiting(t.Context(), daniel)
	nothing, err2 := s.Waiting(t.Context(), priya)

	if err != nil || page.Total != 7 || len(page.Items) != 5 {
		t.Fatalf("Waiting = %d items of %d, %v; want 5 of 7", len(page.Items), page.Total, err)
	}
	want := WaitingItem{ID: "A-1000", Name: "Activity 0", Plan: lubricantsRef, Owner: &priya,
		PlannedCost: &money.Money{Amount: usd(t, "920.00"), Currency: "EUR"}, Level: 1, Levels: 1, WaitingSince: now.Add(-48 * time.Hour)}
	if !reflect.DeepEqual(page.Items[0], want) {
		t.Errorf("first = %+v\nwant %+v", page.Items[0], want)
	}
	if second := page.Items[1]; second.ID != "A-1001" || second.Owner != nil || second.PlannedCost != nil {
		t.Errorf("second = %+v, want no owner and no planned cost", second)
	}
	if err2 != nil || nothing.Total != 0 || nothing.Items == nil || len(nothing.Items) != 0 {
		t.Errorf("Priya = %+v, %v; want an empty list", nothing, err2)
	}
}

// Every Home operation is one pass over its inputs: 50 plans and 5,000 activities answer within 300 ms.
func TestHome_LargeScope_SC004(t *testing.T) {
	if testing.Short() || race.Enabled {
		t.Skip("a timing test: it runs without -short and without the race detector, in CI's Timing tests step")
	}
	f := newFakeSources()
	for i := range 50 {
		p := newPlan(fmt.Sprintf("P-%d", 1001+i), fmt.Sprintf("Plan %02d", i), 2026, lubricantsEurope, mobilityChina)
		p.OwnerID, p.Budget, p.PlannedCost, p.RemainingBudget = sara.ID, usdPtr(t, "1000.00"), usdPtr(t, "1500.00"), usdPtr(t, "-1.00")
		p.ActivityCount, p.ItemsBelow = 100, 100
		f.plans = append(f.plans, p)
		f.containers[sara.ID] = append(f.containers[sara.ID], Container{Type: "campaign", ID: fmt.Sprintf("C-%d", 1001+i),
			Name: "Campaign", Plan: lubricantsRef, Path: []errs.RecordRef{lubricantsRef}, Budget: usdPtr(t, "1.00"),
			PlannedCost: usdPtr(t, "2.00"), UpdatedAt: now})
	}
	statuses := []string{"created", "submitted", "declined", "approved"}
	for i := range 5000 {
		act := Activity{ID: fmt.Sprintf("A-%d", 10001+i), Name: fmt.Sprintf("Activity %04d", i%997), Plan: lubricantsRef,
			Status: statuses[i%4], DisplayStatus: statuses[i%4], PlannedCost: usdPtr(t, "10.00"), Unassigned: usdPtr(t, "-1.00"),
			Missing: []MissingField{{Field: "ownerId", Label: "Owner"}}, UpdatedAt: now.Add(-time.Duration(i) * time.Minute)}
		if i%3 == 0 {
			act.StartDate = ptr(date.MustParse("2026-03-01").AddDays(i % 300))
		}
		f.activities[sara.ID] = append(f.activities[sara.ID], act)
		if i < 200 {
			f.waiting[sara.ID] = append(f.waiting[sara.ID], Waiting{ActivityID: act.ID, Level: 1, Levels: 1, WaitingSince: now})
		}
	}
	s := newService(t, f)

	operations := map[string]func() error{
		"summary":       func() error { _, err := s.Summary(t.Context(), sara, "myTeams"); return err },
		"plans":         func() error { _, err := s.PlansAtAGlance(t.Context(), sara, "everything"); return err },
		"attention":     func() error { _, err := s.Attention(t.Context(), sara); return err },
		"waiting":       func() error { _, err := s.Waiting(t.Context(), sara); return err },
		"my activities": func() error { _, err := s.MyActivities(t.Context(), sara); return err },
	}
	for name, operation := range operations {
		started := time.Now()
		err := operation()
		if took := time.Since(started); err != nil || took > 300*time.Millisecond {
			t.Errorf("%s took %v, %v; want under 300 ms", name, took, err)
		}
	}
}

// My teams without team plans answers an empty page without asking the activity log.
func TestRecentChanges_Scope_FR015(t *testing.T) {
	t.Parallel()
	f := newFakeSources()
	f.plans = []Plan{
		newPlan("P-1001", "FY2026 Lubricants Europe", 2026, lubricantsEurope),
		newPlan("P-1002", "FY2026 Mobility China", 2026, mobilityChina),
		newPlan("P-1003", "FY2026 Global brand", 2026, lubricantsEurope, mobilityChina),
	}
	entry := map[string]any{"id": "LOG-1001"}
	f.logAnswer = RecentChangesPage{Items: []any{entry}, Total: 12}
	service := newService(t, f)

	mine, err := service.RecentChanges(t.Context(), priya, "myTeams")
	if err != nil || mine.Total != 12 || len(mine.Items) != 1 {
		t.Fatalf("My teams = %+v, %v; want the answer of the log", mine, err)
	}
	if _, err := service.RecentChanges(t.Context(), sara, "everything"); err != nil {
		t.Fatalf("Everything: %v", err)
	}
	if want := [][]string{{"P-1001", "P-1003"}, nil}; !reflect.DeepEqual(f.logAsked, want) || f.logLimit != 10 {
		t.Errorf("asked the log for %v (limit %d), want %v (limit 10)", f.logAsked, f.logLimit, want)
	}

	none, err := service.RecentChanges(t.Context(), nina, "myTeams") // in no team
	if err != nil || none.Total != 0 || none.Items == nil || len(none.Items) != 0 {
		t.Errorf("My teams without team plans = %+v, %v; want an empty page", none, err)
	}
	if len(f.logAsked) != 2 {
		t.Errorf("the log was asked %d times, want 2 (not for a person without team plans)", len(f.logAsked))
	}

	_, err = service.RecentChanges(t.Context(), priya, "all")
	assertError(t, err, scopeRefused)
}
