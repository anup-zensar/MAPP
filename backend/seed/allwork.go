package seed

import (
	"context"
	"slices"

	"mapp/internal/allwork"
	"mapp/internal/platform/ids"
)

// The saved views of feature 010 (plan "Demo data"): two of Priya Raman's, neither her default, so every
// visit starts from the standard view in demonstrations and tests. Saved views are personal settings, not
// business records (D-121).

// addSavedViews saves Priya's views "Lubricants by month" (Budget & cost of FY2026 Lubricants Europe, by
// month) and "Declined activities" (the grid of 2026 with Activity status Declined).
func (d *demo) addSavedViews(ctx context.Context) error {
	var r resolver
	priya := r.id(d.people, "person", "Priya Raman")
	lubricants := r.id(d.plans, "plan", "FY2026 Lubricants Europe")
	if r.err != nil {
		return r.err
	}
	byMonth := standardViewState()
	byMonth.View = allwork.ViewBudget
	byMonth.Filters.Plans = []allwork.FilterValue{{ID: lubricants, Label: "FY2026 Lubricants Europe"}}
	declined := standardViewState()
	declined.Filters.ActivityStatuses = []string{"declined"}

	repo := d.stores.AllWork
	for _, v := range []struct {
		name  string
		state allwork.ViewState
	}{{"Lubricants by month", byMonth}, {"Declined activities", declined}} {
		id, err := d.next(ctx, ids.PrefixSavedView)
		if err != nil {
			return err
		}
		view := allwork.StoredView{ID: id, PersonID: priya, Name: v.name, State: v.state, CreatedAt: d.at, UpdatedAt: d.at}
		if err := repo.PutView(ctx, view); err != nil {
			return err
		}
	}
	return nil
}

// standardViewState is the state of All work's standard view in 2026: the grid, no filters, the default
// columns, plans, campaigns and initiatives open, by month (FR-009).
func standardViewState() allwork.ViewState {
	return allwork.ViewState{
		View: allwork.ViewGrid, Year: 2026, Columns: slices.Clone(allwork.DefaultColumns),
		Filters: allwork.Filters{
			Plans: []allwork.FilterValue{}, PlanStatuses: []string{}, ContainerStatuses: []string{}, ActivityStatuses: []string{},
			Owners: []allwork.FilterValue{}, Teams: []allwork.FilterValue{}, BusinessUnits: []allwork.FilterValue{},
			ActivityTypes: []allwork.FilterValue{}, Strategic: allwork.StrategicInclude, ShowLevels: allwork.ShowEverything,
		},
		Expansion:      allwork.Expansion{Mode: "default", Toggled: []string{}},
		ShowUnassigned: true, Period: allwork.PeriodMonth, CalendarLayout: "timeline",
	}
}
