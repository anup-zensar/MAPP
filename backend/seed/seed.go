// Package seed is NorthStar's demo data: the fictitious records a demonstration starts from (FR-028,
// docs/product/demo-users-and-data.md). The command go run ./cmd/seed loads it into an empty data folder,
// and the API's contract tests start from it. Each later feature adds its own records here.
package seed

import (
	"context"
	"errors"
	"time"

	"northstar/internal/people"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/record"
	"northstar/internal/platform/store"
)

// ErrNotEmpty means the store already has data. The demo data is loaded only into an empty store, so it is
// never mixed with other data or loaded twice.
var ErrNotEmpty = errors.New("seed: the store already has data")

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
// gives, so tests can fix it. Call Load outside transactions.
func Load(ctx context.Context, db *store.DB, now clock.Clock) error {
	if !db.IsEmpty() {
		return ErrNotEmpty
	}
	generator := ids.NewGenerator(ids.NewJSONCounters())
	createdAt := now()
	return db.Write(ctx, func(ctx context.Context) error {
		return addPeople(ctx, generator, createdAt)
	})
}

// addPeople creates the demo people through the people repository. Each gets the next ID of the counter
// U, like every person created later, so IDs are never given out twice (FR-027).
func addPeople(ctx context.Context, generator *ids.Generator, createdAt time.Time) error {
	repo := people.NewJSONRepository()
	for _, p := range demoPeople {
		id, err := generator.Next(ctx, "U")
		if err != nil {
			return err
		}
		person := people.Person{
			ID:       id,
			Name:     p.name,
			Email:    p.email,
			JobTitle: p.jobTitle,
			Active:   true,
			Meta:     record.NewMeta(actor.System, createdAt),
		}
		if err := repo.Put(ctx, person); err != nil {
			return err
		}
	}
	return nil
}
