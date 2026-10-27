package plan

import (
	"context"
	"testing"
)

// Levels below the plan register once each, with a name and the functions AddFacts and Copy; anything
// else stops the API from starting (D-072).
func TestRegisterLevel_RefusesIncompleteOrTwice(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	addFacts := func(context.Context, []string, map[string]*Facts) error { return nil }
	copyLevel := func(context.Context, *CopyJob) error { return nil }
	f.addLevel(t, "campaign", nil)

	tests := map[string]Level{
		"no name":     {AddFacts: addFacts, Copy: copyLevel},
		"twice":       {Name: "campaign", AddFacts: addFacts, Copy: copyLevel},
		"no AddFacts": {Name: "initiative", Copy: copyLevel},
		"no Copy":     {Name: "initiative", AddFacts: addFacts},
	}
	for name, level := range tests {
		if err := f.service.RegisterLevel(level); err == nil {
			t.Errorf("%s: RegisterLevel accepted it", name)
		}
	}
	if len(f.service.levels) != 1 {
		t.Errorf("%d levels registered, want only the campaign", len(f.service.levels))
	}
}
