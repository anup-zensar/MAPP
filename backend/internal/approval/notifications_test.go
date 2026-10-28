package approval

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

var everyone = []actor.Actor{alex, priya, daniel, mei, carlos, sara}

func (f *fixture) notifiedSince(t *testing.T, counts map[string]int) []string {
	t.Helper()
	var lines []string
	for _, p := range everyone {
		mine := f.notifications(t, p.ID)
		fresh := mine[:len(mine)-counts[p.ID]]
		if len(fresh) == 0 {
			continue
		}
		var kinds []string
		for _, n := range slices.Backward(fresh) {
			kinds = append(kinds, string(n.Kind))
		}
		lines = append(lines, p.Name+": "+strings.Join(kinds, ", "))
	}
	return lines
}

func (f *fixture) counts(t *testing.T) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, p := range everyone {
		counts[p.ID] = len(f.notifications(t, p.ID))
	}
	return counts
}

func TestNotify_Recipients_FR035(t *testing.T) {
	t.Parallel()
	const twoLevels = "WF-1003"
	tests := []struct {
		name  string
		setUp func(t *testing.T, f *fixture)
		act   func(t *testing.T, f *fixture)
		want  []string
	}{
		{"submit", func(*testing.T, *fixture) {}, func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") },
			[]string{"Daniel Okoro: approvalNeeded"}},
		{"re-submit", func(t *testing.T, f *fixture) {
			t.Helper()
			f.submit(t, alex, "A-1004", "")
			f.decline(t, daniel, "A-1004", "No.")
		}, func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") },
			[]string{"Daniel Okoro: approvalNeeded"}},
		{"approve a level before the last", func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") },
			func(t *testing.T, f *fixture) { t.Helper(); f.approve(t, daniel, "A-1004", "") },
			[]string{"Priya Raman: levelApproved", "Carlos Mendes: approvalNeeded", "Sara Novak: approvalNeeded"}},
		{"approve the last level", func(t *testing.T, f *fixture) {
			t.Helper()
			f.submit(t, alex, "A-1004", "")
			f.approve(t, daniel, "A-1004", "")
		}, func(t *testing.T, f *fixture) { t.Helper(); f.approve(t, carlos, "A-1004", "") },
			[]string{"Priya Raman: approved"}},
		{"decline", func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") },
			func(t *testing.T, f *fixture) { t.Helper(); f.decline(t, daniel, "A-1004", "No.") },
			[]string{"Priya Raman: declined"}},
		{"recall", func(t *testing.T, f *fixture) {
			t.Helper()
			f.submit(t, alex, "A-1004", "")
			f.approve(t, daniel, "A-1004", "")
		}, func(t *testing.T, f *fixture) { t.Helper(); f.recall(t, alex, "A-1004", "") },
			[]string{"Priya Raman: recalled", "Carlos Mendes: recalled", "Sara Novak: recalled"}},
		{"restart of a Submitted activity", func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") },
			func(t *testing.T, f *fixture) { t.Helper(); f.restart(t, alex, "A-1004") },
			[]string{"Priya Raman: restarted", "Daniel Okoro: restarted"}},
		{"restart of an Approved activity", func(t *testing.T, f *fixture) {
			t.Helper()
			f.submit(t, alex, "A-1004", "")
			f.approve(t, daniel, "A-1004", "")
			f.approve(t, sara, "A-1004", "")
		}, func(t *testing.T, f *fixture) { t.Helper(); f.restart(t, alex, "A-1004") },
			[]string{"Priya Raman: restarted", "Daniel Okoro: restarted", "Sara Novak: restarted"}},
		{"restore from the recycle bin", func(t *testing.T, f *fixture) { t.Helper(); f.submit(t, alex, "A-1004", "") }, func(t *testing.T, f *fixture) {
			t.Helper()
			f.write(t, func(ctx context.Context) error {
				_, err := f.service.ActivityRestored(ctx, history.RestoreContext{Actor: alex, Now: start, DeletedAt: start, DeletedBy: alex}, "A-1004")
				return err
			})
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.addWorkflow(t, twoLevels, "Two levels", []string{daniel.ID}, []string{carlos.ID, sara.ID})
			f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, twoLevels)
			tt.setUp(t, f)
			counts := f.counts(t)

			tt.act(t, f)

			if got := f.notifiedSince(t, counts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("notified =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestNotify_NeverOwnAction_FR036(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.access.rule = false
	f.addActivity(t, "A-1004", "Trade magazine ads", daniel.ID, lubricantsFlow)
	f.addActivity(t, "A-1005", "Fleet roadshow", daniel.ID, lubricantsFlow)

	f.submit(t, daniel, "A-1004", "")
	f.approve(t, daniel, "A-1004", "")
	f.restart(t, daniel, "A-1004")
	f.submit(t, daniel, "A-1005", "")
	f.recall(t, daniel, "A-1005", "")

	for _, p := range everyone {
		if got := f.notifications(t, p.ID); len(got) != 0 {
			t.Errorf("%s was notified: %+v", p.Name, got)
		}
	}
}

func TestNotify_DeclineQuotesComment_FR035(t *testing.T) {
	t.Parallel()
	short := "Split the agency fee by country."
	long := strings.Repeat("é", 79) + "ab"
	for _, tt := range []struct{ comment, quoted string }{
		{short, short},
		{strings.Repeat("x", 80), strings.Repeat("x", 80)},
		{long, strings.Repeat("é", 79) + "a…"},
	} {
		f := newFixture(t)
		f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
		f.submit(t, priya, "A-1004", "")

		f.decline(t, daniel, "A-1004", tt.comment)

		want := fmt.Sprintf("Daniel Okoro declined 'Trade magazine ads' (A-1004): %q", tt.quoted)
		if got := f.notifications(t, priya.ID); len(got) != 1 || got[0].Title != "Activity declined" || got[0].Text != want {
			t.Errorf("Priya's notifications = %+v, want %s", got, want)
		}
	}
}

func (f *fixture) putNotifications(t *testing.T, recipient actor.Actor, n, read int) []string {
	t.Helper()
	var list []string
	f.write(t, func(ctx context.Context) error {
		for i := range n {
			id, err := f.service.ids.Next(ctx, ids.PrefixNotification)
			if err != nil {
				return err
			}
			at := start.Add(time.Duration(i) * time.Minute)
			notification := Notification{
				ID: id, RecipientID: recipient.ID, Kind: NotifyApprovalNeeded, Title: "Approval needed",
				Text: "Priya Raman submitted 'Trade magazine ads' (A-1004)", SubjectKind: KindActivity, SubjectID: "A-1004", SubjectName: "Trade magazine ads",
				Actor: priya, At: at,
			}
			if i < read {
				notification.Read, notification.ReadAt = true, &at
			}
			if err := f.service.repo.PutNotification(ctx, notification); err != nil {
				return err
			}
			list = append(list, id)
		}
		return nil
	})
	return list
}

func TestListNotifications_FR033(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	all := f.putNotifications(t, daniel, 55, 3)
	f.putNotifications(t, carlos, 2, 0)

	page, err := f.service.ListNotifications(t.Context(), daniel)

	if err != nil {
		t.Fatalf("ListNotifications: %v", err)
	}
	if len(page.Items) != 50 || page.Total != 55 || page.Unread != 52 {
		t.Fatalf("page = %d items, total %d, unread %d; want 50, 55, 52", len(page.Items), page.Total, page.Unread)
	}
	first, last := page.Items[0], page.Items[49]
	if first.ID != all[54] || last.ID != all[5] || !first.At.Equal(start.Add(54*time.Minute)) {
		t.Errorf("items run from %s to %s, want the newest %s to %s", first.ID, last.ID, all[54], all[5])
	}
	wantFirst := NotificationView{
		ID: all[54], Kind: NotifyApprovalNeeded, Title: "Approval needed", Text: "Priya Raman submitted 'Trade magazine ads' (A-1004)",
		Subject: NotificationSubject{Kind: KindActivity, Label: "Activity", ID: "A-1004", Name: "Trade magazine ads", Exists: true}, Activity: &NotificationActivity{ID: "A-1004", Name: "Trade magazine ads", Exists: true}, Actor: priya,
		At: start.Add(54 * time.Minute),
	}
	if !reflect.DeepEqual(first, wantFirst) {
		t.Errorf("first item =\n%+v\nwant\n%+v", first, wantFirst)
	}
	none, err := f.service.ListNotifications(t.Context(), alex)
	if err != nil || none.Items == nil || len(none.Items) != 0 || none.Total != 0 || none.Unread != 0 {
		t.Errorf("Alex's page = %+v, %v; want an empty list", none, err)
	}
}

func TestListNotifications_DeletedActivity_EdgeCase(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	delete(f.activities.facts, "A-1004")

	page, err := f.service.ListNotifications(t.Context(), daniel)

	want := NotificationActivity{ID: "A-1004", Name: "Trade magazine ads", Exists: false}
	if err != nil || len(page.Items) != 1 || page.Items[0].Activity == nil || *page.Items[0].Activity != want || page.Items[0].Subject.Exists {
		t.Errorf("page = %+v, %v; want the notification with its activity gone", page, err)
	}
}

func TestMarkRead_FR034(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1004", "")
	id := f.notifications(t, daniel.ID)[0].ID
	entries := len(f.log(t, "A-1004"))
	f.clock.Advance(time.Hour)
	readAt := f.clock.Now()

	view, err := f.service.MarkRead(t.Context(), daniel, id)
	f.clock.Advance(time.Hour)
	again, againErr := f.service.MarkRead(t.Context(), daniel, id)
	_, notMine := f.service.MarkRead(t.Context(), priya, id)
	_, unknown := f.service.MarkRead(t.Context(), daniel, "NT-9999")

	if err != nil || againErr != nil || !view.Read || !again.Read || view.ID != id {
		t.Fatalf("MarkRead = %+v, %v; again %+v, %v", view, err, again, againErr)
	}
	stored := f.notifications(t, daniel.ID)[0]
	if !stored.Read || stored.ReadAt == nil || !stored.ReadAt.Equal(readAt) {
		t.Errorf("stored = %+v, want read at the first opening", stored)
	}
	gone := errs.NotFound("This notification no longer exists.")
	assertError(t, notMine, gone)
	assertError(t, unknown, gone)
	if got := len(f.log(t, "A-1004")); got != entries {
		t.Errorf("%d log entries, want still %d", got, entries)
	}
	page, _ := f.service.ListNotifications(t.Context(), daniel)
	if page.Unread != 0 {
		t.Errorf("unread = %d, want 0", page.Unread)
	}
}

func TestMarkAllRead_FR034(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1004", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.putNotifications(t, daniel, 55, 3)
	f.putNotifications(t, carlos, 2, 0)

	result, err := f.service.MarkAllRead(t.Context(), daniel)
	again, againErr := f.service.MarkAllRead(t.Context(), daniel)

	if err != nil || againErr != nil || result != (ReadAll{Marked: 52, Unread: 0}) || again != (ReadAll{Marked: 0, Unread: 0}) {
		t.Errorf("MarkAllRead = %+v, %v; again %+v, %v; want 52 marked, then 0", result, err, again, againErr)
	}
	page, _ := f.service.ListNotifications(t.Context(), daniel)
	if page.Unread != 0 || page.Total != 55 || len(page.Items) != 50 {
		t.Errorf("Daniel's page: %d items, total %d, unread %d; want all 55 kept, none unread", len(page.Items), page.Total, page.Unread)
	}
	if others, _ := f.service.ListNotifications(t.Context(), carlos); others.Unread != 2 {
		t.Errorf("Carlos's unread = %d, want still 2", others.Unread)
	}
}
