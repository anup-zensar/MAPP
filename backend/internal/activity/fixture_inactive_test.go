package activity_test

import (
	"context"
	"testing"

	"mapp/internal/approval"
	"mapp/internal/people"
	"mapp/internal/settings"
)

// deactivateType makes the activity type id inactive.
func (f *fixture) deactivateType(t *testing.T, id string) {
	t.Helper()
	repo := settings.NewJSONStore()
	f.change(t, func(ctx context.Context) error {
		activityType, _, err := repo.GetActivityType(ctx, id)
		if err != nil {
			return err
		}
		activityType.Active = false
		return repo.PutActivityType(ctx, activityType)
	})
}

// deactivateOption makes the Shell attribute option id inactive.
func (f *fixture) deactivateOption(t *testing.T, id string) {
	t.Helper()
	repo := settings.NewJSONStore()
	f.change(t, func(ctx context.Context) error {
		option, _, err := repo.GetOption(ctx, id)
		if err != nil {
			return err
		}
		option.Active = false
		return repo.PutOption(ctx, option)
	})
}

// deactivateWorkflow makes the approval workflow id inactive.
func (f *fixture) deactivateWorkflow(t *testing.T, id string) {
	t.Helper()
	repo := approval.NewJSONStore()
	f.change(t, func(ctx context.Context) error {
		workflow, _, err := repo.GetWorkflow(ctx, id)
		if err != nil {
			return err
		}
		workflow.Active = false
		return repo.PutWorkflow(ctx, workflow)
	})
}

// deactivatePerson makes the person id inactive.
func (f *fixture) deactivatePerson(t *testing.T, id string) {
	t.Helper()
	repo := people.NewJSONStore()
	f.change(t, func(ctx context.Context) error {
		person, _, err := repo.Get(ctx, id)
		if err != nil {
			return err
		}
		person.Active = false
		return repo.Put(ctx, person)
	})
}
