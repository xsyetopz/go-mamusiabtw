package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	// SQLite driver (CGO).
	_ "github.com/mattn/go-sqlite3"
)

type Options struct {
	Path string
}

func Open(ctx context.Context, opts Options) (*sql.DB, error) {
	if opts.Path == "" {
		return nil, errors.New("sqlite path is required")
	}

	dir := filepath.Dir(opts.Path)
	if err := os.MkdirAll(dir, sqliteDirPerm); err != nil {
		return nil, fmt.Errorf("create sqlite directory %q: %w", dir, err)
	}

	dsn := buildDSN(opts.Path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetConnMaxLifetime(sqliteConnMaxLifetime)
	db.SetMaxIdleConns(sqliteMaxIdleConns)
	db.SetMaxOpenConns(sqliteMaxOpenConns)

	if pingErr := ping(ctx, db); pingErr != nil {
		_ = db.Close()
		return nil, pingErr
	}

	return db, nil
}

func buildDSN(path string) string {
	// WAL improves concurrency for typical Discord app workloads.
	// _foreign_keys ensures FK enforcement on each connection.
	//
	// busy_timeout is in milliseconds.
	return fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=1&_busy_timeout=%d", path, sqliteBusyTimeoutMS)
}

func ping(ctx context.Context, db *sql.DB) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}
	return nil
}

const (
	sqliteDirPerm         = 0o750
	sqliteConnMaxLifetime = 30 * time.Minute
	sqliteMaxIdleConns    = 2
	sqliteMaxOpenConns    = 10
	sqliteBusyTimeoutMS   = 5000
)
