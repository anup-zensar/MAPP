package lineitem

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"mapp/internal/platform/actor"
)

var alex = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}

var received = time.Date(2026, 3, 13, 10, 30, 0, 0, time.UTC)

func TestSwbsStatusOf_030_FR001(t *testing.T) {
	t.Parallel()
	withCode := Swbs{SwbsCode: "SIM-GB01-0001", SwbsRequestedAt: &received, SwbsReceivedAt: &received}
	tests := []struct {
		name     string
		li       LineItem
		activity string
		want     SwbsStatus
	}{
		{"a code wins while the activity is approved", LineItem{RequestSwbs: true, Swbs: withCode}, "approved", SwbsReceived},
		{"a code stays when the activity is declined", LineItem{RequestSwbs: true, Swbs: withCode}, "declined", SwbsReceived},
		{"a code stays when the activity is back to created", LineItem{RequestSwbs: true, Swbs: withCode}, "created", SwbsReceived},
		{"request off", LineItem{}, "approved", SwbsNotRequested},
		{"approved activity", LineItem{RequestSwbs: true}, "approved", SwbsReadyToSend},
		{"created activity", LineItem{RequestSwbs: true}, "created", SwbsWaitingForApproval},
		{"submitted activity", LineItem{RequestSwbs: true}, "submitted", SwbsWaitingForApproval},
		{"declined activity", LineItem{RequestSwbs: true}, "declined", SwbsWaitingForApproval},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := SwbsStatusOf(tt.li, ActivityFacts{Status: tt.activity}); got != tt.want {
				t.Errorf("SwbsStatusOf() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestHXKey_030_FR016(t *testing.T) {
	t.Parallel()
	for id, want := range map[string]string{"L-1003": "HX1003", "L-1": "HX1", "L-120034": "HX120034"} {
		if got := (LineItem{ID: id}).HXKey(); got != want {
			t.Errorf("HXKey(%s) = %s, want %s", id, got, want)
		}
	}
}

func (f *fixture) receive(t *testing.T, codes ...SwbsCodeReceived) error {
	t.Helper()
	return f.db.Write(t.Context(), func(ctx context.Context) error {
		return ReceiveSwbsCodes(ctx, newFakeStore(f.db), SwbsReceipt{By: alex, At: received, Codes: codes})
	})
}

type swbsSummary struct {
	Status              SwbsStatus
	Code, HXKey         string
	Requested, Received *time.Time
	Version             int
	By                  actor.Actor
}

func swbsSummaryOf(v View) swbsSummary {
	return swbsSummary{Status: v.SwbsStatus, Code: v.SwbsCode, HXKey: v.HXKey, Requested: v.SwbsRequestedAt,
		Received: v.SwbsReceivedAt, Version: v.Version, By: v.UpdatedBy}
}

func TestReceiveSwbsCodes_StoresCodeAndTimes_030_FR002(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, complete(input("Venue hire", "USD", "1000.00")))

	if err := f.receive(t, SwbsCodeReceived{LineItemID: venue.ID, Code: "SIM-DE03-0001"}); err != nil {
		t.Fatalf("receive: %v", err)
	}
	details, err := f.service.Get(t.Context(), priya, venue.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	want := swbsSummary{Status: SwbsReceived, Code: "SIM-DE03-0001", HXKey: "HX1001", Requested: &received,
		Received: &received, Version: venue.Version + 1, By: alex}
	if diff := cmp.Diff(want, swbsSummaryOf(details.LineItem)); diff != "" {
		t.Errorf("received line item (-want +got):\n%s", diff)
	}
}

func TestReceiveSwbsCodes_RefusesASecondCode_030_FR003(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	venue := f.create(t, priya, socialOnAir, complete(input("Venue hire", "USD", "1000.00")))
	stand := f.create(t, priya, socialOnAir, complete(input("Stand build", "USD", "2000.00")))
	if err := f.receive(t, SwbsCodeReceived{LineItemID: venue.ID, Code: "SIM-DE03-0001"}); err != nil {
		t.Fatalf("receive: %v", err)
	}

	again := f.receive(t, SwbsCodeReceived{LineItemID: stand.ID, Code: "SIM-DE03-0002"},
		SwbsCodeReceived{LineItemID: venue.ID, Code: "SIM-DE03-0003"})
	gone := f.receive(t, SwbsCodeReceived{LineItemID: "L-9999", Code: "SIM-DE03-0004"})

	if again == nil || gone == nil {
		t.Fatalf("a second code and an unknown line item = %v, %v; want errors", again, gone)
	}
	if code := f.storedByID(t, stand.ID).SwbsCode; code != "" {
		t.Errorf("Stand build kept %q from a failed write", code)
	}
	assertCode(t, f.storedByID(t, venue.ID))
}
