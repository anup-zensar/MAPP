package allwork

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// danielActs is a second actor of the service tests.
var danielActs = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}

// Without a change a person sees the default columns and the plan tree's order; the columns, their order
// and the sort they choose are remembered for them alone, Name never among them, and the column chooser
// keeps offering an inactive attribute they still use (FR-019, FR-021).
func TestPreferences_DefaultsAndRemembered_FR019_FR021(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	ctx := t.Context()

	defaults, err := s.Preferences(ctx, priyaActs)
	if err != nil || !reflect.DeepEqual(defaults, Preferences{Columns: DefaultColumns}) {
		t.Fatalf("defaults = %+v, %v", defaults, err)
	}
	chosen := Preferences{Columns: []string{"plannedCost", "name", "attr.SA-1004", "id"}, Sort: &Sort{Column: "plannedCost", Direction: "desc"}}
	saved, err := s.UpdatePreferences(ctx, priyaActs, chosen)
	if err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	want := Preferences{Columns: []string{"plannedCost", "attr.SA-1004", "id"}, Sort: &Sort{Column: "plannedCost", Direction: "desc"}}
	if !reflect.DeepEqual(saved, want) {
		t.Errorf("saved = %+v, want %+v", saved, want)
	}
	if again, _ := s.Preferences(ctx, priyaActs); !reflect.DeepEqual(again, want) {
		t.Errorf("remembered = %+v, want %+v", again, want)
	}
	if daniels, _ := s.Preferences(ctx, danielActs); !reflect.DeepEqual(daniels.Columns, DefaultColumns) || daniels.Sort != nil {
		t.Errorf("Daniel's = %+v, want the defaults", daniels)
	}
	options, err := s.Options(ctx, priyaActs, 0)
	if err != nil {
		t.Fatalf("Options: %v", err)
	}
	var campaign []string
	for _, c := range options.ColumnGroups[2].Columns {
		campaign = append(campaign, c.Key)
	}
	if !reflect.DeepEqual(campaign, []string{"campaign.description", "attr.SA-1003", "attr.SA-1004"}) {
		t.Errorf("Campaign group = %q: the inactive Region Priya still shows is offered", campaign)
	}
	// Saving the same choice again changes nothing.
	if same, err := s.UpdatePreferences(ctx, priyaActs, want); err != nil || !reflect.DeepEqual(same, want) {
		t.Errorf("same choice = %+v, %v", same, err)
	}
	// An unknown column is kept (an attribute may come back), and no sort is the plan tree's order.
	kept, err := s.UpdatePreferences(ctx, priyaActs, Preferences{Columns: []string{"attr.SA-9999"}})
	if err != nil || !reflect.DeepEqual(kept, Preferences{Columns: []string{"attr.SA-9999"}}) {
		t.Errorf("kept = %+v, %v", kept, err)
	}
}

// More than 100 columns, a column chosen twice and a sort on no column of the grid are refused, all at
// once (FR-021).
func TestPreferences_RefusesDuplicatesAndTooMany_FR021(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	many := make([]string, 101)
	for i := range many {
		many[i] = "attr.SA-" + strconv.Itoa(2001+i)
	}

	_, tooMany := s.UpdatePreferences(t.Context(), priyaActs, Preferences{Columns: many, Sort: &Sort{Column: "nowhere", Direction: "asc"}})
	_, twice := s.UpdatePreferences(t.Context(), priyaActs, Preferences{Columns: []string{"id", "owner", "id"}})
	_, direction := s.UpdatePreferences(t.Context(), priyaActs, Preferences{Columns: []string{}, Sort: &Sort{Column: "id", Direction: "up"}})

	assertProblems(t, tooMany, []errs.FieldError{
		{Field: "columns", Message: "Choose at most 100 columns."},
		{Field: "sort", Message: "Choose a column of the grid to sort by."},
	})
	assertProblems(t, twice, []errs.FieldError{{Field: "columns", Message: "Each column can be chosen once."}})
	assertProblems(t, direction, []errs.FieldError{{Field: "sort", Message: "Choose a column of the grid to sort by."}})
	if p, _ := s.Preferences(t.Context(), priyaActs); !reflect.DeepEqual(p.Columns, DefaultColumns) {
		t.Errorf("a refused change saved %+v", p)
	}
}

