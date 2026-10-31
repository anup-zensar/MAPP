package performance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/database/sqlcdb"
)

// The reads below serve the recycle bin, which collects an activity's children before deleting it, so they
// carry no scope: history has already decided the actor may bin the activity.
func (s *PostgresStore) FactIDsOf(ctx context.Context, activityIDs []string) ([]string, error) {
	ids, err := s.db.Queries(ctx).FactIDsOfActivities(ctx, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("select the facts of %d activities: %w", len(activityIDs), err)
	}
	return ids, nil
}

func (s *PostgresStore) MediaCodeIDsOf(ctx context.Context, activityIDs []string) ([]string, error) {
	ids, err := s.db.Queries(ctx).MediaCodeIDsOfActivities(ctx, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("select the media codes of %d activities: %w", len(activityIDs), err)
	}
	return ids, nil
}

func (s *PostgresStore) AliasIDsOf(ctx context.Context, activityIDs []string) ([]string, error) {
	ids, err := s.db.Queries(ctx).AliasIDsOfActivities(ctx, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("select the aliases of %d activities: %w", len(activityIDs), err)
	}
	return ids, nil
}

func (s *PostgresStore) IntegrationIDsOf(ctx context.Context, activityIDs []string) ([]string, error) {
	ids, err := s.db.Queries(ctx).IntegrationIDsOfActivities(ctx, activityIDs)
	if err != nil {
		return nil, fmt.Errorf("select the integration ids of %d activities: %w", len(activityIDs), err)
	}
	return ids, nil
}

func (s *PostgresStore) GetFact(ctx context.Context, id string) (Fact, bool, error) {
	row, err := s.db.Queries(ctx).GetPerformanceFact(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Fact{}, false, nil
	}
	if err != nil {
		return Fact{}, false, fmt.Errorf("select performance fact %s: %w", id, err)
	}
	return factOf(row.PerformanceFact), true, nil
}

func (s *PostgresStore) GetMediaCode(ctx context.Context, id string) (MediaCode, bool, error) {
	row, err := s.db.Queries(ctx).GetMediaCode(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return MediaCode{}, false, nil
	}
	if err != nil {
		return MediaCode{}, false, fmt.Errorf("select media code %s: %w", id, err)
	}
	m := row.MediaCode
	code := MediaCode{
		ID: m.ID, Code: m.Code, ActivityID: m.ActivityID, LandingPage: m.LandingPage,
		Status: Status(m.Status),
		Meta: metaOf(m.Version, metaRow{m.CreatedAt, m.CreatedByID, m.CreatedByName,
			m.UpdatedAt, m.UpdatedByID, m.UpdatedByName}),
	}
	if len(m.Attributes) > 0 {
		if err := json.Unmarshal(m.Attributes, &code.Attributes); err != nil {
			return MediaCode{}, false, fmt.Errorf("read the attributes of media code %s: %w", id, err)
		}
	}
	return code, true, nil
}

func (s *PostgresStore) GetAlias(ctx context.Context, id string) (Alias, bool, error) {
	row, err := s.db.Queries(ctx).GetAlias(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Alias{}, false, nil
	}
	if err != nil {
		return Alias{}, false, fmt.Errorf("select alias %s: %w", id, err)
	}
	a := row.ActivityExternalNameAlias
	return Alias{ID: a.ID, ActivityID: a.ActivityID, Name: a.Name,
		Meta: metaOf(a.Version, metaRow{a.CreatedAt, a.CreatedByID, a.CreatedByName,
			a.UpdatedAt, a.UpdatedByID, a.UpdatedByName})}, true, nil
}

func (s *PostgresStore) GetIntegrationID(ctx context.Context, id string) (IntegrationID, bool, error) {
	row, err := s.db.Queries(ctx).GetIntegrationID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return IntegrationID{}, false, nil
	}
	if err != nil {
		return IntegrationID{}, false, fmt.Errorf("select integration id %s: %w", id, err)
	}
	k := row.ActivityIntegrationID
	return IntegrationID{ID: k.ID, ActivityID: k.ActivityID, SourceID: k.SourceID, Value: k.Value,
		Meta: metaOf(k.Version, metaRow{k.CreatedAt, k.CreatedByID, k.CreatedByName,
			k.UpdatedAt, k.UpdatedByID, k.UpdatedByName})}, true, nil
}

func (s *PostgresStore) DeleteFacts(ctx context.Context, ids []string) error {
	if err := s.db.Queries(ctx).DeletePerformanceFacts(ctx, ids); err != nil {
		return fmt.Errorf("delete %d performance facts: %w", len(ids), err)
	}
	return nil
}

func (s *PostgresStore) DeleteMediaCodes(ctx context.Context, ids []string) error {
	if err := s.db.Queries(ctx).DeleteMediaCodes(ctx, ids); err != nil {
		return fmt.Errorf("delete %d media codes: %w", len(ids), err)
	}
	return nil
}

func (s *PostgresStore) DeleteAliases(ctx context.Context, ids []string) error {
	if err := s.db.Queries(ctx).DeleteAliases(ctx, ids); err != nil {
		return fmt.Errorf("delete %d aliases: %w", len(ids), err)
	}
	return nil
}

func (s *PostgresStore) DeleteIntegrationIDs(ctx context.Context, ids []string) error {
	if err := s.db.Queries(ctx).DeleteIntegrationIDs(ctx, ids); err != nil {
		return fmt.Errorf("delete %d integration ids: %w", len(ids), err)
	}
	return nil
}

func (s *PostgresStore) PutAlias(ctx context.Context, a Alias) error {
	err := s.db.Queries(ctx).PutAlias(ctx, sqlcdb.PutAliasParams{
		ID: a.ID, ActivityID: a.ActivityID, Name: a.Name,
		Version: a.Version, CreatedAt: a.CreatedAt, CreatedByID: a.CreatedBy.ID,
		CreatedByName: a.CreatedBy.Name, UpdatedAt: a.UpdatedAt, UpdatedByID: a.UpdatedBy.ID,
		UpdatedByName: a.UpdatedBy.Name,
	})
	if err != nil {
		return fmt.Errorf("save alias %s: %w", a.ID, err)
	}
	return nil
}

func (s *PostgresStore) ExternalNameHolders(ctx context.Context, name string) ([]string, error) {
	ids, err := s.db.Queries(ctx).ExternalNameHolders(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("select who holds the external name %q: %w", name, err)
	}
	return ids, nil
}
