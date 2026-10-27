package allwork

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
)

var priyaActs = actor.Actor{ID: "U-1002", Name: "Priya Raman"}

// newTestService returns the service on an empty database with src, at 18 Oct 2026 14:05 UTC.
func newTestService(t *testing.T, src Source) (*Service, *fake.DB) {
	t.Helper()
	db := fake.NewDB()
	tools := history.Tools{Tx: db, IDs: ids.NewGenerator(fake.NewCounters(db)), Now: func() time.Time { return fixtureNow }}
	return NewService(tools, newFakeStore(db), src), db
}

func TestQuery_DefaultYearIsCurrentYear_FR004(t *testing.T) {
	t.Parallel()
	src := newFakeSource(seedYear())
	src.currencies["U-1002"] = Currency{Code: "EUR", Rate: mustRate("0.92")}
	s, _ := newTestService(t, src)

	r, err := s.Query(t.Context(), priyaActs, Query{View: ViewGrid})

	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if r.Year != 2026 || r.Today.String() != "2026-10-18" || r.DisplayCurrency != "EUR" || !r.GeneratedAt.Equal(fixtureNow) ||
		len(r.Rows) == 0 || r.LeftOut == nil {
		t.Errorf("result: year %d, today %s, currency %s, generated %s, %d rows", r.Year, r.Today, r.DisplayCurrency, r.GeneratedAt, len(r.Rows))
	}
	// Another year with no plans has no rows, and its cards are empty.
	other, err := s.Query(t.Context(), priyaActs, Query{Year: 2030, View: ViewGrid})
	if err != nil || other.Year != 2030 || len(other.Rows) != 0 || other.Summary.PlannedCost != nil || other.Count.Total != 0 {
		t.Errorf("2030: %+v, %v", other, err)
	}
}

