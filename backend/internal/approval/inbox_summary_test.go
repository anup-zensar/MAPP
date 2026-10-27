package approval

import (
	"context"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// The summary counts equal the tabs' rows; Approved and Declined this month count the rounds that ended so
// this calendar month (FR-028, SC-007).
func TestSummary_MatchesTabs_FR028_SC007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004"} {
		f.addActivity(t, id, "Activity "+id, priya.ID, lubricantsFlow)
	}
	f.addActivity(t, "A-1005", "Fleet card launch", mei.ID, chinaFlow)
	for _, id := range []string{"A-1001", "A-1002", "A-1003", "A-1004", "A-1005"} {
		f.submit(t, priya, id, "")
	}
	f.approve(t, daniel, "A-1001", "")
	f.decline(t, daniel, "A-1002", "No.")
	f.submit(t, priya, "A-1002", "Again.")
	f.decline(t, daniel, "A-1002", "Still no.")

	for _, person := range []actor.Actor{alex, priya, daniel, carlos} {
		counts, err := f.service.Summary(t.Context(), person, nil, nil)
		if err != nil {
			t.Fatalf("Summary(%s): %v", person.Name, err)
		}
		want := Counts{
			WaitingForMe: len(f.inbox(t, person, TabWaiting)), AllOpen: len(f.inbox(t, person, TabOpen)),
			ApprovedThisMonth: 1, DeclinedThisMonth: 2,
		}
		if counts != want {
			t.Errorf("summary for %s = %+v, want %+v", person.Name, counts, want)
		}
	}
	if counts, _ := f.service.Summary(t.Context(), daniel, nil, nil); counts.WaitingForMe != 2 || counts.AllOpen != 3 {
		t.Errorf("Daniel's summary = %+v, want 2 waiting for him of 3 open", counts)
	}
}

// The month is the viewer's calendar month as two instants, end excluded; without them, the clock's UTC
// month. Only one bound, or an end not after the start, is refused (FR-028).
func TestSummary_MonthBounds_FR028(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Trade magazine ads", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet roadshow", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, priya, "A-1002", "")
	f.approve(t, daniel, "A-1001", "") // 12 Mar 2026 09:00 UTC
	f.clock.Advance(time.Date(2026, 3, 31, 22, 30, 0, 0, time.UTC).Sub(f.clock.Now()))
	f.decline(t, daniel, "A-1002", "No.") // 31 Mar 22:30 UTC: 1 April in Amsterdam (UTC+2)

	amsterdamMarch := [2]time.Time{time.Date(2026, 2, 28, 23, 0, 0, 0, time.UTC), time.Date(2026, 3, 31, 22, 0, 0, 0, time.UTC)}
	amsterdamApril := [2]time.Time{amsterdamMarch[1], time.Date(2026, 4, 30, 22, 0, 0, 0, time.UTC)}
	for _, c := range []struct {
		name  string
		month *[2]time.Time
		want  Counts
	}{
		{"UTC month of the clock", nil, Counts{ApprovedThisMonth: 1, DeclinedThisMonth: 1}},
		{"March in Amsterdam", &amsterdamMarch, Counts{ApprovedThisMonth: 1}},
		{"April in Amsterdam", &amsterdamApril, Counts{DeclinedThisMonth: 1}},
	} {
		var from, until *time.Time
		if c.month != nil {
			from, until = &c.month[0], &c.month[1]
		}
		counts, err := f.service.Summary(t.Context(), priya, from, until)
		if err != nil || counts != c.want {
			t.Errorf("%s: summary = %+v (%v), want %+v", c.name, counts, err, c.want)
		}
	}

	_, onlyStart := f.service.Summary(t.Context(), priya, &amsterdamMarch[0], nil)
	assertError(t, onlyStart, errs.Validation(errs.FieldError{Field: "monthStart", Message: "Reload the page and try again."}))
	_, backwards := f.service.Summary(t.Context(), priya, &amsterdamMarch[1], &amsterdamMarch[0])
	assertError(t, backwards, errs.Validation(errs.FieldError{Field: "monthEnd", Message: "Reload the page and try again."}))
}

// The summary's Waiting for me equals the Waiting for me tab, delegated activities included (017 FR-031,
// SC-007).
func TestSummary_MatchesWaitingWithDelegates_SC007(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.addActivity(t, "A-1002", "Fleet card launch", mei.ID, chinaFlow)
	f.submit(t, priya, "A-1001", "")
	f.submit(t, mei, "A-1002", "")
	f.write(t, func(ctx context.Context) error {
		return f.service.repo.PutDelegation(ctx, Delegation{ID: "DLG-1001", PersonID: daniel.ID, DelegateID: carlos.ID,
			StartDate: date.Today(start), EndDate: date.Today(start)})
	})

	for _, p := range []actor.Actor{carlos, daniel, alex} {
		counts, err := f.service.Summary(t.Context(), p, nil, nil)
		if err != nil {
			t.Fatalf("Summary: %v", err)
		}
		if rows := f.inbox(t, p, TabWaiting); counts.WaitingForMe != len(rows) {
			t.Errorf("%s: count %d, tab %d rows", p.Name, counts.WaitingForMe, len(rows))
		}
	}
	if counts, _ := f.service.Summary(t.Context(), carlos, nil, nil); counts.WaitingForMe != 2 {
		t.Errorf("Carlos waits for %d, want 2: his own level and Daniel's as delegate", counts.WaitingForMe)
	}
}
