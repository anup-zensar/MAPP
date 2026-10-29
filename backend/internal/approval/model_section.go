package approval

import (
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

type ActionAvailability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

type Viewer struct {
	IsOwner            bool         `json:"isOwner"`
	IsCurrentApprover  bool         `json:"isCurrentApprover"`
	OnBehalfOf         []PersonView `json:"onBehalfOf"`
	CanDecide          bool         `json:"canDecide"`
	Capacity           *Capacity    `json:"capacity"`
	AsDelegateOf       []PersonView `json:"asDelegateOf"`
	CannotDecideReason string       `json:"cannotDecideReason"`
	IsSubmitter        bool         `json:"isSubmitter"`
}

type ActionSet struct {
	Submit   ActionAvailability `json:"submit"`
	Resubmit ActionAvailability `json:"resubmit"`
	Approve  ActionAvailability `json:"approve"`
	Decline  ActionAvailability `json:"decline"`
	Recall   ActionAvailability `json:"recall"`
}

type SubjectView struct {
	Kind          SubjectKind `json:"kind"`
	Label         string      `json:"label"`
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Status        string      `json:"status"`
	DisplayStatus string      `json:"displayStatus"`
	Version       int         `json:"version,omitempty"`
}

// Section is a record's approval section, whatever its record type (contract ApprovalSection).
type Section struct {
	Subject        SubjectView  `json:"subject"`
	OwnerID        string       `json:"ownerId"`
	Owner          *PersonView  `json:"owner"`
	Workflow       *WorkflowRef `json:"workflow"`
	UpcomingLevels []LevelView  `json:"upcomingLevels"`
	Rounds         []RoundView  `json:"rounds"`
	LatestDecline  *DeclineNote `json:"latestDecline"`
	Viewer         Viewer       `json:"viewer"`
	Actions        ActionSet    `json:"actions"`
}

type ActivityApproval struct {
	ActivityID      string       `json:"activityId"`
	ActivityName    string       `json:"activityName"`
	ActivityVersion int          `json:"activityVersion"`
	Status          string       `json:"status"`
	DisplayStatus   string       `json:"displayStatus"`
	OwnerID         string       `json:"ownerId"`
	Owner           *PersonView  `json:"owner"`
	Workflow        *WorkflowRef `json:"workflow"`
	UpcomingLevels  []LevelView  `json:"upcomingLevels"`
	Rounds          []RoundView  `json:"rounds"`
	LatestDecline   *DeclineNote `json:"latestDecline"`
	Viewer          Viewer       `json:"viewer"`
	Actions         ActionSet    `json:"actions"`
	OverPlannedCost []Overage    `json:"overPlannedCost"`
}

// Overage is an activity whose line items exceed its planned cost (contract ActivityOverage).
type Overage struct {
	Activity errs.RecordRef `json:"activity"`
	OverBy   money.Money    `json:"overBy"`
}

func activityApproval(sec Section) ActivityApproval {
	return ActivityApproval{
		ActivityID: sec.Subject.ID, ActivityName: sec.Subject.Name, ActivityVersion: sec.Subject.Version,
		Status: sec.Subject.Status, DisplayStatus: sec.Subject.DisplayStatus, OwnerID: sec.OwnerID, Owner: sec.Owner,
		Workflow: sec.Workflow, UpcomingLevels: sec.UpcomingLevels, Rounds: sec.Rounds, LatestDecline: sec.LatestDecline,
		Viewer: sec.Viewer, Actions: sec.Actions, OverPlannedCost: []Overage{},
	}
}

type SubmitProblem struct {
	Field      string `json:"field"`
	Group      string `json:"group"`
	LineItemID string `json:"lineItemId"`
	Message    string `json:"message"`
}

type Submission struct {
	ActivityID            string            `json:"activityId"`
	ActivityName          string            `json:"activityName"`
	ActivityVersion       int               `json:"activityVersion"`
	Kind                  string            `json:"kind"`
	Available             bool              `json:"available"`
	Reason                string            `json:"reason"`
	RoundNumber           int               `json:"roundNumber"`
	Ready                 bool              `json:"ready"`
	Problems              []SubmitProblem   `json:"problems"`
	Workflow              *WorkflowRef      `json:"workflow"`
	Levels                []LevelView       `json:"levels"`
	UnassignedPlannedCost *money.Money      `json:"unassignedPlannedCost"`
	Outcomes              []SubmissionLevel `json:"outcomes"`
	ApprovedAutomatically bool              `json:"approvedAutomatically"`
	Replaces              *ReplacedRound    `json:"replaces"`
}

// ReplacedRound is the open round a re-submission would end.
type ReplacedRound struct {
	RoundNumber int    `json:"roundNumber"`
	Level       int    `json:"level"`
	LevelLabel  string `json:"levelLabel"`
}

type SubmissionLevel struct {
	Number     int            `json:"number"`
	Label      string         `json:"label"`
	Outcome    string         `json:"outcome"`
	Approvers  []PersonView   `json:"approvers"`
	Source     string         `json:"source"`
	Mode       string         `json:"mode"`
	RuleText   string         `json:"ruleText"`
	Excluded   []ExcludedView `json:"excluded"`
	BackupUsed bool           `json:"backupUsed"`
	NoApprover bool           `json:"noApprover"`
	Advice     string         `json:"advice"`
}

type SubmitInput struct {
	Version *int   `json:"version"`
	Comment string `json:"comment"`
}

type DecisionInput struct {
	RoundID string `json:"roundId"`
	Level   *int   `json:"level"`
	Comment string `json:"comment"`
}

// SubjectDecision is an approval (Action ActionApproved) or a decline (ActionDeclined) of a record's open round.
type SubjectDecision struct {
	Action ActionKind
	DecisionInput
}

type RecallInput struct {
	RoundID string `json:"roundId"`
	Reason  string `json:"reason"`
}
