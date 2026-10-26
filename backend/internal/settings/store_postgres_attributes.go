package settings

import (
	"context"
	"fmt"
	"slices"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetAttribute(ctx context.Context, id string) (Attribute, bool, error) {
	attributes, err := s.attributes(ctx)
	row, found := attributes.Find(id)
	if err != nil || !found {
		return Attribute{}, false, err
	}
	return attributeFromRow(row.ShellAttribute, row.Condition), true, nil
}

func (s *PostgresStore) ListAttributes(ctx context.Context) ([]Attribute, error) {
	rows, err := s.attributes(ctx)
	if err != nil {
		return nil, err
	}
	attributes := make([]Attribute, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		attributes = append(attributes, attributeFromRow(row.ShellAttribute, row.Condition))
	}
	return attributes, nil
}

func (s *PostgresStore) PutAttribute(ctx context.Context, a Attribute) error {
	q := s.change(ctx)
	err := q.PutAttribute(ctx, sqlcdb.PutAttributeParams{
		ID: a.ID, Label: a.Label, HelpText: a.HelpText, Level: string(a.Level), Type: string(a.Type),
		Marking: string(a.Marking), Section: a.Section, Position: a.Position,
		ParentAttributeID: database.NullIfEmpty(a.ParentAttributeID), Role: a.Role, Active: a.Active,
		Version: a.Version, CreatedAt: a.CreatedAt, CreatedByID: a.CreatedBy.ID, CreatedByName: a.CreatedBy.Name,
		UpdatedAt: a.UpdatedAt, UpdatedByID: a.UpdatedBy.ID, UpdatedByName: a.UpdatedBy.Name,
	})
	if err == nil {
		err = q.DeleteAttributeConditions(ctx, a.ID)
	}
	if err == nil {
		err = q.InsertAttributeConditions(ctx, sqlcdb.InsertAttributeConditionsParams{AttributeID: a.ID, OptionIds: a.Condition})
	}
	if err != nil {
		return fmt.Errorf("save Shell attribute %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteAttribute(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteAttribute(ctx, id); err != nil {
		return fmt.Errorf("delete Shell attribute %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) GetOption(ctx context.Context, id string) (Option, bool, error) {
	options, err := s.optionRows(ctx)
	row, found := options.Find(id)
	if err != nil || !found {
		return Option{}, false, err
	}
	return optionFromRow(row.ShellAttributeOption, row.ParentOptionIds), true, nil
}

func (s *PostgresStore) ListOptions(ctx context.Context) ([]Option, error) {
	return s.options(ctx, nil)
}

func (s *PostgresStore) OptionsOf(ctx context.Context, attributeID string) ([]Option, error) {
	return s.options(ctx, &attributeID)
}

func (s *PostgresStore) PutOption(ctx context.Context, o Option) error {
	q := s.change(ctx)
	err := q.PutOption(ctx, optionParams(o))
	if err == nil {
		err = q.DeleteOptionParents(ctx, o.ID)
	}
	if err == nil {
		err = q.InsertOptionParents(ctx, sqlcdb.InsertOptionParentsParams{OptionID: o.ID, ParentOptionIds: o.ParentOptionIDs})
	}
	if err != nil {
		return fmt.Errorf("save option %s: %w", o.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteOption(ctx context.Context, id string) error {
	if err := s.change(ctx).DeleteOption(ctx, id); err != nil {
		return fmt.Errorf("delete option %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) options(ctx context.Context, attributeID *string) ([]Option, error) {
	rows, err := s.optionRows(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]Option, 0, len(rows.Rows))
	for _, row := range rows.Rows {
		if attributeID == nil || row.ShellAttributeOption.AttributeID == *attributeID {
			options = append(options, optionFromRow(row.ShellAttributeOption, row.ParentOptionIds))
		}
	}
	return options, nil
}

func attributeFromRow(row sqlcdb.ShellAttribute, condition []string) Attribute {
	return Attribute{
		ID: row.ID, Label: row.Label, HelpText: row.HelpText, Level: Level(row.Level), Type: AttributeType(row.Type),
		Marking: Marking(row.Marking), Section: row.Section, Position: row.Position,
		ParentAttributeID: database.EmptyIfNull(row.ParentAttributeID), Condition: slices.Clone(condition), Role: row.Role,
		Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}

func optionFromRow(row sqlcdb.ShellAttributeOption, parentOptionIDs []string) Option {
	return Option{
		ID: row.ID, AttributeID: row.AttributeID, Label: row.Label, Position: row.Position,
		ParentOptionIDs: slices.Clone(parentOptionIDs), Active: row.Active,
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
