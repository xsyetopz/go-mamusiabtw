package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
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
	db, err := sql.Open("sqlite", dsn)
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
	params := url.Values{}
	params.Add("_pragma", "journal_mode(WAL)")
	params.Add("_pragma", "foreign_keys(1)")
	params.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", sqliteBusyTimeoutMS))

	dsn := (&url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: params.Encode(),
	}).String()
	return "file:" + strings.TrimPrefix(dsn, "file://")
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