// standardState is the state of the standard view: 2026, the grid, no filters, the default columns.
func standardState() ViewState {
	return ViewState{
		View: ViewGrid, Year: 2026, Columns: slices.Clone(DefaultColumns),
		Filters:        Filters{Strategic: StrategicInclude, ShowLevels: ShowEverything},
		Expansion:      Expansion{Mode: "default"},
		ShowUnassigned: true, Period: PeriodMonth, CalendarLayout: "timeline",
	}
}

// byMonth is the state of "Lubricants by month": Budget & cost of FY2026 Lubricants Europe.
func byMonth() *ViewState {
	s := standardState()
	s.View = ViewBudget
	s.Filters.Plans = []FilterValue{{ID: "P-1001", Label: "FY2026 Lubricants Europe"}}
	return &s
}

// A view is saved under a name that is required, at most 60 characters and unique among the person's
// views ignoring letter case — another person may use it — with a state that holds only the listed values;
// every problem comes at once (FR-029, US6 scenario 1).
func TestCreateView_NameRequiredMax60UniquePerPersonIgnoringCase_FR029(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	ctx := t.Context()

	saved, err := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "  Lubricants by month ", State: byMonth()})

	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	if saved.ID != "SV-1001" || saved.Name != "Lubricants by month" || saved.IsDefault || saved.State.View != ViewBudget ||
		!saved.CreatedAt.Equal(fixtureNow) || len(saved.State.Filters.Owners) != 0 || saved.State.Filters.Owners == nil {
		t.Errorf("saved = %+v", saved)
	}
	_, empty := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "   "})
	assertProblems(t, empty, []errs.FieldError{
		{Field: "name", Message: "Enter a name for the view."},
		{Field: "state", Message: "The view can't be saved — reload the page and try again."},
	})
	_, long := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: strings.Repeat("é", 61), State: byMonth()})
	assertProblems(t, long, []errs.FieldError{{Field: "name", Message: "Use 60 characters or fewer."}})
	if _, err := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: strings.Repeat("é", 60), State: byMonth()}); err != nil {
		t.Errorf("60 characters are allowed: %v", err)
	}
	_, taken := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "LUBRICANTS BY MONTH", State: byMonth()})
	assertProblems(t, taken, []errs.FieldError{{Field: "name", Message: "You already have a view called 'Lubricants by month'."}})
	if _, err := s.CreateView(ctx, danielActs, SavedViewCreate{Name: "Lubricants by month", State: byMonth()}); err != nil {
		t.Errorf("Daniel may use the name too: %v", err)
	}
	wrong := standardState()
	wrong.View, wrong.Filters.ActivityStatuses = "table", []string{"lost"}
	_, state := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "Wrong", State: &wrong})
	assertProblems(t, state, []errs.FieldError{{Field: "state", Message: "The view can't be saved — reload the page and try again."}})
	tooMany := standardState()
	tooMany.Columns = []string{"id", "id"}
	_, twice := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "Twice", State: &tooMany})
	assertProblems(t, twice, []errs.FieldError{{Field: "state", Message: "The view can't be saved — reload the page and try again."}})
}

// Saved views are personal (FR-030): Daniel lists none of Priya's, and cannot change or delete them — they
// are "not found" for him.
func TestViews_PersonalOnly_FR030(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	ctx := t.Context()
	priyas, err := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "Mine", State: byMonth()})
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}

	list, err := s.ListViews(ctx, danielActs)
	rename := "His now"
	_, patch := s.UpdateView(ctx, danielActs, priyas.ID, SavedViewUpdate{Name: &rename})
	_, del := s.DeleteView(ctx, danielActs, priyas.ID)

	if err != nil || list.Total != 0 || len(list.Items) != 0 {
		t.Errorf("Daniel's views = %+v, %v", list, err)
	}
	assertNotFound(t, patch, "This view no longer exists — it may have been deleted in another tab.")
	assertNotFound(t, del, "This view no longer exists — it may have been deleted in another tab.")
	if mine, _ := s.ListViews(ctx, priyaActs); mine.Total != 1 || mine.Items[0].Name != "Mine" {
		t.Errorf("Priya's views = %+v", mine)
	}
}

