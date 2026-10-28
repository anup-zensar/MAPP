package history

import (
	"context"
	"errors"
	"testing"
)

func TestMoveToBin_GuardRefuses(t *testing.T) {
	t.Parallel()
	f := newBareFixture(t)
	f.register(t, f.standardKinds()...)
	f.addTree(t)
	refused := errors.New("a purchase request draws on it")
	var seen []string
	f.service.UseDeleteGuard(func(_ context.Context, snapshots []Snapshot) error {
		for _, s := range snapshots {
			seen = append(seen, s.ID)
		}
		return refused
	})

	_, err := f.service.MoveToBin(t.Context(), priya, "parent", "P-1")

	if !errors.Is(err, refused) {
		t.Fatalf("032 FR-019 MoveToBin = %v, want the guard's refusal", err)
	}
	if _, stillThere := f.stored(t, parentsTable, "P-1"); !stillThere || len(f.entries(t)) != 0 || len(seen) < 2 {
		t.Errorf("032 FR-019 nothing removed or logged: stored %v, entries %d, guard saw %v", stillThere, len(f.entries(t)), seen)
	}
}
