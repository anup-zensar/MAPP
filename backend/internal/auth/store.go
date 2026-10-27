package auth

import (
	"context"

	"mapp/internal/people"
)

type Transactor interface {
	Read(ctx context.Context, fn func(ctx context.Context) error) error
	Write(ctx context.Context, fn func(ctx context.Context) error) error
}

// Store lists the sessions of every person ordered by id; deleting a missing session changes nothing.
type Store interface {
	Get(ctx context.Context, id string) (Session, bool, error)
	Put(ctx context.Context, s Session) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]Session, error)
}

// People is what sign-in reads of people and teams.
type People interface {
	Get(ctx context.Context, id string) (people.Person, bool, error)
	FindByEmail(ctx context.Context, email string) (people.Person, bool, error)
}
