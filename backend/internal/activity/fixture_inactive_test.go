package activity_test

import (
	"context"
	"testing"

	"mapp/internal/approval/approvaltest"
	"mapp/internal/people/peopletest"
	"mapp/internal/settings/settingstest"
)

func (f *fixture) deactivateType(t *testing.T, id string) {
	t.Helper()
	repo := settingstest.NewStore(f.db)
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
	repo := settingstest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		option, _, err := repo.GetOption(ctx, id)
		if err != nil {
			return err
		}
		option.Active = false
		return repo.PutOption(ctx, option)
	})
}

func (f *fixture) deactivateWorkflow(t *testing.T, id string) {
	t.Helper()
	repo := approvaltest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		workflow, _, err := repo.GetWorkflow(ctx, id)
		if err != nil {
			return err
		}
		workflow.Active = false
		return repo.PutWorkflow(ctx, workflow)
	})
}

func (f *fixture) deactivatePerson(t *testing.T, id string) {
	t.Helper()
	repo := peopletest.NewStore(f.db)
	f.change(t, func(ctx context.Context) error {
		person, _, err := repo.Get(ctx, id)
		if err != nil {
			return err
		}
		person.Active = false
		return repo.Put(ctx, person)
	})
}
