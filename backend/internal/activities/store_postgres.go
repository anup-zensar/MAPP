package activities

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// PostgresStore is the Repository on PostgreSQL. ListInScope and GetInScope keep only the activities the
// person acting may see (activity_in_scope: their plan's scope, their owner, their named approvers).
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Activity, bool, error) {
	return firstActivity(s.list(ctx, sqlcdb.ListActivitiesParams{ID: &id}))
}

func (s *PostgresStore) List(ctx context.Context) ([]Activity, error) {
	return s.list(ctx, sqlcdb.ListActivitiesParams{})
}

func (s *PostgresStore) ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error) {
	return s.list(ctx, sqlcdb.ListActivitiesParams{InitiativeIds: []string{initiativeID}})
}

func (s *PostgresStore) ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error) {
	return s.list(ctx, sqlcdb.ListActivitiesParams{InitiativeIds: append([]string{}, initiativeIDs...)})
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope authz.Scope) ([]Activity, error) {
	return s.listInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (Activity, bool, error) {
	return firstActivity(s.listInScope(ctx, scope, &id))
}

func (s *PostgresStore) Put(ctx context.Context, a Activity) error {
	attributes, err := a.Attributes.Column()
	if err != nil {
		return fmt.Errorf("encode activity %s: %w", a.ID, err)
	}
	costMinor, costCurrency := database.MoneyColumns(a.PlannedCost)
	revenueMinor, revenueCurrency := database.MoneyColumns(a.Revenue)
	err = s.db.Queries(ctx).PutActivity(ctx, sqlcdb.PutActivityParams{
		ID: a.ID, InitiativeID: a.InitiativeID, Name: a.Name, ExternalName: a.ExternalName,
		ActivityTypeID: a.ActivityTypeID, OwnerID: database.NullIfEmpty(a.OwnerID),
		StartDate: database.DateColumn(a.StartDate), EndDate: database.DateColumn(a.EndDate), Ongoing: a.Ongoing,
		PlannedCostMinor: costMinor, PlannedCostCurrency: costCurrency,
		ApprovalWorkflowID: database.NullIfEmpty(a.ApprovalWorkflowID), Description: a.Description,
		Attributes: attributes, Inq: a.INQ, SetLaterStagesMyself: a.SetLaterStagesMyself, Mql: a.MQL,
		RevenueMinor: revenueMinor, RevenueCurrency: revenueCurrency, Status: string(a.Status),
		SplitFromID: database.NullIfEmpty(a.SplitFromID), Version: a.Version, CreatedAt: a.CreatedAt,
		CreatedByID: a.CreatedBy.ID, CreatedByName: a.CreatedBy.Name, UpdatedAt: a.UpdatedAt,
		UpdatedByID: a.UpdatedBy.ID, UpdatedByName: a.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save activity %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteActivity(ctx, id); err != nil {
		return fmt.Errorf("delete activity %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) list(ctx context.Context, filter sqlcdb.ListActivitiesParams) ([]Activity, error) {
	rows, err := s.db.Queries(ctx).ListActivities(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select activities: %w", err)
	}
	return activitiesFromRows(rows)
}

func (s *PostgresStore) listInScope(ctx context.Context, scope authz.Scope, id *string) ([]Activity, error) {
	rows, err := s.db.Queries(ctx).ListActivitiesInScope(ctx, sqlcdb.ListActivitiesInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select activities in scope: %w", err)
	}
	return activitiesFromRows(rows)
}

func firstActivity(list []Activity, err error) (Activity, bool, error) {
	if err != nil || len(list) == 0 {
		return Activity{}, false, err
	}
	return list[0], true, nil
}

func activitiesFromRows(rows []sqlcdb.Activity) ([]Activity, error) {
	list := make([]Activity, 0, len(rows))
	for _, r := range rows {
		attributes, err := settings.AttributeValuesOf(r.Attributes)
		if err != nil {
			return nil, fmt.Errorf("decode activity %s: %w", r.ID, err)
		}
		list = append(list, Activity{
			ID: r.ID, InitiativeID: r.InitiativeID, Name: r.Name, ExternalName: r.ExternalName,
			ActivityTypeID: r.ActivityTypeID, OwnerID: database.EmptyIfNull(r.OwnerID),
			StartDate: database.DateOf(r.StartDate), EndDate: database.DateOf(r.EndDate), Ongoing: r.Ongoing,
			PlannedCost:        database.MoneyOf(r.PlannedCostMinor, r.PlannedCostCurrency),
			ApprovalWorkflowID: database.EmptyIfNull(r.ApprovalWorkflowID), Description: r.Description,
			Attributes: attributes, INQ: r.Inq, SetLaterStagesMyself: r.SetLaterStagesMyself, MQL: r.Mql,
			Revenue: database.MoneyOf(r.RevenueMinor, r.RevenueCurrency), Status: Status(r.Status),
			SplitFromID: database.EmptyIfNull(r.SplitFromID),
			Meta: record.Meta{
				Version:   r.Version,
				CreatedAt: r.CreatedAt,
				CreatedBy: actor.Actor{ID: r.CreatedByID, Name: r.CreatedByName},
				UpdatedAt: r.UpdatedAt,
				UpdatedBy: actor.Actor{ID: r.UpdatedByID, Name: r.UpdatedByName},
			},
		})
	}
	return list, nil
}
