package activities

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// keptArea is what the transaction memo keeps of activities.
const keptArea = "activities"

// PostgresStore is the Store on PostgreSQL. ListInScope and GetInScope keep only the activities the
// person acting may see (activity_in_scope: their plan's scope, their owner, their named approvers).
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Activity, bool, error) {
	return firstActivity(activitiesFromRows(s.records().ByID(ctx, []string{id})))
}

func (s *PostgresStore) List(ctx context.Context) ([]Activity, error) {
	return s.list(ctx, sqlcdb.ListActivitiesParams{})
}

func (s *PostgresStore) ListByInitiative(ctx context.Context, initiativeID string) ([]Activity, error) {
	return activitiesFromRows(s.records().ByParent(ctx, []string{initiativeID}))
}

func (s *PostgresStore) ListByInitiatives(ctx context.Context, initiativeIDs []string) ([]Activity, error) {
	return activitiesFromRows(s.records().ByParent(ctx, initiativeIDs))
}

func (s *PostgresStore) ListSplitFrom(ctx context.Context, activityID string) ([]Activity, error) {
	return s.list(ctx, sqlcdb.ListActivitiesParams{SplitFromID: &activityID})
}

func (s *PostgresStore) ListInScope(ctx context.Context, scope authz.Scope) ([]Activity, error) {
	return s.listInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetInScope(ctx context.Context, scope authz.Scope, id string) (Activity, bool, error) {
	return firstActivity(s.listInScope(ctx, scope, &id))
}

func (s *PostgresStore) Put(ctx context.Context, a Activity) error {
	params, err := activityParams(a)
	if err == nil {
		err = s.change(ctx).PutActivity(ctx, params)
	}
	if err != nil {
		return fmt.Errorf("save activity %s: %w", a.ID, err)
	}
	return nil
}

// AddMany stores new activities in one statement whatever their number.
func (s *PostgresStore) AddMany(ctx context.Context, list []Activity) error {
	if len(list) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddActivitiesParams, 0, len(list))
	for _, a := range list {
		params, err := activityParams(a)
		if err != nil {
			return err
		}
		rows = append(rows, sqlcdb.AddActivitiesParams(params))
	}
	if _, err := s.change(ctx).AddActivities(ctx, rows); err != nil {
		return fmt.Errorf("add activities: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteMany(ctx context.Context, ids []string) error {
	if err := s.change(ctx).DeleteActivities(ctx, ids); err != nil {
		return fmt.Errorf("delete activities: %w", err)
	}
	return nil
}

func activityParams(a Activity) (sqlcdb.PutActivityParams, error) {
	attributes, err := a.Attributes.Column()
	if err != nil {
		return sqlcdb.PutActivityParams{}, fmt.Errorf("encode activity %s: %w", a.ID, err)
	}
	costMinor, costCurrency := database.MoneyColumns(a.PlannedCost)
	revenueMinor, revenueCurrency := database.MoneyColumns(a.Revenue)
	return sqlcdb.PutActivityParams{
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
	}, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteActivity(ctx, id); err != nil {
		return fmt.Errorf("delete activity %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) list(ctx context.Context, filter sqlcdb.ListActivitiesParams) ([]Activity, error) {
	rows, err := s.db.Queries(ctx).ListActivities(ctx, filter)
	s.records().Keep(ctx, rows)
	return activitiesFromRows(rows, err)
}

func (s *PostgresStore) listInScope(ctx context.Context, scope authz.Scope, id *string) ([]Activity, error) {
	rows, err := s.db.Queries(ctx).ListActivitiesInScope(ctx, sqlcdb.ListActivitiesInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select activities in scope: %w", err)
	}
	s.records().Keep(ctx, rows)
	return activitiesFromRows(rows, nil)
}

// records keeps the activities a transaction reads, by id and by initiative (memo): pages read the activity
// of each line item they show. A change forgets them.
func (s *PostgresStore) records() memo.Records[sqlcdb.Activity] {
	return memo.Records[sqlcdb.Activity]{
		Area: keptArea, ID: func(r sqlcdb.Activity) string { return r.ID },
		Parent: func(r sqlcdb.Activity) string { return r.InitiativeID },
		Read: func(ctx context.Context, ids, initiativeIDs []string) ([]sqlcdb.Activity, error) {
			return s.db.Queries(ctx).ListActivities(ctx, sqlcdb.ListActivitiesParams{Ids: ids, InitiativeIds: initiativeIDs})
		},
	}
}

func (s *PostgresStore) change(ctx context.Context) *sqlcdb.Queries {
	memo.Forget(ctx, keptArea)
	return s.db.Queries(ctx)
}

func firstActivity(list []Activity, err error) (Activity, bool, error) {
	if err != nil || len(list) == 0 {
		return Activity{}, false, err
	}
	return list[0], true, nil
}

func activitiesFromRows(rows []sqlcdb.Activity, err error) ([]Activity, error) {
	if err != nil {
		return nil, fmt.Errorf("select activities: %w", err)
	}
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
			Attributes: attributes, INQ: clonePtr(r.Inq), SetLaterStagesMyself: r.SetLaterStagesMyself, MQL: clonePtr(r.Mql),
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
