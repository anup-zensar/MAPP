package purchaserequest

import (
	"testing"
	"time"

	"mapp/internal/platform/money"
)

func TestReservation_038_FR001_C1(t *testing.T) {
	t.Parallel()
	eur, err := money.ParseRate("0.92")
	if err != nil {
		t.Fatal(err)
	}
	rates := money.Rates{"USD": money.One, "EUR": eur}
	eurItem := LineItemFacts{ID: "L-1", Currency: "EUR", PlannedCost: amount(t, "15800.00")}
	usdItem := LineItemFacts{ID: "L-2", Currency: "USD", PlannedCost: amount(t, "20000.00")}
	line := func(id, li, value string) Line {
		return Line{ID: id, LineItemID: li, Currency: "EUR", Amount: amount(t, value)}
	}
	approvedAs := func(lines ...Line) *Version {
		return PurchaseRequest{Lines: lines}.snapshot(time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC))
	}
	tests := []struct {
		name string
		li   LineItemFacts
		pr   PurchaseRequest
		want string
	}{
		{name: "lowered 15,800 to 12,000 keeps 15,800 reserved", li: eurItem,
			pr: changing([]Line{line("PRL-1", "L-1", "12000.00")}, approvedAs(line("PRL-1", "L-1", "15800.00"))), want: "0.00"},
		{name: "raised 10,000 to 12,000 reserves 12,000", li: eurItem,
			pr: changing([]Line{line("PRL-1", "L-1", "12000.00")}, approvedAs(line("PRL-1", "L-1", "10000.00"))), want: "3,800.00"},
		{name: "a line the change adds reserves its changed amount", li: eurItem,
			pr: changing([]Line{line("PRL-1", "L-1", "10000.00"), line("PRL-2", "L-1", "2000.00")},
				approvedAs(line("PRL-1", "L-1", "10000.00"))), want: "3,800.00"},
		{name: "a line the change moves off the line item keeps its approved amount", li: eurItem,
			pr: changing([]Line{line("PRL-1", "L-2", "15800.00")}, approvedAs(line("PRL-1", "L-1", "15800.00"))), want: "0.00"},
		{name: "another currency at today's rate", li: usdItem,
			pr: changing([]Line{line("PRL-1", "L-2", "4600.00")}, approvedAs(line("PRL-1", "L-2", "9200.00"))), want: "10,000.00"},
		{name: "no change in review counts the lines as they are", li: eurItem,
			pr: PurchaseRequest{ID: "PR-1", Status: StatusApproved, Lines: []Line{line("PRL-1", "L-1", "12000.00")},
				Approved: approvedAs(line("PRL-1", "L-1", "12000.00"))}, want: "3,800.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := ClaimsOf([]PurchaseRequest{tt.pr}, []string{tt.li.ID})
			got, err := balance{rates: rates, claims: claims}.available(tt.li, "")
			if err != nil || got.Format() != tt.want {
				t.Errorf("available = %s, %v; want %s", got.Format(), err, tt.want)
			}
		})
	}
}

func changing(lines []Line, approved *Version) PurchaseRequest {
	for i := range lines {
		lines[i].PurchaseRequestID = "PR-1"
	}
	return PurchaseRequest{ID: "PR-1", Name: "Stand", Status: StatusInReview, ChangePending: true, Lines: lines,
		Approved: approved}
}
