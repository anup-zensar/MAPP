package approval

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type Outcome string

const (
	OutcomeOpen      Outcome = "open"
	OutcomeApproved  Outcome = "approved"
	OutcomeDeclined  Outcome = "declined"
	OutcomeRecalled  Outcome = "recalled"
	OutcomeRestarted Outcome = "restarted"
	OutcomeCancelled Outcome = "cancelled"
	// OutcomeSuperseded: the record was re-submitted while the round was open.
	OutcomeSuperseded Outcome = "superseded"
)

// Round is one attempt to get one record approved; the rounds of a record share one numbering.
type Round struct {
	ID             string       `json:"id"`
	SubjectKind    SubjectKind  `json:"subjectKind"`
	SubjectID      string       `json:"subjectId"`
	Number         int          `json:"number"`
	WorkflowID     string       `json:"workflowId"`
	WorkflowName   string       `json:"workflowName"`
	Levels         []RoundLevel `json:"levels"`
	CurrentLevel   int          `json:"currentLevel"`
	Outcome        Outcome      `json:"outcome"`
	SubmittedAt    time.Time    `json:"submittedAt"`
	SubmittedBy    actor.Actor  `json:"submittedBy"`
	Comment        string       `json:"comment"`
	EndedAt        *time.Time   `json:"endedAt"`
	Facts          *RuleFacts   `json:"facts"`
	ReminderDays   *int         `json:"reminderDays"`
	EscalationDays *int         `json:"escalationDays"`
	AutoApproved   bool         `json:"autoApproved"`
	record.Meta
}

type RuleFacts struct {
	PlannedCost   *money.Money `json:"plannedCost"`
	ActivityType  *FactRef     `json:"activityType"`
	BusinessUnits []FactOption `json:"businessUnits"`
}

type FactRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FactOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type RoundLevel struct {
	Number       int           `json:"number"`
	Name         string        `json:"name"`
	Approvers    []actor.Actor `json:"approvers"`
	ApproverType string        `json:"approverType"`
	Source       string        `json:"source"`
	RoleID       string        `json:"roleId"`
	RoleName     string        `json:"roleName"`
	Mode         string        `json:"mode"`
	Resolution   string        `json:"resolution"`
	RuleText     string        `json:"ruleText"`
	Excluded     []Excluded    `json:"excluded"`
	BackupUsed   bool          `json:"backupUsed"`
	NoApprover   bool          `json:"noApprover"`
	Backup       *actor.Actor  `json:"backup"`
	StartedAt    *time.Time    `json:"startedAt"`
	RemindedAt   *time.Time    `json:"remindedAt"`
	EscalatedAt  *time.Time    `json:"escalatedAt"`
}

const (
	ResolutionDecide       = "decide"
	ResolutionSkipped      = "skipped"
	ResolutionAutoApproved = "autoApproved"
)

type Excluded struct {
	Person actor.Actor `json:"person"`
	Reason string      `json:"reason"`
}

const (
	ReasonInactive      = "inactive"
	ReasonCannotApprove = "cannotApprove"
	ReasonOwner         = "owner"
	ReasonSubmitter     = "submitter"
)

const (
	LevelApproved     = "approved"
	LevelDeclined     = "declined"
	LevelWaiting      = "waiting"
	LevelNotReached   = "notReached"
	LevelNotDecided   = "notDecided"
	LevelSkipped      = "skipped"
	LevelAutoApproved = "autoApproved"
)

type ChainLevel struct {
	Number        int            `json:"number"`
	Label         string         `json:"label"`
	Approvers     []PersonView   `json:"approvers"`
	State         string         `json:"state"`
	Decision      *Decision      `json:"decision"`
	Mode          string         `json:"mode"`
	Source        string         `json:"source"`
	RuleText      string         `json:"ruleText"`
	Decisions     []Decision     `json:"decisions"`
	ApprovedCount int            `json:"approvedCount"`
	RequiredCount int            `json:"requiredCount"`
	Excluded      []ExcludedView `json:"excluded"`
	Backup        *PersonView    `json:"backup"`
	BackupUsed    bool           `json:"backupUsed"`
	WaitingSince  *time.Time     `json:"waitingSince"`
	RemindedAt    *time.Time     `json:"remindedAt"`
	EscalatedAt   *time.Time     `json:"escalatedAt"`
	EscalatedTo   *PersonView    `json:"escalatedTo"`
	NoApprover    bool           `json:"noApprover"`
	Advice        string         `json:"advice"`
}

type ExcludedView struct {
	Person PersonView `json:"person"`
	Reason string     `json:"reason"`
}

type RuleFactsView struct {
	PlannedCost   *money.Money    `json:"plannedCost"`
	ActivityType  *errs.RecordRef `json:"activityType"`
	BusinessUnits []OptionRef     `json:"businessUnits"`
}

type RoundWorkflow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RoundView struct {
	ID             string             `json:"id"`
	Kind           SubjectKind        `json:"kind"`
	Number         int                `json:"number"`
	Outcome        Outcome            `json:"outcome"`
	Workflow       RoundWorkflow      `json:"workflow"`
	SubmittedAt    time.Time          `json:"submittedAt"`
	SubmittedBy    actor.Actor        `json:"submittedBy"`
	Comment        string             `json:"comment"`
	EndedAt        *time.Time         `json:"endedAt"`
	CurrentLevel   int                `json:"currentLevel"`
	LevelCount     int                `json:"levelCount"`
	Levels         []ChainLevel       `json:"levels"`
	History        []HistoryEntryView `json:"history"`
	Facts          *RuleFactsView     `json:"facts"`
	ReminderDays   *int               `json:"reminderDays"`
	EscalationDays *int               `json:"escalationDays"`
	AutoApproved   bool               `json:"autoApproved"`
}

type DeclineNote struct {
	RoundNumber  int           `json:"roundNumber"`
	Level        int           `json:"level"`
	LevelLabel   string        `json:"levelLabel"`
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Capacity     Capacity      `json:"capacity"`
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"`
}

// kind is the round's record type: rounds stored before feature 031 are about activities.
func (r Round) kind() SubjectKind {
	if r.SubjectKind == "" {
		return KindActivity
	}
	return r.SubjectKind
}
