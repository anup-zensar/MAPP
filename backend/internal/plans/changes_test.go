package plans

import (
	"reflect"
	"slices"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// Cross-cutting rules.

// Every change of a plan writes exactly one activity-log entry naming who made it: create, edit, publish,
// unpublish, copy (on the new plan) and delete (FR-031).
func TestEveryChangeLogged_FR031(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	edit := updateOf(f.stored(t, plan.ID))
	edit.Description = "Lubricants marketing."
	steps := []func() error{
		func() error { _, err := f.service.Update(t.Context(), daniel, plan.ID, edit); return err },
		func() error { _, err := f.service.Publish(t.Context(), priya, plan.ID); return err },
		func() error {
			_, err := f.service.Unpublish(t.Context(), daniel, plan.ID, UnpublishRequest{Reason: "Review"})
			return err
		},
		func() error {
			_, err := f.service.Copy(t.Context(), daniel, plan.ID, CopyRequest{Name: "FY2027 Lubricants Test", Year: ptr(2027)})
			return err
		},
		func() error { _, err := f.service.Delete(t.Context(), priya, plan.ID); return err },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i+1, err)
		}
	}

	type logged struct {
		action history.Action
		verb   string
		actor  actor.Actor
	}
	var got []logged
	for _, e := range slices.Backward(f.entries(t)) {
		got = append(got, logged{e.Action, e.Verb, e.Actor})
	}
	want := []logged{
		{history.ActionCreated, "created", priya}, {history.ActionChanged, "changed", daniel},
		{history.ActionStatusChanged, "published", priya}, {history.ActionStatusChanged, "unpublished", daniel},
		{history.ActionCopied, "copied", daniel}, {history.ActionDeleted, "deleted", priya},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log, oldest first = %+v\nwant                %+v", got, want)
	}
}

// There are no role checks: anyone signed in may edit, publish, unpublish, copy and delete a plan someone else
// created, even someone in none of its teams (FR-032, D-004).
func TestNoRoleChecks_FR032(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	plan := f.create(t, priya, input("FY2026 Lubricants Test"))
	edit := updateOf(f.stored(t, plan.ID))
	edit.Description = "Edited by Nina."

	_, updateErr := f.service.Update(t.Context(), nina, plan.ID, edit)
	_, publishErr := f.service.Publish(t.Context(), nina, plan.ID)
	_, unpublishErr := f.service.Unpublish(t.Context(), nina, plan.ID, UnpublishRequest{Reason: "Review"})
	_, copyErr := f.service.Copy(t.Context(), nina, plan.ID, CopyRequest{Name: "Copied by Nina", Year: ptr(2026)})
	_, deleteErr := f.service.Delete(t.Context(), nina, plan.ID)

	for name, err := range map[string]error{"Update": updateErr, "Publish": publishErr, "Unpublish": unpublishErr,
		"Copy": copyErr, "Delete": deleteErr} {
		if err != nil {
			t.Errorf("%s by Nina: %v, want no refusal", name, err)
		}
	}
}
