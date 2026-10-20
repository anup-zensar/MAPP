package plans

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"northstar/internal/history"
	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
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

// fakeSettings is Settings with two active models, one inactive, two finance budgets and a few Shell
// attributes. CheckAttributes keeps every value and adds problems, when the test sets them.
type fakeSettings struct {
	models            map[string]settings.Model
	budgets           map[string]settings.FinanceBudget
	attributes        []settings.Attribute
	options           map[string]settings.Option
	attributeProblems []errs.FieldError
}

func newFakeSettings() *fakeSettings {
	model := func(id, name, r1, r2, deal string, active bool) settings.Model {
		p1, _ := money.ParsePercent(r1)
		p2, _ := money.ParsePercent(r2)
		size, _ := money.ParseAmount(deal)
		return settings.Model{ID: id, Name: name, InqToMqlRate: p1, MqlToDealRate: p2, AverageDealSize: money.USD(size),
			ActivityTypeIDs: []string{"AT-1", "AT-2"}, Active: active}
	}
	return &fakeSettings{
		models: map[string]settings.Model{
			lubricantsModel: model(lubricantsModel, "Lubricants 1.0", "2.00", "5.00", "10000.00", true),
			mobilityModel:   model(mobilityModel, "Mobility 1.0", "3.00", "4.00", "5000.00", true),
			oldModel:        model(oldModel, "Old model", "1.00", "1.00", "1000.00", false),
		},
		budgets: map[string]settings.FinanceBudget{
			budget2026: {ID: budget2026, Year: 2026, Name: "2026 Lubricants Budget"},
			budget2027: {ID: budget2027, Year: 2027, Name: "2027 Lubricants Budget"},
		},
		attributes: []settings.Attribute{
			{ID: businessUnit, Label: "Business unit", Level: settings.LevelPlan, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToPublish, Position: 1, Role: settings.RoleBusinessUnit, Active: true},
			{ID: smp, Label: "Strategic Marketing Plan (SMP)", Level: settings.LevelPlan, Type: settings.TypeYesNo,
				Marking: settings.MarkingOptional, Position: 2, Condition: []string{mobility}, Active: true},
			{ID: brands, Label: "Brands", Level: settings.LevelActivity, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToSubmit, Position: 1, Condition: []string{lubricants}, Active: true},
			{ID: sector, Label: "Sector", Level: settings.LevelActivity, Type: settings.TypeMultipleChoice,
				Marking: settings.MarkingNeededToSubmit, Position: 2, Condition: []string{lubricants}, Active: true},
			{ID: functionMob, Label: "Function (Mobility)", Level: settings.LevelActivity, Type: settings.TypeSingleChoice,
				Marking: settings.MarkingOptional, Position: 3, Condition: []string{mobility}, Active: true},
		},
		options: map[string]settings.Option{
			mobility:   {ID: mobility, AttributeID: businessUnit, Label: "Mobility", Position: 1, Active: true},
			lubricants: {ID: lubricants, AttributeID: businessUnit, Label: "Lubricants", Position: 2, Active: true},
			marine:     {ID: marine, AttributeID: businessUnit, Label: "Marine", Position: 3, Active: false},
		},
	}
}

// fakeCurrencies are the currencies of the tests (016): USD, EUR, GBP active, INR inactive, at the rates of
// the 016 examples.
var fakeCurrencies = map[string]settings.Currency{
	"USD": {Code: "USD", Rate: money.One, Active: true},
	"EUR": {Code: "EUR", Rate: money.RateFromMicros(920_000), Active: true},
	"GBP": {Code: "GBP", Rate: money.RateFromMicros(790_000), Active: true},
	"INR": {Code: "INR", Rate: money.RateFromMicros(83_500_000), Active: false},
}

func (f *fakeSettings) Currency(_ context.Context, code string) (settings.Currency, bool, error) {
	c, found := fakeCurrencies[code]
	return c, found, nil
}

