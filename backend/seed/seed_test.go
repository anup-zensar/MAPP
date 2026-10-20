package seed_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"northstar/internal/activities"
	"northstar/internal/approvals"
	"northstar/internal/campaigns"
	"northstar/internal/history"
	"northstar/internal/lineitems"
	"northstar/internal/people"
	"northstar/internal/plans"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/date"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
	"northstar/seed"
)

// loadedAt is the time the tests pass to Load.
var loadedAt = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)

// The seed loads the six demo people of docs/product/demo-users-and-data.md in the order of that table, all
// active and created by NorthStar at the time given (FR-028, data model §1).
func TestLoad_SixDemoPeople_FR028(t *testing.T) {
	db := loadedStore(t)

	want := []people.Person{
		demoPerson("U-1001", "Alex Morgan", "alex.morgan@northstar.demo", "Marketing administrator"),
		demoPerson("U-1002", "Priya Raman", "priya.raman@northstar.demo", "Marketer"),
		demoPerson("U-1003", "Daniel Okoro", "daniel.okoro@northstar.demo", "Marketing team leader"),
		demoPerson("U-1004", "Mei Lin", "mei.lin@northstar.demo", "Marketer"),
		demoPerson("U-1005", "Carlos Mendes", "carlos.mendes@northstar.demo", "Business champion"),
		demoPerson("U-1006", "Sara Novak", "sara.novak@northstar.demo", "Local finance advisor"),
	}
	got := storedPeople(t, db)
	if len(got) != len(want) {
		t.Fatalf("loaded %d people, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("person %d =\n%+v\nwant\n%+v", i+1, got[i], want[i])
		}
	}
}

