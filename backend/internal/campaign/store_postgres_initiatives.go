package campaign

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

func (s *PostgresStore) GetInitiative(ctx context.Context, id string) (Initiative, bool, error) {
	rows, err := s.initiativeRecords().ByID(ctx, []string{id})
	return first(initiativesFromRows(rows, err))
}

func (s *PostgresStore) ListInitiatives(ctx context.Context) ([]Initiative, error) {
	return s.initiatives(ctx, sqlcdb.ListInitiativesParams{})
}

func (s *PostgresStore) InitiativesOfCampaign(ctx context.Context, campaignID string) ([]Initiative, error) {
	return initiativesFromRows(s.initiativeRecords().ByParent(ctx, []string{campaignID}))
}

func (s *PostgresStore) InitiativesOfCampaigns(ctx context.Context, campaignIDs []string) ([]Initiative, error) {
	return initiativesFromRows(s.initiativeRecords().ByParent(ctx, campaignIDs))
}

func (s *PostgresStore) InitiativesInScope(ctx context.Context, scope authz.Scope) ([]Initiative, error) {
	return s.initiativesInScope(ctx, scope, nil)
}

func (s *PostgresStore) GetInitiativeInScope(ctx context.Context, scope authz.Scope, id string) (Initiative, bool, error) {
	return first(s.initiativesInScope(ctx, scope, &id))
}

func (s *PostgresStore) PutInitiative(ctx context.Context, i Initiative) error {
	params, err := initiativeParams(i)
	if err != nil {
		return err
	}
	months, amounts := i.BudgetPhasing.Normalised().MonthColumns()
	q := s.change(ctx)
	err = q.PutInitiative(ctx, params)
	if err == nil {
		err = q.DeleteInitiativePhasingMonths(ctx, i.ID)
	}
	if err == nil {
		err = q.InsertInitiativePhasingMonths(ctx, sqlcdb.InsertInitiativePhasingMonthsParams{InitiativeID: i.ID, Months: months, Amounts: amounts})
	}
	if err != nil {
		return fmt.Errorf("save initiative %s: %w", i.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteInitiative(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteInitiative(ctx, id); err != nil {
		return fmt.Errorf("delete initiative %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) initiatives(ctx context.Context, filter sqlcdb.ListInitiativesParams) ([]Initiative, error) {
	rows, err := s.db.Queries(ctx).ListInitiatives(ctx, filter)
	s.initiativeRecords().Keep(ctx, rows)
	return initiativesFromRows(rows, err)
}

func (s *PostgresStore) initiativesInScope(ctx context.Context, scope authz.Scope, id *string) ([]Initiative, error) {
	rows, err := s.db.Queries(ctx).ListInitiativesInScope(ctx, sqlcdb.ListInitiativesInScopeParams{
		ID: id, SeesEverything: scope.Everything, ViewerID: scope.PersonID, ViewsPlans: scope.Holds,
		ViewerTeamIds: scope.TeamIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("select initiatives in scope: %w", err)
	}
	listRows := make([]sqlcdb.ListInitiativesRow, 0, len(rows))
	for _, row := range rows {
		listRows = append(listRows, sqlcdb.ListInitiativesRow(row))
	}
	s.initiativeRecords().Keep(ctx, listRows)
	return initiativesFromRows(listRows, nil)
}

func initiativesFromRows(rows []sqlcdb.ListInitiativesRow, err error) ([]Initiative, error) {
	if err != nil {
		return nil, fmt.Errorf("select initiatives: %w", err)
	}
	list := make([]Initiative, 0, len(rows))
	for _, row := range rows {
		r := row.Initiative
		attributes, err := settings.AttributeValuesOf(r.Attributes)
		if err != nil {
			return nil, fmt.Errorf("decode initiative %s: %w", r.ID, err)
		}
		list = append(list, Initiative{
			ID: r.ID, CampaignID: r.CampaignID, Name: r.Name, Description: r.Description, OwnerID: r.OwnerID,
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
