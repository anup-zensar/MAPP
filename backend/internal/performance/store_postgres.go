package performance

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/record"
)

var _ Store = (*PostgresStore)(nil)

type PostgresStore struct {
	db *database.DB
}

func NewPostgresStore(db *database.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

type metaRow struct {
	createdAt     time.Time
	createdByID   string
	createdByName string
	updatedAt     time.Time
	updatedByID   string
	updatedByName string
}

// optional turns an empty string into the NULL a nullable column expects.
func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func text(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func metaOf(version int, row metaRow) record.Meta {
	return record.Meta{
		Version: version, CreatedAt: row.createdAt, CreatedBy: actor.Actor{ID: row.createdByID, Name: row.createdByName},
		UpdatedAt: row.updatedAt, UpdatedBy: actor.Actor{ID: row.updatedByID, Name: row.updatedByName},
	}
}

func (s *PostgresStore) ListSources(ctx context.Context, id string) ([]SourceView, error) {
	rows, err := s.db.Queries(ctx).ListPerformanceSources(ctx, optional(id))
	if err != nil {
		return nil, fmt.Errorf("select performance sources: %w", err)
	}
	sources := make([]SourceView, 0, len(rows))
	for _, r := range rows {
		p := r.PerformanceSource
		sources = append(sources, SourceView{
			Source: Source{
				ID: p.ID, Name: p.Name, Description: p.Description, OwnerID: p.OwnerID,
				Category: Category(p.Category), Status: Status(p.Status),
				DeliveryRoute: DeliveryRoute(p.DeliveryRoute), Cadence: Cadence(p.Cadence),
				IntegrationID: p.IntegrationID,
				Meta: metaOf(p.Version, metaRow{p.CreatedAt, p.CreatedByID, p.CreatedByName,
					p.UpdatedAt, p.UpdatedByID, p.UpdatedByName}),
			},
			LastLoadedAt: r.LastLoadedAt, DataThrough: r.DataThrough, FactCount: r.FactCount,
		})
	}
	return sources, nil
}

func (s *PostgresStore) PutSource(ctx context.Context, src Source) error {
	err := s.db.Queries(ctx).PutPerformanceSource(ctx, sqlcdb.PutPerformanceSourceParams{
		ID: src.ID, Name: src.Name, Description: src.Description, OwnerID: src.OwnerID,
		Category: string(src.Category), Status: string(src.Status),
		DeliveryRoute: string(src.DeliveryRoute), Cadence: string(src.Cadence),
		IntegrationID: src.IntegrationID,
		Version:       src.Version, CreatedAt: src.CreatedAt, CreatedByID: src.CreatedBy.ID,
		CreatedByName: src.CreatedBy.Name, UpdatedAt: src.UpdatedAt, UpdatedByID: src.UpdatedBy.ID,
		UpdatedByName: src.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save performance source %s: %w", src.ID, err)
	}
	return nil
}

// DeleteSource reports false when the source still holds facts, which the query refuses in one statement
// rather than reading and then deciding (FR-002).
func (s *PostgresStore) DeleteSource(ctx context.Context, id string) (bool, error) {
	deleted, err := s.db.Queries(ctx).DeletePerformanceSource(ctx, id)
	if err != nil {
		return false, fmt.Errorf("delete performance source %s: %w", id, err)
	}
	return deleted > 0, nil
}

func (s *PostgresStore) ListMappings(ctx context.Context, sourceIDs []string) ([]Mapping, error) {
	rows, err := s.db.Queries(ctx).ListPerformanceMappings(ctx, sourceIDs)
	if err != nil {
		return nil, fmt.Errorf("select column mappings: %w", err)
	}
	mappings := make([]Mapping, 0, len(rows))
	for _, r := range rows {
		m := r.PerformanceSourceMapping
		mappings = append(mappings, Mapping{
			ID: m.ID, SourceID: m.SourceID, ColumnName: m.ColumnName, Meaning: Meaning(m.Meaning),
			Stage: m.Stage, WebMetric: WebMetric(m.WebMetric), ModelID: text(m.ModelID),
			Meta: metaOf(m.Version, metaRow{m.CreatedAt, m.CreatedByID, m.CreatedByName,
				m.UpdatedAt, m.UpdatedByID, m.UpdatedByName}),
		})
	}
	return mappings, nil
}

func (s *PostgresStore) PutMapping(ctx context.Context, m Mapping) error {
	err := s.db.Queries(ctx).PutPerformanceMapping(ctx, sqlcdb.PutPerformanceMappingParams{
		ID: m.ID, SourceID: m.SourceID, ColumnName: m.ColumnName, Meaning: string(m.Meaning),
		Stage: m.Stage, WebMetric: string(m.WebMetric), ModelID: optional(m.ModelID),
		Version: m.Version, CreatedAt: m.CreatedAt, CreatedByID: m.CreatedBy.ID,
		CreatedByName: m.CreatedBy.Name, UpdatedAt: m.UpdatedAt, UpdatedByID: m.UpdatedBy.ID,
		UpdatedByName: m.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save column mapping %s: %w", m.ID, err)
	}
	return nil
}

func (s *PostgresStore) DeleteMapping(ctx context.Context, id string) error {
	if err := s.db.Queries(ctx).DeletePerformanceMapping(ctx, id); err != nil {
		return fmt.Errorf("delete column mapping %s: %w", id, err)
	}
	return nil
}

func (s *PostgresStore) ActiveSuppliers(ctx context.Context, modelID string) ([]Supplier, error) {
	rows, err := s.db.Queries(ctx).ActiveSuppliersForModel(ctx, optional(modelID))
	if err != nil {
		return nil, fmt.Errorf("select the active suppliers of model %s: %w", modelID, err)
	}
	suppliers := make([]Supplier, 0, len(rows))
	for _, r := range rows {
		suppliers = append(suppliers, Supplier{
			MappingID: r.ID, SourceID: r.SourceID, SourceName: r.SourceName,
			Meaning: Meaning(r.Meaning), Stage: r.Stage,
		})
	}
	return suppliers, nil
}
