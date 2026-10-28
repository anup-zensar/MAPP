package app

import (
	"context"
	"encoding/json"

	"mapp/internal/history"
)

const (
	kindPurchaseRequest     = "purchaseRequest"
	kindPurchaseRequestLine = "purchaseRequestLine"
)

type entryRecord struct {
	kind, id  string
	path      []history.RecordRef
	snapshots []history.Snapshot
}

// planIDOf is the plan a record's log and bin entries follow (D-238): a plan itself, the top of a planning
// record's path, and for purchase requests and their lines the plan the purchase request belongs to.
func (h historyAccess) planIDOf(ctx context.Context, r entryRecord) (string, bool) {
	switch r.kind {
	case planKind:
		return r.id, true
	case kindPurchaseRequest, kindPurchaseRequestLine:
		return h.purchaseRequestPlan(ctx, r)
	}
	if len(r.path) == 0 || r.path[0].Type != planKind {
		return "", false
	}
	return r.path[0].ID, true
}

func (h historyAccess) purchaseRequestPlan(ctx context.Context, r entryRecord) (string, bool) {
	prID, known := purchaseRequestOf(r)
	if !known || h.purchaseRequests == nil {
		return "", false
	}
	if pr, found, err := h.purchaseRequests.Get(ctx, prID); err == nil && found {
		return pr.PlanID, true
	}
	for _, s := range r.snapshots {
		var stored struct {
			PlanID string `json:"planId"`
		}
		if s.Kind == kindPurchaseRequest && s.ID == prID && json.Unmarshal(s.Data, &stored) == nil {
			return stored.PlanID, true
		}
	}
	return "", false
}

func purchaseRequestOf(r entryRecord) (string, bool) {
	if r.kind == kindPurchaseRequest {
		return r.id, true
	}
	if len(r.path) == 0 {
		return "", false
	}
	return r.path[len(r.path)-1].ID, true
}
