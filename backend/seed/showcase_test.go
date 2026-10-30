package seed_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"mapp/internal/platform/apitest"
	"mapp/internal/platform/fake"
	"mapp/internal/stores/storestest"
	"mapp/seed"
)

func TestLoad_ShowcaseActivitiesAreCompleteAndDemoIDsStay(t *testing.T) {
	t.Parallel()
	db := fake.NewDB()
	if err := seed.Load(t.Context(), storestest.New(db), func() time.Time { return loadedAt }, seed.WithShowcase()); err != nil {
		t.Fatalf("Load(WithShowcase) error: %v", err)
	}
	alex := apitest.New(t, apitest.WithDB(db)).SignIn(t, "alex.morgan@shell.com")

	for _, id := range []string{"P-1001", "P-1005", "P-1006"} {
		var plan struct{ Name string }
		alex.Do(t, http.MethodGet, "/api/v1/plans/"+id, nil).Decode(t, &plan)
		if want := map[string]string{"P-1001": "FY2026 Lubricants Europe", "P-1005": "FY2026 Mobility China – V-Power & Recharge",
			"P-1006": "FY2026 Lubricants B2B Europe – Rimula & Tellus"}[id]; plan.Name != want {
			t.Errorf("plan %s = %q, want %q", id, plan.Name, want)
		}
	}
	for n := 1014; n <= 1033; n++ {
		id := fmt.Sprintf("A-%d", n)
		var view struct {
			Name         string
			Completeness struct{ MissingFields, LineItemsNeedingAttention int }
		}
		got := alex.Do(t, http.MethodGet, "/api/v1/activities/"+id, nil)
		if got.Status != http.StatusOK {
			t.Fatalf("GET activity %s status = %d, want 200", id, got.Status)
		}
		got.Decode(t, &view)
		if view.Completeness.MissingFields != 0 || view.Completeness.LineItemsNeedingAttention != 0 {
			t.Errorf("%s %q completeness = %+v, want nothing missing", id, view.Name, view.Completeness)
		}
	}
}
