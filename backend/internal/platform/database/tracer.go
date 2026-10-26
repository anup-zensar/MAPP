package database

import (
	"context"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/querycount"
)

type counter struct{}

func (counter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	querycount.Add(ctx)
	return ctx
}

func (counter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {} // counted at the start

func (counter) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceCopyFromStartData) context.Context {
	querycount.Add(ctx)
	return ctx
}

func (counter) TraceCopyFromEnd(context.Context, *pgx.Conn, pgx.TraceCopyFromEndData) {} // counted at the start
