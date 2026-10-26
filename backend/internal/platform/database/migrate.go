package database

import (
	"context"
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
func Migrate(ctx context.Context, d *DB) (err error) {
	sqlDB := stdlib.OpenDBFromPool(d.pool)
	defer func() { err = errors.Join(err, sqlDB.Close()) }()

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migration lock: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, Migrations(), goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	if _, err = provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
