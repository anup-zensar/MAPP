package approval

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

const maxNotifications = 50

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
		shown := mine[:min(len(mine), maxNotifications)]
		stored, err := s.subjectsStored(ctx, shown)
		if err != nil {
			return err
		}
		for _, n := range shown {
			page.Items = append(page.Items, notificationView(n, stored[n.SubjectID]))
		}
		return nil
	})
	if err != nil {
		return NotificationPage{}, err
	}
	return page, nil
}

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
		stored, err := s.subjectsStored(ctx, []Notification{n})
		if err != nil {
			return err
		}
		view = notificationView(n, stored[n.SubjectID])
		return nil
	})
	if err != nil {
		return NotificationView{}, err
	}
	return view, nil
}

func (s *Service) MarkAllRead(ctx context.Context, a actor.Actor) (ReadAll, error) {
	marked := 0
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		mine, err := s.repo.NotificationsOf(ctx, a.ID)
		if err != nil {
			return err
		}
		var unread []string
		for _, n := range mine {
			if !n.Read {
				unread = append(unread, n.ID)
			}
		}
		marked = len(unread)
		return s.repo.MarkNotificationsRead(ctx, unread, s.now())
	})
	if err != nil {
		return ReadAll{}, err
	}
	return ReadAll{Marked: marked, Unread: 0}, nil
}

// notificationView shows n with its record; exists says whether the record is still stored and visible.
func notificationView(n Notification, exists bool) NotificationView {
	kind := n.subjectKind()
	view := NotificationView{
		ID: n.ID, Kind: n.Kind, Title: n.Title, Text: n.Text, Actor: n.Actor, At: n.At, Read: n.Read,
		Subject: NotificationSubject{Kind: kind, Label: kind.Label(), ID: n.SubjectID, Name: n.SubjectName, Exists: exists},
	}
	if kind == KindActivity {
		view.Activity = &NotificationActivity{ID: n.SubjectID, Name: n.SubjectName, Exists: exists}
	}
	return view
}
