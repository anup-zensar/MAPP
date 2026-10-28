package approval

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// inboxScoped is a's tab for a person who views the plans of the teams teamIDs only.
func (f *fixture) inboxScoped(t *testing.T, a actor.Actor, tab string, teamIDs ...string) []Row {
	t.Helper()
	roles := []authz.GrantedRole{{ID: "RL-1001", Name: "Viewer", Scope: authz.ScopeTeams, TeamIDs: teamIDs}}
	grants := authz.NewGrants(a.ID, roles, nil, map[string][]authz.Permission{"RL-1001": {authz.PlansView}}, nil, authz.Rules{})
	rows, err := f.service.Inbox(authz.With(t.Context(), grants), a, tab)
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	return rows
}

// rowKind is what the record-type tests compare of a row.
type rowKind struct {
	Subject     SubjectView
	HasActivity bool
	Cost        string
}

func kindOf(row Row) rowKind {
	got := rowKind{Subject: row.Subject, HasActivity: row.Activity != nil}
	if row.PlannedCost != nil {
		got.Cost = row.PlannedCost.Amount.String()
	}
	return got
}

func TestInboxCarriesKind(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-026 activity and purchase-request rows say what they are", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{workflow: prFlow, amount: "8000.00"},
			kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
		f.submit(t, priya, "A-1004", "")
		f.clock.Advance(time.Hour)
		f.submitPR(t, KindPurchaseRequest)

		var got []rowKind
		for _, row := range f.inbox(t, daniel, TabWaiting) {
			got = append(got, kindOf(row))
		}

		want := []rowKind{
			{Subject: SubjectView{Kind: KindActivity, Label: "Activity", ID: "A-1004", Name: "Trade magazine ads",
				Status: StatusSubmitted, DisplayStatus: StatusSubmitted}, HasActivity: true, Cost: "50000.00"},
			{Subject: SubjectView{Kind: KindPurchaseRequest, Label: "Purchase request", ID: purchaseRequestA,
				Name: "Agency fee " + purchaseRequestA, Status: StatusSubmitted, DisplayStatus: StatusSubmitted}, Cost: "8000.00"},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("rows (-want +got):\n%s", diff)
		}
	})
	t.Run("031 FR-026 purchase requests outside the person's scope are not listed", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)
		facts := prs.facts[purchaseRequestA]
		facts.Access.TeamIDs = []string{"T-9999"}
		prs.facts[purchaseRequestA] = facts

		outside, inside := f.inboxScoped(t, daniel, TabOpen, "T-1001"), f.inboxScoped(t, daniel, TabOpen, "T-9999")

		if len(outside) != 0 || len(inside) != 1 {
			t.Errorf("rows outside %d, inside %d; want 0 and 1", len(outside), len(inside))
		}
	})
}

func TestMonitoringKind(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-027 waiting by level says what the workflow approves", func(t *testing.T) {
		t.Parallel()
		f, _ := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)

		m, err := f.service.Monitoring(t.Context(), alex, nil, nil)

		if err != nil || len(m.ByLevel) != 1 || m.ByLevel[0].Kind != KindPurchaseRequest {
			t.Errorf("by level = %+v, %v; want the purchase-request workflow", m.ByLevel, err)
		}
	})
}

func TestNotificationSubject(t *testing.T) {
	t.Parallel()
	t.Run("031 FR-028 notifications name the purchase request and its type", func(t *testing.T) {
		t.Parallel()
		f, prs := newPurchaseFixture(t, purchaseRequest{workflow: prFlow}, kindFlow(prFlow, KindPurchaseRequest, named(daniel)))
		f.submitPR(t, KindPurchaseRequest)
		f.mustDecide(t, daniel, ActionApproved, "")

		page, err := f.service.ListNotifications(t.Context(), priya)
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("page = %+v, %v; want Priya's approval notice", page, err)
		}
		n := page.Items[0]
		want := NotificationSubject{Kind: KindPurchaseRequest, Label: "Purchase request", ID: purchaseRequestA,
			Name: "Agency fee " + purchaseRequestA, Exists: true}
		if n.Title != "Purchase request approved" || n.Subject != want || n.Activity != nil {
			t.Errorf("notification = %+v, want the purchase request approved", n)
		}
		delete(prs.facts, purchaseRequestA)
		if page, _ = f.service.ListNotifications(t.Context(), priya); page.Items[0].Subject.Exists {
			t.Errorf("subject of a deleted purchase request still exists: %+v", page.Items[0].Subject)
		}
	})
}
