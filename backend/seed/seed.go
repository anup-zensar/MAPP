// Package seed is NorthStar's demo data: the fictitious records a demonstration starts from (FR-028,
// docs/product/demo-users-and-data.md). The command go run ./cmd/seed loads it into an empty data folder,
// and the API's contract tests start from it.
//
// Load writes the steps in the order of decision D-202, one file per feature: people (this file), Settings
// (settings.go, 002), teams (teams.go, 003), plans (plans.go, 004), campaigns and initiatives
// (campaigns.go, 005), approval workflows and activities (activities.go, 006), line items (lineitems.go,
// 007), approval rounds and notifications (approvals.go, 008). Every step writes through its feature's
// repository with IDs from the generator, looks up earlier records by name, and writes no activity-log
// entries (feature 011's step will add the demo history).
package seed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/money"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
	"northstar/internal/settings"
)

// ErrNotEmpty means the store already has data. The demo data is loaded only into an empty store, so it is
// never mixed with other data or loaded twice.
var ErrNotEmpty = errors.New("seed: the store already has data")

// Loaded says what Load loads, for the seed command's message; a test checks it against the store.
const Loaded = "6 people, 3 teams, 2 models, 6 activity types, 35 Shell attributes with 110 options, " +
	"5 currencies, 6 finance budget items, 3 plans, 4 campaigns, 8 initiatives, 10 activities, 8 line items, " +
	"2 approval workflows, 8 approval rounds"

// demoPeople are the demo people of docs/product/demo-users-and-data.md, in the order of its table, which
// gives them the IDs U-1001 to U-1006.
var demoPeople = []struct {
	name, email, jobTitle string
}{
	{"Alex Morgan", "alex.morgan@northstar.demo", "Marketing administrator"},
	{"Priya Raman", "priya.raman@northstar.demo", "Marketer"},
	{"Daniel Okoro", "daniel.okoro@northstar.demo", "Marketing team leader"},
	{"Mei Lin", "mei.lin@northstar.demo", "Marketer"},
	{"Carlos Mendes", "carlos.mendes@northstar.demo", "Business champion"},
	{"Sara Novak", "sara.novak@northstar.demo", "Local finance advisor"},
}

// Load loads the demo data into db, which must be empty (ErrNotEmpty otherwise), in one Write: all of it,
// or nothing when something fails. NorthStar itself (actor.System) creates every record, at the time now
// gives, so tests can fix it; the approval history is dated before that time. Call Load outside
// transactions.
func Load(ctx context.Context, db *store.DB, now clock.Clock) error {
	if !db.IsEmpty() {
		return ErrNotEmpty
	}
	d := &demo{ids: ids.NewGenerator(ids.NewJSONCounters()), at: now()}
	return db.Write(ctx, func(ctx context.Context) error {
		steps := []func(context.Context) error{
			d.addPeople, d.addSettings, d.addTeams, d.addPlans, d.addCampaigns, d.addActivities,
			d.addLineItems, d.addApprovals,
		}
		for _, step := range steps {
			if err := step(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

// demo is the state of one Load: the generator, the seed time, and the IDs of the records written so far,
// by name, so that each step finds what the earlier ones wrote.
type demo struct {
	ids *ids.Generator
	at  time.Time // the seed time: NorthStar creates every record then

	people      map[string]string // person name → ID
	settings    SettingsSeed      // Settings records by name
	teams       map[string]string // team name → ID
	plans       map[string]string // plan name → ID
	campaigns   map[string]string // "<plan>/<campaign>" → ID
	initiatives map[string]string // "<campaign>/<initiative>" → ID
	workflows   map[string]string // approval workflow name → ID
	activities  map[string]string // "<plan>/<activity>" → ID
}

// next returns the next readable ID of prefix.
func (d *demo) next(ctx context.Context, prefix string) (string, error) {
	return d.ids.Next(ctx, prefix)
}

// meta is the bookkeeping of a record NorthStar creates at the seed time.
func (d *demo) meta() record.Meta {
	return record.NewMeta(actor.System, d.at)
}

// find returns the ID of name in ids, or an error naming what is missing: a seed step refers to a record an
// earlier step should have written.
func find(ids map[string]string, kind, name string) (string, error) {
	id, found := ids[name]
	if !found {
		return "", fmt.Errorf("seed: there is no %s called '%s'", kind, name)
	}
	return id, nil
}

// resolver looks up many names in a row and keeps the first error, so a step can build its records and
// check once: every lookup after a failed one returns "".
type resolver struct {
	err error
}

// id returns the ID of name in ids; the first missing name is kept as the resolver's error.
func (r *resolver) id(ids map[string]string, kind, name string) string {
	if r.err != nil {
		return ""
	}
	id, err := find(ids, kind, name)
	r.err = err
	return id
}

// addPeople creates the demo people through the people repository. Each gets the next ID of the counter
// U, like every person created later, so IDs are never given out twice (FR-027).
func (d *demo) addPeople(ctx context.Context) error {
	repo := people.NewJSONRepository()
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
	return nil
}

// Shell attribute values, as the planning records keep them (002 data model §9).

// share is one option of a % split, with its share as text ("60.00").
type share struct {
	optionID, percent string
}

// value encodes v as the JSON of an attribute value.
func value(v any) json.RawMessage {
	encoded, err := json.Marshal(v)
	if err != nil {
		panic(err) // strings, bools and lists of them always encode
	}
	return encoded
}

// splitValue encodes the shares of a multiple-choice-with-%-split value.
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

// usd returns the US-dollar amount of text, such as "1200000.00".
func usd(text string) (money.Money, error) {
	amount, err := money.ParseAmount(text)
	if err != nil {
		return money.Money{}, fmt.Errorf("seed: amount %q: %w", text, err)
	}
	return money.USD(amount), nil
}
