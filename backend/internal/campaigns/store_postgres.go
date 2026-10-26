package campaigns

import (
	"context"
	"fmt"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
	"mapp/internal/rollup"
	"mapp/internal/settings"
)

// PostgresStore is the Repository on PostgreSQL; initiatives are in store_postgres_initiatives.go. The
// ...InScope reads keep only the records of the plans the person acting may see.
type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) GetCampaign(ctx context.Context, id string) (Campaign, bool, error) {
	return first(s.campaigns(ctx, sqlcdb.ListCampaignsParams{ID: &id}))
}

func (s *PostgresStore) ListCampaigns(ctx context.Context) ([]Campaign, error) {
	return s.campaigns(ctx, sqlcdb.ListCampaignsParams{})
}

func (s *PostgresStore) CampaignsOfPlan(ctx context.Context, planID string) ([]Campaign, error) {
	return s.campaigns(ctx, sqlcdb.ListCampaignsParams{PlanIds: []string{planID}})
}

func (s *PostgresStore) CampaignsOfPlans(ctx context.Context, planIDs []string) ([]Campaign, error) {
	return s.campaigns(ctx, sqlcdb.ListCampaignsParams{PlanIds: append([]string{}, planIDs...)})
}

func (s *PostgresStore) CampaignsInScope(ctx context.Context, scope authz.Scope) ([]Campaign, error) {
	return s.campaignsInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetCampaignInScope(ctx context.Context, scope authz.Scope, id string) (Campaign, bool, error) {
	return first(s.campaignsInScope(ctx, scope, &id))
}

func (s *PostgresStore) PutCampaign(ctx context.Context, c Campaign) error {
	attributes, err := c.Attributes.Column()
	if err != nil {
		return fmt.Errorf("encode campaign %s: %w", c.ID, err)
	}
	minor, currency := database.MoneyColumns(c.Budget)
	phasing := c.BudgetPhasing.Normalised()
	months, amounts := phasing.MonthColumns()
	q := s.db.Queries(ctx)
	err = q.PutCampaign(ctx, sqlcdb.PutCampaignParams{
		ID: c.ID, PlanID: c.PlanID, Name: c.Name, Description: c.Description, OwnerID: ownerColumn(c.OwnerID),
		StartDate: database.DateColumn(c.StartDate), EndDate: database.DateColumn(c.EndDate), BudgetMinor: minor,
		BudgetCurrency: currency, Attributes: attributes, PhasingMethod: string(phasing.Method), Version: c.Version,
		CreatedAt: c.CreatedAt, CreatedByID: c.CreatedBy.ID, CreatedByName: c.CreatedBy.Name, UpdatedAt: c.UpdatedAt,
		UpdatedByID: c.UpdatedBy.ID, UpdatedByName: c.UpdatedBy.Name,
	})
	if err == nil {
		err = q.DeleteCampaignPhasingMonths(ctx, c.ID)
	}
	if err == nil {
		err = q.InsertCampaignPhasingMonths(ctx, sqlcdb.InsertCampaignPhasingMonthsParams{CampaignID: c.ID, Months: months, Amounts: amounts})
	}
	if err != nil {
		return fmt.Errorf("save campaign %s: %w", c.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteCampaign(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeleteCampaign(ctx, id); err != nil {
		return fmt.Errorf("delete campaign %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) campaigns(ctx context.Context, filter sqlcdb.ListCampaignsParams) ([]Campaign, error) {
	rows, err := s.db.Queries(ctx).ListCampaigns(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("select campaigns: %w", err)
	}
	return campaignsFromRows(rows)
}

func (s *PostgresStore) campaignsInScope(ctx context.Context, scope authz.Scope, id *string) ([]Campaign, error) {
	rows, err := s.db.Queries(ctx).ListCampaignsInScope(ctx, sqlcdb.ListCampaignsInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.ViewsPlans,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select campaigns in scope: %w", err)
	}
	listRows := make([]sqlcdb.ListCampaignsRow, 0, len(rows))
	for _, row := range rows {
		listRows = append(listRows, sqlcdb.ListCampaignsRow(row))
	}
	return campaignsFromRows(listRows)
}

func campaignsFromRows(rows []sqlcdb.ListCampaignsRow) ([]Campaign, error) {
	list := make([]Campaign, 0, len(rows))
	for _, row := range rows {
		r := row.Campaign
		attributes, err := settings.AttributeValuesOf(r.Attributes)
		if err != nil {
			return nil, fmt.Errorf("decode campaign %s: %w", r.ID, err)
		}
		list = append(list, Campaign{
			ID: r.ID, PlanID: r.PlanID, Name: r.Name, Description: r.Description, OwnerID: r.OwnerID,
			StartDate: database.DateOf(r.StartDate), EndDate: database.DateOf(r.EndDate),
			Budget: database.MoneyOf(r.BudgetMinor, r.BudgetCurrency), Attributes: attributes,
			BudgetPhasing: rollup.PhasingOf(r.PhasingMethod, row.PhasingMonths, row.PhasingAmounts).Normalised(),
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

// ownerColumn treats an owner of "" as no owner, so the column's foreign key holds.
func ownerColumn(ownerID *string) *string {
	if ownerID == nil {
		return nil
	}
	return database.NullIfEmpty(*ownerID)
}

func first[T any](list []T, err error) (T, bool, error) {
	var zero T
	if err != nil || len(list) == 0 {
		return zero, false, err
	}
	return list[0], true, nil
}
