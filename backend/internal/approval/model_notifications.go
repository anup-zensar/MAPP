package approval

import (
	"time"

	"mapp/internal/platform/actor"
)

type NotificationKind string

const (
	NotifyApprovalNeeded NotificationKind = "approvalNeeded"
	NotifyLevelApproved  NotificationKind = "levelApproved"
	NotifyApproved       NotificationKind = "approved"
	NotifyDeclined       NotificationKind = "declined"
	NotifyRecalled       NotificationKind = "recalled"
	NotifyRestarted      NotificationKind = "restarted"
	NotifyAutoApproved   NotificationKind = "autoApproved"
	NotifyReminder       NotificationKind = "reminder"
	NotifyEscalated      NotificationKind = "escalated"
)

// Notification is an in-app message to one person; not a business record, so it has no version.
type Notification struct {
	ID          string           `json:"id"`
	RecipientID string           `json:"recipientId"`
	Kind        NotificationKind `json:"kind"`
	Title       string           `json:"title"`
	Text        string           `json:"text"`
	SubjectKind SubjectKind      `json:"subjectKind"`
	SubjectID   string           `json:"subjectId"`
	SubjectName string           `json:"subjectName"`
	Actor       actor.Actor      `json:"actor"`
	At          time.Time        `json:"at"`
	Read        bool             `json:"read"`
	ReadAt      *time.Time       `json:"readAt"`
}

func (n Notification) subjectKind() SubjectKind {
	if n.SubjectKind == "" {
		return KindActivity
	}
	return n.SubjectKind
}

type NotificationActivity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

type NotificationSubject struct {
	Kind   SubjectKind `json:"kind"`
	Label  string      `json:"label"`
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Exists bool        `json:"exists"`
}

type NotificationView struct {
	ID       string                `json:"id"`
	Kind     NotificationKind      `json:"kind"`
	Title    string                `json:"title"`
	Text     string                `json:"text"`
	Subject  NotificationSubject   `json:"subject"`
	Activity *NotificationActivity `json:"activity"`
	Actor    actor.Actor           `json:"actor"`
	At       time.Time             `json:"at"`
	Read     bool                  `json:"read"`
}

type NotificationPage struct {
	Items  []NotificationView `json:"items"`
	Total  int                `json:"total"`
	Unread int                `json:"unread"`
}

type ReadAll struct {
	Marked int `json:"marked"`
	Unread int `json:"unread"`
}
