package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Runner struct {
	dir string
}

func New(dir string) (Runner, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return Runner{}, errors.New("migrations dir is required")
	}

	return Runner{dir: dir}, nil
}

func (r Runner) Run(ctx context.Context, db *sql.DB) error {
	if err := ensureTable(ctx, db); err != nil {
		return err
	}

	files, err := listSQLFiles(r.dir)
	if err != nil {
		return err
	}

	applied, err := loadApplied(ctx, db)
	if err != nil {
		return err
	}

	for _, file := range files {
		if applied[file] {
			continue
		}

		if applyErr := applyOne(ctx, db, r.dir, file); applyErr != nil {
			return applyErr
		}
	}

	return nil
}

func ensureTable(ctx context.Context, db *sql.DB) error {
	const query = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename TEXT PRIMARY KEY,
	applied_at INTEGER NOT NULL
);`

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func listSQLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		out = append(out, name)
	}

	sort.Strings(out)
	return out, nil
}

func loadApplied(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, fmt.Errorf("scan schema_migrations row: %w", scanErr)
		}
		out[name] = true
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate schema_migrations: %w", rowsErr)
	}

	return out, nil
}

func applyOne(ctx context.Context, db *sql.DB, dir, filename string) error {
	full := filepath.Join(dir, filename)
	bytes, err := os.ReadFile(full)
	if err != nil {
		return fmt.Errorf("read migration %q: %w", filename, err)
	}

	sqlText := strings.TrimSpace(string(bytes))
	if sqlText == "" {
		return fmt.Errorf("migration %q is empty", filename)
	}

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx for migration %q: %w", filename, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, execErr := tx.ExecContext(ctx, sqlText); execErr != nil {
		return fmt.Errorf("exec migration %q: %w", filename, execErr)
	}

	if _, execErr := tx.ExecContext(
		ctx,
		"INSERT INTO schema_migrations(filename, applied_at) VALUES (?, ?)",
		filename,
		time.Now().Unix(),
	); execErr != nil {
		return fmt.Errorf("record migration %q: %w", filename, execErr)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return fmt.Errorf("commit migration %q: %w", filename, commitErr)
	}

	return nil
}
