package spa

import (
	"fmt"
	"strings"

	"mapp/internal/platform/authz"
)

const msgAddLineItemFirst = "Add a line item to this activity first."

// rowFacts are what every row of the activity table shares.
type rowFacts struct {
	sees    bool
	raise   authz.Availability
	channel string // the key of the Channel / audience dimension; "" when Settings have none
	show    shower
}

// activityRows is FR-019: one row per activity of the scope, in plan-tree order, with its line items.
func activityRows(acts []*activity, f rowFacts) []ActivityRow {
	rows := make([]ActivityRow, 0, len(acts))
	for _, a := range acts {
		fig := figures{planned: a.planned, actual: a.spend.Actual}
		row := ActivityRow{
			ID: a.rec.ID, Name: a.rec.Name, Campaign: refOf(a.campaign), Initiative: refOf(a.initiative),
			Owner: optionalText(a.rec.Owner), ActivityType: optionalText(a.rec.ActivityType),
			Channel: optionalText(labelsOf(a.rec.Values[f.channel])), StartQuarter: startQuarter(a.rec),
			Status: a.rec.Status, PlannedCost: f.show.money(a.planned), Committed: f.show.financial(f.sees, a.spend.Committed),
			ActualSpend: f.show.financial(f.sees, a.spend.Actual), LineItems: lineItemRows(a, f),
			RaisePurchaseRequest: raiseFor(a, f.raise),
		}
		if f.sees {
			row.VariancePercent, row.Overspend = fig.variance(), fig.overspend()
		}
		rows = append(rows, row)
	}
	return rows
}

func lineItemRows(a *activity, f rowFacts) []LineItemRow {
	rows := make([]LineItemRow, 0, len(a.lineItems))
	for _, li := range a.lineItems {
		rows = append(rows, LineItemRow{ID: li.rec.ID, Name: li.rec.Name, PlannedCost: f.show.money(li.planned),
			Committed: f.show.financial(f.sees, li.spend.Committed), ActualSpend: f.show.financial(f.sees, li.spend.Actual)})
	}
	return rows
}

// raiseFor is FR-022: a purchase request draws on line items, so an activity without one cannot have one.
func raiseFor(a *activity, raise authz.Availability) authz.Availability {
	if len(a.lineItems) == 0 && raise.Available {
		return authz.Unavailable(msgAddLineItemFirst)
	}
	return raise
}

func refOf(r *Record) RecordRef {
	return RecordRef{Type: r.Level, ID: r.ID, Name: r.Name}
}

func startQuarter(r *Record) *string {
	if r.StartDate == nil {
		return nil
	}
	return text(fmt.Sprintf("Q%d %d", (int(r.StartDate.Month())+2)/3, r.StartDate.Year()))
}

func labelsOf(values []Value) string {
	labels := make([]string, 0, len(values))
	for _, v := range values {
		labels = append(labels, v.Label)
	}
	return strings.Join(labels, ", ")
}

func optionalText(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
