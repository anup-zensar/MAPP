package campaign

import (
	"context"
	"fmt"

	"mapp/internal/overspend"

	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
)

// AddCampaigns stores new campaigns, each with its phasing months, in two statements whatever their number.
func (s *PostgresStore) AddCampaigns(ctx context.Context, list []Campaign) error {
	if len(list) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddCampaignsParams, 0, len(list))
	var months []sqlcdb.AddCampaignPhasingMonthsParams
	for _, c := range list {
		params, err := campaignParams(c)
		if err != nil {
			return err
		}
		rows = append(rows, sqlcdb.AddCampaignsParams(params))
		phasingMonths, amounts := c.BudgetPhasing.Normalised().MonthColumns()
		for i, month := range phasingMonths {
			months = append(months, sqlcdb.AddCampaignPhasingMonthsParams{CampaignID: c.ID, Month: month, AmountMinor: amounts[i]})
		}
	}
	q := s.change(ctx)
	if _, err := q.AddCampaigns(ctx, rows); err != nil {
		return fmt.Errorf("add campaigns: %w", err)
	}
	if len(months) == 0 {
		return nil
	}
	if _, err := q.AddCampaignPhasingMonths(ctx, months); err != nil {
		return fmt.Errorf("add campaign phasing: %w", err)
	}
	return nil
}

// AddInitiatives stores new initiatives, each with its phasing months, in two statements whatever their
// number.
func (s *PostgresStore) AddInitiatives(ctx context.Context, list []Initiative) error {
	if len(list) == 0 {
		return nil
	}
	rows := make([]sqlcdb.AddInitiativesParams, 0, len(list))
	var months []sqlcdb.AddInitiativePhasingMonthsParams
	for _, i := range list {
		params, err := initiativeParams(i)
		if err != nil {
			return err
		}
		rows = append(rows, sqlcdb.AddInitiativesParams(params))
		phasingMonths, amounts := i.BudgetPhasing.Normalised().MonthColumns()
		for n, month := range phasingMonths {
			months = append(months, sqlcdb.AddInitiativePhasingMonthsParams{InitiativeID: i.ID, Month: month, AmountMinor: amounts[n]})
		}
	}
	q := s.change(ctx)
	if _, err := q.AddInitiatives(ctx, rows); err != nil {
		return fmt.Errorf("add initiatives: %w", err)
	}
	if len(months) == 0 {
		return nil
	}
	if _, err := q.AddInitiativePhasingMonths(ctx, months); err != nil {
		return fmt.Errorf("add initiative phasing: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteCampaigns(ctx context.Context, ids []string) error {
	if err := s.change(ctx).DeleteCampaigns(ctx, ids); err != nil {
		return fmt.Errorf("delete campaigns: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteInitiatives(ctx context.Context, ids []string) error {
	if err := s.change(ctx).DeleteInitiatives(ctx, ids); err != nil {
		return fmt.Errorf("delete initiatives: %w", err)
	}
	return nil
}

func campaignParams(c Campaign) (sqlcdb.PutCampaignParams, error) {
	attributes, err := c.Attributes.Column()
	if err != nil {
		return sqlcdb.PutCampaignParams{}, fmt.Errorf("encode campaign %s: %w", c.ID, err)
	}
	minor, currency := database.MoneyColumns(c.Budget)
	return sqlcdb.PutCampaignParams{
		ID: c.ID, PlanID: c.PlanID, Name: c.Name, Description: c.Description, OwnerID: ownerColumn(c.OwnerID),
		StartDate: database.DateColumn(c.StartDate), EndDate: database.DateColumn(c.EndDate), BudgetMinor: minor,
		BudgetCurrency: currency, Attributes: attributes, PhasingMethod: string(c.BudgetPhasing.Normalised().Method),
		Version: c.Version, CreatedAt: c.CreatedAt, CreatedByID: c.CreatedBy.ID, CreatedByName: c.CreatedBy.Name,
		UpdatedAt: c.UpdatedAt, UpdatedByID: c.UpdatedBy.ID, UpdatedByName: c.UpdatedBy.Name,
		Overspend: overspend.Stored(c.Overspend),
	}, nil
}

func initiativeParams(i Initiative) (sqlcdb.PutInitiativeParams, error) {
	attributes, err := i.Attributes.Column()
	if err != nil {
		return sqlcdb.PutInitiativeParams{}, fmt.Errorf("encode initiative %s: %w", i.ID, err)
	}
	minor, currency := database.MoneyColumns(i.Budget)
	return sqlcdb.PutInitiativeParams{
		ID: i.ID, CampaignID: i.CampaignID, Name: i.Name, Description: i.Description, OwnerID: ownerColumn(i.OwnerID),
		StartDate: database.DateColumn(i.StartDate), EndDate: database.DateColumn(i.EndDate), BudgetMinor: minor,
		BudgetCurrency: currency, Attributes: attributes, PhasingMethod: string(i.BudgetPhasing.Normalised().Method),
		Version: i.Version, CreatedAt: i.CreatedAt, CreatedByID: i.CreatedBy.ID, CreatedByName: i.CreatedBy.Name,
		UpdatedAt: i.UpdatedAt, UpdatedByID: i.UpdatedBy.ID, UpdatedByName: i.UpdatedBy.Name,
		Overspend: overspend.Stored(i.Overspend),
	}, nil
}
