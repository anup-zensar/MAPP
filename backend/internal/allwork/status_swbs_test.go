package allwork

import "testing"

func TestSwbsChip_030_FR017(t *testing.T) {
	t.Parallel()
	received := swbsOf(&Node{Swbs: "received"})
	if received == nil || received.Label != "SWBS code received" || received.Tone != ToneDone {
		t.Fatalf("received chip = %+v", received)
	}
	order := []string{"notRequested", "waitingForApproval", "readyToSend", "received"}
	for i := 1; i < len(order); i++ {
		before, after := lifecycle(swbsStatuses, order[i-1]), lifecycle(swbsStatuses, order[i])
		if before.number >= after.number {
			t.Errorf("%s sorts at %d, not before %s at %d", order[i-1], before.number, order[i], after.number)
		}
	}
}
