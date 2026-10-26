package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var embeddedMigrations embed.FS

func Migrations() fs.FS {
	files, err := fs.Sub(embeddedMigrations, "migrations")
	if err != nil {
		panic(err) // the embedded directory is fixed at build time
	}
	return files
}

// Migrate applies every pending migration; a PostgreSQL session lock makes concurrent runs wait their turn.
func Migrate(ctx context.Context, d *DB) error {
	return withMigrations(d, func(provider *goose.Provider) error {
		if _, err := provider.Up(ctx); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		return nil
	})
}

// HasPending reports whether a migration is still to be applied, without waiting for a running Migrate.
func HasPending(ctx context.Context, d *DB) (bool, error) {
	var pending bool
	err := withMigrations(d, func(provider *goose.Provider) error {
		var err error
		if pending, err = provider.HasPending(ctx); err != nil {
			return fmt.Errorf("check migrations: %w", err)
		}
		return nil
	})
	return pending, err
}

func withMigrations(d *DB, fn func(provider *goose.Provider) error) (err error) {
	sqlDB := stdlib.OpenDBFromPool(d.pool)
	defer func() { err = errors.Join(err, sqlDB.Close()) }()

	provider, err := newProvider(sqlDB)
	if err != nil {
		return err
	}
	return fn(provider)
}

func newProvider(sqlDB *sql.DB) (*goose.Provider, error) {
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return nil, fmt.Errorf("create migration lock: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, Migrations(), goose.WithSessionLocker(locker))
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return provider, nil
}
