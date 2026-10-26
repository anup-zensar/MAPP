package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mapp/internal/platform/database/sqlcdb"
)

const maintenanceDatabase = "postgres"

// CreateIfMissing creates the database dsn names when its server does not have it, connected to the server's
// maintenance database. Identifiers cannot be query parameters, so the statement is built here.
func CreateIfMissing(ctx context.Context, dsn string) error {
	poolConfig, err := pgxpool.ParseConfig(dsn) // takes the pool_* settings out, which a server would refuse
	if err != nil {
		return errMalformedURL
	}
	cfg := poolConfig.ConnConfig
	name := cfg.Database
	if name == "" {
		return nil
	}
	cfg.Database = maintenanceDatabase
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connect to the database server: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	return createDatabase(ctx, conn, name)
}

func createDatabase(ctx context.Context, conn *pgx.Conn, name string) error {
	exists, err := sqlcdb.New(conn).DatabaseExists(ctx, name)
	if err != nil {
		return fmt.Errorf("look for database %s: %w", name, err)
	}
	if exists {
		return nil
	}
	if _, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize()); err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}

// Empty deletes every row of every table but goose's in one TRUNCATE, which the write-once triggers let through.
func Empty(ctx context.Context, d *DB) error {
	tables, err := d.Queries(ctx).DataTables(ctx)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	if len(tables) == 0 {
		return nil
	}
	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, pgx.Identifier{table}.Sanitize())
	}
	if _, err = d.pool.Exec(ctx, "TRUNCATE "+strings.Join(quoted, ", ")); err != nil {
		return fmt.Errorf("empty tables: %w", err)
	}
	return nil
}
