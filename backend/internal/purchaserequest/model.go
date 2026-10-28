// Package purchaserequest holds purchase requests (feature 032): requests raised from line items and reserved
// on them, approved through the approval engine, routed to the requisitioner and answered by GSAP (033).
package purchaserequest

import (
	"bytes"
	"strings"
	"time"

	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
)

type Status string

const (
	StatusCreated   Status = "created"
	StatusSubmitted Status = "submitted"
	StatusInReview  Status = "inReview"
	StatusDeclined  Status = "declined"
	StatusApproved  Status = "approved"
	StatusClosed    Status = "closed"
)

type LinesOnApproved string

const (
	NotAllowed               LinesOnApproved = "notAllowed"
	WithModificationApproval LinesOnApproved = "withModificationApproval"
	WithoutApproval          LinesOnApproved = "withoutApproval"
)

type RoundKind string

const (
	KindPurchaseRequest RoundKind = "purchaseRequest"
	KindModification    RoundKind = "modifiedPurchaseRequest"
)

type RoundEvent string

const (
	EventOpen       RoundEvent = "open"
	EventApproved   RoundEvent = "approved"
	EventDeclined   RoundEvent = "declined"
	EventRecalled   RoundEvent = "recalled"
	EventCancelled  RoundEvent = "cancelled"
	EventSuperseded RoundEvent = "superseded"
)

const (
	SourceSimulatedGSAP = "simulatedGsap"
	SourceGSAP          = "gsap"
)

type PurchaseRequest struct {
	ID                     string                   `json:"id"`
	PlanID                 string                   `json:"planId"`
	Name                   string                   `json:"name"`
	Description            string                   `json:"description"`
	OwnerID                string                   `json:"ownerId"`
	MoAApproverID          string                   `json:"moaApproverId"`
	WorkflowID             string                   `json:"workflowId"`
	ModificationWorkflowID string                   `json:"modificationWorkflowId"`
	Status                 Status                   `json:"status"`
	ChangePending          bool                     `json:"changePending"`
	Attributes             settings.AttributeValues `json:"attributes"`
	Vendor                 string                   `json:"vendor"`
	Approved               *Version                 `json:"approved"`
	GSAP                   GSAPHeader               `json:"gsap"`
	ClosedAt               *time.Time               `json:"closedAt"`
	Lines                  []Line                   `json:"lines"`
	record.Meta
}

type Line struct {
	ID                string                   `json:"id"`
	PurchaseRequestID string                   `json:"purchaseRequestId"`
	Position          int                      `json:"position"`
	LineItemID        string                   `json:"lineItemId"`
	Currency          string                   `json:"currency"`
	Amount            money.Amount             `json:"amount"`
	DeliveryDate      date.Date                `json:"deliveryDate"`
	OverallLimit      *money.Amount            `json:"overallLimit"`
	Description       string                   `json:"description"`
	Attributes        settings.AttributeValues `json:"attributes"`
	SentOn            *date.Date               `json:"sentOn"`
	PO                PurchaseOrderLine        `json:"po"`
	record.Meta
}

// GSAPHeader is GSAP's PR number and date, and who filled them (Source "" until GSAP answers).
type GSAPHeader struct {
	PRNumber string     `json:"prNumber"`
	PRDate   *date.Date `json:"prDate"`
	Source   string     `json:"source"`
}

// PurchaseOrderLine is the PO line GSAP made for a PR line; Line is "" until GSAP answers.
type PurchaseOrderLine struct {
	Number string       `json:"number"`
	Line   string       `json:"line"`
	Status string       `json:"status"`
	Date   *date.Date   `json:"date"`
	Value  *money.Money `json:"value"`
}

// Version is the purchase request as last approved: kept while a change is in review, restored when it is not.
type Version struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	OwnerID       string                   `json:"ownerId"`
	MoAApproverID string                   `json:"moaApproverId"`
	Attributes    settings.AttributeValues `json:"attributes"`
	Vendor        string                   `json:"vendor"`
	Lines         []VersionLine            `json:"lines"`
	ApprovedAt    time.Time                `json:"approvedAt"`
}

