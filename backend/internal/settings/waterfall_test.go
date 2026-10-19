package settings

import (
	"errors"
	"testing"

	"northstar/internal/platform/money"
)

// model returns a model with the rates and deal size given as text.
func model(t *testing.T, inqToMql, mqlToDeal, dealSize string) Model {
	t.Helper()
	inq, err1 := money.ParsePercent(inqToMql)
	mql, err2 := money.ParsePercent(mqlToDeal)
	deal, err3 := money.ParseAmount(dealSize)
	if err := errors.Join(err1, err2, err3); err != nil {
		t.Fatalf("model: %v", err)
	}
	return Model{InqToMqlRate: inq, MqlToDealRate: mql, AverageDealSize: money.USD(deal)}
}

// The waterfall of every seeded model matches domain model §4.5 (data model §2.1 table; the web app's
// waterfall.test.ts uses the same table).
func TestWaterfall_SeededModels_FR010_SC006(t *testing.T) {
	lubricants := model(t, "2.00", "5.00", "10000.00")
	mobility := model(t, "3.00", "4.00", "5000.00")
	tests := []struct {
		name            string
		model           Model
		inq, mql, deals int64
		revenue         string
	}{
		{"Lubricants 1.0", lubricants, 250_000, 5_000, 250, "2500000.00"},
		{"Mobility 1.0", mobility, 250_000, 7_500, 300, "1500000.00"},
		{"Lubricants 1.0, 25 inquiries", lubricants, 25, 1, 0, "0.00"},
		{"Lubricants 1.0, none", lubricants, 0, 0, 0, "0.00"},
	}
	for _, tt := range tests {
		goals, err := Waterfall(tt.inq, tt.model)
		if err != nil || goals.INQ != tt.inq || goals.MQL != tt.mql || goals.Deals != tt.deals || goals.Revenue.String() != tt.revenue {
			t.Errorf("%s: Waterfall(%d) = %+v, %v; want %d MQL, %d deals, %s USD", tt.name, tt.inq, goals, err, tt.mql, tt.deals, tt.revenue)
		}
	}
}

// MQL and deals are rounded half-up to whole numbers, as goals are (FR-010): 2.00 % of 25 is 0.5 → 1,
// 50.00 % of 3 is 1.5 → 2.
func TestWaterfall_RoundsHalfUp_FR010(t *testing.T) {
	goals, err := Waterfall(25, model(t, "2.00", "50.00", "100.00"))
	if err != nil || goals.MQL != 1 || goals.Deals != 1 || goals.Revenue.String() != "100.00" {
		t.Errorf("Waterfall(25) = %+v, %v; want 1 MQL, 1 deal", goals, err)
	}
	goals, err = Waterfall(3, model(t, "100.00", "50.00", "100.00"))
	if err != nil || goals.MQL != 3 || goals.Deals != 2 {
		t.Errorf("Waterfall(3) = %+v, %v; want 3 MQL, 2 deals", goals, err)
	}
}

// A revenue beyond what money keeps exactly is refused, never rounded or wrapped (FR-010).
func TestWaterfall_TooLarge_FR010(t *testing.T) {
	_, err := Waterfall(1_000_000_000_000, model(t, "100.00", "100.00", "100000000.00"))
	if !errors.Is(err, money.ErrOutOfRange) {
		t.Errorf("error = %v, want money.ErrOutOfRange", err)
	}
}
