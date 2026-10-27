package plans

import (
	"context"
	"maps"
	"slices"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
)

// The service tests run on fakes of Settings, Directory and the levels below the plan (004 plan "Tests"),
// with a fixed clock: 12 Mar 2026 09:00 UTC, so the current year is 2026.

// The people of these tests.
var (
	priya  = actor.Actor{ID: "U-1002", Name: "Priya Raman"}
	daniel = actor.Actor{ID: "U-1003", Name: "Daniel Okoro"}
	nina   = actor.Actor{ID: "U-1007", Name: "Nina Berg"} // in no team
)

// start is the time on the clock when every test begins.
var start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The Settings of the fakes.
const (
	lubricantsModel = "MOD-1"   // Lubricants 1.0: 2.00 %, 5.00 %, 10,000.00
	mobilityModel   = "MOD-2"   // Mobility 1.0: 3.00 %, 4.00 %, 5,000.00
	oldModel        = "MOD-3"   // Old model, inactive
	budget2026      = "FB-2026" // 2026 Lubricants Budget
	budget2027      = "FB-2027" // 2027 Lubricants Budget
	businessUnit    = "SA-BU"   // Business unit (plan, needed to publish)
	smp             = "SA-SMP"  // Strategic Marketing Plan (SMP) (plan, yes/no, shown with Mobility)
	brands          = "SA-BRD"  // Brands (activity, shown with Lubricants)
	sector          = "SA-SEC"  // Sector (activity, shown with Lubricants)
	functionMob     = "SA-FUN"  // Function (Mobility) (activity, shown with Mobility)
	mobility        = "SO-MOB"
	lubricants      = "SO-LUB"
	marine          = "SO-MAR" // inactive
)

// The teams of the fakes.
const (
	teamLubricants = "T-1001" // Lubricants Europe: Priya, Daniel, Sara
	teamMobility   = "T-1002" // Mobility China: Mei, Carlos, Sara
	teamGlobal     = "T-1003" // Global Marketing Ops: Alex, Carlos
	teamClosed     = "T-1009" // Closed team, inactive: Priya
)

// fakeDirectory is Directory with the demo people and teams, one inactive person and one inactive team.
type fakeDirectory struct {
	people  map[string]people.PersonRef
	teams   map[string]people.TeamRef
	members map[string][]string
	rates   map[string]people.DisplayRate // person id → display currency; USD when missing
}

func newFakeDirectory() *fakeDirectory {
	person := func(id, name string, active bool) people.PersonRef {
		return people.PersonRef{ID: id, Name: name, Active: active}
	}
	return &fakeDirectory{
		people: map[string]people.PersonRef{
			"U-1001": person("U-1001", "Alex Morgan", true), "U-1002": person("U-1002", "Priya Raman", true),
			"U-1003": person("U-1003", "Daniel Okoro", true), "U-1004": person("U-1004", "Mei Lin", true),
			"U-1005": person("U-1005", "Carlos Mendes", true), "U-1006": person("U-1006", "Sara Novak", true),
			"U-1007": person("U-1007", "Nina Berg", true), "U-1009": person("U-1009", "Old Person", false),
		},
		teams: map[string]people.TeamRef{
			teamLubricants: {ID: teamLubricants, Name: "Lubricants Europe", Colour: "blue", Active: true},
			teamMobility:   {ID: teamMobility, Name: "Mobility China", Colour: "violet", Active: true},
			teamGlobal:     {ID: teamGlobal, Name: "Global Marketing Ops", Colour: "teal", Active: true},
			teamClosed:     {ID: teamClosed, Name: "Closed team", Colour: "slate", Active: false},
		},
		members: map[string][]string{
			teamLubricants: {"U-1002", "U-1003", "U-1006"},
			teamMobility:   {"U-1004", "U-1005", "U-1006"},
			teamGlobal:     {"U-1001", "U-1005"},
			teamClosed:     {"U-1002", "U-1009"},
		},
		rates: map[string]people.DisplayRate{},
	}
}

func (d *fakeDirectory) PersonRefs(_ context.Context, ids []string) (map[string]people.PersonRef, error) {
	refs := map[string]people.PersonRef{}
	for _, id := range ids {
		if p, found := d.people[id]; found {
			refs[id] = p
		}
	}
	return refs, nil
}

func (d *fakeDirectory) TeamRefs(_ context.Context, ids []string) (map[string]people.TeamRef, error) {
	refs := map[string]people.TeamRef{}
	for _, id := range ids {
		if t, found := d.teams[id]; found {
			refs[id] = t
		}
	}
	return refs, nil
}

func (d *fakeDirectory) MemberIDs(_ context.Context, teamIDs []string) (map[string][]string, error) {
	members := map[string][]string{}
	for _, id := range teamIDs {
		members[id] = d.members[id]
	}
	return members, nil
}

func (d *fakeDirectory) TeamIDsOf(_ context.Context, personID string) ([]string, error) {
	teamIDs := []string{}
	for _, id := range slices.Sorted(maps.Keys(d.members)) {
		if slices.Contains(d.members[id], personID) {
			teamIDs = append(teamIDs, id)
		}
	}
	return teamIDs, nil
}

func (d *fakeDirectory) DisplayRateOf(_ context.Context, personID string) (people.DisplayRate, error) {
	if rate, found := d.rates[personID]; found {
		return rate, nil
	}
	return people.DisplayRate{Code: "USD", Rate: "1"}, nil
}

// fakeLevel is a level below the plan whose facts the test sets, and which counts its calls.
type fakeLevel struct {
	name       string
	facts      map[string]Facts                              // plan id → its facts
	disallowed []errs.RecordRef                              // what ActivitiesWithTypeNotAllowed answers
	factCalls  int                                           // calls of AddFacts
	copy       func(ctx context.Context, job *CopyJob) error // what Copy does; nil copies nothing
	copyJobs   []CopyJob                                     // each Copy call's job, as it was then
}

// level returns the fake as the Level it registers.
func (l *fakeLevel) level() Level {
	return Level{
		Name: l.name,
		AddFacts: func(_ context.Context, planIDs []string, facts map[string]*Facts) error {
			l.factCalls++
			for _, id := range planIDs {
				if f, found := l.facts[id]; found {
					*facts[id] = f
				}
			}
			return nil
		},
		Copy: func(ctx context.Context, job *CopyJob) error {
			seen := *job
			seen.NewIDs, seen.Counts = maps.Clone(job.NewIDs), maps.Clone(job.Counts)
			l.copyJobs = append(l.copyJobs, seen)
			if l.copy == nil {
				return nil
			}
			return l.copy(ctx, job)
		},
		ActivitiesWithTypeNotAllowed: func(context.Context, string, []string) ([]errs.RecordRef, error) {
			return l.disallowed, nil
		},
	}
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
