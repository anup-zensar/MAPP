package purchaserequest

import (
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
)

type RoutingType string

const (
	RoutingOriginal     RoutingType = "original"
	RoutingModification RoutingType = "modification"
	RoutingResend       RoutingType = "resend"
	RoutingClose        RoutingType = "close"
)

// Routing is one hand-off of a purchase request to the requisitioner inside MAPP, kept as routed and never changed.
type Routing struct {
	ID                string      `json:"id"`
	PurchaseRequestID string      `json:"purchaseRequestId"`
	Type              RoutingType `json:"type"`
	RoutedAt          time.Time   `json:"routedAt"`
	RoutedBy          actor.Actor `json:"routedBy"`
	Recipients        []string    `json:"recipients"`
	Form              []FormRow   `json:"form"`
}

// FormRow is one row of the PR form, as routed (spec "Requisitioner Attachment").
type FormRow struct {
	PurchaseRequestID   string       `json:"purchaseRequestId"`
	PurchaseRequestName string       `json:"purchaseRequestName"`
	LineID              string       `json:"lineId"`
	HXKey               string       `json:"hxKey"`
	ShortText           string       `json:"shortText"`
	Vendor              string       `json:"vendor"`
	VendorCode          string       `json:"vendorCode"`
	CostCentre          string       `json:"costCentre"`
	SwbsCode            string       `json:"swbsCode"`
	GLCode              string       `json:"glCode"`
	Amount              money.Amount `json:"amount"`
	Currency            string       `json:"currency"`
	DeliveryDate        date.Date    `json:"deliveryDate"`
	MoAApproverName     string       `json:"moaApproverName"`
	MoAApproverEmail    string       `json:"moaApproverEmail"`
	Changed             bool         `json:"changed"`
}

// Configuration is Settings › Purchase requests; without a stored row it is the defaults.
type Configuration struct {
	PurchaseSupport bool            `json:"purchaseSupport"`
	LinesOnApproved LinesOnApproved `json:"linesOnApprovedPurchaseRequests"`
	record.Meta
}

func DefaultConfiguration() Configuration {
	return Configuration{PurchaseSupport: true, LinesOnApproved: WithModificationApproval,
		Meta: record.Meta{Version: 1, CreatedBy: actor.System, UpdatedBy: actor.System}}
}

// ViewScope is where the person acting sees purchase requests: their plans.view and purchaseRequests.list scopes.
type ViewScope struct {
	Plans authz.Scope
	List  authz.Scope
}

type Claim struct {
	LineID            string
	PurchaseRequestID string
	PurchaseRequest   string
	LineItemID        string
	Status            Status
	Amount            money.Money
	Approved          *money.Money
	RemovedByChange   bool
}