// A query leaves every record — the business records and the activity log — as it was.
func TestQuery_NeverWrites_FR002(t *testing.T) {
	t.Parallel()
	s, db := newTestService(t, newFakeSource(seedYear()))
	err := s.tx.Write(t.Context(), func(ctx context.Context) error { // some data first
		return plansOf(db).Put(ctx, map[string]string{"id": "P-1001"})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	before := dataOf(t, db)

	for _, v := range []string{ViewGrid, ViewCalendar, ViewBudget} {
		if _, err := s.Query(t.Context(), priyaActs, Query{View: v}); err != nil {
			t.Fatalf("Query %s: %v", v, err)
		}
	}
	if _, err := s.Locate(t.Context(), priyaActs, "A-1001"); err != nil {
		t.Fatalf("Locate: %v", err)
	}

	if after := dataOf(t, db); !reflect.DeepEqual(after, before) {
		t.Errorf("the data changed:\nbefore %v\nafter  %v", before, after)
	}
}

func plansOf(db *fake.DB) *fake.Table[map[string]string] {
	return fake.TableOf(db, "plans", func(p map[string]string) string { return p["id"] })
}

// dataOf returns everything the service could change: records, log, views, preferences and ID counters.
func dataOf(t *testing.T, db *fake.DB) []any {
	t.Helper()
	repo := newFakeStore(db)
	plans, plansErr := plansOf(db).List()
	log, logErr := historytest.NewStore(db).Entries(t.Context())
	views, viewsErr := repo.views.List()
	preferences, preferencesErr := repo.preferences.List()
	if err := errors.Join(plansErr, logErr, viewsErr, preferencesErr); err != nil {
		t.Fatalf("read the data: %v", err)
	}
	return []any{plans, log, views, preferences, fake.NewCounters(db).Used()}
}

// The Calendar shows the Grid's rows without line items and unassigned rows, and no grid columns.
func TestQuery_CalendarViewHasNoLineItemRows_FR025(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(t, newFakeSource(seedYear()))

	r, err := s.Query(t.Context(), priyaActs, Query{View: ViewCalendar})

	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	for _, row := range r.Rows {
		if row.Level == LevelLineItem || row.Level == LevelUnassigned || len(row.Cells) != 0 {
			t.Errorf("calendar row %s: level %s, %d cells", row.Key, row.Level, len(row.Cells))
		}
	}
	if !hasRow(r.Rows, "A-1001") || rowOf(r.Rows, "A-1001").HasChildren || len(r.Columns) != 0 {
		t.Errorf("activities are shown without children; columns = %+v", r.Columns)
	}
}

// The Calendar's bars and tips need today and each row's dates, Ongoing, owner, status and planned cost.
func TestQuery_TodayAndDatesForCalendar_FR026(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(t, newFakeSource(seedYear()))

	r, err := s.Query(t.Context(), priyaActs, Query{View: ViewCalendar})

	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if r.Today.String() != "2026-10-18" {
		t.Errorf("today = %s, want 2026-10-18", r.Today)
	}
	film := rowOf(r.Rows, "A-1001")
	if film.StartDate.String() != "2026-03-01" || film.EndDate.String() != "2026-05-31" || film.Ongoing ||
		film.Owner == nil || film.Owner.Name != "Priya Raman" || film.Status == nil || film.Status.Tone != ToneNeutral ||
		moneyText(film.PlannedCost) != "120000.00 USD" {
		t.Errorf("Launch film = %+v", film)
	}
	fund := rowOf(r.Rows, "A-1005")
	if !fund.Ongoing || fund.StartDate.String() != "2026-03-01" || !fund.EndDate.IsZero() || fund.Status.Label != "In progress" {
		t.Errorf("ongoing activity = %+v", fund)
	}
	campaign := rowOf(r.Rows, "C-1001")
	if campaign.StartDate.String() != "2026-01-01" || campaign.Status.Tone != ToneReview || moneyText(campaign.PlannedCost) != "275000.00 USD" {
		t.Errorf("campaign = %+v", campaign)
	}
	if undated := rowOf(r.Rows, "A-1009"); !undated.StartDate.IsZero() || !undated.EndDate.IsZero() {
		t.Errorf("an activity without dates has none: %+v", undated)
	}
}

// A deleted record is in the recycle bin; a record that still exists is located with its plan.
func TestLocate_DeletedRecordInRecycleBin_EdgeCase(t *testing.T) {
	t.Parallel()
	src := newFakeSource(seedYear())
	src.inBin["A-9001"] = true
	src.inBin["L-9001"] = true
	s, _ := newTestService(t, src)

	location, err := s.Locate(t.Context(), priyaActs, "L-1002")
	if err != nil || location != (RecordLocation{RecordID: "L-1002", Level: LevelLineItem, Name: "Film production", PlanID: "P-1001", ActivityID: "A-1001"}) {
		t.Errorf("Locate L-1002 = %+v, %v", location, err)
	}
	for id, want := range map[string]string{
		"A-9001": "This activity was deleted — it is in the recycle bin",
		"L-9001": "This line item was deleted — it is in the recycle bin",
	} {
		_, err := s.Locate(t.Context(), priyaActs, id)
		assertNotFound(t, err, want)
	}
}

// An ID of no planning level is a record that does not exist.
func TestLocate_UnknownRecord_EdgeCase(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(t, newFakeSource(seedYear()))

	for id, want := range map[string]string{
		"P-9001":  "This plan no longer exists — someone may have deleted it.",
		"C-9001":  "This campaign no longer exists — someone may have deleted it.",
		"I-9001":  "This initiative no longer exists — someone may have deleted it.",
		"SV-1001": "This record no longer exists — someone may have deleted it.",
		"nothing": "This record no longer exists — someone may have deleted it.",
	} {
		_, err := s.Locate(t.Context(), priyaActs, id)
		assertNotFound(t, err, want)
	}
}

// Years newest first; plans by name; owners by name, inactive ones kept; the person's own teams first.
func TestOptions_YearsOwnersTeamsMineFirst_FR004(t *testing.T) {
	t.Parallel()
	src := newFakeSource(seedYear())
	src.years[2024] = YearData{Nodes: []Node{}}
	src.years[2027] = YearData{Nodes: []Node{{Level: LevelPlan, ID: "P-1003", Name: "FY2027 Global Brand"}}}
	s, _ := newTestService(t, src)

	o, err := s.Options(t.Context(), priyaActs, 0)

	if err != nil {
		t.Fatalf("Options: %v", err)
	}
	if o.Year != 2026 || o.CurrentYear != 2026 || !reflect.DeepEqual(o.Years, []int{2027, 2026, 2024}) {
		t.Errorf("year %d, current %d, years %v", o.Year, o.CurrentYear, o.Years)
	}
	want := []PlanOption{
		{ID: "P-1001", Label: "FY2026 Lubricants Europe", Status: "published", Year: 2026},
		{ID: "P-1002", Label: "FY2026 Mobility China", Status: "draft", Year: 2026},
	}
	if !reflect.DeepEqual(o.Plans, want) {
		t.Errorf("plans = %+v", o.Plans)
	}
	owners := []Option{{ID: daniel.ID, Label: "Daniel Okoro"}, {ID: mei.ID, Label: "Mei Lin"},
		{ID: priya.ID, Label: "Priya Raman"}, {ID: sara.ID, Label: "Sara Novak", Inactive: true}}
	if !reflect.DeepEqual(o.Owners, owners) {
		t.Errorf("owners = %+v", o.Owners)
	}
	var teams []string
	for _, team := range o.Teams {
		teams = append(teams, team.Label)
	}
	if !reflect.DeepEqual(teams, []string{"Lubricants Europe", "Global Marketing Ops", "Mobility China"}) || !o.Teams[0].Mine {
		t.Errorf("teams = %+v, want Priya's own first", o.Teams)
	}
	if len(o.BusinessUnits) != 2 || len(o.ActivityTypes) != 6 || !o.ActivityTypes[5].Inactive ||
		!reflect.DeepEqual(o.DefaultColumns, DefaultColumns) || len(o.ColumnGroups) == 0 || o.ColumnGroups[0].Label != "Standard" {
		t.Errorf("business units %+v, activity types %+v, groups %+v", o.BusinessUnits, o.ActivityTypes, o.ColumnGroups)
	}
	other, err := s.Options(t.Context(), priyaActs, 2027)
	if err != nil || other.Year != 2027 || len(other.Plans) != 1 || len(other.Owners) != 0 {
		t.Errorf("2027 options = %+v, %v", other, err)
	}
}

// Unknown filter values and columns are ignored, the rest applies, and each is named in leftOut.
func TestQuery_UnknownIdsAndColumnsLeftOut_FR031(t *testing.T) {
	t.Parallel()
	s, _ := newTestService(t, newFakeSource(seedYear()))
	q := Query{
		View: ViewGrid, Plans: []string{"P-1001", "P-9001"}, Owners: []string{"U-9001"}, Teams: []string{"T-9001"},
		BusinessUnits: []string{"SO-9001"}, ActivityTypes: []string{"AT-9001"}, Strategic: StrategicInclude,
		ShowLevels: ShowEverything, Columns: []string{"plannedCost", "attr.SA-9001"},
		Sort: &Sort{Column: "attr.SA-9002", Direction: SortAscending}, Period: PeriodMonth,
	}

	r, err := s.Query(t.Context(), priyaActs, q)

	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	want := []LeftOut{
		{Filter: "plan", ID: "P-9001"}, {Filter: "owner", ID: "U-9001"}, {Filter: "team", ID: "T-9001"},
		{Filter: "businessUnit", ID: "SO-9001"}, {Filter: "activityType", ID: "AT-9001"},
		{Filter: "column", ID: "attr.SA-9001"}, {Filter: "sort", ID: "attr.SA-9002"},
	}
	if !reflect.DeepEqual(r.LeftOut, want) {
		t.Errorf("leftOut = %+v\nwant      %+v", r.LeftOut, want)
	}
	if r.Count.Shown != 7 || len(r.Columns) != 1 {
		t.Errorf("the rest applies: %d activities of P-1001, %d columns; want 7 and 1", r.Count.Shown, len(r.Columns))
	}
}

func assertNotFound(t *testing.T, err error, message string) {
	t.Helper()
	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeNotFound || apiErr.Message != message {
		t.Errorf("err = %v, want not_found %q", err, message)
	}
}
