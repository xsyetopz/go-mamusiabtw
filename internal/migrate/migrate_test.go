package migrate_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xsyetopz/jagpda/internal/migrate"
)

func TestRunnerIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	migrationPath := filepath.Join(dir, "001_init.sql")
	migrationSQL := []byte("CREATE TABLE IF NOT EXISTS t1 (id INTEGER PRIMARY KEY);")
	if err := os.WriteFile(migrationPath, migrationSQL, 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}

	r, err := migrate.New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err = r.Run(ctx, db); err != nil {
		t.Fatalf("Run(1): %v", err)
	}
	if err = r.Run(ctx, db); err != nil {
		t.Fatalf("Run(2): %v", err)
	}

	var n int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(1) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 applied migration, got %d", n)
	}
}
