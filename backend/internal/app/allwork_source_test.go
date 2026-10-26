package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"mapp/internal/allwork"
	"mapp/internal/platform/date"
	"mapp/internal/platform/store"
	"mapp/internal/stores"
	"mapp/seed"
)

// everyColumn names the columns whose values the source reads only when a view shows them.
var everyColumn = []string{"completeness", "plan.goalMql", "plan.goalRevenue", "inq", "mql", "revenue"}

// newSeededApp builds the API on the demo data, loaded at 12 Mar 2026 09:00 UTC, and returns it with its
// store.
func newSeededApp(t testing.TB) (*App, *store.DB) {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	now := func() time.Time { return time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC) }
	if err = seed.Load(t.Context(), stores.JSON(db), now); err != nil {
		t.Fatalf("seed.Load: %v", err)
	}
	cfg, err := LoadConfig(func(string) string { return "" })
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	built, err := New(cfg, stores.JSON(db), now)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built, db
}

// All work reads the demo year 2026 with the values the owning features calculate (FR-001): both 2026
// plans with their campaigns, initiatives, activities and line items, each with its status, money, owner
// and place in the plan tree.
func TestPlanningSource_ReadsSeedYear_FR001(t *testing.T) {
	built, db := newSeededApp(t)
	var data allwork.YearData
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		data, err = built.allWork.Year(ctx, 2026, date.MustParse("2026-03-12"), everyColumn)
		return err
	})
	if err != nil {
		t.Fatalf("Year: %v", err)
	}

	counts := map[allwork.Level]int{}
	byName := map[string]allwork.Node{}
	for _, n := range data.Nodes {
		counts[n.Level]++
		byName[n.Name] = n
	}
	want := map[allwork.Level]int{allwork.LevelPlan: 2, allwork.LevelCampaign: 3, allwork.LevelInitiative: 6,
		allwork.LevelActivity: 10, allwork.LevelLineItem: 8}
	for level, n := range want {
		if counts[level] != n {
			t.Errorf("%d %s nodes, want %d", counts[level], level, n)
		}
	}

	plan := byName["FY2026 Lubricants Europe"]
	if plan.ID != "P-1001" || plan.Status != "published" || plan.Budget == nil || plan.Budget.String() != "1200000.00" ||
		plan.Owner == nil || plan.Owner.Name != "Priya Raman" || plan.Missing != 0 || len(plan.TeamIDs) != 1 ||
		len(plan.BusinessUnitIDs) != 1 || plan.Strategic {
		t.Errorf("plan = %+v", plan)
	}
	mobility := byName["FY2026 Mobility China"]
	if mobility.Status != "draft" || len(mobility.BusinessUnitIDs) != 0 || mobility.Fields["plan.goalMql"].Number == nil ||
		mobility.Fields["plan.goalAmount"].Number == nil || *mobility.Fields["plan.goalAmount"].Number != 150000 {
		t.Errorf("Mobility plan = %+v", mobility)
	}
	helix := byName["Helix launch"]
	if helix.ParentID != "P-1001" || helix.PlanID != "P-1001" || helix.Status != "inProgress" || helix.Total != 4 ||
		helix.Timing != "running" || helix.Budget == nil || helix.Budget.String() != "700000.00" {
		t.Errorf("campaign = %+v", helix)
	}
	if cards := byName["Fleet cards"]; cards.Timing != "" || cards.StartDate != nil || cards.Missing != 1 {
		t.Errorf("initiative without dates = %+v", cards)
	}
	film := byName["Launch film"]
	if film.Status != "created" || film.PlannedCost == nil || film.PlannedCost.String() != "120000.00" ||
		film.INQ == nil || *film.INQ != 250000 || film.MQL == nil || *film.MQL != 5000 || film.Revenue == nil ||
		film.Revenue.String() != "2500000.00" || film.Missing != 3 || film.PlanID != "P-1001" ||
		strings.Join(film.Months, ",") != "2026-03,2026-04,2026-05" || film.ActivityTypeID == "" || film.CreatedAt.IsZero() {
		t.Errorf("activity = %+v", film)
	}
	if social := byName["Social always-on"]; social.Status != "inProgress" { // approved and started by 12 Mar
		t.Errorf("Social always-on shows %q, want inProgress", social.Status)
	}
	if fund := byName["Partner co-marketing fund"]; !fund.Ongoing || strings.Join(fund.Months, ",") != "2026-03" {
		t.Errorf("ongoing activity = %+v", fund)
	}
	production := byName["Film production"]
	if production.ParentID != film.ID || production.PlannedCost == nil || production.PlannedCost.String() != "50000.00" ||
		production.LocalPlannedCost == nil || production.LocalPlannedCost.Amount.String() != "46000.00" ||
		production.LocalPlannedCost.Currency != "EUR" || len(production.MonthsUSD) != 3 || production.Missing != 3 ||
		production.Swbs != "waitingForApproval" || production.PlanID != "P-1001" {
		t.Errorf("line item = %+v", production)
	}
	var spread string
	for _, m := range production.MonthsUSD {
		spread += m.Month + "=" + m.USD.String() + " "
	}
	if spread != "2026-03=16666.66 2026-04=16666.66 2026-05=16666.68 " {
		t.Errorf("spread = %s", spread)
	}
}

