package dbtest

import (
	"net/url"
	"testing"

	"mapp/internal/platform/database"
)

// NewFilledDB gives tb its own copy of a database that fill fills once and the server keeps, named after name
// and today's migrations: benchmarks need far more data than a test, and building it takes minutes.
func NewFilledDB(tb testing.TB, name string, fill func(db *database.DB)) *database.DB {
	tb.Helper()
	server := serverURL(tb)
	return create(tb, server, filledTemplate(tb, server, name, fill))
}

func filledTemplate(tb testing.TB, server *url.URL, name string, fill func(db *database.DB)) string {
	tb.Helper()
	migrated := migratedTemplate(tb, server)
	template := "mapp_perf_" + name + "_" + migrationsHash(tb)
	admin := connect(tb, server)
	defer closeConn(tb, admin)

	if _, err := admin.Exec(tb.Context(), "SELECT pg_advisory_lock(hashtext($1))", template); err != nil {
		tb.Fatalf("lock template %s: %v", template, err)
	}
	var exists bool
	err := admin.QueryRow(tb.Context(), "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", template).Scan(&exists)
	if err != nil {
		tb.Fatalf("look for template %s: %v", template, err)
	}
	if exists {
		return template
	}
	building := template + "_building"
	mustExec(tb, admin, "DROP DATABASE IF EXISTS "+quote(building)+" WITH (FORCE)")
	mustExec(tb, admin, "CREATE DATABASE "+quote(building)+" TEMPLATE "+quote(migrated))
	db, err := database.Open(tb.Context(), withDatabase(server, building))
	if err != nil {
		tb.Fatalf("open template %s: %v", building, err)
	}
	fill(db)
	db.Close()
	mustExec(tb, admin, "ALTER DATABASE "+quote(building)+" RENAME TO "+quote(template))
	return template
}
