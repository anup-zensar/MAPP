package approval

import (
	"time"

	"mapp/internal/platform/errs"
)

const (
	BucketUnder1    = "under1"
	BucketDays1to2  = "days1to2"
	BucketDays3to5  = "days3to5"
	BucketDays6to10 = "days6to10"
	BucketOver10    = "over10"
)

type AgeingBucket struct {
	Bucket string `json:"bucket"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
}

type WaitingByLevel struct {
	Kind               SubjectKind   `json:"kind"`
	Workflow           RoundWorkflow `json:"workflow"`
	Level              int           `json:"level"`
	LevelLabel         string        `json:"levelLabel"`
	Waiting            int           `json:"waiting"`
	Overdue            int           `json:"overdue"`
	OldestWaitingSince time.Time     `json:"oldestWaitingSince"`
}

type MonitoringCards struct {
	Overdue               int     `json:"overdue"`
	Escalated             int     `json:"escalated"`
	AverageDecisionDays   *string `json:"averageDecisionDays"`
	DecidedThisMonth      int     `json:"decidedThisMonth"`
	AutoApprovedThisMonth int     `json:"autoApprovedThisMonth"`
}

type Monitoring struct {
	Cards             MonitoringCards  `json:"cards"`
	Ageing            []AgeingBucket   `json:"ageing"`
	ByLevel           []WaitingByLevel `json:"byLevel"`
	Overdue           []Row            `json:"overdue"`
	CanRunEscalations bool             `json:"canRunEscalations"`
}

const (
	EventReminded     = "reminded"
	EventEscalated    = "escalated"
	EventNotEscalated = "notEscalated"
)

type EscalationEvent struct {
	Activity    errs.RecordRef `json:"activity"`
	RoundNumber int            `json:"roundNumber"`
	Level       int            `json:"level"`
	Action      string         `json:"action"`
	To          []PersonView   `json:"to"`
	WaitedDays  int            `json:"waitedDays"`
}

type EscalationRun struct {
	Reminded     int               `json:"reminded"`
	Escalated    int               `json:"escalated"`
	Events       []EscalationEvent `json:"events"`
	NotEscalated []EscalationEvent `json:"notEscalated"`
}
