// Package database runs MAPP's PostgreSQL work: the pool, the migrations and context-carried transactions.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"mapp/internal/platform/database/sqlcdb"
)

const (
	serializationFailure = "40001"
	deadlockDetected     = "40P01"
	maxWriteAttempts     = 5
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

// Write runs fn in one serializable transaction and returns fn's error unchanged. PostgreSQL aborts a
// transaction that lost a race with another, and Write then runs fn again, so fn must change nothing outside
// the transaction. Inside a Write, Write joins it; inside a Read, it returns ErrReadOnly.
func (d *DB) Write(ctx context.Context, fn func(ctx context.Context) error) error {
	if current, ok := ctx.Value(txKey{}).(transaction); ok {
		if current.readOnly {
			return ErrReadOnly
		}
		return fn(ctx)
	}
	serializable := pgx.TxOptions{IsoLevel: pgx.Serializable}
	var err error
	for range maxWriteAttempts {
		err = d.run(ctx, serializable, fn)
		if !isRaceLost(err) {
			return err
		}
	}
	return err
}

// Read runs fn in a read-only transaction that sees one snapshot; inside a transaction, it joins it.
func (d *DB) Read(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(transaction); ok {
		return fn(ctx)
	}
	readOnlySnapshot := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}
	return d.run(ctx, readOnlySnapshot, fn)
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

func isRaceLost(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && (pgErr.Code == serializationFailure || pgErr.Code == deadlockDetected)
}

func scanTimestampsInUTC(_ context.Context, conn *pgx.Conn) error {
	codec := &pgtype.TimestamptzCodec{ScanLocation: time.UTC}
	conn.TypeMap().RegisterType(&pgtype.Type{Name: "timestamptz", OID: pgtype.TimestamptzOID, Codec: codec})
	return nil
}
