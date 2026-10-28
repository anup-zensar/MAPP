package purchaserequest

import (
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

// The API types below follow the contract's schemas of the same name with "PurchaseRequest" in front.

type WorkflowRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type LineItemRef struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	ActivityID   string      `json:"activityId"`
	ActivityName string      `json:"activityName"`
	Currency     string      `json:"currency"`
	PlannedCost  money.Money `json:"plannedCost"`
	SwbsCode     string      `json:"swbsCode"`
	SwbsStatus   string      `json:"swbsStatus"`
	CompanyCode  string      `json:"companyCode"`
	CostCentre   string      `json:"costCentre"`
	GLCode       string      `json:"glCode"`
}

type LineAccess struct {
	Remove authz.Availability `json:"remove"`
}

type LineView struct {
	ID           string                   `json:"id"`
	Position     int                      `json:"position"`
	LineItem     LineItemRef              `json:"lineItem"`
	HXKey        string                   `json:"hxKey"`
	Currency     string                   `json:"currency"`
	Amount       money.Money              `json:"amount"`
	DeliveryDate date.Date                `json:"deliveryDate"`
	OverallLimit *money.Money             `json:"overallLimit"`
	Description  string                   `json:"description"`
	Attributes   settings.AttributeValues `json:"attributes"`
	SentOn       *date.Date               `json:"sentOn"`
	PONumber     string                   `json:"poNumber"`
	POLine       string                   `json:"poLine"`
	POLineStatus string                   `json:"poLineStatus"`
	PODate       *date.Date               `json:"poDate"`
	POValue      *money.Money             `json:"poValue"`
	Invoiced     money.Money              `json:"invoiced"`
	Changed      bool                     `json:"changed"`
	Missing      []settings.MissingField  `json:"missing"`
	Access       LineAccess               `json:"access"`
	GSAPPending  bool                     `json:"gsapPending"`
	Version      int                      `json:"version"`
}

type VersionLineView struct {
	ID           string       `json:"id"`
	LineItemID   string       `json:"lineItemId"`
	HXKey        string       `json:"hxKey"`
	Amount       money.Money  `json:"amount"`
	DeliveryDate date.Date    `json:"deliveryDate"`
	OverallLimit *money.Money `json:"overallLimit"`
	Description  string       `json:"description"`
}

type VersionView struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Owner         *people.PersonRef        `json:"owner"`
	MoAApprover   *people.PersonRef        `json:"moaApprover"`
	Attributes    settings.AttributeValues `json:"attributes"`
	AmountDisplay money.Money              `json:"amountDisplay"`
	Lines         []VersionLineView        `json:"lines"`
	ApprovedAt    time.Time                `json:"approvedAt"`
}

type RoutingView struct {
	ID         string      `json:"id"`
	Type       RoutingType `json:"type"`
	RoutedAt   time.Time   `json:"routedAt"`
	RoutedBy   actor.Actor `json:"routedBy"`
	Recipients []string    `json:"recipients"`
	LineCount  int         `json:"lineCount"`
}

// Problem is one thing that stops a submission (FR-021); Group is purchaseRequest, line, activity or lineItem.
type Problem struct {
	Field    string `json:"field"`
	Group    string `json:"group"`
	RecordID string `json:"recordId"`
	Message  string `json:"message"`
}

type Readiness struct {
	Ready    bool      `json:"ready"`
	Problems []Problem `json:"problems"`
}

type AccessView struct {
	Edit          authz.Availability `json:"edit"`
	Delete        authz.Availability `json:"delete"`
	Submit        authz.Availability `json:"submit"`
	Resubmit      authz.Availability `json:"resubmit"`
	SubmitChanges authz.Availability `json:"submitChanges"`
	Recall        authz.Availability `json:"recall"`
	Close         authz.Availability `json:"close"`
	Resend        authz.Availability `json:"resend"`
	AddLine       authz.Availability `json:"addLine"`
	GSAPOrder     authz.Availability `json:"gsapOrder"`
}

// View is a purchase request's details page (contract PurchaseRequest).
type View struct {
	ID                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	Owner                *people.PersonRef        `json:"owner"`
	MoAApprover          *people.PersonRef        `json:"moaApprover"`
	ApprovalWorkflow     *WorkflowRef             `json:"approvalWorkflow"`
	ModificationWorkflow *WorkflowRef             `json:"modificationWorkflow"`
	Status               Status                   `json:"status"`
	ChangePending        bool                     `json:"changePending"`
	Plan                 errs.RecordRef           `json:"plan"`
	Path                 []errs.RecordRef         `json:"path"`
	CompanyCode          string                   `json:"companyCode"`
	CostCentres          []string                 `json:"costCentres"`
	HXKeys               []string                 `json:"hxKeys"`
	Attributes           settings.AttributeValues `json:"attributes"`
	Vendor               string                   `json:"vendor"`
	AmountDisplay        money.Money              `json:"amountDisplay"`
	InvoicedDisplay      money.Money              `json:"invoicedDisplay"`
	GSAPPRNumber         string                   `json:"gsapPrNumber"`
	GSAPPRDate           *date.Date               `json:"gsapPrDate"`
	GSAPSource           string                   `json:"gsapSource"`
	WaitingForGSAP       bool                     `json:"waitingForGsap"`
	Lines                []LineView               `json:"lines"`
	ApprovedVersion      *VersionView             `json:"approvedVersion"`
	Routings             []RoutingView            `json:"routings"`
	Readiness            Readiness                `json:"readiness"`
	Missing              []settings.MissingField  `json:"missing"`
	Notice               string                   `json:"notice"`
	Access               AccessView               `json:"access"`
	record.Meta
}

type Defaults struct {
	Name        string           `json:"name"`
	Owner       people.PersonRef `json:"owner"`
	Plan        errs.RecordRef   `json:"plan"`
	CompanyCode string           `json:"companyCode"`
	Line        DefaultLine      `json:"line"`
}

type DefaultLine struct {
	LineItem     LineItemRef `json:"lineItem"`
	HXKey        string      `json:"hxKey"`
	Currency     string      `json:"currency"`
	Amount       money.Money `json:"amount"`
	OverallLimit money.Money `json:"overallLimit"`
}

type LineItemOption struct {
	LineItem           LineItemRef      `json:"lineItem"`
	Plan               errs.RecordRef   `json:"plan"`
	Path               []errs.RecordRef `json:"path"`
	HXKey              string           `json:"hxKey"`
	AvailableToRequest money.Money      `json:"availableToRequest"`
	Available          bool             `json:"available"`
	Reason             string           `json:"reason"`
}

type OptionPage struct {
	Items []LineItemOption `json:"items"`
	Total int              `json:"total"`
}

type ConfigurationView struct {
	PurchaseSupport                 bool               `json:"purchaseSupport"`
	LinesOnApprovedPurchaseRequests LinesOnApproved    `json:"linesOnApprovedPurchaseRequests"`
	Version                         int                `json:"version"`
	UpdatedAt                       time.Time          `json:"updatedAt"`
	UpdatedBy                       actor.Actor        `json:"updatedBy"`
	CanChange                       authz.Availability `json:"canChange"`
}

// File is a PR form to download.
type File struct {
	Name        string
	ContentType string
	Body        []byte
}
