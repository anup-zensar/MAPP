package purchaserequest

import (
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/money"
)

type RowLine struct {
	ID            string            `json:"id"`
	LineItem      LineItemRef       `json:"lineItem"`
	LineItemOwner *people.PersonRef `json:"lineItemOwner"`
	HXKey         string            `json:"hxKey"`
	Amount        money.Money       `json:"amount"`
	DeliveryDate  date.Date         `json:"deliveryDate"`
	SentOn        *date.Date        `json:"sentOn"`
	PONumber      string            `json:"poNumber"`
	POLine        string            `json:"poLine"`
	POLineStatus  string            `json:"poLineStatus"`
	Invoiced      money.Money       `json:"invoiced"`
}

// Row is one purchase request of the Purchase requests page (contract PurchaseRequestRow).
type Row struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Vendor           string            `json:"vendor"`
	Status           Status            `json:"status"`
	ChangePending    bool              `json:"changePending"`
	AmountDisplay    money.Money       `json:"amountDisplay"`
	InvoicedDisplay  money.Money       `json:"invoicedDisplay"`
	Owner            *people.PersonRef `json:"owner"`
	CreatedAt        time.Time         `json:"createdAt"`
	ApprovalWorkflow *WorkflowRef      `json:"approvalWorkflow"`
	GSAPPRNumber     string            `json:"gsapPrNumber"`
	WaitingForGSAP   bool              `json:"waitingForGsap"`
	Plan             errs.RecordRef    `json:"plan"`
	Lines            []RowLine         `json:"lines"`
}

type VendorGroupView struct {
	Vendor           string      `json:"vendor"`
	Label            string      `json:"label"`
	PurchaseRequests int         `json:"purchaseRequests"`
	AmountDisplay    money.Money `json:"amountDisplay"`
	InvoicedDisplay  money.Money `json:"invoicedDisplay"`
}

type ListPage struct {
	Items           []Row             `json:"items"`
	Total           int               `json:"total"`
	Page            int               `json:"page"`
	PageSize        int               `json:"pageSize"`
	Vendors         []VendorGroupView `json:"vendors"`
	PurchaseSupport bool              `json:"purchaseSupport"`
}

type CloseLine struct {
	LineID       string      `json:"lineId"`
	LineItemID   string      `json:"lineItemId"`
	HXKey        string      `json:"hxKey"`
	Amount       money.Money `json:"amount"`
	Invoiced     money.Money `json:"invoiced"`
	ClosedAmount money.Money `json:"closedAmount"`
	Released     money.Money `json:"released"`
}

type ClosePreview struct {
	PurchaseRequestID string      `json:"purchaseRequestId"`
	Version           int         `json:"version"`
	Available         bool        `json:"available"`
	Reason            string      `json:"reason"`
	Lines             []CloseLine `json:"lines"`
	ReleasedDisplay   money.Money `json:"releasedDisplay"`
}

type LineItemPRRow struct {
	PurchaseRequestID        string            `json:"purchaseRequestId"`
	Name                     string            `json:"name"`
	Status                   Status            `json:"status"`
	ChangePending            bool              `json:"changePending"`
	LineID                   string            `json:"lineId"`
	Amount                   money.Money       `json:"amount"`
	AmountInLineItemCurrency money.Money       `json:"amountInLineItemCurrency"`
	SentOn                   *date.Date        `json:"sentOn"`
	PONumber                 string            `json:"poNumber"`
	Owner                    *people.PersonRef `json:"owner"`
}

// LineItemSection is a line item's Purchase requests section (contract LineItemPurchaseRequests).
type LineItemSection struct {
	LineItemID         string             `json:"lineItemId"`
	HXKey              string             `json:"hxKey"`
	PlannedCost        money.Money        `json:"plannedCost"`
	Requested          money.Money        `json:"requested"`
	AvailableToRequest money.Money        `json:"availableToRequest"`
	PurchaseRequests   []LineItemPRRow    `json:"purchaseRequests"`
	Create             authz.Availability `json:"create"`
	Notice             string             `json:"notice"`
}

type ActivityLineItem struct {
	LineItemID         string      `json:"lineItemId"`
	Name               string      `json:"name"`
	Requested          money.Money `json:"requested"`
	AvailableToRequest money.Money `json:"availableToRequest"`
	PurchaseRequests   int         `json:"purchaseRequests"`
}

// ActivitySection is an activity's Purchase requests section (contract ActivityPurchaseRequests).
type ActivitySection struct {
	ActivityID string             `json:"activityId"`
	Requested  money.Money        `json:"requested"`
	LineItems  []ActivityLineItem `json:"lineItems"`
}
