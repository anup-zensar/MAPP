package plans

import (
	"reflect"
	"testing"

	"northstar/internal/platform/money"
	"northstar/internal/settings"
)

// model returns a model with the rates r1 and r2 (percent) and the average deal size (USD).
func model(t *testing.T, r1, r2, dealSize string) settings.Model {
	t.Helper()
	p1, err1 := money.ParsePercent(r1)
	p2, err2 := money.ParsePercent(r2)
	if err1 != nil || err2 != nil {
		t.Fatalf("rates %q, %q: %v, %v", r1, r2, err1, err2)
	}
	return settings.Model{InqToMqlRate: p1, MqlToDealRate: p2, AverageDealSize: money.USD(usd(t, dealSize))}
}

// wantWaterfall is a waterfall as the tests write it: nil for a stage that can't be worked out.
func wantWaterfall(t *testing.T, inq, mql, deals *int64, revenue string, reasons GoalReasons) waterfall {
	t.Helper()
	w := waterfall{INQ: inq, MQL: mql, Deals: deals, Reasons: reasons}
	if revenue != "" {
		amount := usd(t, revenue)
		w.Revenue = &amount
	}
	return w
}

// assertWaterfall compares two waterfalls by value.
func assertWaterfall(t *testing.T, got, want waterfall) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("waterfall = %s, want %s", describeWaterfall(got), describeWaterfall(want))
	}
}

func describeWaterfall(w waterfall) string {
	text := func(n *int64) string {
		if n == nil {
			return "nil"
		}
		return money.FromCents(*n * 100).String()
	}
	revenue := "nil"
	if w.Revenue != nil {
		revenue = w.Revenue.String()
	}
	return "INQ " + text(w.INQ) + ", MQL " + text(w.MQL) + ", deals " + text(w.Deals) + ", revenue " + revenue +
		", reasons " + w.Reasons.INQ + "|" + w.Reasons.MQL + "|" + w.Reasons.Deals + "|" + w.Reasons.Revenue
}

// INQ 250,000 with Lubricants 1.0 (2 %, 5 %, 10,000.00) gives MQL 5,000, 250 deals and revenue
// 2,500,000.00 (FR-009, SC-006, US1 scenario 5).
func TestWaterfall_FromINQ_FR009(t *testing.T) {
	got := planWaterfall(StageINQ, "250000", model(t, "2.00", "5.00", "10000.00"))

	assertWaterfall(t, got, wantWaterfall(t, ptr[int64](250_000), ptr[int64](5_000), ptr[int64](250), "2500000.00", GoalReasons{}))
}

// A goal typed as MQL works INQ out backwards and deals and revenue forwards (FR-009).
func TestWaterfall_FromMQL_FR009(t *testing.T) {
	got := planWaterfall(StageMQL, "5000", model(t, "2.00", "5.00", "10000.00"))

	assertWaterfall(t, got, wantWaterfall(t, ptr[int64](250_000), ptr[int64](5_000), ptr[int64](250), "2500000.00", GoalReasons{}))
}

// A goal typed as revenue works every other stage out backwards: 2,500,000.00 → 250 deals → MQL 5,000 →
// INQ 250,000 (FR-009, SC-006).
func TestWaterfall_FromRevenue_FR009(t *testing.T) {
	got := planWaterfall(StageRevenue, "2500000.00", model(t, "2.00", "5.00", "10000.00"))

	assertWaterfall(t, got, wantWaterfall(t, ptr[int64](250_000), ptr[int64](5_000), ptr[int64](250), "2500000.00", GoalReasons{}))
}

