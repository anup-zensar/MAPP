package approval

import (
	"time"

	"mapp/internal/platform/actor"
)

type ActionKind string

const (
	ActionSubmitted    ActionKind = "submitted"
	ActionResubmitted  ActionKind = "resubmitted"
	ActionApproved     ActionKind = "approved"
	ActionDeclined     ActionKind = "declined"
	ActionRecalled     ActionKind = "recalled"
	ActionRestarted    ActionKind = "restarted"
	ActionCancelled    ActionKind = "cancelled"
	ActionAutoApproved ActionKind = "autoApproved"
	ActionSkipped      ActionKind = "skipped"
	ActionReminded     ActionKind = "reminded"
	ActionEscalated    ActionKind = "escalated"
	ActionSuperseded   ActionKind = "superseded"
)

type Capacity string

const (
	CapacityApprover Capacity = "approver"
	CapacityDelegate Capacity = "delegate"
	CapacityBackup   Capacity = "backup"
	CapacityOnBehalf Capacity = "onBehalf"
)

// Action is one approval history entry: written once, never changed or deleted, so it has no record.Meta.
type Action struct {
	ID           string        `json:"id"`
	RoundID      string        `json:"roundId"`
	RoundNumber  int           `json:"roundNumber"`
	SubjectKind  SubjectKind   `json:"subjectKind"`
	SubjectID    string        `json:"subjectId"`
	Action       ActionKind    `json:"action"`
	Level        int           `json:"level"`
	LevelName    string        `json:"levelName"`
	Approvers    []actor.Actor `json:"approvers"`
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Changes      []ChangeText  `json:"changes"`
	Capacity     Capacity      `json:"capacity"`
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"`
	Rule         string        `json:"rule"`
	EscalatedTo  *actor.Actor  `json:"escalatedTo"`
	WaitedDays   int           `json:"waitedDays"`
	SelfApproval bool          `json:"selfApproval"`
}

func (a Action) subjectKind() SubjectKind {
	if a.SubjectKind == "" {
		return KindActivity
	}
	return a.SubjectKind
}

type ChangeText struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

type Decision struct {
	Action       string        `json:"action"`
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Capacity     Capacity      `json:"capacity"`
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"`
	SelfApproval bool          `json:"selfApproval"`
}

type ChangeView struct {
	Label   string `json:"label"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
	Text    string `json:"text"`
}

type HistoryEntryView struct {
	ID           string        `json:"id"`
	RoundNumber  int           `json:"roundNumber"`
	Level        int           `json:"level"`
	LevelLabel   string        `json:"levelLabel"`
	Approvers    []PersonView  `json:"approvers"`
	Actor        actor.Actor   `json:"actor"`
	OnBehalfOf   []actor.Actor `json:"onBehalfOf"`
	Action       string        `json:"action"`
	At           time.Time     `json:"at"`
	Comment      string        `json:"comment"`
	Changes      []ChangeView  `json:"changes"`
	Capacity     Capacity      `json:"capacity"`
	AsDelegateOf *actor.Actor  `json:"asDelegateOf"`
	Rule         string        `json:"rule"`
	EscalatedTo  *actor.Actor  `json:"escalatedTo"`
	WaitedDays   int           `json:"waitedDays"`
	SelfApproval bool          `json:"selfApproval"`
}
