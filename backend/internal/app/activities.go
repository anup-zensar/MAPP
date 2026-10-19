package app

import (
	"context"
	"encoding/json"
	"errors"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// newActivities builds the activities service (feature 006) on the repositories of the features it reads,
// and registers its history kind. Until features 002, 005 and 008 are merged, it reads Settings through
// activities.NewSettingsStandIn, the approval workflows through activities.NewSeededWorkflows, and hangs
// activities under a stand-in initiative kind; at integration, pass settingsService, register 005's kinds
// instead of initiativeStandIn, and hand the service to the plans and campaigns seams (its AddPlanFacts as
// the AddFacts of 004's "activity" level, its Summaries as 005's campaigns.Activities).
func newActivities(db *store.DB, generator *ids.Generator, hist *history.Service, now clock.Clock,
	peopleRepo people.Repository) (*activities.Service, error) {
	campaignsRepo := campaigns.NewJSONRepository()
	service := activities.NewService(db, activities.NewJSONRepository(), generator, hist, now, activities.Deps{
		Plans:     plans.NewJSONRepository(),
		Hierarchy: campaignsRepo,
		Settings:  activities.NewSettingsStandIn(settings.NewJSONRepository()),
		People:    peopleRepo,
		Workflows: activities.NewSeededWorkflows(approvals.NewJSONRepository()),
	})
	if err := hist.Register(initiativeStandIn(campaignsRepo)); err != nil {
		return nil, err
	}
	for _, kind := range service.Kinds() {
		if err := hist.Register(kind); err != nil {
			return nil, err
		}
	}
	return service, nil
}

// errStandIn refuses to delete or restore initiatives through the stand-in kind: only feature 005 does.
var errStandIn = errors.New("app: initiatives are deleted and restored by feature 005")

// initiativeStandIn is the history kind "initiative" until feature 005 registers its own: enough for the
// activity log and the recycle bin to find an activity's initiative (its path and parent). Remove it when
// 005's kinds are registered here.
func initiativeStandIn(repo campaigns.Repository) history.Kind {
	return history.Kind{
		Name: "initiative", Singular: "initiative", Plural: "initiatives",
		Snapshot: func(ctx context.Context, id string) (history.Snapshot, bool, error) {
			initiative, found, err := repo.GetInitiative(ctx, id)
			if err != nil || !found {
				return history.Snapshot{}, false, err
			}
			data, err := json.Marshal(initiative)
			return history.Snapshot{ID: initiative.ID, Name: initiative.Name, Data: data}, true, err
		},
		Remove: func(context.Context, string) error { return errStandIn },
		Put:    func(context.Context, history.Snapshot) error { return errStandIn },
	}
}