// Every stage is rounded half up to a whole number: 2.5 → 3, 2.4999 → 2 (FR-009).
func TestWaterfall_RoundsHalfUp_FR009(t *testing.T) {
	tests := []struct {
		name   string
		stage  GoalStage
		amount string
		model  settings.Model
		want   waterfall
	}{
		{"MQL 2.5 → 3", StageINQ, "125", model(t, "2.00", "50.00", "100.00"),
			wantWaterfall(t, ptr[int64](125), ptr[int64](3), ptr[int64](2), "200.00", GoalReasons{})}, // deals 1.5 → 2
		{"MQL 2.4999 → 2", StageINQ, "24999", model(t, "0.01", "100.00", "1.00"), // 24,999 × 0.01 % = 2.4999
			wantWaterfall(t, ptr[int64](24_999), ptr[int64](2), ptr[int64](2), "2.00", GoalReasons{})},
		{"INQ 2.5 → 3", StageMQL, "1", model(t, "40.00", "100.00", "1.00"),
			wantWaterfall(t, ptr[int64](3), ptr[int64](1), ptr[int64](1), "1.00", GoalReasons{})}, // 1 ÷ 40 % = 2.5
		{"deals 2.5 → 3", StageRevenue, "25.00", model(t, "100.00", "100.00", "10.00"),
			wantWaterfall(t, ptr[int64](3), ptr[int64](3), ptr[int64](3), "25.00", GoalReasons{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWaterfall(t, planWaterfall(tt.stage, tt.amount, tt.model), tt.want)
		})
	}
}

// Working backwards through a rate or a deal size of 0 can't be done: that stage, and every stage worked
// out from it, is empty with the reason (spec edge case). Without a goal amount every stage is empty with
// no reason.
func TestWaterfall_ZeroRateGivesReason_EdgeCase(t *testing.T) {
	r1 := "The model's INQ→MQL rate is 0, so this can't be worked out."
	r2 := "The model's MQL→deal rate is 0, so this can't be worked out."
	deal := "The model's average deal size is 0, so this can't be worked out."
	tests := []struct {
		name   string
		stage  GoalStage
		amount string
		model  settings.Model
		want   waterfall
	}{
		{"INQ→MQL rate 0, from MQL", StageMQL, "5000", model(t, "0.00", "5.00", "10000.00"),
			wantWaterfall(t, nil, ptr[int64](5_000), ptr[int64](250), "2500000.00", GoalReasons{INQ: r1})},
		{"MQL→deal rate 0, from revenue", StageRevenue, "2500000.00", model(t, "2.00", "0.00", "10000.00"),
			wantWaterfall(t, nil, nil, ptr[int64](250), "2500000.00", GoalReasons{INQ: r2, MQL: r2})},
		{"deal size 0, from revenue", StageRevenue, "2500000.00", model(t, "2.00", "5.00", "0.00"),
			wantWaterfall(t, nil, nil, nil, "2500000.00", GoalReasons{INQ: deal, MQL: deal, Deals: deal})},
		{"rates 0 forwards multiply to 0", StageINQ, "250000", model(t, "0.00", "0.00", "0.00"),
			wantWaterfall(t, ptr[int64](250_000), ptr[int64](0), ptr[int64](0), "0.00", GoalReasons{})},
		{"no goal amount", StageINQ, "", model(t, "2.00", "5.00", "10000.00"), waterfall{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWaterfall(t, planWaterfall(tt.stage, tt.amount, tt.model), tt.want)
		})
	}
}

// Goals show how much of the goal the activities already plan, as a whole percent: INQ 150,000 of 250,000
// → 60 %; without a goal, or with a goal of 0, there is no share (FR-016, US3 scenario 5).
func TestPlannedShare_FR016(t *testing.T) {
	m := model(t, "2.00", "5.00", "10000.00")
	activities := GoalTotals{INQ: 150_000, MQL: 2_000, Revenue: money.USD(usd(t, "1000000.00"))}
	tests := []struct {
		name   string
		stage  GoalStage
		amount string
		want   *int
	}{
		{"INQ 150,000 of 250,000", StageINQ, "250000", ptr(60)},
		{"MQL 2,000 of 5,000", StageMQL, "5000", ptr(40)},
		{"revenue 1,000,000.00 of 2,500,000.00", StageRevenue, "2500000.00", ptr(40)},
		{"no goal", StageINQ, "", nil},
		{"goal 0", StageINQ, "0", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plannedShare(tt.stage, planWaterfall(tt.stage, tt.amount, m), activities)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("share = %v, want %v", deref(got), deref(tt.want))
			}
		})
	}
}

// deref shows an optional number in a failure message.
func deref(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
