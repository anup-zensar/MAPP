package lineitem

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

type SwbsStatus string

const (
	SwbsNotRequested       SwbsStatus = "notRequested"
	SwbsWaitingForApproval SwbsStatus = "waitingForApproval"
	SwbsReadyToSend        SwbsStatus = "readyToSend"
	SwbsReceived           SwbsStatus = "received"
)

const activityApproved = "approved"

const swbsKeptOnMessage = "This line item already has its SWBS code from the simulated GSAP, so the request stays on."

// Swbs is what GSAP answered to the line item's SWBS request: empty until a code is received, then kept.
type Swbs struct {
	SwbsCode        string     `json:"swbsCode"`
	SwbsRequestedAt *time.Time `json:"swbsRequestedAt"`
	SwbsReceivedAt  *time.Time `json:"swbsReceivedAt"`
}

type SwbsCodeReceived struct {
	LineItemID string
	Code       string
}

// SwbsReceipt is what GSAP answered, when, and who asked: received and requested at At.
type SwbsReceipt struct {
	By    actor.Actor
	At    time.Time
	Codes []SwbsCodeReceived
}

// SwbsStatusOf counts an activity shown In progress or Complete as approved: its stored status is.
func SwbsStatusOf(li LineItem, f ActivityFacts) SwbsStatus {
	switch {
	case li.SwbsCode != "":
		return SwbsReceived
	case !li.RequestSwbs:
		return SwbsNotRequested
	case f.Status == activityApproved:
		return SwbsReadyToSend
	}
	return SwbsWaitingForApproval
}

// HXKey is the tracking number GSAP data finds the line item by: L-1003 is HX1003.
func (li LineItem) HXKey() string {
	return "HX" + strings.TrimPrefix(li.ID, "L-")
}

// ReceiveSwbsCodes fails for a line item that is gone or already has a code: the caller's Write saves nothing.
func ReceiveSwbsCodes(ctx context.Context, repo Store, receipt SwbsReceipt) error {
	ids := make([]string, 0, len(receipt.Codes))
	for _, r := range receipt.Codes {
		ids = append(ids, r.LineItemID)
	}
	items, err := repo.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	for _, r := range receipt.Codes {
		i := slices.IndexFunc(items, func(li LineItem) bool { return li.ID == r.LineItemID })
		if i < 0 || items[i].SwbsCode != "" {
			return fmt.Errorf("lineitem: line item %s is gone or already has an SWBS code", r.LineItemID)
		}
		li := items[i]
		requested, received := receipt.At, receipt.At
		li.Swbs = Swbs{SwbsCode: r.Code, SwbsRequestedAt: &requested, SwbsReceivedAt: &received}
		li.Touch(receipt.By, receipt.At)
		if err := repo.Put(ctx, li); err != nil {
			return err
		}
	}
	return nil
}

func checkSwbsKept(problems *errs.Problems, before LineItem, requestSwbs bool) {
	if before.SwbsCode != "" && !requestSwbs {
		problems.Add("requestSwbs", swbsKeptOnMessage)
	}
}
