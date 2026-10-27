package seed_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"mapp/internal/access"
	"mapp/internal/access/accesstest"
	"mapp/internal/activity"
	"mapp/internal/activity/activitytest"
	"mapp/internal/approval"
	"mapp/internal/approval/approvaltest"
	"mapp/internal/campaign"
	"mapp/internal/campaign/campaigntest"
	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/lineitem"
	"mapp/internal/lineitem/lineitemtest"
	"mapp/internal/people"
	"mapp/internal/people/peopletest"
	"mapp/internal/plan"
	"mapp/internal/plan/plantest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/apitest"
	"mapp/internal/platform/date"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
	"mapp/internal/settings/settingstest"
)

type demoData struct {
	people        []people.Person
	teams         []people.Team
	memberships   []people.Membership
	currencies    []settings.Currency
	rates         []settings.ExchangeRate
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

// loadDemo reads back everything the seed stored.
func loadDemo(t *testing.T) demoData {
	t.Helper()
	db := loadedStore(t)
	var d demoData
	read(t, db, func(ctx context.Context) error {
		peopleRepo, settingsRepo, approvalsRepo := peopletest.NewStore(db), settingstest.NewStore(db), approvaltest.NewStore(db)
		campaignsRepo := campaigntest.NewStore(db)
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
		d.plans, err = plantest.NewStore(db).List(ctx)
		collect(err)
		d.campaigns, err = campaignsRepo.ListCampaigns(ctx)
		collect(err)
		d.initiatives, err = campaignsRepo.ListInitiatives(ctx)
		collect(err)
		d.activities, err = activitytest.NewStore(db).List(ctx)
		collect(err)
		d.lineItems, err = lineitemtest.NewStore(db).List(ctx)
		collect(err)
		d.workflows, err = approvalsRepo.ListWorkflows(ctx)
		collect(err)
		d.rounds, err = approvalsRepo.ListRounds(ctx)
		collect(err)
		accessRepo := accesstest.NewStore(db)
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

func (d demoData) label(optionID string) string {
	return d.optionByID[optionID].Label
}

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

// loadedStore returns a test's own copy of the demo data, which apitest loads at loadedAt.
func loadedStore(t *testing.T) *fake.DB {
	t.Helper()
	return apitest.DemoDB(t, false)
}

func read(t *testing.T, db *fake.DB, fn func(ctx context.Context) error) {
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
func storedPeople(t *testing.T, db *fake.DB) []people.Person {
	t.Helper()
	var list []people.Person
	read(t, db, func(ctx context.Context) error {
		var err error
		list, err = peopletest.NewStore(db).List(ctx)
		return err
	})
	return list
}

// currentRates returns each currency's latest dated rate on or before the seed day; USD is 1.
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
func demoLog(t *testing.T, db *fake.DB) ([]history.Entry, []history.BinEntry) {
	t.Helper()
	var entries []history.Entry
	var bin []history.BinEntry
	read(t, db, func(ctx context.Context) error {
		repo := historytest.NewStore(db)
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
