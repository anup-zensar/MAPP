package approvals

import (
	"time"

	"mapp/internal/platform/errs"
)

// The ageing buckets of the Monitoring tab (ApprovalAgeingBucket.bucket), in order.
const (
	BucketUnder1    = "under1"
	BucketDays1to2  = "days1to2"
	BucketDays3to5  = "days3to5"
	BucketDays6to10 = "days6to10"
	BucketOver10    = "over10"
)

// AgeingBucket is one bar of "How long approvals have waited" (ApprovalAgeingBucket).
type AgeingBucket struct {
	Bucket string `json:"bucket"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
}

// WaitingByLevel is one row of "Waiting by level" (ApprovalLevelWaiting).
type WaitingByLevel struct {
	Workflow           RoundWorkflow `json:"workflow"`
	Level              int           `json:"level"`
	LevelLabel         string        `json:"levelLabel"`
	Waiting            int           `json:"waiting"`
	Overdue            int           `json:"overdue"`
	OldestWaitingSince time.Time     `json:"oldestWaitingSince"`
}

// MonitoringCards are the four summary cards of the Monitoring tab (ApprovalMonitoring.cards).
type MonitoringCards struct {
	Overdue               int     `json:"overdue"`
	Escalated             int     `json:"escalated"`
	AverageDecisionDays   *string `json:"averageDecisionDays"` // one decimal, "2.4"; nil when nothing was decided
	DecidedThisMonth      int     `json:"decidedThisMonth"`
	AutoApprovedThisMonth int     `json:"autoApprovedThisMonth"`
}

// Monitoring is the answer of GET /api/v1/approvals/monitoring (ApprovalMonitoring).
type Monitoring struct {
	Cards             MonitoringCards  `json:"cards"`
	Ageing            []AgeingBucket   `json:"ageing"` // always the five buckets, in order
	ByLevel           []WaitingByLevel `json:"byLevel"`
	Overdue           []Row            `json:"overdue"` // oldest waiting first
	CanRunEscalations bool             `json:"canRunEscalations"`
}

// The actions of an escalation run's events (EscalationEvent.action).
const (
	EventReminded     = "reminded"
	EventEscalated    = "escalated"
	EventNotEscalated = "notEscalated"
)

// EscalationEvent is one reminder, escalation or level that could not be escalated (EscalationEvent).
type EscalationEvent struct {
	Activity    errs.RecordRef `json:"activity"`
	RoundNumber int            `json:"roundNumber"`
	Level       int            `json:"level"`
	Action      string         `json:"action"`
	To          []PersonView   `json:"to"` // reminded: who was told; escalated: the backup approver; else empty
	WaitedDays  int            `json:"waitedDays"`
}

// EscalationRun is the answer of POST /api/v1/approvals/escalations/run (EscalationRunResult).
type EscalationRun struct {
	Reminded     int               `json:"reminded"`
	Escalated    int               `json:"escalated"`
	Events       []EscalationEvent `json:"events"`
	NotEscalated []EscalationEvent `json:"notEscalated"`
}
