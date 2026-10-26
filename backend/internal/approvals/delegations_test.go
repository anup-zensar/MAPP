package approvals

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// day writes the fixture's today plus n days as a request sends it.
func (f *fixture) day(n int) string {
	return f.today().AddDays(n).String()
}

// createDelegation creates a delegation as a and stops the test when that fails.
func (f *fixture) createDelegation(t *testing.T, a actor.Actor, in DelegationInput) DelegationView {
	t.Helper()
	view, err := f.service.CreateDelegation(t.Context(), a, in)
	if err != nil {
		t.Fatalf("CreateDelegation: %v", err)
	}
	return view
}

// A person delegates their approvals to one active person who can approve, for at most 90 days from today
// or later, with an optional note; every problem is reported at once, and a second delegation on the same
// days is refused (017 FR-022, data-model §7). The delegation is logged.
func TestCreateDelegation_Rules_FR022(t *testing.T) {
	f := newFixture(t)
	f.access.cannotApprove[sara.ID] = true
	f.deactivatePerson(t, mei)

	created := f.createDelegation(t, daniel, DelegationInput{DelegateID: carlos.ID, StartDate: f.day(0), EndDate: f.day(2), Note: " Annual leave "})

	if created.ID != "DLG-1001" || created.Person.Name != "Daniel Okoro" || created.Delegate.Name != "Carlos Mendes" ||
		created.State != DelegationCurrent || !created.CanChange || created.Note != "Annual leave" || created.Version != 1 {
		t.Errorf("created = %+v, want a current delegation to Carlos Mendes", created)
	}
	assertLog(t, f, "DLG-1001", []string{"Daniel Okoro created approval delegation 'Daniel Okoro → Carlos Mendes'"})

	tests := []struct {
		name string
		in   DelegationInput
		want []errs.FieldError
	}{
		{"nothing chosen", DelegationInput{}, []errs.FieldError{
			{Field: "delegateId", Message: "Choose who approves for you."},
			{Field: "startDate", Message: "Enter the first day."},
			{Field: "endDate", Message: "Enter the last day."}}},
		{"to oneself, from yesterday, 91 days", DelegationInput{DelegateID: daniel.ID, StartDate: f.day(-1), EndDate: f.day(89)}, []errs.FieldError{
			{Field: "delegateId", Message: "You can't delegate to yourself."},
			{Field: "startDate", Message: "The first day must be today or later."},
			{Field: "endDate", Message: "A delegation can last at most 90 days."}}},
		{"cannot approve, last before first", DelegationInput{DelegateID: sara.ID, StartDate: f.day(10), EndDate: f.day(9), Note: strings.Repeat("x", 501)},
			[]errs.FieldError{
				{Field: "delegateId", Message: "Sara Novak can't approve activities. Choose someone with a role that approves."},
				{Field: "endDate", Message: "The last day must be on or after the first day."},
				{Field: "note", Message: "Use 500 characters or fewer."}}},
		{"inactive, unknown day", DelegationInput{DelegateID: mei.ID, StartDate: "next week", EndDate: f.day(3)}, []errs.FieldError{
			{Field: "delegateId", Message: "Mei Lin is inactive — choose an active person."},
			{Field: "startDate", Message: "Enter the first day."}}},
		{"overlap", DelegationInput{DelegateID: alex.ID, StartDate: f.day(2), EndDate: f.day(5)}, []errs.FieldError{
			{Field: "startDate", Message: "You already delegate to Carlos Mendes on some of these days. Change that delegation instead."}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.service.CreateDelegation(t.Context(), daniel, tt.in)
			assertError(t, err, errs.Validation(tt.want...))
		})
	}
	if _, err := f.service.CreateDelegation(t.Context(), daniel, DelegationInput{DelegateID: alex.ID, StartDate: f.day(3), EndDate: f.day(92)}); err != nil {
		t.Errorf("a delegation of 90 days right after the first: %v", err)
	}
}

