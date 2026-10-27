package approvals

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

// Outcome is how an approval round stands or ended (data model §2).
type Outcome string

// The round outcomes. At most one round per activity is open, exactly while the activity is Submitted.
const (
	OutcomeOpen      Outcome = "open"
	OutcomeApproved  Outcome = "approved"
	OutcomeDeclined  Outcome = "declined"
	OutcomeRecalled  Outcome = "recalled"
	OutcomeRestarted Outcome = "restarted" // a re-approval change (006, 007)
	OutcomeCancelled Outcome = "cancelled" // the activity was restored from the recycle bin
)

// Round is one attempt to get one activity approved (table "approval_rounds", data model §2). It keeps
// its own copy of the workflow's levels, so a later edit of the workflow never changes it (FR-006).
type Round struct {
	ID           string       `json:"id"`           // "AR-1001"
	ActivityID   string       `json:"activityId"`   // never changes
	Number       int          `json:"number"`       // 1 for the activity's first round, then the highest + 1
	WorkflowID   string       `json:"workflowId"`   // the activity's workflow at submission
	WorkflowName string       `json:"workflowName"` // its name at submission
	Levels       []RoundLevel `json:"levels"`       // copy of the workflow's levels at submission
	CurrentLevel int          `json:"currentLevel"` // 1 at submission; after the round ends, the level it ended at
	Outcome      Outcome      `json:"outcome"`
	SubmittedAt  time.Time    `json:"submittedAt"`
	SubmittedBy  actor.Actor  `json:"submittedBy"`
	Comment      string       `json:"comment"` // the submit comment, 0–2,000 characters
	EndedAt      *time.Time   `json:"endedAt"` // nil while open
	// Facts are the values the rules used at submission (017 FR-010); nil for rounds stored before 017.
	Facts *RuleFacts `json:"facts"`
	// ReminderDays and EscalationDays are copied from the workflow at submission (008 FR-006 holds).
	ReminderDays   *int `json:"reminderDays"`
	EscalationDays *int `json:"escalationDays"`
	AutoApproved   bool `json:"autoApproved"` // no level needed a decision: approved at submission (017 FR-014)
	record.Meta
}

// RuleFacts are the values of an activity the rules look at, taken once at submission (data-model §3).
type RuleFacts struct {
	PlannedCost   *money.Money `json:"plannedCost"`   // USD; nil when the activity has none
	ActivityType  *FactRef     `json:"activityType"`  // nil when the activity has no type
	BusinessUnits []FactOption `json:"businessUnits"` // the plan's business units; never nil
}

// FactRef is a Settings record as the rules saw it: its id and name at submission.
type FactRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// FactOption is a Shell attribute option as the rules saw it: its id and label at submission.
type FactOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// RoundLevel is a round's copy of one workflow level, with its approvers' names at submission. Feature 017
// keeps in it what the rules decided for the level and who was found to approve it (data-model §4).
type RoundLevel struct {
	Number    int           `json:"number"`    // 1-based
	Name      string        `json:"name"`      // "" means "Level <n>"
	Approvers []actor.Actor `json:"approvers"` // the resolved deciders; empty for skipped and automatic levels
	// The level's rule, copied: its approver source, the text of how its approvers were found ("" for named
	// people, "line manager of Priya Raman", "role Approver"), its role and its mode.
	ApproverType string `json:"approverType"`
	Source       string `json:"source"`
	RoleID       string `json:"roleId"`
	RoleName     string `json:"roleName"`
	Mode         string `json:"mode"`
	// What the rules decided at submission (ResolutionDecide, ResolutionSkipped, ResolutionAutoApproved), and
	// why for the last two.
	Resolution string `json:"resolution"`
	RuleText   string `json:"ruleText"`
	// The people left out when the approvers were resolved, whether the backup approver became the approver,
	// whether nobody can approve but someone deciding on behalf, and the backup who can take the level on
	// escalation (nil when none can, data-model §3 steps 4–6).
	Excluded   []Excluded   `json:"excluded"`
	BackupUsed bool         `json:"backupUsed"`
	NoApprover bool         `json:"noApprover"`
	Backup     *actor.Actor `json:"backup"`
	// StartedAt is when the round reached the level; RemindedAt and EscalatedAt when the job reminded its
	// approvers and escalated it (017 FR-026 – FR-029). nil until then.
	StartedAt   *time.Time `json:"startedAt"`
	RemindedAt  *time.Time `json:"remindedAt"`
	EscalatedAt *time.Time `json:"escalatedAt"`
}