func (f *fakeSettings) Rates(context.Context) (money.Rates, error) {
	rates := money.Rates{}
	for code, c := range fakeCurrencies {
		rates[code] = c.Rate
	}
	return rates, nil
}

func (f *fakeSettings) Model(_ context.Context, id string) (settings.Model, bool, error) {
	m, found := f.models[id]
	return m, found, nil
}

func (f *fakeSettings) Models(context.Context) ([]settings.Model, error) {
	models := slices.Collect(maps.Values(f.models))
	slices.SortFunc(models, func(x, y settings.Model) int { return strings.Compare(x.Name, y.Name) })
	return models, nil
}

func (f *fakeSettings) FinanceBudget(_ context.Context, id string) (settings.FinanceBudget, bool, error) {
	b, found := f.budgets[id]
	return b, found, nil
}

func (f *fakeSettings) Attributes(_ context.Context, level settings.Level) ([]settings.Attribute, error) {
	ofLevel := []settings.Attribute{}
	for _, a := range f.attributes {
		if a.Level == level {
			ofLevel = append(ofLevel, a)
		}
	}
	return ofLevel, nil
}

func (f *fakeSettings) Option(_ context.Context, id string) (settings.Option, bool, error) {
	o, found := f.options[id]
	return o, found, nil
}

func (f *fakeSettings) CheckAttributes(ctx context.Context, level settings.Level, in settings.AttributeInput) (settings.AttributeResult, error) {
	missing, _ := f.MissingAttributes(ctx, level, in.Values, nil)
	return settings.AttributeResult{
		Problems: slices.Clone(f.attributeProblems), Cleared: []settings.ClearedValue{}, Missing: missing,
		Values: maps.Clone(in.Values),
	}, nil
}

// MissingAttributes reports the Business unit when it is empty.
func (f *fakeSettings) MissingAttributes(ctx context.Context, _ settings.Level, values settings.AttributeValues, _ []string) ([]settings.MissingField, error) {
	if unit, _ := f.BusinessUnit(ctx, values); len(unit) > 0 {
		return []settings.MissingField{}, nil
	}
	return []settings.MissingField{{Field: "attributes." + businessUnit, Label: "Business unit"}}, nil
}

func (f *fakeSettings) BusinessUnit(_ context.Context, values settings.AttributeValues) ([]string, error) {
	unit := []string{}
	_ = json.Unmarshal(values[businessUnit], &unit)
	return unit, nil
}

// DescribeAttribute names the attribute and the labels of its options.
func (f *fakeSettings) DescribeAttribute(_ context.Context, field string, value any) (string, string, error) {
	id := strings.TrimPrefix(field, "attributes.")
	i := slices.IndexFunc(f.attributes, func(a settings.Attribute) bool { return a.ID == id })
	if i < 0 {
		return "", "", nil
	}
	var labels []string
	list, _ := value.([]any)
	for _, item := range list {
		labels = append(labels, f.options[item.(string)].Label)
	}
	return f.attributes[i].Label, strings.Join(labels, ", "), nil
}

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

// fixture is the plans service on a temporary store, with its fakes and history.
type fixture struct {
	service   *Service
	history   *history.Service
	db        *store.DB
	clock     *testClock
	settings  *fakeSettings
	directory *fakeDirectory
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	clock := &testClock{now: start}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	hist := history.NewService(db, history.NewJSONRepository(), generator, clock.Now)
	f := &fixture{history: hist, db: db, clock: clock, settings: newFakeSettings(), directory: newFakeDirectory()}
	f.service = NewService(db, NewJSONRepository(), generator, hist, f.settings, f.directory, clock.Now)
	for _, kind := range f.service.Kinds() {
		if err := hist.Register(kind); err != nil {
			t.Fatalf("Register(%s): %v", kind.Name, err)
		}
	}
	return f
}

