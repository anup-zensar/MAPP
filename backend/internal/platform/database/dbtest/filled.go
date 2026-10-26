package dbtest

import (
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/database"
)

type filling struct {
	server             *url.URL
	template, migrated string
	fill               func(db *database.DB)
}

// NewFilledDB gives tb its own copy of a database that fill fills once and the server keeps, named after name
// and today's migrations: benchmarks need far more data than a test, and building it takes minutes.
func NewFilledDB(tb testing.TB, name string, fill func(db *database.DB)) *database.DB {
	tb.Helper()
	server := serverURL(tb)
	return create(tb, server, filledTemplate(tb, server, name, fill))
}

func filledTemplate(tb testing.TB, server *url.URL, name string, fill func(db *database.DB)) string {
	tb.Helper()
	f := filling{
		server: server, template: "mapp_perf_" + name + "_" + migrationsHash(tb), migrated: migratedTemplate(tb, server),
		fill: fill,
	}
	admin := connect(tb, server)
	defer closeConn(tb, admin)
	if !lockedTemplateExists(tb, admin, f.template) {
		buildFilled(tb, admin, f)
	}
	return f.template
}

// lockedTemplateExists waits for any other build of template, which holds the lock until its connection ends.
func lockedTemplateExists(tb testing.TB, admin *pgx.Conn, template string) bool {
	tb.Helper()
	if _, err := admin.Exec(tb.Context(), "SELECT pg_advisory_lock(hashtext($1))", template); err != nil {
		tb.Fatalf("lock template %s: %v", template, err)
	}
	var exists bool
	err := admin.QueryRow(tb.Context(), "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", template).Scan(&exists)
	if err != nil {
		tb.Fatalf("look for template %s: %v", template, err)
	}
	return exists
}

func buildFilled(tb testing.TB, admin *pgx.Conn, f filling) {
	tb.Helper()
	building := f.template + "_building"
	mustExec(tb, admin, "DROP DATABASE IF EXISTS "+quote(building)+" WITH (FORCE)")
	mustExec(tb, admin, "CREATE DATABASE "+quote(building)+" TEMPLATE "+quote(f.migrated))
	fillDatabase(tb, withDatabase(f.server, building), f.fill)
	mustExec(tb, admin, "ALTER DATABASE "+quote(building)+" RENAME TO "+quote(f.template))
}

func fillDatabase(tb testing.TB, dsn string, fill func(db *database.DB)) {
	tb.Helper()
	db, err := database.Open(tb.Context(), dsn)
	if err != nil {
		tb.Fatalf("open the template being built: %v", err)
	}
	defer db.Close()
	fill(db)
}
