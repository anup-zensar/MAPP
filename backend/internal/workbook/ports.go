package workbook

import (
	"context"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
)

// Planning reads the plan (not_found when unseen) and deletes for an import's undo; it refuses with *errs.Error.
type Planning interface {
	Plan(ctx context.Context, a actor.Actor, planID string) (Plan, error)
	PlanFacts(ctx context.Context, planID string) (authz.PlanFacts, bool, error)
	Prepare(ctx context.Context, a actor.Actor, p Plan) (Writer, error)
	Delete(ctx context.Context, a actor.Actor, level Level, id string) error
	Exists(ctx context.Context, level Level, id string) (bool, error)
}

// Writer applies the single operations' rules, reading what they share once. A created record may
// have a provisional id until Flush stores what is kept; the applier flushes after each sheet.
type Writer interface {
	Create(ctx context.Context, a actor.Actor, r NewRecord) (string, error)
	Update(ctx context.Context, a actor.Actor, c Change) error
	Flush(ctx context.Context) (map[string]string, error)
}

type NewRecord struct {
	Level    Level
	ParentID string
	Values   Values
}

type Change struct {
	ID      string
	Values  Values
	Present map[string]bool
}

type Plan struct {
	ID       string
	Name     string
	Currency string
	Records  []Stored
	Choices  Choices
}

type Stored struct {
	Level    Level
	ID       string
	ParentID string
	Status   string
	Values   Values
}

type Choices struct {
	People        []Person
	ActivityTypes []Option
	ModelName     string
	Workflows     []Option
	Currencies    []Option
	Attributes    []Attribute
}

type Person struct {
	ID     string
	Email  string
	Active bool
}

type Option struct {
	ID     string
	Label  string
	Active bool
}

type Attribute struct {
	ID      string
	Label   string
	Level   Level
	Type    string
	Marking string
	Options []Option
}
