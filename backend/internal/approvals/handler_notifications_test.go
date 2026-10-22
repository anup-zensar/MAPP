package approvals_test

import (
	"net/http"
	"testing"

	"mapp/internal/approvals"
	"mapp/internal/platform/apitest"
)

// User story 6 — be told when something needs me, on the demo data: Daniel Okoro has one unread
// notification (the re-submitted Trade magazine ads), Priya Raman one unread and five read ones.

// Priya re-submits the fixed Fleet roadshow: Daniel's bell has one more unread, "Approval needed — Priya
// Raman re-submitted 'Fleet roadshow' (A-1005)". Daniel declines it: Priya is told with the start of his
// comment. Nobody is told about their own action (US6, FR-033, FR-035, FR-036).
func TestNotifications_SubmitThenDecline_US6(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	submission := decodeApproval(t, priya.Do(t, http.MethodGet, "/api/activities/"+fleetRoadshow+"/approval", nil))

	submitted := priya.Do(t, http.MethodPost, "/api/activities/"+fleetRoadshow+"/approval/submit", map[string]any{"version": submission.ActivityVersion})
	round := decodeApproval(t, submitted).Rounds[0]
	danielsBell := notifications(t, daniel)
	declined := daniel.Do(t, http.MethodPost, "/api/activities/"+fleetRoadshow+"/approval/decline",
		map[string]any{"roundId": round.ID, "level": 1, "comment": "Please move half of the cost to Q4."})
	decodeApproval(t, declined)
	priyasBell := notifications(t, priya)

	first := danielsBell.Items[0]
	if danielsBell.Unread != 2 || danielsBell.Total != 2 || first.Kind != approvals.NotifyApprovalNeeded || first.Title != "Approval needed" ||
		first.Text != "Priya Raman re-submitted 'Fleet roadshow' (A-1005)" || first.Read || !first.Activity.Exists ||
		first.Actor.Name != "Priya Raman" || !first.At.Equal(now) {
		t.Errorf("Daniel's bell: %d unread of %d, first %+v; want the new approval needed on top", danielsBell.Unread, danielsBell.Total, first)
	}
	newest := priyasBell.Items[0]
	if priyasBell.Unread != 2 || priyasBell.Total != 7 || newest.Kind != approvals.NotifyDeclined || newest.Title != "Activity declined" ||
		newest.Text != `Daniel Okoro declined 'Fleet roadshow' (A-1005): "Please move half of the cost to Q4."` {
		t.Errorf("Priya's bell: %d unread of %d, newest %+v; want the decline on top and nothing about her own submission",
			priyasBell.Unread, priyasBell.Total, newest)
	}
	if after := notifications(t, daniel); after.Total != 2 {
		t.Errorf("Daniel's bell after his decline holds %d, want still 2", after.Total)
	}
}

// Opening a notification marks it read, and the count drops; Mark all as read marks the rest and keeps them
// listed; a notification about a deleted activity says so (FR-034, edge case).
func TestMarkReadAndAll_FR034(t *testing.T) {
	server := apitest.New(t)
	priya := server.SignIn(t, priyasEmail)
	daniel := server.SignIn(t, danielsEmail)
	before := notifications(t, priya)
	unread := before.Items[0] // the decline of Fleet roadshow, her newest

	opened := priya.Do(t, http.MethodPost, "/api/notifications/"+unread.ID+"/read", nil)
	afterOpening := notifications(t, priya)
	deleteActivity(t, priya, tradeMagazineAds)
	all := daniel.Do(t, http.MethodPost, "/api/notifications/read-all", nil)
	danielsBell := notifications(t, daniel)

	if opened.Status != http.StatusOK {
		t.Fatalf("read = %d %s, want 200", opened.Status, opened.Body)
	}
	var view approvals.NotificationView
	opened.Decode(t, &view)
	if before.Unread != 1 || unread.Read || !view.Read || view.ID != unread.ID || afterOpening.Unread != 0 || afterOpening.Total != before.Total {
		t.Errorf("unread %d → %d of %d, opened %+v; want the one unread read, all kept", before.Unread, afterOpening.Unread, afterOpening.Total, view)
	}
	if all.Status != http.StatusOK {
		t.Fatalf("read-all = %d %s, want 200", all.Status, all.Body)
	}
	var result approvals.ReadAll
	all.Decode(t, &result)
	if result != (approvals.ReadAll{Marked: 1, Unread: 0}) || danielsBell.Unread != 0 || len(danielsBell.Items) != 1 {
		t.Errorf("read-all = %+v, bell %d unread of %d items; want 1 marked, the item kept", result, danielsBell.Unread, len(danielsBell.Items))
	}
	if item := danielsBell.Items[0]; item.Activity.ID != tradeMagazineAds || item.Activity.Exists || item.Activity.Name != "Trade magazine ads" {
		t.Errorf("item = %+v, want Trade magazine ads flagged as deleted", item.Activity)
	}
}

// A person can't mark another person's notification read: it is answered like one that does not exist,
// and it stays unread (FR-034).
func TestMarkRead_OtherPersons404_FR034(t *testing.T) {
	server := apitest.New(t)
	alex := server.SignIn(t, alexsEmail)
	daniel := server.SignIn(t, danielsEmail)
	danielsOwn := notifications(t, daniel).Items[0]

	response := alex.Do(t, http.MethodPost, "/api/notifications/"+danielsOwn.ID+"/read", nil)
	unknown := daniel.Do(t, http.MethodPost, "/api/notifications/NT-9999/read", nil)

	assertError(t, response, http.StatusNotFound, "not_found", "This notification no longer exists.")
	assertError(t, unknown, http.StatusNotFound, "not_found", "This notification no longer exists.")
	if after := notifications(t, daniel); after.Unread != 1 || after.Items[0].Read {
		t.Errorf("Daniel's bell = %d unread, first read %v; want it still unread", after.Unread, after.Items[0].Read)
	}
}

// notifications reads the bell of client through GET /api/notifications.
func notifications(t *testing.T, client *apitest.Client) approvals.NotificationPage {
	t.Helper()
	response := client.Do(t, http.MethodGet, "/api/notifications", nil)
	if response.Status != http.StatusOK {
		t.Fatalf("GET /api/notifications = %d %s, want 200", response.Status, response.Body)
	}
	var page approvals.NotificationPage
	response.Decode(t, &page)
	return page
}