// One view at most is the default of each person (FR-031): making another the default takes the place,
// and the default can be removed; another person's default is untouched.
func TestViews_OneDefaultPerPerson_FR031(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	ctx := t.Context()
	first, _ := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "First", State: byMonth(), IsDefault: true})
	_, _ = s.CreateView(ctx, danielActs, SavedViewCreate{Name: "His", State: byMonth(), IsDefault: true})
	_, _ = s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "Second", State: byMonth(), IsDefault: true})

	assertDefaults(t, s, priyaActs, "Second")
	yes, no := true, false
	if _, err := s.UpdateView(ctx, priyaActs, first.ID, SavedViewUpdate{IsDefault: &yes}); err != nil {
		t.Fatalf("UpdateView: %v", err)
	}
	assertDefaults(t, s, priyaActs, "First")
	if _, err := s.UpdateView(ctx, priyaActs, first.ID, SavedViewUpdate{IsDefault: &no}); err != nil {
		t.Fatalf("UpdateView: %v", err)
	}
	assertDefaults(t, s, priyaActs)
	assertDefaults(t, s, danielActs, "His")
}

// assertDefaults checks the names of a's default views.
func assertDefaults(t *testing.T, s *Service, a actor.Actor, want ...string) {
	t.Helper()
	page, err := s.ListViews(t.Context(), a)
	if err != nil {
		t.Fatalf("ListViews: %v", err)
	}
	got := []string{}
	for _, v := range page.Items {
		if v.IsDefault {
			got = append(got, v.Name)
		}
	}
	if want == nil {
		want = []string{}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s's default views = %q, want %q", a.Name, got, want)
	}
}

// A view can be renamed, its state replaced (Update view) and deleted; deleting answers the view, so Undo
// can save it again under a new ID; the list is by name (FR-031).
func TestViews_RenameUpdateStateDelete_FR031(t *testing.T) {
	s, _ := newTestService(t, newFakeSource(seedYear()))
	ctx := t.Context()
	v, _ := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "zeta", State: byMonth()})
	_, _ = s.CreateView(ctx, priyaActs, SavedViewCreate{Name: "Alpha", State: byMonth()})

	name := "  Lubricants by quarter  "
	quarterly := byMonth()
	quarterly.Period = PeriodQuarter
	renamed, err := s.UpdateView(ctx, priyaActs, v.ID, SavedViewUpdate{Name: &name, State: quarterly})
	if err != nil || renamed.Name != "Lubricants by quarter" || renamed.State.Period != PeriodQuarter || renamed.ID != v.ID {
		t.Fatalf("UpdateView = %+v, %v", renamed, err)
	}
	same := "ALPHA"
	_, taken := s.UpdateView(ctx, priyaActs, v.ID, SavedViewUpdate{Name: &same})
	assertProblems(t, taken, []errs.FieldError{{Field: "name", Message: "You already have a view called 'Alpha'."}})
	recased := "LUBRICANTS by quarter"
	if again, err := s.UpdateView(ctx, priyaActs, v.ID, SavedViewUpdate{Name: &recased}); err != nil || again.Name != recased {
		t.Errorf("a view may change the case of its own name: %+v, %v", again, err)
	}
	list, _ := s.ListViews(ctx, priyaActs)
	if len(list.Items) != 2 || list.Items[0].Name != "Alpha" {
		t.Errorf("views by name = %+v", list.Items)
	}

	deleted, err := s.DeleteView(ctx, priyaActs, v.ID)
	if err != nil || deleted.ID != v.ID || deleted.State.Period != PeriodQuarter {
		t.Fatalf("DeleteView = %+v, %v", deleted, err)
	}
	if list, _ := s.ListViews(ctx, priyaActs); list.Total != 1 {
		t.Errorf("after delete: %+v", list)
	}
	restored, err := s.CreateView(ctx, priyaActs, SavedViewCreate{Name: deleted.Name, State: &deleted.State, IsDefault: deleted.IsDefault})
	if err != nil || restored.ID == v.ID || restored.Name != deleted.Name {
		t.Errorf("Undo = %+v, %v", restored, err)
	}
	_, gone := s.DeleteView(ctx, priyaActs, v.ID)
	assertNotFound(t, gone, "This view no longer exists — it may have been deleted in another tab.")
}

// assertProblems checks that err is validation_failed with exactly the problems want.
func assertProblems(t *testing.T, err error, want []errs.FieldError) {
	t.Helper()
	var apiErr *errs.Error
	if !errors.As(err, &apiErr) || apiErr.Code != errs.CodeValidationFailed || !reflect.DeepEqual(apiErr.Fields, want) {
		t.Errorf("err = %v %+v, want problems %+v", err, apiErr, want)
	}
}
