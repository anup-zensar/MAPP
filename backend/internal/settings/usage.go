package settings

import (
	"context"

	"northstar/internal/history"
	"northstar/internal/platform/money"
)

// Reference is one record outside Settings and the Settings records it uses (data model §10, D-042):
// one per plan, campaign, initiative, activity, line item and person, live or in the recycle bin. Features
// 003–007 report them through a ReferenceSource, so Settings counts "Used by", refuses in-use deletes and
// states impacts without importing those features.
type Reference struct {
	Record              history.RecordRef // {type: "activity", id: "A-1003", name: "Launch film"}; plans carry their name
	InBin               bool              // true while the record is in the recycle bin
	Level               Level             // its Shell-attribute level; "" for people
	PlanID              string            // its plan (a plan: its own id); "" for people
	ActivityID          string            // an activity: its own id; a line item: its activity; "" otherwise
	Attributes          AttributeValues   // its Shell attribute values; nil when none
	ModelID             string            // plans
	ActivityTypeID      string            // activities
	Currency            string            // line items: their currency; people: their display currency
	PlannedCost         *money.Amount     // line items: planned cost in Currency
	FinanceBudgetID     string            // plans: default finance budget
	FinanceBudgetItemID string            // line items: linked finance budget item
}

// ReferenceSource lists every record of one kind, live and in the recycle bin (read with
// history.SnapshotsInBin), inside the caller's transaction, which ctx carries. Features 003–007 each provide
// one; internal/app registers them with (*Service).AddReferenceSource.
type ReferenceSource func(ctx context.Context) ([]Reference, error)
