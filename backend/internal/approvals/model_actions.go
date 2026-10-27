package approvals

import (
	"time"

	"mapp/internal/platform/actor"
)

// ActionKind is what an approval history entry records.
type ActionKind string

// The approval history actions.
const (
	ActionSubmitted   ActionKind = "submitted"
	ActionResubmitted ActionKind = "resubmitted"
	ActionApproved    ActionKind = "approved"
	ActionDeclined    ActionKind = "declined"
	ActionRecalled    ActionKind = "recalled"
	ActionRestarted   ActionKind = "restarted"
	ActionCancelled   ActionKind = "cancelled"
	// Feature 017: what MAPP records by itself (actor.System).
	ActionAutoApproved ActionKind = "autoApproved" // a level approved automatically at submission
	ActionSkipped      ActionKind = "skipped"      // a level not needed at submission
	ActionReminded     ActionKind = "reminded"     // the level's approvers were reminded
	ActionEscalated    ActionKind = "escalated"    // the level was escalated to its backup approver
)

// Capacity is the capacity a decision was made in (017 FR-019, data-model §6).
type Capacity string

// The capacities of a decision. Decisions stored before 017 have none ("").
const (
	CapacityApprover Capacity = "approver" // a resolved approver of the level
	CapacityDelegate Capacity = "delegate" // the current delegate of a waiting approver
	CapacityBackup   Capacity = "backup"   // the level's backup approver, after escalation
	CapacityOnBehalf Capacity = "onBehalf" // someone allowed to decide on behalf of the waiting approvers
)

// Action is one approval history entry: one step of a round (table "approval_actions", data model §3).
// Written once, never changed or deleted, so it has no record.Meta (008-P4).
type Action struct {
	ID          string        `json:"id"` // "AH-1001"; its number orders entries of the same millisecond
	RoundID     string        `json:"roundId"`
	RoundNumber int           `json:"roundNumber"`
	ActivityID  string        `json:"activityId"` // redundant with the round, for direct lookups
	Action      ActionKind    `json:"action"`
	Level       int           `json:"level"`      // the level concerned; 0 for a restart of an approved activity
	LevelName   string        `json:"levelName"`  // copied from the round level; "" when Level is 0
	Approvers   []actor.Actor `json:"approvers"`  // the level's designated approvers; empty when Level is 0
	Actor       actor.Actor   `json:"actor"`      // who acted (cancelled: who deleted the activity)
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"` // approved/declined by someone else than the approvers: the approvers
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"` // submit, approve, decline comment or recall reason
	Changes     []ChangeText  `json:"changes"` // restarted only: the changed values
	// Feature 017 (data-model §5): the capacity of a decision and, for a delegate's, the approver they acted
	// for; the rule of an autoApproved or skipped entry; the backup approver of an escalation; and the whole
	// days a reminded or escalated level had waited.
	Capacity     Capacity     `json:"capacity"`
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
	Rule         string       `json:"rule"`
	EscalatedTo  *actor.Actor `json:"escalatedTo"`
	WaitedDays   int          `json:"waitedDays"`
}

// ChangeText is one changed value of a restart, copied from the saving feature's log entry.
type ChangeText struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// Decision is a decision on one level of a round (ApprovalDecision).
type Decision struct {
	Action       string        `json:"action"` // "approved" or "declined"
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Capacity     Capacity      `json:"capacity"`     // "" for decisions recorded before 017
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"` // delegate decisions only
}

// ChangeView is one changed value of a restart, with its sentence (ApprovalChange).
type ChangeView struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
	Text    string `json:"text"` // "<label> changed from <old> to <new>", "<label> added: <new>", …
}

// HistoryEntryView is one approval history entry as the API shows it (ApprovalHistoryEntry).
type HistoryEntryView struct {
	ID          string        `json:"id"`
	RoundNumber int           `json:"roundNumber"`
	Level       int           `json:"level"`
	LevelLabel  string        `json:"levelLabel"`
	Approvers   []PersonView  `json:"approvers"`
	Actor       actor.Actor   `json:"actor"`
	OnBehalfOf  []actor.Actor `json:"onBehalfOf"`
	Action      string        `json:"action"`
	At          time.Time     `json:"at"`
	Comment     string        `json:"comment"`
	Changes     []ChangeView  `json:"changes"`
	// Feature 017 (FR-034): a decision's capacity, the rule of an automatic entry, the backup approver of an
	// escalation and the days a reminded or escalated level had waited.
	Capacity     Capacity     `json:"capacity"`
	AsDelegateOf *actor.Actor `json:"asDelegateOf"`
	Rule         string       `json:"rule"`
	EscalatedTo  *actor.Actor `json:"escalatedTo"`
	WaitedDays   int          `json:"waitedDays"`
}
