// Package database runs MAPP's PostgreSQL work: the pool, the migrations and context-carried transactions.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"mapp/internal/platform/database/sqlcdb"
	"mapp/internal/platform/memo"
)

var (
	ErrReadOnly     = errors.New("database: data cannot be changed inside a Read")
	errMalformedURL = errors.New("parse database url: malformed url or parameters")
)

// DB is the pool of one PostgreSQL database; services use it as their store.Transactor.
type DB struct {
	pool *pgxpool.Pool
}

type txKey struct{}

type transaction struct {
	tx       pgx.Tx
	readOnly bool
}

func Open(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errMalformedURL
	}
	cfg.AfterConnect = scanTimestampsInUTC
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &DB{pool: pool}, nil
}

func (d *DB) Close() {
	d.pool.Close()
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// Write runs fn in one transaction once the Writes before it have ended (one at a time, as on the JSON store),
// and returns fn's error unchanged. Inside a Write it joins it (fn passes ctx on); inside a Read, ErrReadOnly.
func (d *DB) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if current, ok := ctx.Value(txKey{}).(transaction); ok {
		if current.readOnly {
			return ErrReadOnly
		}
		return fn(ctx)
	}
	return d.run(ctx, pgx.TxOptions{}, func(ctx context.Context) error {
		if err := d.Queries(ctx).LockWrites(ctx); err != nil {
			return fmt.Errorf("wait for the other writes: %w", err)
		}
		return fn(ctx)
	})
}

// Read runs fn in a read-only transaction that sees one snapshot; inside a transaction, it joins it.
func (d *DB) Read(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(transaction); ok {
		return fn(ctx)
	}
	readOnlySnapshot := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	return d.run(memo.With(ctx), readOnlySnapshot, fn)
}

func (d *DB) Queries(ctx context.Context) *sqlcdb.Queries {
	if current, ok := ctx.Value(txKey{}).(transaction); ok {
		return sqlcdb.New(current.tx)
	}
	return sqlcdb.New(d.pool)
}

func (d *DB) run(ctx context.Context, options pgx.TxOptions, fn func(ctx context.Context) error) error {
	tx, err := d.pool.BeginTx(ctx, options)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current := transaction{tx: tx, readOnly: options.AccessMode == pgx.ReadOnly}
	if err = fn(context.WithValue(ctx, txKey{}, current)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func scanTimestampsInUTC(_ context.Context, conn *pgx.Conn) error {
	codec := &pgtype.TimestamptzCodec{ScanLocation: time.UTC}
	conn.TypeMap().RegisterType(&pgtype.Type{Name: "timestamptz", OID: pgtype.TimestamptzOID, Codec: codec})
	return nil
}
