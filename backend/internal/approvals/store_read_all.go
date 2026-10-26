package approvals

import (
	"context"
	"fmt"
	"time"

	"mapp/internal/platform/database/sqlcdb"
)

func (s *PostgresStore) MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.change(ctx).MarkNotificationsRead(ctx, sqlcdb.MarkNotificationsReadParams{Ids: ids, ReadAt: &at}); err != nil {
		return fmt.Errorf("mark notifications read: %w", err)
	}
	return nil
}

func (r jsonRepository) MarkNotificationsRead(ctx context.Context, ids []string, at time.Time) error {
	for _, id := range ids {
		n, found, err := r.GetNotification(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		n.Read, n.ReadAt = true, &at
		if err := r.PutNotification(ctx, n); err != nil {
			return err
		}
	}
	return nil
}