// addLevel registers a fake level with facts and returns it.
func (f *fixture) addLevel(t *testing.T, name string, facts map[string]Facts) *fakeLevel {
	t.Helper()
	l := &fakeLevel{name: name, facts: facts}
	if err := f.service.RegisterLevel(l.level()); err != nil {
		t.Fatalf("RegisterLevel(%s): %v", name, err)
	}
	return l
}

// input is a complete plan form: FY2026 Lubricants Test with Lubricants 1.0, Lubricants Europe and Priya
// as owner, business unit Lubricants.
func input(name string) PlanInput {
	return PlanInput{
		Name: name, Year: ptr(2026), ModelID: ptr(lubricantsModel), TeamIDs: []string{teamLubricants},
		OwnerID: ptr(priya.ID), DefaultSpread: string(SpreadEvenly), GoalStage: string(StageINQ),
		Attributes: settings.AttributeValues{businessUnit: json.RawMessage(`["` + lubricants + `"]`)},
	}
}

// updateOf is the form of the stored plan p, as the edit page sends it unchanged.
func updateOf(p Plan) PlanUpdate {
	u := PlanUpdate{
		Version: ptr(p.Version), Name: p.Name, ModelID: ptr(p.ModelID), TeamIDs: slices.Clone(p.TeamIDs), AOP: p.AOP,
		Description: p.Description, DefaultSpread: string(p.DefaultSpread), BCPIR: p.BCPIR,
		GoalStage: string(p.GoalStage), Attributes: maps.Clone(p.Attributes),
	}
	if p.OwnerID != "" {
		u.OwnerID = ptr(p.OwnerID)
	}
	if p.Budget != nil {
		u.Budget = &money.Input{Amount: p.Budget.String(), Currency: "USD"}
	}
	if p.DefaultFinanceBudgetID != "" {
		u.DefaultFinanceBudgetID = ptr(p.DefaultFinanceBudgetID)
	}
	if p.GoalAmount != "" {
		u.GoalAmount = ptr(p.GoalAmount)
	}
	return u
}

// create creates a plan as a and stops the test when that fails.
func (f *fixture) create(t *testing.T, a actor.Actor, in PlanInput) PlanView {
	t.Helper()
	view, err := f.service.Create(t.Context(), a, in)
	if err != nil {
		t.Fatalf("Create(%q): %v", in.Name, err)
	}
	return view
}

// stored reads the plan id as stored.
func (f *fixture) stored(t *testing.T, id string) Plan {
	t.Helper()
	var p Plan
	err := f.db.Read(t.Context(), func(ctx context.Context) error {
		var err error
		p, err = f.service.stored(ctx, id)
		return err
	})
	if err != nil {
		t.Fatalf("stored(%s): %v", id, err)
	}
	return p
}

// entries returns the activity-log entries, newest first.
func (f *fixture) entries(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), priya, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

// ptr returns a pointer to v.
func ptr[T any](v T) *T {
	return &v
}

// usd returns the USD amount of text, such as "1200000.00".
func usd(t *testing.T, text string) money.Amount {
	t.Helper()
	amount, err := money.ParseAmount(text)
	if err != nil {
		t.Fatalf("ParseAmount(%q): %v", text, err)
	}
	return amount
}

// assertError checks that err is exactly want.
func assertError(t *testing.T, err error, want *errs.Error) {
	t.Helper()
	var got *errs.Error
	if !errors.As(err, &got) || !reflect.DeepEqual(got, want) {
		t.Errorf("error = %#v\nwant    %#v", err, want)
	}
}

// problems returns the validation error of the field problems, given as field and message pairs.
func problems(pairs ...string) *errs.Error {
	var fields []errs.FieldError
	for i := 0; i+1 < len(pairs); i += 2 {
		fields = append(fields, errs.FieldError{Field: pairs[i], Message: pairs[i+1]})
	}
	return errs.Validation(fields...)
}

// testClock is a clock that shows a fixed time until Advance moves it.
type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time          { return c.now }
func (c *testClock) Advance(d time.Duration) { c.now = c.now.Add(d) }
