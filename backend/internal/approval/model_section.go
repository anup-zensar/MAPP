package approval

import (
	"mapp/internal/platform/money"
)

// ActionAvailability says whether an approval action can be used, and why not (ApprovalActionAvailability).
type ActionAvailability struct {
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
}

// Viewer is the signed-in person's relation to the activity (ApprovalViewer). Feature 017 adds whether and
// in which capacity they may decide the current level, and why not (data-model §6).
type Viewer struct {
	IsOwner            bool         `json:"isOwner"`
	IsCurrentApprover  bool         `json:"isCurrentApprover"`
	OnBehalfOf         []PersonView `json:"onBehalfOf"` // the waiting approvers, for capacity onBehalf only
	CanDecide          bool         `json:"canDecide"`
	Capacity           *Capacity    `json:"capacity"`     // nil when they may not decide
	AsDelegateOf       []PersonView `json:"asDelegateOf"` // capacity delegate: every approver they act for
	CannotDecideReason string       `json:"cannotDecideReason"`
	IsSubmitter        bool         `json:"isSubmitter"`
}

// ActionSet is the availability of each approval action (ActivityApproval.actions).
type ActionSet struct {
	Submit   ActionAvailability `json:"submit"`
	Resubmit ActionAvailability `json:"resubmit"`
	Approve  ActionAvailability `json:"approve"`
	Decline  ActionAvailability `json:"decline"`
	Recall   ActionAvailability `json:"recall"`
}

// ActivityApproval is an activity's Approval section (ActivityApproval).
type ActivityApproval struct {
	ActivityID      string       `json:"activityId"`
	ActivityName    string       `json:"activityName"`
	ActivityVersion int          `json:"activityVersion"` // sent back with Submit
	Status          string       `json:"status"`
	DisplayStatus   string       `json:"displayStatus"`
	OwnerID         string       `json:"ownerId"`
	Owner           *PersonView  `json:"owner"` // the owner with their name now; nil without one (D-214)
	Workflow        *WorkflowRef `json:"workflow"`
	UpcomingLevels  []LevelView  `json:"upcomingLevels"`
	Rounds          []RoundView  `json:"rounds"` // newest first
	LatestDecline   *DeclineNote `json:"latestDecline"`
	Viewer          Viewer       `json:"viewer"`
	Actions         ActionSet    `json:"actions"`
}

// SubmitProblem is one problem that stops a submission (SubmitProblem).
type SubmitProblem struct {
	Field      string `json:"field"`
	Group      string `json:"group"` // "activity" or "lineItem"
	LineItemID string `json:"lineItemId"`
	Message    string `json:"message"`
}

// Submission is the answer of GET /api/v1/activities/{activityId}/approval/submission (ApprovalSubmission).
type Submission struct {
	ActivityID            string          `json:"activityId"`
	ActivityName          string          `json:"activityName"`
	ActivityVersion       int             `json:"activityVersion"`
	Kind                  string          `json:"kind"` // "submit" or "resubmit"
	Available             bool            `json:"available"`
	Reason                string          `json:"reason"`
	RoundNumber           int             `json:"roundNumber"`
	Ready                 bool            `json:"ready"`
	Problems              []SubmitProblem `json:"problems"`
	Workflow              *WorkflowRef    `json:"workflow"`
	Levels                []LevelView     `json:"levels"`
	UnassignedPlannedCost *money.Money    `json:"unassignedPlannedCost"` // a positive one, as a note
	// Outcomes say what submitting would do with each level (017 FR-013); empty without a usable workflow.
	Outcomes []SubmissionLevel `json:"outcomes"`
	// ApprovedAutomatically: no level needs a decision, so confirming approves the activity (017 FR-014).
	ApprovedAutomatically bool `json:"approvedAutomatically"`
}

// SubmissionLevel is what a submission would do with one level (ApprovalSubmissionLevel).
type SubmissionLevel struct {
	Number     int            `json:"number"`
	Label      string         `json:"label"`
	Outcome    string         `json:"outcome"` // ResolutionDecide, ResolutionSkipped or ResolutionAutoApproved
	Approvers  []PersonView   `json:"approvers"`
	Source     string         `json:"source"`
	Mode       string         `json:"mode"`
	RuleText   string         `json:"ruleText"`
	Excluded   []ExcludedView `json:"excluded"`
	BackupUsed bool           `json:"backupUsed"`
	NoApprover bool           `json:"noApprover"`
	Advice     string         `json:"advice"`
}

// SubmitInput is the body of POST …/approval/submit (SubmitRequest).
type SubmitInput struct {
	Version *int   `json:"version"` // the activity version the person saw
	Comment string `json:"comment"`
}

// DecisionInput is the body of POST …/approval/approve and …/decline (DecisionRequest).
type DecisionInput struct {
	RoundID string `json:"roundId"`
	Level   *int   `json:"level"` // the level the person is deciding
	Comment string `json:"comment"`
}

// RecallInput is the body of POST …/approval/recall (RecallRequest).
type RecallInput struct {
	RoundID string `json:"roundId"`
	Reason  string `json:"reason"`
}
