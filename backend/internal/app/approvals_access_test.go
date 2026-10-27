package app

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"

	accessservice "mapp/internal/access"
	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
	"mapp/internal/stores"
	"mapp/seed"
)

// seededStore returns a store with the demo data, at 12 Mar 2026 09:00 UTC.
func seededStore(t *testing.T) (*store.DB, func() time.Time) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err := seed.Load(t.Context(), stores.JSON(db), now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	return db, now
}

// The facts approvals evaluates its rules on (017 FR-010, data-model §3): the activity's plan with its
// teams, owner and creator, its activity type id, and the plan's business units with their labels.
func TestAdapter_RuleFacts_FR010(t *testing.T) {
	t.Parallel()
	db, now := seededStore(t)
	generator := ids.NewGenerator(ids.NewJSONCounters())
	settingsService := settings.NewService(history.Tools{
		Tx: db, IDs: generator, Log: history.NewService(db, history.NewJSONStore(), generator, now), Now: now,
	}, settings.NewJSONStore())
	adapter := &approvalsActivities{
		activities: activity.NewJSONStore(), lineItems: lineitem.NewJSONStore(), campaigns: campaign.NewJSONStore(),
		plans: plan.NewJSONStore(), settings: settings.NewJSONStore(), rates: settingsService.Rates, now: now,
	}

	var fleet, launch approval.ActivityFacts
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if fleet, _, err = adapter.Activity(ctx, "A-1009"); err != nil {
			return err
		}
		launch, _, err = adapter.Activity(ctx, "A-1001")
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// FY2026 Mobility China has no business unit on purpose (seed/plans.go).
	if fleet.Plan.ID != "P-1002" || len(fleet.Plan.TeamIDs) == 0 || fleet.Plan.OwnerID != "U-1004" || fleet.Plan.CreatedByID != "system" ||
		fleet.ActivityTypeID == "" || fleet.BusinessUnits == nil || len(fleet.BusinessUnits) != 0 {
		t.Errorf("Fleet card launch = plan %+v, type %q, business units %+v; want FY2026 Mobility China's", fleet.Plan, fleet.ActivityTypeID, fleet.BusinessUnits)
	}
	if launch.Plan.ID != "P-1001" || len(launch.BusinessUnits) != 1 || launch.BusinessUnits[0].Label != "Lubricants" || !launch.BusinessUnits[0].Active {
		t.Errorf("Launch film = plan %+v, business units %+v; want FY2026 Lubricants Europe's", launch.Plan, launch.BusinessUnits)
	}
}

// 014's hooks as approvals reads them (017 FR-001, FR-025): the roles of 014's seed, who may approve, decide on
// behalf and manage workflows, and the holders of a role whose scope covers a plan — Daniel Okoro holds
// Approver for his own teams' plans only.
func TestApprovalsAccess_RoleHolders_FR001(t *testing.T) {
	t.Parallel()
	db, now := seededStore(t)
	generator := ids.NewGenerator(ids.NewJSONCounters())
	historyService := history.NewService(db, history.NewJSONStore(), generator, now)
	peopleService := people.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, people.NewJSONStore())
	access := approvalsAccess{access: accessservice.NewService(
		history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, accessservice.NewJSONStore(),
		accessDirectory{people: peopleService})}
	choices := approvalRuleChoices{settings: settings.NewService(history.Tools{Tx: db, IDs: generator, Log: historyService, Now: now}, settings.NewJSONStore())}

	err := db.Read(t.Context(), func(ctx context.Context) error {
		lubricants, found, err := plan.NewJSONStore().Get(ctx, "P-1001")
		if err != nil || !found {
			t.Fatalf("plan P-1001: %v %v", found, err)
		}
		holders, err := access.RoleHolders(ctx, "RL-1004", approval.PlanFacts{ID: lubricants.ID, TeamIDs: lubricants.TeamIDs})
		if err != nil {
			return err
		}
		elsewhere, err := access.RoleHolders(ctx, "RL-1004", approval.PlanFacts{ID: "P-9999", TeamIDs: []string{"T-9999"}})
		if err != nil {
			return err
		}
		admins, err := access.RoleHolders(ctx, "RL-1001", approval.PlanFacts{ID: "P-9999"})
		if err != nil {
			return err
		}
		if !slices.Equal(holders, []string{"U-1003"}) || len(elsewhere) != 0 || !slices.Equal(admins, []string{"U-1001"}) {
			t.Errorf("Approver holders = %v (elsewhere %v), Administrator %v", holders, elsewhere, admins)
		}

		can, err := access.CanApprove(ctx, []string{"U-1001", "U-1002", "U-1003", "U-1005", "U-1006"})
		if err != nil {
			return err
		}
		want := map[string]bool{"U-1001": true, "U-1002": false, "U-1003": true, "U-1005": true, "U-1006": false}
		if !reflect.DeepEqual(can, want) {
			t.Errorf("CanApprove = %v, want %v", can, want)
		}
		behalf, err := access.Check(ctx, "U-1005", approval.PermissionDecideOnBehalf, &approval.PlanFacts{ID: "P-1001"})
		if err != nil {
			return err
		}
		manage, err := access.Check(ctx, "U-1003", approval.PermissionManageWorkflows, nil)
		if err != nil {
			return err
		}
		if behalf.Allowed || behalf.Refusal != "Deciding on behalf of other approvers isn't part of your role (Global planner). Ask an administrator if you need it." ||
			manage.Allowed || manage.Refusal != "Changing approval workflows isn't part of your roles (Approver, Planner). Ask an administrator if you need it." {
			t.Errorf("checks = %+v / %+v", behalf, manage)
		}
		roles, err := access.Roles(ctx)
		if err != nil {
			return err
		}
		var approving []string
		for _, r := range roles {
			if r.Approves {
				approving = append(approving, r.Name)
			}
		}
		if !slices.Equal(approving, []string{"Administrator", "Approver", "Global planner"}) {
			t.Errorf("approving roles = %v", approving)
		}
		units, err := choices.BusinessUnits(ctx)
		if err != nil {
			return err
		}
		if len(units) != 2 || units[0].Label != "Mobility" {
			t.Errorf("business units = %+v, want Mobility and Lubricants", units)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
}