// The outcomes of a level at submission (RoundLevel.Resolution, 017 FR-010).
const (
	ResolutionDecide       = "decide"       // its approvers decide
	ResolutionSkipped      = "skipped"      // Not needed: its "Only needed when" condition did not match
	ResolutionAutoApproved = "autoApproved" // Approved automatically: its "Approve automatically when" matched
)

// Excluded is a person left out of a level when its approvers were resolved, and why (017 FR-011).
type Excluded struct {
	Person actor.Actor `json:"person"`
	Reason string      `json:"reason"` // ReasonInactive, ReasonCannotApprove, ReasonOwner or ReasonSubmitter
}

// The reasons a person is left out of a level (Excluded.Reason), in the order they are checked.
const (
	ReasonInactive      = "inactive"
	ReasonCannotApprove = "cannotApprove" // holds no role with approvals.decide (014)
	ReasonOwner         = "owner"         // owns the activity (014's access rule, 017 FR-018)
	ReasonSubmitter     = "submitter"     // is submitting it (014's access rule, 017 FR-018)
)

// The states of a level in a round's chain (ChainLevel.State).
const (
	LevelApproved     = "approved"
	LevelDeclined     = "declined"
	LevelWaiting      = "waiting"
	LevelNotReached   = "notReached"
	LevelNotDecided   = "notDecided"   // waiting when the round was recalled, restarted or cancelled
	LevelSkipped      = "skipped"      // 017: Not needed
	LevelAutoApproved = "autoApproved" // 017: Approved automatically
)

// ChainLevel is one level of a round as the Approval section shows it (ApprovalChainLevel), with what the
// rules decided, who was found to approve it and how far it got (017 FR-033).
type ChainLevel struct {
	Number        int            `json:"number"`
	Label         string         `json:"label"`
	Approvers     []PersonView   `json:"approvers"`
	State         string         `json:"state"`
	Decision      *Decision      `json:"decision"` // the decision that completed or declined it
	Mode          string         `json:"mode"`
	Source        string         `json:"source"`
	RuleText      string         `json:"ruleText"`
	Decisions     []Decision     `json:"decisions"` // every decision on the level, oldest first
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

// ExcludedView is a person left out of a level, and why (ApprovalExcludedPerson).
type ExcludedView struct {
	Person PersonView `json:"person"`
	Reason string     `json:"reason"`
}

// RuleFactsView is the values the rules used at submission (ApprovalRuleFacts).
type RuleFactsView struct {
	PlannedCost   *money.Money    `json:"plannedCost"`
	ActivityType  *errs.RecordRef `json:"activityType"`
	BusinessUnits []OptionRef     `json:"businessUnits"`
}

// RoundWorkflow is the workflow of a round as it was at submission.
type RoundWorkflow struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RoundView is a round as the API returns it (ApprovalRound).
type RoundView struct {
	ID           string             `json:"id"`
	Number       int                `json:"number"`
	Outcome      Outcome            `json:"outcome"`
	Workflow     RoundWorkflow      `json:"workflow"`
	SubmittedAt  time.Time          `json:"submittedAt"`
	SubmittedBy  actor.Actor        `json:"submittedBy"`
	Comment      string             `json:"comment"`
	EndedAt      *time.Time         `json:"endedAt"`
	CurrentLevel int                `json:"currentLevel"`
	LevelCount   int                `json:"levelCount"`
	Levels       []ChainLevel       `json:"levels"`
	History      []HistoryEntryView `json:"history"` // newest first
	// Feature 017: the values the rules used, the timings copied at submission, and whether it was approved
	// at submission.
	Facts          *RuleFactsView `json:"facts"`
	ReminderDays   *int           `json:"reminderDays"`
	EscalationDays *int           `json:"escalationDays"`
	AutoApproved   bool           `json:"autoApproved"`
}

// DeclineNote is the latest decline, shown while the activity is Declined (DeclineNote).
type DeclineNote struct {
	RoundNumber int           `json:"roundNumber"`
	Level       int           `json:"level"`
	LevelLabel  string        `json:"levelLabel"`
	Actor       actor.Actor   `json:"actor"`
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"`
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"`
	Capacity    Capacity      `json:"capacity"`
	// AsDelegateOf is the approver a delegate declined for.
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
}