// The delegations panel lists the person's delegations, as delegating person or delegate: current and
// planned by first day, then ended and cancelled, latest last day first; each with its state and whether
// the person may change it (017 FR-024).
func TestDelegation_States_FR024(t *testing.T) {
	f := newFixture(t)
	f.access.managers = []string{alex.ID}
	today := f.today()
	cancelledAt := start
	f.write(t, func(ctx context.Context) error {
		for _, dl := range []Delegation{
			{ID: "DLG-1001", PersonID: daniel.ID, DelegateID: carlos.ID, StartDate: today.AddDays(-30), EndDate: today.AddDays(-20)},
			{ID: "DLG-1002", PersonID: daniel.ID, DelegateID: carlos.ID, StartDate: today.AddDays(10), EndDate: today.AddDays(12)},
			{ID: "DLG-1003", PersonID: daniel.ID, DelegateID: alex.ID, StartDate: today.AddDays(-1), EndDate: today.AddDays(1)},
			{ID: "DLG-1004", PersonID: daniel.ID, DelegateID: alex.ID, StartDate: today.AddDays(20), EndDate: today.AddDays(22),
				CancelledAt: &cancelledAt, CancelledBy: &daniel},
			{ID: "DLG-1005", PersonID: carlos.ID, DelegateID: daniel.ID, StartDate: today.AddDays(5), EndDate: today.AddDays(6)},
			{ID: "DLG-1006", PersonID: carlos.ID, DelegateID: alex.ID, StartDate: today, EndDate: today},
		} {
			if err := f.service.repo.PutDelegation(ctx, dl); err != nil {
				return err
			}
		}
		return nil
	})

	page, err := f.service.ListDelegations(t.Context(), daniel, "")
	if err != nil {
		t.Fatalf("ListDelegations: %v", err)
	}
	var got []string
	for _, v := range page.Items {
		got = append(got, v.ID+" "+v.State+" "+map[bool]string{true: "change", false: "read-only"}[v.CanChange])
	}
	want := []string{"DLG-1003 current change", "DLG-1005 planned read-only", "DLG-1002 planned change",
		"DLG-1004 cancelled read-only", "DLG-1001 ended read-only"}
	if !slices.Equal(got, want) || page.Total != 5 {
		t.Errorf("Daniel's delegations = %q, want %q", got, want)
	}
	_, err = f.service.ListDelegations(t.Context(), daniel, "theirs")
	assertError(t, err, errs.Validation(errs.FieldError{Field: "scope", Message: "Choose mine or all."}))
}

