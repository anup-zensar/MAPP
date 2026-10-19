package approvals

import (
	"context"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// maxNotifications is how many notifications the bell lists (FR-033).
const maxNotifications = 50

// The approval actions write the notifications (notifier in service.go, data model §4). The bell reads
// them here: a person only ever sees their own. Notifications are not business records, so reading them
// or marking them read writes no activity-log entry and no version (008-P9, D-163).

// ListNotifications returns a's 50 most recent notifications, newest first, with how many a has in all and
// how many of them are unread (FR-033). Each says whether its activity is still stored: one in the recycle
// bin is shown with a pointer there (edge case).
func (s *Service) ListNotifications(ctx context.Context, a actor.Actor) (NotificationPage, error) {
	var page NotificationPage
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		mine, err := s.repo.NotificationsOf(ctx, a.ID)
		if err != nil {
			return err
		}
		page = NotificationPage{Items: make([]NotificationView, 0, min(len(mine), maxNotifications)), Total: len(mine)}
		for _, n := range mine {
			if !n.Read {
				page.Unread++
			}
		}
		stored := map[string]bool{} // activity id → still stored, asked once per activity
		for _, n := range mine[:min(len(mine), maxNotifications)] {
			exists, asked := stored[n.ActivityID]
			if !asked {
				if _, exists, err = s.activities.Activity(ctx, n.ActivityID); err != nil {
					return err
				}
				stored[n.ActivityID] = exists
			}
			page.Items = append(page.Items, notificationView(n, exists))
		}
		return nil
	})
	if err != nil {
		return NotificationPage{}, err
	}
	return page, nil
}

// MarkRead marks a's notification id as read, when a opens it (FR-034), and returns it. Someone else's
// notification is answered like one that does not exist; marking a read notification again changes
// nothing.
func (s *Service) MarkRead(ctx context.Context, a actor.Actor, id string) (NotificationView, error) {
	var view NotificationView
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		n, found, err := s.repo.GetNotification(ctx, id)
		if err != nil {
			return err
		}
		if !found || n.RecipientID != a.ID {
			return errs.NotFound(notificationGoneMessage)
		}
		if !n.Read {
			now := s.now()
			n.Read, n.ReadAt = true, &now
			if err := s.repo.PutNotification(ctx, n); err != nil {
				return err
			}
		}
		_, exists, err := s.activities.Activity(ctx, n.ActivityID)
		if err != nil {
			return err
		}
		view = notificationView(n, exists)
		return nil
	})
	if err != nil {
		return NotificationView{}, err
	}
	return view, nil
}

// MarkAllRead marks every unread notification of a as read (FR-034) and says how many it marked; they stay
// listed.
func (s *Service) MarkAllRead(ctx context.Context, a actor.Actor) (ReadAll, error) {
	marked := 0
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		mine, err := s.repo.NotificationsOf(ctx, a.ID)
		if err != nil {
			return err
		}
		now := s.now()
		for _, n := range mine {
			if n.Read {
				continue
			}
			n.Read, n.ReadAt = true, &now
			if err := s.repo.PutNotification(ctx, n); err != nil {
				return err
			}
			marked++
		}
		return nil
	})
	if err != nil {
		return ReadAll{}, err
	}
	return ReadAll{Marked: marked, Unread: 0}, nil
}

// notificationView returns n as the API shows it; exists says whether its activity is still stored.
func notificationView(n Notification, exists bool) NotificationView {
	return NotificationView{
		ID: n.ID, Kind: n.Kind, Title: n.Title, Text: n.Text,
		Activity: NotificationActivity{ID: n.ActivityID, Name: n.ActivityName, Exists: exists},
		Actor:    n.Actor, At: n.At, Read: n.Read,
	}
}
