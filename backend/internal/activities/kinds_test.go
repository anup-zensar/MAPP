package activities_test

import (
	"testing"

	"mapp/internal/activities"
	"mapp/internal/campaigns"
	"mapp/internal/history"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

// Activities have one history kind, "activity", under feature 005's initiative kind: history refuses it
// until the initiative kind is registered, and takes it afterwards (FR-031, data model §5).
func TestKinds_RegisterAfterInitiative(t *testing.T) {
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	hist := history.NewService(db, history.NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), clock.Now)
	service := activities.NewService(db, activities.NewJSONRepository(), ids.NewGenerator(ids.NewJSONCounters()), hist, clock.Now, activities.Deps{})

	kinds := service.Kinds()

	if len(kinds) != 1 || kinds[0].Name != "activity" || kinds[0].Parent != "initiative" ||
		kinds[0].Singular != "activity" || kinds[0].Plural != "activities" {
		t.Fatalf("kinds = %+v, want the one kind activity under initiative", kinds)
	}
	if err := hist.Register(kinds[0]); err == nil {
		t.Error("history took the activity kind before the initiative kind")
	}
	if err := hist.Register(initiativeKind(campaigns.NewJSONRepository())); err != nil {
		t.Fatalf("Register(initiative): %v", err)
	}
	if err := hist.Register(kinds[0]); err != nil {
		t.Errorf("Register(activity) after initiative: %v", err)
	}
}
