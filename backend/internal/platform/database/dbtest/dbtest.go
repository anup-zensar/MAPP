// Package dbtest gives each store test its own PostgreSQL database on the server named by TEST_DATABASE_URL,
// copied from a migrated template and dropped when the test ends. Under -short, store tests are skipped.
package dbtest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"mapp/internal/platform/database"
)

const (
	urlVariable     = "TEST_DATABASE_URL"
	emptyTemplate   = "template0"
	cleanupDeadline = 30 * time.Second
)

func NewDB(tb testing.TB) *database.DB {
	tb.Helper()
	server := serverURL(tb)
	return create(tb, server, migratedTemplate(tb, server))
}

func serverURL(tb testing.TB) *url.URL {
	tb.Helper()
	if testing.Short() {
		tb.Skip("store test skipped under -short: it needs PostgreSQL at " + urlVariable)
	}
	raw := os.Getenv(urlVariable)
	if raw == "" {
		tb.Fatal(urlVariable + " is not set: point it at a PostgreSQL server, or run go test -short")
	}
	server, err := url.Parse(raw)
	if err != nil || (server.Scheme != "postgres" && server.Scheme != "postgresql") {
		tb.Fatal(urlVariable + " must be a postgres:// URL")
	}
	return server
}

// migratedTemplate returns the template holding today's migrations, named after their hash. It is built once,
// under an advisory lock, so parallel test processes never build it twice.
func migratedTemplate(tb testing.TB, server *url.URL) string {
	tb.Helper()
	ctx := tb.Context()
	name := "mapp_template_" + migrationsHash(tb)
	admin := connect(tb, server)
	defer closeConn(tb, admin) // ending the session releases the advisory lock

	mustExec(tb, admin, "SELECT pg_advisory_lock(hashtext('mapp.dbtest'))")
	var exists bool
	err := admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists)
	if err != nil {
		tb.Fatalf("look for template %s: %v", name, err)
	}
	if !exists {
		buildTemplate(tb, admin, server, name)
	}
	return name
}

// buildTemplate migrates a database under a temporary name and renames it only when complete, so a run that
// stopped half-way never leaves a broken template behind.
func buildTemplate(tb testing.TB, admin *pgx.Conn, server *url.URL, name string) {
	tb.Helper()
	building := name + "_building"
	mustExec(tb, admin, "DROP DATABASE IF EXISTS "+quote(building)+" WITH (FORCE)")
	mustExec(tb, admin, "CREATE DATABASE "+quote(building)+" TEMPLATE "+emptyTemplate)
	db, err := database.Open(tb.Context(), withDatabase(server, building))
	if err != nil {
		tb.Fatalf("open template %s: %v", building, err)
	}
	err = database.Migrate(tb.Context(), db)
	db.Close()
	if err != nil {
		tb.Fatalf("migrate template %s: %v", building, err)
	}
	mustExec(tb, admin, "ALTER DATABASE "+quote(building)+" RENAME TO "+quote(name))
}

func create(tb testing.TB, server *url.URL, template string) *database.DB {
	tb.Helper()
	name := "mapp_test_" + strings.ToLower(rand.Text())
	admin := connect(tb, server)
	mustExec(tb, admin, "CREATE DATABASE "+quote(name)+" TEMPLATE "+quote(template))
	closeConn(tb, admin)

	db, err := database.Open(tb.Context(), withDatabase(server, name))
	if err != nil {
		tb.Fatalf("open test database %s: %v", name, err)
	}
	cleanupCtx := context.WithoutCancel(tb.Context())
	tb.Cleanup(func() {
		db.Close()
		drop(cleanupCtx, tb, server, name)
	})
	return db
}

func drop(ctx context.Context, tb testing.TB, server *url.URL, name string) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(ctx, cleanupDeadline)
	defer cancel()
	admin, err := pgx.Connect(ctx, server.String())
	if err != nil {
		tb.Errorf("drop test database %s: %v", name, err)
		return
	}
	defer func() { _ = admin.Close(ctx) }()
	if _, err = admin.Exec(ctx, "DROP DATABASE IF EXISTS "+quote(name)+" WITH (FORCE)"); err != nil {
		tb.Errorf("drop test database %s: %v", name, err)
	}
}

func migrationsHash(tb testing.TB) string {
	tb.Helper()
	files := database.Migrations()
	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		tb.Fatalf("list migrations: %v", err)
	}
	hash := sha256.New()
	for _, name := range names {
		content, readErr := fs.ReadFile(files, name)
		if readErr != nil {
			tb.Fatalf("read migration %s: %v", name, readErr)
		}
		hash.Write([]byte(name))
		hash.Write(content)
	}
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

func connect(tb testing.TB, server *url.URL) *pgx.Conn {
	tb.Helper()
	conn, err := pgx.Connect(tb.Context(), server.String())
	if err != nil {
		tb.Fatalf("connect to %s: %v", urlVariable, err)
	}
	return conn
}

func closeConn(tb testing.TB, conn *pgx.Conn) {
	tb.Helper()
	if err := conn.Close(tb.Context()); err != nil {
		tb.Errorf("close connection: %v", err)
	}
}

func mustExec(tb testing.TB, conn *pgx.Conn, sql string) {
	tb.Helper()
	if _, err := conn.Exec(tb.Context(), sql); err != nil {
		tb.Fatalf("%s: %v", sql, err)
	}
}

func withDatabase(server *url.URL, name string) string {
	target := *server
	target.Path = "/" + name
	return target.String()
}

func quote(name string) string {
	return pgx.Identifier{name}.Sanitize()
}
