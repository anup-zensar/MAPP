package approvals

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/date"
	"mapp/internal/platform/record"
)

func (s *PostgresStore) GetNotification(ctx context.Context, id string) (Notification, bool, error) {
	row, err := s.db.Queries(ctx).GetNotification(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, false, nil
	}
	if err != nil {
		return Notification{}, false, fmt.Errorf("select notification %s: %w", id, err)
	}
	return notificationFromRow(row), true, nil
}

func (s *PostgresStore) NotificationsOf(ctx context.Context, personID string) ([]Notification, error) {
	rows, err := s.db.Queries(ctx).NotificationsOf(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("select notifications of %s: %w", personID, err)
	}
	notifications := make([]Notification, 0, len(rows))
	for _, row := range rows {
		notifications = append(notifications, notificationFromRow(row))
	}
	return notifications, nil
}

func (s *PostgresStore) PutNotification(ctx context.Context, n Notification) error {
	err := s.db.Queries(ctx).PutNotification(ctx, sqlcdb.PutNotificationParams{
		ID: n.ID, RecipientID: n.RecipientID, Kind: string(n.Kind), Title: n.Title, Text: n.Text,
		ActivityID: n.ActivityID, ActivityName: n.ActivityName, ActorID: n.Actor.ID, ActorName: n.Actor.Name,
		At: n.At, Read: n.Read, ReadAt: n.ReadAt,
	})
	if err != nil {
		return fmt.Errorf("save notification %s: %w", n.ID, err)
	}
	return nil
}

func (s *PostgresStore) GetDelegation(ctx context.Context, id string) (Delegation, bool, error) {
	row, err := s.db.Queries(ctx).GetDelegation(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Delegation{}, false, nil
	}
	if err != nil {
		return Delegation{}, false, fmt.Errorf("select approval delegation %s: %w", id, err)
	}
	return delegationFromRow(row), true, nil
}

func (s *PostgresStore) ListDelegations(ctx context.Context) ([]Delegation, error) {
	rows, err := s.db.Queries(ctx).ListDelegations(ctx)
	if err != nil {
		return nil, fmt.Errorf("select approval delegations: %w", err)
	}
	delegations := make([]Delegation, 0, len(rows))
	for _, row := range rows {
		delegations = append(delegations, delegationFromRow(row))
	}
	return delegations, nil
}

func (s *PostgresStore) PutDelegation(ctx context.Context, dl Delegation) error {
	params := sqlcdb.PutDelegationParams{
		ID: dl.ID, PersonID: dl.PersonID, DelegateID: dl.DelegateID, StartDate: dl.StartDate.Time(),
		EndDate: dl.EndDate.Time(), Note: dl.Note, CancelledAt: dl.CancelledAt, Version: dl.Version,
		CreatedAt: dl.CreatedAt, CreatedByID: dl.CreatedBy.ID, CreatedByName: dl.CreatedBy.Name,
		UpdatedAt: dl.UpdatedAt, UpdatedByID: dl.UpdatedBy.ID, UpdatedByName: dl.UpdatedBy.Name,
	}
	params.CancelledByID, params.CancelledByName = actorColumns(dl.CancelledBy)
	if err := s.db.Queries(ctx).PutDelegation(ctx, params); err != nil {
		return fmt.Errorf("save approval delegation %s: %w", dl.ID, err)
	}
	return nil
}

func notificationFromRow(row sqlcdb.Notification) Notification {
	return Notification{
		ID: row.ID, RecipientID: row.RecipientID, Kind: NotificationKind(row.Kind), Title: row.Title, Text: row.Text,
		ActivityID: row.ActivityID, ActivityName: row.ActivityName, Actor: actor.Actor{ID: row.ActorID, Name: row.ActorName},
		At: row.At, Read: row.Read, ReadAt: row.ReadAt,
	}
}

func delegationFromRow(row sqlcdb.ApprovalDelegation) Delegation {
	return Delegation{
		ID: row.ID, PersonID: row.PersonID, DelegateID: row.DelegateID, StartDate: date.Today(row.StartDate),
		EndDate: date.Today(row.EndDate), Note: row.Note, CancelledAt: row.CancelledAt,
		CancelledBy: actorOf(row.CancelledByID, row.CancelledByName),
		Meta: record.Meta{
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			CreatedBy: actor.Actor{ID: row.CreatedByID, Name: row.CreatedByName},
			UpdatedAt: row.UpdatedAt,
			UpdatedBy: actor.Actor{ID: row.UpdatedByID, Name: row.UpdatedByName},
		},
	}
}
