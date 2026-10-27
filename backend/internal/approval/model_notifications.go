package approval

import (
	"time"

	"mapp/internal/platform/actor"
)

// NotificationKind is what a notification is about.
type NotificationKind string

// The notification kinds.
const (
	NotifyApprovalNeeded NotificationKind = "approvalNeeded"
	NotifyLevelApproved  NotificationKind = "levelApproved"
	NotifyApproved       NotificationKind = "approved"
	NotifyDeclined       NotificationKind = "declined"
	NotifyRecalled       NotificationKind = "recalled"
	NotifyRestarted      NotificationKind = "restarted"
	NotifyAutoApproved   NotificationKind = "autoApproved" // 017: approved at submission, to the owner
	NotifyReminder       NotificationKind = "reminder"     // 017: a level has waited the reminder days
	NotifyEscalated      NotificationKind = "escalated"    // 017: a level was escalated to its backup approver
)

// Notification is an in-app message to one person (table "notifications", data model §4). Not a business
// record: no version, and marking it read writes no log entry (008-P9).
type Notification struct {
	ID           string           `json:"id"`          // "NT-1001"
	RecipientID  string           `json:"recipientId"` // never the actor of the action (FR-036)
	Kind         NotificationKind `json:"kind"`
	Title        string           `json:"title"` // fixed at creation
	Text         string           `json:"text"`  // one line naming the activity and the actor
	ActivityID   string           `json:"activityId"`
	ActivityName string           `json:"activityName"` // the activity's name at the time
	Actor        actor.Actor      `json:"actor"`        // who caused it
	At           time.Time        `json:"at"`
	Read         bool             `json:"read"`
	ReadAt       *time.Time       `json:"readAt"` // nil while unread
}

// NotificationActivity is the activity of a notification, and whether it is still stored.
type NotificationActivity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Exists bool   `json:"exists"`
}

// NotificationView is a notification as the API returns it (Notification).
type NotificationView struct {
	ID       string               `json:"id"`
	Kind     NotificationKind     `json:"kind"`
	Title    string               `json:"title"`
	Text     string               `json:"text"`
	Activity NotificationActivity `json:"activity"`
	Actor    actor.Actor          `json:"actor"`
	At       time.Time            `json:"at"`
	Read     bool                 `json:"read"`
}

// NotificationPage is the answer of GET /api/v1/notifications (NotificationPage).
type NotificationPage struct {
	Items  []NotificationView `json:"items"`  // at most 50, newest first
	Total  int                `json:"total"`  // all of the person's notifications
	Unread int                `json:"unread"` // all unread ones
}

// ReadAll is the answer of POST /api/v1/notifications/read-all (NotificationReadAll).
type ReadAll struct {
	Marked int `json:"marked"`
	Unread int `json:"unread"` // always 0 after the call
}
