package approvals

import (
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

// RowRound is the round of an Approvals row (ApprovalRowRound).
type RowRound struct {
	ID           string       `json:"id"`
	Number       int          `json:"number"`
	Outcome      Outcome      `json:"outcome"`
	CurrentLevel int          `json:"currentLevel"`
	LevelCount   int          `json:"levelCount"`
	LevelLabel   string       `json:"levelLabel"`
	Approvers    []PersonView `json:"approvers"`
	// 017: the current level's mode and, on "all must approve" levels, how many have approved of how many.
	Mode          string `json:"mode"`
	ApprovedCount int    `json:"approvedCount"`
	RequiredCount int    `json:"requiredCount"`
}

// MyDecision is the signed-in person's latest decision on an activity (MyDecision).
type MyDecision struct {
	Action      string    `json:"action"`
	At          time.Time `json:"at"`
	RoundNumber int       `json:"roundNumber"`
	Level       int       `json:"level"`
}

// RowActivity is the activity of an Approvals row (ApprovalRow.activity).
type RowActivity struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	DisplayStatus string `json:"displayStatus"`
}

// Row is one activity of the Approvals page (ApprovalRow).
type Row struct {
	Activity        RowActivity      `json:"activity"`
	Path            []errs.RecordRef `json:"path"`
	Owner           *PersonView      `json:"owner"`
	PlannedCost     *money.Money     `json:"plannedCost"` // USD
	Workflow        WorkflowRef      `json:"workflow"`
	Round           RowRound         `json:"round"`
	SubmittedAt     time.Time        `json:"submittedAt"`
	WaitingSince    *time.Time       `json:"waitingSince"`
	StartDate       *date.Date       `json:"startDate"`
	StartDatePassed bool             `json:"startDatePassed"`
	MyDecision      *MyDecision      `json:"myDecision"` // decided tab only
	// Feature 017 (FR-031): in Waiting for me, the approvers the person acts for as delegate; the backup
	// approver of an escalated level; whether the level is overdue; the whole days it has waited (nil without
	// an open round); and whether nobody waiting can act.
	AsDelegateOf []PersonView `json:"asDelegateOf"`
	EscalatedTo  *PersonView  `json:"escalatedTo"`
	Overdue      bool         `json:"overdue"`
	WaitingDays  *int         `json:"waitingDays"`
	NoApprover   bool         `json:"noApprover"`
}

// RowPage is the answer of GET /api/v1/approvals?tab= (ApprovalRowPage).
type RowPage struct {
	Items []Row `json:"items"`
	Total int   `json:"total"`
}

// The tabs of the Approvals page.
const (
	TabWaiting = "waiting"
	TabOpen    = "open"
	TabDecided = "decided"
)

// Counts are the summary cards of the Approvals page (ApprovalCounts).
type Counts struct {
	WaitingForMe      int `json:"waitingForMe"`
	AllOpen           int `json:"allOpen"`
	ApprovedThisMonth int `json:"approvedThisMonth"`
	DeclinedThisMonth int `json:"declinedThisMonth"`
}

// MissingFieldRef is an empty needed-to-submit field of the previewed activity (MissingFieldRef).
type MissingFieldRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// PreviewLineItem is a line item of the preview (PreviewLineItem).
type PreviewLineItem struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	PlannedCost money.Money `json:"plannedCost"` // in its own currency
	// PlannedCostDisplay is its USD planned cost as Activities.LineItems gives it; Preview answers it in the
	// reader's display currency (D-213). Nil when its currency has no rate.
	PlannedCostDisplay *money.Money `json:"plannedCostDisplay"`
	SpendType          string       `json:"spendType"`
	GLCode             string       `json:"glCode"`
	FinanceBudgetItem  string       `json:"financeBudgetItem"`
}

// PreviewActivity is the activity of the preview (ApprovalPreview.activity).
type PreviewActivity struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	DisplayStatus string            `json:"displayStatus"`
	Missing       []MissingFieldRef `json:"missing"`
	Path          []errs.RecordRef  `json:"path"`
	Owner         *PersonView       `json:"owner"`
	StartDate     *date.Date        `json:"startDate"`
	EndDate       *date.Date        `json:"endDate"`
	Ongoing       bool              `json:"ongoing"`
	PlannedCost   *money.Money      `json:"plannedCost"`
	TypeName      string            `json:"typeName"`
	Description   string            `json:"description"`
}

// Preview is the answer of GET /api/v1/approvals/activities/{activityId} (ApprovalPreview).
type Preview struct {
	Activity              PreviewActivity   `json:"activity"`
	LineItems             []PreviewLineItem `json:"lineItems"`
	UnassignedPlannedCost *money.Money      `json:"unassignedPlannedCost"`
	Approval              ActivityApproval  `json:"approval"`
}