type VersionLine struct {
	ID           string                   `json:"id"`
	LineItemID   string                   `json:"lineItemId"`
	Currency     string                   `json:"currency"`
	Amount       money.Amount             `json:"amount"`
	DeliveryDate date.Date                `json:"deliveryDate"`
	OverallLimit *money.Amount            `json:"overallLimit"`
	Description  string                   `json:"description"`
	Attributes   settings.AttributeValues `json:"attributes"`
}

// HXKey is the tracking key GSAP finds a line item by: "HX" followed by its number, L-1006 is HX1006.
func HXKey(lineItemID string) string {
	return "HX" + strings.TrimPrefix(lineItemID, "L-")
}

func (pr PurchaseRequest) hasPO() bool {
	return pr.GSAP.PRNumber != ""
}

// gsapPending reports whether the next GSAP run sends the line: no PO line yet, or a PO value that no longer
// matches its amount and currency.
func (l Line) gsapPending() bool {
	if l.PO.Line == "" || l.PO.Value == nil {
		return true
	}
	return l.PO.Value.Currency != l.Currency || l.PO.Value.Amount.Cmp(l.Amount) != 0
}

func (l Line) money() money.Money {
	return money.Money{Amount: l.Amount, Currency: l.Currency}
}

func (pr PurchaseRequest) snapshot(at time.Time) *Version {
	v := &Version{
		Name: pr.Name, Description: pr.Description, OwnerID: pr.OwnerID, MoAApproverID: pr.MoAApproverID,
		Attributes: pr.Attributes, Vendor: pr.Vendor, Lines: make([]VersionLine, 0, len(pr.Lines)), ApprovedAt: at,
	}
	for _, l := range pr.Lines {
		v.Lines = append(v.Lines, VersionLine{
			ID: l.ID, LineItemID: l.LineItemID, Currency: l.Currency, Amount: l.Amount, DeliveryDate: l.DeliveryDate,
			OverallLimit: l.OverallLimit, Description: l.Description, Attributes: l.Attributes,
		})
	}
	return v
}

// approvedLine returns the line's values in the approved version, and false for a line added since.
func (v *Version) approvedLine(id string) (VersionLine, bool) {
	if v == nil {
		return VersionLine{}, false
	}
	for _, l := range v.Lines {
		if l.ID == id {
			return l, true
		}
	}
	return VersionLine{}, false
}

// changedSince reports whether the line differs from its approved values, or was added after the approval.
func (l Line) changedSince(v *Version) bool {
	before, found := v.approvedLine(l.ID)
	return !found || !sameValues(before, l.values())
}

func (l Line) sameAs(o Line) bool {
	return sameValues(l.values(), o.values())
}

func (l Line) values() VersionLine {
	return VersionLine{ID: l.ID, LineItemID: l.LineItemID, Currency: l.Currency, Amount: l.Amount,
		DeliveryDate: l.DeliveryDate, OverallLimit: l.OverallLimit, Description: l.Description, Attributes: l.Attributes}
}

func sameValues(a, b VersionLine) bool {
	sameMoney := a.Currency == b.Currency && a.Amount.Cmp(b.Amount) == 0
	sameText := a.DeliveryDate == b.DeliveryDate && a.Description == b.Description
	return sameMoney && sameText && sameAmount(a.OverallLimit, b.OverallLimit) && sameAttributes(a.Attributes, b.Attributes)
}

func sameAttributes(a, b settings.AttributeValues) bool {
	if len(a) != len(b) {
		return false
	}
	for id, value := range a {
		if !bytes.Equal(value, b[id]) {
			return false
		}
	}
	return true
}

func sameAmount(a, b *money.Amount) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Cmp(*b) == 0
}
