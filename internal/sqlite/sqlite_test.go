package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func TestBuildDSNIncludesExpectedPragmas(t *testing.T) {
	dsn := buildDSN("test.sqlite")
	for _, want := range []string{
		"file:test.sqlite?",
		"_pragma=journal_mode%28WAL%29",
		"_pragma=foreign_keys%281%29",
		"_pragma=busy_timeout%285000%29",
	} {
		if !strings.Contains(dsn, want) {
			t.Fatalf("dsn %q does not contain %q", dsn, want)
		}
	}
}

func TestOpenAppliesPragmas(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(context.Background(), Options{Path: dir + "/test.sqlite"})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	assertPragmaInt(t, db, "journal_mode", "wal")
	assertPragmaInt(t, db, "foreign_keys", int64(1))
	assertPragmaInt(t, db, "busy_timeout", int64(sqliteBusyTimeoutMS))
}

func assertPragmaInt[T comparable](t *testing.T, db *sql.DB, pragma string, want T) {
	t.Helper()

	row := db.QueryRow("PRAGMA " + pragma)
	var got T
	if err := row.Scan(&got); err != nil {
		t.Fatalf("scan pragma %s: %v", pragma, err)
	}
	if got != want {
		t.Fatalf("pragma %s = %v, want %v", pragma, got, want)
	}
}
