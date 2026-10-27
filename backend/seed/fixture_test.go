package seed_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"mapp/internal/access"
	"mapp/internal/activity"
	"mapp/internal/approval"
	"mapp/internal/campaign"
	"mapp/internal/history"
	"mapp/internal/lineitem"
	"mapp/internal/people"
	"mapp/internal/plan"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/date"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/platform/store"
	"mapp/internal/settings"
)

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
	plans         []plan.Plan
	campaigns     []campaign.Campaign
	initiatives   []campaign.Initiative
	activities    []activity.Activity
	lineItems     []lineitem.LineItem
	workflows     []approval.Workflow
	rounds        []approval.Round
	roles         []access.Role
	assignments   []access.Assignment
	accessRules   access.Rules
	actions       []approval.Action
	notifications map[string][]approval.Notification // by recipient id
	delegations   []approval.Delegation

	attributeByID map[string]settings.Attribute
	optionByID    map[string]settings.Option
}

// loadDemo loads the demo data into a new store and reads it all back.
func loadDemo(t *testing.T) demoData {
	t.Helper()
	db := loadedStore(t)
	var d demoData
	read(t, db, func(ctx context.Context) error {
		peopleRepo, settingsRepo, approvalsRepo := people.NewJSONStore(), settings.NewJSONStore(), approval.NewJSONStore()
		campaignsRepo := campaign.NewJSONStore()
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
		d.plans, err = plan.NewJSONStore().List(ctx)
		collect(err)
		d.campaigns, err = campaignsRepo.ListCampaigns(ctx)
		collect(err)
		d.initiatives, err = campaignsRepo.ListInitiatives(ctx)
		collect(err)
		d.activities, err = activity.NewJSONStore().List(ctx)
		collect(err)
		d.lineItems, err = lineitem.NewJSONStore().List(ctx)
		collect(err)
		d.workflows, err = approvalsRepo.ListWorkflows(ctx)
		collect(err)
		d.rounds, err = approvalsRepo.ListRounds(ctx)
		collect(err)
		accessRepo := access.NewJSONStore()
		d.roles, err = accessRepo.ListRoles(ctx)
		collect(err)
		d.assignments, err = accessRepo.ListAssignments(ctx)
		collect(err)
		d.accessRules, _, err = accessRepo.GetRules(ctx)
		collect(err)
		d.actions, err = approvalsRepo.ListActions(ctx)
		collect(err)
		d.delegations, err = approvalsRepo.ListDelegations(ctx)
		collect(err)
		d.notifications = map[string][]approval.Notification{}
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

func openStore(t *testing.T, dir string) *store.DB {
	t.Helper()
	db, err := store.Open(dir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}

// loadedStore returns a new store with a copy of the demo data that Load stored at loadedAt once per test binary.
func loadedStore(t *testing.T) *store.DB {
	t.Helper()
	dir := t.TempDir()
	apitest.CopyDemoData(t, dir, false)
	return openStore(t, dir)
}

// read runs fn in a Read of db and stops the test on its error.
func read(t *testing.T, db *store.DB, fn func(ctx context.Context) error) {
	t.Helper()
	if err := db.Read(t.Context(), fn); err != nil {
		t.Fatalf("Read: %v", err)
	}
}

// demoPerson is a seeded person: active, created by MAPP itself at loadedAt.
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
		list, err = people.NewJSONStore().List(ctx)
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

// demoLog reads the activity log and the recycle bin the seed wrote.
func demoLog(t *testing.T, db *store.DB) ([]history.Entry, []history.BinEntry) {
	t.Helper()
	var entries []history.Entry
	var bin []history.BinEntry
	read(t, db, func(ctx context.Context) error {
		repo := history.NewJSONStore()
		var err error
		if entries, err = repo.Entries(ctx); err != nil {
			return err
		}
		bin, err = repo.BinEntries(ctx)
		return err
	})
	return entries, bin
}

// entryText is an entry as a failure message shows it.
func entryText(e history.Entry) string {
	return fmt.Sprintf("%s %s %s %s %s %s '%s'", e.ID, e.At.Format(time.RFC3339), e.Actor.Name, e.Verb, e.RecordType, e.RecordID, e.RecordName)
}