// The level-specific columns come with the demo year's values (FR-018, FR-020): the plan's model and teams,
// a campaign's Cluster, an activity's type and a line item's local amount and spread; every Shell attribute
// of Settings is a column of its level.
func TestPlanningSource_LevelFieldsAndAttributes_FR018(t *testing.T) {
	built, db := newSeededApp(t)
	var data allwork.YearData
	var attributes []allwork.Attribute
	err := db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		if attributes, err = built.allWork.Attributes(ctx); err != nil {
			return err
		}
		data, err = built.allWork.Year(ctx, 2026, date.MustParse("2026-03-12"), everyColumn)
		return err
	})
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(attributes) != 35 {
		t.Errorf("%d attributes, want the 35 of the seed", len(attributes))
	}
	cluster := ""
	for _, a := range attributes {
		if a.Label == "Cluster" && a.Level == allwork.LevelCampaign {
			cluster = "attr." + a.ID
		}
	}
	byName := map[string]allwork.Node{}
	for _, n := range data.Nodes {
		byName[n.Name] = n
	}
	plan := byName["FY2026 Lubricants Europe"].Fields
	if plan["plan.model"].Text != "Lubricants 1.0" || len(plan["plan.teams"].Choices) != 1 ||
		plan["plan.teams"].Choices[0].Label != "Lubricants Europe" || plan["plan.year"].Number == nil ||
		*plan["plan.year"].Number != 2026 || plan["plan.defaultSpread"].Text != "Evenly" {
		t.Errorf("plan fields = %+v", plan)
	}
	if helix := byName["Helix launch"].Fields[cluster]; len(helix.Choices) != 1 || helix.Choices[0].Label != "Lubricants – Europe" {
		t.Errorf("Helix launch %s = %+v", cluster, helix)
	}
	if film := byName["Launch film"].Fields; film["activity.type"].Text != "Content production" ||
		film["activity.approvalWorkflow"].Text == "" || film["activity.ongoing"].Yes == nil || *film["activity.ongoing"].Yes {
		t.Errorf("Launch film fields = %+v", film)
	}
	production := byName["Film production"].Fields
	if local := production["lineItem.localPlannedCost"].Money; local == nil || local.Amount.String() != "46000.00" ||
		local.Currency != "EUR" || production["lineItem.spread"].Text != "Evenly" || production["lineItem.currency"].Text != "EUR" {
		t.Errorf("Film production fields = %+v", production)
	}
}

// The source finds a stored record with its plan, says when a deleted one is in the recycle bin, and reads
// the person's display currency (FR-016, FR-017).
func TestPlanningSource_LocateBinAndCurrency_FR016(t *testing.T) {
	built, db := newSeededApp(t)
	err := db.Read(t.Context(), func(ctx context.Context) error {
		location, found, err := built.allWork.Locate(ctx, "L-1002")
		if err != nil || !found || location != (allwork.Location{Level: allwork.LevelLineItem, Name: "Film production", PlanID: "P-1001", ActivityID: "A-1001"}) {
			t.Errorf("Locate L-1002 = %+v, %v, %v", location, found, err)
		}
		if _, found, err := built.allWork.Locate(ctx, "A-9999"); found || err != nil {
			t.Errorf("Locate A-9999 = %v, %v; want not found", found, err)
		}
		inBin, err := built.allWork.InRecycleBin(ctx, "A-1001")
		if err != nil || inBin {
			t.Errorf("InRecycleBin A-1001 = %v, %v; want false", inBin, err)
		}
		currency, err := built.allWork.Currency(ctx, "U-1002")
		if err != nil || currency.Code != "USD" || !currency.Rate.IsOne() {
			t.Errorf("Currency = %+v, %v; want USD", currency, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
}
