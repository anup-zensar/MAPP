// Package seed is MAPP's demo data, the fictitious records a demonstration starts from (docs/product/demo-users-and-data.md).
// go run ./cmd/seed loads it into an empty PostgreSQL database; the API tests start from it through apitest.DemoDB.
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/money"
	"mapp/internal/platform/record"
	"mapp/internal/settings"
	"mapp/internal/stores"
)

var ErrNotEmpty = errors.New("seed: the store already has data")

// Loaded says what Load loads, for the seed command's message; a test checks it against the store.
const Loaded = "7 people, 3 teams, 2 models, 6 activity types, 56 Shell attributes with 185 options, " +
	"5 currencies, 6 finance budget items, 4 plans, 5 campaigns, 10 initiatives, 12 activities, 10 line items, " +
	"5 approval workflows, 8 approval rounds, 1 approval delegation, 6 roles, 8 role assignments, 1 purchase request, " +
	"54 activity-log entries, 1 recycle-bin entry"

// demoPeople follow the table of docs/product/demo-users-and-data.md, whose order gives them U-1001 to U-1007.
var demoPeople = []struct {
	name, email, jobTitle string
}{
	{"Alex Morgan", "alex.morgan@shell.com", "Marketing administrator"},
	{"Priya Raman", "priya.raman@shell.com", "Marketer"},
	{"Daniel Okoro", "daniel.okoro@shell.com", "Marketing team leader"},
	{"Mei Lin", "mei.lin@shell.com", "Marketer"},
	{"Carlos Mendes", "carlos.mendes@shell.com", "Business champion"},
	{"Sara Novak", "sara.novak@shell.com", "Local finance advisor"},
	{leaSantos, leasEmail, requisitioner},
}

type Option func(*options)

type options struct {
	withoutHistory bool
}

// WithoutHistory leaves the log, the bin and their IDs out, so tests of their rules do not depend on the demo story.
func WithoutHistory() Option {
	return func(o *options) { o.withoutHistory = true }
}

// Load loads the demo data into the empty stores of set (ErrNotEmpty otherwise) in one Write, all or nothing; call it
// outside transactions. MAPP creates every record at now(); the approval and demo histories are dated before it. The
// steps run in the order of D-202 and D-306; only the last writes log and bin entries, in time order (D-188).
func Load(ctx context.Context, set stores.Set, now clock.Clock, opts ...Option) error {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	empty, err := set.IsEmpty(ctx)
	if err != nil {
		return err
	}
	if !empty {
		return ErrNotEmpty
	}
	d := &demo{stores: set, ids: ids.NewGenerator(set.Counters), at: now()}
	return set.Tx.Write(ctx, func(ctx context.Context) error {
		steps := []func(context.Context) error{
			d.addPeople, d.addSettings, d.addTeams, d.addAccess, d.addPlans,
			d.addGlobalBudgets,
			d.addCampaigns, d.addActivities,
			d.addLineItems, d.addApprovals,
			d.addFinancials,
			d.addSavedViews,
			d.addPurchaseRequests,
		}
		if !o.withoutHistory {
			steps = append(steps, d.addHistory)
		}
		for _, step := range steps {
			if err := step(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

type demo struct {
	stores stores.Set
	ids    *ids.Generator
	at     time.Time

	people         map[string]string
	settings       SettingsSeed
	rates          money.Rates
	teams          map[string]string
	plans          map[string]string
	planCurrencies map[string]string
	campaigns      map[string]string
	initiatives    map[string]string
	workflows      map[string]string
	activities     map[string]string
}

func (d *demo) next(ctx context.Context, prefix string) (string, error) {
	return d.ids.Next(ctx, prefix)
}

func (d *demo) meta() record.Meta {
	return record.NewMeta(actor.System, d.at)
}

func find(ids map[string]string, kind, name string) (string, error) {
	id, found := ids[name]
	if !found {
		return "", fmt.Errorf("seed: there is no %s called '%s'", kind, name)
	}
	return id, nil
}

type resolver struct {
	err error
}

func (r *resolver) id(ids map[string]string, kind, name string) string {
	if r.err != nil {
		return ""
	}
	id, err := find(ids, kind, name)
	r.err = err
	return id
}

func (d *demo) addPeople(ctx context.Context) error {
	repo := d.stores.People
	d.people = map[string]string{}
	for _, p := range demoPeople {
		id, err := d.next(ctx, ids.PrefixPerson)
		if err != nil {
			return err
		}
		person := people.Person{ID: id, Name: p.name, Email: p.email, JobTitle: p.jobTitle, Active: true, Meta: d.meta()}
		if err := repo.Put(ctx, person); err != nil {
			return err
		}
		d.people[p.name] = id
	}
	return d.addLineManagers(ctx)
}

type share struct {
	optionID, percent string
}

func value(v any) json.RawMessage {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err) // strings, bools and lists of them always encode
	}
	return encoded
}

func splitValue(shares ...share) (json.RawMessage, error) {
	list := make([]settings.SplitShare, 0, len(shares))
	for _, s := range shares {
		p, err := money.ParsePercent(s.percent)
		if err != nil {
			return nil, fmt.Errorf("seed: share %q: %w", s.percent, err)
		}
		list = append(list, settings.SplitShare{OptionID: s.optionID, Share: p})
	}
	return value(list), nil
}

func usd(text string) (money.Money, error) {
	amount, err := money.ParseAmount(text)
	if err != nil {
		return money.Money{}, fmt.Errorf("seed: amount %q: %w", text, err)
	}
	return money.USD(amount), nil
}
