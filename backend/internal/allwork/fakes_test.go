package allwork

import (
	"context"
	"slices"

	"mapp/internal/platform/date"
)

// fakeSource is an in-memory Source: its years, a display currency per person, the records of the recycle
// bin, and the number of times Year was read.
type fakeSource struct {
	years      map[int]YearData
	currencies map[string]Currency
	inBin      map[string]bool
	reads      int
}

// newFakeSource returns a Source with the year 2026 of b.
func newFakeSource(b *yearBuilder) *fakeSource {
	return &fakeSource{years: map[int]YearData{2026: b.data()}, currencies: map[string]Currency{}, inBin: map[string]bool{}}
}

func (f *fakeSource) Years(context.Context) ([]int, error) {
	years := []int{}
	for y := range f.years {
		years = append(years, y)
	}
	return years, nil
}

func (f *fakeSource) Choices(_ context.Context, personID string) (Choices, error) {
	return fixtureChoices(personID), nil
}

func (f *fakeSource) Attributes(context.Context) ([]Attribute, error) {
	return fixtureAttributes, nil
}

func (f *fakeSource) Year(_ context.Context, year int, _ date.Date, _ []string) (YearData, error) {
	f.reads++
	data, found := f.years[year]
	if !found {
		return YearData{Nodes: []Node{}}, nil
	}
	return data, nil
}

func (f *fakeSource) PlanYear(ctx context.Context, planID string, year int, today date.Date) (YearData, error) {
	data, err := f.Year(ctx, year, today, nil)
	data.Nodes = slices.DeleteFunc(slices.Clone(data.Nodes), func(n Node) bool { return n.PlanID != planID })
	return data, err
}

// RatesUsed answers the rates of the fixtures (016 FR-009).
func (f *fakeSource) RatesUsed(context.Context) ([]string, error) {
	return []string{"1 USD = 0.92 EUR · rate of 18 Oct 2026 · GSAP import", "1 USD = 0.79 GBP · rate of 18 Oct 2026 · Entered by hand"}, nil
}

func (f *fakeSource) Currency(_ context.Context, personID string) (Currency, error) {
	if c, found := f.currencies[personID]; found {
		return c, nil
	}
	return usd, nil
}

func (f *fakeSource) Locate(_ context.Context, recordID string) (Location, bool, error) {
	for _, data := range f.years {
		for _, n := range data.Nodes {
			if n.ID == recordID {
				l := Location{Level: n.Level, Name: n.Name, PlanID: n.PlanID}
				if n.Level == LevelLineItem {
					l.ActivityID = n.ParentID
				}
				return l, true, nil
			}
		}
	}
	return Location{}, false, nil
}

func (f *fakeSource) InRecycleBin(_ context.Context, recordID string) (bool, error) {
	return f.inBin[recordID], nil
}

// The compiler checks here that fakeSource implements Source.
var _ Source = (*fakeSource)(nil)
