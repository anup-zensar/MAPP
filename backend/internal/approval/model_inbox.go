package approval

import (
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

type RowRound struct {
	ID            string       `json:"id"`
	Number        int          `json:"number"`
	Outcome       Outcome      `json:"outcome"`
	CurrentLevel  int          `json:"currentLevel"`
	LevelCount    int          `json:"levelCount"`
	LevelLabel    string       `json:"levelLabel"`
	Approvers     []PersonView `json:"approvers"`
	Mode          string       `json:"mode"`
	ApprovedCount int          `json:"approvedCount"`
	RequiredCount int          `json:"requiredCount"`
}

type MyDecision struct {
	Action      string    `json:"action"`
	At          time.Time `json:"at"`
	RoundNumber int       `json:"roundNumber"`
	Level       int       `json:"level"`
}

type RowActivity struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	DisplayStatus string `json:"displayStatus"`
}

// Row is one record of the Approvals page; Activity is set for activity rows only.
type Row struct {
	Subject         SubjectView      `json:"subject"`
	Activity        *RowActivity     `json:"activity"`
	Path            []errs.RecordRef `json:"path"`
	Owner           *PersonView      `json:"owner"`
	PlannedCost     *money.Money     `json:"plannedCost"`
	Workflow        WorkflowRef      `json:"workflow"`
	Round           RowRound         `json:"round"`
	SubmittedAt     time.Time        `json:"submittedAt"`
	WaitingSince    *time.Time       `json:"waitingSince"`
	StartDate       *date.Date       `json:"startDate"`
	StartDatePassed bool             `json:"startDatePassed"`
	MyDecision      *MyDecision      `json:"myDecision"`
	AsDelegateOf    []PersonView     `json:"asDelegateOf"`
	EscalatedTo     *PersonView      `json:"escalatedTo"`
	Overdue         bool             `json:"overdue"`
	WaitingDays     *int             `json:"waitingDays"`
	NoApprover      bool             `json:"noApprover"`
}

type RowPage struct {
	Items []Row `json:"items"`
	Total int   `json:"total"`
}

const (
	TabWaiting = "waiting"
	TabOpen    = "open"
	TabDecided = "decided"
)

type Counts struct {
	WaitingForMe      int `json:"waitingForMe"`
	AllOpen           int `json:"allOpen"`
	ApprovedThisMonth int `json:"approvedThisMonth"`
	DeclinedThisMonth int `json:"declinedThisMonth"`
}

type MissingFieldRef struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type PreviewLineItem struct {
	ID                 string       `json:"id"`
	Name               string       `json:"name"`
	PlannedCost        money.Money  `json:"plannedCost"`
	PlannedCostDisplay *money.Money `json:"plannedCostDisplay"`
	SpendType          string       `json:"spendType"`
	GLCode             string       `json:"glCode"`
	FinanceBudgetItem  string       `json:"financeBudgetItem"`
}

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

type Preview struct {
	Activity              PreviewActivity   `json:"activity"`
	LineItems             []PreviewLineItem `json:"lineItems"`
	UnassignedPlannedCost *money.Money      `json:"unassignedPlannedCost"`
	Approval              ActivityApproval  `json:"approval"`
}
