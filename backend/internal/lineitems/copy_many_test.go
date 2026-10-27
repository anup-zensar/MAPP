package lineitems

import (
	"context"
	"testing"
)

// The line items of many activities are copied at once, each activity's in the order of the pairs, so the
// copies get the IDs one activity after the other gave them (028 FR-004).
func TestCopyLineItemsOf_IDsActivityByActivity_028_FR004(t *testing.T) {
	t.Parallel()
	f := newTestService(t)
	early := f.create(t, priya, retailEvent, input("Stand build", "USD", "5.00"))
	late := f.create(t, priya, launchFilm, input("Agency fee", "USD", "7.00"))
	pairs := []CopyPair{{FromID: launchFilm, ToID: "A-2001"}, {FromID: retailEvent, ToID: "A-2002"}}

	var copied int
	err := f.db.Write(t.Context(), func(ctx context.Context) error {
		var err error
		copied, err = f.service.CopyLineItemsOf(ctx, daniel, pairs, 0)
		return err
	})

	if err != nil || copied != 2 {
		t.Fatalf("CopyLineItemsOf = %d, %v; want 2 line items", copied, err)
	}
	stored := f.stored(t)
	lateCopy, earlyCopy := lineItemNamed(t, stored, "A-2001", late.Name), lineItemNamed(t, stored, "A-2002", early.Name)
	firstPairFirst := early.ID < late.ID && lateCopy.ID < earlyCopy.ID
	if !firstPairFirst {
		t.Errorf("copies of %s and %s are %s and %s; want the first pair's copy first", late.ID, early.ID, lateCopy.ID, earlyCopy.ID)
	}
}