// Every ID comes from the readable-ID counters, so the next record of each type continues after the seed
// (FR-027, research R4, R22; the "next IDs" of the data models of 002–008).
func TestLoad_IDsComeFromTheCounters_FR027(t *testing.T) {
	db := loadedStore(t)

	want := map[string]string{
		ids.PrefixPerson: "U-1007", ids.PrefixActivityType: "AT-1007", ids.PrefixModel: "MOD-1003",
		ids.PrefixShellAttribute: "SA-1036", ids.PrefixShellOption: "SO-1111", ids.PrefixFinanceBudget: "FB-1003",
		ids.PrefixFinanceBudgetItem: "FBI-1007", ids.PrefixTeam: "T-1004", ids.PrefixTeamMembership: "TM-1009",
		ids.PrefixPlan: "P-1005", ids.PrefixCampaign: "C-1006", ids.PrefixInitiative: "I-1011", ids.PrefixActivity: "A-1013",
		ids.PrefixLineItem: "L-1011", ids.PrefixApprovalWorkflow: "WF-1003", ids.PrefixApprovalRound: "AR-1009",
		ids.PrefixExchangeRate: "XR-1125", ids.PrefixRateImport: "RI-1002",
		ids.PrefixApprovalAction: "AH-1016", ids.PrefixNotification: "NT-1010",
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	err := db.Write(t.Context(), func(ctx context.Context) error {
		for prefix, wantID := range want {
			next, err := generator.Next(ctx, prefix)
			if err != nil {
				return err
			}
			if next != wantID {
				t.Errorf("next %s ID = %s, want %s", prefix, next, wantID)
			}
		}
		return errors.New("roll back") // leave the counters as the seed left them
	})
	if err == nil || err.Error() != "roll back" {
		t.Fatalf("Write: %v", err)
	}
}

// The demo data is loaded only into an empty store; a store with any data is refused and left as it was
// (FR-028).
func TestLoad_RefusesAStoreWithData_FR028(t *testing.T) {
	db := openStore(t)
	err := db.Write(t.Context(), func(ctx context.Context) error {
		return store.Put(ctx, "sample_folders", "SF-1001", map[string]string{"id": "SF-1001"})
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	err = seed.Load(t.Context(), db, fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 0 {
		t.Errorf("people = %+v, want none", got)
	}
}

// Loading twice is refused the second time, so the demo data is never doubled (FR-028).
func TestLoad_Twice_FR028(t *testing.T) {
	db := loadedStore(t)

	err := seed.Load(t.Context(), db, fixedClock)

	if !errors.Is(err, seed.ErrNotEmpty) {
		t.Errorf("second Load = %v, want ErrNotEmpty", err)
	}
	if got := storedPeople(t, db); len(got) != 6 {
		t.Errorf("%d people after loading twice, want 6", len(got))
	}
}

// The seed writes no activity-log entries and nothing in the recycle bin: only feature 011's step will add
// the demo history (D-202, D-188).
func TestLoad_WritesNoHistory_D202(t *testing.T) {
	db := loadedStore(t)
	read(t, db, func(ctx context.Context) error {
		entries, err := history.NewJSONRepository().Entries(ctx)
		if err != nil {
			return err
		}
		bin, err := history.NewJSONRepository().BinEntries(ctx)
		if err != nil {
			return err
		}
		if len(entries) != 0 || len(bin) != 0 {
			t.Errorf("%d log entries and %d recycle-bin entries, want none", len(entries), len(bin))
		}
		return nil
	})
}

// seed.Loaded, which the seed command prints, says exactly what is stored.
func TestLoaded_SaysWhatIsLoaded_FR028(t *testing.T) {
	d := loadDemo(t)
	activeOptions := 0
	for _, o := range d.options {
		if o.Active {
			activeOptions++
		}
	}
	got := fmt.Sprintf("%d people, %d teams, %d models, %d activity types, %d Shell attributes with %d options, "+
		"%d currencies, %d finance budget items, %d plans, %d campaigns, %d initiatives, %d activities, %d line items, "+
		"%d approval workflows, %d approval rounds",
		len(d.people), len(d.teams), len(d.models), len(d.activityTypes), len(d.attributes), activeOptions,
		len(d.currencies), len(d.items), len(d.plans), len(d.campaigns), len(d.initiatives), len(d.activities),
		len(d.lineItems), len(d.workflows), len(d.rounds))
	if got != seed.Loaded {
		t.Errorf("stored: %s\nseed.Loaded: %s", got, seed.Loaded)
	}
}

// Every Shell attribute value of every planning record belongs to an attribute of the record's level and
// holds options of that attribute; a dependent value (Region, GL code) is linked to a chosen parent option
// (002 data model §9, D-048).
func TestLoad_AttributeValuesFitTheirAttributes_D048(t *testing.T) {
	d := loadDemo(t)
	check := func(record string, level settings.Level, values settings.AttributeValues) {
		if values == nil {
			t.Errorf("%s: attributes are nil, want {}", record)
		}
		for attributeID, raw := range values {
			attribute, found := d.attributeByID[attributeID]
			if !found || attribute.Level != level {
				t.Errorf("%s: %s is not a %s-level attribute", record, attributeID, level)
				continue
			}
			for _, optionID := range optionIDs(t, attribute, raw) {
				option, found := d.optionByID[optionID]
				if !found || option.AttributeID != attributeID {
					t.Errorf("%s: %s is not an option of %s", record, optionID, attribute.Label)
					continue
				}
				if attribute.ParentAttributeID == "" {
					continue
				}
				chosen := optionIDs(t, d.attributeByID[attribute.ParentAttributeID], values[attribute.ParentAttributeID])
				if !slices.ContainsFunc(option.ParentOptionIDs, func(p string) bool { return slices.Contains(chosen, p) }) {
					t.Errorf("%s: %s '%s' is not linked to a chosen parent option %v", record, attribute.Label, option.Label, chosen)
				}
			}
		}
	}
	for _, p := range d.plans {
		check(p.ID, settings.LevelPlan, p.Attributes)
	}
	for _, c := range d.campaigns {
		check(c.ID, settings.LevelCampaign, c.Attributes)
	}
	for _, i := range d.initiatives {
		check(i.ID, settings.LevelInitiative, i.Attributes)
	}
	for _, a := range d.activities {
		check(a.ID, settings.LevelActivity, a.Attributes)
	}
	for _, l := range d.lineItems {
		check(l.ID, settings.LevelLineItem, l.Attributes)
	}
}

// optionIDs returns the option ids of a choice value: one for single choice, the list for multiple choice,
// the options of the shares for a % split; none for other types or no value.
func optionIDs(t *testing.T, attribute settings.Attribute, raw json.RawMessage) []string {
	t.Helper()
	if raw == nil {
		return nil
	}
	var list []string
	var err error
	switch attribute.Type {
	case settings.TypeSingleChoice:
		var one string
		err = json.Unmarshal(raw, &one)
		list = []string{one}
	case settings.TypeMultipleChoice:
		err = json.Unmarshal(raw, &list)
	case settings.TypeMultipleChoiceSplit:
		var shares []settings.SplitShare
		err = json.Unmarshal(raw, &shares)
		for _, s := range shares {
			list = append(list, s.OptionID)
		}
	}
	if err != nil {
		t.Errorf("value of %s: %s: %v", attribute.Label, raw, err)
	}
	return list
}

// demoData is everything the seed stored, read through the repositories.
type demoData struct {
	people        []people.Person
	teams         []people.Team
	memberships   []people.Membership
	currencies    []settings.Currency
	rates         []settings.ExchangeRate // 016: the dated rates
	rateImports   []settings.RateImport
	activityTypes []settings.ActivityType
	models        []settings.Model
	budgets       []settings.FinanceBudget
	items         []settings.FinanceBudgetItem
	attributes    []settings.Attribute
	options       []settings.Option
	plans         []plans.Plan
	campaigns     []campaigns.Campaign
	initiatives   []campaigns.Initiative
	activities    []activities.Activity
	lineItems     []lineitems.LineItem
	workflows     []approvals.Workflow
	rounds        []approvals.Round
	actions       []approvals.Action
	notifications map[string][]approvals.Notification // by recipient id

	attributeByID map[string]settings.Attribute
	optionByID    map[string]settings.Option
}

// loadDemo loads the demo data into a new store and reads it all back.
func loadDemo(t *testing.T) demoData {
	t.Helper()
	db := loadedStore(t)
	var d demoData
	read(t, db, func(ctx context.Context) error {
		peopleRepo, settingsRepo, approvalsRepo := people.NewJSONRepository(), settings.NewJSONRepository(), approvals.NewJSONRepository()
		campaignsRepo := campaigns.NewJSONRepository()
		var errs []error
		collect := func(err error) { errs = append(errs, err) }
		var err error
		d.people, err = peopleRepo.List(ctx)
		collect(err)
		d.teams, err = peopleRepo.ListTeams(ctx)
		collect(err)
		d.memberships, err = peopleRepo.ListMemberships(ctx)
		collect(err)
		d.currencies, err = settingsRepo.ListCurrencies(ctx)
		collect(err)
		d.rates, err = settingsRepo.ListRates(ctx)
		collect(err)
		d.rateImports, err = settingsRepo.ListRateImports(ctx)
		collect(err)
		d.activityTypes, err = settingsRepo.ListActivityTypes(ctx)
		collect(err)
		d.models, err = settingsRepo.ListModels(ctx)
		collect(err)
		d.budgets, err = settingsRepo.ListFinanceBudgets(ctx)
		collect(err)
		d.items, err = settingsRepo.ListFinanceBudgetItems(ctx)
		collect(err)
		d.attributes, err = settingsRepo.ListAttributes(ctx)
		collect(err)
		d.options, err = settingsRepo.ListOptions(ctx)
		collect(err)
		d.plans, err = plans.NewJSONRepository().List(ctx)
		collect(err)
		d.campaigns, err = campaignsRepo.ListCampaigns(ctx)
		collect(err)
		d.initiatives, err = campaignsRepo.ListInitiatives(ctx)
		collect(err)
		d.activities, err = activities.NewJSONRepository().List(ctx)
		collect(err)
		d.lineItems, err = lineitems.NewJSONRepository().List(ctx)
		collect(err)
		d.workflows, err = approvalsRepo.ListWorkflows(ctx)
		collect(err)
		d.rounds, err = approvalsRepo.ListRounds(ctx)
		collect(err)
		d.actions, err = approvalsRepo.ListActions(ctx)
		collect(err)
		d.notifications = map[string][]approvals.Notification{}
		for _, p := range d.people {
			d.notifications[p.ID], err = approvalsRepo.NotificationsOf(ctx, p.ID)
			collect(err)
		}
		return errors.Join(errs...)
	})
	d.attributeByID, d.optionByID = map[string]settings.Attribute{}, map[string]settings.Option{}
	for _, a := range d.attributes {
		d.attributeByID[a.ID] = a
	}
	for _, o := range d.options {
		d.optionByID[o.ID] = o
	}
	return d
}

// label returns the label of the option id.
func (d demoData) label(optionID string) string {
	return d.optionByID[optionID].Label
}

// attributeID returns the id of the attribute label at level.
func (d demoData) attributeID(t *testing.T, level settings.Level, label string) string {
	t.Helper()
	for _, a := range d.attributes {
		if a.Level == level && a.Label == label {
			return a.ID
		}
	}
	t.Fatalf("no %s-level attribute called %s", level, label)
	return ""
}

func fixedClock() time.Time {
	return loadedAt
}

func openStore(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

// loadedStore returns a new store with the demo data.
func loadedStore(t *testing.T) *store.DB {
	t.Helper()
	db := openStore(t)
	if err := seed.Load(t.Context(), db, fixedClock); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return db
}

// read runs fn in a Read of db and stops the test on its error.
func read(t *testing.T, db *store.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Read(t.Context(), fn); err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// demoPerson is a seeded person: active, created by NorthStar itself at loadedAt.
func demoPerson(id, name, email, jobTitle string) people.Person {
	return people.Person{
		ID:       id,
		Name:     name,
		Email:    email,
		JobTitle: jobTitle,
		Active:   true,
		Meta:     record.NewMeta(actor.System, loadedAt),
	}
}

// storedPeople returns every stored person, ordered by id.
func storedPeople(t *testing.T, db *store.DB) []people.Person {
	t.Helper()
	var list []people.Person
	read(t, db, func(ctx context.Context) error {
		var err error
		list, err = people.NewJSONRepository().List(ctx)
		return err
	})
	return list
}

// currentRates returns the rate of every currency current on the seed day: its latest dated rate on or
// before it (016 FR-002); USD is 1.
func (d demoData) currentRates() money.Rates {
	rates := money.Rates{money.BaseCurrency: money.One}
	latest := map[string]date.Date{}
	today := date.Today(loadedAt)
	for _, r := range d.rates {
		if !r.Date.After(today) && (latest[r.Currency].IsZero() || r.Date.After(latest[r.Currency])) {
			rates[r.Currency], latest[r.Currency] = r.Rate, r.Date
		}
	}
	return rates
}