// A planned or current delegation can be changed — a current one keeps its first day — and cancelled, by
// the delegating person, from the version they saw; ended and cancelled ones are kept read-only; changes
// and the cancel are logged (017 FR-024, FR-035).
func TestUpdateCancelDelegation_FR024(t *testing.T) {
	f := newFixture(t)
	planned := f.createDelegation(t, daniel, DelegationInput{DelegateID: carlos.ID, StartDate: f.day(7), EndDate: f.day(9)})
	current := f.createDelegation(t, daniel, DelegationInput{DelegateID: alex.ID, StartDate: f.day(0), EndDate: f.day(2)})
	end, note := f.day(11), "Annual leave"

	changed, err := f.service.UpdateDelegation(t.Context(), daniel, planned.ID, DelegationUpdate{EndDate: &end, Note: &note, Version: version(1)})
	if err != nil {
		t.Fatalf("UpdateDelegation: %v", err)
	}
	if changed.EndDate != f.today().AddDays(11) || changed.Note != "Annual leave" || changed.Version != 2 {
		t.Errorf("changed = %+v, want the new last day and note in version 2", changed)
	}
	var changes []string
	for _, c := range f.log(t, planned.ID)[0].Changes {
		changes = append(changes, c.Label+": "+c.OldText+" → "+c.NewText)
	}
	wantChanges := []string{"To: " + date.Format(f.today().AddDays(9)) + " → " + date.Format(f.today().AddDays(11)), "Note: Not filled in → Annual leave"}
	if !slices.Equal(changes, wantChanges) {
		t.Errorf("logged changes = %q, want %q", changes, wantChanges)
	}

	later := f.day(1)
	_, err = f.service.UpdateDelegation(t.Context(), daniel, current.ID, DelegationUpdate{StartDate: &later, Version: version(1)})
	assertError(t, err, errs.Validation(errs.FieldError{Field: "startDate", Message: "The delegation has started, so its first day can't change."}))
	f.access.managers = []string{alex.ID}
	_, err = f.service.UpdateDelegation(t.Context(), carlos, current.ID, DelegationUpdate{Note: &note, Version: version(1)})
	assertError(t, err, errs.Forbidden("Only Daniel Okoro or someone who can change approval workflows can change this delegation."))
	_, err = f.service.CancelDelegation(t.Context(), daniel, current.ID, DelegationCancel{Version: version(9)})
	if e, ok := err.(*errs.Error); !ok || e.Code != errs.CodeVersionConflict {
		t.Errorf("cancel from an old version = %v, want version_conflict", err)
	}

	f.clock.Advance(time.Minute)
	cancelled, err := f.service.CancelDelegation(t.Context(), daniel, current.ID, DelegationCancel{Version: version(1)})
	if err != nil {
		t.Fatalf("CancelDelegation: %v", err)
	}
	if cancelled.State != DelegationCancelled || cancelled.CanChange || cancelled.CancelledBy == nil || *cancelled.CancelledBy != daniel {
		t.Errorf("cancelled = %+v, want cancelled by Daniel Okoro, read-only", cancelled)
	}
	if line := logLine(f.log(t, current.ID)[0]); !strings.HasPrefix(line, "Daniel Okoro cancelled approval delegation 'Daniel Okoro → Alex Morgan'") {
		t.Errorf("log = %s", line)
	}
	_, err = f.service.CancelDelegation(t.Context(), daniel, current.ID, DelegationCancel{Version: version(2)})
	assertError(t, err, errs.NotAllowed("Ended and cancelled delegations can't be changed. They are kept for the audit trail."))
	_, err = f.service.CancelDelegation(t.Context(), daniel, "DLG-9999", DelegationCancel{Version: version(1)})
	assertError(t, err, errs.NotFound("This delegation no longer exists."))
}

// People who can change approval workflows manage everyone's delegations; others only their own (017
// FR-022, D-296): creating one for someone else and listing everyone's are refused with 014's message.
func TestDelegations_ForOthersNeedWorkflowsManage_FR022(t *testing.T) {
	f := newFixture(t)
	f.access.managers = []string{alex.ID}
	refusal := errs.Forbidden("Changing approval workflows isn't part of your role (Planner). Ask an administrator if you need it.")

	_, err := f.service.CreateDelegation(t.Context(), priya, DelegationInput{PersonID: daniel.ID, DelegateID: carlos.ID, StartDate: f.day(0), EndDate: f.day(1)})
	assertError(t, err, refusal)
	_, err = f.service.ListDelegations(t.Context(), priya, ScopeAll)
	assertError(t, err, refusal)

	forDaniel := f.createDelegation(t, alex, DelegationInput{PersonID: daniel.ID, DelegateID: carlos.ID, StartDate: f.day(0), EndDate: f.day(1)})
	_, err = f.service.CreateDelegation(t.Context(), alex, DelegationInput{PersonID: daniel.ID, DelegateID: daniel.ID, StartDate: f.day(1), EndDate: f.day(2)})
	assertError(t, err, errs.Validation(
		errs.FieldError{Field: "delegateId", Message: "Daniel Okoro can't delegate to themselves."},
		errs.FieldError{Field: "startDate", Message: "Daniel Okoro already delegates to Carlos Mendes on some of these days. Change that delegation instead."},
	))
	all, err := f.service.ListDelegations(t.Context(), alex, ScopeAll)
	if err != nil {
		t.Fatalf("ListDelegations: %v", err)
	}
	if !all.CanManageOthers || all.Total != 1 || !all.Items[0].CanChange || forDaniel.Person.Name != "Daniel Okoro" {
		t.Errorf("all = %+v, want Daniel's delegation, changeable by Alex", all)
	}
	if mine, _ := f.service.ListDelegations(t.Context(), priya, ScopeMine); mine.CanManageOthers || mine.Total != 0 {
		t.Errorf("Priya's = %+v, want none and no management", mine)
	}
}

// On each day of a current delegation the delegate may decide wherever the delegating person can as an
// approver, sees those activities in Waiting for me marked "as delegate of", and is counted in the summary;
// the delegating person keeps every right. A planned delegation, an inactive delegate, a delegate who
// cannot approve and a delegate's own delegate have no effect (017 FR-023).
func TestDelegation_Effect_FR023(t *testing.T) {
	f := newFixture(t)
	f.access.onBehalf = []string{alex.ID}
	f.addActivity(t, "A-1001", "Launch film", priya.ID, lubricantsFlow)
	f.submit(t, priya, "A-1001", "")
	f.delegate(t, "DLG-1001", daniel, carlos, f.today(), f.today().AddDays(1))
	f.delegate(t, "DLG-1002", daniel, mei, f.today().AddDays(3), f.today().AddDays(4))
	f.delegate(t, "DLG-1003", carlos, sara, f.today(), f.today().AddDays(1)) // no chains

	rows := f.inbox(t, carlos, TabWaiting)
	if len(rows) != 1 || rows[0].Activity.ID != "A-1001" || len(rows[0].AsDelegateOf) != 1 || rows[0].AsDelegateOf[0].Name != "Daniel Okoro" {
		t.Errorf("Carlos's Waiting for me = %+v, want A-1001 as delegate of Daniel Okoro", rows)
	}
	if counts, _ := f.service.Summary(t.Context(), carlos, nil, nil); counts.WaitingForMe != 1 {
		t.Errorf("Carlos's count = %d, want 1", counts.WaitingForMe)
	}
	for _, p := range []actor.Actor{mei, sara} {
		if ids := f.inboxIDs(t, p, TabWaiting); len(ids) != 0 {
			t.Errorf("%s's Waiting for me = %v, want nothing", p.Name, ids)
		}
	}
	if ids := f.inboxIDs(t, daniel, TabWaiting); !slices.Equal(ids, []string{"A-1001"}) {
		t.Errorf("Daniel's Waiting for me = %v, want he keeps A-1001", ids)
	}

	f.access.cannotApprove[carlos.ID] = true
	if ids := f.inboxIDs(t, carlos, TabWaiting); len(ids) != 0 {
		t.Errorf("a delegate who cannot approve still has %v", ids)
	}
	delete(f.access.cannotApprove, carlos.ID)
	f.deactivatePerson(t, carlos)
	if ids := f.inboxIDs(t, carlos, TabWaiting); len(ids) != 0 {
		t.Errorf("an inactive delegate still has %v", ids)
	}
	f.write(t, func(ctx context.Context) error {
		p, _, err := f.service.people.Get(ctx, carlos.ID)
		if err != nil {
			return err
		}
		p.Active = true
		return f.service.people.Put(ctx, p)
	})

	approval := f.approve(t, carlos, "A-1001", "")
	if d := approval.Rounds[0].Levels[0].Decision; approval.Status != StatusApproved || d == nil || d.Capacity != CapacityDelegate ||
		d.AsDelegateOf == nil || *d.AsDelegateOf != daniel {
		t.Errorf("decision = %+v, want Carlos Mendes as delegate of Daniel Okoro", d)
	}
	if !reflect.DeepEqual(f.inboxIDs(t, daniel, TabWaiting), []string{}) {
		t.Error("the activity is still waiting for Daniel")
	}
}
